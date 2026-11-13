// Package presets is the catalogue of businesses Crew can run out of the box.
//
// Each category defines a small, opinionated team and three starter tasks. A
// business contributes its own focus sentence, integrations and tasks, so the
// resulting team feels specific without duplicating a hundred paragraphs.
package presets

import "strings"

type AgentPreset struct {
	Name         string
	Role         string
	Emoji        string
	Instructions string
}

type TaskPreset struct {
	Title   string
	Details string
	Agent   string
}

type Business struct {
	Key          string
	Name         string
	Category     string
	Emoji        string
	Blurb        string
	Focus        string
	Integrations []string
	Agents       []AgentPreset
	Tasks        []TaskPreset
}

type roleSpec struct {
	name, role, emoji, instructions string
}

type taskSpec struct {
	title, details, agent string
}

type category struct {
	key   string
	roles []roleSpec
	tasks []taskSpec
}

type businessSeed struct {
	key, name, category, emoji, blurb, focus string
	integrations                             []string
}

var categories = map[string]category{
	"retail": {
		key: "retail",
		roles: []roleSpec{
			{"Operations lead", "Operations", "🧭", "You run day-to-day operations for {{company}}. {focus} Watch orders, stock and fulfilment, and flag anything that delays a customer or costs money. Keep updates short and specific."},
			{"Customer support lead", "Support", "🛟", "You answer customers for {{company}} in a {{tone}} tone. {focus} Resolve politely, stay inside the refund policy, and escalate anything angry, legal or safety-related."},
			{"Marketing lead", "Marketing", "📣", "You grow sales for {{company}}. {focus} Write offers, emails and product copy for {{audience}}. Never invent a discount or a claim we cannot support."},
		},
		tasks: []taskSpec{
			{"Review yesterday's orders and flag anything stuck", "Look for unfulfilled, delayed or refunded orders. Give me a short list with the next action for each.", "Operations lead"},
			{"Draft replies to the three most common customer questions", "Use our policies and a {{tone}} voice. Keep each reply under 120 words.", "Customer support lead"},
			{"Write a five-email welcome series for new subscribers", "One idea per email, one clear call to action, no invented discounts.", "Marketing lead"},
		},
	},
	"software": {
		key: "software",
		roles: []roleSpec{
			{"Product lead", "Product", "🧭", "You own the roadmap for {{company}}. {focus} Turn requests into small, testable steps and say no to anything off-strategy."},
			{"Engineer", "Engineering", "💻", "You build and review {{company}}'s product. {focus} Prefer small changes, tests and clear commit messages. Never touch production without approval."},
			{"Support engineer", "Support", "🛟", "You turn user reports into reproducible issues for {{company}}. {focus} Ask for steps, version and screenshots, then tag severity."},
		},
		tasks: []taskSpec{
			{"Triage this week's user feedback", "Group it into bugs, confusion and requests. Pick the top three by impact.", "Product lead"},
			{"Write the release notes for the next version", "Plain language, what changed and why it matters to {{audience}}.", "Engineer"},
			{"Reproduce the top bug and write the issue", "Include steps, expected vs actual, and the smallest fix you would try.", "Support engineer"},
		},
	},
	"agency": {
		key: "agency",
		roles: []roleSpec{
			{"Account lead", "Client success", "🤝", "You look after clients for {{company}}. {focus} Keep them informed before they ask, and flag scope creep early."},
			{"Delivery lead", "Delivery", "🧭", "You run delivery for {{company}}. {focus} Break work into milestones, protect the team's focus, and surface blockers the same day."},
			{"Growth lead", "Marketing", "📣", "You win new work for {{company}}. {focus} Write outreach and case studies for {{audience}}. No hype, no fake numbers."},
		},
		tasks: []taskSpec{
			{"Follow up every lead from the last two weeks", "One short, personal email each. Reference what they asked for.", "Growth lead"},
			{"Write a one-page status update for each active client", "What moved, what is next, what we need from them.", "Account lead"},
			{"Turn the last project into a case study", "Problem, approach, result. Only use numbers we can defend.", "Growth lead"},
		},
	},
	"content": {
		key: "content",
		roles: []roleSpec{
			{"Editor", "Editorial", "✍️", "You edit for {{company}}. {focus} Cut anything that does not earn its place, keep the {{tone}} voice, and check every claim."},
			{"Producer", "Production", "🎬", "You plan and ship content for {{company}}. {focus} Keep a calendar, brief the work, and make sure every piece has one clear idea."},
			{"Growth lead", "Audience", "📈", "You grow the audience for {{company}}. {focus} Repurpose what already works and write hooks for {{audience}} without clickbait."},
		},
		tasks: []taskSpec{
			{"Plan next month's content calendar", "Two pieces a week, one theme per week, with the hook for each.", "Producer"},
			{"Draft the next issue or episode", "One idea, a strong opening, and a single call to action.", "Editor"},
			{"Repurpose the best piece into five short posts", "Keep the voice, change the angle for each platform.", "Growth lead"},
		},
	},
	"health": {
		key: "health",
		roles: []roleSpec{
			{"Front desk", "Patient experience", "🛟", "You handle bookings and questions for {{company}}. {focus} Be warm and precise. Never give medical advice; route clinical questions to a professional."},
			{"Practitioner assistant", "Clinical support", "🩺", "You support the practitioners at {{company}}. {focus} Prepare notes and reminders, but never diagnose, prescribe or promise an outcome."},
			{"Community lead", "Marketing", "📣", "You write for {{company}}'s community. {focus} Keep it {{tone}} and evidence-based. No before/after claims or scare tactics."},
		},
		tasks: []taskSpec{
			{"Send tomorrow's appointment reminders", "Friendly, short, include time and what to bring.", "Front desk"},
			{"Write the follow-up message for this week's new clients", "Check in, share one useful tip, invite questions.", "Practitioner assistant"},
			{"Draft answers to the five questions we get most", "General information only, with a clear 'ask your clinician' where needed.", "Community lead"},
		},
	},
	"trades": {
		key: "trades",
		roles: []roleSpec{
			{"Dispatcher", "Operations", "🧭", "You schedule jobs for {{company}}. {focus} Keep the calendar realistic, confirm every visit, and tell customers early if we are running late."},
			{"Estimator", "Sales", "📐", "You prepare quotes for {{company}}. {focus} Ask for the details that change the price, and never promise a number you cannot stand behind."},
			{"Reviews lead", "Reputation", "⭐", "You look after {{company}}'s reputation. {focus} Reply to every review within a day, thank the good ones and take complaints offline."},
		},
		tasks: []taskSpec{
			{"Follow up the quotes sent in the last 7 days", "One message each, offer two times for a call.", "Estimator"},
			{"Confirm tomorrow's jobs with every customer", "Time, address, who is coming, and what to expect.", "Dispatcher"},
			{"Reply to this month's reviews", "Warm and specific. No copy-paste.", "Reviews lead"},
		},
	},
	"finance": {
		key: "finance",
		roles: []roleSpec{
			{"Client manager", "Client success", "🤝", "You look after clients for {{company}}. {focus} Keep requests clear and chase documents politely."},
			{"Compliance checker", "Compliance", "✅", "You check work for {{company}}. {focus} Flag anything that looks regulated, incomplete or out of date. You do not give legal or investment advice."},
			{"Analyst", "Analysis", "📊", "You prepare summaries for {{company}}. {focus} Show the numbers, the assumptions and the questions a client should ask."},
		},
		tasks: []taskSpec{
			{"Chase the documents we are missing this month", "One list, one message per client, clear deadline.", "Client manager"},
			{"Run the month-end checklist", "Mark what is done, what is missing and what needs a human.", "Compliance checker"},
			{"Write a one-page summary for each client", "What changed, what it means, what to do next.", "Analyst"},
		},
	},
	"education": {
		key: "education",
		roles: []roleSpec{
			{"Admissions lead", "Enrolment", "🎓", "You help people join {{company}}. {focus} Answer honestly, explain the time and cost, and follow up without pressure."},
			{"Instructor", "Teaching", "📚", "You prepare lessons for {{company}}. {focus} Start from the outcome, keep each session small, and check understanding."},
			{"Community lead", "Community", "💬", "You keep {{company}}'s learners engaged. {focus} Ask questions, celebrate progress, and spot people who are stuck."},
		},
		tasks: []taskSpec{
			{"Write the welcome sequence for new learners", "Three emails: what to expect, how to start, where to ask for help.", "Admissions lead"},
			{"Plan the next lesson", "One outcome, three steps, one way to check understanding.", "Instructor"},
			{"Message everyone who has not logged in for a week", "Short, kind, one question.", "Community lead"},
		},
	},
	"hospitality": {
		key: "hospitality",
		roles: []roleSpec{
			{"Reservations lead", "Reservations", "🛎️", "You handle bookings for {{company}}. {focus} Confirm details, note allergies and accessibility needs, and never overbook."},
			{"Guest experience lead", "Guest experience", "⭐", "You look after guests at {{company}}. {focus} Reply fast, fix what you can, and take complaints offline."},
			{"Marketing lead", "Marketing", "📣", "You fill the calendar for {{company}}. {focus} Write offers for {{audience}} that we can actually deliver."},
		},
		tasks: []taskSpec{
			{"Confirm this week's bookings and special requests", "One message per booking with the details we have.", "Reservations lead"},
			{"Reply to last month's reviews", "Specific, warm, no excuses.", "Guest experience lead"},
			{"Write a quiet-season offer", "One clear reason to book, a real deadline, no fake scarcity.", "Marketing lead"},
		},
	},
	"legal": {
		key: "legal",
		roles: []roleSpec{
			{"Client intake", "Intake", "📞", "You handle new enquiries for {{company}}. {focus} Be kind, capture the facts, and never promise an outcome. Route urgent matters to a person."},
			{"Case manager", "Operations", "🗂️", "You keep matters moving at {{company}}. {focus} Track deadlines, chase documents and flag anything that needs a lawyer."},
			{"Compliance checker", "Compliance", "✅", "You check work for {{company}}. {focus} Flag missing signatures, dates and conflicts. You do not give legal advice."},
		},
		tasks: []taskSpec{
			{"Follow up every new enquiry from this week", "One short message each: what happens next and what we need.", "Client intake"},
			{"Review open matters for missing documents", "List the matter, the missing item and the deadline.", "Case manager"},
			{"Run the conflict and deadline check", "Flag anything that needs a human before close of business.", "Compliance checker"},
		},
	},
	"nonprofit": {
		key: "nonprofit",
		roles: []roleSpec{
			{"Programs lead", "Programs", "🌱", "You run programmes for {{company}}. {focus} Keep the plan small and measurable, and report what actually happened."},
			{"Volunteer coordinator", "Volunteers", "🤝", "You look after volunteers for {{company}}. {focus} Be clear about time, tasks and impact, and thank people properly."},
			{"Donor care lead", "Fundraising", "💛", "You keep donors close to {{company}}. {focus} Report impact honestly, never exaggerate, and make giving easy."},
		},
		tasks: []taskSpec{
			{"Write this month's volunteer call-out", "What, when, where, who it helps, and how to sign up.", "Volunteer coordinator"},
			{"Draft the donor update", "One story, one number, one ask.", "Donor care lead"},
			{"Review the programme plan against what happened", "What worked, what did not, what changes next month.", "Programs lead"},
		},
	},
}

