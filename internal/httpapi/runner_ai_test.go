package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davepartner/crew/internal/config"
	"github.com/davepartner/crew/internal/payauth"
	"github.com/davepartner/crew/internal/secret"
	"github.com/davepartner/crew/internal/store"
)

func TestRunnerAIUsesPlatformKeyAndTracksSpend(t *testing.T) {
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-platform" {
			t.Errorf("missing platform key")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"content": "Proxy answer",
					"tool_calls": []map[string]any{{
						"id": "call-1",
						"function": map[string]string{
							"name": "delegate_task", "arguments": `{"agent":"Bob","title":"Check prices"}`,
						},
					}},
				},
			}},
			"usage": map[string]int{"prompt_tokens": 1000, "completion_tokens": 500},
		})
	}))
	defer modelServer.Close()

	ctx := context.Background()
	st, err := store.Open(t.TempDir() + "/crew.db")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	user, _ := st.UpsertUser(ctx, "ext-ai", "proxy@example.com", "Proxy")
	company, _ := st.CreateCompany(ctx, user.ID, store.CompanyInput{Name: "Proxy Co", BusinessKey: "custom"})
	agent, _ := st.CreateAgent(ctx, user.ID, company.ID, "Ada", "Operations", "🧭", "", nil, hashKey("crew_test_key"))

	cipher, _ := secret.New(bytes.Repeat([]byte{4}, 32))
	server := New(config.Config{
		TaskLease:        time.Hour,
		SessionCacheTTL:  time.Minute,
		PlatformProvider: "deepseek",
		PlatformModel:    "deepseek-chat",
		PlatformBaseURL:  modelServer.URL,
		PlatformAPIKey:   "sk-platform",
	}, st, payauth.NewClient("http://127.0.0.1:1", time.Second), cipher)

	call := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{
			"system": "You are helpful.", "prompt": "Do the thing", "maxTokens": 200, "taskId": "task-1", "tools": true,
		})
		request := httptest.NewRequest(http.MethodPost, "/api/runner/ai", bytes.NewReader(body))
		request = request.WithContext(context.WithValue(request.Context(), agentKey, &agentIdentity{Agent: agent}))
		recorder := httptest.NewRecorder()
		server.handleRunnerAI(recorder, request)
		return recorder
	}

	recorder := call()
	if recorder.Code != 200 {
		t.Fatalf("call: %d %s", recorder.Code, recorder.Body.String())
	}

	var result struct {
		Text  string      `json:"text"`
		Cost  float64     `json:"cost"`
		Usage interface{} `json:"usage"`
	}
	_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	if result.Text != "Proxy answer" || result.Cost <= 0 {
		t.Fatalf("result: %+v", result)
	}

	spend, err := st.CompanySpend(ctx, company.ID, store.MonthStart())
	if err != nil || spend <= 0 {
		t.Fatalf("spend not recorded: %v %v", spend, err)
	}

	_, err = st.UpdateCompany(ctx, user.ID, company.ID, store.CompanyInput{
		Name: "Proxy Co", BusinessKey: "custom", MonthlyBudgetUSD: 0.0001,
	})
	if err != nil {
		t.Fatalf("budget: %v", err)
	}

	blocked := call()
	if blocked.Code != http.StatusPaymentRequired {
		t.Fatalf("expected budget block, got %d %s", blocked.Code, blocked.Body.String())
	}
}
