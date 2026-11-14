package cti

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/store"
)

type Providers struct {
	VirusTotal string
	AbuseIPDB  string
	OTX        string
	Shodan     string
}

type Result struct {
	Findings   []store.CTIFinding
	Indicators []store.CTIIndicator
	Reports    []store.CTIReport
}

func Scan(ctx context.Context, assets []store.CTIAsset, providers Providers) Result {
	result := Result{}
	client := &http.Client{Timeout: 10 * time.Second}

	domains := map[string]bool{}
	ips := map[string]bool{}
	urls := map[string]bool{}
	for _, asset := range assets {
		switch asset.Type {
		case "domain":
			domains[asset.Value] = true
		case "email":
			if at := strings.LastIndex(asset.Value, "@"); at >= 0 {
				domains[strings.ToLower(asset.Value[at+1:])] = true
			}
		case "ip":
			ips[asset.Value] = true
		case "url":
			urls[asset.Value] = true
			if parsed, err := url.Parse(asset.Value); err == nil {
				domains[parsed.Hostname()] = true
			}
		}
	}

	for domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}
		result.Findings = append(result.Findings, dnsFindings(ctx, domain)...)
		result.Findings = append(result.Findings, emailFindings(ctx, domain)...)
		result.Findings = append(result.Findings, tlsFindings(ctx, domain)...)
		result.Findings = append(result.Findings, httpFindings(ctx, client, domain)...)
		result.Findings = append(result.Findings, urlhaus(ctx, client, domain, &result)...)
		result.Findings = append(result.Findings, threatfox(ctx, client, domain, &result)...)

		if providers.VirusTotal != "" {
			result.Findings = append(result.Findings, virusTotal(ctx, client, domain, providers.VirusTotal, &result)...)
		}
		if providers.OTX != "" {
			result.Findings = append(result.Findings, otx(ctx, client, domain, providers.OTX, &result)...)
		}
	}

	for ip := range ips {
		if providers.AbuseIPDB != "" {
			result.Findings = append(result.Findings, abuseIPDB(ctx, client, ip, providers.AbuseIPDB, &result)...)
		}
		if providers.Shodan != "" {
			result.Findings = append(result.Findings, shodan(ctx, client, ip, providers.Shodan)...)
		}
	}

	// URLhaus can also search a URL directly.
	for raw := range urls {
		if strings.HasPrefix(raw, "http") {
			// The host lookup already covers the domain. Keep the URL as a
			// local indicator only if URLhaus returns it.
			_ = raw
		}
	}

	return result
}

func dnsFindings(ctx context.Context, domain string) []store.CTIFinding {
	findings := []store.CTIFinding{}
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	addresses, err := net.DefaultResolver.LookupHost(lookupCtx, domain)
	if err != nil || len(addresses) == 0 {
		findings = append(findings, finding("dns", "high", "Domain does not resolve",
			"The main domain could not be resolved. This can break email, the site, or both.",
			domain, "Check the DNS zone and registrar."))
		return findings
	}

	findings = append(findings, finding("dns", "info", "Domain resolves",
		"The domain resolves to "+strings.Join(addresses, ", ")+".", domain, "No action."))
	return findings
}

func emailFindings(ctx context.Context, domain string) []store.CTIFinding {
	findings := []store.CTIFinding{}
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	txt, _ := net.DefaultResolver.LookupTXT(lookupCtx, domain)
	spf := false
	for _, record := range txt {
		if strings.HasPrefix(strings.ToLower(record), "v=spf1") {
			spf = true
			if strings.Contains(strings.ToLower(record), "-all") {
				findings = append(findings, finding("email", "info", "SPF is enforcing",
					"An SPF record exists with a hard fail.", record, "No action."))
			} else {
				findings = append(findings, finding("email", "medium", "SPF is soft",
					"SPF exists but does not use -all, so spoofed mail may still pass.", record,
					"Change the SPF policy to -all after checking every sender."))
			}
		}
	}
	if !spf {
		findings = append(findings, finding("email", "high", "No SPF record",
			"Anyone can send email claiming to be from this domain more easily.",
			domain, "Publish an SPF record covering your real senders."))
	}

	dmarc, _ := net.DefaultResolver.LookupTXT(lookupCtx, "_dmarc."+domain)
	hasDMARC := false
	for _, record := range dmarc {
		if strings.HasPrefix(strings.ToLower(record), "v=dmarc1") {
			hasDMARC = true
			policy := "none"
			lower := strings.ToLower(record)
			if strings.Contains(lower, "p=reject") {
				policy = "reject"
			} else if strings.Contains(lower, "p=quarantine") {
				policy = "quarantine"
			}
			severity := "medium"
			if policy == "reject" {
				severity = "info"
			}
			findings = append(findings, finding("email", severity, "DMARC policy: "+policy,
				"DMARC tells receivers what to do with spoofed mail.", record,
				"Move to p=reject when the reports look clean."))
		}
	}
	if !hasDMARC {
		findings = append(findings, finding("email", "high", "No DMARC record",
			"Without DMARC, receivers cannot reliably reject spoofed mail from this domain.",
			"_dmarc."+domain, "Add a DMARC record starting at p=none, then tighten to reject."))
	}
	return findings
}

