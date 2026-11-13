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

type adCampaign struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Budget   float64 `json:"budget"`
	Currency string  `json:"currency"`
}

func (s *Server) handleAdAccounts(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	accounts := make([]map[string]any, 0, 3)

	for _, platform := range []string{"meta", "tiktok", "google"} {
		kind := platform + "_ads"
		config, err := s.loadIntegration(r.Context(), company.ID, kind)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		campaigns, err := s.listAdCampaigns(ctx, platform, config)
		cancel()

		item := map[string]any{"platform": platform, "configured": true, "campaigns": campaigns}
		if err != nil {
			item["error"] = err.Error()
		}
		accounts = append(accounts, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "accounts": accounts})
}

func (s *Server) handleCreateAdCampaign(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	platform := r.PathValue("platform")
	config, err := s.loadIntegration(r.Context(), company.ID, platform+"_ads")
	if err != nil {
		writeError(w, http.StatusConflict, "Connect that ad account first.")
		return
	}

	var body struct {
		Name   string  `json:"name"`
		Budget float64 `json:"budget"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if trim(body.Name, 120) == "" || body.Budget <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "Add a campaign name and a daily budget.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	id, err := s.createAdCampaign(ctx, platform, config, trim(body.Name, 120), body.Budget)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id, "status": "PAUSED"})
}

func (s *Server) handleAdCampaignStatus(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	platform := r.PathValue("platform")
	config, err := s.loadIntegration(r.Context(), company.ID, platform+"_ads")
	if err != nil {
		writeError(w, http.StatusConflict, "Connect that ad account first.")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := strings.ToUpper(trim(body.Status, 20))
	if status != "PAUSED" && status != "ACTIVE" && status != "ENABLE" && status != "DISABLE" {
		writeError(w, http.StatusUnprocessableEntity, "Status must be pause or enable.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	if err := s.setAdCampaignStatus(ctx, platform, config, r.PathValue("id"), status); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdCampaignBudget(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	platform := r.PathValue("platform")
	config, err := s.loadIntegration(r.Context(), company.ID, platform+"_ads")
	if err != nil {
		writeError(w, http.StatusConflict, "Connect that ad account first.")
		return
	}

	var body struct {
		Budget float64 `json:"budget"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Budget <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "Add a daily budget above zero.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	if err := s.setAdCampaignBudget(ctx, platform, config, r.PathValue("id"), body.Budget); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) listAdCampaigns(ctx context.Context, platform string, config map[string]string) ([]adCampaign, error) {
	switch platform {
	case "meta":
		account := strings.TrimPrefix(config["account_id"], "act_")
		endpoint := "https://graph.facebook.com/v20.0/act_" + url.PathEscape(account) +
			"/campaigns?fields=id,name,status,objective,daily_budget&limit=25&access_token=" + url.QueryEscape(config["access_token"])
		var payload struct {
			Data []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Status      string `json:"status"`
				DailyBudget string `json:"daily_budget"`
			} `json:"data"`
		}
		if err := getJSON(ctx, endpoint, nil, &payload); err != nil {
			return nil, fmt.Errorf("Meta: %w", err)
		}
		out := make([]adCampaign, 0, len(payload.Data))
		for _, item := range payload.Data {
			var cents float64
			fmt.Sscanf(item.DailyBudget, "%f", &cents)
			out = append(out, adCampaign{ID: item.ID, Name: item.Name, Status: item.Status, Budget: cents / 100, Currency: "account currency"})
		}
		return out, nil

	case "tiktok":
		endpoint := "https://business-api.tiktok.com/open_api/v1.3/campaign/get/?advertiser_id=" +
			url.QueryEscape(config["advertiser_id"]) + "&page_size=25"
		var payload struct {
			Data struct {
				List []struct {
					ID     string  `json:"campaign_id"`
					Name   string  `json:"campaign_name"`
					Status string  `json:"operation_status"`
					Budget float64 `json:"budget"`
				} `json:"list"`
			} `json:"data"`
		}
		if err := getJSON(ctx, endpoint, map[string]string{"Access-Token": config["access_token"]}, &payload); err != nil {
			return nil, fmt.Errorf("TikTok: %w", err)
		}
		out := make([]adCampaign, 0, len(payload.Data.List))
		for _, item := range payload.Data.List {
			out = append(out, adCampaign{ID: item.ID, Name: item.Name, Status: item.Status, Budget: item.Budget, Currency: "account currency"})
		}
		return out, nil

	case "google":
		accessToken, err := googleAccessToken(ctx, config)
		if err != nil {
			return nil, err
		}
		customer := strings.ReplaceAll(config["customer_id"], "-", "")
		query := "SELECT campaign.id, campaign.name, campaign.status FROM campaign ORDER BY campaign.id LIMIT 25"
		var payload struct {
			Results []struct {
				Campaign struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Status string `json:"status"`
				} `json:"campaign"`
			} `json:"results"`
		}
		if err := googleRequest(ctx, config, accessToken, customer, "customers/"+customer+"/googleAds:search",
			map[string]string{"query": query}, &payload); err != nil {
			return nil, fmt.Errorf("Google Ads: %w", err)
		}
		out := make([]adCampaign, 0, len(payload.Results))
		for _, item := range payload.Results {
			out = append(out, adCampaign{ID: item.Campaign.ID, Name: item.Campaign.Name, Status: item.Campaign.Status, Currency: "account currency"})
		}
		return out, nil
	}

	return nil, fmt.Errorf("unsupported platform")
}

