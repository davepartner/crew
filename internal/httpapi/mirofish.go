package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/llm"
	"github.com/davepartner/crew/internal/modelcatalog"
	"github.com/davepartner/crew/internal/store"
)

type mirofishEstimate struct {
	Tokens        int     `json:"tokens"`
	PromptTokens  int     `json:"prompt_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	Seconds       int     `json:"seconds"`
	SecondsMin    int     `json:"seconds_min"`
	SecondsMax    int     `json:"seconds_max"`
	CostUSD       float64 `json:"cost_usd"`
	Calls         int     `json:"calls"`
	Model         string  `json:"model"`
	Agents        int     `json:"agents"`
	Rounds        int     `json:"rounds"`
	EstimateLabel string  `json:"estimate_label"`
}

type mirofishPage struct {
	Company     *store.Company      `json:"company"`
	Products    []store.Company     `json:"products"`
	Suggestions []string            `json:"suggestions"`
	Runs        []store.MiroFishRun `json:"runs"`
	ModelReady  bool                `json:"model_ready"`
	Model       string              `json:"model"`
}

func (s *Server) handleMiroFish(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	products, _ := s.store.ListProducts(r.Context(), company.ID)
	runs, _ := s.store.ListMiroFishRuns(r.Context(), company.ID, 10)

	modelReady := false
	modelLabel := ""
	if model, err := marketingModelFor(s, r.Context(), identityFrom(r).UserID); err == nil {
		modelReady = true
		modelLabel = model.Provider.Name + " · " + model.Model
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"mirofish": mirofishPage{
			Company: company, Products: products, Suggestions: mirofishSuggestions(company, products),
			Runs: runs, ModelReady: modelReady, Model: modelLabel,
		},
	})
}

func (s *Server) handleMiroFishEstimate(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)
	var body struct {
		Question string `json:"question"`
		Agents   int    `json:"agents"`
		Rounds   int    `json:"rounds"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	question := trim(body.Question, 1200)
	if question == "" {
		writeError(w, http.StatusUnprocessableEntity, "Ask a question first.")
		return
	}
	agents, rounds := clampSimulation(body.Agents, body.Rounds)

	model, err := marketingModelFor(s, r.Context(), id.UserID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"ok": false, "code": "no_model",
			"error": "Add a model key in Settings → AI models first.",
		})
		return
	}
	products, _ := s.store.ListProducts(r.Context(), company.ID)
	estimate := estimateMiroFish(question, agents, rounds, model, company, products)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "estimate": estimate})
}

