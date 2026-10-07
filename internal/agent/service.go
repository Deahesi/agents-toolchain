package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type AgentService struct {
	configs *config.ConfigService
	runtime *genkit.Genkit
	ui      domain.UI
}

func NewAgentService(ui domain.UI, configs *config.ConfigService) *AgentService {
	return &AgentService{
		ui:      ui,
		configs: configs,
	}
}

func (s *AgentService) Run(ctx context.Context, name, input string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	configuration, err := s.configs.LoadAgent(ctx, name)
	if err != nil {
		return err
	}

	s.runtime, err = initialize(ctx, configuration.Agent.Provider)
	if err != nil {
		return err
	}

	if strings.TrimSpace(input) == "" {
		input = "Follow the task described in the system prompt."
	}

	if err := s.stream(ctx, configuration.Agent, input); err != nil {
		return fmt.Errorf("run agent %q: %w", name, err)
	}

	return nil
}

func (s *AgentService) stream(ctx context.Context, agent domain.Agent, input string) (err error) {
	var text strings.Builder
	spinner, err := s.ui.LogSpinnerTimer(time.Second, "Generating response")
	if err != nil {
		return err
	}
	area, err := s.ui.Area("Output")
	if err != nil {
		spinner.Fail("Failed to start output")
		return err
	}
	defer area.Stop()

	response, err := genkit.Generate(ctx, s.runtime,
		ai.WithModelName(fmt.Sprintf("%s/%s", agent.Provider, agent.Model)),
		ai.WithConfig(map[string]any{"temperature": agent.Temperature}),
		ai.WithSystemParts(ai.NewTextPart(agent.SystemPrompt)),
		ai.WithPromptParts(ai.NewTextPart(input)),
		ai.WithStreaming(func(_ context.Context, chunk *ai.ModelResponseChunk) error {
			chunkText := chunk.Text()
			if chunkText == "" {
				return nil
			}
			text.WriteString(chunkText)
			area.Update(text.String())
			return nil
		}),
	)

	if err != nil {
		spinner.Fail()
		return err
	}
	if text.Len() == 0 {
		text.WriteString(response.Text())
	}
	if text.Len() > 0 && !strings.HasSuffix(text.String(), "\n") {
		text.WriteString("\n")
	}
	area.Update(text.String())

	spinner.Success()
	return nil
}
