package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/davepartner/crew/internal/advise"
	"github.com/davepartner/crew/internal/llm"
	"github.com/davepartner/crew/internal/presets"
	"github.com/davepartner/crew/internal/research"
	"github.com/davepartner/crew/internal/store"
)

func (s *Server) handleAdvice(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	input, err := s.adviceInput(r.Context(), company.ID, id.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't study this company. Please refresh.")
		return
	}

	report := advise.Build(input)

	var cached any
	if advice, err := s.store.GetCompanyAdvice(r.Context(), company.ID); err == nil {
		cached = advice
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"report":  report,
		"advice":  cached,
		"company": company,
	})
}

func (s *Server) handleStudyCompany(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	input, err := s.adviceInput(r.Context(), company.ID, id.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't study this company. Please refresh.")
		return
	}
	report := advise.Build(input)

	model, err := marketingModelFor(s, r.Context(), id.UserID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"ok": false, "code": "no_model",
			"error": "Add a model key in Settings → AI models, or use the included model.",
		})
		return
	}

	if blocked, err := s.checkBudget(r.Context(), id.UserID, company.ID, ""); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't check the AI budget.")
		return
	} else if blocked != "" {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"ok": false, "code": "budget", "error": blocked})
		return
	}

	studyContext := map[string]any{
		"name": company.Name, "business": company.BusinessKey, "country": company.Country,
		"website": company.Website, "support_email": company.SupportEmail,
		"description": company.Description, "audience": company.Audience, "tone": company.Tone,
		"stage": report.Stage, "score": report.Score, "country_profile": report.Country,
		"sections": report.Sections, "next": report.Next,
	}
	if snapshots, err := s.store.ListFinancialSnapshots(r.Context(), company.ID); err == nil && len(snapshots) > 0 {
		studyContext["financial_model"] = financialContext(snapshots)
	}
	if files, err := s.store.ListCompanyFiles(r.Context(), company.ID); err == nil {
		studyContext["files"] = fileContext(files, 7000)
	}
	if researchText, _ := s.webResearch(r.Context(), company, company.Name+" "+company.BusinessKey+" "+company.Country+" market customers competitors pricing"); researchText != "" {
		studyContext["web_research"] = researchText
	}
	companyJSON, _ := json.Marshal(studyContext)

	system := `You are a practical business operator advising a small company.
Use only the supplied company data and the country profile. Do not invent revenue,
customers, market sizes or performance. Be specific and blunt. Return valid JSON only:
{"headline":"one sentence","constraints":[{"title":"","why":"","fix":""}],
"next_7_days":[""],"opportunities":[""],"risks":[""]}
Give 3 constraints, 5 next actions, 3 opportunities and 3 risks.`

	ctx, cancel := contextWithTimeout(r, 90*time.Second)
	defer cancel()

	result, err := llm.CompleteResult(ctx, llm.Config{
		Kind: model.Provider.Kind, BaseURL: model.Provider.BaseURL, Model: model.Model, APIKey: model.APIKey,
	}, system, "Company data:\n"+string(companyJSON)+"\n\nStudy this company and tell the founder what to do next.", 2200)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The model couldn't study the company: "+err.Error())
		return
	}

	document, err := extractJSONObject(result.Text)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The model returned an unreadable study. Try again.")
		return
	}

	cost := s.saveUsage(ctx, id.UserID, company.ID, "", "", model, result.Usage)
	_ = s.store.SaveCompanyAdvice(r.Context(), company.ID, document, model.Provider.Name+" · "+model.Model)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"advice": store.CompanyAdvice{Analysis: document, Model: model.Provider.Name + " · " + model.Model, CreatedAt: time.Now().UTC()},
		"cost":   cost,
	})
}

type chatTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatSource struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (s *Server) handleAdviceChat(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r)
	company := companyFrom(r)

	var body struct {
		Message string     `json:"message"`
		History []chatTurn `json:"history"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	message := trim(body.Message, 2000)
	if message == "" {
		writeError(w, http.StatusUnprocessableEntity, "Ask a question first.")
		return
	}

	input, err := s.adviceInput(r.Context(), company.ID, id.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't read this company.")
		return
	}
	report := advise.Build(input)

	model, err := marketingModelFor(s, r.Context(), id.UserID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "code": "no_model", "error": "Add a model key to chat."})
		return
	}

	if blocked, err := s.checkBudget(r.Context(), id.UserID, company.ID, ""); err != nil {
		writeError(w, http.StatusInternalServerError, "We couldn't check the AI budget.")
		return
	} else if blocked != "" {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"ok": false, "code": "budget", "error": blocked})
		return
	}

	// Internet context: the company's own site and a Wikipedia summary for the
	// country and business type. No scraping of search engines.
	sources := make([]chatSource, 0, 3)
	webContext := ""

	if company.Website != "" {
		page, err := fetchPage(r.Context(), company.Website)
		if err == nil {
			text := page.Text
			if len(text) > 7000 {
				text = text[:7000]
			}
			webContext += "\n\nCOMPANY WEBSITE (" + page.URL + "):\n" + text
			sources = append(sources, chatSource{Title: page.Title + " (company website)", URL: page.URL})
		}
	}

	businessName := company.BusinessKey
	if preset, ok := presets.ByKey(company.BusinessKey); ok {
		businessName = preset.Name
	}
	for _, title := range []string{company.Country, businessName} {
		if title == "" {
			continue
		}
		if summary, pageURL, ok := wikiSummary(r.Context(), title); ok {
			webContext += "\n\nWIKIPEDIA — " + title + ":\n" + summary
			sources = append(sources, chatSource{Title: "Wikipedia: " + title, URL: pageURL})
		}
	}

	for _, result := range wikiSearch(r.Context(), message, 2) {
		webContext += "\n\nWIKIPEDIA — " + result.Title + ":\n" + result.Snippet
		sources = append(sources, chatSource{Title: "Wikipedia: " + result.Title, URL: result.URL})
	}

	if researchText, webSources := s.webResearchDetailed(r.Context(), company, message+" "+company.Name+" "+businessName+" "+company.Country); researchText != "" {
		webContext += "\n\nLIVE WEB / INDUSTRY:\n" + researchText
		sources = append(sources, webSources...)
	}

	chatContext := map[string]any{
		"company": company, "stage": report.StageLabel, "score": report.Score,
		"next": report.Next, "sections": report.Sections,
		"agents": summarizeAgents(input.Agents), "tasks": summarizeTasks(input.Tasks),
		"campaigns": len(input.Campaigns), "country": report.Country,
	}
	if snapshots, err := s.store.ListFinancialSnapshots(r.Context(), company.ID); err == nil && len(snapshots) > 0 {
		chatContext["financial_model"] = financialContext(snapshots)
	}
	if files, err := s.store.ListCompanyFiles(r.Context(), company.ID); err == nil {
		chatContext["files"] = fileContext(files, 6000)
	}
	contextJSON, _ := json.Marshal(chatContext)

	system := `You are the founder's operator. Answer questions about this company using the supplied
company data and web extracts. Be concrete, use numbers from the data, and say "I don't know"
when the data does not support an answer. Do not invent testimonials, revenue or performance.
Cite a source URL when you use one. Keep answers under 300 words unless asked for a plan.`

	prompt := "Company context:\n" + string(contextJSON) + "\n\nWeb extracts:" + webContext +
		"\n\nQuestion: " + message

	ctx, cancel := contextWithTimeout(r, 90*time.Second)
	defer cancel()

	history := body.History
	if len(history) > 8 {
		history = history[len(history)-8:]
	}
	var conversation strings.Builder
	for _, turn := range history {
		if turn.Content == "" || (turn.Role != "user" && turn.Role != "assistant") {
			continue
		}
		conversation.WriteString(turn.Role + ": " + trim(turn.Content, 1500) + "\n")
	}
	if conversation.Len() > 0 {
		prompt = "Recent conversation:\n" + conversation.String() + "\n" + prompt
	}

	result, err := llm.CompleteResult(ctx, llm.Config{
		Kind: model.Provider.Kind, BaseURL: model.Provider.BaseURL, Model: model.Model, APIKey: model.APIKey,
	}, system, prompt, 1400)
	if err != nil {
		writeError(w, http.StatusBadGateway, "The advisor couldn't answer: "+err.Error())
		return
	}

	cost := s.saveUsage(ctx, id.UserID, company.ID, "", "", model, result.Usage)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"answer":  result.Text,
		"sources": sources,
		"cost":    cost,
	})
}

func (s *Server) adviceInput(ctx context.Context, companyID, userID string) (advise.Input, error) {
	company, err := s.store.GetCompany(ctx, userID, companyID)
	if err != nil {
		return advise.Input{}, err
	}
	agents, err := s.store.ListAgents(ctx, companyID)
	if err != nil {
		return advise.Input{}, err
	}
	tasks, err := s.store.ListTasks(ctx, companyID, "")
	if err != nil {
		return advise.Input{}, err
	}
	routines, _ := s.store.ListRoutines(ctx, companyID)
	goals, _ := s.store.ListGoals(ctx, companyID)
	members, _ := s.store.ListMembers(ctx, companyID)
	campaigns, _ := s.store.ListAdCampaigns(ctx, companyID, 20)

	hasModel := false
	if _, ok := s.platformModel(); ok {
		hasModel = true
	}
	if !hasModel {
		if keys, err := s.store.ListProviderKeys(ctx, userID); err == nil && len(keys) > 0 {
			hasModel = true
		}
	}

	return advise.Input{
		Company: company, Agents: agents, Tasks: tasks, Routines: routines,
		Goals: goals, Members: members, Campaigns: campaigns, HasModel: hasModel,
	}, nil
}

func summarizeAgents(agents []store.Agent) []map[string]any {
	out := make([]map[string]any, 0, len(agents))
	for _, agent := range agents {
		out = append(out, map[string]any{
			"name": agent.Name, "role": agent.Role, "skills": agent.Skills,
			"delegates": agent.CanDelegate, "budget_usd": agent.MonthlyBudgetUSD,
		})
	}
	return out
}

func summarizeTasks(tasks []store.Task) []map[string]any {
	out := make([]map[string]any, 0, 10)
	for i, task := range tasks {
		if i >= 10 {
			break
		}
		out = append(out, map[string]any{"title": task.Title, "status": task.Status})
	}
	return out
}

func wikiSummary(ctx context.Context, title string) (string, string, bool) {
	endpoint := "https://en.wikipedia.org/api/rest_v1/page/summary/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", false
	}
	request.Header.Set("User-Agent", "CrewAdvisor/1.0 (https://crew.paytrackify.com)")
	request.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		return "", "", false
	}
	defer response.Body.Close()

	var payload struct {
		Extract     string `json:"extract"`
		ContentURLs struct {
			Desktop struct {
				Page string `json:"page"`
			} `json:"desktop"`
		} `json:"content_urls"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil || payload.Extract == "" {
		return "", "", false
	}
	return payload.Extract, payload.ContentURLs.Desktop.Page, true
}