func (s *Server) handleMiroFishRun(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	var body struct {
		Question string `json:"question"`
		Product  string `json:"product"`
		Agents   int    `json:"agents"`
		Rounds   int    `json:"rounds"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	question := trim(body.Question, 1200)
	if question == "" {
		writeError(w, http.StatusUnprocessableEntity, "Ask a question first.")
		return
	}
	agents, rounds := clampSimulation(body.Agents, body.Rounds)

	model, err := marketingModelFor(s, r.Context(), id.UserID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "code": "no_model", "error": "Add a model key first."})
		return
	}
	if blocked, err := s.checkBudget(r.Context(), id.UserID, company.ID, ""); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't check the AI budget.")
		return
	} else if blocked != "" {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"ok": false, "code": "budget", "error": blocked})
		return
	}

	products, _ := s.store.ListProducts(r.Context(), company.ID)
	estimate := estimateMiroFish(question, agents, rounds, model, company, products)
	run, err := s.store.CreateMiroFishRun(r.Context(), company.ID, id.UserID, store.MiroFishRun{
		Question: question, Product: trim(body.Product, 120), Agents: agents, Rounds: rounds,
		Status: "queued", EstimateTokens: estimate.Tokens,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't start the simulation.")
		return
	}

	companyCopy := *company
	go s.executeMiroFish(run.ID, id.UserID, companyCopy, question, agents, rounds, model)

	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "run": run, "estimate": estimate})
}

func (s *Server) handleMiroFishRunStatus(w http.ResponseWriter, r *http.Request) {
	company := companyFrom(r)
	run, err := s.store.GetMiroFishRun(r.Context(), company.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "That simulation doesn't exist.")
		return
	}
	if run.Result != "" {
		var result any
		if json.Unmarshal([]byte(run.Result), &result) == nil {
			run.ResultJSON = result
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "run": run})
}

func (s *Server) executeMiroFish(runID, userID string, company store.Company, question string, agents, rounds int, model resolvedModel) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	run, err := s.store.GetMiroFishRun(ctx, company.ID, runID)
	if err != nil {
		return
	}
	run.Status = "running"
	_ = s.store.UpdateMiroFishRun(ctx, runID, run)

	fail := func(message string) {
		run.Status = "failed"
		run.Error = message
		run.ElapsedMS = time.Since(started).Milliseconds()
		run.CompletedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.store.UpdateMiroFishRun(context.Background(), runID, run)
	}

	products, _ := s.store.ListProducts(ctx, company.ID)
	contextJSON, _ := json.Marshal(map[string]any{
		"company": company.Name, "description": company.Description, "audience": company.Audience,
		"country": company.Country, "website": company.Website, "products": productNames(&company, products),
	})

	personaSystem := `You are the casting director for a business simulation. Create a diverse set of
customer, competitor, partner and operator personas relevant to the question. Use only the supplied
company facts. Return valid JSON only:
{"personas":[{"name":"","role":"","stake":"","bias":"","knowledge":""}]}`
	personaPrompt := fmt.Sprintf("Number of personas: %d\nQuestion: %s\nCompany context: %s", agents, question, contextJSON)
	personaResult, err := llm.CompleteResult(ctx, llm.Config{Kind: model.Provider.Kind, BaseURL: model.Provider.BaseURL, Model: model.Model, APIKey: model.APIKey}, personaSystem, personaPrompt, 2200)
	if err != nil {
		fail("The persona generation failed: " + err.Error())
		return
	}
	inputTokens, outputTokens := personaResult.Usage.PromptTokens, personaResult.Usage.CompletionTokens
	_ = s.saveUsage(ctx, userID, company.ID, "", "", model, personaResult.Usage)

	document, err := extractJSONObject(personaResult.Text)
	if err != nil {
		fail("The model returned an unreadable persona set.")
		return
	}
	var personaSet struct {
		Personas []map[string]any `json:"personas"`
	}
	if err := json.Unmarshal(document, &personaSet); err != nil || len(personaSet.Personas) == 0 {
		fail("The model returned no personas.")
		return
	}
	if len(personaSet.Personas) > agents {
		personaSet.Personas = personaSet.Personas[:agents]
	}

	previous := "No previous round."
	allReactions := []map[string]any{}
	batchSize := 8
	for round := 1; round <= rounds; round++ {
		for start := 0; start < len(personaSet.Personas); start += batchSize {
			end := start + batchSize
			if end > len(personaSet.Personas) {
				end = len(personaSet.Personas)
			}
			batch := personaSet.Personas[start:end]
			batchJSON, _ := json.Marshal(batch)
			roundSystem := `You are running one round of a multi-agent business simulation. Each persona reacts
from its own incentives. Do not invent facts. Return valid JSON only:
{"reactions":[{"persona":"","stance":"supportive|opposed|mixed|uncertain","reaction":"","confidence":0}]}
Keep each reaction under 45 words.`
			prompt := fmt.Sprintf("Round %d of %d\nQuestion: %s\nCompany: %s\nPrevious round: %s\nPersonas: %s",
				round, rounds, question, contextJSON, previous, batchJSON)
			result, err := llm.CompleteResult(ctx, llm.Config{Kind: model.Provider.Kind, BaseURL: model.Provider.BaseURL, Model: model.Model, APIKey: model.APIKey}, roundSystem, prompt, 1600)
			if err != nil {
				fail(fmt.Sprintf("Round %d failed: %s", round, err.Error()))
				return
			}
			inputTokens += result.Usage.PromptTokens
			outputTokens += result.Usage.CompletionTokens
			_ = s.saveUsage(ctx, userID, company.ID, "", "", model, result.Usage)

			roundDoc, err := extractJSONObject(result.Text)
			if err != nil {
				continue
			}
			var roundPayload struct {
				Reactions []map[string]any `json:"reactions"`
			}
			if json.Unmarshal(roundDoc, &roundPayload) == nil {
				allReactions = append(allReactions, roundPayload.Reactions...)
				previous = summarizeReactions(roundPayload.Reactions)
			}
		}
	}

	reactionJSON, _ := json.Marshal(allReactions)
	synthesisSystem := `You are the report agent for a business simulation. Use only the supplied simulation
and company facts. Separate prediction from certainty. Return valid JSON only:
{"headline":"","prediction":"","confidence":0,
"scenarios":[{"name":"","probability":0,"description":""}],
"agent_reactions":[{"persona":"","stance":"","quote":""}],
"second_order_effects":[""],"recommended_actions":[""],"dissent":[""],"unknowns":[""]}
Confidence and probabilities are 0-100. Give 2-3 scenarios and at most 5 reactions.`
	synthesisPrompt := fmt.Sprintf("Question: %s\nCompany: %s\nPersonas: %s\nReactions: %s",
		question, contextJSON, string(miroFishJSON(personaSet.Personas)), string(reactionJSON))
	synthesis, err := llm.CompleteResult(ctx, llm.Config{Kind: model.Provider.Kind, BaseURL: model.Provider.BaseURL, Model: model.Model, APIKey: model.APIKey}, synthesisSystem, synthesisPrompt, 2200)
	if err != nil {
		fail("The report generation failed: " + err.Error())
		return
	}
	inputTokens += synthesis.Usage.PromptTokens
	outputTokens += synthesis.Usage.CompletionTokens
	_ = s.saveUsage(ctx, userID, company.ID, "", "", model, synthesis.Usage)

	report, err := extractJSONObject(synthesis.Text)
	if err != nil {
		fail("The model returned an unreadable report.")
		return
	}

	run.Status = "done"
	run.InputTokens = inputTokens
	run.OutputTokens = outputTokens
	run.ElapsedMS = time.Since(started).Milliseconds()
	run.Result = string(report)
	run.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	_ = s.store.UpdateMiroFishRun(context.Background(), runID, run)
}

func estimateMiroFish(question string, agents, rounds int, model resolvedModel, company *store.Company, products []store.Company) mirofishEstimate {
	contextChars := len(company.Name) + len(company.Description) + len(company.Audience) + 300
	for _, product := range products {
		contextChars += len(product.Name) + len(product.Description) + 40
	}
	base := int(math.Ceil(float64(contextChars+len(question)) / 3.6))
	batches := int(math.Ceil(float64(agents) / 8.0))
	calls := 1 + batches*rounds + 1
	promptTokens := base + 500 // persona call
	promptTokens += batches * rounds * (base/2 + 350)
	promptTokens += base + agents*90 + 900 // synthesis
	outputTokens := agents*90 + batches*rounds*650 + 1800
	if outputTokens < 1200 {
		outputTokens = 1200
	}
	total := promptTokens + outputTokens
	seconds := int(math.Ceil(float64(calls)*1.5 + float64(total)/320.0))
	if seconds < 10 {
		seconds = 10
	}
	cost := modelcatalog.EstimateCost(model.Provider.Key, model.Model, promptTokens, outputTokens)
	return mirofishEstimate{
		Tokens: total, PromptTokens: promptTokens, OutputTokens: outputTokens, Calls: calls,
		Seconds: seconds, SecondsMin: int(math.Round(float64(seconds) * 0.7)), SecondsMax: int(math.Round(float64(seconds) * 1.6)),
		CostUSD: cost, Model: model.Provider.Name + " · " + model.Model, Agents: agents, Rounds: rounds,
		EstimateLabel: fmt.Sprintf("about %s tokens and %ds–%ds", formatInt(total), int(math.Round(float64(seconds)*0.7)), int(math.Round(float64(seconds)*1.6))),
	}
}

func mirofishSuggestions(company *store.Company, products []store.Company) []string {
	product := company.Name
	if len(products) > 0 {
		product = products[0].Name
	}
	suggestions := []string{
		"What will customers say if we raise the price of " + product + " by 15%?",
		"Which customer segment is most likely to adopt " + product + " first?",
		"What happens if a larger competitor copies " + product + "?",
		"What is the biggest risk to " + company.Name + " over the next 12 months?",
		"What would make " + product + " fail in its first 90 days?",
	}
	if len(products) > 1 {
		suggestions = append(suggestions, "Should we bundle "+products[0].Name+" and "+products[1].Name+"?")
	}
	return suggestions
}

func productNames(company *store.Company, products []store.Company) []string {
	names := []string{company.Name}
	for _, product := range products {
		names = append(names, product.Name)
	}
	return names
}

func clampSimulation(agents, rounds int) (int, int) {
	if agents < 6 {
		agents = 12
	}
	if agents > 30 {
		agents = 30
	}
	if rounds < 1 {
		rounds = 3
	}
	if rounds > 5 {
		rounds = 5
	}
	return agents, rounds
}

func summarizeReactions(reactions []map[string]any) string {
	parts := make([]string, 0, len(reactions))
	for _, reaction := range reactions {
		persona, _ := reaction["persona"].(string)
		text, _ := reaction["reaction"].(string)
		stance, _ := reaction["stance"].(string)
		if persona != "" && text != "" {
			parts = append(parts, persona+" ("+stance+"): "+text)
		}
	}
	return strings.Join(parts, "\n")
}

func miroFishJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}

func formatInt(value int) string {
	text := fmt.Sprintf("%d", value)
	if len(text) <= 3 {
		return text
	}
	var out []byte
	for i, char := range text {
		if i > 0 && (len(text)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(char))
	}
	return string(out)
}
