package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/modelcatalog"
	"github.com/davepartner/crew/internal/presets"
	"github.com/davepartner/crew/internal/store"
)

var supportEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)

type companyInput struct {
	Name         string `json:"name"`
	BusinessKey  string `json:"businessKey"`
	Website      string `json:"website"`
	SupportEmail string `json:"supportEmail"`
	Description  string `json:"description"`
	Audience     string `json:"audience"`
	Tone         string `json:"tone"`
	CreateTeam   *bool  `json:"createTeam"`
}

func (s *Server) handleListCompanies(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	companies, err := s.store.ListCompanies(r.Context(), id.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your companies. Please refresh.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "companies": companies})
}

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	var body companyInput
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := trim(body.Name, 80)
	if len([]rune(name)) < 1 {
		writeError(w, http.StatusUnprocessableEntity, "Give your company a name.")
		return
	}

	website := strings.TrimSpace(body.Website)
	if website != "" && !strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://") {
		website = "https://" + website
	}

	support := strings.ToLower(strings.TrimSpace(body.SupportEmail))
	if support != "" && (len(support) > 254 || !supportEmailPattern.MatchString(support)) {
		writeError(w, http.StatusUnprocessableEntity, "That support email doesn't look right.")
		return
	}

	businessKey := strings.TrimSpace(body.BusinessKey)
	if businessKey == "" {
		businessKey = "custom"
	}

	business, ok := presets.ByKey(businessKey)
	if !ok {
		business = presets.CustomTeam()
		businessKey = "custom"
	}

	company, err := s.store.CreateCompany(r.Context(), id.UserID, store.CompanyInput{
		Name:         name,
		BusinessKey:  businessKey,
		Website:      website,
		SupportEmail: support,
		Description:  trim(body.Description, 2000),
		Audience:     trim(body.Audience, 200),
		Tone:         trim(body.Tone, 40),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't create that company. Please try again.")
		return
	}

	createdAgents := 0
	createdTasks := 0

	if body.CreateTeam == nil || *body.CreateTeam {
		vars := presets.Vars{
			Company:  company.Name,
			Website:  company.Website,
			Support:  company.SupportEmail,
			Audience: company.Audience,
			Tone:     company.Tone,
		}

		agentByName := map[string]string{}
		agentByRole := map[string]string{}

		for _, preset := range business.Agents {
			agent, err := s.store.CreateAgent(r.Context(), id.UserID, company.ID, preset.Name, preset.Role,
				preset.Emoji, presets.Render(preset.Instructions, vars), hashKey(newAgentKey()))
			if err != nil {
				continue
			}
			agentByName[preset.Name] = agent.ID
			agentByRole[preset.Role] = agent.ID
			createdAgents++
		}

		for _, preset := range business.Tasks {
			agentID := agentByName[preset.Agent]
			if agentID == "" {
				agentID = agentByRole[preset.Agent]
			}
			_, err := s.store.CreateTask(r.Context(), id.UserID, company.ID,
				presets.Render(preset.Title, vars), presets.Render(preset.Details, vars), agentID)
			if err == nil {
				createdTasks++
			}
		}
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, company.ID, "company_created",
		"You started "+company.Name, "", "")

	writeJSON(w, http.StatusCreated, map[string]any{
		"ok":            true,
		"company":       company,
		"agentsCreated": createdAgents,
		"tasksCreated":  createdTasks,
	})
}

func (s *Server) handleGetCompany(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	company, err := s.store.GetCompany(r.Context(), id.UserID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That company doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "company": company})
}

func (s *Server) handleUpdateCompany(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	current, err := s.store.GetCompany(r.Context(), id.UserID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That company doesn't exist.")
		return
	}

	var body companyInput
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := current.Name
	if strings.TrimSpace(body.Name) != "" {
		name = trim(body.Name, 80)
	}

	website := current.Website
	if body.Website != "" {
		website = strings.TrimSpace(body.Website)
		if !strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://") {
			website = "https://" + website
		}
	}

	support := current.SupportEmail
	if body.SupportEmail != "" {
		support = strings.ToLower(strings.TrimSpace(body.SupportEmail))
		if !supportEmailPattern.MatchString(support) {
			writeError(w, http.StatusUnprocessableEntity, "That support email doesn't look right.")
			return
		}
	}

	company, err := s.store.UpdateCompany(r.Context(), id.UserID, current.ID, store.CompanyInput{
		Name:         name,
		BusinessKey:  current.BusinessKey,
		Website:      website,
		SupportEmail: support,
		Description:  trim(body.Description, 2000),
		Audience:     trim(body.Audience, 200),
		Tone:         trim(body.Tone, 40),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that company. Please try again.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "company": company})
}