var seeds = []businessSeed{
	// Retail & e-commerce
	{"online-store", "Online store", "retail", "🛍️", "Sell physical products online.", "We sell physical products online and ship to customers ourselves.", []string{"Shopify", "Stripe", "Gorgias"}},
	{"dropshipping", "Dropshipping store", "retail", "📦", "Sell without holding stock.", "We sell products that suppliers ship directly to our customers.", []string{"Shopify", "AutoDS", "Stripe"}},
	{"print-on-demand", "Print-on-demand brand", "retail", "👕", "Custom products printed per order.", "We sell custom-printed apparel and gifts made only after an order.", []string{"Printful", "Etsy", "Shopify"}},
	{"subscription-box", "Subscription box", "retail", "🎁", "Recurring boxes delivered monthly.", "We ship a themed box to subscribers every month and must keep churn low.", []string{"Stripe", "Recharge", "Shopify"}},
	{"handmade", "Handmade shop", "retail", "🧶", "Small-batch handmade goods.", "We make small-batch goods by hand, so lead times and stock are limited.", []string{"Etsy", "Instagram", "Stripe"}},
	{"fashion-brand", "Fashion brand", "retail", "👗", "Clothing and accessories.", "We design and sell clothing, with seasonal drops and size exchanges.", []string{"Shopify", "Klaviyo", "Instagram"}},
	{"beauty-brand", "Beauty brand", "retail", "💄", "Skincare and cosmetics.", "We sell skincare and cosmetics with strict ingredient and returns questions.", []string{"Shopify", "TikTok Shop", "Klaviyo"}},
	{"home-goods", "Home goods store", "retail", "🪑", "Furniture and homeware.", "We sell furniture and homeware, where delivery damage and returns matter.", []string{"Shopify", "ShipStation", "Stripe"}},

	// Software & apps
	{"saas", "SaaS product", "software", "🧩", "Recurring software subscriptions.", "We run a subscription software product with trials, churn and support tickets.", []string{"Stripe", "GitHub", "Intercom"}},
	{"mobile-app", "Mobile app", "software", "📱", "iOS and Android app.", "We ship a mobile app and depend on store reviews and crash reports.", []string{"App Store Connect", "Firebase", "Sentry"}},
	{"ai-tool", "AI tool", "software", "🤖", "An AI-powered product.", "We sell an AI tool, so usage cost, model quality and trust are central.", []string{"OpenAI", "Stripe", "PostHog"}},
	{"marketplace", "Marketplace", "software", "🏪", "Buyers and sellers in one place.", "We run a two-sided marketplace and must keep both buyers and sellers happy.", []string{"Stripe Connect", "Algolia", "Intercom"}},
	{"dev-agency", "Software studio", "software", "🛠️", "Build software for clients.", "We build software for clients on fixed scopes and deadlines.", []string{"GitHub", "Linear", "Slack"}},
	{"no-code-studio", "No-code studio", "software", "🧱", "Automation and no-code builds.", "We build automations and internal tools for small teams.", []string{"Airtable", "Zapier", "Make"}},

	// Services & agencies
	{"marketing-agency", "Marketing agency", "agency", "📣", "Campaigns and content for clients.", "We run campaigns and content for clients who expect measurable results.", []string{"Meta Ads", "Google Ads", "Slack"}},
	{"design-studio", "Design studio", "agency", "🎨", "Brand and product design.", "We deliver brand and product design on tight review cycles.", []string{"Figma", "Notion", "Slack"}},
	{"seo-agency", "SEO agency", "agency", "🔍", "Search visibility for clients.", "We improve search visibility and must explain slow, compounding results.", []string{"Ahrefs", "Google Search Console", "Notion"}},
	{"consulting", "Consulting practice", "agency", "🧠", "Advice for a niche.", "We sell expert advice and need crisp proposals and follow-ups.", []string{"Notion", "Calendly", "Stripe"}},
	{"recruiting", "Recruiting firm", "agency", "🧑‍💼", "Place candidates with clients.", "We place candidates and live on follow-ups, references and clear role briefs.", []string{"LinkedIn", "Greenhouse", "Gmail"}},

	// Content & media
	{"newsletter", "Newsletter", "content", "📬", "A paid or free email publication.", "We publish a newsletter and need consistent issues and subscriber growth.", []string{"Beehiiv", "Stripe", "Substack"}},
	{"youtube", "YouTube channel", "content", "▶️", "Video content on YouTube.", "We publish videos and care about retention, thumbnails and sponsorship.", []string{"YouTube Studio", "Notion", "Patreon"}},
	{"podcast", "Podcast", "content", "🎙️", "An interview or narrative show.", "We publish a podcast and repurpose every episode into clips and posts.", []string{"Riverside", "Spotify", "Notion"}},
	{"social-media", "Social media brand", "content", "📲", "Short-form social content.", "We post daily short-form content and must keep a steady, on-brand cadence.", []string{"Instagram", "TikTok", "Buffer"}},
	{"photography", "Photography business", "content", "📷", "Shoots and edited galleries.", "We shoot events and portraits and need fast client communication.", []string{"Lightroom", "Pixieset", "Instagram"}},

	// Health & wellness
	{"gym", "Gym or studio", "health", "🏋️", "Memberships and classes.", "We run a gym with memberships, class schedules and retention challenges.", []string{"Mindbody", "Stripe", "Instagram"}},
	{"therapy-practice", "Therapy practice", "health", "🛋️", "Talk therapy sessions.", "We run a therapy practice with strict privacy and scheduling needs.", []string{"SimplePractice", "Google Calendar", "Stripe"}},
	{"dental-clinic", "Dental clinic", "health", "🦷", "Appointments and follow-ups.", "We run a dental clinic with recalls, insurance questions and no-show risk.", []string{"Open Dental", "Twilio", "Google Calendar"}},
	{"nutrition-coach", "Nutrition coaching", "health", "🥗", "Habit-based coaching.", "We coach nutrition habits and must avoid medical claims.", []string{"Practice Better", "Stripe", "Instagram"}},

	// Trades & local services
	{"plumbing", "Plumbing company", "trades", "🔧", "Emergency and scheduled plumbing.", "We handle plumbing call-outs, where speed and clear quotes win the job.", []string{"Jobber", "Google Business Profile", "Twilio"}},
	{"roofing", "Roofing company", "trades", "🏠", "Roof repair and replacement.", "We quote roofing work and need strong follow-up on big-ticket jobs.", []string{"Jobber", "Google Ads", "CompanyCam"}},
	{"landscaping", "Landscaping company", "trades", "🌿", "Garden and yard work.", "We maintain gardens on recurring schedules and upsell seasonal work.", []string{"Jobber", "Google Business Profile", "Instagram"}},
	{"cleaning", "Cleaning company", "trades", "🧽", "Recurring home and office cleans.", "We run recurring cleans and rely on reliability and reviews.", []string{"Jobber", "Stripe", "Google Business Profile"}},
	{"auto-repair", "Auto repair shop", "trades", "🚗", "Service and repairs.", "We service vehicles and need approvals before extra work.", []string{"Shopmonkey", "Google Business Profile", "Twilio"}},
	{"solar", "Solar installer", "trades", "☀️", "Solar systems and quotes.", "We sell solar systems with long consideration cycles and site visits.", []string{"HubSpot", "Aurora", "Google Ads"}},

	// Finance & professional
	{"bookkeeping", "Bookkeeping firm", "finance", "📒", "Monthly books for small businesses.", "We keep monthly books and chase receipts every month.", []string{"QuickBooks", "Xero", "Gmail"}},
	{"accounting", "Accounting firm", "finance", "🧾", "Tax and accounting services.", "We handle tax and accounting deadlines where missing documents cost money.", []string{"Xero", "Dext", "Gmail"}},
	{"insurance", "Insurance broker", "finance", "🛡️", "Policies and renewals.", "We broker insurance and must track renewals and claims carefully.", []string{"HubSpot", "Applied Epic", "Gmail"}},
	{"real-estate", "Real estate agency", "finance", "🏡", "Listings and viewings.", "We sell property and need fast lead response and viewing follow-ups.", []string{"Follow Up Boss", "Zillow", "Google Calendar"}},

	// Education
	{"tutoring", "Tutoring business", "education", "📖", "One-to-one tutoring.", "We tutor students and coordinate parents, schedules and progress.", []string{"Calendly", "Stripe", "Google Classroom"}},
	{"online-course", "Online course", "education", "🎓", "Recorded courses.", "We sell recorded courses and care about completion and testimonials.", []string{"Teachable", "Stripe", "Kajabi"}},
	{"language-school", "Language school", "education", "🗣️", "Language classes.", "We run language classes with levels, trials and re-enrolment.", []string{"Calendly", "Stripe", "WhatsApp"}},

	// Hospitality & travel
	{"restaurant", "Restaurant", "hospitality", "🍽️", "Dining and reservations.", "We run a restaurant with reservations, reviews and a changing menu.", []string{"OpenTable", "Google Business Profile", "Instagram"}},
	{"cafe", "Cafe", "hospitality", "☕", "Coffee and light food.", "We run a cafe where local reviews and repeat visits matter most.", []string{"Square", "Google Business Profile", "Instagram"}},
	{"bakery", "Bakery", "hospitality", "🥐", "Baked goods and custom orders.", "We bake daily and take custom orders with tight pickup windows.", []string{"Square", "Instagram", "Google Business Profile"}},
	{"hotel", "Small hotel", "hospitality", "🏨", "Rooms and guest stays.", "We run a small hotel with direct bookings and OTA competition.", []string{"Cloudbeds", "Stripe", "Booking.com"}},
	{"travel-agency", "Travel agency", "hospitality", "✈️", "Trip planning and bookings.", "We plan trips and must handle changes, documents and nervous travellers.", []string{"Travefy", "Stripe", "WhatsApp"}},

	// Nonprofit & community
	{"charity", "Charity", "nonprofit", "💛", "A registered nonprofit.", "We run programmes funded by donors and volunteers.", []string{"Donorbox", "Mailchimp", "Google Workspace"}},
	{"church", "Church or community group", "nonprofit", "⛪", "A local congregation.", "We run a congregation with services, volunteers and pastoral care.", []string{"Planning Center", "Mailchimp", "Google Calendar"}},
	{"coaching", "Coaching practice", "nonprofit", "🧭", "One-to-one coaching.", "We coach individuals and need clear session follow-ups and boundaries.", []string{"Calendly", "Stripe", "Notion"}},
	{"amazon-fba", "Amazon FBA business", "retail", "📦", "Sell on Amazon with FBA.", "We sell on Amazon and rely on FBA stock levels, listing quality and reviews.", []string{"Amazon Seller Central", "Helium 10", "Stripe"}},
	{"veterinary", "Veterinary clinic", "health", "🐾", "Animal care and appointments.", "We run a veterinary clinic with appointments, reminders and nervous pet owners.", []string{"Vetstoria", "Twilio", "Google Calendar"}},
	{"pet-grooming", "Pet grooming", "trades", "✂️", "Grooming appointments.", "We groom pets by appointment and live on repeat visits and reviews.", []string{"Square", "Google Business Profile", "Instagram"}},
	{"wedding-planner", "Wedding planner", "hospitality", "💍", "Plan weddings end to end.", "We plan weddings with many suppliers, tight timelines and emotional clients.", []string{"Aisle Planner", "Pinterest", "Stripe"}},
	{"event-planner", "Event planner", "hospitality", "🎪", "Conferences and private events.", "We plan events with venues, suppliers and guest lists that change constantly.", []string{"Eventbrite", "Notion", "Stripe"}},
	{"videographer", "Videography business", "content", "🎥", "Films and edited videos.", "We shoot and edit videos, and clients expect fast previews and clear revisions.", []string{"Frame.io", "Vimeo", "Notion"}},
	{"music-school", "Music school", "education", "🎹", "Lessons for instruments.", "We teach music lessons with weekly schedules, recitals and parent updates.", []string{"Calendly", "Stripe", "WhatsApp"}},
	{"driving-school", "Driving school", "education", "🚦", "Driving lessons and tests.", "We teach driving lessons with test dates, cancellations and nervous learners.", []string{"Calendly", "Stripe", "Google Business Profile"}},
	{"home-care", "Home care agency", "health", "🏠", "Care workers in people's homes.", "We place care workers in homes and must coordinate families, rotas and safeguarding.", []string{"Birdie", "Twilio", "Google Calendar"}},
	{"daycare", "Daycare", "education", "🧸", "Childcare for working parents.", "We run a daycare with ratios, pickups, allergies and worried parents.", []string{"Brightwheel", "Stripe", "WhatsApp"}},
	{"mortgage-broker", "Mortgage broker", "finance", "🏦", "Mortgage advice and applications.", "We broker mortgages with document-heavy applications and rate changes.", []string{"HubSpot", "DocuSign", "Gmail"}},
	{"hr-consultancy", "HR consultancy", "agency", "🧑‍🤝‍🧑", "HR advice for small firms.", "We advise small firms on hiring, policies and difficult conversations.", []string{"Notion", "DocuSign", "Slack"}},
	{"translation-service", "Translation service", "agency", "🌍", "Translation and localisation.", "We translate and localise content with tight deadlines and sensitive terminology.", []string{"Crowdin", "DeepL", "Stripe"}},
	{"lead-gen-agency", "Lead generation agency", "agency", "🎯", "Qualified leads for clients.", "We generate leads for clients and are judged on volume and quality.", []string{"Apollo", "HubSpot", "Instantly"}},
	{"automation-agency", "Automation agency", "software", "⚙️", "Automate client operations.", "We automate client operations with no-code and AI, and must document every workflow.", []string{"Make", "Zapier", "Airtable"}},
	{"law-firm", "Law firm", "legal", "⚖️", "Legal matters and clients.", "We run a law firm with enquiries, deadlines, documents and strict confidentiality.", []string{"Clio", "DocuSign", "Gmail"}},
	{"immigration-consultant", "Immigration consultant", "legal", "🛂", "Visa and residency advice.", "We help people with visas, where documents and deadlines decide the outcome.", []string{"Clio", "DocuSign", "WhatsApp"}},
}

