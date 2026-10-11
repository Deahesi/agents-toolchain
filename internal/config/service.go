package config

import (
	"errors"

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
