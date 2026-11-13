package advise

import (
	"testing"

	"github.com/davepartner/crew/internal/store"
)

func TestBuildMarksFoundationsAndStage(t *testing.T) {
	report := Build(Input{
		Company: store.Company{
			Name: "Northstar", Country: "Nigeria", BusinessKey: "custom",
			Description: "We help clinics keep clean books every month.",
			Audience:    "clinic owners", Tone: "Direct",
		},
		Agents: []store.Agent{{
			Name: "Ada", Role: "Marketing", Skills: []string{"copywriting"}, CanDelegate: true,
		}},
		Tasks:    []store.Task{{Title: "Write the offer", Status: store.StatusDone}},
		Routines: []store.Routine{{Title: "Weekly review"}},
		Members:  []store.Member{{Name: "Jane"}, {Name: "Editor"}},
	})

	if report.CompanyName != "Northstar" || report.Country.Name != "Nigeria" {
		t.Fatalf("report: %+v", report)
	}
	if report.Score <= 0 || report.Score > 100 {
		t.Fatalf("score: %d", report.Score)
	}
	if report.Stage != "early-traction" && report.Stage != "growth" {
		t.Fatalf("stage: %s", report.Stage)
	}
	if len(report.Sections) < 8 {
		t.Fatalf("sections: %d", len(report.Sections))
	}
	if len(report.Next) == 0 {
		t.Fatal("next actions missing")
	}

	foundation := report.Sections[0]
	if foundation.Status == "todo" {
		t.Fatalf("foundation should have progress: %+v", foundation)
	}
}

func TestCountryFallback(t *testing.T) {
	profile := Country("Atlantis")
	if profile.Name != "Atlantis" || len(profile.Registration) == 0 {
		t.Fatalf("fallback: %+v", profile)
	}
	if Country("nigeria").Currency == "" {
		t.Fatal("nigeria profile missing")
	}
}
