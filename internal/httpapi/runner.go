package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/davepartner/crew/internal/store"
)

const agentKey ctxKey = "agent"

type agentIdentity struct {
	Agent  store.Agent
	UserID string
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

		agent, userID, err := s.store.AgentForKeyHash(r.Context(), hashKey(key))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "That helper key isn't valid.")
			return
		}

		_ = s.store.TouchAgent(r.Context(), agent.ID)

		ctx := context.WithValue(r.Context(), agentKey, &agentIdentity{Agent: agent, UserID: userID})
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

	task, err := s.store.ClaimNextTask(r.Context(), agent.UserID, agent.Agent.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": nil})
			return
		}
		writeError(w, http.StatusInternalServerError, "We couldn't hand you a task. Try again.")
		return
	}

	_ = s.store.AddActivity(r.Context(), agent.UserID, "task_started",
		agent.Agent.Name+" started “"+task.Title+"”", task.ID, agent.Agent.ID)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"task":         task,
		"instructions": agent.Agent.Instructions,
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

	_ = s.store.AddActivity(r.Context(), agent.UserID, "progress",
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

	updated, err := s.store.UpdateTask(r.Context(), agent.UserID, task.ID, store.TaskPatch{Status: &status})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that result.")
		return
	}

	_, _ = s.store.AddComment(r.Context(), task.ID, "helper", agent.Agent.Name, summary)

	message := agent.Agent.Name + " finished “" + task.Title + "”"
	if body.NeedsReview {
		message = agent.Agent.Name + " finished “" + task.Title + "” and wants your OK"
	}
	_ = s.store.AddActivity(r.Context(), agent.UserID, "task_finished", message, task.ID, agent.Agent.ID)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": updated})
}

// runnerTask loads a task and proves it is checked out to this helper, so one
// helper can never write to another's work.
func (s *Server) runnerTask(w http.ResponseWriter, r *http.Request, agent *agentIdentity) (store.Task, bool) {
	task, err := s.store.GetTask(r.Context(), agent.UserID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That task doesn't exist.")
		return store.Task{}, false
	}

	if task.AgentID == nil || *task.AgentID != agent.Agent.ID {
		writeError(w, http.StatusForbidden, "That task isn't yours.")
		return store.Task{}, false
	}

	return task, true
}
