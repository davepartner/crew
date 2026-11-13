package httpapi

import (
	"errors"
	"net/http"

	"github.com/davepartner/crew/internal/store"
)

func (s *Server) handleListRoutines(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	routines, err := s.store.ListRoutines(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your routines. Please refresh.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "routines": routines})
}

func (s *Server) handleCreateRoutine(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	var body struct {
		Title   string `json:"title"`
		Details string `json:"details"`
		AgentID string `json:"agentId"`
		Cadence string `json:"cadence"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	title := trim(body.Title, 200)
	if len([]rune(title)) < 1 {
		writeError(w, http.StatusUnprocessableEntity, "Give the routine a title.")
		return
	}

	if !store.ValidCadence(body.Cadence) {
		writeError(w, http.StatusUnprocessableEntity, "Choose daily, weekdays, weekly or monthly.")
		return
	}

	agentID := ""
	if body.AgentID != "" {
		agent, err := s.store.GetAgent(r.Context(), company.ID, body.AgentID)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "That helper doesn't exist.")
			return
		}
		agentID = agent.ID
	}

	routine, err := s.store.CreateRoutine(r.Context(), id.UserID, company.ID, title, trim(body.Details, 8000), agentID, body.Cadence)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that routine. Please try again.")
		return
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, company.ID, "routine_created",
		"You scheduled “"+routine.Title+"”", "", agentID)

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "routine": routine})
}

func (s *Server) handleUpdateRoutine(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	var body struct {
		Title   *string `json:"title"`
		Details *string `json:"details"`
		AgentID *string `json:"agentId"`
		Cadence *string `json:"cadence"`
		Enabled *bool   `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	patch := store.RoutinePatch{
		Details: body.Details,
		Enabled: body.Enabled,
	}

	if body.Title != nil {
		title := trim(*body.Title, 200)
		if len([]rune(title)) < 1 {
			writeError(w, http.StatusUnprocessableEntity, "Give the routine a title.")
			return
		}
		patch.Title = &title
	}

	if body.Cadence != nil {
		if !store.ValidCadence(*body.Cadence) {
			writeError(w, http.StatusUnprocessableEntity, "Choose daily, weekdays, weekly or monthly.")
			return
		}
		patch.Cadence = body.Cadence
	}

	if body.AgentID != nil {
		value := *body.AgentID
		if value != "" {
			if _, err := s.store.GetAgent(r.Context(), company.ID, value); err != nil {
				writeError(w, http.StatusUnprocessableEntity, "That helper doesn't exist.")
				return
			}
		}
		patch.AgentID = &value
	}

	routine, err := s.store.UpdateRoutine(r.Context(), company.ID, r.PathValue("id"), patch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "That routine doesn't exist.")
			return
		}
		writeError(w, http.StatusInternalServerError, "We couldn't save that routine. Please try again.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "routine": routine})
}

func (s *Server) handleDeleteRoutine(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	if err := s.store.DeleteRoutine(r.Context(), company.ID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "That routine doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
