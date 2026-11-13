package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
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

func TestPublicAddress(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1":    false,
		"10.0.0.1":     false,
		"192.168.1.5":  false,
		"169.254.1.1":  false,
		"100.64.0.1":   false,
		"0.0.0.0":      false,
		"::1":          false,
		"fc00::1":      false,
		"fe80::1":      false,
		"8.8.8.8":      true,
		"1.1.1.1":      true,
		"2606:4700::1": true,
	}

	for raw, want := range cases {
		if got := publicAddress(net.ParseIP(raw)); got != want {
			t.Errorf("publicAddress(%s) = %v, want %v", raw, got, want)
		}
	}
}

func TestExtractTextDropsCodeAndDecodes(t *testing.T) {
	document := `<html><head><title>Acme</title><style>.x{color:red}</style></head>
	<body><script>alert(1)</script><h1>Run your company</h1><p>For&nbsp;small teams &amp; solo founders.</p></body></html>`

	text := extractText(document)
	if strings.Contains(text, "alert") || strings.Contains(text, "color:red") {
		t.Fatalf("script or style leaked: %q", text)
	}
	if !strings.Contains(text, "Run your company") || !strings.Contains(text, "small teams & solo founders") {
		t.Fatalf("text missing: %q", text)
	}
}

func TestFetchPageReadsALocalPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><title>Acme Store</title>
			<meta name="description" content="Shoes for runners">
			</head><body><h1>Shoes that go the distance</h1>
			<p>Free returns for 30 days. Start your trial today.</p></body></html>`))
	}))
	defer server.Close()

	previous := addressAllowed
	addressAllowed = func(net.IP) bool { return true }
	previousPort := portAllowed
	portAllowed = func(string) bool { return true }
	defer func() {
		addressAllowed = previous
		portAllowed = previousPort
	}()

	page, err := fetchPage(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if page.Title != "Acme Store" || page.Description != "Shoes for runners" {
		t.Fatalf("metadata: %+v", page)
	}
	if !strings.Contains(page.Text, "Free returns") {
		t.Fatalf("text: %q", page.Text)
	}
}

func TestFetchPageBlocksPrivateAndBadURLs(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "http://127.0.0.1/", "http://localhost:8080/", "http://10.0.0.5/"} {
		if _, err := fetchPage(context.Background(), raw); err == nil {
			t.Errorf("fetchPage(%s) should fail", raw)
		}
	}
}

func TestMarketingPromptIncludesEvidence(t *testing.T) {
	prompt := buildMarketingPrompt(pageEvidence{
		URL:         "https://acme.test",
		Title:       "Acme",
		Description: "Shoes",
		Text:        "Free returns",
	}, "how do they price?", "funnel-monetization")

	for _, want := range []string{"https://acme.test", "Acme", "how do they price?", "Free returns", "funnel-monetization"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestMarketingAnalyzeEndToEnd(t *testing.T) {
	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Acme</title></head><body>
			<h1>Run your company on autopilot</h1>
			<p>Start free, then $29 a month. Book a demo today.</p></body></html>`))
	}))
	defer pageServer.Close()

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "FAKE TEARDOWN REPORT"}}},
		})
	}))
	defer modelServer.Close()

	previousAddress, previousPort := addressAllowed, portAllowed
	addressAllowed = func(net.IP) bool { return true }
	portAllowed = func(string) bool { return true }
	previousModel := marketingModelFor
	marketingModelFor = func(_ *Server, _ context.Context, _ string) (string, modelcatalog.Provider, string, error) {
		return "fake", modelcatalog.Provider{
			Key: "fake", Name: "Fake", Kind: "openai", BaseURL: modelServer.URL, DefaultModel: "fake-model",
		}, "sk-fake", nil
	}
	defer func() {
		addressAllowed, portAllowed = previousAddress, previousPort
		marketingModelFor = previousModel
	}()

	ctx := context.Background()
	st, err := store.Open(t.TempDir() + "/crew.db")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	user, err := st.UpsertUser(ctx, "ext-1", "jane@example.com", "Jane")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	company, err := st.CreateCompany(ctx, user.ID, store.CompanyInput{Name: "Acme Co", BusinessKey: "custom"})
	if err != nil {
		t.Fatalf("company: %v", err)
	}
	if _, err := st.CreateAgent(ctx, user.ID, company.ID, "Marketer", "Marketing", "📣", "", nil, "hash"); err != nil {
		t.Fatalf("agent: %v", err)
	}

	cipher, err := secret.New(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}

	server := New(config.Config{TaskLease: time.Hour, SessionCacheTTL: time.Minute}, st,
		payauth.NewClient("http://127.0.0.1:1", time.Second), cipher)

	body, _ := json.Marshal(map[string]string{
		"url": pageServer.URL, "focus": "how do they price?", "playbook": "funnel-monetization",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/marketing/analyze", bytes.NewReader(body))
	request = request.WithContext(context.WithValue(request.Context(), identityKey, &identity{
		UserID: user.ID, Email: user.Email, Name: user.Name,
	}))
	request = request.WithContext(context.WithValue(request.Context(), companyKey, &company))

	recorder := httptest.NewRecorder()
	server.handleMarketingAnalyze(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
	}

	var result struct {
		Report string         `json:"report"`
		Task   map[string]any `json:"task"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Report != "FAKE TEARDOWN REPORT" {
		t.Fatalf("report: %q", result.Report)
	}
	if result.Task["status"] != "review" {
		t.Fatalf("task: %v", result.Task)
	}

	taskID, _ := result.Task["id"].(string)
	task, err := st.GetTask(ctx, company.ID, taskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if len(task.Comments) == 0 || task.Comments[0].Body != "FAKE TEARDOWN REPORT" {
		t.Fatalf("comments: %+v", task.Comments)
	}
}
