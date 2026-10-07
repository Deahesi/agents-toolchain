package config

import "github.com/Deahesi/agents-toolchain/internal/domain"

func defaultAgent(name string) *domain.AgentConfig {
	return &domain.AgentConfig{
		Version: domain.ConfigVersion,
		Agent: domain.Agent{
			Name:         name,
			Description:  "Agent " + name,
			Provider:     "openai",
			Model:        "gpt-4o",
			Temperature:  0.2,
			SystemPrompt: "You are a helpful assistant. You answer clearly and to the point.\n",
			Memory: &domain.Memory{
				Type: "local_file",
				Path: "./.agent_history.json",
			},
			Tools: []domain.Tool{},
		},
	}
}
