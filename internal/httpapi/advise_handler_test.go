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
	"github.com/davepartner/crew/internal/modelcatalog"
	"github.com/davepartner/crew/internal/payauth"
	"github.com/davepartner/crew/internal/secret"
	"github.com/davepartner/crew/internal/store"
)

func TestStudyCompanyEndToEnd(t *testing.T) {
	analysis := `{
		"headline":"You have an offer but no repeatable acquisition.",
		"constraints":[{"title":"No channel","why":"Nothing repeats","fix":"Pick one channel for 14 days"}],
		"next_7_days":["Write the offer","Publish 5 posts","DM 20 leads","Set a goal","Review replies"],
		"opportunities":["Niche down"],"risks":["No proof"]
	}`

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": analysis}}},
			"usage":   map[string]int{"prompt_tokens": 1200, "completion_tokens": 500},
		})
	}))
	defer modelServer.Close()

	ctx := context.Background()
	st, _ := store.Open(t.TempDir() + "/crew.db")
	defer st.Close()

	user, _ := st.UpsertUser(ctx, "advice-user", "advice@example.com", "Advice")
	company, _ := st.CreateCompany(ctx, user.ID, store.CompanyInput{
		Name: "Advice Co", BusinessKey: "custom", Country: "Nigeria",
		Description: "We help clinics keep clean books every month.", Audience: "clinic owners",
	})

	cipher, _ := secret.New(bytes.Repeat([]byte{9}, 32))
	server := New(config.Config{TaskLease: time.Hour, SessionCacheTTL: time.Minute, Env: "test"}, st,
		payauth.NewClient("http://127.0.0.1:1", time.Second), cipher)

	previous := marketingModelFor
	marketingModelFor = func(_ *Server, _ context.Context, _ string) (resolvedModel, error) {
		return resolvedModel{
			Provider: modelcatalog.Provider{Key: "deepseek", Name: "DeepSeek", Kind: "openai", BaseURL: modelServer.URL, DefaultModel: "deepseek-chat"},
			Model:    "deepseek-chat", APIKey: "sk-test", Source: "platform",
		}, nil
	}
	defer func() { marketingModelFor = previous }()

	request := httptest.NewRequest(http.MethodPost, "/api/advice/study", nil)
	request = request.WithContext(context.WithValue(request.Context(), identityKey, &identity{
		UserID: user.ID, Email: user.Email, Name: user.Name,
	}))
	request = request.WithContext(context.WithValue(request.Context(), companyKey, &company))

	recorder := httptest.NewRecorder()
	server.handleStudyCompany(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "no repeatable acquisition") {
		t.Fatalf("study body: %s", recorder.Body.String())
	}

	stored, err := st.GetCompanyAdvice(ctx, company.ID)
	if err != nil || !strings.Contains(string(stored.Analysis), "No channel") {
		t.Fatalf("stored: %+v %v", stored, err)
	}
}
