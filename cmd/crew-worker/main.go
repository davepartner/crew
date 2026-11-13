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

	"github.com/davepartner/crew/internal/llm"
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
	Mode       string `json:"mode"`
}

func (w *worker) claim() (*claimResponse, error) {
	var out claimResponse
	if err := w.do(http.MethodPost, "/api/runner/claim", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (w *worker) askModel(model modelConfig, t *task, instructions string) (string, []llm.ToolCall, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	system := strings.TrimSpace(instructions)
	if system == "" {
		system = "You are a helpful assistant. Do the task and return the finished work."
	}

	prompt := "Task: " + t.Title + "\n\n" + t.Details + "\n\nReturn the finished work, ready for a human to review. Be concise and concrete."

	if model.Mode == "proxy" {
		var out struct {
			Text      string         `json:"text"`
			Cost      float64        `json:"cost"`
			ToolCalls []llm.ToolCall `json:"tool_calls"`
		}
		err := w.do(http.MethodPost, "/api/runner/ai", map[string]any{
			"system":    system,
			"prompt":    prompt,
			"maxTokens": 1600,
			"taskId":    t.ID,
			"tools":     true,
		}, &out)
		if err == nil && out.Cost > 0 {
			log.Printf("model cost $%.4f", out.Cost)
		}
		return out.Text, out.ToolCalls, err
	}

	text, err := llm.Complete(ctx, llm.Config{
		Kind:    model.Kind,
		BaseURL: model.BaseURL,
		Model:   model.Model,
		APIKey:  model.APIKey,
	}, system, prompt, 1600)
	return text, nil, err
}

func (w *worker) run(t *task, instructions string) {
	_ = w.do(http.MethodPost, "/api/runner/tasks/"+t.ID+"/progress",
		map[string]string{"message": "Working on it…"}, nil)

	if w.mode != "command" {
		var model modelConfig
		if err := w.do(http.MethodGet, "/api/runner/model", nil, &model); err == nil && model.Configured {
			result, calls, err := w.askModel(model, t, instructions)
			if err == nil && (strings.TrimSpace(result) != "" || len(calls) > 0) {
				delegated := make([]string, 0)
				for _, call := range calls {
					if call.Name != "delegate_task" {
						continue
					}

					var args struct {
						Agent   string `json:"agent"`
						Title   string `json:"title"`
						Details string `json:"details"`
					}
					if json.Unmarshal([]byte(call.Arguments), &args) != nil || args.Agent == "" || args.Title == "" {
						continue
					}

					var out struct {
						Agent string `json:"agent"`
					}
					if err := w.do(http.MethodPost, "/api/runner/delegate", args, &out); err == nil {
						delegated = append(delegated, out.Agent)
						_ = w.do(http.MethodPost, "/api/runner/tasks/"+t.ID+"/progress",
							map[string]string{"message": "Handed “" + args.Title + "” to " + out.Agent}, nil)
					}
				}

				summary := result
				if len(delegated) > 0 {
					summary = "Delegated to " + strings.Join(delegated, ", ") + ".\n\n" + summary
				}
				_ = w.do(http.MethodPost, "/api/runner/tasks/"+t.ID+"/complete",
					map[string]any{"summary": summary, "needsReview": w.review}, nil)
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
