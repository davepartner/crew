package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/davepartner/crew/internal/finance"
	"github.com/davepartner/crew/internal/store"
)

func (s *Server) handleFinancials(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	snapshots, err := s.store.ListFinancialSnapshots(r.Context(), company.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't load the financials.")
		return
	}

	var model *finance.Model
	if len(snapshots) > 0 {
		value := finance.Build(snapshots[0])
		model = &value
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "snapshots": snapshots, "model": model})
}

func (s *Server) handleCreateFinancials(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	var body struct {
		Period      string  `json:"period"`
		Revenue     float64 `json:"revenue"`
		COGS        float64 `json:"cogs"`
		Opex        float64 `json:"opex"`
		Cash        float64 `json:"cash"`
		Receivables float64 `json:"receivables"`
		Payables    float64 `json:"payables"`
		Source      string  `json:"source"`
		Audited     bool    `json:"audited"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if body.Period == "" {
		body.Period = "This month"
	}
	if body.Revenue < 0 || body.COGS < 0 || body.Opex < 0 || body.Cash < 0 {
		writeError(w, http.StatusUnprocessableEntity, "Amounts can't be negative.")
		return
	}

	snapshot, err := s.store.CreateFinancialSnapshot(r.Context(), company.ID, store.FinancialSnapshot{
		Period: trim(body.Period, 80), Revenue: body.Revenue, COGS: body.COGS, Opex: body.Opex,
		Cash: body.Cash, Receivables: body.Receivables, Payables: body.Payables,
		Source: trim(body.Source, 120), Audited: body.Audited,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save those numbers.")
		return
	}

	_ = s.store.DeleteCompanyAdvice(r.Context(), company.ID)
	value := finance.Build(snapshot)
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "snapshot": snapshot, "model": value})
}

func (s *Server) handleDeleteFinancials(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	if err := s.store.DeleteFinancialSnapshot(r.Context(), company.ID, r.PathValue("id")); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "That snapshot doesn't exist.")
			return
		}
		writeError(w, http.StatusInternalServerError, "We couldn't delete that snapshot.")
		return
	}

	_ = s.store.DeleteCompanyAdvice(r.Context(), company.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func financialContext(snapshots []store.FinancialSnapshot) any {
	if len(snapshots) == 0 {
		return nil
	}
	return finance.Build(snapshots[0])
}

func latestFinancials(snapshots []store.FinancialSnapshot) *finance.Model {
	if len(snapshots) == 0 {
		return nil
	}
	value := finance.Build(snapshots[0])
	return &value
}

var _ = strings.TrimSpace

func (s *Server) handleAuditFinancials(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)

	var body struct {
		FileID  string `json:"fileId"`
		Auditor string `json:"auditor"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.FileID == "" {
		writeError(w, http.StatusUnprocessableEntity, "Choose the audited statement to compare against.")
		return
	}

	snapshot, err := s.store.GetFinancialSnapshot(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That financial snapshot doesn't exist.")
		return
	}
	file, err := s.store.GetCompanyFile(r.Context(), company.ID, body.FileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "That file doesn't exist.")
		return
	}
	if len(strings.TrimSpace(file.Text)) < 20 {
		writeError(w, http.StatusUnprocessableEntity, "We couldn't read text from that file. Upload a text-based PDF or image.")
		return
	}

	status, note := auditNumbers(snapshot, file.Text)
	if err := s.store.SaveFinancialAudit(r.Context(), company.ID, snapshot.ID, file.ID, status, note, trim(body.Auditor, 160)); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't save the audit result.")
		return
	}
	snapshot.AuditStatus, snapshot.AuditNote, snapshot.Auditor, snapshot.AuditFileID = status, note, trim(body.Auditor, 160), file.ID
	_ = s.store.DeleteCompanyAdvice(r.Context(), company.ID)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "snapshot": snapshot, "status": status, "note": note})
}

func auditNumbers(snapshot store.FinancialSnapshot, text string) (string, string) {
	checks := []struct {
		label string
		value float64
	}{
		{"revenue", snapshot.Revenue},
		{"cost of sales", snapshot.COGS},
		{"operating costs", snapshot.Opex},
	}

	matched := 0
	missing := make([]string, 0, 3)
	for _, check := range checks {
		if check.value <= 0 {
			continue
		}
		if containsAmount(text, check.value) {
			matched++
		} else {
			missing = append(missing, check.label)
		}
	}

	total := 0
	for _, check := range checks {
		if check.value > 0 {
			total++
		}
	}

	switch {
	case matched == total && total > 0:
		return "verified", "All declared totals match the attached statement."
	case matched > 0:
		return "discrepancies", "Some totals were not found in the statement: " + strings.Join(missing, ", ") + "."
	default:
		return "evidence_attached", "The statement is attached, but the declared totals were not found as written."
	}
}

func containsAmount(text string, value float64) bool {
	normalized := strings.ToLower(text)
	normalized = strings.NewReplacer(",", "", " ", "", "\u00a0", "", "₦", "", "$", "", "ngn", "", "usd", "").Replace(normalized)
	formats := []string{
		strconv.FormatFloat(value, 'f', 0, 64),
		strconv.FormatFloat(value, 'f', 2, 64),
		strconv.FormatFloat(value/1000, 'f', 1, 64) + "k",
	}
	for _, candidate := range formats {
		if strings.Contains(normalized, candidate) {
			return true
		}
	}
	return false
}
