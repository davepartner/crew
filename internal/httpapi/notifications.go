package httpapi

import "net/http"

func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	items, unread, err := s.store.ListNotifications(r.Context(), id.UserID, company.ID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load your notifications.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "notifications": items, "unread": unread})
}

func (s *Server) handleReadNotification(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)

	if err := s.store.MarkNotification(r.Context(), id.UserID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't mark that as read.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleReadAllNotifications(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	if err := s.store.MarkAllNotifications(r.Context(), id.UserID, company.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't mark those as read.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
