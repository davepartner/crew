package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/mailer"
)

func (s *Server) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	var body struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	to := trim(body.To, 200)
	subject := trim(body.Subject, 200)
	if to == "" || subject == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add a recipient and a subject.")
		return
	}
	if body.Body == "" {
		body.Body = subject
	}

	err := mailer.Send(mailer.Config{
		Addr: s.cfg.SMTPAddr, From: s.cfg.SMTPFrom, User: s.cfg.SMTPUser, Pass: s.cfg.SMTPPass,
	}, to, subject, body.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The email couldn't be sent: "+err.Error())
		return
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, company.ID, "email_sent",
		"You emailed "+to, "", "")

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSendWhatsApp(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	config, err := s.loadIntegration(r.Context(), company.ID, "whatsapp")
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"ok": false, "code": "not_connected",
			"error": "Connect WhatsApp in Settings → Integrations first.",
		})
		return
	}

	var body struct {
		To       string `json:"to"`
		Message  string `json:"message"`
		Template bool   `json:"template"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	to := strings.TrimPrefix(strings.TrimSpace(body.To), "+")
	if to == "" {
		to = config["default_to"]
	}
	message := trim(body.Message, 3000)
	if to == "" || message == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add a recipient and a message.")
		return
	}

	var payload string
	if body.Template {
		name := strings.TrimSpace(config["template_name"])
		language := strings.TrimSpace(config["template_language"])
		if name == "" || language == "" {
			writeError(w, http.StatusUnprocessableEntity, "Add an approved template name and language in Settings → Integrations.")
			return
		}
		encoded, _ := json.Marshal(map[string]any{
			"messaging_product": "whatsapp", "to": to, "type": "template",
			"template": map[string]any{
				"name": name, "language": map[string]string{"code": language},
				"components": []map[string]any{{
					"type":       "body",
					"parameters": []map[string]any{{"type": "text", "text": message}},
				}},
			},
		})
		payload = string(encoded)
	} else {
		payload = fmt.Sprintf(`{"messaging_product":"whatsapp","to":%q,"type":"text","text":{"body":%q}}`, to, message)
	}
	endpoint := "https://graph.facebook.com/v20.0/" + url.PathEscape(config["phone_number_id"]) + "/messages"

	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, strings.NewReader(payload))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't send that message.")
		return
	}
	request.Header.Set("Authorization", "Bearer "+config["token"])
	request.Header.Set("Content-Type", "application/json")

	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "WhatsApp couldn't be reached: "+err.Error())
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		writeError(w, http.StatusBadGateway, "WhatsApp rejected that message. Check the 24-hour window and the number.")
		return
	}

	_ = s.store.AddActivity(r.Context(), id.UserID, company.ID, "whatsapp_sent",
		"You messaged +"+to+" on WhatsApp", "", "")

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
