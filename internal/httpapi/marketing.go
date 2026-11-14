package httpapi

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/llm"
	"github.com/davepartner/crew/internal/modelcatalog"
	"github.com/davepartner/crew/internal/skills"
	"github.com/davepartner/crew/internal/store"
)

type pageEvidence struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Text        string `json:"-"`
	HTML        string `json:"-"`
	Characters  int    `json:"characters"`
	Truncated   bool   `json:"truncated"`
}

func (s *Server) handleMarketingAnalyze(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	var body struct {
		URL      string `json:"url"`
		Focus    string `json:"focus"`
		Playbook string `json:"playbook"`
		AgentID  string `json:"agentId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rawURL := strings.TrimSpace(body.URL)
	if rawURL == "" {
		writeError(w, http.StatusUnprocessableEntity, "Paste the company's website first.")
		return
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	playbook := strings.TrimSpace(body.Playbook)
	if playbook == "" {
		playbook = "funnel-monetization"
	}
	if _, ok := skills.PlaybookBody(playbook); !ok {
		writeError(w, http.StatusUnprocessableEntity, "That playbook doesn't exist.")
		return
	}

	page, err := fetchPage(r.Context(), rawURL)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	agentID := body.AgentID
	if agentID != "" {
		if _, err := s.store.GetAgent(r.Context(), company.ID, agentID); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "That helper doesn't exist.")
			return
		}
	}

	agents, err := s.store.ListAgents(r.Context(), company.ID)
	if err == nil {
		if agentID == "" {
			for _, agent := range agents {
				if agent.Role == "Marketing" || agent.Role == "Audience" || agent.Role == "Growth" {
					agentID = agent.ID
					break
				}
			}
			if agentID == "" && len(agents) > 0 {
				agentID = agents[0].ID
			}
		}
	}

	model, err := marketingModelFor(s, r.Context(), id.UserID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"ok":    false,
			"code":  "no_model",
			"error": "Add an AI model key in Settings → AI models, then try again.",
		})
		return
	}

	if blocked, err := s.checkBudget(r.Context(), id.UserID, company.ID, agentID); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't check the AI budget.")
		return
	} else if blocked != "" {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"ok": false, "code": "budget", "error": blocked})
		return
	}

	system := skills.Harness(playbook, 0)
	prompt := buildMarketingPrompt(page, trim(body.Focus, 400), playbook)

	ctx, cancel := context.WithTimeout(r.Context(), 150*time.Second)
	defer cancel()

	result, err := llm.CompleteResult(ctx, llm.Config{
		Kind:    model.Provider.Kind,
		BaseURL: model.Provider.BaseURL,
		Model:   model.Model,
		APIKey:  model.APIKey,
	}, system, prompt, 2600)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The model couldn't finish that analysis: "+err.Error())
		return
	}

	report := strings.TrimSpace(result.Text)
	if report == "" {
		writeError(w, http.StatusBadGateway, "The model returned an empty report. Please try again.")
		return
	}

	cost := s.saveUsage(r.Context(), id.UserID, company.ID, agentID, "", model, result.Usage)

	host := page.URL
	if parsed, err := url.Parse(page.URL); err == nil {
		host = parsed.Host
	}

	title := "Marketing teardown: " + host
	details := "Reverse-engineer how " + page.URL + " acquires customers.\n"
	if body.Focus != "" {
		details += "Focus: " + trim(body.Focus, 400) + "\n"
	}
	details += "Playbook: " + playbook + "\n"

	task, taskErr := s.store.CreateTask(r.Context(), id.UserID, company.ID, title, details, agentID)
	if taskErr == nil && task.ID != "" {
		_, _ = s.store.AddComment(r.Context(), task.ID, "helper", "Marketing lab", report)
		status := store.StatusReview
		if updated, err := s.store.UpdateTask(r.Context(), company.ID, task.ID, store.TaskPatch{Status: &status}); err == nil {
			task = updated
		}
		_ = s.store.AddActivity(r.Context(), id.UserID, company.ID, "marketing_analysis",
			"Marketing lab finished a teardown of "+host, task.ID, agentID)
		_ = s.store.NotifyCompany(r.Context(), company.ID, "marketing",
			"Marketing teardown ready", host, task.ID)
	} else {
		task = store.Task{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"report": report,
		"model":  model.Provider.Name + " · " + model.Model,
		"cost":   cost,
		"evidence": map[string]any{
			"url":         page.URL,
			"title":       page.Title,
			"description": page.Description,
			"characters":  page.Characters,
			"truncated":   page.Truncated,
		},
		"task":  task,
		"agent": agentID,
	})
}

// marketingModelFor is a seam for tests; production always uses the signed-in
// person's first configured provider.
var marketingModelFor = func(s *Server, ctx context.Context, userID string) (resolvedModel, error) {
	return s.modelForMarketing(ctx, userID)
}

func (s *Server) modelForMarketing(ctx context.Context, userID string) (resolvedModel, error) {
	keys, err := s.store.ListProviderKeys(ctx, userID)
	if err != nil {
		return resolvedModel{}, err
	}

	for _, key := range keys {
		provider, ok := modelcatalog.ByKey(key.Provider)
		if !ok || provider.Key == "ollama" {
			continue
		}

		cipherText, err := s.store.ProviderKeyCipher(ctx, userID, provider.Key)
		if err != nil {
			continue
		}
		apiKey, err := s.cipher.Decrypt(cipherText)
		if err != nil || apiKey == "" {
			continue
		}

		return resolvedModel{Provider: provider, Model: provider.DefaultModel, APIKey: apiKey, Source: "user"}, nil
	}

	if platform, ok := s.platformModel(); ok {
		return platform, nil
	}

	return resolvedModel{}, store.ErrNotFound
}

func buildMarketingPrompt(page pageEvidence, focus, playbook string) string {
	var builder strings.Builder

	builder.WriteString("Reverse-engineer the marketing of this company. Use only the evidence below; ")
	builder.WriteString("mark anything you could not observe as an unknown and say how to check it.\n\n")
	builder.WriteString("Website: " + page.URL + "\n")
	builder.WriteString("Title: " + page.Title + "\n")
	if page.Description != "" {
		builder.WriteString("Meta description: " + page.Description + "\n")
	}
	builder.WriteString("Playbook: " + playbook + "\n")
	if focus != "" {
		builder.WriteString("The founder wants to know: " + focus + "\n")
	}
	builder.WriteString("\n## Evidence: page text\n\n")
	builder.WriteString(page.Text)
	if page.Truncated {
		builder.WriteString("\n\n[page text truncated]")
	}
	builder.WriteString("\n\nWrite the report in markdown, following the template. Be concrete: quote headlines, ")
	builder.WriteString("pricing, offers and CTAs you can see. Do not invent ad spend, traffic or revenue.\n")

	return builder.String()
}

var (
	scriptPattern  = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	stylePattern   = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	svgPattern     = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`)
	commentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)
	titlePattern   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	metaPattern    = regexp.MustCompile(`(?is)<meta[^>]+(?:name|property)=["'](?:description|og:description)["'][^>]*content=["']([^"']*)["']`)
	blockPattern   = regexp.MustCompile(`(?is)</?(?:p|div|section|article|header|footer|li|ul|ol|h[1-6]|tr|table|br|main|nav)[^>]*>`)
	tagPattern     = regexp.MustCompile(`(?s)<[^>]+>`)
	spacePattern   = regexp.MustCompile(`[ \t\f\v]+`)
	newlinePattern = regexp.MustCompile(`\n{3,}`)
)

