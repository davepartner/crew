package httpapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/llm"
	"github.com/davepartner/crew/internal/mailer"
	"github.com/davepartner/crew/internal/research"
	"github.com/davepartner/crew/internal/store"
)

type outreachPayload struct {
	Prospects    []store.OutreachProspect    `json:"prospects"`
	Campaigns    []store.OutreachCampaign    `json:"campaigns"`
	Messages     []store.OutreachMessage     `json:"messages"`
	Suppressions []store.OutreachSuppression `json:"suppressions"`
	Settings     store.OutreachSettings      `json:"settings"`
	Stats        map[string]int              `json:"stats"`
	SendReady    bool                        `json:"send_ready"`
	FromAddress  string                      `json:"from_address"`
}

var outreachEmailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
var roleEmailPattern = regexp.MustCompile(`^(info|sales|support|hello|contact|admin|office|team|hi)@`)

func (s *Server) handleOutreach(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	payload, err := s.outreachData(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load outreach.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "outreach": payload, "company": company})
}

func (s *Server) handleOutreachProspect(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	var body struct {
		Name     string `json:"name"`
		Website  string `json:"website"`
		Industry string `json:"industry"`
		Location string `json:"location"`
		Notes    string `json:"notes"`
		Email    string `json:"email"`
		Contact  string `json:"contact"`
		Role     string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if trim(body.Name, 160) == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add a company name.")
		return
	}
	website := normalizeWebsite(body.Website)
	domain := domainOf(website)
	prospect, err := s.store.CreateOutreachProspect(r.Context(), company.ID, store.OutreachProspect{
		Name: trim(body.Name, 160), Website: website, Domain: domain,
		Industry: trim(body.Industry, 120), Location: trim(body.Location, 120),
		Notes: trim(body.Notes, 2000), Source: "manual",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't add that prospect.")
		return
	}
	email := strings.ToLower(trim(body.Email, 200))
	if email != "" {
		contact, _ := s.store.AddOutreachContact(r.Context(), company.ID, prospect.ID, store.OutreachContact{
			Name: trim(body.Contact, 120), Role: trim(body.Role, 120), Email: email,
			EmailStatus: verifyEmailStatus(r.Context(), email), Source: "manual", IsPrimary: true,
		})
		prospect.Contacts = []store.OutreachContact{contact}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "prospect": prospect})
}

func (s *Server) handleOutreachImport(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Choose a CSV file.")
		return
	}
	defer file.Close()

	reader := csv.NewReader(io.LimitReader(file, 8<<20))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		writeError(w, http.StatusUnprocessableEntity, "That CSV couldn't be read.")
		return
	}
	headerRow := rows[0]
	index := map[string]int{}
	for i, column := range headerRow {
		index[strings.ToLower(strings.TrimSpace(column))] = i
	}
	value := func(row []string, key string) string {
		i, ok := index[key]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	if _, ok := index["company_name"]; !ok {
		if _, ok := index["company"]; !ok {
			writeError(w, http.StatusUnprocessableEntity, "The CSV needs a company_name column.")
			return
		}
	}

	existing, _ := s.store.ListOutreachProspects(r.Context(), company.ID)
	known := map[string]bool{}
	for _, prospect := range existing {
		known[strings.ToLower(prospect.Domain)] = true
		contacts, _ := s.store.ListOutreachContacts(r.Context(), company.ID, prospect.ID)
		for _, contact := range contacts {
			known[strings.ToLower(contact.Email)] = true
		}
	}

	imported := 0
	for _, row := range rows[1:] {
		name := value(row, "company_name")
		if name == "" {
			name = value(row, "company")
		}
		if name == "" {
			continue
		}
		email := strings.ToLower(value(row, "email"))
		website := normalizeWebsite(value(row, "website"))
		domain := domainOf(website)
		if domain == "" && strings.Contains(email, "@") {
			domain = strings.Split(email, "@")[1]
		}
		if known[strings.ToLower(domain)] || (email != "" && known[strings.ToLower(email)]) {
			continue
		}

		score, _ := parseScore(value(row, "score"))
		stage := value(row, "status")
		if stage == "" {
			stage = "new"
		}
		prospect, err := s.store.CreateOutreachProspect(r.Context(), company.ID, store.OutreachProspect{
			Name: trim(name, 160), Website: website, Domain: domain,
			Industry: trim(value(row, "industry"), 120), Source: "sellbird/csv",
			Stage: stage, Score: score, Notes: trim(value(row, "personalization"), 2000),
		})
		if err != nil {
			continue
		}
		if email != "" {
			status := value(row, "email_status")
			if status == "" {
				status = verifyEmailStatus(r.Context(), email)
			}
			_, _ = s.store.AddOutreachContact(r.Context(), company.ID, prospect.ID, store.OutreachContact{
				Name: trim(strings.TrimSpace(value(row, "first_name")+" "+value(row, "last_name")), 120),
				Role: trim(value(row, "title"), 120), Email: email, EmailStatus: status,
				Source: "sellbird/csv", IsPrimary: true,
			})
		}
		known[strings.ToLower(domain)] = true
		if email != "" {
			known[strings.ToLower(email)] = true
		}
		imported++
	}

	_ = header
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "imported": imported})
}