func (s *Server) createAdCampaign(ctx context.Context, platform string, config map[string]string, name string, budget float64) (string, error) {
	switch platform {
	case "meta":
		account := strings.TrimPrefix(config["account_id"], "act_")
		form := url.Values{}
		form.Set("name", name)
		form.Set("objective", "OUTCOME_TRAFFIC")
		form.Set("status", "PAUSED")
		form.Set("special_ad_categories", "[]")
		form.Set("daily_budget", fmt.Sprintf("%.0f", budget*100))
		form.Set("access_token", config["access_token"])
		var payload struct {
			ID string `json:"id"`
		}
		if err := postFormJSON(ctx, "https://graph.facebook.com/v20.0/act_"+url.PathEscape(account)+"/campaigns", form, &payload); err != nil {
			return "", fmt.Errorf("Meta rejected the campaign: %w", err)
		}
		return payload.ID, nil

	case "tiktok":
		body := map[string]any{
			"advertiser_id":    config["advertiser_id"],
			"campaign_name":    name,
			"objective_type":   "TRAFFIC",
			"budget_mode":      "BUDGET_MODE_DAY",
			"budget":           budget,
			"operation_status": "DISABLE",
		}
		var payload struct {
			Data struct {
				CampaignID string `json:"campaign_id"`
			} `json:"data"`
		}
		if err := postJSON(ctx, "https://business-api.tiktok.com/open_api/v1.3/campaign/create/",
			map[string]string{"Access-Token": config["access_token"]}, body, &payload); err != nil {
			return "", fmt.Errorf("TikTok rejected the campaign: %w", err)
		}
		return payload.Data.CampaignID, nil

	case "google":
		accessToken, err := googleAccessToken(ctx, config)
		if err != nil {
			return "", err
		}
		customer := strings.ReplaceAll(config["customer_id"], "-", "")

		var budgetPayload struct {
			Results []struct {
				ResourceName string `json:"resourceName"`
			} `json:"results"`
		}
		budgetBody := map[string]any{
			"operations": []map[string]any{{
				"create": map[string]any{
					"name": "Crew " + name, "amountMicros": fmt.Sprintf("%.0f", budget*1_000_000),
					"deliveryMethod": "STANDARD", "explicitlyShared": false,
				},
			}},
		}
		if err := googleRequest(ctx, config, accessToken, customer, "customers/"+customer+"/campaignBudgets:mutate", budgetBody, &budgetPayload); err != nil {
			return "", fmt.Errorf("Google Ads budget: %w", err)
		}
		if len(budgetPayload.Results) == 0 {
			return "", fmt.Errorf("Google Ads returned no budget")
		}

		var campaignPayload struct {
			Results []struct {
				ResourceName string `json:"resourceName"`
			} `json:"results"`
		}
		campaignBody := map[string]any{
			"operations": []map[string]any{{
				"create": map[string]any{
					"name": name, "status": "PAUSED", "advertisingChannelType": "SEARCH",
					"campaignBudget": budgetPayload.Results[0].ResourceName, "manualCpc": map[string]any{},
				},
			}},
		}
		if err := googleRequest(ctx, config, accessToken, customer, "customers/"+customer+"/campaigns:mutate", campaignBody, &campaignPayload); err != nil {
			return "", fmt.Errorf("Google Ads campaign: %w", err)
		}
		if len(campaignPayload.Results) == 0 {
			return "", fmt.Errorf("Google Ads returned no campaign")
		}
		return strings.TrimPrefix(campaignPayload.Results[0].ResourceName, "customers/"+customer+"/campaigns/"), nil
	}

	return "", fmt.Errorf("unsupported platform")
}

