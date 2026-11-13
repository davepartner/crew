// Package llm calls the model providers Crew supports. It is shared by the
// server (marketing lab) and the reference worker (helper tasks).
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	Kind    string // openai, anthropic, google
	BaseURL string
	Model   string
	APIKey  string
}

var client = &http.Client{Timeout: 120 * time.Second}

// Complete returns the model's text for one system + user prompt.
func Complete(ctx context.Context, cfg Config, system, prompt string, maxTokens int) (string, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return "", errors.New("model provider has no base URL")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return "", errors.New("no model selected")
	}
	if maxTokens <= 0 {
		maxTokens = 1600
	}

	switch cfg.Kind {
	case "anthropic":
		return anthropic(ctx, cfg, system, prompt, maxTokens)
	case "google":
		return google(ctx, cfg, system, prompt)
	default:
		return openAI(ctx, cfg, system, prompt)
	}
}

func openAI(ctx context.Context, cfg Config, system, prompt string) (string, error) {
	payload := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}

	return postJSON(ctx, strings.TrimRight(cfg.BaseURL, "/")+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + cfg.APIKey,
	}, payload, func(raw []byte) (string, error) {
		var out struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return "", err
		}
		if len(out.Choices) == 0 {
			return "", errors.New("the model returned no choices")
		}
		return out.Choices[0].Message.Content, nil
	})
}

func anthropic(ctx context.Context, cfg Config, system, prompt string, maxTokens int) (string, error) {
	payload := map[string]any{
		"model":      cfg.Model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	}

	return postJSON(ctx, strings.TrimRight(cfg.BaseURL, "/")+"/messages", map[string]string{
		"x-api-key":         cfg.APIKey,
		"anthropic-version": "2023-06-01",
	}, payload, func(raw []byte) (string, error) {
		var out struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return "", err
		}
		parts := make([]string, 0, len(out.Content))
		for _, part := range out.Content {
			if part.Text != "" {
				parts = append(parts, part.Text)
			}
		}
		return strings.Join(parts, "\n"), nil
	})
}

func google(ctx context.Context, cfg Config, system, prompt string) (string, error) {
	payload := map[string]any{
		"systemInstruction": map[string]any{"parts": []map[string]string{{"text": system}}},
		"contents":          []map[string]any{{"parts": []map[string]string{{"text": prompt}}}},
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/models/" + cfg.Model + ":generateContent?key=" + cfg.APIKey

	return postJSON(ctx, url, nil, payload, func(raw []byte) (string, error) {
		var out struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return "", err
		}
		if len(out.Candidates) == 0 {
			return "", errors.New("the model returned no candidates")
		}

		parts := make([]string, 0)
		for _, part := range out.Candidates[0].Content.Parts {
			if part.Text != "" {
				parts = append(parts, part.Text)
			}
		}
		return strings.Join(parts, "\n"), nil
	})
}

func postJSON(ctx context.Context, url string, headers map[string]string, payload any, decode func([]byte) (string, error)) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		message := strings.TrimSpace(string(raw))
		if len(message) > 500 {
			message = message[:500]
		}
		return "", fmt.Errorf("%s: %s", resp.Status, message)
	}

	return decode(raw)
}
