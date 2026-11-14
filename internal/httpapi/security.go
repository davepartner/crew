package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/cti"
	"github.com/davepartner/crew/internal/llm"
	"github.com/davepartner/crew/internal/store"
)

type securityProvider struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	Docs       string `json:"docs"`
	Configured bool   `json:"configured"`
	Free       bool   `json:"free"`
}

type securityPayload struct {
	Assets        []store.CTIAsset        `json:"assets"`
	Indicators    []store.CTIIndicator    `json:"indicators"`
	Actors        []store.CTIActor        `json:"actors"`
	Reports       []store.CTIReport       `json:"reports"`
	Relationships []store.CTIRelationship `json:"relationships"`
	Findings      []store.CTIFinding      `json:"findings"`
	Assessment    *store.CTIAssessment    `json:"assessment"`
	Providers     []securityProvider      `json:"providers"`
	Summary       map[string]any          `json:"summary"`
	ScannedAt     string                  `json:"scanned_at"`
}

func (s *Server) handleSecurity(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	if _, err := s.ensureCTIAssets(r.Context(), company); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't prepare the security assets.")
		return
	}

	payload, err := s.securityData(r.Context(), company)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load the security workspace.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "security": payload, "company": company})
}

func (s *Server) handleSecurityScan(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	assets, err := s.ensureCTIAssets(r.Context(), company)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't prepare the security assets.")
		return
	}
	if len(assets) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "Add at least one domain, IP, email or URL asset first.")
		return
	}

	providers := s.ctiProviders(r.Context(), company.ID)
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	result := cti.Scan(ctx, assets, providers)
	if err := s.store.ReplaceCTIFindings(r.Context(), company.ID, result.Findings); err != nil {
		writeError(w, http.StatusInternalServerError, "The scan finished but the findings weren't saved.")
		return
	}

	for _, indicator := range result.Indicators {
		_, _ = s.store.UpsertCTIIndicator(r.Context(), company.ID, indicator)
	}

	existingReports, _ := s.store.ListCTIReports(r.Context(), company.ID)
	for _, report := range result.Reports {
		duplicate := false
		for _, existing := range existingReports {
			if existing.SourceURL == report.SourceURL && existing.Title == report.Title {
				duplicate = true
				break
			}
		}
		if !duplicate {
			_, _ = s.store.CreateCTIReport(r.Context(), company.ID, report)
		}
	}

	for _, indicator := range result.Indicators {
		for _, asset := range assets {
			if asset.Type == "domain" && strings.Contains(strings.ToLower(indicator.Value), strings.ToLower(asset.Value)) {
				_, _ = s.store.CreateCTIRelationship(r.Context(), company.ID, store.CTIRelationship{
					SourceType: "indicator", SourceID: indicator.Value,
					TargetType: "asset", TargetID: asset.ID, Relation: "related-to",
					Confidence: 70, Description: "Indicator references this asset.",
				})
			}
		}
	}

	payload, _ := s.securityData(r.Context(), company)
	assessment := s.assessSecurity(r.Context(), id.UserID, company, payload)
	if assessment != nil {
		payload.Assessment = assessment
	}
	payload.ScannedAt = time.Now().UTC().Format(time.RFC3339)

	if _, err := s.ensureCTIAssets(r.Context(), company); err != nil {
		_ = err
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "security": payload})
}

