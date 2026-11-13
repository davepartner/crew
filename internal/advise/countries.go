package advise

import "strings"

type CountryProfile struct {
	Name         string   `json:"name"`
	Currency     string   `json:"currency"`
	Registration []string `json:"registration"`
	Tax          []string `json:"tax"`
	Payments     []string `json:"payments"`
	Ads          []string `json:"ads"`
	Data         []string `json:"data"`
	Notes        []string `json:"notes"`
}

var countries = map[string]CountryProfile{
	"nigeria": {
		Name:     "Nigeria",
		Currency: "NGN (₦)",
		Registration: []string{
			"Register a business name or limited company with the CAC.",
			"Get a TIN and open a corporate bank account.",
			"Keep a written contract, invoice and receipt for every client.",
		},
		Tax: []string{
			"Confirm VAT (7.5%) and annual return obligations with an accountant.",
			"Separate business and personal money from day one.",
		},
		Payments: []string{
			"Use Paystack or Flutterwave payment links for local clients.",
			"Take 50% upfront; monthly retainers are paid in advance.",
			"For foreign clients, use a compliant USD account (Payoneer, Wise, Grey or similar).",
		},
		Ads: []string{
			"Instagram, WhatsApp and Meta are the default local channels.",
			"Follow ARCON advertising rules; no fake testimonials or guaranteed income claims.",
			"Track every lead in a sheet or CRM, not only in WhatsApp chats.",
		},
		Data: []string{
			"Follow the Nigeria Data Protection Act; publish a privacy policy and keep customer data minimal.",
		},
		Notes: []string{
			"Price in Naira for local clients and review for FX quarterly.",
			"Power and internet are delivery risks: schedule backups and offline drafts.",
		},
	},
	"kenya": {
		Name: "Kenya", Currency: "KES (KSh)",
		Registration: []string{"Register with the Business Registration Service (eCitizen).", "Get a KRA PIN and open a business bank account."},
		Tax:          []string{"Confirm VAT and turnover tax thresholds with a KRA-registered accountant."},
		Payments:     []string{"Use M-Pesa Paybill/Till and a payment link for cards.", "Take a deposit before delivery."},
		Ads:          []string{"WhatsApp, Instagram and TikTok are strong; track leads outside DMs."},
		Data:         []string{"Follow the Data Protection Act 2019; register with the ODPC if required."},
	},
	"ghana": {
		Name: "Ghana", Currency: "GHS (₵)",
		Registration: []string{"Register with the Office of the Registrar of Companies.", "Get a TIN and open a business account."},
		Tax:          []string{"Confirm VAT, levies and annual returns with a local accountant."},
		Payments:     []string{"Use Mobile Money and Paystack/Flutterwave links.", "Invoice with clear payment terms."},
		Ads:          []string{"WhatsApp, Instagram and Facebook lead locally; keep a lead list."},
		Data:         []string{"Follow the Data Protection Act 2012."},
	},
	"south africa": {
		Name: "South Africa", Currency: "ZAR (R)",
		Registration: []string{"Register with CIPC; get a tax number from SARS.", "Open a business bank account."},
		Tax:          []string{"Confirm VAT registration and provisional tax with an accountant."},
		Payments:     []string{"Use PayFast, Yoco or a business bank payment link.", "Deposits before production."},
		Ads:          []string{"Meta, Google and WhatsApp work; respect the Consumer Protection Act."},
		Data:         []string{"Follow POPIA; keep a privacy policy and consent records."},
	},
	"united kingdom": {
		Name: "United Kingdom", Currency: "GBP (£)",
		Registration: []string{"Register with Companies House; register for Self Assessment or Corporation Tax.", "Open a business bank account."},
		Tax:          []string{"Confirm VAT threshold and Making Tax Digital with an accountant."},
		Payments:     []string{"Use Stripe or GoCardless; invoice with VAT rules where applicable."},
		Ads:          []string{"Meta, Google and TikTok; follow ASA/CAP advertising rules."},
		Data:         []string{"Follow UK GDPR and PECR; keep a privacy policy and lawful basis."},
	},
	"united states": {
		Name: "United States", Currency: "USD ($)",
		Registration: []string{"Form an LLC or corporation in your state; get an EIN.", "Open a business bank account."},
		Tax:          []string{"Confirm federal, state and sales-tax obligations with a CPA."},
		Payments:     []string{"Use Stripe, PayPal or a business bank; invoice with clear terms."},
		Ads:          []string{"Meta, Google, TikTok and YouTube; follow FTC endorsement and claim rules."},
		Data:         []string{"Follow applicable state privacy laws; publish a privacy policy."},
	},
	"canada": {
		Name: "Canada", Currency: "CAD ($)",
		Registration: []string{"Incorporate or register a sole proprietorship; get a BN.", "Open a business account."},
		Tax:          []string{"Confirm GST/HST and payroll obligations with an accountant."},
		Payments:     []string{"Use Stripe, Moneris or Interac e-Transfer."},
		Ads:          []string{"Meta, Google and TikTok; follow Competition Bureau and ad standards."},
		Data:         []string{"Follow PIPEDA and provincial privacy laws."},
	},
	"india": {
		Name: "India", Currency: "INR (₹)",
		Registration: []string{"Register a company/LLP with MCA; get GST and PAN.", "Open a current account."},
		Tax:          []string{"Confirm GST, TDS and annual filings with a CA."},
		Payments:     []string{"Use Razorpay/UPI and a business bank; invoice with GST details."},
		Ads:          []string{"Meta, Google and WhatsApp; follow ASCI advertising standards."},
		Data:         []string{"Follow the DPDP Act and sector rules."},
	},
	"united arab emirates": {
		Name: "United Arab Emirates", Currency: "AED (د.إ)",
		Registration: []string{"Choose a free zone or mainland licence; get a trade licence.", "Open a corporate bank account."},
		Tax:          []string{"Confirm corporate tax and VAT registration with an accountant."},
		Payments:     []string{"Use Stripe, Network International or bank transfer."},
		Ads:          []string{"Meta, Google and TikTok; follow UAE advertising rules."},
		Data:         []string{"Follow UAE data protection law and free-zone rules."},
	},
}

func Country(name string) CountryProfile {
	key := strings.ToLower(strings.TrimSpace(name))
	if profile, ok := countries[key]; ok {
		return profile
	}
	if strings.Contains(key, "nigeria") {
		return countries["nigeria"]
	}

	return CountryProfile{
		Name:     strings.TrimSpace(name),
		Currency: "local currency",
		Registration: []string{
			"Register the business with the local authority and get a tax ID.",
			"Open a business bank account and keep business money separate.",
		},
		Tax:      []string{"Confirm local VAT/sales tax and annual filing with an accountant."},
		Payments: []string{"Use a local payment provider and take a deposit before delivery."},
		Ads:      []string{"Start with the channel where your customers already spend time."},
		Data:     []string{"Follow local data protection rules and publish a privacy policy."},
	}
}

func Countries() []CountryProfile {
	names := []string{"Nigeria", "Kenya", "Ghana", "South Africa", "United Kingdom", "United States", "Canada", "India", "United Arab Emirates"}
	out := make([]CountryProfile, 0, len(names))
	for _, name := range names {
		out = append(out, countries[strings.ToLower(name)])
	}
	return out
}
