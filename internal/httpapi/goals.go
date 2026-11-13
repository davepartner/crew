package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/davepartner/crew/internal/store"
)

func (s *Server) handleListGoals(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	goals, err := s.store.ListGoals(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your goals. Please refresh.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "goals": goals})
}

func (s *Server) handleCreateGoal(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	var body struct {
		Title   string `json:"title"`
		Target  int    `json:"target"`
		DueDate string `json:"dueDate"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	title := trim(body.Title, 160)
	if len([]rune(title)) < 1 {
		writeError(w, http.StatusUnprocessableEntity, "Give the goal a title.")
		return
	}

	if body.Target < 1 {
		body.Target = 1
	}
	if body.Target > 100000 {
		body.Target = 100000
	}

	if body.DueDate != "" {
		if _, err := time.Parse("2006-01-02", body.DueDate); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "Use a date like 2027-03-31.")
			return
		}
	}

	goal, err := s.store.CreateGoal(r.Context(), identityFrom(r).UserID, company.ID, title, body.Target, body.DueDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that goal. Please try again.")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "goal": goal})
}

func (s *Server) handleUpdateGoal(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	var body struct {
		Title   *string `json:"title"`
		Target  *int    `json:"target"`
		Current *int    `json:"current"`
		DueDate *string `json:"dueDate"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	patch := store.GoalPatch{
		Target:  body.Target,
		Current: body.Current,
		DueDate: body.DueDate,
	}

	if body.Title != nil {
		title := trim(*body.Title, 160)
		if len([]rune(title)) < 1 {
			writeError(w, http.StatusUnprocessableEntity, "Give the goal a title.")
			return
		}
		patch.Title = &title
	}

	if body.DueDate != nil && *body.DueDate != "" {
		if _, err := time.Parse("2006-01-02", *body.DueDate); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "Use a date like 2027-03-31.")
			return
		}
	}

	goal, err := s.store.UpdateGoal(r.Context(), company.ID, r.PathValue("id"), patch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "That goal doesn't exist.")
			return
		}
		writeError(w, http.StatusInternalServerError, "We couldn't save that goal. Please try again.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "goal": goal})
}

func (s *Server) handleDeleteGoal(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	if err := s.store.DeleteGoal(r.Context(), company.ID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "That goal doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