func fetchPage(ctx context.Context, rawURL string) (pageEvidence, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return pageEvidence{}, errors.New("That doesn't look like a website address.")
	}
	if parsed.User != nil {
		return pageEvidence{}, errors.New("That address isn't allowed.")
	}
	if port := parsed.Port(); port != "" && !portAllowed(port) {
		return pageEvidence{}, errors.New("Only standard web ports are allowed.")
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, errors.New("could not resolve that address")
			}

			for _, candidate := range ips {
				if !addressAllowed(candidate.IP) {
					return nil, errors.New("that address is not public")
				}
			}

			return (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 4 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("unsupported redirect")
			}
			return nil
		},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return pageEvidence{}, errors.New("That address isn't valid.")
	}
	request.Header.Set("User-Agent", "CrewMarketingBot/1.0 (+https://crew.paytrackify.com)")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.8")

	response, err := client.Do(request)
	if err != nil {
		return pageEvidence{}, errors.New("We couldn't read that website. Check the address and try again.")
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return pageEvidence{}, fmt.Errorf("That website returned %s.", response.Status)
	}

	contentType := response.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "text/plain") {
		return pageEvidence{}, errors.New("That address isn't a web page.")
	}

	raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return pageEvidence{}, errors.New("We couldn't read that page.")
	}

	document := string(raw)
	title := ""
	if match := titlePattern.FindStringSubmatch(document); len(match) > 1 {
		title = cleanText(match[1])
	}
	description := ""
	if match := metaPattern.FindStringSubmatch(document); len(match) > 1 {
		description = cleanText(match[1])
	}

	text := extractText(document)
	const limit = 14000
	truncated := false
	if len(text) > limit {
		text = text[:limit]
		truncated = true
	}

	if len(text) < 80 {
		return pageEvidence{}, errors.New("That page had almost no readable text. Try the main page or a pricing page.")
	}

	return pageEvidence{
		URL:         response.Request.URL.String(),
		Title:       title,
		Description: description,
		Text:        text,
		HTML:        document,
		Characters:  len(text),
		Truncated:   truncated,
	}, nil
}

func extractText(document string) string {
	document = scriptPattern.ReplaceAllString(document, " ")
	document = stylePattern.ReplaceAllString(document, " ")
	document = svgPattern.ReplaceAllString(document, " ")
	document = commentPattern.ReplaceAllString(document, " ")
	document = blockPattern.ReplaceAllString(document, "\n")
	document = tagPattern.ReplaceAllString(document, " ")
	document = html.UnescapeString(document)
	document = strings.ReplaceAll(document, "\u00a0", " ")

	lines := strings.Split(document, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = spacePattern.ReplaceAllString(strings.TrimSpace(line), " ")
		if line != "" {
			out = append(out, line)
		}
	}

	return newlinePattern.ReplaceAllString(strings.Join(out, "\n"), "\n\n")
}

func cleanText(value string) string {
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(value), " ")
}

// addressAllowed is a variable so tests can reach a local page server. It is
// always publicAddress in production.
var addressAllowed = publicAddress

var portAllowed = func(port string) bool { return port == "" || port == "80" || port == "443" }

func publicAddress(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	// Carrier-grade NAT is not public either.
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return false
	}
	return true
}
