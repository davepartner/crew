package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/davepartner/crew/internal/llm"
	"github.com/davepartner/crew/internal/skills"
	"github.com/davepartner/crew/internal/store"
)

const agentKey ctxKey = "agent"

type agentIdentity struct {
	Agent store.Agent
}

func (s *Server) requireAgent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			writeError(w, http.StatusUnauthorized, "This needs a helper key.")
			return
		}

		key := strings.TrimSpace(header[7:])
		if !strings.HasPrefix(key, "crew_") {
			writeError(w, http.StatusUnauthorized, "That helper key isn't valid.")
			return
		}

		agent, err := s.store.AgentForKeyHash(r.Context(), hashKey(key))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "That helper key isn't valid.")
			return
		}

		_ = s.store.TouchAgent(r.Context(), agent.ID)

		ctx := context.WithValue(r.Context(), agentKey, &agentIdentity{Agent: agent})
		next(w, r.WithContext(ctx))
	}
}

func agentFrom(r *http.Request) *agentIdentity {
	value, _ := r.Context().Value(agentKey).(*agentIdentity)
	return value
}

func (s *Server) handleRunnerHeartbeat(w http.ResponseWriter, r *http.Request) {
	agent := agentFrom(r)
	_ = s.store.TouchAgent(r.Context(), agent.Agent.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": agent.Agent.Name})
}

func (s *Server) handleRunnerClaim(w http.ResponseWriter, r *http.Request) {
	agent := agentFrom(r)

	task, err := s.store.ClaimNextTask(r.Context(), agent.Agent.CompanyID, agent.Agent.ID, s.cfg.TaskLease)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": nil})
			return
		}
		writeError(w, http.StatusInternalServerError, "We couldn't hand you a task. Try again.")
		return
	}

	_ = s.store.AddActivity(r.Context(), agent.Agent.UserID, agent.Agent.CompanyID, "task_started",
		agent.Agent.Name+" started “"+task.Title+"”", task.ID, agent.Agent.ID)

	instructions := agent.Agent.Instructions
	if len(agent.Agent.Skills) > 0 {
		instructions += skills.Prompt(agent.Agent.Skills, 24000)
	}
	if agent.Agent.CanDelegate {
		instructions += "\n\nYou can hand part of this work to another helper with the delegate_task tool. " +
			"Use the helper's exact name. Delegated tasks are counted against the company budget."
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"task":         task,
		"instructions": instructions,
		"skills":       agent.Agent.Skills,
	})
}

func (s *Server) handleRunnerProgress(w http.ResponseWriter, r *http.Request) {
	agent := agentFrom(r)

	task, ok := s.runnerTask(w, r, agent)
	if !ok {
		return
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	message := trim(body.Message, 2000)
	if message == "" {
		writeError(w, http.StatusUnprocessableEntity, "Say what you did first.")
		return
	}

	comment, err := s.store.AddComment(r.Context(), task.ID, "helper", agent.Agent.Name, message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that update.")
		return
	}

	// Posting progress renews the lease so a long job isn't mistaken for a
	// crashed helper.
	_ = s.store.TouchTask(r.Context(), agent.Agent.CompanyID, task.ID)

	_ = s.store.AddActivity(r.Context(), agent.Agent.UserID, agent.Agent.CompanyID, "progress",
		agent.Agent.Name+" shared an update on “"+task.Title+"”", task.ID, agent.Agent.ID)

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "comment": comment})
}

