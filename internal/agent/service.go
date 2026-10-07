package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type AgentService struct {
	configs *config.ConfigService
	runtime *genkit.Genkit
}

func NewAgentService(configs *config.ConfigService) *AgentService {
	return &AgentService{configs: configs}
}

func (s *AgentService) Run(ctx context.Context, name, input string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	configuration, err := s.configs.LoadAgent(ctx, name)
	if err != nil {
		return err
	}

	runtime, err := initialize(ctx, configuration.Agent.Model)
	if err != nil {
		return err
	}

	if strings.TrimSpace(input) == "" {
		input = "Follow the task described in the system prompt."
	}

	if err := stream(ctx, runtime, configuration.Agent, input, output); err != nil {
		return fmt.Errorf("run agent %q: %w", name, err)
	}

	return nil
}

func stream(ctx context.Context, runtime *genkit.Genkit, agent domain.Agent, input string, output io.Writer) error {
	lastText := ""
	response, err := genkit.Generate(ctx, runtime,
		ai.WithModelName(agent.Model),
		ai.WithConfig(map[string]any{"temperature": agent.Temperature}),
		ai.WithSystemParts(ai.NewTextPart(agent.SystemPrompt)),
		ai.WithPromptParts(ai.NewTextPart(input)),
		ai.WithStreaming(func(_ context.Context, chunk *ai.ModelResponseChunk) error {
			text := chunk.Text()
			if text == "" {
				return nil
			}
			lastText = text
			_, err := io.WriteString(output, text)
			return err
		}),
	)
	if err != nil {
		return err
	}
	if lastText == "" {
		lastText = response.Text()
		if _, err := io.WriteString(output, lastText); err != nil {
			return err
		}
	}
	if lastText != "" && !strings.HasSuffix(lastText, "\n") {
		_, err := io.WriteString(output, "\n")
		return err
	}
	return nil
}