type wikiResult struct {
	Title   string
	Snippet string
	URL     string
}

var wikiTagPattern = regexp.MustCompile(`<[^>]+>`)

func wikiSearch(ctx context.Context, query string, limit int) []wikiResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if len([]rune(query)) > 180 {
		query = string([]rune(query)[:180])
	}

	endpoint := "https://en.wikipedia.org/w/api.php?action=query&list=search&format=json&utf8=1&srsearch=" +
		url.QueryEscape(query) + "&srlimit=2"
	if limit <= 0 || limit > 3 {
		limit = 2
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil
	}
	request.Header.Set("User-Agent", "CrewAdvisor/1.0 (https://crew.paytrackify.com)")
	request.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		return nil
	}
	defer response.Body.Close()

	var payload struct {
		Query struct {
			Search []struct {
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
				PageID  int    `json:"pageid"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil
	}

	out := make([]wikiResult, 0, limit)
	for _, item := range payload.Query.Search {
		if len(out) >= limit {
			break
		}
		snippet := html.UnescapeString(wikiTagPattern.ReplaceAllString(item.Snippet, " "))
		out = append(out, wikiResult{
			Title:   item.Title,
			Snippet: strings.Join(strings.Fields(snippet), " "),
			URL:     "https://en.wikipedia.org/?curid=" + strconv.Itoa(item.PageID),
		})
	}
	return out
}

func fileContext(files []store.CompanyFile, budget int) []map[string]string {
	out := make([]map[string]string, 0, 3)
	used := 0
	for _, file := range files {
		text := strings.TrimSpace(file.Text)
		if text == "" || used >= budget {
			continue
		}
		if len(text) > 2500 {
			text = text[:2500]
		}
		if used+len(text) > budget {
			text = text[:budget-used]
		}
		out = append(out, map[string]string{"name": file.Name, "mime": file.Mime, "text": text})
		used += len(text)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func (s *Server) webResearch(ctx context.Context, company *store.Company, query string) (string, []chatSource) {
	return s.webResearchDetailed(ctx, company, query)
}

func (s *Server) webResearchDetailed(ctx context.Context, company *store.Company, query string) (string, []chatSource) {
	if s.cfg.Env == "test" {
		return "", nil
	}

	braveKey := ""
	if config, err := s.loadIntegration(ctx, company.ID, "brave_search"); err == nil {
		braveKey = config["api_key"]
	}

	var builder strings.Builder
	sources := make([]chatSource, 0, 5)

	for _, result := range research.Search(ctx, query, braveKey) {
		if builder.Len() > 3500 {
			break
		}
		builder.WriteString("\n- " + result.Title + " (" + result.Provider + "): " + result.Snippet + "\n  " + result.URL)
		sources = append(sources, chatSource{Title: result.Provider + ": " + result.Title, URL: result.URL})
	}

	if company.Ticker != "" {
		if quote, err := research.Market(ctx, company.Ticker); err == nil {
			builder.WriteString(fmt.Sprintf("\n- Market: %s %s %.2f, previous %.2f, change %.2f%%, currency %s, as of %s\n  %s",
				quote.Ticker, "", quote.Price, quote.Previous, quote.ChangePct, quote.Currency, quote.AsOf, quote.URL))
			sources = append(sources, chatSource{Title: "Market data: " + quote.Ticker, URL: quote.URL})
		}
	}

	return builder.String(), sources
}

var errNoModel = errors.New("no model")