func tlsFindings(ctx context.Context, domain string) []store.CTIFinding {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	connection, err := tls.DialWithDialer(dialer, "tcp", domain+":443", &tls.Config{ServerName: domain, MinVersion: tls.VersionTLS12})
	if err != nil {
		return []store.CTIFinding{finding("tls", "medium", "TLS handshake failed",
			"Port 443 did not complete a modern TLS handshake.", err.Error(),
			"Check the certificate chain and TLS configuration.")}
	}
	defer connection.Close()

	cert := connection.ConnectionState().PeerCertificates[0]
	days := int(time.Until(cert.NotAfter).Hours() / 24)
	severity := "info"
	title := "TLS certificate is valid"
	if days < 0 {
		severity, title = "critical", "TLS certificate has expired"
	} else if days < 14 {
		severity, title = "high", "TLS certificate expires soon"
	} else if days < 30 {
		severity, title = "medium", "TLS certificate expiry is close"
	}

	return []store.CTIFinding{finding("tls", severity, title,
		fmt.Sprintf("Certificate for %s expires in %d days. Issuer: %s.", domain, days, cert.Issuer.CommonName),
		cert.Subject.CommonName, "Renew the certificate and automate renewal.")}
}

func httpFindings(ctx context.Context, client *http.Client, domain string) []store.CTIFinding {
	findings := []store.CTIFinding{}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+domain, nil)
	request.Header.Set("User-Agent", "CrewSecurity/1.0")
	response, err := client.Do(request)
	if err != nil {
		return []store.CTIFinding{finding("web", "medium", "HTTPS homepage is unreachable",
			"The domain did not answer a normal HTTPS request.", err.Error(),
			"Check the web server, DNS and certificate.")}
	}
	defer response.Body.Close()

	headers := response.Header
	for _, check := range []struct {
		header   string
		severity string
		title    string
		fix      string
	}{
		{"Strict-Transport-Security", "medium", "No HSTS header",
			"Add Strict-Transport-Security with a long max-age after testing HTTPS."},
		{"Content-Security-Policy", "medium", "No Content-Security-Policy",
			"Add a CSP to reduce script injection impact."},
		{"X-Frame-Options", "low", "No clickjacking header",
			"Add X-Frame-Options: DENY or a frame-ancestors CSP."},
		{"Referrer-Policy", "low", "No Referrer-Policy",
			"Add Referrer-Policy: strict-origin-when-cross-origin."},
	} {
		if headers.Get(check.header) == "" {
			findings = append(findings, finding("web", check.severity, check.title,
				"The homepage response does not include "+check.header+".", "https://"+domain, check.fix))
		}
	}

	securityTxt, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+domain+"/.well-known/security.txt", nil)
	if securityResponse, err := client.Do(securityTxt); err == nil {
		defer securityResponse.Body.Close()
		if securityResponse.StatusCode == http.StatusOK {
			findings = append(findings, finding("web", "info", "security.txt exists",
				"Researchers have a published way to report vulnerabilities.", "https://"+domain+"/.well-known/security.txt",
				"No action."))
		}
	} else {
		findings = append(findings, finding("web", "low", "No security.txt",
			"There is no published security contact for researchers.", "https://"+domain+"/.well-known/security.txt",
			"Publish a security.txt with a security contact."))
	}

	return findings
}