var all []Business

func init() {
	all = make([]Business, 0, len(seeds))

	for _, seed := range seeds {
		cat := categories[seed.category]

		agents := make([]AgentPreset, 0, len(cat.roles))
		for _, role := range cat.roles {
			agents = append(agents, AgentPreset{
				Name:         role.name,
				Role:         role.role,
				Emoji:        role.emoji,
				Instructions: strings.ReplaceAll(role.instructions, "{focus}", seed.focus),
			})
		}

		tasks := make([]TaskPreset, 0, len(cat.tasks))
		for _, task := range cat.tasks {
			tasks = append(tasks, TaskPreset{
				Title:   task.title,
				Details: task.details,
				Agent:   task.agent,
			})
		}

		all = append(all, Business{
			Key:          seed.key,
			Name:         seed.name,
			Category:     categoryLabel(seed.category),
			Emoji:        seed.emoji,
			Blurb:        seed.blurb,
			Focus:        seed.focus,
			Integrations: seed.integrations,
			Agents:       agents,
			Tasks:        tasks,
		})
	}
}

func All() []Business { return all }

func ByKey(key string) (Business, bool) {
	if key == "custom" {
		return CustomTeam(), true
	}

	for _, business := range all {
		if business.Key == key {
			return business, true
		}
	}
	return Business{}, false
}

