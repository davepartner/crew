package skills

import (
	"strings"
	"testing"
)

func TestCatalogue(t *testing.T) {
	if Count() != 53 {
		t.Fatalf("expected 53 skills, got %d", Count())
	}

	seen := map[string]bool{}
	for _, skill := range All() {
		if seen[skill.Key] {
			t.Fatalf("duplicate %s", skill.Key)
		}
		seen[skill.Key] = true
		if skill.Name == "" || skill.Description == "" || skill.Category == "" {
			t.Errorf("incomplete skill %+v", skill)
		}
		if _, ok := Body(skill.Key); !ok {
			t.Errorf("missing body for %s", skill.Key)
		}
	}
}

func TestPromptStaysBounded(t *testing.T) {
	prompt := Prompt([]string{"copywriting", "content-strategy", "pricing"}, 4000)
	if len(prompt) > 4000 {
		t.Fatalf("prompt is %d chars", len(prompt))
	}
	if !strings.Contains(prompt, "Copywriting") {
		t.Fatal("selected skill missing")
	}

	if Prompt(nil, 0) != "" {
		t.Fatal("empty selection should render nothing")
	}
	if Prompt([]string{"not-a-skill"}, 0) != "" {
		t.Fatal("unknown skill should render nothing")
	}
}

func TestHarnessIncludesPlaybook(t *testing.T) {
	harness := Harness("paid-meta", 0)
	if !strings.Contains(harness, "growth analyst") {
		t.Fatal("system prompt missing")
	}
	if !strings.Contains(harness, "Paid Meta") {
		t.Fatal("playbook missing")
	}
	if len(Playbooks()) != 10 {
		t.Fatalf("expected 10 playbooks, got %d", len(Playbooks()))
	}
}

func TestValidate(t *testing.T) {
	if err := Validate([]string{"copywriting", "emails"}); err != nil {
		t.Fatal(err)
	}
	if err := Validate([]string{"nope"}); err == nil {
		t.Fatal("expected unknown skill error")
	}
}