func (s *Server) handleRunnerComplete(w http.ResponseWriter, r *http.Request) {
	agent := agentFrom(r)

	task, ok := s.runnerTask(w, r, agent)
	if !ok {
		return
	}

	var body struct {
		Summary     string `json:"summary"`
		NeedsReview bool   `json:"needsReview"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	summary := trim(body.Summary, 4000)
	if summary == "" {
		summary = "Finished."
	}

	status := store.StatusDone
	if body.NeedsReview {
		status = store.StatusReview
	}

	updated, err := s.store.UpdateTask(r.Context(), agent.Agent.CompanyID, task.ID, store.TaskPatch{Status: &status})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that result.")
		return
	}

	_, _ = s.store.AddComment(r.Context(), task.ID, "helper", agent.Agent.Name, summary)

	message := agent.Agent.Name + " finished “" + task.Title + "”"
	if body.NeedsReview {
		message = agent.Agent.Name + " finished “" + task.Title + "” and wants your OK"
	}
	_ = s.store.AddActivity(r.Context(), agent.Agent.UserID, agent.Agent.CompanyID, "task_finished", message, task.ID, agent.Agent.ID)

	if body.NeedsReview {
		_ = s.store.NotifyCompany(r.Context(), agent.Agent.CompanyID, "review",
			agent.Agent.Name+" needs your OK", task.Title, task.ID)
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": updated})
}

// handleRunnerModel tells the helper which model Crew will run for it. The
// key stays on the server; the helper calls the proxy below.
func (s *Server) handleRunnerModel(w http.ResponseWriter, r *http.Request) {
	agent := agentFrom(r)

	model, err := s.modelForAgent(r.Context(), agent.Agent)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "configured": false, "reason": "no_model"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"configured": true,
		"provider":   model.Provider.Key,
		"kind":       model.Provider.Kind,
		"base_url":   model.Provider.BaseURL,
		"model":      model.Model,
		"mode":       "proxy",
		"platform":   model.Source == "platform",
	})
}

type runnerAIRequest struct {
	System    string `json:"system"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"maxTokens"`
	TaskID    string `json:"taskId"`
	Tools     bool   `json:"tools"`
}

// handleRunnerAI is the only place a model key is used for helper work. The
// helper never receives the key, so the shared platform credential cannot be
// extracted by creating a helper.
func (s *Server) handleRunnerAI(w http.ResponseWriter, r *http.Request) {
	agent := agentFrom(r)

	var body runnerAIRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if trim(body.Prompt, 60000) == "" {
		writeError(w, http.StatusUnprocessableEntity, "The prompt is empty.")
		return
	}

	model, err := s.modelForAgent(r.Context(), agent.Agent)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"ok":    false,
			"code":  "no_model",
			"error": "No model is available for this helper. Add a key in Settings → AI models.",
		})
		return
	}

	blocked, err := s.checkBudget(r.Context(), agent.Agent.UserID, agent.Agent.CompanyID, agent.Agent.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't check the AI budget.")
		return
	}
	if blocked != "" {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{
			"ok":    false,
			"code":  "budget",
			"error": blocked,
		})
		return
	}

	maxTokens := body.MaxTokens
	if maxTokens <= 0 || maxTokens > 8000 {
		maxTokens = 1600
	}

	var tools []llm.Tool
	if body.Tools && agent.Agent.CanDelegate {
		tools = []llm.Tool{{
			Name:        "delegate_task",
			Description: "Hand a piece of work to another helper on the same team. Use their exact name.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agent":   map[string]any{"type": "string", "description": "The other helper's exact name."},
					"title":   map[string]any{"type": "string", "description": "A short task title."},
					"details": map[string]any{"type": "string", "description": "What they should do."},
				},
				"required": []string{"agent", "title"},
			},
		}}
	}

	result, err := llm.CompleteTools(r.Context(), llm.Config{
		Kind:    model.Provider.Kind,
		BaseURL: model.Provider.BaseURL,
		Model:   model.Model,
		APIKey:  model.APIKey,
	}, body.System, body.Prompt, maxTokens, tools)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The model couldn't answer: "+err.Error())
		return
	}

	cost := s.saveUsage(r.Context(), agent.Agent.UserID, agent.Agent.CompanyID, agent.Agent.ID, body.TaskID, model, result.Usage)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"text":       result.Text,
		"usage":      result.Usage,
		"cost":       cost,
		"model":      model.Model,
		"source":     model.Source,
		"tool_calls": result.ToolCalls,
	})
}

func (s *Server) handleRunnerDelegate(w http.ResponseWriter, r *http.Request) {
	agent := agentFrom(r)

	var body struct {
		Agent   string `json:"agent"`
		Title   string `json:"title"`
		Details string `json:"details"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !agent.Agent.CanDelegate {
		writeError(w, http.StatusForbidden, "This helper isn't allowed to delegate.")
		return
	}

	title := trim(body.Title, 200)
	if title == "" {
		writeError(w, http.StatusUnprocessableEntity, "The delegated task needs a title.")
		return
	}

	target, err := s.store.FindAgentByName(r.Context(), agent.Agent.CompanyID, strings.TrimSpace(body.Agent))
	if err != nil {
		writeError(w, http.StatusNotFound, "There is no helper with that name.")
		return
	}
	if target.ID == agent.Agent.ID {
		writeError(w, http.StatusUnprocessableEntity, "A helper can't delegate to itself.")
		return
	}

	task, err := s.store.CreateTask(r.Context(), agent.Agent.UserID, agent.Agent.CompanyID, title, trim(body.Details, 8000), target.ID)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "That task couldn't be created. The monthly task budget may be used up.")
		return
	}

	_ = s.store.AddActivity(r.Context(), agent.Agent.UserID, agent.Agent.CompanyID, "delegated",
		agent.Agent.Name+" handed “"+task.Title+"” to "+target.Name, task.ID, target.ID)

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "task": task, "agent": target.Name})
}

// runnerTask loads a task and proves it is checked out to this helper, so one
// helper can never write to another's work.
func (s *Server) runnerTask(w http.ResponseWriter, r *http.Request, agent *agentIdentity) (store.Task, bool) {
	task, err := s.store.GetTask(r.Context(), agent.Agent.CompanyID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That task doesn't exist.")
		return store.Task{}, false
	}

	if task.AgentID == nil || *task.AgentID != agent.Agent.ID {
		writeError(w, http.StatusForbidden, "That task isn't yours.")
		return store.Task{}, false
	}

	if task.Status == store.StatusDone {
		writeError(w, http.StatusConflict, "That task is already finished.")
		return store.Task{}, false
	}

	return task, true
}
