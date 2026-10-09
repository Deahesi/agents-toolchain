package domain

import (
	"fmt"

	"github.com/invopop/validation"
)

const ConfigVersion = "1.0"

const temperatureError = "agent.temperature must be a finite number between 0 and 2"

type ProjectConfig struct {
	AgentsDir string `yaml:"agents_dir"`
}

func (c *ProjectConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.AgentsDir, localPath("agents_dir must be a relative directory inside the project")),
	)
}

type AgentConfig struct {
	Version string `yaml:"version"`
	Agent   Agent  `yaml:"agent"`
	Path    string `yaml:"-"`
}

func (c *AgentConfig) Validate() error {
	message := fmt.Sprintf("version must be %q, got %q", ConfigVersion, c.Version)
	return validation.ValidateStruct(c,
		validation.Field(&c.Version, validation.Required.Error(message), validation.In(ConfigVersion).Error(message)),
		validation.Field(&c.Agent, validation.By(func(any) error {
			return c.Agent.Validate()
		})),
	)
}

type Agent struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`

	MaxOutputTokens *int   `yaml:"max_output_tokens,omitempty"`
	Reasoning       *bool  `yaml:"reasoning,omitempty"`
	WorkDir         string `yaml:"work_dir"`

	Think        *bool        `yaml:"think,omitempty"`
	SystemPrompt string       `yaml:"system_prompt"`
	Memory       *Memory      `yaml:"memory,omitempty"`
	Tools        []ToolConfig `yaml:"tools"`
}

func ValidateProvider(p any) error {
	providers := make([]string, len(Providers))
	for i, provider := range Providers {
		providers[i] = string(provider)
	}
	return validation.Validate(p,
		requiredText("Provider is required"),
		oneOf(fmt.Sprintf("unsupported agent.provider %q", p), providers...),
	)
}

func ValidateModel(m any) error {
	return validation.Validate(m,
		requiredText("agent.model is required"),
		withoutWhitespace("agent.model must not contain whitespace"),
	)
}

func ValidateWorkDir(d any) error {
	return validation.Validate(d,
		requiredText("Work dir is required"),
		existingDirectory("Work dir must point to an existing directory"),
	)
}

func ValidateTemperature(t any) error {
	return validation.Validate(t,
		finiteNumber(temperatureError),
		validation.Min(0.0).Error(temperatureError),
		validation.Max(2.0).Error(temperatureError),
	)
}

func ValidateSystemPrompt(p any) error {
	return validation.Validate(p, requiredText("System prompt is required"))
}

func (a *Agent) Validate() error {
	if a.Think != nil && ProviderKey(a.Provider) != Ollama {
		return fmt.Errorf("think is only supported for the ollama provider")
	}
	return validation.ValidateStruct(a,
		validation.Field(&a.Name, validation.By(func(value any) error {
			return ValidateAgentName(value.(string))
		})),
		validation.Field(&a.Provider, validation.By(ValidateProvider)),
		validation.Field(&a.Model, validation.By(ValidateModel)),
		validation.Field(&a.Temperature, validation.By(ValidateTemperature)),
		validation.Field(&a.SystemPrompt, validation.By(ValidateSystemPrompt)),

		validation.Field(&a.MaxOutputTokens, positiveInteger("agent.max_output_tokens must be a positive integer")),
		validation.Field(&a.WorkDir, validation.By(ValidateWorkDir)),

		validation.Field(&a.Memory),
		validation.Field(&a.Tools, uniqueBy("duplicate agent tool name", func(tool ToolConfig) string { return tool.Name })),
	)
}

type Memory struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

func (m *Memory) Validate() error {
	message := fmt.Sprintf("unsupported agent.memory.type %q", m.Type)
	return validation.ValidateStruct(m,
		validation.Field(&m.Type, validation.Required.Error(message), validation.In("local_file").Error(message)),
		validation.Field(&m.Path, localPath("agent.memory.path must be a relative path inside the agent directory")),
	)
}

type ToolType string

const (
	ToolTypeBuiltin ToolType = "builtin"
	ToolTypeMCP     ToolType = "mcp"
	ToolTypeCustom  ToolType = "custom"
)

type ToolConfig struct {
	Type                ToolType       `yaml:"type"`
	Name                string         `yaml:"name"`
	Description         string         `yaml:"description,omitempty"`
	Server              string         `yaml:"server,omitempty"`
	Config              map[string]any `yaml:"config,omitempty"`
	AllowWithoutConfirm bool           `yaml:"allow_without_confirm,omitempty"`
}

func (t *ToolConfig) Validate() error {
	const message = "each agent tool requires name and type"
	return validation.ValidateStruct(&t,
		validation.Field(&t.Name, requiredText(message)),
		validation.Field(&t.Type, requiredText(message)),
	)
}
