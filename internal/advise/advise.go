package advise

import (
	"fmt"
	"strings"

	"github.com/davepartner/crew/internal/store"
)

type Check struct {
	Label    string `json:"label"`
	Done     bool   `json:"done"`
	Evidence string `json:"evidence"`
	Action   string `json:"action"`
}

type AgentRec struct {
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Emoji        string   `json:"emoji"`
	Instructions string   `json:"instructions"`
	Skills       []string `json:"skills"`
}

type Section struct {
	Key         string     `json:"key"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	Checks      []Check    `json:"checks"`
	Actions     []string   `json:"actions"`
	Agents      []AgentRec `json:"agents"`
	CountryNote string     `json:"country_note"`
}

type Report struct {
	CompanyName string         `json:"company_name"`
	Country     CountryProfile `json:"country"`
	Stage       string         `json:"stage"`
	StageLabel  string         `json:"stage_label"`
	Score       int            `json:"score"`
	Done        int            `json:"done"`
	Total       int            `json:"total"`
	Sections    []Section      `json:"sections"`
	Next        []string       `json:"next"`
	HasModel    bool           `json:"has_model"`
}

type Input struct {
	Company   store.Company
	Agents    []store.Agent
	Tasks     []store.Task
	Routines  []store.Routine
	Goals     []store.Goal
	Members   []store.Member
	Campaigns []store.AdCampaign
	HasModel  bool
}

func Build(input Input) Report {
	company := input.Company
	completed := 0
	for _, task := range input.Tasks {
		if task.Status == store.StatusDone {
			completed++
		}
	}

	hasMarketing := hasRole(input.Agents, "Marketing", "Audience", "Growth")
	hasSupport := hasRole(input.Agents, "Support", "Client success", "Patient experience")
	hasOps := hasRole(input.Agents, "Operations", "Delivery", "Product")
	hasSkills := false
	for _, agent := range input.Agents {
		if len(agent.Skills) > 0 {
			hasSkills = true
		}
	}
	hasRoutine := len(input.Routines) > 0
	hasGoal := len(input.Goals) > 0
	hasCampaign := len(input.Campaigns) > 0
	hasTeammate := len(input.Members) > 1
	hasTeardown := false
	for _, task := range input.Tasks {
		if strings.HasPrefix(strings.ToLower(task.Title), "marketing teardown") {
			hasTeardown = true
		}
	}

	profile := Country(company.Country)

	sections := []Section{
		section("foundation", "Foundation", "Who you are, what you sell and who buys it.",
			[]Check{
				check("Company name and description", trimLen(company.Description) >= 40, "Add a clear paragraph about what the business does.", "Write a 3-sentence description of what you sell and for whom."),
				check("Target audience", company.Audience != "", "Say who the business serves.", "Fill in the audience field in Settings → Company."),
				check("Tone of voice", company.Tone != "", "Choose how the brand should sound.", "Pick a tone in Settings → Company."),
				check("Country set", company.Country != "", "Country drives the legal, tax and payment advice.", "Set your country in Settings → Company."),
			},
			[]string{
				"Write one sentence: we help [audience] get [outcome] without [pain].",
				"Put that sentence on the website, Instagram bio and WhatsApp catalogue.",
			},
			[]AgentRec{{
				Name: "General manager", Role: "Operations", Emoji: "🧭",
				Instructions: "You are the general manager for " + company.Name + ". Keep the week organised, surface decisions, and turn vague goals into next actions. Read the company profile before advising.",
			}},
			"Register the business and keep business money separate: "+strings.Join(profile.Registration, " "),
		),
		section("customer", "Customer & research", "Understand the person who pays, in their own words.",
			[]Check{
				check("Audience is specific", len(company.Audience) >= 12, "The audience is still broad.", "Narrow it to a person, place and situation, not “everyone”."),
				check("Customer research skill", hasSkill(input.Agents, "customer-research"), "No helper has the customer research playbook.", "Attach the customer-research skill to a helper."),
				check("A support helper exists", hasSupport, "No one owns customer questions.", "Create a support helper and give it your refund and escalation policy."),
				check("Real customer evidence", hasTeardown || completed > 0, "No completed customer-facing work yet.", "Run one customer interview or teardown and save the notes as a task."),
			},
			[]string{
				"Ask five customers: what did you try before us, what almost stopped you, and what would you miss?",
				"Save their exact words; those phrases become your ads.",
			},
			[]AgentRec{{
				Name: "Customer researcher", Role: "Researcher", Emoji: "🔎",
				Instructions: "You research customers for " + company.Name + ". Turn interviews, reviews and support messages into pains, triggers and objections. Quote real words; never invent quotes.",
				Skills:       []string{"customer-research"},
			}},
			"",
		),
		section("offer", "Offer & pricing", "A reason to buy now, not just a product description.",
			[]Check{
				check("Clear product description", trimLen(company.Description) >= 80, "The offer is still vague.", "Describe the outcome, the mechanism and what is included."),
				check("Pricing skill", hasSkill(input.Agents, "pricing"), "No helper owns pricing.", "Attach the pricing skill and write your price logic down."),
				check("A measurable goal", hasGoal, "There is no goal to test the offer against.", "Set one goal: demos, sales, leads or revenue."),
				check("Marketing helper exists", hasMarketing, "No one is responsible for taking the offer to market.", "Create a marketing helper."),
			},
			[]string{
				"Write the offer as: outcome + timeframe + risk reversal.",
				"Test one price and one guarantee; do not discount without a reason.",
			},
			[]AgentRec{{
				Name: "Offer strategist", Role: "Marketing", Emoji: "🎯",
				Instructions: "You design the offer and pricing for " + company.Name + ". Make the outcome specific, list what is included, and surface every assumption that needs a real price test.",
				Skills:       []string{"offers", "pricing"},
			}},
			"Use "+profile.Currency+" for local pricing and review it when the exchange rate moves.",
		),
		section("brand", "Brand & website", "Make the business look real before someone pays.",
			[]Check{
				check("Website", company.Website != "", "There is no website on the profile.", "Add the site, or use a one-page site with offer, proof and contact."),
				check("Support email", company.SupportEmail != "", "Customers have no clear way to reach you.", "Add a support email and put it on the site."),
				check("Copywriting skill", hasSkill(input.Agents, "copywriting"), "No helper owns the words.", "Attach the copywriting skill."),
				check("Trust proof", strings.Contains(strings.ToLower(company.Description), "review") || strings.Contains(strings.ToLower(company.Description), "years") || hasCampaign, "There is no visible proof.", "Add real reviews, numbers or a demo; never invent them."),
			},
			[]string{
				"Put the promise, proof and one CTA above the fold.",
				"Use the same offer on the site, ads and WhatsApp replies.",
			},
			[]AgentRec{{
				Name: "Brand writer", Role: "Writer", Emoji: "✍️",
				Instructions: "You write the website and brand copy for " + company.Name + ". Be specific, use the customer's words, and never invent proof. Keep one promise and one CTA per page.",
				Skills:       []string{"copywriting", "copy-editing"},
			}},
			"",
		),
		section("acquisition", "Acquisition", "A repeatable way to get in front of buyers.",
			[]Check{
				check("Marketing helper", hasMarketing, "No marketing owner.", "Create a growth/marketing helper."),
				check("Ad campaign generated", hasCampaign, "No ad concepts have been generated.", "Use Marketing → Ads Studio for one campaign."),
				check("Competitor teardown", hasTeardown, "No competitor study saved.", "Run Marketing → Reverse-engineer on one competitor."),
				check("Marketing skill attached", hasSkills, "No marketing playbooks attached to helpers.", "Attach copywriting, content strategy or ads skills."),
			},
			[]string{
				"Pick one channel and one offer for 14 days; do not spread across five channels.",
				"Publish or send something every weekday and track replies in one sheet.",
			},
			[]AgentRec{{
				Name: "Growth lead", Role: "Marketing", Emoji: "📣",
				Instructions: "You find customers for " + company.Name + ". Run one channel at a time, write native ads and posts, and report replies, leads and cost per lead. No invented performance claims.",
				Skills:       []string{"ads", "content-strategy", "ad-creative"},
			}},
			"Start with the channel where customers already are: "+strings.Join(profile.Ads, " "),
		),
		section("sales", "Sales & follow-up", "Most small businesses lose money after the first reply.",
			[]Check{
				check("Someone owns replies", hasSupport, "No one owns the inbox.", "Create a support/sales helper."),
				check("Follow-up routine", hasRoutine, "No scheduled follow-up.", "Create a weekday routine to follow up every lead."),
				check("A written reply script", hasGoal, "No reply or offer script.", "Write the first reply, the quote and the follow-up."),
				check("A lead tracker", len(input.Tasks) >= 3, "Leads are only in chat.", "Create one task per lead or use a simple sheet."),
			},
			[]string{
				"Reply within one hour during work hours.",
				"Every conversation ends with a next step and a date.",
			},
			[]AgentRec{{
				Name: "Sales follow-up", Role: "Sales", Emoji: "🤝",
				Instructions: "You handle leads for " + company.Name + ". Reply fast, qualify, quote clearly, and follow up until there is a yes or a no. Escalate anything legal or angry.",
				Skills:       []string{"cold-email", "offers"},
			}},
			"",
		),
		section("delivery", "Delivery & operations", "Keep promises and remove the founder from every step.",
			[]Check{
				check("Operations helper", hasOps, "No one owns delivery.", "Create an operations helper."),
				check("A completed task", completed > 0, "Nothing has been completed yet.", "Finish one paid or internal task end to end."),
				check("A recurring routine", hasRoutine, "Operations are not scheduled.", "Create one weekly delivery/review routine."),
				check("Delegation enabled", anyCanDelegate(input.Agents), "Helpers cannot hand work on.", "Enable delegation on at least one helper."),
			},
			[]string{
				"Write the delivery checklist once; reuse it every time.",
				"Track turnaround time and the first thing that breaks.",
			},
			[]AgentRec{{
				Name: "Delivery lead", Role: "Operations", Emoji: "🛠️",
				Instructions: "You run delivery for " + company.Name + ". Turn every order into a checklist, keep the customer informed, and flag delays before they become complaints.",
			}},
			"",
		),
		section("money", "Money, legal & country", "The boring parts that decide whether you keep the profit.",
			[]Check{
				check("Country set", company.Country != "", "No country on the company.", "Set the country."),
				check("AI/task budget", company.MonthlyBudgetUSD > 0 || company.MonthlyTaskLimit > 0, "No spending guardrail.", "Set a monthly AI or task budget."),
				check("A money goal", hasGoal, "No revenue/cost goal.", "Set one money goal for the month."),
				check("Compliance helper", hasRole(input.Agents, "Compliance", "Analysis"), "No one checks claims and documents.", "Create a compliance/analysis helper."),
			},
			[]string{
				"Open a business account and take a deposit before work starts.",
				"Keep every invoice, receipt and contract in one folder.",
			},
			[]AgentRec{{
				Name: "Finance checker", Role: "Compliance", Emoji: "🧾",
				Instructions: "You check money and claims for " + company.Name + ". Flag missing invoices, unsupported ad claims, tax dates and cash risks. Do not give legal or tax advice; route to a professional.",
				Skills:       []string{"pricing"},
			}},
			strings.Join(append(append([]string{}, profile.Tax...), profile.Payments...), " "),
		),
		section("team", "Team & systems", "Build the smallest team that can run without you.",
			[]Check{
				check("A second person", hasTeammate, "It is only you.", "Invite an editor, VA or partner as a viewer/editor."),
				check("Helpers with skills", hasSkills, "Helpers have no playbooks.", "Attach at least one skill per helper."),
				check("Routines", hasRoutine, "Nothing repeats automatically.", "Schedule a weekly review."),
				check("Goals", hasGoal, "No shared scoreboard.", "Set one goal the team can see."),
			},
			[]string{
				"Write down what only you can do; delegate the rest.",
				"Review the week every Friday: leads, delivery, money, one fix.",
			},
			[]AgentRec{{
				Name: "Operations lead", Role: "Operations", Emoji: "🧭",
				Instructions: "You keep " + company.Name + " running week to week. Maintain the priorities, chase blockers, and prepare a Friday review with numbers and decisions.",
			}},
			"",
		),
		section("growth", "Growth", "Compound what works instead of starting over.",
			[]Check{
				check("Ads tested", hasCampaign, "No ad campaign yet.", "Generate and run one campaign."),
				check("Completed work", completed >= 3, "Too little delivered to learn from.", "Deliver three jobs and record what happened."),
				check("A growth goal", hasGoal, "No growth target.", "Set a 30-day growth goal."),
				check("A team to scale", hasTeammate || len(input.Agents) >= 3, "The founder is still the whole team.", "Add one helper or teammate before increasing spend."),
			},
			[]string{
				"Double down on the one channel and one offer that produced the best conversations.",
				"Raise price or add a retainer before adding more clients.",
			},
			[]AgentRec{{
				Name: "Growth lead", Role: "Audience", Emoji: "📈",
				Instructions: "You grow " + company.Name + " after first traction. Find the winning message, turn it into a repeatable weekly cadence, and report one number that matters.",
				Skills:       []string{"marketing-plan", "analytics"},
			}},
			"",
		),
	}

	done, total := 0, 0
	next := make([]string, 0, 6)
	for _, section := range sections {
		for _, check := range section.Checks {
			total++
			if check.Done {
				done++
			}
		}
		if section.Status != "done" {
			for _, check := range section.Checks {
				if !check.Done {
					next = append(next, check.Action)
					break
				}
			}
		}
	}
	if len(next) > 5 {
		next = next[:5]
	}

	stage, label := stageFor(input, completed)

	score := 0
	if total > 0 {
		score = done * 100 / total
	}

	return Report{
		CompanyName: company.Name,
		Country:     profile,
		Stage:       stage,
		StageLabel:  label,
		Score:       score,
		Done:        done,
		Total:       total,
		Sections:    sections,
		Next:        next,
		HasModel:    input.HasModel,
	}
}

func section(key, title, summary string, checks []Check, actions []string, agents []AgentRec, countryNote string) Section {
	done := 0
	for _, check := range checks {
		if check.Done {
			done++
		}
	}
	status := "todo"
	if done == len(checks) {
		status = "done"
	} else if done > 0 {
		status = "partial"
	}
	progress := 0
	if len(checks) > 0 {
		progress = done * 100 / len(checks)
	}

	return Section{
		Key: key, Title: title, Summary: summary, Status: status, Progress: progress,
		Checks: checks, Actions: actions, Agents: agents, CountryNote: countryNote,
	}
}

func check(label string, done bool, evidence, action string) Check {
	return Check{Label: label, Done: done, Evidence: evidence, Action: action}
}

func hasRole(agents []store.Agent, roles ...string) bool {
	for _, agent := range agents {
		for _, role := range roles {
			if strings.EqualFold(agent.Role, role) {
				return true
			}
		}
	}
	return false
}

func hasSkill(agents []store.Agent, skill string) bool {
	for _, agent := range agents {
		for _, key := range agent.Skills {
			if key == skill {
				return true
			}
		}
	}
	return false
}

func anyCanDelegate(agents []store.Agent) bool {
	for _, agent := range agents {
		if agent.CanDelegate {
			return true
		}
	}
	return false
}

func trimLen(value string) int {
	return len([]rune(strings.TrimSpace(value)))
}

func stageFor(input Input, completed int) (string, string) {
	if len(input.Agents) == 0 && len(input.Tasks) == 0 {
		return "idea", "Idea — before the first real customer"
	}
	if completed == 0 {
		return "validation", "Validation — testing whether people will pay"
	}
	if completed < 5 || len(input.Campaigns) == 0 {
		return "early-traction", "Early traction — some customers, not yet repeatable"
	}
	if len(input.Members) > 1 || len(input.Routines) > 0 {
		return "growth", "Growth — the model works; build the machine"
	}
	return "early-traction", "Early traction — some customers, not yet repeatable"
}

func Summary(report Report) string {
	return fmt.Sprintf("%s · %s · %d%% foundation score", report.CompanyName, report.StageLabel, report.Score)
}
