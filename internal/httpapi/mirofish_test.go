package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davepartner/crew/internal/config"
	"github.com/davepartner/crew/internal/modelcatalog"
	"github.com/davepartner/crew/internal/store"
)

func TestMiroFishEstimate(t *testing.T) {
	previous := marketingModelFor
	marketingModelFor = func(_ *Server, _ context.Context, _ string) (resolvedModel, error) {
		return resolvedModel{
			Provider: modelcatalog.Provider{Key: "deepseek", Name: "DeepSeek", Kind: "openai", DefaultModel: "deepseek-chat"},
			Model:    "deepseek-chat", APIKey: "sk-test",
		}, nil
	}
	defer func() { marketingModelFor = previous }()

	st, err := store.Open(t.TempDir() + "/crew.db")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/mirofish/estimate", strings.NewReader(`{"question":"What if we raise prices 15%?","agents":8,"rounds":2}`))
	request = request.WithContext(context.WithValue(request.Context(), identityKey, &identity{UserID: "x", Email: "jane@example.com"}))
	request = request.WithContext(context.WithValue(request.Context(), companyKey, &store.Company{ID: "c", Name: "Test Co", Description: "A test company", Country: "Nigeria"}))
	recorder := httptest.NewRecorder()

	server := New(config.Config{}, st, nil, nil)
	server.handleMiroFishEstimate(recorder, request)
	if recorder.Code != 200 {
		t.Fatalf("estimate: %d %s", recorder.Code, recorder.Body.String())
	}

	var result struct {
		Estimate struct {
			Tokens  int `json:"tokens"`
			Seconds int `json:"seconds"`
			Calls   int `json:"calls"`
		} `json:"estimate"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil || result.Estimate.Tokens <= 0 || result.Estimate.Seconds <= 0 || result.Estimate.Calls < 3 {
		t.Fatalf("estimate body: %s %v", recorder.Body.String(), err)
	}
}
