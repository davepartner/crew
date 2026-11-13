package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/davepartner/crew/internal/modelcatalog"
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

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": updated})
}

// handleRunnerModel hands the helper the model configuration it should use.
// The key is decrypted only for the holder of a valid helper key.
func (s *Server) handleRunnerModel(w http.ResponseWriter, r *http.Request) {
	agent := agentFrom(r)

	if agent.Agent.ModelProvider == "" || agent.Agent.ModelName == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "configured": false, "reason": "no_model"})
		return
	}

	provider, ok := modelcatalog.ByKey(agent.Agent.ModelProvider)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "configured": false, "reason": "unknown_provider"})
		return
	}

	if provider.Key == "ollama" {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"configured": true,
			"provider":   provider.Key,
			"kind":       provider.Kind,
			"base_url":   provider.BaseURL,
			"model":      agent.Agent.ModelName,
			"api_key":    "",
		})
		return
	}

	cipherText, err := s.store.ProviderKeyCipher(r.Context(), agent.Agent.UserID, provider.Key)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "configured": false, "reason": "no_key"})
		return
	}

	key, err := s.cipher.Decrypt(cipherText)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't read that model key.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"configured": true,
		"provider":   provider.Key,
		"kind":       provider.Kind,
		"base_url":   provider.BaseURL,
		"model":      agent.Agent.ModelName,
		"api_key":    key,
	})
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
