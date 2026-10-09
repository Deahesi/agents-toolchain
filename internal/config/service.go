package config

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
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

func readProjectConfig(root *files.Root) (*domain.ProjectConfig, error) {
	content, err := root.ReadFile(ProjectFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
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
