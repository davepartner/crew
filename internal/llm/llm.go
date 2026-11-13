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

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type Result struct {
	Text      string     `json:"text"`
	Usage     Usage      `json:"usage"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

var client = &http.Client{Timeout: 120 * time.Second}

// Complete returns the model's text for one system + user prompt.
func Complete(ctx context.Context, cfg Config, system, prompt string, maxTokens int) (string, error) {
	result, err := CompleteResult(ctx, cfg, system, prompt, maxTokens)
	return result.Text, err
}

func CompleteTools(ctx context.Context, cfg Config, system, prompt string, maxTokens int, tools []Tool) (Result, error) {
	if cfg.Kind != "" && cfg.Kind != "openai" {
		return CompleteResult(ctx, cfg, system, prompt, maxTokens)
	}
	return complete(ctx, cfg, system, prompt, maxTokens, tools)
}

func CompleteResult(ctx context.Context, cfg Config, system, prompt string, maxTokens int) (Result, error) {
	return complete(ctx, cfg, system, prompt, maxTokens, nil)
}

func complete(ctx context.Context, cfg Config, system, prompt string, maxTokens int, tools []Tool) (Result, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return Result{}, errors.New("model provider has no base URL")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return Result{}, errors.New("no model selected")
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
		return openAI(ctx, cfg, system, prompt, tools)
	}
}

func openAI(ctx context.Context, cfg Config, system, prompt string, tools []Tool) (Result, error) {
	payload := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}

	if len(tools) > 0 {
		definitions := make([]map[string]any, 0, len(tools))
		for _, tool := range tools {
			definitions = append(definitions, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.Parameters,
				},
			})
		}
		payload["tools"] = definitions
		payload["tool_choice"] = "auto"
	}

	return postJSON(ctx, strings.TrimRight(cfg.BaseURL, "/")+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + cfg.APIKey,
	}, payload, func(raw []byte) (Result, error) {
		var out struct {
			Choices []struct {
				Message struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return Result{}, err
		}
		if len(out.Choices) == 0 {
			return Result{}, errors.New("the model returned no choices")
		}
		calls := make([]ToolCall, 0, len(out.Choices[0].Message.ToolCalls))
		for _, call := range out.Choices[0].Message.ToolCalls {
			calls = append(calls, ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments})
		}

		return Result{
			Text:      out.Choices[0].Message.Content,
			Usage:     Usage{PromptTokens: out.Usage.PromptTokens, CompletionTokens: out.Usage.CompletionTokens},
			ToolCalls: calls,
		}, nil
	})
}

func anthropic(ctx context.Context, cfg Config, system, prompt string, maxTokens int) (Result, error) {
	payload := map[string]any{
		"model":      cfg.Model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	}

	return postJSON(ctx, strings.TrimRight(cfg.BaseURL, "/")+"/messages", map[string]string{
		"x-api-key":         cfg.APIKey,
		"anthropic-version": "2023-06-01",
	}, payload, func(raw []byte) (Result, error) {
		var out struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return Result{}, err
		}
		parts := make([]string, 0, len(out.Content))
		for _, part := range out.Content {
			if part.Text != "" {
				parts = append(parts, part.Text)
			}
		}
		return Result{
			Text:  strings.Join(parts, "\n"),
			Usage: Usage{PromptTokens: out.Usage.InputTokens, CompletionTokens: out.Usage.OutputTokens},
		}, nil
	})
}

func google(ctx context.Context, cfg Config, system, prompt string) (Result, error) {
	payload := map[string]any{
		"systemInstruction": map[string]any{"parts": []map[string]string{{"text": system}}},
		"contents":          []map[string]any{{"parts": []map[string]string{{"text": prompt}}}},
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/models/" + cfg.Model + ":generateContent?key=" + cfg.APIKey

	return postJSON(ctx, url, nil, payload, func(raw []byte) (Result, error) {
		var out struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return Result{}, err
		}
		if len(out.Candidates) == 0 {
			return Result{}, errors.New("the model returned no candidates")
		}

		parts := make([]string, 0)
		for _, part := range out.Candidates[0].Content.Parts {
			if part.Text != "" {
				parts = append(parts, part.Text)
			}
		}
		return Result{
			Text: strings.Join(parts, "\n"),
			Usage: Usage{
				PromptTokens:     out.UsageMetadata.PromptTokenCount,
				CompletionTokens: out.UsageMetadata.CandidatesTokenCount,
			},
		}, nil
	})
}

func postJSON(ctx context.Context, url string, headers map[string]string, payload any, decode func([]byte) (Result, error)) (Result, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		message := strings.TrimSpace(string(raw))
		if len(message) > 500 {
			message = message[:500]
		}
		return Result{}, fmt.Errorf("%s: %s", resp.Status, message)
	}

	return decode(raw)
}