func Categories() []string {
	seen := map[string]bool{}
	out := []string{}
	for _, business := range all {
		if !seen[business.Category] {
			seen[business.Category] = true
			out = append(out, business.Category)
		}
	}
	return out
}

func categoryLabel(key string) string {
	switch key {
	case "retail":
		return "Retail & e-commerce"
	case "software":
		return "Software & apps"
	case "agency":
		return "Services & agencies"
	case "content":
		return "Content & media"
	case "health":
		return "Health & wellness"
	case "trades":
		return "Trades & local services"
	case "finance":
		return "Finance & professional"
	case "education":
		return "Education"
	case "hospitality":
		return "Hospitality & travel"
	case "nonprofit":
		return "Nonprofit & community"
	case "legal":
		return "Legal services"
	default:
		return key
	}
}

// CustomTeam is the fallback when someone runs something we did not anticipate.
func CustomTeam() Business {
	return Business{
		Key:      "custom",
		Name:     "Something else",
		Category: "Custom",
		Emoji:    "✨",
		Blurb:    "Describe it in your own words.",
		Focus:    "We are building this business from scratch.",
		Agents: []AgentPreset{
			{"General manager", "Operations", "🧭", "You are the general manager for {{company}}. {focus} Keep the week organised, surface decisions, and turn vague goals into next actions."},
			{"Customer lead", "Support", "🛟", "You look after customers for {{company}} in a {{tone}} tone. {focus} Reply fast, solve what you can, and escalate the rest."},
			{"Growth lead", "Marketing", "📣", "You find customers for {{company}}. {focus} Write clear offers for {{audience}} and test one channel at a time."},
		},
		Tasks: []TaskPreset{
			{"Write down what we sell and who it is for", "One paragraph, no jargon. This becomes the brief every helper reads.", "General manager"},
			{"List the five questions customers ask most", "Then draft a short answer to each in our voice.", "Customer lead"},
			{"Pick one channel and plan two weeks of posts", "Small, specific, doable without a designer.", "Growth lead"},
		},
	}
}

type Vars struct {
	Company  string
	Website  string
	Support  string
	Audience string
	Tone     string
}

func Render(text string, vars Vars) string {
	audience := strings.TrimSpace(vars.Audience)
	if audience == "" {
		audience = "our customers"
	}
	tone := strings.TrimSpace(vars.Tone)
	if tone == "" {
		tone = "friendly and clear"
	}

	replacer := strings.NewReplacer(
		"{{company}}", strings.TrimSpace(vars.Company),
		"{{website}}", strings.TrimSpace(vars.Website),
		"{{support_email}}", strings.TrimSpace(vars.Support),
		"{{audience}}", audience,
		"{{tone}}", tone,
	)

	return replacer.Replace(text)
}
