package modelcatalog

// Provider describes an AI model provider Crew can use. "Kind" decides how the
// worker talks to it: openai-compatible, anthropic, google, or local ollama.
type Provider struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	BaseURL      string   `json:"base_url"`
	KeysURL      string   `json:"keys_url"`
	DocsURL      string   `json:"docs_url"`
	Free         bool     `json:"free"`
	FreeNote     string   `json:"free_note"`
	DefaultModel string   `json:"default_model"`
	Models       []string `json:"models"`
}

var providers = []Provider{
	{
		Key: "openrouter", Name: "OpenRouter", Kind: "openai",
		BaseURL: "https://openrouter.ai/api/v1",
		KeysURL: "https://openrouter.ai/keys", DocsURL: "https://openrouter.ai/docs",
		Free: true, FreeNote: "Free models are marked :free and need no card.",
		DefaultModel: "meta-llama/llama-3.3-70b-instruct:free",
		Models: []string{
			"meta-llama/llama-3.3-70b-instruct:free",
			"deepseek/deepseek-chat-v3-0324:free",
			"google/gemini-2.0-flash-exp:free",
			"anthropic/claude-3.5-haiku",
			"openai/gpt-4o-mini",
		},
	},
	{
		Key: "groq", Name: "Groq", Kind: "openai",
		BaseURL: "https://api.groq.com/openai/v1",
		KeysURL: "https://console.groq.com/keys", DocsURL: "https://console.groq.com/docs",
		Free: true, FreeNote: "Generous free tier with very fast responses.",
		DefaultModel: "llama-3.3-70b-versatile",
		Models:       []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant"},
	},
	{
		Key: "google", Name: "Google Gemini", Kind: "google",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
		KeysURL: "https://aistudio.google.com/app/apikey", DocsURL: "https://ai.google.dev/gemini-api/docs",
		Free: true, FreeNote: "Free tier in AI Studio; limits reset daily.",
		DefaultModel: "gemini-2.5-flash",
		Models:       []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash"},
	},
	{
		Key: "mistral", Name: "Mistral", Kind: "openai",
		BaseURL: "https://api.mistral.ai/v1",
		KeysURL: "https://console.mistral.ai/api-keys", DocsURL: "https://docs.mistral.ai",
		Free: true, FreeNote: "Free experiment tier for small models.",
		DefaultModel: "mistral-small-latest",
		Models:       []string{"mistral-small-latest", "open-mistral-nemo"},
	},
	{
		Key: "cerebras", Name: "Cerebras", Kind: "openai",
		BaseURL: "https://api.cerebras.ai/v1",
		KeysURL: "https://cloud.cerebras.ai", DocsURL: "https://inference-docs.cerebras.ai",
		Free: true, FreeNote: "Free developer tier with fast inference.",
		DefaultModel: "llama-3.3-70b",
		Models:       []string{"llama-3.3-70b", "llama3.1-8b"},
	},
	{
		Key: "openai", Name: "OpenAI", Kind: "openai",
		BaseURL: "https://api.openai.com/v1",
		KeysURL: "https://platform.openai.com/api-keys", DocsURL: "https://platform.openai.com/docs",
		Free: false, FreeNote: "Pay as you go.",
		DefaultModel: "gpt-4o-mini",
		Models:       []string{"gpt-4o-mini", "gpt-4.1-mini", "o4-mini"},
	},
	{
		Key: "anthropic", Name: "Anthropic Claude", Kind: "anthropic",
		BaseURL: "https://api.anthropic.com/v1",
		KeysURL: "https://console.anthropic.com/settings/keys", DocsURL: "https://docs.anthropic.com",
		Free: false, FreeNote: "Pay as you go.",
		DefaultModel: "claude-sonnet-4-5",
		Models:       []string{"claude-sonnet-4-5", "claude-haiku-4-5"},
	},
	{
		Key: "deepseek", Name: "DeepSeek", Kind: "openai",
		BaseURL: "https://api.deepseek.com/v1",
		KeysURL: "https://platform.deepseek.com/api_keys", DocsURL: "https://api-docs.deepseek.com",
		Free: false, FreeNote: "Low cost per token.",
		DefaultModel: "deepseek-chat",
		Models:       []string{"deepseek-chat", "deepseek-reasoner"},
	},
	{
		Key: "xai", Name: "xAI Grok", Kind: "openai",
		BaseURL: "https://api.x.ai/v1",
		KeysURL: "https://console.x.ai", DocsURL: "https://docs.x.ai",
		Free: false, FreeNote: "Pay as you go.",
		DefaultModel: "grok-3-mini",
		Models:       []string{"grok-3-mini", "grok-3"},
	},
	{
		Key: "together", Name: "Together AI", Kind: "openai",
		BaseURL: "https://api.together.xyz/v1",
		KeysURL: "https://api.together.xyz/settings/api-keys", DocsURL: "https://docs.together.ai",
		Free: false, FreeNote: "Starter credit, then pay as you go.",
		DefaultModel: "meta-llama/Llama-3.3-70B-Instruct-Turbo",
		Models:       []string{"meta-llama/Llama-3.3-70B-Instruct-Turbo", "mistralai/Mixtral-8x7B-Instruct-v0.1"},
	},
	{
		Key: "ollama", Name: "Ollama (your computer)", Kind: "openai",
		BaseURL: "http://127.0.0.1:11434/v1",
		KeysURL: "https://ollama.com/download", DocsURL: "https://github.com/ollama/ollama/blob/main/docs/api.md",
		Free: true, FreeNote: "Runs locally. No key, no bill, needs a capable machine.",
		DefaultModel: "llama3.2",
		Models:       []string{"llama3.2", "qwen2.5", "mistral"},
	},
}

func All() []Provider { return providers }

func ByKey(key string) (Provider, bool) {
	for _, provider := range providers {
		if provider.Key == key {
			return provider, true
		}
	}
	return Provider{}, false
}

func Public() []Provider {
	out := make([]Provider, len(providers))
	copy(out, providers)
	return out
}

// Price is US dollars per one million tokens. It is an estimate for budgets,
// not an invoice; providers change prices and some plans are free.
type Price struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

var prices = map[string]Price{
	"deepseek/deepseek-chat":                           {0.27, 1.10},
	"deepseek/deepseek-reasoner":                       {0.55, 2.19},
	"openai/gpt-4o-mini":                               {0.15, 0.60},
	"openai/gpt-4.1-mini":                              {0.40, 1.60},
	"openai/o4-mini":                                   {1.10, 4.40},
	"anthropic/claude-sonnet-4-5":                      {3.00, 15.00},
	"anthropic/claude-haiku-4-5":                       {1.00, 5.00},
	"google/gemini-2.5-flash":                          {0.30, 2.50},
	"google/gemini-2.5-pro":                            {1.25, 10.00},
	"google/gemini-2.0-flash":                          {0.10, 0.40},
	"mistral/mistral-small-latest":                     {0.10, 0.30},
	"xai/grok-3-mini":                                  {0.30, 0.50},
	"together/meta-llama/Llama-3.3-70B-Instruct-Turbo": {0.88, 0.88},
}

func PriceFor(providerKey, model string) Price {
	if price, ok := prices[providerKey+"/"+model]; ok {
		return price
	}
	switch providerKey {
	case "openrouter", "groq", "cerebras", "ollama":
		return Price{}
	default:
		return Price{Input: 0.50, Output: 2.00}
	}
}

func EstimateCost(providerKey, model string, promptTokens, completionTokens int) float64 {
	price := PriceFor(providerKey, model)
	return (float64(promptTokens)*price.Input + float64(completionTokens)*price.Output) / 1_000_000
}
