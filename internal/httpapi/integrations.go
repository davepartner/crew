package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var integrationKinds = []string{"whatsapp", "meta_ads", "tiktok_ads", "google_ads", "brave_search"}

func integrationFields(kind string) []string {
	switch kind {
	case "whatsapp":
		return []string{"phone_number_id", "token", "default_to"}
	case "meta_ads":
		return []string{"account_id", "access_token"}
	case "tiktok_ads":
		return []string{"advertiser_id", "access_token"}
	case "google_ads":
		return []string{"customer_id", "developer_token", "client_id", "client_secret", "refresh_token", "login_customer_id"}
	case "brave_search":
		return []string{"api_key"}
	}
	return nil
}

func (s *Server) handleListIntegrations(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	out := make([]map[string]any, 0, len(integrationKinds))
	for _, kind := range integrationKinds {
		config, err := s.loadIntegration(r.Context(), company.ID, kind)
		if err != nil {
			out = append(out, map[string]any{"kind": kind, "configured": false, "details": map[string]string{}})
			continue
		}
		out = append(out, map[string]any{
			"kind": kind, "configured": integrationReady(kind, config), "details": publicIntegration(kind, config),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "integrations": out,
		"email": map[string]any{"available": strings.TrimSpace(s.cfg.SMTPFrom) != "", "from": s.cfg.SMTPFrom},
	})
}

func (s *Server) handleSaveIntegration(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	kind := r.PathValue("kind")
	if !validIntegration(kind) {
		writeError(w, http.StatusNotFound, "Unknown integration.")
		return
	}

	var body map[string]string
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, _ := s.loadIntegration(r.Context(), company.ID, kind)
	if existing == nil {
		existing = map[string]string{}
	}
	for _, field := range integrationFields(kind) {
		if value := strings.TrimSpace(body[field]); value != "" {
			existing[field] = value
		}
	}

	if !integrationReady(kind, existing) {
		writeError(w, http.StatusUnprocessableEntity, "Fill in all required fields for this integration.")
		return
	}

	encoded, _ := json.Marshal(existing)
	encrypted, err := s.cipher.Encrypt(string(encoded))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that integration.")
		return
	}
	if err := s.store.SaveIntegration(r.Context(), company.ID, kind, encrypted); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that integration.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "integration": map[string]any{
			"kind": kind, "configured": true, "details": publicIntegration(kind, existing),
		},
	})
}

