// crew-worker is the reference helper. Point it at a Crew server with a helper
// key and it will claim tasks, run your command, and report back.
//
//	go run ./cmd/crew-worker --server http://127.0.0.1:8787 --key crew_xxx
//
// The command receives CREW_TASK_ID, CREW_TASK_TITLE and CREW_TASK_DETAILS in
// its environment. It runs through `sh -c`, locally, as you.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type task struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Details string `json:"details"`
	Status  string `json:"status"`
}

type claimResponse struct {
	OK           bool   `json:"ok"`
	Task         *task  `json:"task"`
	Instructions string `json:"instructions"`
}

func main() {
	server := flag.String("server", "http://127.0.0.1:8787", "Crew server URL")
	key := flag.String("key", os.Getenv("CREW_HELPER_KEY"), "helper key (crew_...)")
	command := flag.String("command", `printf 'Done: %s\n' "$CREW_TASK_TITLE"`, "shell command when no model is configured")
	once := flag.Bool("once", false, "claim and run one task, then exit")
	review := flag.Bool("review", true, "ask for a human OK when finished")
	poll := flag.Duration("poll", 3*time.Second, "how often to look for work")
	mode := flag.String("mode", "auto", "auto, model or command")
	flag.Parse()

	if *key == "" {
		log.Fatal("missing --key (create a helper in Crew, then copy its key)")
	}

	worker := &worker{
		server:  strings.TrimRight(*server, "/"),
		key:     *key,
		command: *command,
		review:  *review,
		mode:    *mode,
		http:    &http.Client{Timeout: 120 * time.Second},
	}

	log.Printf("helper connected to %s — waiting for work", worker.server)

	for {
		claimed, err := worker.claim()
		if err != nil {
			log.Printf("could not check for work: %v", err)
			time.Sleep(*poll * 2)
			continue
		}

		if claimed == nil || claimed.Task == nil {
			if *once {
				log.Print("no tasks waiting")
				return
			}
			time.Sleep(*poll)
			continue
		}

		log.Printf("picked up: %s", claimed.Task.Title)
		worker.run(claimed.Task, claimed.Instructions)

		if *once {
			return
		}
	}
}

type worker struct {
	server  string
	key     string
	command string
	review  bool
	mode    string
	http    *http.Client
}

type modelConfig struct {
	Configured bool   `json:"configured"`
	Provider   string `json:"provider"`
	Kind       string `json:"kind"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
	APIKey     string `json:"api_key"`
}

func (w *worker) claim() (*claimResponse, error) {
	var out claimResponse
	if err := w.do(http.MethodPost, "/api/runner/claim", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (w *worker) askModel(model modelConfig, t *task, instructions string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	system := strings.TrimSpace(instructions)
	if system == "" {
		system = "You are a helpful assistant. Do the task and return the finished work."
	}

	prompt := "Task: " + t.Title + "\n\n" + t.Details + "\n\nReturn the finished work, ready for a human to review. Be concise and concrete."

	switch model.Kind {
	case "anthropic":
		return w.askAnthropic(ctx, model, system, prompt)
	case "google":
		return w.askGoogle(ctx, model, system, prompt)
	default:
		return w.askOpenAI(ctx, model, system, prompt)
	}
}

func (w *worker) askOpenAI(ctx context.Context, model modelConfig, system, prompt string) (string, error) {
	payload := map[string]any{
		"model": model.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}
	return w.postJSON(ctx, model.BaseURL+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + model.APIKey,
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
			return "", fmt.Errorf("empty response")
		}
		return out.Choices[0].Message.Content, nil
	})
}

func (w *worker) askAnthropic(ctx context.Context, model modelConfig, system, prompt string) (string, error) {
	payload := map[string]any{
		"model":      model.Model,
		"max_tokens": 1600,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	}
	return w.postJSON(ctx, model.BaseURL+"/messages", map[string]string{
		"x-api-key":         model.APIKey,
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

func (w *worker) askGoogle(ctx context.Context, model modelConfig, system, prompt string) (string, error) {
	payload := map[string]any{
		"systemInstruction": map[string]any{"parts": []map[string]string{{"text": system}}},
		"contents":          []map[string]any{{"parts": []map[string]string{{"text": prompt}}}},
	}
	url := model.BaseURL + "/models/" + model.Model + ":generateContent?key=" + model.APIKey
	return w.postJSON(ctx, url, nil, payload, func(raw []byte) (string, error) {
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
			return "", fmt.Errorf("empty response")
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

func (w *worker) postJSON(ctx context.Context, url string, headers map[string]string, payload any, decode func([]byte) (string, error)) (string, error) {
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

	resp, err := w.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	return decode(raw)
}

func (w *worker) run(t *task, instructions string) {
	_ = w.do(http.MethodPost, "/api/runner/tasks/"+t.ID+"/progress",
		map[string]string{"message": "Working on it…"}, nil)

	if w.mode != "command" {
		var model modelConfig
		if err := w.do(http.MethodGet, "/api/runner/model", nil, &model); err == nil && model.Configured {
			result, err := w.askModel(model, t, instructions)
			if err == nil && strings.TrimSpace(result) != "" {
				_ = w.do(http.MethodPost, "/api/runner/tasks/"+t.ID+"/complete",
					map[string]any{"summary": result, "needsReview": w.review}, nil)
				log.Printf("finished with %s: %s", model.Model, t.Title)
				return
			}
			log.Printf("model call failed, falling back: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", w.command)
	cmd.Env = append(os.Environ(),
		"CREW_TASK_ID="+t.ID,
		"CREW_TASK_TITLE="+t.Title,
		"CREW_TASK_DETAILS="+t.Details,
	)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()

	summary := strings.TrimSpace(output.String())
	if len(summary) > 4000 {
		summary = summary[:4000] + "\n…"
	}
	if err != nil {
		summary = fmt.Sprintf("I couldn't finish this: %v\n\n%s", err, summary)
	}

	if summary == "" {
		summary = "Finished."
	}

	_ = w.do(http.MethodPost, "/api/runner/tasks/"+t.ID+"/complete",
		map[string]any{"summary": summary, "needsReview": w.review || err != nil}, nil)

	log.Printf("finished: %s", t.Title)
}

func (w *worker) do(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, w.server+path, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+w.key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &apiErr)
		if apiErr.Error == "" {
			apiErr.Error = strings.TrimSpace(string(raw))
		}
		return fmt.Errorf("%d: %s", resp.StatusCode, apiErr.Error)
	}

	if out != nil {
		return json.Unmarshal(raw, out)
	}

	return nil
}
