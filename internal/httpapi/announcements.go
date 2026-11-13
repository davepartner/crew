package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

type appAnnouncement struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Level     string `json:"level"`
	Published string `json:"published_at"`
}

func (s *Server) handleAnnouncements(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AdminToken == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "announcements": []appAnnouncement{}})
		return
	}

	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
		s.cfg.PaytrackifyURL+"/api/app/announcements", nil)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "announcements": []appAnnouncement{}})
		return
	}
	request.Header.Set("X-Crew-Token", s.cfg.AdminToken)
	request.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "announcements": []appAnnouncement{}})
		return
	}
	defer response.Body.Close()

	var payload struct {
		Announcements []appAnnouncement `json:"announcements"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&payload) != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "announcements": []appAnnouncement{}})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "announcements": payload.Announcements})
}