func (s *Server) handleDeleteIntegration(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	kind := r.PathValue("kind")
	if !validIntegration(kind) {
		writeError(w, http.StatusNotFound, "Unknown integration.")
		return
	}
	_ = s.store.DeleteIntegration(r.Context(), company.ID, kind)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleTestIntegration(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	kind := r.PathValue("kind")

	config, err := s.loadIntegration(r.Context(), company.ID, kind)
	if err != nil {
		writeError(w, http.StatusNotFound, "Connect this integration first.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	detail, err := s.testIntegration(ctx, kind, config)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "detail": detail, "details": publicIntegration(kind, config)})
}

func (s *Server) handleIntegrationOverview(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	kind := r.PathValue("kind")

	config, err := s.loadIntegration(r.Context(), company.ID, kind)
	if err != nil {
		writeError(w, http.StatusNotFound, "Connect this integration first.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	data, err := s.integrationOverview(ctx, kind, config)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "overview": data})
}

func (s *Server) loadIntegration(ctx context.Context, companyID, kind string) (map[string]string, error) {
	row, err := s.store.GetIntegration(ctx, companyID, kind)
	if err != nil {
		return nil, err
	}
	plain, err := s.cipher.Decrypt(row.Config)
	if err != nil {
		return nil, err
	}
	var config map[string]string
	if err := json.Unmarshal([]byte(plain), &config); err != nil {
		return nil, err
	}
	return config, nil
}

func validIntegration(kind string) bool {
	for _, item := range integrationKinds {
		if item == kind {
			return true
		}
	}
	return false
}

func integrationReady(kind string, config map[string]string) bool {
	for _, field := range integrationFields(kind) {
		if field == "default_to" || field == "login_customer_id" {
			continue
		}
		if strings.TrimSpace(config[field]) == "" {
			return false
		}
	}
	return true
}

func publicIntegration(kind string, config map[string]string) map[string]string {
	out := map[string]string{}
	for _, field := range integrationFields(kind) {
		if field == "token" || field == "access_token" || field == "api_key" || field == "client_secret" || strings.Contains(field, "secret") {
			continue
		}
		if config[field] != "" {
			out[field] = config[field]
		}
	}
	return out
}

func (s *Server) testIntegration(ctx context.Context, kind string, config map[string]string) (map[string]any, error) {
	switch kind {
	case "whatsapp":
		var payload map[string]any
		endpoint := "https://graph.facebook.com/v20.0/" + url.PathEscape(config["phone_number_id"]) +
			"?fields=display_phone_number,verified_name&access_token=" + url.QueryEscape(config["token"])
		if err := getJSON(ctx, endpoint, nil, &payload); err != nil {
			return nil, fmt.Errorf("WhatsApp rejected those credentials: %w", err)
		}
		return payload, nil
	case "meta_ads":
		return s.metaOverview(ctx, config)
	case "tiktok_ads":
		return s.tiktokOverview(ctx, config)
	case "google_ads":
		return s.googleAdsOverview(ctx, config)
	case "brave_search":
		var payload map[string]any
		endpoint := "https://api.search.brave.com/res/v1/web/search?q=paytrackify&count=1"
		if err := getJSON(ctx, endpoint, map[string]string{"X-Subscription-Token": config["api_key"], "Accept": "application/json"}, &payload); err != nil {
			return nil, fmt.Errorf("Brave Search rejected that key: %w", err)
		}
		return map[string]any{"connected": true}, nil
	}
	return nil, fmt.Errorf("unsupported integration")
}

func (s *Server) integrationOverview(ctx context.Context, kind string, config map[string]string) (map[string]any, error) {
	switch kind {
	case "whatsapp":
		return s.testIntegration(ctx, kind, config)
	case "meta_ads":
		return s.metaOverview(ctx, config)
	case "tiktok_ads":
		return s.tiktokOverview(ctx, config)
	case "google_ads":
		return s.googleAdsOverview(ctx, config)
	}
	return map[string]any{"connected": true}, nil
}

func (s *Server) metaOverview(ctx context.Context, config map[string]string) (map[string]any, error) {
	account := strings.TrimPrefix(config["account_id"], "act_")
	endpoint := "https://graph.facebook.com/v20.0/act_" + url.PathEscape(account) +
		"/campaigns?fields=name,status,objective&limit=10&access_token=" + url.QueryEscape(config["access_token"])
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := getJSON(ctx, endpoint, nil, &payload); err != nil {
		return nil, fmt.Errorf("Meta rejected that account: %w", err)
	}
	return map[string]any{"platform": "meta", "account_id": account, "campaigns": payload.Data}, nil
}

func (s *Server) tiktokOverview(ctx context.Context, config map[string]string) (map[string]any, error) {
	endpoint := "https://business-api.tiktok.com/open_api/v1.3/campaign/get/?advertiser_id=" +
		url.QueryEscape(config["advertiser_id"]) + "&page_size=10"
	var payload struct {
		Data struct {
			List []map[string]any `json:"list"`
		} `json:"data"`
	}
	if err := getJSON(ctx, endpoint, map[string]string{"Access-Token": config["access_token"]}, &payload); err != nil {
		return nil, fmt.Errorf("TikTok rejected that advertiser: %w", err)
	}
	return map[string]any{"platform": "tiktok", "advertiser_id": config["advertiser_id"], "campaigns": payload.Data.List}, nil
}

func (s *Server) googleAdsOverview(ctx context.Context, config map[string]string) (map[string]any, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", config["client_id"])
	form.Set("client_secret", config["client_secret"])
	form.Set("refresh_token", config["refresh_token"])

	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("Google OAuth failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google OAuth rejected those credentials")
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil || token.AccessToken == "" {
		return nil, fmt.Errorf("Google OAuth returned no token")
	}

	endpoint := "https://googleads.googleapis.com/v21/customers:listAccessibleCustomers"
	headers := map[string]string{
		"Authorization":   "Bearer " + token.AccessToken,
		"developer-token": config["developer_token"],
	}
	if config["login_customer_id"] != "" {
		headers["login-customer-id"] = strings.ReplaceAll(config["login_customer_id"], "-", "")
	}

	var payload map[string]any
	if err := getJSON(ctx, endpoint, headers, &payload); err != nil {
		return nil, fmt.Errorf("Google Ads rejected that developer token: %w", err)
	}
	return map[string]any{"platform": "google", "customers": payload["resourceNames"]}, nil
}

func getJSON(ctx context.Context, endpoint string, headers map[string]string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "Crew/1.0")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	return json.Unmarshal(body, target)
}
