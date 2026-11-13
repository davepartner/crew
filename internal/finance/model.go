package finance

import "github.com/davepartner/crew/internal/store"

type Model struct {
	Period         string  `json:"period"`
	Revenue        float64 `json:"revenue"`
	COGS           float64 `json:"cogs"`
	GrossProfit    float64 `json:"gross_profit"`
	GrossMargin    float64 `json:"gross_margin"`
	Opex           float64 `json:"opex"`
	NetProfit      float64 `json:"net_profit"`
	Cash           float64 `json:"cash"`
	Receivables    float64 `json:"receivables"`
	Payables       float64 `json:"payables"`
	RunwayMonths   float64 `json:"runway_months"`
	BreakEvenSales float64 `json:"break_even_sales"`
	AnnualRevenue  float64 `json:"annual_revenue"`
	Quality        string  `json:"quality"`
	Source         string  `json:"source"`
}

func Build(snapshot store.FinancialSnapshot) Model {
	model := Model{
		Period: snapshot.Period, Revenue: snapshot.Revenue, COGS: snapshot.COGS,
		Opex: snapshot.Opex, Cash: snapshot.Cash, Receivables: snapshot.Receivables,
		Payables: snapshot.Payables, Source: snapshot.Source,
	}
	model.GrossProfit = snapshot.Revenue - snapshot.COGS
	if snapshot.Revenue > 0 {
		model.GrossMargin = model.GrossProfit / snapshot.Revenue
	}
	model.NetProfit = model.GrossProfit - snapshot.Opex
	model.AnnualRevenue = snapshot.Revenue * 12

	if model.NetProfit < 0 && snapshot.Cash > 0 {
		model.RunwayMonths = snapshot.Cash / -model.NetProfit
	}
	if model.GrossMargin > 0 {
		model.BreakEvenSales = snapshot.Opex / model.GrossMargin
	}

	if snapshot.Audited {
		model.Quality = "Audited"
	} else if snapshot.Source != "" {
		model.Quality = "Management accounts"
	} else {
		model.Quality = "Estimate"
	}

	return model
}
