package runtime

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Deahesi/agents-toolchain/internal/agents"
	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/tools"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type RuntimeService struct {
	configs          *config.ConfigService
	runtime          *genkit.Genkit
	ui               domain.UI
	toolRegistration *tools.ToolRegistration
}

func NewRuntimeService(ui domain.UI, configs *config.ConfigService) *RuntimeService {
	return &RuntimeService{
		ui:      ui,
		configs: configs,
	}
}

func (s *RuntimeService) Run(ctx context.Context, name, input string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	configuration, err := s.configs.LoadAgent(ctx, name)
	if err != nil {
		return err
	}

	providerPlugin, err := agents.GetProviderPlugin(ctx, configuration.Agent.Provider)
	if err != nil {
		return err
	}

	s.runtime = genkit.Init(
		ctx,
		providerPlugin,
	)
	s.toolRegistration = tools.NewToolRegistration(s.runtime, configuration.Agent.Tools)
	s.toolRegistration.Register()

	if strings.TrimSpace(input) == "" {
		input = "Follow the task described in the system prompt."
	}

	if err := s.stream(ctx, configuration.Agent, input); err != nil {
		return fmt.Errorf("run agent %q: %w", name, err)
	}

	return nil
}

func (s *RuntimeService) stream(ctx context.Context, agent domain.Agent, input string) (err error) {
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

	var lastPart ai.PartKind
	response, err := genkit.Generate(ctx, s.runtime,
		ai.WithModelName(fmt.Sprintf("%s/%s", agent.Provider, agent.Model)),
		ai.WithConfig(map[string]any{"temperature": agent.Temperature}),
		ai.WithSystemParts(ai.NewTextPart(agent.SystemPrompt)),
		ai.WithPromptParts(ai.NewTextPart(input)),
		ai.WithTools(s.toolRegistration.Tools...),
		ai.WithStreaming(func(_ context.Context, chunk *ai.ModelResponseChunk) error {
			for _, part := range chunk.Content {
				if lastPart != part.Kind {
					area.Update("\n")
				}

				switch {
				case part.IsText():
					text.WriteString(part.Text)
					area.Update(part.Text, "#FFFFFF")
				case part.IsReasoning():
					area.Update(part.Text, "#7A7A7A")
				case part.IsToolRequest():
					if part.ToolRequest.Name != "" {
						area.Update(fmt.Sprintf("[calling %s]", part.ToolRequest.Name), "#4e579d")
					}
				case part.IsToolResponse():
					if part.ToolResponse.Name != "" {
						area.Update(fmt.Sprintf("[%s done]", part.ToolResponse.Name), "#338053")
					}
				}

				lastPart = part.Kind
			}
			return nil
		}),
	)

	if err != nil {
		spinner.Fail()
		return err
	}
	if text.Len() == 0 {
		text.WriteString(response.Text())
		area.Update(response.Text(), "#FFFFFF")
	}
	if text.Len() > 0 && !strings.HasSuffix(text.String(), "\n") {
		area.Update("\n")
	}

	spinner.Success()
	return nil
}
