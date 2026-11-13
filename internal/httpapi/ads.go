package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/adplaybooks"
	"github.com/davepartner/crew/internal/llm"
)

type adsBrief struct {
	Title     string   `json:"title"`
	Product   string   `json:"product"`
	Audience  string   `json:"audience"`
	Problem   string   `json:"problem"`
	Proof     string   `json:"proof"`
	Price     string   `json:"price"`
	URL       string   `json:"url"`
	Objective string   `json:"objective"`
	Tone      string   `json:"tone"`
	Extra     string   `json:"extra"`
	Platforms []string `json:"platforms"`
}

func (s *Server) handleGenerateAds(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	var brief adsBrief
	if err := decodeJSON(r, &brief); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	brief.Title = trim(brief.Title, 120)
	brief.Product = trim(brief.Product, 2000)
	brief.Audience = trim(brief.Audience, 1000)
	brief.Problem = trim(brief.Problem, 1000)
	brief.Proof = trim(brief.Proof, 1500)
	brief.Price = trim(brief.Price, 200)
	brief.URL = trim(brief.URL, 500)
	brief.Objective = strings.TrimSpace(brief.Objective)
	brief.Tone = trim(brief.Tone, 80)
	brief.Extra = trim(brief.Extra, 1500)

	if brief.Product == "" || brief.Audience == "" {
		writeError(w, http.StatusUnprocessableEntity, "Tell us what you sell and who it is for.")
		return
	}
	if brief.Objective == "" {
		brief.Objective = "sales"
	}
	if brief.Tone == "" {
		brief.Tone = "direct and human"
	}

	seen := map[string]bool{}
	platforms := make([]string, 0, len(brief.Platforms))
	for _, key := range brief.Platforms {
		if seen[key] {
			continue
		}
		if _, ok := adplaybooks.ByKey(key); !ok {
			writeError(w, http.StatusUnprocessableEntity, "Choose Meta, TikTok or YouTube.")
			return
		}
		seen[key] = true
		platforms = append(platforms, key)
	}
	if len(platforms) == 0 {
		platforms = []string{"meta", "tiktok", "youtube"}
	}
	brief.Platforms = platforms

	model, err := marketingModelFor(s, r.Context(), id.UserID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"ok": false, "code": "no_model",
			"error": "Add a model key in Settings → AI models, or use the included model.",
		})
		return
	}

	if blocked, err := s.checkBudget(r.Context(), id.UserID, company.ID, ""); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't check the AI budget.")
		return
	} else if blocked != "" {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"ok": false, "code": "budget", "error": blocked})
		return
	}

	payload, _ := json.MarshalIndent(brief, "", "  ")
	userPrompt := "Campaign brief:\n" + string(payload) +
		"\n\nRequested platforms: " + strings.Join(platforms, ", ") +
		"\n\nWrite the campaign now. JSON only."

	ctx, cancel := contextWithTimeout(r, 180*time.Second)
	defer cancel()

	result, err := llm.CompleteResult(ctx, llm.Config{
		Kind:    model.Provider.Kind,
		BaseURL: model.Provider.BaseURL,
		Model:   model.Model,
		APIKey:  model.APIKey,
	}, adplaybooks.SystemPrompt(), userPrompt, 6000)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The model couldn't write that campaign: "+err.Error())
		return
	}

	document, err := extractJSONObject(result.Text)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The model returned an unreadable campaign. Try again.")
		return
	}

	var parsed struct {
		CampaignName string `json:"campaign_name"`
	}
	_ = json.Unmarshal(document, &parsed)

	title := trim(parsed.CampaignName, 120)
	if title == "" {
		title = brief.Title
	}
	if title == "" {
		title = "Ad campaign"
	}

	cost := s.saveUsage(ctx, id.UserID, company.ID, "", "", model, result.Usage)

	campaign, err := s.store.CreateAdCampaign(r.Context(), id.UserID, company.ID, title, mustJSON(brief), document,
		model.Provider.Name+" · "+model.Model, cost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that campaign. Please try again.")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "campaign": campaign})
}

func (s *Server) handleListAdCampaigns(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	campaigns, err := s.store.ListAdCampaigns(r.Context(), company.ID, 30)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your campaigns.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "campaigns": campaigns})
}

func (s *Server) handleGetAdCampaign(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	campaign, err := s.store.GetAdCampaign(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That campaign doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "campaign": campaign})
}

func (s *Server) handleDeleteAdCampaign(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	if err := s.store.DeleteAdCampaign(r.Context(), company.ID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "That campaign doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func extractJSONObject(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	value = strings.TrimSpace(value)

	start := strings.Index(value, "{")
	end := strings.LastIndex(value, "}")
	if start < 0 || end <= start {
		return nil, errors.New("no json object")
	}

	document := []byte(value[start : end+1])
	if !json.Valid(document) {
		return nil, errors.New("invalid json")
	}
	return document, nil
}

func mustJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return raw
}
