package httpapi

import (
	"net/http"

	"github.com/davepartner/crew/internal/store"
)

var allowedStatuses = map[string]bool{
	store.StatusTodo:    true,
	store.StatusWorking: true,
	store.StatusReview:  true,
	store.StatusDone:    true,
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	status := r.URL.Query().Get("status")
	if status != "" && !allowedStatuses[status] {
		writeError(w, http.StatusUnprocessableEntity, "That filter doesn't exist.")
		return
	}

	tasks, err := s.store.ListTasks(r.Context(), id.UserID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your tasks. Please refresh.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "tasks": tasks})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	var body struct {
		Title   string `json:"title"`
		Details string `json:"details"`
		AgentID string `json:"agentId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	title := trim(body.Title, 200)
	if len([]rune(title)) < 1 {
		writeError(w, http.StatusUnprocessableEntity, "Give the task a title.")
		return
	}

	agentID := ""
	if body.AgentID != "" {
		agent, err := s.store.GetAgent(r.Context(), id.UserID, body.AgentID)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "That helper doesn't exist.")
			return
		}
		agentID = agent.ID
	}

	task, err := s.store.CreateTask(r.Context(), id.UserID, title, trim(body.Details, 8000), agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that task. Please try again.")
		return
	}

	message := "You added a task"
	if task.Agent != nil {
		message = "You gave " + task.Agent.Name + " a task"
	}
	_ = s.store.AddActivity(r.Context(), id.UserID, "task_created", message, task.ID, agentID)

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "task": task})
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	task, err := s.store.GetTask(r.Context(), id.UserID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That task doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": task})
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	var body struct {
		Title    *string `json:"title"`
		Details  *string `json:"details"`
		Status   *string `json:"status"`
		AgentID  *string `json:"agentId"`
		Unassign bool    `json:"unassign"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	patch := store.TaskPatch{
		Details: body.Details,
		Clear:   body.Unassign,
	}

	if body.Title != nil {
		title := trim(*body.Title, 200)
		if len([]rune(title)) < 1 {
			writeError(w, http.StatusUnprocessableEntity, "Give the task a title.")
			return
		}
		patch.Title = &title
	}

	if body.Status != nil {
		if !allowedStatuses[*body.Status] {
			writeError(w, http.StatusUnprocessableEntity, "That status doesn't exist.")
			return
		}
		patch.Status = body.Status
	}

	if body.AgentID != nil && *body.AgentID != "" {
		if _, err := s.store.GetAgent(r.Context(), id.UserID, *body.AgentID); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "That helper doesn't exist.")
			return
		}
		patch.AgentID = body.AgentID
		patch.Clear = false
	}

	task, err := s.store.UpdateTask(r.Context(), id.UserID, r.PathValue("id"), patch)
	if err != nil {
		writeError(w, http.StatusNotFound, "That task doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": task})
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	if err := s.store.DeleteTask(r.Context(), id.UserID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "That task doesn't exist.")
		return
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, "task_deleted", "You deleted a task", "", "")

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAddComment(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	taskID := r.PathValue("id")

	task, err := s.store.GetTask(r.Context(), id.UserID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "That task doesn't exist.")
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	text := trim(body.Body, 4000)
	if len([]rune(text)) < 1 {
		writeError(w, http.StatusUnprocessableEntity, "Write a message first.")
		return
	}

	comment, err := s.store.AddComment(r.Context(), task.ID, "you", id.Name, text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't post that. Please try again.")
		return
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, "comment", "You commented on “"+task.Title+"”", task.ID, "")

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "comment": comment})
}

func (s *Server) handleApproveTask(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	taskID := r.PathValue("id")

	task, err := s.store.GetTask(r.Context(), id.UserID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "That task doesn't exist.")
		return
	}

	if task.Status != store.StatusReview {
		writeError(w, http.StatusConflict, "That task isn't waiting for your OK.")
		return
	}

	status := store.StatusDone
	updated, err := s.store.UpdateTask(r.Context(), id.UserID, task.ID, store.TaskPatch{Status: &status})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't approve that. Please try again.")
		return
	}

	_, _ = s.store.AddComment(r.Context(), task.ID, "system", "Crew", "You approved this. Nice.")
	_ = s.store.AddActivity(r.Context(), id.UserID, "approved", "You approved “"+task.Title+"”", task.ID, "")

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": updated})
}

func (s *Server) handleSendBack(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	taskID := r.PathValue("id")

	task, err := s.store.GetTask(r.Context(), id.UserID, taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "That task doesn't exist.")
		return
	}

	var body struct {
		Note string `json:"note"`
	}
	_ = decodeJSON(r, &body)

	if task.Status != store.StatusReview && task.Status != store.StatusDone {
		writeError(w, http.StatusConflict, "That task isn't ready to send back yet.")
		return
	}

	note := trim(body.Note, 4000)
	if note == "" {
		note = "Please take another look at this."
	}

	status := store.StatusTodo
	updated, err := s.store.UpdateTask(r.Context(), id.UserID, task.ID, store.TaskPatch{Status: &status})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't send that back. Please try again.")
		return
	}

	_, _ = s.store.AddComment(r.Context(), task.ID, "you", id.Name, "Sent back: "+note)
	_ = s.store.AddActivity(r.Context(), id.UserID, "sent_back", "You sent “"+task.Title+"” back", task.ID, "")

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": updated})
}