func (s *Server) handleOutreachDiscover(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	var body struct {
		Query    string `json:"query"`
		Location string `json:"location"`
		Limit    int    `json:"limit"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if trim(body.Query, 120) == "" || trim(body.Location, 120) == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add what to look for and a city or region.")
		return
	}
	if body.Limit <= 0 || body.Limit > 30 {
		body.Limit = 15
	}

	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()
	results, err := discoverOSM(ctx, trim(body.Query, 120), trim(body.Location, 120), body.Limit, s.cfg.SearchURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Discovery failed: "+err.Error())
		return
	}

	created := []store.OutreachProspect{}
	for _, item := range results {
		prospect, err := s.store.CreateOutreachProspect(r.Context(), company.ID, item)
		if err == nil {
			created = append(created, prospect)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "prospects": created, "found": len(results)})
}

func (s *Server) handleOutreachScan(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	prospect, err := s.store.GetOutreachProspect(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That prospect doesn't exist.")
		return
	}
	if prospect.Website == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add a website first.")
		return
	}

	page, err := fetchPage(r.Context(), prospect.Website)
	if err != nil {
		writeError(w, http.StatusBadGateway, "We couldn't read that website: "+err.Error())
		return
	}
	signals, score := outreachSignals(page.Text, page.HTML)
	prospect.Signals = signals
	prospect.Score = score
	prospect.Notes = page.Title
	if err := s.store.UpdateOutreachProspect(r.Context(), company.ID, prospect.ID, prospect); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save the signals.")
		return
	}

	contacts, _ := s.store.ListOutreachContacts(r.Context(), company.ID, prospect.ID)
	if len(contacts) == 0 {
		for _, email := range extractEmails(page.HTML + " " + page.Text) {
			status := verifyEmailStatus(r.Context(), email)
			contact, err := s.store.AddOutreachContact(r.Context(), company.ID, prospect.ID, store.OutreachContact{
				Name: contactName(email), Role: "Contact", Email: email, EmailStatus: status,
				Source: "website scan", IsPrimary: true,
			})
			if err == nil {
				contacts = append(contacts, contact)
				break
			}
		}
	}
	prospect.Contacts = contacts

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "prospect": prospect})
}

func (s *Server) handleOutreachContact(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	prospectID := r.PathValue("id")
	if _, err := s.store.GetOutreachProspect(r.Context(), company.ID, prospectID); err != nil {
		writeError(w, http.StatusNotFound, "That prospect doesn't exist.")
		return
	}
	var body store.OutreachContact
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Email = strings.ToLower(trim(body.Email, 200))
	if body.Email == "" || !strings.Contains(body.Email, "@") {
		writeError(w, http.StatusUnprocessableEntity, "Add a valid email.")
		return
	}
	body.EmailStatus = verifyEmailStatus(r.Context(), body.Email)
	contact, err := s.store.AddOutreachContact(r.Context(), company.ID, prospectID, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't add that contact.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "contact": contact})
}

func (s *Server) handleOutreachCampaign(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	var body struct {
		store.OutreachCampaign
		Steps []store.OutreachStep `json:"steps"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if trim(body.Name, 160) == "" {
		writeError(w, http.StatusUnprocessableEntity, "Give the campaign a name.")
		return
	}
	campaign, err := s.store.CreateOutreachCampaign(r.Context(), company.ID, body.OutreachCampaign)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't create that campaign.")
		return
	}
	if len(body.Steps) > 0 {
		_ = s.store.ReplaceOutreachSteps(r.Context(), company.ID, campaign.ID, body.Steps)
		campaign.Steps, _ = s.store.ListOutreachSteps(r.Context(), company.ID, campaign.ID)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "campaign": campaign})
}

