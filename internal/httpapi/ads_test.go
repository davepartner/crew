package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/davepartner/crew/internal/config"
	"github.com/davepartner/crew/internal/payauth"
	"github.com/davepartner/crew/internal/secret"
	"github.com/davepartner/crew/internal/store"
)

func TestAdsGenerationEndToEnd(t *testing.T) {
	campaign := map[string]any{
		"campaign_name": "Clinic books",
		"big_idea":      "Stop losing weekends to receipts.",
		"strategy":      map[string]any{"objective": "leads", "audience_summary": "clinic owners", "offer": "from $249", "positioning": "done-for-you books", "proof": []string{}, "unknowns": []string{}},
		"platforms": map[string]any{
			"meta": map[string]any{"concepts": []map[string]any{{
				"name": "Weekend back", "angle": "time", "hook": "Still doing receipts on Sunday?",
				"primary_text": "We reconcile Xero and chase receipts.", "headline": "Get your weekends back",
				"description": "Bookkeeping for clinics", "cta": "Learn more", "format": "Reels",
				"placement": "Instagram Reels", "visual": "Founder at a kitchen table", "script_beats": []string{"Hook"},
			}}, "notes": []string{}},
		},
		"hooks":        []map[string]any{{"text": "Still doing receipts on Sunday?", "category": "question", "platform": "meta", "why": "relatable"}},
		"testing_plan": []map[string]any{{"priority": 1, "element": "hook", "a": "A", "b": "B", "hypothesis": "time wins"}},
		"budget_split": []map[string]any{{"platform": "meta", "percent": 100, "why": "only requested"}},
		"compliance":   map[string]any{"claims_to_verify": []string{}, "avoid": []string{}, "spec_checks": []string{"verify in Ads Manager"}},
		"scores":       map[string]int{"hook": 8, "clarity": 8, "platform_fit": 8, "distinctiveness": 7, "proof": 4, "cta": 7, "compliance": 8, "overall": 7},
	}
	encoded, _ := json.Marshal(campaign)

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": string(encoded)}}},
			"usage":   map[string]int{"prompt_tokens": 2000, "completion_tokens": 1500},
		})
	}))
	defer modelServer.Close()

	ctx := context.Background()
	st, err := store.Open(t.TempDir() + "/crew.db")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	user, _ := st.UpsertUser(ctx, "ads-user", "ads@example.com", "Ads")
	company, _ := st.CreateCompany(ctx, user.ID, store.CompanyInput{Name: "Ads Co", BusinessKey: "custom"})

	cipher, _ := secret.New(bytes.Repeat([]byte{8}, 32))
	server := New(config.Config{
		TaskLease: time.Hour, SessionCacheTTL: time.Minute,
		PlatformProvider: "deepseek", PlatformModel: "deepseek-chat",
		PlatformBaseURL: modelServer.URL, PlatformAPIKey: "sk-ads",
	}, st, payauth.NewClient("http://127.0.0.1:1", time.Second), cipher)

	body, _ := json.Marshal(map[string]any{
		"product": "Bookkeeping for clinics", "audience": "clinic owners",
		"platforms": []string{"meta"}, "objective": "leads",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/ads/generate", bytes.NewReader(body))
	request = request.WithContext(context.WithValue(request.Context(), identityKey, &identity{
		UserID: user.ID, Email: user.Email, Name: user.Name,
	}))
	request = request.WithContext(context.WithValue(request.Context(), companyKey, &company))

	recorder := httptest.NewRecorder()
	server.handleGenerateAds(recorder, request)

	if recorder.Code != 201 {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Campaign store.AdCampaign `json:"campaign"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Campaign.Title != "Clinic books" || !strings.Contains(string(response.Campaign.Result), "Stop losing weekends") {
		t.Fatalf("campaign: %+v", response.Campaign)
	}

	saved, err := st.GetAdCampaign(ctx, company.ID, response.Campaign.ID)
	if err != nil || saved.Title != "Clinic books" {
		t.Fatalf("saved: %+v %v", saved, err)
	}

	spend, _ := st.CompanySpend(ctx, company.ID, store.MonthStart())
	if spend <= 0 {
		t.Fatal("spend not recorded")
	}
}

func TestExtractJSONObjectHandlesFences(t *testing.T) {
	raw := "Here you go:\n```json\n{\"campaign_name\":\"X\"}\n```\n"
	document, err := extractJSONObject(raw)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !json.Valid(document) || !strings.Contains(string(document), "campaign_name") {
		t.Fatalf("document: %s", document)
	}

	if _, err := extractJSONObject("not json"); err == nil {
		t.Fatal("expected error")
	}
}
