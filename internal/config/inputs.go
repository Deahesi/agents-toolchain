package config

import (
	"strconv"

	"github.com/Deahesi/agents-toolchain/internal/domain"
)

func (s *ConfigService) getProvider(defaultAgent *domain.AgentConfig) string {
	providers := make([]string, len(domain.Providers))
	for i, provider := range domain.Providers {
		providers[i] = string(provider)
	}

	for {
		res, err := s.ui.InteractiveSelect("Choose LLM Provider", providers...)

		if err != nil {
			s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.Provider)
			return defaultAgent.Agent.Provider
		}

		err = domain.ValidateProvider(res)
		if err != nil {
			s.ui.LogError("Invalid provider: ", err)
			continue
		}

		return res
	}

}

func (s *ConfigService) getModel(defaultAgent *domain.AgentConfig) string {
	for {
		res, err := s.ui.TextInput("Write Agent model")

		if err != nil {
			s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.Model)
			return defaultAgent.Agent.Model
		}

		err = domain.ValidateModel(res)
		if err != nil {
			s.ui.LogError("Invalid model: ", err)
			continue
		}

		return res
	}
}

func (s *ConfigService) getDescription(defaultAgent *domain.AgentConfig) string {
	res, err := s.ui.TextInputMultiline("Write Agent Description")

	if err != nil {
		s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.Description)
		return defaultAgent.Agent.Description
	}

	return res
}

func (s *ConfigService) getTemperature(defaultAgent *domain.AgentConfig) float64 {
	for {
		res, err := s.ui.TextInput("Write Agent temperatre (number from 0.0 to 2.0)")

		if err != nil {
			s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.Temperature)
			return defaultAgent.Agent.Temperature
		}

		fRes, err := strconv.ParseFloat(res, 64)
		if err != nil {
			s.ui.LogError("Invalid temperature: ", err)
			continue
		}

		err = domain.ValidateTemperature(fRes)
		if err != nil {
			s.ui.LogError("Invalid temperature: ", err)
			continue
		}

		return fRes
	}
}

func (s *ConfigService) getSystemPrompt(defaultAgent *domain.AgentConfig) string {
	for {
		res, err := s.ui.TextInputMultiline("Write System Prompt")

		if err != nil {
			s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.SystemPrompt)
			return defaultAgent.Agent.SystemPrompt
		}

		err = domain.ValidateSystemPrompt(res)
		if err != nil {
			s.ui.LogError("Invalid system prompt: ", err)
			continue
		}

		return res
	}
}