func (s *Server) setAdCampaignStatus(ctx context.Context, platform string, config map[string]string, id, status string) error {
	switch platform {
	case "meta":
		form := url.Values{}
		form.Set("status", status)
		form.Set("access_token", config["access_token"])
		var payload map[string]any
		if err := postFormJSON(ctx, "https://graph.facebook.com/v20.0/"+url.PathEscape(id), form, &payload); err != nil {
			return fmt.Errorf("Meta rejected the status: %w", err)
		}
		return nil

	case "tiktok":
		operation := "ENABLE"
		if status == "PAUSED" || status == "DISABLE" {
			operation = "DISABLE"
		}
		body := map[string]any{
			"advertiser_id":    config["advertiser_id"],
			"campaign_ids":     []string{id},
			"operation_status": operation,
		}
		var payload map[string]any
		if err := postJSON(ctx, "https://business-api.tiktok.com/open_api/v1.3/campaign/update/",
			map[string]string{"Access-Token": config["access_token"]}, body, &payload); err != nil {
			return fmt.Errorf("TikTok rejected the status: %w", err)
		}
		return nil

	case "google":
		accessToken, err := googleAccessToken(ctx, config)
		if err != nil {
			return err
		}
		customer := strings.ReplaceAll(config["customer_id"], "-", "")
		googleStatus := status
		if status == "ACTIVE" {
			googleStatus = "ENABLED"
		}
		body := map[string]any{
			"operations": []map[string]any{{
				"update": map[string]any{
					"resourceName": "customers/" + customer + "/campaigns/" + id,
					"status":       googleStatus,
				},
				"updateMask": "status",
			}},
		}
		var payload map[string]any
		if err := googleRequest(ctx, config, accessToken, customer, "customers/"+customer+"/campaigns:mutate", body, &payload); err != nil {
			return fmt.Errorf("Google Ads rejected the status: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unsupported platform")
}

func (s *Server) setAdCampaignBudget(ctx context.Context, platform string, config map[string]string, id string, budget float64) error {
	switch platform {
	case "meta":
		form := url.Values{}
		form.Set("daily_budget", fmt.Sprintf("%.0f", budget*100))
		form.Set("access_token", config["access_token"])
		var payload map[string]any
		if err := postFormJSON(ctx, "https://graph.facebook.com/v20.0/"+url.PathEscape(id), form, &payload); err != nil {
			return fmt.Errorf("Meta rejected the budget: %w", err)
		}
		return nil

	case "tiktok":
		body := map[string]any{
			"advertiser_id": config["advertiser_id"],
			"campaign_ids":  []string{id},
			"budget":        budget,
		}
		var payload map[string]any
		if err := postJSON(ctx, "https://business-api.tiktok.com/open_api/v1.3/campaign/update/",
			map[string]string{"Access-Token": config["access_token"]}, body, &payload); err != nil {
			return fmt.Errorf("TikTok rejected the budget: %w", err)
		}
		return nil
	}
	return fmt.Errorf("budget updates are available for Meta and TikTok")
}

func postFormJSON(ctx context.Context, endpoint string, form url.Values, target any) error {
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
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

func postJSON(ctx context.Context, endpoint string, headers map[string]string, payload any, target any) error {
	encoded, _ := json.Marshal(payload)
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(encoded)))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
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

func googleAccessToken(ctx context.Context, config map[string]string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", config["client_id"])
	form.Set("client_secret", config["client_secret"])
	form.Set("refresh_token", config["refresh_token"])
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := (&http.Client{Timeout: 15 * time.Second}).Do(request)
	if err != nil {
		return "", fmt.Errorf("Google OAuth failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google OAuth rejected those credentials")
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil || token.AccessToken == "" {
		return "", fmt.Errorf("Google OAuth returned no token")
	}
	return token.AccessToken, nil
}

func googleRequest(ctx context.Context, config map[string]string, accessToken, customer, path string, payload any, target any) error {
	encoded, _ := json.Marshal(payload)
	endpoint := "https://googleads.googleapis.com/v21/" + path
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(encoded)))
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("developer-token", config["developer_token"])
	request.Header.Set("Content-Type", "application/json")
	if config["login_customer_id"] != "" {
		request.Header.Set("login-customer-id", strings.ReplaceAll(config["login_customer_id"], "-", ""))
	}
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
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
