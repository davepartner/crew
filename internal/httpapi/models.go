package httpapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/davepartner/crew/internal/llm"
	"github.com/davepartner/crew/internal/modelcatalog"
	"github.com/davepartner/crew/internal/store"
)

type resolvedModel struct {
	Provider modelcatalog.Provider
	Model    string
	APIKey   string
	Source   string // platform, user
}

func (s *Server) platformModel() (resolvedModel, bool) {
	if !s.cfg.PlatformEnabled() {
		return resolvedModel{}, false
	}

	provider, ok := modelcatalog.ByKey(s.cfg.PlatformProvider)
	if !ok || s.cfg.PlatformModel == "" {
		return resolvedModel{}, false
	}

	if s.cfg.PlatformBaseURL != "" {
		provider.BaseURL = s.cfg.PlatformBaseURL
	}

	return resolvedModel{
		Provider: provider,
		Model:    s.cfg.PlatformModel,
		APIKey:   s.cfg.PlatformAPIKey,
		Source:   "platform",
	}, true
}

// modelForAgent prefers the helper's own provider and key. A helper with no
// model selected uses the included platform model. A helper that explicitly
// picked a provider without a key is an error, not a silent platform charge.
func (s *Server) modelForAgent(ctx context.Context, agent store.Agent) (resolvedModel, error) {
	if agent.ModelProvider != "" && agent.ModelName != "" {
		provider, ok := modelcatalog.ByKey(agent.ModelProvider)
		if !ok {
			return resolvedModel{}, errors.New("that provider is not available")
		}

		if provider.Key == "ollama" {
			return resolvedModel{Provider: provider, Model: agent.ModelName, APIKey: "", Source: "user"}, nil
		}

		cipherText, err := s.store.ProviderKeyCipher(ctx, agent.UserID, provider.Key)
		if err != nil {
			// The included platform model is available without a personal key.
			if platform, ok := s.platformModel(); ok && platform.Provider.Key == provider.Key {
				platform.Model = agent.ModelName
				return platform, nil
			}
			return resolvedModel{}, errors.New("no key for that provider")
		}
		key, err := s.cipher.Decrypt(cipherText)
		if err != nil || key == "" {
			if platform, ok := s.platformModel(); ok && platform.Provider.Key == provider.Key {
				platform.Model = agent.ModelName
				return platform, nil
			}
			return resolvedModel{}, errors.New("could not read that provider key")
		}

		return resolvedModel{Provider: provider, Model: agent.ModelName, APIKey: key, Source: "user"}, nil
	}

	if platform, ok := s.platformModel(); ok {
		return platform, nil
	}

	return resolvedModel{}, errors.New("no model configured")
}

// checkBudget stops a call before it starts once the month's estimate is used
// up. It allows the call that crosses the line, then blocks the next one, which
// keeps a single answer from being cut in half.
func (s *Server) checkBudget(ctx context.Context, userID, companyID, agentID string) (string, error) {
	company, err := s.store.GetCompany(ctx, userID, companyID)
	if err != nil {
		return "", nil
	}

	month := store.MonthStart()

	if company.MonthlyBudgetUSD > 0 {
		spent, err := s.store.CompanySpend(ctx, companyID, month)
		if err != nil {
			return "", err
		}
		if spent >= company.MonthlyBudgetUSD {
			return fmt.Sprintf("This company has used its $%.2f monthly AI budget. Raise it in Settings → Company.", company.MonthlyBudgetUSD), nil
		}
	}

	if agentID != "" {
		agent, err := s.store.GetAgent(ctx, companyID, agentID)
		if err == nil && agent.MonthlyBudgetUSD > 0 {
			spent, err := s.store.AgentSpend(ctx, companyID, agentID, month)
			if err != nil {
				return "", err
			}
			if spent >= agent.MonthlyBudgetUSD {
				return fmt.Sprintf("%s has used their $%.2f monthly AI budget. Raise it on the helper page.", agent.Name, agent.MonthlyBudgetUSD), nil
			}
		}
	}

	return "", nil
}

func (s *Server) saveUsage(ctx context.Context, userID, companyID, agentID, taskID string, model resolvedModel, usage llm.Usage) float64 {
	cost := modelcatalog.EstimateCost(model.Provider.Key, model.Model, usage.PromptTokens, usage.CompletionTokens)

	_ = s.store.RecordUsage(ctx, store.UsageInput{
		UserID:           userID,
		CompanyID:        companyID,
		AgentID:          agentID,
		TaskID:           taskID,
		Provider:         model.Provider.Key,
		Model:            model.Model,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		CostUSD:          cost,
	})

	return cost
}
