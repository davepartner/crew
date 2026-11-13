package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/davepartner/crew/internal/store"
)

func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	members, err := s.store.ListMembers(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load the team. Please refresh.")
		return
	}

	invites := []store.Invite{}
	if company.Role == "owner" {
		invites, _ = s.store.ListInvites(r.Context(), company.ID)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"members": members,
		"invites": invites,
		"role":    company.Role,
	})
}

func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" || !supportEmailPattern.MatchString(email) {
		writeError(w, http.StatusUnprocessableEntity, "Enter a valid email address.")
		return
	}
	if body.Role != "viewer" {
		body.Role = "editor"
	}

	user, err := s.store.FindUserByEmail(r.Context(), email)
	if err == nil {
		if user.ID == identityFrom(r).UserID {
			writeError(w, http.StatusUnprocessableEntity, "You're already on this team.")
			return
		}
		if err := s.store.AddMember(r.Context(), company.ID, user.ID, body.Role); err != nil {
			writeError(w, http.StatusInternalServerError, "We couldn't add them. Please try again.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "member": map[string]any{
			"user_id": user.ID, "name": user.Name, "email": user.Email, "role": body.Role, "is_owner": false,
		}})
		return
	}
	if !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "We couldn't look up that email.")
		return
	}

	invite, err := s.store.CreateInvite(r.Context(), company.ID, email, body.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that invitation.")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "invite": invite})
}

func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	userID := r.PathValue("id")

	members, err := s.store.ListMembers(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load the team.")
		return
	}
	for _, member := range members {
		if member.UserID == userID && member.IsOwner {
			writeError(w, http.StatusUnprocessableEntity, "The owner can't be removed.")
			return
		}
	}

	if err := s.store.RemoveMember(r.Context(), company.ID, userID); err != nil {
		writeError(w, http.StatusNotFound, "That person isn't on this team.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteInvite(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	if err := s.store.DeleteInvite(r.Context(), company.ID, r.PathValue("id")); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't remove that invitation.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