func (s *Server) handleOutreachCampaignUpdate(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	campaign, err := s.store.GetOutreachCampaign(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That campaign doesn't exist.")
		return
	}
	var body struct {
		store.OutreachCampaign
		Steps []store.OutreachStep `json:"steps"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if trim(body.Name, 160) != "" {
		campaign.Name = trim(body.Name, 160)
	}
	if body.Offer != "" {
		campaign.Offer = trim(body.Offer, 2000)
	}
	if body.Audience != "" {
		campaign.Audience = trim(body.Audience, 2000)
	}
	if body.Tone != "" {
		campaign.Tone = trim(body.Tone, 80)
	}
	if body.Status != "" {
		campaign.Status = body.Status
	}
	if body.DailyLimit > 0 {
		campaign.DailyLimit = body.DailyLimit
	}
	campaign.Autopilot = body.Autopilot
	if err := s.store.UpdateOutreachCampaign(r.Context(), company.ID, campaign.ID, campaign); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't update that campaign.")
		return
	}
	if body.Steps != nil {
		_ = s.store.ReplaceOutreachSteps(r.Context(), company.ID, campaign.ID, body.Steps)
	}
	campaign.Steps, _ = s.store.ListOutreachSteps(r.Context(), company.ID, campaign.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "campaign": campaign})
}

func (s *Server) handleOutreachDraft(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)
	var body struct {
		CampaignID string `json:"campaignId"`
		ProspectID string `json:"prospectId"`
		ContactID  string `json:"contactId"`
		Step       int    `json:"step"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	prospect, err := s.store.GetOutreachProspect(r.Context(), company.ID, body.ProspectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "That prospect doesn't exist.")
		return
	}
	campaign, err := s.store.GetOutreachCampaign(r.Context(), company.ID, body.CampaignID)
	if err != nil {
		writeError(w, http.StatusNotFound, "That campaign doesn't exist.")
		return
	}
	steps, _ := s.store.ListOutreachSteps(r.Context(), company.ID, campaign.ID)
	if len(steps) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "Add at least one sequence step.")
		return
	}
	step := steps[0]
	if body.Step > 0 && body.Step <= len(steps) {
		step = steps[body.Step-1]
	}
	contacts, _ := s.store.ListOutreachContacts(r.Context(), company.ID, prospect.ID)
	var contact store.OutreachContact
	for _, item := range contacts {
		if item.ID == body.ContactID || (contact.ID == "" && item.Email != "") {
			contact = item
		}
	}
	if contact.Email == "" {
		writeError(w, http.StatusUnprocessableEntity, "Add a contact email before drafting.")
		return
	}

	subject, text, modelName := s.writeOutreachDraft(r.Context(), id.UserID, company, prospect, contact, campaign, step)
	status := "draft"
	if campaign.Autopilot {
		status = "approved"
	}
	message, err := s.store.CreateOutreachMessage(r.Context(), company.ID, store.OutreachMessage{
		CampaignID: campaign.ID, ProspectID: prospect.ID, ContactID: contact.ID, Step: step.Position,
		To: contact.Email, Subject: subject, Body: text, Status: status, ScheduledAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save the draft.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "message": message, "model": modelName})
}

