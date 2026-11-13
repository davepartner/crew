package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/davepartner/crew/internal/store"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	user, err := s.store.GetUser(r.Context(), id.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Please sign in again.")
		return
	}

	tasks, err := s.store.ListTasks(r.Context(), company.ID, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your dashboard. Please refresh.")
		return
	}

	agents, err := s.store.ListAgents(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your helpers. Please refresh.")
		return
	}

	activity, err := s.store.ListActivity(r.Context(), company.ID, 8)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load recent activity. Please refresh.")
		return
	}

	goals, err := s.store.ListGoals(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your goals. Please refresh.")
		return
	}

	counts, err := s.store.StatusCounts(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your dashboard. Please refresh.")
		return
	}

	needsYou := []store.Task{}
	working := []store.Task{}
	upNext := []store.Task{}

	for _, task := range tasks {
		switch task.Status {
		case store.StatusReview:
			if len(needsYou) < 4 {
				needsYou = append(needsYou, task)
			}
		case store.StatusWorking:
			if len(working) < 4 {
				working = append(working, task)
			}
		case store.StatusTodo:
			if len(upNext) < 4 {
				upNext = append(upNext, task)
			}
		}
	}

	openByAgent := map[string]int{}
	workingByAgent := map[string]int{}
	for _, task := range tasks {
		if task.AgentID == nil {
			continue
		}
		if task.Status == store.StatusWorking {
			workingByAgent[*task.AgentID]++
		}
		if task.Status == store.StatusTodo || task.Status == store.StatusWorking {
			openByAgent[*task.AgentID]++
		}
	}

	views := make([]agentView, 0, len(agents))
	for _, agent := range agents {
		views = append(views, agentView{
			Agent:        agent,
			Status:       agentStatus(agent, workingByAgent[agent.ID]),
			OpenTasks:    openByAgent[agent.ID],
			WorkingTasks: workingByAgent[agent.ID],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"greeting": greeting(),
		"name":     user.Name,
		"company": map[string]any{
			"id":           company.ID,
			"name":         company.Name,
			"business_key": company.BusinessKey,
			"website":      company.Website,
		},
		"counts":   counts,
		"needsYou": needsYou,
		"working":  working,
		"upNext":   upNext,
		"agents":   views,
		"activity": activity,
		"goals":    goals,
	})
}

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	items, err := s.store.ListActivity(r.Context(), company.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load activity. Please refresh.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "activity": items})
}

func greeting() string {
	hour := time.Now().Hour()
	switch {
	case hour < 12:
		return "Good morning"
	case hour < 18:
		return "Good afternoon"
	default:
		return "Good evening"
	}
}
