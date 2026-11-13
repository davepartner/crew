package finance

import (
	"testing"

	"github.com/davepartner/crew/internal/store"
)

func TestBuildProfitAndLoss(t *testing.T) {
	model := Build(store.FinancialSnapshot{
		Period: "October", Revenue: 1000000, COGS: 400000, Opex: 300000,
		Cash: 600000, Source: "Audited statements", Audited: true,
	})
	if model.GrossProfit != 600000 || model.NetProfit != 300000 {
		t.Fatalf("profit: %+v", model)
	}
	if model.GrossMargin != 0.6 || model.BreakEvenSales != 500000 {
		t.Fatalf("margin: %+v", model)
	}
	if model.RunwayMonths != 0 || model.Quality != "Audited" {
		t.Fatalf("quality: %+v", model)
	}
}

func TestBuildRunway(t *testing.T) {
	model := Build(store.FinancialSnapshot{
		Period: "October", Revenue: 500000, COGS: 300000, Opex: 400000, Cash: 600000,
	})
	if model.NetProfit != -200000 {
		t.Fatalf("net: %+v", model)
	}
	if model.RunwayMonths != 3 {
		t.Fatalf("runway: %+v", model)
	}
	if model.BreakEvenSales != 1000000 {
		t.Fatalf("break even: %+v", model)
	}
}
