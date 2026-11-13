package httpapi

import (
	"net/http"

	"github.com/davepartner/crew/internal/skills"
)

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"skills":     skills.All(),
		"categories": skills.Categories(),
		"count":      skills.Count(),
		"playbooks":  skills.Playbooks(),
		"sources": []map[string]string{
			{"name": skills.SourceName, "url": skills.SourceURL, "license": "MIT"},
			{"name": skills.RemName, "url": skills.RemURL, "license": "MIT"},
		},
	})
}

func (s *Server) handleSkill(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	skill, ok := skills.ByKey(key)
	if !ok {
		writeError(w, http.StatusNotFound, "We don't have that skill.")
		return
	}

	body, _ := skills.Body(key)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"skill": skill,
		"body":  body,
	})
}
