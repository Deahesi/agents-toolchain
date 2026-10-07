package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Deahesi/agents-toolchain/internal/domain"
)

const (
	ProjectFile = "agents-toolchain.yml"
	AgentFile   = "agent.yml"
)

var (
	ErrNotInitialized = errors.New("project is not initialized; run agent-toolchain init")
	ErrInitialized    = errors.New("project config already exists")
	ErrAgentExists    = errors.New("agent directory already exists")
)

type ConfigService struct {
	workspaceDir string
	ui           domain.UI
}

func NewConfigService(ui domain.UI, workspaceDir string) *ConfigService {
	return &ConfigService{
		ui:           ui,
		workspaceDir: workspaceDir,
	}
}

func (s *ConfigService) Init(ctx context.Context, agentsDir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	spinner, err := s.ui.LogSpinner("Checking project configuration")
	if err != nil {
		return "", err
	}
	content, err := encode(&domain.ProjectConfig{
		AgentsDir: agentsDir,
	})
	if err != nil {
		spinner.Fail()
		return "", err
	}
	spinner.Success()

	root, err := os.OpenRoot(s.workspaceDir)
	if err != nil {
		return "", fmt.Errorf("open project: %w", err)
	}
	defer root.Close()

	if _, err := root.Stat(ProjectFile); err == nil {
		return "", ErrInitialized
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("check project config: %w", err)
	}

	spinner, err = s.ui.LogSpinner("Creating agents directory: ", agentsDir)
	if err != nil {
		return "", err
	}
	if err := root.MkdirAll(agentsDir, 0o755); err != nil {
		spinner.Fail()
		return "", fmt.Errorf("create agents directory: %w", err)
	}
	spinner.Success()

	spinner, err = s.ui.LogSpinner("Writing project config: ", ProjectFile)
	if err != nil {
		return "", err
	}
	if err := writeNewFile(root, ProjectFile, content); err != nil {
		spinner.Fail()

		if errors.Is(err, os.ErrExist) {
			return "", ErrInitialized
		}
		return "", fmt.Errorf("create project config: %w", err)
	}
	spinner.Success()

	return ProjectFile, nil
}

func (s *ConfigService) CreateAgent(ctx context.Context, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := domain.ValidateAgentName(name); err != nil {
		return "", err
	}

	spinner, err := s.ui.LogSpinner("Checking project configuration")
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(s.workspaceDir)
	if err != nil {
		spinner.Fail()
		return "", fmt.Errorf("open project: %w", err)
	}
	defer root.Close()

	project, err := readProjectConfig(root)
	if err != nil {
		spinner.Fail()
		return "", err
	}
	spinner.Success()

	spinner, err = s.ui.LogSpinner("Checking agents directory")
	if err != nil {
		return "", err
	}
	if err := root.MkdirAll(project.AgentsDir, 0o755); err != nil {
		spinner.Fail()
		return "", fmt.Errorf("create agents directory: %w", err)
	}

	agentsDir, err := root.OpenRoot(project.AgentsDir)
	if err != nil {
		spinner.Fail()
		return "", fmt.Errorf("open agents directory: %w", err)
	}
	defer agentsDir.Close()
	spinner.Success()

	agentConfigPath := filepath.Join(name, AgentFile)

	config := defaultAgent(name)
	config.Agent.Description = s.getDescription(config)
	config.Agent.Provider = s.getProvider(config)
	config.Agent.Model = s.getModel(config)
	config.Agent.Temperature = s.getTemperature(config)
	config.Agent.SystemPrompt = s.getSystemPrompt(config)
	if err := ctx.Err(); err != nil {
		return "", err
	}

	configBytes, err := encode(config)
	if err != nil {
		return "", err
	}

	if err := agentsDir.Mkdir(name, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("%w: %s", ErrAgentExists, name)
		}
		return "", fmt.Errorf("create agent directory: %w", err)
	}

	if err := writeNewFile(agentsDir, agentConfigPath, configBytes); err != nil {
		return "", errors.Join(fmt.Errorf("create agent config: %w", err), agentsDir.Remove(name))
	}

	agentPath := filepath.Join(project.AgentsDir, name, AgentFile)
	s.ui.LogSuccess("Agent path: ", agentPath)

	return agentPath, nil
}

func (s *ConfigService) LoadAgent(ctx context.Context, name string) (*domain.AgentConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := domain.ValidateAgentName(name); err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(s.workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}
	defer root.Close()
	project, err := readProjectConfig(root)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(project.AgentsDir, name, AgentFile)

	content, err := root.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent config %q: %w", path, err)
	}

	configuration, err := decode[*domain.AgentConfig](content)
	if err != nil {
		return nil, fmt.Errorf("load agent config %q: %w", path, err)
	}

	if configuration.Agent.Name != name {
		return nil, fmt.Errorf("agent.name %q does not match directory name %q", configuration.Agent.Name, name)
	}

	configuration.Path, err = filepath.Abs(filepath.Join(s.workspaceDir, path))
	if err != nil {
		return nil, fmt.Errorf("resolve agent config path: %w", err)
	}

	return configuration, nil
}

func readProjectConfig(root *os.Root) (*domain.ProjectConfig, error) {
	content, err := root.ReadFile(ProjectFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotInitialized
		}
		return nil, fmt.Errorf("read project config: %w", err)
	}
	configuration, err := decode[*domain.ProjectConfig](content)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", ProjectFile, err)
	}
	return configuration, nil
}

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
