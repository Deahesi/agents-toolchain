package config

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
)

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

	root, err := files.OpenRoot(s.workspaceDir)
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

	config := defaultAgent(name)
	config.Agent.Description = s.getDescription(config)
	config.Agent.Provider = s.getProvider(config)
	config.Agent.Model = s.getModel(config)
	config.Agent.Temperature = s.getTemperature(config)
	config.Agent.SystemPrompt = s.getSystemPrompt(config)
	config.Agent.WorkDir = s.getWorkDir(config)
	config.Agent.MaxOutputTokens = s.GetMaxOutputTokens(config)
	// config.Agent.Reasoning = s.Get(config)

	if err := ctx.Err(); err != nil {
		return "", err
	}

	configBytes, err := encode(config)
	if err != nil {
		return "", err
	}

	if err := agentsDir.Mkdir(name, 0o755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%w: %s", ErrAgentExists, name)
		}
		return "", fmt.Errorf("create agent directory: %w", err)
	}

	agentDir, err := agentsDir.OpenRoot(name)
	if err != nil {
		return "", errors.Join(fmt.Errorf("open agent directory: %w", err), agentsDir.Remove(name))
	}

	writeErr := agentDir.WriteNewFile(AgentFile, configBytes, 0o644)
	if err := errors.Join(writeErr, agentDir.Close()); err != nil {
		return "", errors.Join(fmt.Errorf("create agent config: %w", err), agentsDir.Remove(name))
	}

	agentPath := filepath.Join(project.AgentsDir, name, AgentFile)
	s.ui.LogSuccess("Agent path: ", agentPath)

	return agentPath, nil
}

func (s *ConfigService) LoadAgent(ctx context.Context, name string) (*domain.AgentConfig, error) {
	agentDir, err := s.openAgentDir(ctx, name)
	if err != nil {
		return nil, err
	}
	defer agentDir.Close()

	content, err := agentDir.ReadFile(AgentFile)
	if err != nil {
		return nil, fmt.Errorf("read agent config %q: %w", name, err)
	}

	configuration, err := decode[*domain.AgentConfig](content)
	if err != nil {
		return nil, fmt.Errorf("load agent config %q: %w", name, err)
	}

	if configuration.Agent.Name != name {
		return nil, fmt.Errorf("agent.name %q does not match directory name %q", configuration.Agent.Name, name)
	}

	configuration.Path, err = agentDir.Path(AgentFile)
	if err != nil {
		return nil, fmt.Errorf("resolve agent config path: %w", err)
	}

	return configuration, nil
}

func (s *ConfigService) openAgentDir(ctx context.Context, name string) (*files.Root, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := domain.ValidateAgentName(name); err != nil {
		return nil, err
	}

	root, err := files.OpenRoot(s.workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}
	defer root.Close()
	project, err := readProjectConfig(root)
	if err != nil {
		return nil, err
	}

	agentsDir, err := root.OpenRoot(project.AgentsDir)
	if err != nil {
		return nil, fmt.Errorf("open agents directory: %w", err)
	}
	defer agentsDir.Close()

	agentDir, err := agentsDir.OpenRoot(name)
	if err != nil {
		return nil, fmt.Errorf("open agent directory %q: %w", name, err)
	}
	return agentDir, nil
}

func (s *ConfigService) GetAllAgents(ctx context.Context) ([]domain.AgentConfig, error) {
	root, err := files.OpenRoot(s.workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}
	defer root.Close()
	project, err := readProjectConfig(root)
	if err != nil {
		return nil, err
	}

	entries, err := root.ReadDir(project.AgentsDir)

	if err != nil {
		return nil, err
	}

	agents := make([]domain.AgentConfig, 0, len(entries))
	for _, entry := range entries {

		agent, err := s.LoadAgent(ctx, entry.Name())
		if err != nil {
			s.ui.LogError(fmt.Sprintf("Cannot read %s:%s", entry.Name(), err))
			continue
		}

		agents = append(agents, *agent)
	}

	return agents, nil
}

func (s *ConfigService) GetAgentValue(ctx context.Context, name string, field string) (string, error) {
	agentDir, err := s.openAgentDir(ctx, name)
	if err != nil {
		return "", err
	}
	defer agentDir.Close()
	content, err := agentDir.ReadFile(AgentFile)
	if err != nil {
		return "", fmt.Errorf("read agent config %q: %w", name, err)
	}
	val, err := getAgentYAMLValue(content, field)

	return val, err
}

func (s *ConfigService) SetAgentValue(ctx context.Context, name, field, value string) error {
	agentDir, err := s.openAgentDir(ctx, name)
	if err != nil {
		return err
	}
	defer agentDir.Close()
	content, err := agentDir.ReadFile(AgentFile)
	if err != nil {
		return fmt.Errorf("read agent config %q: %w", name, err)
	}
	updated, err := setAgentYAMLValue(content, field, value)
	if err != nil {
		return fmt.Errorf("set field %q: %w", field, err)
	}
	configuration, err := decode[*domain.AgentConfig](updated)
	if err != nil {
		return err
	}
	if configuration.Agent.Name != name {
		return fmt.Errorf("agent.name %q does not match directory name %q", configuration.Agent.Name, name)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := agentDir.ReplaceFile(AgentFile, updated); err != nil {
		return fmt.Errorf("save agent config %q: %w", name, err)
	}

	return nil
}