func (s *Server) handleDeleteCompany(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	if err := s.store.DeleteCompany(r.Context(), id.UserID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "That company doesn't exist.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePresets(w http.ResponseWriter, r *http.Request) {
	list := make([]map[string]any, 0, len(presets.All()))
	for _, business := range presets.All() {
		list = append(list, map[string]any{
			"key":          business.Key,
			"name":         business.Name,
			"category":     business.Category,
			"emoji":        business.Emoji,
			"blurb":        business.Blurb,
			"agents":       len(business.Agents),
			"tasks":        len(business.Tasks),
			"integrations": business.Integrations,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"businesses": list,
		"categories": presets.Categories(),
	})
}

func (s *Server) handlePreset(w http.ResponseWriter, r *http.Request) {
	business, ok := presets.ByKey(r.PathValue("key"))
	if !ok {
		writeError(w, http.StatusNotFound, "We don't have a preset for that business yet.")
		return
	}

	agents := make([]map[string]any, 0, len(business.Agents))
	for _, agent := range business.Agents {
		agents = append(agents, map[string]any{
			"name":  agent.Name,
			"role":  agent.Role,
			"emoji": agent.Emoji,
		})
	}

	tasks := make([]map[string]any, 0, len(business.Tasks))
	for _, task := range business.Tasks {
		tasks = append(tasks, map[string]any{
			"title": task.Title,
			"agent": task.Agent,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"key":          business.Key,
		"name":         business.Name,
		"category":     business.Category,
		"emoji":        business.Emoji,
		"blurb":        business.Blurb,
		"agents":       agents,
		"tasks":        tasks,
		"integrations": business.Integrations,
	})
}

/* ------------------------------ providers ------------------------------ */

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	keys, _ := s.store.ListProviderKeys(r.Context(), id.UserID)
	configured := map[string]store.ProviderKey{}
	for _, key := range keys {
		configured[key.Provider] = key
	}

	list := make([]map[string]any, 0, len(modelcatalog.All()))
	for _, provider := range modelcatalog.Public() {
		key, ok := configured[provider.Key]
		list = append(list, map[string]any{
			"key":           provider.Key,
			"name":          provider.Name,
			"kind":          provider.Kind,
			"base_url":      provider.BaseURL,
			"keys_url":      provider.KeysURL,
			"docs_url":      provider.DocsURL,
			"free":          provider.Free,
			"free_note":     provider.FreeNote,
			"default_model": provider.DefaultModel,
			"models":        provider.Models,
			"configured":    ok,
			"last4":         key.Last4,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "providers": list})
}

func (s *Server) handleSetProviderKey(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	providerKey := r.PathValue("key")

	provider, ok := modelcatalog.ByKey(providerKey)
	if !ok {
		writeError(w, http.StatusNotFound, "We don't know that model provider.")
		return
	}

	var body struct {
		APIKey string `json:"apiKey"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	key := strings.TrimSpace(body.APIKey)
	if len(key) < 8 || len(key) > 400 {
		writeError(w, http.StatusUnprocessableEntity, "That doesn't look like an API key.")
		return
	}

	cipherText, err := s.cipher.Encrypt(key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that key.")
		return
	}

	last4 := key
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}

	if err := s.store.UpsertProviderKey(r.Context(), id.UserID, provider.Key, cipherText, last4); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that key.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"provider":   provider.Key,
		"configured": true,
		"last4":      last4,
	})
}

func (s *Server) handleDeleteProviderKey(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	if err := s.store.DeleteProviderKey(r.Context(), id.UserID, r.PathValue("key")); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't remove that key.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleTestProvider(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	provider, ok := modelcatalog.ByKey(r.PathValue("key"))
	if !ok {
		writeError(w, http.StatusNotFound, "We don't know that model provider.")
		return
	}

	cipherText, err := s.store.ProviderKeyCipher(r.Context(), id.UserID, provider.Key)
	if err != nil {
		writeError(w, http.StatusNotFound, "Add a key for that provider first.")
		return
	}

	key, err := s.cipher.Decrypt(cipherText)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't read that key.")
		return
	}

	okay, message := testProvider(r.Context(), provider, key)
	writeJSON(w, http.StatusOK, map[string]any{"ok": okay, "message": message})
}

func testProvider(ctx context.Context, provider modelcatalog.Provider, key string) (bool, string) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	var request *http.Request
	var err error

	switch provider.Kind {
	case "anthropic":
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, provider.BaseURL+"/models", nil)
		if err == nil {
			request.Header.Set("x-api-key", key)
			request.Header.Set("anthropic-version", "2023-06-01")
		}
	case "google":
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, provider.BaseURL+"/models?key="+key, nil)
	default:
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, provider.BaseURL+"/models", nil)
		if err == nil {
			request.Header.Set("Authorization", "Bearer "+key)
		}
	}

	if err != nil {
		return false, "We couldn't build that request."
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return false, "We couldn't reach " + provider.Name + ". Check the key and try again."
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))

	switch {
	case response.StatusCode == http.StatusOK:
		return true, "That key works."
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
		return false, "That key was rejected. Check it in " + provider.Name + "."
	case response.StatusCode == http.StatusTooManyRequests:
		return true, "The key is valid but rate-limited right now."
	default:
		var body struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&body)
		if body.Error.Message != "" {
			return false, trim(body.Error.Message, 200)
		}
		return false, "That provider returned " + response.Status + "."
	}
}
