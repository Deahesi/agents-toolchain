package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

	spinner, _ := s.ui.LogSpinner("Checking project configuration")
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

	spinner, _ = s.ui.LogSpinner("Creating agents directory: ", agentsDir)
	if err := root.MkdirAll(agentsDir, 0o755); err != nil {
		spinner.Fail()
		return "", fmt.Errorf("create agents directory: %w", err)
	}
	spinner.Success()

	spinner, _ = s.ui.LogSpinner("Writing project config: ", ProjectFile)
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
	if err := domain.ValidateAgentName(name); err != nil {
		return "", err
	}

	spinner, _ := s.ui.LogSpinner("Checking project configuration")
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

	spinner, _ = s.ui.LogSpinner("Checking agents directory")
	if err := root.MkdirAll(project.AgentsDir, 0o755); err != nil {
		spinner.Fail()
		return "", fmt.Errorf("create agents directory: %w", err)
	}

	agents, err := root.OpenRoot(project.AgentsDir)
	if err != nil {
		spinner.Fail()
		return "", fmt.Errorf("open agents directory: %w", err)
	}
	defer agents.Close()
	spinner.Success()

	spinner, _ = s.ui.LogSpinner("Creating agent")
	if err := agents.Mkdir(name, 0o755); err != nil {
		spinner.Fail()

		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("%w: %s", ErrAgentExists, name)
		}
		return "", fmt.Errorf("create agent directory: %w", err)
	}

	agentConfigPath := filepath.Join(name, AgentFile)

	configBytes, err := encode(defaultAgent(name))
	if err != nil {
		spinner.Fail()
		return "", err
	}

	if err := writeNewFile(agents, agentConfigPath, configBytes); err != nil {
		spinner.Fail()
		return "", errors.Join(fmt.Errorf("create agent config: %w", err), agents.Remove(name))
	}

	spinner.Success()

	agentPath := filepath.Join(project.AgentsDir, name, AgentFile)
	s.ui.LogStep("Agent path: ", agentPath)

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