func (s *Server) handleOutreachMessageStatus(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	message, err := s.store.GetOutreachMessage(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That message doesn't exist.")
		return
	}
	var body struct {
		Status    string `json:"status"`
		ReplyText string `json:"replyText"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := strings.ToLower(trim(body.Status, 20))
	switch status {
	case "approved":
		if message.Status != "draft" {
			writeError(w, http.StatusUnprocessableEntity, "Only a draft can be approved.")
			return
		}
		_ = s.store.UpdateOutreachMessageStatus(r.Context(), company.ID, message.ID, "approved", "", "")
	case "replied":
		_ = s.store.UpdateOutreachMessageStatus(r.Context(), company.ID, message.ID, "replied", "", trim(body.ReplyText, 4000))
		prospect, _ := s.store.GetOutreachProspect(r.Context(), company.ID, message.ProspectID)
		prospect.Stage = "replied"
		_ = s.store.UpdateOutreachProspect(r.Context(), company.ID, prospect.ID, prospect)
	case "optout":
		_, _ = s.store.AddOutreachSuppression(r.Context(), company.ID, message.To, "opt-out")
		_ = s.store.UpdateOutreachMessageStatus(r.Context(), company.ID, message.ID, "optout", "", "")
		prospect, _ := s.store.GetOutreachProspect(r.Context(), company.ID, message.ProspectID)
		prospect.Stage = "lost"
		_ = s.store.UpdateOutreachProspect(r.Context(), company.ID, prospect.ID, prospect)
	default:
		writeError(w, http.StatusUnprocessableEntity, "Status must be approved, replied or optout.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleOutreachSend(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)
	settings, _ := s.store.GetOutreachSettings(r.Context(), company.ID)
	sent, skipped, err := s.sendApprovedOutreach(r.Context(), id.UserID, company, settings, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "sent": sent, "skipped": skipped})
}

func (s *Server) handleOutreachSendBatch(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)
	settings, _ := s.store.GetOutreachSettings(r.Context(), company.ID)
	messages, _ := s.store.ListOutreachMessages(r.Context(), company.ID)
	sent, skipped := 0, 0
	for _, message := range messages {
		if message.Status != "approved" {
			continue
		}
		count, skip, err := s.sendApprovedOutreach(r.Context(), id.UserID, company, settings, message.ID)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		sent += count
		skipped += skip
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "sent": sent, "skipped": skipped})
}

func (s *Server) sendApprovedOutreach(ctx context.Context, userID string, company *store.Company, settings store.OutreachSettings, messageID string) (int, int, error) {
	message, err := s.store.GetOutreachMessage(ctx, company.ID, messageID)
	if err != nil || message.Status != "approved" {
		return 0, 1, nil
	}
	if s.store.IsOutreachSuppressed(ctx, company.ID, message.To) {
		_ = s.store.UpdateOutreachMessageStatus(ctx, company.ID, message.ID, "optout", "", "")
		return 0, 1, nil
	}
	limit := settings.DailyLimit
	if limit <= 0 {
		limit = 20
	}
	today, _ := s.store.CountOutreachSentToday(ctx, company.ID)
	if today >= limit {
		return 0, 1, fmt.Errorf("daily sending limit reached (%d)", limit)
	}

	from := strings.TrimSpace(settings.FromEmail)
	if from == "" {
		from = s.cfg.SMTPFrom
	}
	if settings.FromName != "" && from != "" {
		from = settings.FromName + " <" + extractEmail(from) + ">"
	}
	body := message.Body
	if settings.Signature != "" {
		body += "\n\n" + settings.Signature
	}
	if err := mailer.Send(mailer.Config{Addr: s.cfg.SMTPAddr, From: from, User: s.cfg.SMTPUser, Pass: s.cfg.SMTPPass},
		message.To, message.Subject, body); err != nil {
		return 0, 0, err
	}
	_ = s.store.UpdateOutreachMessageStatus(ctx, company.ID, message.ID, "sent", time.Now().UTC().Format(time.RFC3339), "")
	prospect, _ := s.store.GetOutreachProspect(ctx, company.ID, message.ProspectID)
	prospect.Stage = "contacted"
	prospect.LastContacted = time.Now().UTC().Format(time.RFC3339)
	_ = s.store.UpdateOutreachProspect(ctx, company.ID, prospect.ID, prospect)
	_ = s.store.AddActivity(ctx, userID, company.ID, "outreach_sent", "You sent outreach to "+message.To, "", "")
	return 1, 0, nil
}

func (s *Server) handleOutreachExport(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	prospects, _ := s.store.ListOutreachProspects(r.Context(), company.ID)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="crew-outreach.csv"`)
	writer := csv.NewWriter(w)
	defer writer.Flush()
	_ = writer.Write([]string{"email", "first_name", "last_name", "company_name", "title", "website", "industry", "personalization", "email_status", "score", "source", "status", "created_at"})
	for _, prospect := range prospects {
		contacts, _ := s.store.ListOutreachContacts(r.Context(), company.ID, prospect.ID)
		if len(contacts) == 0 {
			_ = writer.Write([]string{"", "", "", prospect.Name, "", prospect.Website, prospect.Industry, prospect.Notes, "", fmt.Sprint(prospect.Score), prospect.Source, prospect.Stage, prospect.CreatedAt.Format(time.RFC3339)})
			continue
		}
		for _, contact := range contacts {
			first, last := splitName(contact.Name)
			_ = writer.Write([]string{contact.Email, first, last, prospect.Name, contact.Role, prospect.Website,
				prospect.Industry, prospect.Notes, contact.EmailStatus, fmt.Sprint(prospect.Score), prospect.Source,
				prospect.Stage, prospect.CreatedAt.Format(time.RFC3339)})
		}
	}
}

