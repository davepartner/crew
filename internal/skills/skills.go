// Package skills embeds the marketing skill library and the reverse-engineering
// analyst harness. Both are MIT-licensed; see THIRD_PARTY_NOTICES.md.
package skills

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed catalog.json
var catalogJSON []byte

//go:embed data/*.md
var dataFS embed.FS

//go:embed reverse/system.md reverse/guardrails.md reverse/report-template.md reverse/lessons.md reverse/manifest.json reverse/playbooks/*.md
var reverseFS embed.FS

type Skill struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Chars       int    `json:"chars"`
	Source      string `json:"source"`
	SourceURL   string `json:"source_url"`
}

type Playbook struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	File    string `json:"file"`
}

const (
	SourceName = "Corey Haines' marketingskills"
	SourceURL  = "https://github.com/coreyhaines31/marketingskills"
	RemName    = "Reverse Engineer Marketing by DansUGC"
	RemURL     = "https://github.com/dansugc/reverse-engineer-marketing"
)

var (
	catalog   []Skill
	byKey     map[string]Skill
	playbooks []Playbook
)

func init() {
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		panic("skills: invalid catalog: " + err.Error())
	}

	byKey = make(map[string]Skill, len(catalog))
	for _, skill := range catalog {
		byKey[skill.Key] = skill
	}

	var manifest struct {
		Version   int        `json:"version"`
		Playbooks []Playbook `json:"playbooks"`
	}
	raw, err := reverseFS.ReadFile("reverse/manifest.json")
	if err != nil {
		panic("skills: missing reverse manifest")
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		panic("skills: invalid reverse manifest")
	}
	playbooks = manifest.Playbooks
}

func All() []Skill {
	out := make([]Skill, len(catalog))
	copy(out, catalog)
	return out
}

func ByKey(key string) (Skill, bool) {
	skill, ok := byKey[key]
	return skill, ok
}

func Categories() []string {
	seen := map[string]bool{}
	var out []string
	for _, skill := range catalog {
		if !seen[skill.Category] {
			seen[skill.Category] = true
			out = append(out, skill.Category)
		}
	}
	sort.Strings(out)
	return out
}

func Body(key string) (string, bool) {
	if _, ok := byKey[key]; !ok {
		return "", false
	}

	raw, err := dataFS.ReadFile("data/" + key + ".md")
	if err != nil {
		return "", false
	}
	return string(raw), true
}

// Prompt renders the selected skills for a helper's system prompt. It keeps
// the total bounded so a task cannot blow the model's context by accident.
func Prompt(keys []string, maxChars int) string {
	if len(keys) == 0 {
		return ""
	}
	if maxChars <= 0 {
		maxChars = 24000
	}

	var builder strings.Builder
	seen := map[string]bool{}
	total := 0
	included := 0

	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true

		body, ok := Body(key)
		if !ok {
			continue
		}

		block := "\n\n### " + byKey[key].Name + "\n\n" + strings.TrimSpace(body)
		header := ""
		if included == 0 {
			header = "\n\n## Marketing skills you can use\n"
		}

		if total+len(header)+len(block) > maxChars {
			note := "\n\n[skill truncated]"
			remaining := maxChars - total - len(header) - len(note)
			if remaining > 800 {
				if header != "" {
					builder.WriteString(header)
					total += len(header)
				}
				builder.WriteString(block[:remaining] + note)
				total = maxChars
				included++
			}
			break
		}

		builder.WriteString(header)
		builder.WriteString(block)
		total += len(header) + len(block)
		included++
	}

	return builder.String()
}

func Playbooks() []Playbook {
	out := make([]Playbook, len(playbooks))
	copy(out, playbooks)
	return out
}

func PlaybookBody(key string) (string, bool) {
	for _, playbook := range playbooks {
		if playbook.Key == key {
			raw, err := reverseFS.ReadFile("reverse/" + playbook.File)
			if err != nil {
				return "", false
			}
			return string(raw), true
		}
	}
	return "", false
}

// Harness assembles the analyst instructions for one playbook.
func Harness(playbookKey string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 32000
	}

	parts := []string{
		"# Reverse-engineer the marketing\n",
		"Harness version 3. Source: " + RemURL + "\n",
	}

	for _, name := range []string{"system.md", "guardrails.md", "report-template.md", "lessons.md"} {
		raw, err := reverseFS.ReadFile("reverse/" + name)
		if err != nil {
			continue
		}
		parts = append(parts, "\n---\n\n"+strings.TrimSpace(string(raw))+"\n")
	}

	if playbookKey != "" {
		if body, ok := PlaybookBody(playbookKey); ok {
			parts = append(parts, "\n---\n\n"+strings.TrimSpace(body)+"\n")
		}
	}

	joined := strings.Join(parts, "")
	if len(joined) > maxChars {
		joined = joined[:maxChars] + "\n\n[harness truncated]"
	}
	return joined
}

func Count() int { return len(catalog) }

func Describe(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if skill, ok := byKey[key]; ok {
			out = append(out, skill.Name)
		}
	}
	return out
}

func Validate(keys []string) error {
	if len(keys) > 8 {
		return fmt.Errorf("choose at most 8 skills")
	}
	for _, key := range keys {
		if _, ok := byKey[key]; !ok {
			return fmt.Errorf("unknown skill %q", key)
		}
	}
	return nil
}