func (s *Server) handleSecurityAsset(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	var body struct {
		Type        string `json:"type"`
		Value       string `json:"value"`
		Label       string `json:"label"`
		Criticality string `json:"criticality"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Type = strings.ToLower(trim(body.Type, 20))
	body.Value = trim(body.Value, 300)
	if body.Value == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add the asset value.")
		return
	}
	switch body.Type {
	case "domain", "ip", "email", "url", "brand":
	default:
		writeError(w, http.StatusUnprocessableEntity, "Asset type must be domain, ip, email, url or brand.")
		return
	}
	criticality := strings.ToLower(trim(body.Criticality, 20))
	if criticality == "" {
		criticality = "medium"
	}

	asset, err := s.store.EnsureCTIAsset(r.Context(), company.ID, body.Type, body.Value,
		trim(body.Label, 120), criticality, "manual")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't add that asset.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "asset": asset})
}

func (s *Server) handleDeleteSecurityAsset(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	if err := s.store.DeleteCTIAsset(r.Context(), company.ID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "That asset doesn't exist.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSecurityIndicator(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	var body struct {
		Type        string   `json:"type"`
		Value       string   `json:"value"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Confidence  int      `json:"confidence"`
		TLP         string   `json:"tlp"`
		Tags        []string `json:"tags"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if trim(body.Type, 20) == "" || trim(body.Value, 500) == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add an indicator type and value.")
		return
	}
	confidence := body.Confidence
	if confidence <= 0 || confidence > 100 {
		confidence = 50
	}
	tlp := strings.ToLower(trim(body.TLP, 10))
	if tlp == "" {
		tlp = "amber"
	}

	indicator, err := s.store.UpsertCTIIndicator(r.Context(), company.ID, store.CTIIndicator{
		Type: strings.ToLower(trim(body.Type, 30)), Value: trim(body.Value, 500),
		Name: trim(body.Name, 200), Description: trim(body.Description, 2000),
		Confidence: confidence, TLP: tlp, Source: "manual", Tags: body.Tags,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that indicator.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "indicator": indicator})
}

func (s *Server) handleSecurityIndicatorStatus(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := strings.ToLower(trim(body.Status, 20))
	if status != "active" && status != "benign" && status != "resolved" {
		writeError(w, http.StatusUnprocessableEntity, "Status must be active, benign or resolved.")
		return
	}
	if err := s.store.UpdateCTIIndicatorStatus(r.Context(), company.ID, r.PathValue("id"), status); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't update that indicator.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteSecurityIndicator(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	if err := s.store.DeleteCTIIndicator(r.Context(), company.ID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't delete that indicator.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSecurityReport(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	var body store.CTIReport
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Title = trim(body.Title, 240)
	body.Summary = trim(body.Summary, 4000)
	body.Source = trim(body.Source, 120)
	if body.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add a report title.")
		return
	}
	if body.Confidence <= 0 || body.Confidence > 100 {
		body.Confidence = 50
	}
	report, err := s.store.CreateCTIReport(r.Context(), company.ID, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that report.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "report": report})
}

func (s *Server) ensureCTIAssets(ctx context.Context, company *store.Company) ([]store.CTIAsset, error) {
	if company.Website != "" {
		if parsed, err := url.Parse(company.Website); err == nil {
			host := strings.ToLower(parsed.Hostname())
			if host != "" {
				_, _ = s.store.EnsureCTIAsset(ctx, company.ID, "domain", host, "Website", "high", "company profile")
			}
		}
	}
	if company.SupportEmail != "" {
		domain := company.SupportEmail
		if at := strings.LastIndex(domain, "@"); at >= 0 {
			domain = domain[at+1:]
		}
		_, _ = s.store.EnsureCTIAsset(ctx, company.ID, "email", company.SupportEmail, "Support inbox", "high", "company profile")
		if domain != "" {
			_, _ = s.store.EnsureCTIAsset(ctx, company.ID, "domain", strings.ToLower(domain), "Email domain", "high", "company profile")
		}
	}
	return s.store.ListCTIAssets(ctx, company.ID)
}

func (s *Server) ctiProviders(ctx context.Context, companyID string) cti.Providers {
	providers := cti.Providers{}
	if config, err := s.loadIntegration(ctx, companyID, "virustotal"); err == nil {
		providers.VirusTotal = config["api_key"]
	}
	if config, err := s.loadIntegration(ctx, companyID, "abuseipdb"); err == nil {
		providers.AbuseIPDB = config["api_key"]
	}
	if config, err := s.loadIntegration(ctx, companyID, "otx"); err == nil {
		providers.OTX = config["api_key"]
	}
	if config, err := s.loadIntegration(ctx, companyID, "shodan"); err == nil {
		providers.Shodan = config["api_key"]
	}
	return providers
}

func (s *Server) securityData(ctx context.Context, company *store.Company) (securityPayload, error) {
	assets, err := s.store.ListCTIAssets(ctx, company.ID)
	if err != nil {
		return securityPayload{}, err
	}
	indicators, _ := s.store.ListCTIIndicators(ctx, company.ID)
	actors, _ := s.store.ListCTIActors(ctx, company.ID)
	reports, _ := s.store.ListCTIReports(ctx, company.ID)
	relationships, _ := s.store.ListCTIRelationships(ctx, company.ID)
	findings, _ := s.store.ListCTIFindings(ctx, company.ID)
	assessment, _ := s.store.GetCTIAssessment(ctx, company.ID)

	counts := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0}
	for _, finding := range findings {
		counts[finding.Severity]++
	}
	active := 0
	for _, indicator := range indicators {
		if indicator.Status == "active" {
			active++
		}
	}

	providers := []securityProvider{
		{Key: "virustotal", Name: "VirusTotal", URL: "https://www.virustotal.com/gui/my-apikey", Docs: "https://docs.virustotal.com/reference/overview"},
		{Key: "abuseipdb", Name: "AbuseIPDB", URL: "https://www.abuseipdb.com/account/api", Docs: "https://docs.abuseipdb.com/"},
		{Key: "otx", Name: "AlienVault OTX", URL: "https://otx.alienvault.com/api", Docs: "https://docs.alienvault.com/docs/v2/"},
		{Key: "shodan", Name: "Shodan", URL: "https://account.shodan.io/", Docs: "https://developer.shodan.io/api"},
	}
	for i := range providers {
		if _, err := s.loadIntegration(ctx, company.ID, providers[i].Key); err == nil {
			providers[i].Configured = true
		}
	}

	var scored *store.CTIAssessment
	scannedAt := ""
	if assessment.Analysis != nil {
		scored = &assessment
		scannedAt = assessment.CreatedAt.Format(time.RFC3339)
	}

	return securityPayload{
		Assets: assets, Indicators: indicators, Actors: actors, Reports: reports,
		Relationships: relationships, Findings: findings, Assessment: scored,
		Providers: providers,
		Summary: map[string]any{
			"severity": counts, "active_indicators": active, "total_indicators": len(indicators),
			"assets": len(assets), "findings": len(findings),
		},
		ScannedAt: scannedAt,
	}, nil
}

func (s *Server) assessSecurity(ctx context.Context, userID string, company *store.Company, payload securityPayload) *store.CTIAssessment {
	if blocked, err := s.checkBudget(ctx, userID, company.ID, ""); err != nil || blocked != "" {
		return nil
	}
	model, err := marketingModelFor(s, ctx, userID)
	if err != nil {
		return nil
	}

	contextJSON, _ := json.Marshal(map[string]any{
		"company": company.Name, "country": company.Country, "assets": payload.Assets,
		"findings": payload.Findings, "indicators": payload.Indicators,
	})

	system := `You are a defensive cyber threat intelligence analyst. Use only the supplied findings.
Return valid JSON only:
{"threat_level":"low|guarded|elevated|high|critical","exposure_score":0,
"summary":"2 sentences","risks":[{"title":"","severity":"","why":"","fix":""}],
"actions":[""],"business_impact":""}
Give at most 4 risks and 5 actions. Do not invent incidents or claim a breach.`

	assessmentCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	result, err := llm.CompleteResult(assessmentCtx, llm.Config{
		Kind: model.Provider.Kind, BaseURL: model.Provider.BaseURL, Model: model.Model, APIKey: model.APIKey,
	}, system, "Security data:\n"+string(contextJSON), 1200)
	if err != nil {
		return nil
	}
	document, err := extractJSONObject(result.Text)
	if err != nil {
		return nil
	}
	_ = s.store.SaveCTIAssessment(ctx, company.ID, document, model.Provider.Name+" · "+model.Model)
	_ = s.saveUsage(ctx, userID, company.ID, "", "", model, result.Usage)
	return &store.CTIAssessment{Analysis: document, Model: model.Provider.Name + " · " + model.Model, CreatedAt: time.Now().UTC()}
}

func (s *Server) handleSecurityStix(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	payload, err := s.securityData(r.Context(), company)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't build the STIX bundle.")
		return
	}

	objects := []map[string]any{{
		"type": "identity", "spec_version": "2.1", "id": stixID("identity", company.ID),
		"created": time.Now().UTC().Format(time.RFC3339), "modified": time.Now().UTC().Format(time.RFC3339),
		"name": company.Name, "identity_class": "organization",
	}}
	for _, indicator := range payload.Indicators {
		pattern := stixPattern(indicator.Type, indicator.Value)
		objects = append(objects, map[string]any{
			"type": "indicator", "spec_version": "2.1", "id": stixID("indicator", indicator.ID),
			"created": indicator.CreatedAt.Format(time.RFC3339), "modified": indicator.CreatedAt.Format(time.RFC3339),
			"name": indicator.Name, "description": indicator.Description, "pattern": pattern,
			"pattern_type": "stix", "valid_from": indicator.FirstSeen, "confidence": indicator.Confidence,
			"labels": indicator.Tags,
		})
	}
	for _, report := range payload.Reports {
		objects = append(objects, map[string]any{
			"type": "report", "spec_version": "2.1", "id": stixID("report", report.ID),
			"created": report.CreatedAt.Format(time.RFC3339), "modified": report.CreatedAt.Format(time.RFC3339),
			"name": report.Title, "description": report.Summary, "published": report.PublishedAt,
			"confidence": report.Confidence, "labels": report.Tags, "object_refs": []string{},
		})
	}
	for _, actor := range payload.Actors {
		objects = append(objects, map[string]any{
			"type": "threat-actor", "spec_version": "2.1", "id": stixID("threat-actor", actor.ID),
			"created": actor.CreatedAt.Format(time.RFC3339), "modified": actor.CreatedAt.Format(time.RFC3339),
			"name": actor.Name, "description": actor.Description, "aliases": actor.Aliases,
			"threat_actor_types": []string{actor.Motivation}, "confidence": actor.Confidence,
		})
	}

	bundle := map[string]any{
		"type": "bundle", "id": stixID("bundle", company.ID), "spec_version": "2.1", "objects": objects,
	}
	w.Header().Set("Content-Type", "application/json")
	short := company.ID
	if len(short) > 8 {
		short = short[:8]
	}
	w.Header().Set("Content-Disposition", `attachment; filename="crew-threat-intel-`+short+`.json"`)
	_ = json.NewEncoder(w).Encode(bundle)
}

func stixID(kind, value string) string {
	value = strings.ReplaceAll(value, "-", "")
	for len(value) < 32 {
		value += "0"
	}
	value = value[:32]
	return fmt.Sprintf("%s--%s-%s-%s-%s-%s", kind, value[:8], value[8:12], value[12:16], value[16:20], value[20:32])
}

func stixPattern(indicatorType, value string) string {
	switch indicatorType {
	case "ip":
		return "[ipv4-addr:value = '" + strings.ReplaceAll(value, "'", "") + "']"
	case "domain":
		return "[domain-name:value = '" + strings.ReplaceAll(value, "'", "") + "']"
	case "url":
		return "[url:value = '" + strings.ReplaceAll(value, "'", "") + "']"
	case "email":
		return "[email-addr:value = '" + strings.ReplaceAll(value, "'", "") + "']"
	case "md5", "sha1", "sha256":
		return "[file:hashes.'" + strings.ToUpper(indicatorType) + "' = '" + strings.ReplaceAll(value, "'", "") + "']"
	default:
		return "[x-crew-observable:value = '" + strings.ReplaceAll(value, "'", "") + "']"
	}
}