func (s *Server) handleOutreachSettings(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	if r.Method == http.MethodGet {
		settings, _ := s.store.GetOutreachSettings(r.Context(), company.ID)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "settings": settings})
		return
	}
	var settings store.OutreachSettings
	if err := decodeJSON(r, &settings); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if settings.DailyLimit <= 0 {
		settings.DailyLimit = 20
	}
	if settings.DailyLimit > 200 {
		settings.DailyLimit = 200
	}
	if err := s.store.SaveOutreachSettings(r.Context(), company.ID, settings); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save outreach settings.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "settings": settings})
}

func (s *Server) handleOutreachSuppression(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	var body struct {
		Email  string `json:"email"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !strings.Contains(body.Email, "@") {
		writeError(w, http.StatusUnprocessableEntity, "Add a valid email.")
		return
	}
	suppression, err := s.store.AddOutreachSuppression(r.Context(), company.ID, body.Email, trim(body.Reason, 200))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save that suppression.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "suppression": suppression})
}

func (s *Server) outreachData(ctx context.Context, companyID string) (outreachPayload, error) {
	prospects, err := s.store.ListOutreachProspects(ctx, companyID)
	if err != nil {
		return outreachPayload{}, err
	}
	for i := range prospects {
		prospects[i].Contacts, _ = s.store.ListOutreachContacts(ctx, companyID, prospects[i].ID)
	}
	campaigns, _ := s.store.ListOutreachCampaigns(ctx, companyID)
	for i := range campaigns {
		campaigns[i].Steps, _ = s.store.ListOutreachSteps(ctx, companyID, campaigns[i].ID)
	}
	messages, _ := s.store.ListOutreachMessages(ctx, companyID)
	suppressions, _ := s.store.ListOutreachSuppressions(ctx, companyID)
	settings, _ := s.store.GetOutreachSettings(ctx, companyID)
	stats, _ := s.store.OutreachStats(ctx, companyID)
	if prospects == nil {
		prospects = []store.OutreachProspect{}
	}
	if campaigns == nil {
		campaigns = []store.OutreachCampaign{}
	}
	if messages == nil {
		messages = []store.OutreachMessage{}
	}
	if suppressions == nil {
		suppressions = []store.OutreachSuppression{}
	}
	return outreachPayload{
		Prospects: prospects, Campaigns: campaigns, Messages: messages,
		Suppressions: suppressions, Settings: settings, Stats: stats,
		SendReady: strings.TrimSpace(s.cfg.SMTPFrom) != "", FromAddress: s.cfg.SMTPFrom,
	}, nil
}

func (s *Server) writeOutreachDraft(ctx context.Context, userID string, company *store.Company, prospect store.OutreachProspect, contact store.OutreachContact, campaign store.OutreachCampaign, step store.OutreachStep) (string, string, string) {
	fallbackSubject := step.Subject
	if fallbackSubject == "" {
		fallbackSubject = "Quick question about " + prospect.Name
	}
	fallbackMessage := "Hi " + firstName(contact.Name) + ",\n\n" + campaign.Offer
	if campaign.Offer == "" {
		fallbackMessage = "Hi " + firstName(contact.Name) + ",\n\nI noticed " + signalSummary(prospect.Signals) + " at " + prospect.Name + ". We help " + campaign.Audience + " with " + company.Name + "."
	}
	fallbackMessage += "\n\nWould it be worth a short call?\n\n" + contact.Name

	model, err := marketingModelFor(s, ctx, userID)
	if err != nil {
		return fallbackSubject, fallbackMessage, "template"
	}
	if blocked, _ := s.checkBudget(ctx, userID, company.ID, ""); blocked != "" {
		return fallbackSubject, fallbackMessage, "template"
	}

	contextJSON, _ := json.Marshal(map[string]any{
		"company": company, "prospect": prospect, "contact": contact, "campaign": campaign,
		"step": step.Position, "step_instruction": step.Body, "signals": prospect.Signals,
	})
	system := `You write one-to-one B2B cold emails. Use only supplied facts. Do not invent
results, customers, funding or urgency. Mention one concrete observed signal. Keep it under
120 words, no more than one link, and end with one simple question. Return valid JSON only:
{"subject":"","body":""}`
	prompt := "Campaign step " + fmt.Sprint(step.Position) + " of " + fmt.Sprint(len([]int{step.Position})) + ". Draft the email.\n\n" + string(contextJSON)
	result, err := llm.CompleteResult(ctx, llm.Config{Kind: model.Provider.Kind, BaseURL: model.Provider.BaseURL, Model: model.Model, APIKey: model.APIKey}, system, prompt, 700)
	if err != nil {
		return fallbackSubject, fallbackMessage, "template"
	}
	document, err := extractJSONObject(result.Text)
	if err != nil {
		return fallbackSubject, fallbackMessage, "template"
	}
	var draft struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.Unmarshal(document, &draft); err != nil || strings.TrimSpace(draft.Body) == "" {
		return fallbackSubject, fallbackMessage, "template"
	}
	_ = s.saveUsage(ctx, userID, company.ID, "", "", model, result.Usage)
	return trim(draft.Subject, 180), trim(draft.Body, 8000), model.Provider.Name + " · " + model.Model
}

func outreachSignals(text, document string) ([]store.OutreachSignal, int) {
	lowerText := strings.ToLower(text)
	lowerHTML := strings.ToLower(document)
	signals := []store.OutreachSignal{
		{Key: "no_online_booking", Label: "No online booking", Found: !containsAny(lowerText+lowerHTML, []string{"book now", "book online", "calendly", "booking", "schedule a call", "resy"}), Detail: "Leads have to phone or email."},
		{Key: "no_live_chat", Label: "No live chat", Found: !containsAny(lowerHTML, []string{"intercom", "crisp.chat", "tawk.to", "drift.com", "livechat"}), Detail: "After-hours questions have no immediate answer."},
		{Key: "no_schema", Label: "No schema markup", Found: !strings.Contains(lowerHTML, "application/ld+json"), Detail: "Search engines get less structured data."},
		{Key: "ads_running", Label: "Advertising pixel present", Found: containsAny(lowerHTML, []string{"facebook.net", "googletagmanager", "gtag(", "linkedin.com/px"}), Detail: "They already spend on acquisition."},
		{Key: "hiring", Label: "Hiring signal", Found: containsAny(lowerText, []string{"we are hiring", "careers", "open roles", "join our team"}), Detail: "A role may indicate a budget or capacity gap."},
		{Key: "blog_stale", Label: "No fresh blog", Found: !containsAny(lowerHTML, []string{"/blog", "insights", "resources"}), Detail: "Content marketing has not started or is hard to find."},
		{Key: "mobile_ready", Label: "Mobile viewport", Found: strings.Contains(lowerHTML, "name=\"viewport\""), Detail: "The site declares a mobile viewport."},
		{Key: "contact_email", Label: "Public email", Found: outreachEmailPattern.MatchString(text + document), Detail: "A public email address is available."},
	}

	score := 10
	for _, signal := range signals {
		if !signal.Found {
			continue
		}
		switch signal.Key {
		case "no_online_booking":
			score += 15
		case "no_live_chat":
			score += 10
		case "ads_running":
			score += 15
		case "hiring":
			score += 10
		case "contact_email":
			score += 20
		case "blog_stale":
			score += 5
		case "no_schema":
			score += 5
		}
	}
	if score > 100 {
		score = 100
	}
	return signals, score
}

func discoverOSM(ctx context.Context, query, location string, limit int, searchURL string) ([]store.OutreachProspect, error) {
	geocodeURL := "https://nominatim.openstreetmap.org/search?format=json&limit=1&q=" + url.QueryEscape(location)
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, geocodeURL, nil)
	request.Header.Set("User-Agent", "CrewOutreach/1.0 (https://crew.paytrackify.com)")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return webDiscovery(ctx, query, location, limit, searchURL)
	}
	defer response.Body.Close()
	var places []struct {
		BoundingBox []string `json:"boundingbox"`
		DisplayName string   `json:"display_name"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&places); err != nil || len(places) == 0 {
		return webDiscovery(ctx, query, location, limit, searchURL)
	}
	box := places[0].BoundingBox
	if len(box) < 4 {
		return nil, fmt.Errorf("location has no bounding box")
	}
	south, west, north, east := box[0], box[2], box[1], box[3]
	clean := strings.ReplaceAll(strings.ReplaceAll(query, `"`, ""), `\`, "")
	overpass := fmt.Sprintf(`[out:json][timeout:25];nwr["name"~"%s",i](%s,%s,%s,%s);out center %d;`, clean, south, west, north, east, limit)
	request, _ = http.NewRequestWithContext(ctx, http.MethodPost, "https://overpass-api.de/api/interpreter", strings.NewReader("data="+url.QueryEscape(overpass)))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "CrewOutreach/1.0")
	response, err = (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return webDiscovery(ctx, query, location, limit, searchURL)
	}
	defer response.Body.Close()
	var payload struct {
		Elements []struct {
			Tags map[string]string `json:"tags"`
		} `json:"elements"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload); err != nil {
		return webDiscovery(ctx, query, location, limit, searchURL)
	}

	out := []store.OutreachProspect{}
	for _, element := range payload.Elements {
		name := element.Tags["name"]
		if name == "" {
			continue
		}
		website := normalizeWebsite(element.Tags["website"])
		out = append(out, store.OutreachProspect{
			Name: name, Website: website, Domain: domainOf(website),
			Industry: element.Tags["shop"], Location: element.Tags["addr:city"], Source: "OpenStreetMap",
		})
	}
	if len(out) > 0 {
		return out, nil
	}
	return webDiscovery(ctx, query, location, limit, searchURL)
}

