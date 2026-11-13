package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("missing bearer")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "hello"}}},
		})
	}))
	defer server.Close()

	got, err := Complete(context.Background(), Config{Kind: "openai", BaseURL: server.URL, Model: "gpt", APIKey: "sk-test"}, "system", "prompt", 100)
	if err != nil || got != "hello" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestAnthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-ant" {
			t.Errorf("missing anthropic key")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"text": "first"}, {"text": "second"}},
		})
	}))
	defer server.Close()

	got, err := Complete(context.Background(), Config{Kind: "anthropic", BaseURL: server.URL, Model: "claude", APIKey: "sk-ant"}, "s", "p", 100)
	if err != nil || got != "first\nsecond" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestGoogle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "key=goog") {
			t.Errorf("missing key: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{"parts": []map[string]string{{"text": "gemini"}}},
			}},
		})
	}))
	defer server.Close()

	got, err := Complete(context.Background(), Config{Kind: "google", BaseURL: server.URL, Model: "gemini", APIKey: "goog"}, "s", "p", 100)
	if err != nil || got != "gemini" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestErrorsAreReadable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := Complete(context.Background(), Config{Kind: "openai", BaseURL: server.URL, Model: "gpt", APIKey: "bad"}, "s", "p", 100)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
}
