package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Deahesi/agents-toolchain/internal/agents"
	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
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

type MemoryStore interface {
	Close() error
	Load(ctx context.Context) ([]*ai.Message, error)
	Save(ctx context.Context, messages []*ai.Message) error
}

func NewRuntimeService(ui domain.UI, configs *config.ConfigService) *RuntimeService {
	return &RuntimeService{
		ui:      ui,
		configs: configs,
	}
}

func (s *RuntimeService) Run(
	ctx context.Context,
	input string,
	config *domain.AgentConfig,
	memory MemoryStore,
	cb func(ctx context.Context, chunk *ai.ModelResponseChunk) error,
) (*ai.ModelResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	//AGENT WORK DIR
	root, err := files.OpenRoot(config.Agent.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("open agent work directory: %w", err)
	}
	defer root.Close()

	//PROVIDER
	providerPlugin, err := agents.GetProviderPlugin(ctx, config.Agent.Provider)
	if err != nil {
		return nil, err
	}

	s.runtime = genkit.Init(
		ctx,
		providerPlugin,
	)
	s.toolRegistration = tools.NewToolRegistration(s.runtime, config.Agent.Tools, root)
	err = s.toolRegistration.Register()

	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(input) == "" {
		input = "Follow the task described in the system prompt."
	}

	response, err := s.stream(ctx, config.Agent, memory, input, cb)

	if err != nil {
		return nil, fmt.Errorf("run agent %q: %w", config.Agent.Name, err)
	}

	return response, nil
}

func (s *RuntimeService) stream(
	ctx context.Context,
	agent domain.Agent,
	store MemoryStore,
	input string,
	cb func(ctx context.Context, chunk *ai.ModelResponseChunk) error,
) (*ai.ModelResponse, error) {
	config, err := agents.GetProviderConfig(ctx, &agent)
	if err != nil {
		return nil, fmt.Errorf("configure provider %q: %w", agent.Provider, err)
	}

	var text strings.Builder

	spinner, err := s.ui.LogSpinnerTimer(time.Second, "Generating response")
	if err != nil {
		return nil, err
	}
	area, err := s.ui.Area("Output")
	if err != nil {
		spinner.Fail("Failed to start output")
		return nil, err
	}
	defer area.Stop()

	var messages []*ai.Message = nil

	if store != nil {
		messages, err = store.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("load memory: %w", err)
		}
	}

	var lastPart ai.PartKind
	response, err := genkit.Generate(ctx, s.runtime,
		ai.WithModelName(fmt.Sprintf("%s/%s", agent.Provider, agent.Model)),
		ai.WithConfig(config),
		ai.WithSystemParts(ai.NewTextPart(agent.SystemPrompt)),
		ai.WithPromptParts(ai.NewTextPart(input)),
		ai.WithTools(s.toolRegistration.Tools...),
		ai.WithMessages(messages...),
		ai.WithStreaming(func(ctx context.Context, chunk *ai.ModelResponseChunk) error {
			if cb != nil {
				cb(ctx, chunk)
			}

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
		return nil, err
	}
	if text.Len() == 0 {
		text.WriteString(response.Text())
		area.Update(response.Text(), "#FFFFFF")
	}
	if text.Len() > 0 && !strings.HasSuffix(text.String(), "\n") {
		area.Update("\n")
	}

	if store != nil {
		if err = store.Save(ctx, response.History()); err != nil {
			spinner.Fail()
			return nil, fmt.Errorf("fail to save history: %w", err)
		}
	}

	if strings.TrimSpace(text.String()) == "" {
		spinner.Fail()
		return nil, fmt.Errorf("model returned no final text (only reasoning or an empty response)")
	}

	spinner.Success()
	return response, nil
}