func webDiscovery(ctx context.Context, query, location string, limit int, searchURL string) ([]store.OutreachProspect, error) {
	results := research.Search(ctx, query+" in "+location, "", searchURL)
	blocked := []string{"wikipedia.org", "facebook.com", "instagram.com", "linkedin.com", "yelp.", "tripadvisor.", "yellowpages.", "reddit.com", "youtube.com", "pinterest."}
	out := []store.OutreachProspect{}
	seen := map[string]bool{}
	for _, result := range results {
		if len(out) >= limit {
			break
		}
		parsed, err := url.Parse(result.URL)
		if err != nil || parsed.Hostname() == "" {
			continue
		}
		domain := strings.ToLower(parsed.Hostname())
		skip := false
		for _, item := range blocked {
			if strings.Contains(domain, item) {
				skip = true
				break
			}
		}
		if skip || seen[domain] {
			continue
		}
		seen[domain] = true
		name := strings.TrimSpace(result.Title)
		if name == "" {
			name = domain
		}
		out = append(out, store.OutreachProspect{
			Name: name, Website: "https://" + domain, Domain: domain,
			Location: location, Source: "web search",
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no businesses found; OpenStreetMap may be unavailable and web search returned no usable sites")
	}
	return out, nil
}

func verifyEmailStatus(ctx context.Context, email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	if !outreachEmailPattern.MatchString(email) {
		return "invalid"
	}
	domain := domainOf("@" + email)
	if domain == "" {
		return "invalid"
	}
	mx, err := net.LookupMX(domain)
	if err != nil || len(mx) == 0 {
		return "invalid"
	}
	if roleEmailPattern.MatchString(email) {
		return "risky"
	}
	return "mx_valid"
}

func extractEmails(text string) []string {
	raw := outreachEmailPattern.FindAllString(text, 20)
	seen := map[string]bool{}
	out := []string{}
	for _, email := range raw {
		email = strings.ToLower(strings.Trim(email, ".,;:()[]<>"))
		if strings.HasSuffix(email, ".png") || strings.HasSuffix(email, ".jpg") || seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, email)
	}
	return out
}

func contactName(email string) string {
	local := email
	if at := strings.Index(local, "@"); at >= 0 {
		local = local[:at]
	}
	local = strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(local)
	parts := strings.Fields(local)
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, " ")
}

func firstName(name string) string {
	if fields := strings.Fields(name); len(fields) > 0 {
		return fields[0]
	}
	return "there"
}

func splitName(name string) (string, string) {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return "", ""
	}
	if len(fields) == 1 {
		return fields[0], ""
	}
	return fields[0], strings.Join(fields[1:], " ")
}

func signalSummary(signals []store.OutreachSignal) string {
	for _, signal := range signals {
		if signal.Found && signal.Key == "no_online_booking" {
			return "there is no online booking route"
		}
	}
	return "the site is missing an obvious conversion path"
}

func normalizeWebsite(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "https://" + value
	}
	return value
}

func domainOf(value string) string {
	if strings.Contains(value, "@") {
		parts := strings.Split(value, "@")
		return strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func extractEmail(value string) string {
	if start := strings.Index(value, "<"); start >= 0 {
		if end := strings.Index(value[start:], ">"); end > 0 {
			return value[start+1 : start+end]
		}
	}
	return value
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func parseScore(value string) (int, error) {
	var score int
	_, err := fmt.Sscanf(value, "%d", &score)
	return score, err
}