func urlhaus(ctx context.Context, client *http.Client, domain string, result *Result) []store.CTIFinding {
	form := url.Values{}
	form.Set("host", domain)
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://urlhaus-api.abuse.ch/v1/host/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}

	var payload struct {
		QueryStatus string `json:"query_status"`
		URLs        []struct {
			URL    string   `json:"url"`
			Threat string   `json:"threat"`
			Tags   []string `json:"tags"`
			Date   string   `json:"date_added"`
		} `json:"urls"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
		return nil
	}
	if payload.QueryStatus != "ok" || len(payload.URLs) == 0 {
		return nil
	}

	findings := []store.CTIFinding{}
	for _, item := range payload.URLs {
		findings = append(findings, finding("threat", "high", "URLhaus lists a malicious URL",
			"A URL on this domain is in URLhaus: "+item.Threat+".", item.URL,
			"Remove the payload, rotate credentials and review access logs."))
		result.Indicators = append(result.Indicators, store.CTIIndicator{
			Type: "url", Value: item.URL, Name: "URLhaus " + item.Threat,
			Description: "Listed by URLhaus", Confidence: 85, TLP: "amber",
			Source: "URLhaus", SourceURL: item.URL, Tags: item.Tags,
		})
	}
	result.Reports = append(result.Reports, store.CTIReport{
		Title: "URLhaus match for " + domain, Summary: "URLhaus returned malicious URLs for this host.",
		Source: "URLhaus", SourceURL: "https://urlhaus.abuse.ch/host/" + domain, Confidence: 85, TLP: "amber",
		Tags: []string{"urlhaus", "malware"},
	})
	return findings
}

func threatfox(ctx context.Context, client *http.Client, domain string, result *Result) []store.CTIFinding {
	body := fmt.Sprintf(`{"query":"search_ioc","search_term":%q}`, domain)
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://threatfox-api.abuse.ch/api/v1/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}

	var payload struct {
		QueryStatus string `json:"query_status"`
		Data        []struct {
			IOC              string `json:"ioc"`
			ThreatType       string `json:"threat_type"`
			MalwarePrintable string `json:"malware_printable"`
			ConfidenceLevel  int    `json:"confidence_level"`
			FirstSeen        string `json:"first_seen"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
		return nil
	}
	if payload.QueryStatus != "ok" {
		return nil
	}

	findings := []store.CTIFinding{}
	for _, item := range payload.Data {
		findings = append(findings, finding("threat", "high", "ThreatFox indicator match",
			"ThreatFox associates this domain with "+item.MalwarePrintable+" ("+item.ThreatType+").",
			item.IOC, "Block the indicator and investigate hosts that contacted it."))
		result.Indicators = append(result.Indicators, store.CTIIndicator{
			Type: "domain", Value: domain, Name: "ThreatFox " + item.MalwarePrintable,
			Description: item.ThreatType, Confidence: maxInt(item.ConfidenceLevel, 50), TLP: "amber",
			Source: "ThreatFox", SourceURL: "https://threatfox.abuse.ch/", FirstSeen: item.FirstSeen,
		})
	}
	return findings
}

