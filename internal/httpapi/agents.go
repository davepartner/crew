package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/skills"
	"github.com/davepartner/crew/internal/store"
)

var allowedRoles = map[string]bool{
	"Operations":         true,
	"Support":            true,
	"Marketing":          true,
	"Product":            true,
	"Engineering":        true,
	"Client success":     true,
	"Delivery":           true,
	"Editorial":          true,
	"Production":         true,
	"Audience":           true,
	"Patient experience": true,
	"Clinical support":   true,
	"Sales":              true,
	"Reputation":         true,
	"Compliance":         true,
	"Analysis":           true,
	"Enrolment":          true,
	"Teaching":           true,
	"Community":          true,
	"Reservations":       true,
	"Guest experience":   true,
	"Programs":           true,
	"Volunteers":         true,
	"Fundraising":        true,
	"Researcher":         true,
	"Writer":             true,
	"Coder":              true,
	"Designer":           true,
	"Planner":            true,
	"Helper":             true,
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func newAgentKey() string {
	return "crew_" + randomToken(24)
}

type agentView struct {
	store.Agent
	Status       string `json:"status"`
	OpenTasks    int    `json:"open_tasks"`
	WorkingTasks int    `json:"working_tasks"`
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	agents, err := s.store.ListAgents(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your helpers. Please refresh.")
		return
	}

	tasks, err := s.store.ListTasks(r.Context(), company.ID, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your helpers. Please refresh.")
		return
	}

	open := map[string][2]int{}
	for _, task := range tasks {
		if task.AgentID == nil {
			continue
		}
		counts := open[*task.AgentID]
		if task.Status == store.StatusWorking {
			counts[1]++
		}
		if task.Status == store.StatusTodo || task.Status == store.StatusWorking {
			counts[0]++
		}
		open[*task.AgentID] = counts
	}

	views := make([]agentView, 0, len(agents))
	for _, agent := range agents {
		counts := open[agent.ID]
		views = append(views, agentView{
			Agent:        agent,
			Status:       agentStatus(agent, counts[1]),
			OpenTasks:    counts[0],
			WorkingTasks: counts[1],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "agents": views})
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	var body struct {
		Name         string   `json:"name"`
		Role         string   `json:"role"`
		Emoji        string   `json:"emoji"`
		Instructions string   `json:"instructions"`
		Skills       []string `json:"skills"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := trim(body.Name, 40)
	if len([]rune(name)) < 1 {
		writeError(w, http.StatusUnprocessableEntity, "Give your helper a name.")
		return
	}

	role := body.Role
	if !allowedRoles[role] {
		role = "Helper"
	}

	emoji := strings.TrimSpace(body.Emoji)
	if emoji == "" || len([]rune(emoji)) > 4 {
		emoji = "🤖"
	}

	count, err := s.store.CountAgents(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't add that helper. Please try again.")
		return
	}
	if count >= 50 {
		writeError(w, http.StatusUnprocessableEntity, "You have 50 helpers, which is the limit. Remove one before adding another.")
		return
	}

	if err := skills.Validate(body.Skills); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	key := newAgentKey()

	agent, err := s.store.CreateAgent(r.Context(), id.UserID, company.ID, name, role, emoji,
		trim(body.Instructions, 4000), body.Skills, hashKey(key))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't add that helper. Please try again.")
		return
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, company.ID, "agent_created", "You added "+agent.Name, "", agent.ID)

	writeJSON(w, http.StatusCreated, map[string]any{
		"ok": true,
		"agent": agentView{
			Agent:  agent,
			Status: "not_connected",
		},
		"apiKey": key,
	})
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	agent, err := s.store.GetAgent(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That helper doesn't exist.")
		return
	}

	tasks, err := s.store.OpenTasksForAgent(r.Context(), company.ID, agent.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load that helper. Please refresh.")
		return
	}

	spend, _ := s.store.AgentSpend(r.Context(), company.ID, agent.ID, store.MonthStart())

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"agent":     agent,
		"tasks":     tasks,
		"status":    agentStatus(agent, countWorking(tasks)),
		"spend_usd": spend,
	})
}

func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	agentID := r.PathValue("id")

	current, err := s.store.GetAgent(r.Context(), company.ID, agentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "That helper doesn't exist.")
		return
	}

	var body struct {
		Name             *string   `json:"name"`
		Role             *string   `json:"role"`
		Emoji            *string   `json:"emoji"`
		Instructions     *string   `json:"instructions"`
		ModelProvider    *string   `json:"modelProvider"`
		ModelName        *string   `json:"modelName"`
		Skills           *[]string `json:"skills"`
		MonthlyBudgetUSD *float64  `json:"monthlyBudgetUsd"`
		CanDelegate      *bool     `json:"canDelegate"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := current.Name
	if body.Name != nil {
		name = trim(*body.Name, 40)
		if len([]rune(name)) < 1 {
			writeError(w, http.StatusUnprocessableEntity, "Give your helper a name.")
			return
		}
	}

	role := current.Role
	if body.Role != nil && allowedRoles[*body.Role] {
		role = *body.Role
	}

	emoji := current.Emoji
	if body.Emoji != nil {
		if value := strings.TrimSpace(*body.Emoji); value != "" && len([]rune(value)) <= 4 {
			emoji = value
		}
	}

	instructions := current.Instructions
	if body.Instructions != nil {
		instructions = trim(*body.Instructions, 4000)
	}

	provider := current.ModelProvider
	model := current.ModelName
	if body.ModelProvider != nil {
		provider = strings.TrimSpace(*body.ModelProvider)
	}
	if body.ModelName != nil {
		model = strings.TrimSpace(*body.ModelName)
	}
	if provider == "" {
		model = ""
	}

	skillKeys := current.Skills
	if body.Skills != nil {
		if err := skills.Validate(*body.Skills); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		skillKeys = *body.Skills
	}

	budget := current.MonthlyBudgetUSD
	if body.MonthlyBudgetUSD != nil {
		budget = *body.MonthlyBudgetUSD
		if budget < 0 {
			budget = 0
		}
	}

	canDelegate := current.CanDelegate
	if body.CanDelegate != nil {
		canDelegate = *body.CanDelegate
	}

	agent, err := s.store.UpdateAgent(r.Context(), company.ID, agentID, name, role, emoji, instructions, provider, model, skillKeys, budget, canDelegate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that. Please try again.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "agent": agent})
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	if err := s.store.DeleteAgent(r.Context(), company.ID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "That helper doesn't exist.")
		return
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, company.ID, "agent_removed", "A helper was removed", "", "")

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRotateAgentKey(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	agentID := r.PathValue("id")

	key := newAgentKey()

	if err := s.store.RotateAgentKey(r.Context(), company.ID, agentID, hashKey(key)); err != nil {
		writeError(w, http.StatusNotFound, "That helper doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "apiKey": key})
}

func agentStatus(agent store.Agent, working int) string {
	if working > 0 {
		return "working"
	}
	if agent.LastSeenAt == nil {
		return "not_connected"
	}
	if time.Since(*agent.LastSeenAt) < 2*time.Minute {
		return "ready"
	}
	return "offline"
}

func countWorking(tasks []store.Task) int {
	count := 0
	for _, task := range tasks {
		if task.Status == store.StatusWorking {
			count++
		}
	}
	return count
}
