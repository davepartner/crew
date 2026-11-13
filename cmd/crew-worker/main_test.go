package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAskOpenAIReadsTheMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("missing bearer")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "hello from the model"}}},
		})
	}))
	defer server.Close()

	worker := &worker{http: server.Client()}
	got, err := worker.askOpenAI(context.Background(), modelConfig{BaseURL: server.URL, APIKey: "sk-test"}, "system", "prompt")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got != "hello from the model" {
		t.Fatalf("got %q", got)
	}
}

func TestAskAnthropicJoinsContentParts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-ant" {
			t.Errorf("missing anthropic key")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"text": "first"}, {"text": "second"}},
		})
	}))
	defer server.Close()

	worker := &worker{http: server.Client()}
	got, err := worker.askAnthropic(context.Background(), modelConfig{BaseURL: server.URL, APIKey: "sk-ant"}, "system", "prompt")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got != "first\nsecond" {
		t.Fatalf("got %q", got)
	}
}

func TestAskGoogleReadsCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "key=goog-key") {
			t.Errorf("missing key query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{"parts": []map[string]string{{"text": "gemini says hi"}}},
			}},
		})
	}))
	defer server.Close()

	worker := &worker{http: server.Client()}
	got, err := worker.askGoogle(context.Background(), modelConfig{BaseURL: server.URL, APIKey: "goog-key", Model: "gemini"}, "system", "prompt")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if got != "gemini says hi" {
		t.Fatalf("got %q", got)
	}
}

func TestModelErrorsAreReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	worker := &worker{http: server.Client()}
	if _, err := worker.askOpenAI(context.Background(), modelConfig{BaseURL: server.URL, APIKey: "bad"}, "s", "p"); err == nil {
		t.Fatal("expected an error")
	}
}
