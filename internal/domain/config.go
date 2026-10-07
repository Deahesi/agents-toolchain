package domain

import (
	"fmt"

	"github.com/invopop/validation"
)

const ConfigVersion = "1.0"

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
	Name         string  `yaml:"name"`
	Description  string  `yaml:"description"`
	Model        string  `yaml:"model"`
	Temperature  float64 `yaml:"temperature"`
	SystemPrompt string  `yaml:"system_prompt"`
	Memory       *Memory `yaml:"memory,omitempty"`
	Tools        []Tool  `yaml:"tools"`
}

func (a *Agent) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Name, validation.By(func(value any) error {
			return ValidateAgentName(value.(string))
		})),
		validation.Field(&a.Model, validation.By(validateModel)),
		validation.Field(&a.Temperature,
			validation.By(validateFiniteNumber),
			validation.Min(0.0).Error(temperatureError),
			validation.Max(2.0).Error(temperatureError),
		),
		validation.Field(&a.SystemPrompt, requiredText("agent.system_prompt is required")),
		validation.Field(&a.Memory),
		validation.Field(&a.Tools, validation.By(validateUniqueToolNames)),
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

type Tool struct {
	Name                string `yaml:"name"`
	Type                string `yaml:"type"`
	Description         string `yaml:"description"`
	AllowWithoutConfirm bool   `yaml:"allow_without_confirm"`
}

func (t Tool) Validate() error {
	const message = "each agent tool requires name and type"
	return validation.ValidateStruct(&t,
		validation.Field(&t.Name, requiredText(message)),
		validation.Field(&t.Type, requiredText(message)),
	)
}
