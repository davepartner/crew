package presets

import "testing"

func TestCatalogueIsComplete(t *testing.T) {
	all := All()
	t.Logf("catalogue: %d businesses, %d categories", len(all), len(Categories()))
	if len(all) < 60 {
		t.Fatalf("expected a broad catalogue, got %d businesses", len(all))
	}

	seen := map[string]bool{}
	for _, business := range all {
		if seen[business.Key] {
			t.Fatalf("duplicate business key %q", business.Key)
		}
		seen[business.Key] = true

		if len(business.Agents) < 3 {
			t.Errorf("%s has %d agents", business.Key, len(business.Agents))
		}
		if len(business.Tasks) < 3 {
			t.Errorf("%s has %d tasks", business.Key, len(business.Tasks))
		}

		names := map[string]bool{}
		for _, agent := range business.Agents {
			names[agent.Name] = true
			if agent.Instructions == "" {
				t.Errorf("%s agent %q has no instructions", business.Key, agent.Name)
			}
		}

		for _, task := range business.Tasks {
			if !names[task.Agent] {
				t.Errorf("%s task %q points at unknown agent %q", business.Key, task.Title, task.Agent)
			}
		}
	}

	if len(Categories()) < 8 {
		t.Fatalf("expected several categories, got %d", len(Categories()))
	}
}

func TestCustomTeamRenders(t *testing.T) {
	team := CustomTeam()
	if len(team.Agents) != 3 || len(team.Tasks) != 3 {
		t.Fatalf("custom team is incomplete: %d/%d", len(team.Agents), len(team.Tasks))
	}

	rendered := Render(team.Agents[0].Instructions, Vars{Company: "Acme", Audience: "shops", Tone: "Direct"})
	if rendered == "" || contains(rendered, "{{") {
		t.Fatalf("placeholders left in %q", rendered)
	}
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
