package config

import (
	"fmt"
	"strconv"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/invopop/validation"
)

func checkDefaultExist(val any) bool {
	val, isNil := validation.Indirect(val)
	if isNil {
		return false
	}
	if text, ok := val.(string); ok {
		return text != ""
	}
	return true
}

func addDefaultIfExist(label string, val any) string {
	if checkDefaultExist(val) {
		val, _ = validation.Indirect(val)
		return fmt.Sprintf("%s (default: %v)", label, val)
	}

	return label
}

func (s *ConfigService) getProvider(defaultAgent *domain.AgentConfig) string {
	providers := make([]string, len(domain.Providers))
	for i, provider := range domain.Providers {
		providers[i] = string(provider)
	}

	for {
		res, err := s.ui.InteractiveSelect(addDefaultIfExist("Choose LLM Provider", defaultAgent.Agent.Provider), providers...)

		if err != nil {
			s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.Provider)
			return defaultAgent.Agent.Provider
		}
		if res == "" && checkDefaultExist(defaultAgent.Agent.Provider) {
			s.ui.LogStep("Set defaults: ", defaultAgent.Agent.Provider)
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
		res, err := s.ui.TextInput(addDefaultIfExist("Write Agent model", defaultAgent.Agent.Model))

		if err != nil {
			s.ui.LogStep("Set defaults: ", defaultAgent.Agent.Model)
			return defaultAgent.Agent.Model
		}
		if res == "" && checkDefaultExist(defaultAgent.Agent.Model) {
			s.ui.LogStep("Set defaults: ", defaultAgent.Agent.Model)
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
	res, err := s.ui.TextInputMultiline(addDefaultIfExist("Write Agent Description", defaultAgent.Agent.Description))

	if err != nil {
		s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.Description)
		return defaultAgent.Agent.Description
	}
	if res == "" && checkDefaultExist(defaultAgent.Agent.Description) {
		s.ui.LogStep("Set defaults: ", defaultAgent.Agent.Description)
		return defaultAgent.Agent.Description
	}

	return res
}

func (s *ConfigService) getTemperature(defaultAgent *domain.AgentConfig) float64 {
	for {
		res, err := s.ui.TextInput(addDefaultIfExist("Write Agent temperatre (number from 0.0 to 2.0)", defaultAgent.Agent.Temperature))

		if err != nil {
			s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.Temperature)
			return defaultAgent.Agent.Temperature
		}
		if res == "" && checkDefaultExist(defaultAgent.Agent.Temperature) {
			s.ui.LogStep("Set defaults: ", defaultAgent.Agent.Temperature)
			return defaultAgent.Agent.Temperature
		}

		fRes, err := strconv.ParseFloat(res, 64)
		if err != nil {
			s.ui.LogError("Invalid temperature")
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
		res, err := s.ui.TextInputMultiline(addDefaultIfExist("Write System Prompt", defaultAgent.Agent.SystemPrompt))

		if err != nil {
			s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.SystemPrompt)
			return defaultAgent.Agent.SystemPrompt
		}
		if res == "" && checkDefaultExist(defaultAgent.Agent.SystemPrompt) {
			s.ui.LogStep("Set defaults: ", defaultAgent.Agent.SystemPrompt)
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

func (s *ConfigService) GetMaxOutputTokens(defaultAgent *domain.AgentConfig) *int {
	for {
		res, err := s.ui.TextInput(addDefaultIfExist("Write max output tokens", defaultAgent.Agent.MaxOutputTokens))

		if err != nil {
			s.ui.LogStep("No output tokens limit")
			return nil
		}
		if res == "" {
			if checkDefaultExist(defaultAgent.Agent.MaxOutputTokens) {
				s.ui.LogStep("Set defaults: ", defaultAgent.Agent.MaxOutputTokens)
				return defaultAgent.Agent.MaxOutputTokens
			}
			s.ui.LogStep("No output tokens limit")
			return nil
		}

		n, err := strconv.Atoi(res)
		if err != nil {
			s.ui.LogError("Invalid number")
			continue
		}
		if n <= 0 {
			s.ui.LogError("Invalid number: must be greater than zero")
			continue
		}

		return &n
	}
}

func (s *ConfigService) getWorkDir(defaultAgent *domain.AgentConfig) string {
	for {
		res, err := s.ui.TextInput(addDefaultIfExist("Write work directory", defaultAgent.Agent.WorkDir))

		if err != nil {
			s.ui.LogError("Input error. Set defaults: ", defaultAgent.Agent.WorkDir)
			return defaultAgent.Agent.WorkDir
		}
		if res == "" && checkDefaultExist(defaultAgent.Agent.WorkDir) {
			s.ui.LogStep("Set defaults: ", defaultAgent.Agent.WorkDir)
			return defaultAgent.Agent.WorkDir
		}

		err = domain.ValidateWorkDir(res)
		if err != nil {
			s.ui.LogError("Invalid work dir: ", err)
			continue
		}

		return res
	}
}
