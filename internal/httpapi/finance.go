package httpapi

import (
	"errors"
	"net/http"
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