func virusTotal(ctx context.Context, client *http.Client, domain, key string, result *Result) []store.CTIFinding {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.virustotal.com/api/v3/domains/"+url.PathEscape(domain), nil)
	request.Header.Set("x-apikey", key)
	response, err := client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		return []store.CTIFinding{finding("provider", "info", "VirusTotal lookup failed",
			"VirusTotal did not return a domain report.", domain, "Check the VirusTotal key.")}
	}
	defer response.Body.Close()

	var payload struct {
		Data struct {
			Attributes struct {
				LastAnalysisStats struct {
					Malicious  int `json:"malicious"`
					Suspicious int `json:"suspicious"`
					Harmless   int `json:"harmless"`
				} `json:"last_analysis_stats"`
				Reputation int `json:"reputation"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
		return nil
	}

	malicious := payload.Data.Attributes.LastAnalysisStats.Malicious
	suspicious := payload.Data.Attributes.LastAnalysisStats.Suspicious
	severity := "info"
	if malicious > 0 {
		severity = "critical"
	} else if suspicious > 0 {
		severity = "high"
	}
	findings := []store.CTIFinding{finding("provider", severity, "VirusTotal domain reputation",
		fmt.Sprintf("%d malicious, %d suspicious engines. Reputation %d.", malicious, suspicious, payload.Data.Attributes.Reputation),
		domain, "Review the detections before changing DNS.")}

	if malicious > 0 || suspicious > 0 {
		result.Indicators = append(result.Indicators, store.CTIIndicator{
			Type: "domain", Value: domain, Name: "VirusTotal detections",
			Description: fmt.Sprintf("%d malicious / %d suspicious", malicious, suspicious),
			Confidence:  80, TLP: "amber", Source: "VirusTotal",
			SourceURL: "https://www.virustotal.com/gui/domain/" + domain,
		})
	}
	return findings
}

func abuseIPDB(ctx context.Context, client *http.Client, ip, key string, result *Result) []store.CTIFinding {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.abuseipdb.com/api/v2/check?ipAddress="+url.QueryEscape(ip)+"&maxAgeInDays=90", nil)
	request.Header.Set("Key", key)
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		return nil
	}
	defer response.Body.Close()

	var payload struct {
		Data struct {
			AbuseConfidenceScore int    `json:"abuseConfidenceScore"`
			TotalReports         int    `json:"totalReports"`
			CountryCode          string `json:"countryCode"`
			ISP                  string `json:"isp"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return nil
	}
	if payload.Data.AbuseConfidenceScore < 25 {
		return []store.CTIFinding{finding("provider", "info", "AbuseIPDB is clean",
			fmt.Sprintf("%s has an abuse score of %d.", ip, payload.Data.AbuseConfidenceScore), ip, "No action.")}
	}

	result.Indicators = append(result.Indicators, store.CTIIndicator{
		Type: "ip", Value: ip, Name: "AbuseIPDB abusive host",
		Description: fmt.Sprintf("Score %d, %d reports, %s", payload.Data.AbuseConfidenceScore, payload.Data.TotalReports, payload.Data.ISP),
		Confidence:  payload.Data.AbuseConfidenceScore, TLP: "amber", Source: "AbuseIPDB",
		SourceURL: "https://www.abuseipdb.com/check/" + ip,
	})
	return []store.CTIFinding{finding("provider", "high", "AbuseIPDB reports abuse",
		fmt.Sprintf("%s has an abuse score of %d from %d reports.", ip, payload.Data.AbuseConfidenceScore, payload.Data.TotalReports),
		ip, "Investigate what is hosted on this IP and block outbound access if unexplained.")}
}

func shodan(ctx context.Context, client *http.Client, ip, key string) []store.CTIFinding {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.shodan.io/shodan/host/"+url.PathEscape(ip)+"?key="+url.QueryEscape(key), nil)
	response, err := client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		return nil
	}
	defer response.Body.Close()

	var payload struct {
		Ports []int    `json:"ports"`
		Vulns []string `json:"vulns"`
		Org   string   `json:"org"`
		OS    string   `json:"os"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
		return nil
	}
	findings := []store.CTIFinding{finding("provider", "info", "Shodan exposure",
		fmt.Sprintf("%s exposes %d ports. Org: %s.", ip, len(payload.Ports), payload.Org), ip,
		"Close ports that do not need to be public.")}
	if len(payload.Vulns) > 0 {
		findings = append(findings, finding("provider", "high", "Shodan lists known vulnerabilities",
			fmt.Sprintf("%d vulnerabilities are associated with this host.", len(payload.Vulns)),
			strings.Join(payload.Vulns, ", "), "Patch the exposed services."))
	}
	return findings
}

func otx(ctx context.Context, client *http.Client, domain, key string, result *Result) []store.CTIFinding {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://otx.alienvault.com/api/v1/indicators/domain/"+url.PathEscape(domain)+"/general", nil)
	request.Header.Set("X-OTX-API-KEY", key)
	response, err := client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		return nil
	}
	defer response.Body.Close()

	var payload struct {
		PulseInfo struct {
			Count int `json:"count"`
		} `json:"pulse_info"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return nil
	}
	if payload.PulseInfo.Count == 0 {
		return []store.CTIFinding{finding("provider", "info", "AlienVault OTX has no pulses",
			"No OTX pulses reference this domain.", domain, "No action.")}
	}

	result.Reports = append(result.Reports, store.CTIReport{
		Title:   "AlienVault OTX pulses for " + domain,
		Summary: fmt.Sprintf("%d OTX pulses reference this domain.", payload.PulseInfo.Count),
		Source:  "AlienVault OTX", SourceURL: "https://otx.alienvault.com/indicator/domain/" + domain,
		Confidence: 65, TLP: "amber", Tags: []string{"otx", "pulses"},
	})
	return []store.CTIFinding{finding("provider", "medium", "OTX pulses mention this domain",
		fmt.Sprintf("%d pulses reference it.", payload.PulseInfo.Count), domain,
		"Review the pulses and block the listed indicators if relevant.")}
}

func finding(category, severity, title, detail, evidence, fix string) store.CTIFinding {
	return store.CTIFinding{Category: category, Severity: severity, Title: title, Detail: detail, Evidence: evidence, Fix: fix}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
