package httpapi

import (
	"net/http"
	"strings"

	"github.com/davepartner/crew/internal/advise"
	"github.com/davepartner/crew/internal/finance"
	"github.com/davepartner/crew/internal/store"
)

type portfolioRow struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	IsProduct  bool    `json:"is_product"`
	Country    string  `json:"country"`
	Currency   string  `json:"currency"`
	Stage      string  `json:"stage"`
	Score      int     `json:"score"`
	Revenue    float64 `json:"revenue"`
	NetProfit  float64 `json:"net_profit"`
	Cash       float64 `json:"cash"`
	Tasks      int     `json:"tasks"`
	Agents     int     `json:"agents"`
	AuditState string  `json:"audit_state"`
}

type portfolioTotal struct {
	Currency  string  `json:"currency"`
	Revenue   float64 `json:"revenue"`
	NetProfit float64 `json:"net_profit"`
	Cash      float64 `json:"cash"`
}

func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	parentID := company.ID
	if company.IsProduct {
		parentID = company.ParentID
	}
	parent, err := s.store.GetCompany(r.Context(), id.UserID, parentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find the parent company.")
		return
	}

	products, _ := s.store.ListProducts(r.Context(), parent.ID)
	all := append([]store.Company{parent}, products...)

	rows := make([]portfolioRow, 0, len(all))
	totals := map[string]*portfolioTotal{}

	for _, item := range all {
		agents, _ := s.store.ListAgents(r.Context(), item.ID)
		tasks, _ := s.store.ListTasks(r.Context(), item.ID, "")
		routines, _ := s.store.ListRoutines(r.Context(), item.ID)
		goals, _ := s.store.ListGoals(r.Context(), item.ID)
		members, _ := s.store.ListMembers(r.Context(), item.ID)
		campaigns, _ := s.store.ListAdCampaigns(r.Context(), item.ID, 20)

		report := advise.Build(advise.Input{
			Company: item, Agents: agents, Tasks: tasks, Routines: routines,
			Goals: goals, Members: members, Campaigns: campaigns,
		})

		row := portfolioRow{
			ID: item.ID, Name: item.Name, IsProduct: item.IsProduct, Country: item.Country,
			Currency: report.Country.Currency, Stage: report.Stage, Score: report.Score,
			Tasks: len(tasks), Agents: len(agents), AuditState: "none",
		}

		if snapshot, err := s.store.LatestFinancialSnapshot(r.Context(), item.ID); err == nil {
			model := finance.Build(snapshot)
			row.Revenue = model.Revenue
			row.NetProfit = model.NetProfit
			row.Cash = model.Cash
			row.AuditState = snapshot.AuditStatus
		}

		rows = append(rows, row)

		key := row.Currency
		if key == "" {
			key = "unknown"
		}
		if totals[key] == nil {
			totals[key] = &portfolioTotal{Currency: row.Currency}
		}
		totals[key].Revenue += row.Revenue
		totals[key].NetProfit += row.NetProfit
		totals[key].Cash += row.Cash
	}

	totalList := make([]portfolioTotal, 0, len(totals))
	for _, total := range totals {
		totalList = append(totalList, *total)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "parent": parent, "products": products,
		"rows": rows, "totals": totalList,
	})
}

func currencyKey(value string) string {
	if index := strings.Index(value, " "); index > 0 {
		return value[:index]
	}
	return value
}
