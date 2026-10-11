package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Deahesi/agents-toolchain/internal/domain"
)

type uiMock struct {
	InteractiveSelectFunc  func(message string, options ...string) (string, error)
	TextInputFunc          func(message ...string) (string, error)
	TextInputMultilineFunc func(message ...string) (string, error)

	LogErrorCalledTimes int
}

var _ domain.UI = (*uiMock)(nil)

func (m *uiMock) PrintFields(string, []domain.Field) {}
func (m *uiMock) PrintTable([]string, [][]string)    {}
func (m *uiMock) LogStep(messages ...any)            {}
func (m *uiMock) LogSuccess(messages ...any)         {}
func (m *uiMock) LogError(messages ...any) {
	m.LogErrorCalledTimes++
}
func (m *uiMock) LogSpinner(messages ...any) (domain.Spinner, error) {
	return nil, nil
}
func (m *uiMock) LogSpinnerTimer(roundingFactor time.Duration, message ...any) (domain.Spinner, error) {
	return nil, nil
}
func (m *uiMock) TextInput(message ...string) (string, error) {
	if m.TextInputFunc != nil {
		return m.TextInputFunc(message...)
	}
	return "", errors.New("TextInputFunc not implemented")
}
func (m *uiMock) TextInputMultiline(message ...string) (string, error) {
	if m.TextInputMultilineFunc != nil {
		return m.TextInputMultilineFunc(message...)
	}
	return "", errors.New("TextInputMultilineFunc not implemented")
}
func (m *uiMock) InteractiveSelect(message string, options ...string) (string, error) {
	if m.InteractiveSelectFunc != nil {
		return m.InteractiveSelectFunc(message, options...)
	}

	return "", errors.New("InteractiveSelectFunc not implemented")
}
func (m *uiMock) Area(message ...any) (domain.Area, error) {
	return nil, nil
}

func TestGetProvider(t *testing.T) {
	defaultConfig := defaultAgent("test")
	defaultConfig.Agent.Provider = "anthropic"
	cases := []inputTestCase[string]{
		{name: "input error returns default", inputs: []inputResponse{{err: errInput}}, want: "anthropic", wantErrors: 1},
		{name: "empty input returns default", inputs: []inputResponse{{}}, want: "anthropic"},
		{name: "invalid values then valid", inputs: []inputResponse{{text: "invalid"}, {text: "openrouter"}}, want: "openrouter", wantErrors: 1},
		{name: "input error after invalid value", inputs: []inputResponse{{text: "invalid"}, {err: errInput}}, want: "anthropic", wantErrors: 2},
	}
	for _, provider := range domain.Providers {
		cases = append(cases, inputTestCase[string]{name: "valid " + string(provider), inputs: []inputResponse{{text: string(provider)}}, want: string(provider)})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			respond := inputSequence(t, tc.inputs)
			mock := &uiMock{InteractiveSelectFunc: func(message string, options ...string) (string, error) {
				wantOptions := []string{"openai", "anthropic", "ollama", "openrouter"}
				if !slices.Equal(options, wantOptions) {
					t.Errorf("provider choices = %v; want %v", options, wantOptions)
				}
				return respond(message)
			}}
			got := NewConfigService(mock, "").getProvider(defaultConfig)
			if got != tc.want {
				t.Errorf("getProvider = %q; want %q", got, tc.want)
			}
			checkErrorCount(t, mock, tc.wantErrors)
		})
	}
}

func TestGetModel(t *testing.T) {
	defaultConfig := defaultAgent("test")
	defaultConfig.Agent.Model = "fallback-model"
	runTextInputTests(t, defaultConfig, (*ConfigService).getModel, false, []inputTestCase[string]{
		{name: "valid model", inputs: []inputResponse{{text: "qwen3:8b"}}, want: "qwen3:8b"},
		{name: "empty input returns default", inputs: []inputResponse{{}}, want: "fallback-model"},
		{name: "input error returns default", inputs: []inputResponse{{err: errInput}}, want: "fallback-model"},
		{name: "invalid values then valid", inputs: []inputResponse{{text: " \t"}, {text: "two words"}, {text: "model\nname"}, {text: "vendor/model"}}, want: "vendor/model", wantErrors: 3},
		{name: "input error after invalid value", inputs: []inputResponse{{text: " \t"}, {err: errInput}}, want: "fallback-model", wantErrors: 1},
	})
}

func TestGetDescription(t *testing.T) {
	defaultConfig := defaultAgent("test")
	defaultConfig.Agent.Description = "Fallback description"
	runTextInputTests(t, defaultConfig, (*ConfigService).getDescription, true, []inputTestCase[string]{
		{name: "multiline description", inputs: []inputResponse{{text: "Описание агента\nSecond line"}}, want: "Описание агента\nSecond line"},
		{name: "empty input returns default", inputs: []inputResponse{{}}, want: "Fallback description"},
		{name: "whitespace is preserved", inputs: []inputResponse{{text: " \t\n"}}, want: " \t\n"},
		{name: "input error returns default", inputs: []inputResponse{{err: errInput}}, want: "Fallback description", wantErrors: 1},
	})
}

func TestGetTemperature(t *testing.T) {
	defaultConfig := defaultAgent("test")
	defaultConfig.Agent.Temperature = 0.7
	runTextInputTests(t, defaultConfig, (*ConfigService).getTemperature, false, []inputTestCase[float64]{
		{name: "valid temperature", inputs: []inputResponse{{text: "1.25"}}, want: 1.25},
		{name: "empty input returns default", inputs: []inputResponse{{}}, want: 0.7},
		{name: "lower bound", inputs: []inputResponse{{text: "0"}}, want: 0},
		{name: "upper bound", inputs: []inputResponse{{text: "2"}}, want: 2},
		{name: "input error returns default", inputs: []inputResponse{{err: errInput}}, want: 0.7, wantErrors: 1},
		{name: "invalid values then valid", inputs: []inputResponse{
			{text: "not a number"}, {text: "-0.1"}, {text: "2.1"}, {text: "NaN"}, {text: "+Inf"}, {text: "-Inf"}, {text: "1e999"}, {text: "1.5"},
		}, want: 1.5, wantErrors: 7},
		{name: "input error after invalid value", inputs: []inputResponse{{text: "bad"}, {err: errInput}}, want: 0.7, wantErrors: 2},
	})
}

func TestGetSystemPrompt(t *testing.T) {
	defaultConfig := defaultAgent("test")
	defaultConfig.Agent.SystemPrompt = "Fallback system prompt"
	runTextInputTests(t, defaultConfig, (*ConfigService).getSystemPrompt, true, []inputTestCase[string]{
		{name: "multiline prompt", inputs: []inputResponse{{text: "Первая строка\nSecond line\n"}}, want: "Первая строка\nSecond line\n"},
		{name: "input error returns default", inputs: []inputResponse{{err: errInput}}, want: "Fallback system prompt", wantErrors: 1},
		{name: "empty input returns default", inputs: []inputResponse{{}}, want: "Fallback system prompt"},
		{name: "invalid values then valid", inputs: []inputResponse{{text: " \t\n"}, {text: "Prompt"}}, want: "Prompt", wantErrors: 1},
		{name: "input error after invalid value", inputs: []inputResponse{{text: " \t\n"}, {err: errInput}}, want: "Fallback system prompt", wantErrors: 2},
	})
}

func TestGetMaxOutputTokens(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	defaultLimit := 2048
	defaultConfig := defaultAgent("test")
	defaultConfig.Agent.MaxOutputTokens = &defaultLimit
	for _, tc := range []struct {
		inputTestCase[int]
		wantNil bool
	}{
		{inputTestCase: inputTestCase[int]{name: "valid decimal integer", inputs: []inputResponse{{text: "1024"}}, want: 1024}},
		{inputTestCase: inputTestCase[int]{name: "smallest positive integer", inputs: []inputResponse{{text: "1"}}, want: 1}},
		{inputTestCase: inputTestCase[int]{name: "largest platform integer", inputs: []inputResponse{{text: strconv.Itoa(maxInt)}}, want: maxInt}},
		{inputTestCase: inputTestCase[int]{name: "empty input returns default", inputs: []inputResponse{{}}, want: defaultLimit}},
		{inputTestCase: inputTestCase[int]{name: "input error returns nil", inputs: []inputResponse{{err: errInput}}}, wantNil: true},
		{inputTestCase: inputTestCase[int]{name: "invalid values then valid", inputs: []inputResponse{
			{text: "bad"}, {text: "1.5"}, {text: "99999999999999999999999999"},
			{text: strconv.FormatUint(uint64(maxInt)+1, 10)}, {text: "0"}, {text: "-1"}, {text: "32"},
		}, want: 32, wantErrors: 6}},
		{inputTestCase: inputTestCase[int]{name: "input error after invalid value", inputs: []inputResponse{{text: "bad"}, {err: errInput}}, wantErrors: 1}, wantNil: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := &uiMock{TextInputFunc: inputSequence(t, tc.inputs)}
			got := NewConfigService(mock, "").GetMaxOutputTokens(defaultConfig)
			if tc.wantNil {
				if got != nil {
					t.Errorf("GetMaxOutputTokens = %d; want nil", *got)
				}
			} else if got == nil || *got != tc.want {
				t.Errorf("GetMaxOutputTokens = %v; want pointer to %d", got, tc.want)
			}
			if *defaultConfig.Agent.MaxOutputTokens != defaultLimit {
				t.Error("default token limit was modified")
			}
			checkErrorCount(t, mock, tc.wantErrors)
		})
	}
}

func TestGetWorkDir(t *testing.T) {
	defaultConfig := defaultAgent("test")
	defaultConfig.Agent.WorkDir = t.TempDir()
	dir := filepath.Join(t.TempDir(), "directory with spaces")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTextInputTests(t, defaultConfig, (*ConfigService).getWorkDir, false, []inputTestCase[string]{
		{name: "absolute directory with spaces", inputs: []inputResponse{{text: dir}}, want: dir},
		{name: "relative directory", inputs: []inputResponse{{text: "."}}, want: "."},
		{name: "empty input returns default", inputs: []inputResponse{{}}, want: defaultConfig.Agent.WorkDir},
		{name: "input error returns default", inputs: []inputResponse{{err: errInput}}, want: defaultConfig.Agent.WorkDir, wantErrors: 1},
		{name: "invalid paths then valid", inputs: []inputResponse{
			{text: " \t"}, {text: filepath.Join(dir, "missing")}, {text: file}, {text: "invalid\x00path"}, {text: dir},
		}, want: dir, wantErrors: 4},
		{name: "input error after invalid path", inputs: []inputResponse{{text: file}, {err: errInput}}, want: defaultConfig.Agent.WorkDir, wantErrors: 2},
	})
}

func TestEmptyInputWithoutDefault(t *testing.T) {
	for _, tc := range []struct {
		inputTestCase[string]
		read func(*ConfigService, *domain.AgentConfig) string
	}{
		{inputTestCase[string]{name: "provider", inputs: []inputResponse{{}, {text: "ollama"}}, want: "ollama", wantErrors: 1}, (*ConfigService).getProvider},
		{inputTestCase[string]{name: "model", inputs: []inputResponse{{}, {text: "qwen3"}}, want: "qwen3", wantErrors: 1}, (*ConfigService).getModel},
		{inputTestCase[string]{name: "description", inputs: []inputResponse{{}}, want: ""}, (*ConfigService).getDescription},
		{inputTestCase[string]{name: "system prompt", inputs: []inputResponse{{}, {text: "Prompt"}}, want: "Prompt", wantErrors: 1}, (*ConfigService).getSystemPrompt},
		{inputTestCase[string]{name: "work directory", inputs: []inputResponse{{}, {text: "."}}, want: ".", wantErrors: 1}, (*ConfigService).getWorkDir},
	} {
		t.Run(tc.name, func(t *testing.T) {
			respond := inputSequence(t, tc.inputs)
			mock := &uiMock{
				TextInputFunc: respond, TextInputMultilineFunc: respond,
				InteractiveSelectFunc: func(message string, options ...string) (string, error) { return respond(message) },
			}
			if got := tc.read(NewConfigService(mock, ""), &domain.AgentConfig{}); got != tc.want {
				t.Errorf("result = %q; want %q", got, tc.want)
			}
			checkErrorCount(t, mock, tc.wantErrors)
		})
	}
	t.Run("zero temperature default", func(t *testing.T) {
		mock := &uiMock{TextInputFunc: inputSequence(t, []inputResponse{{}})}
		if got := NewConfigService(mock, "").getTemperature(&domain.AgentConfig{}); got != 0 {
			t.Errorf("temperature = %v; want zero default", got)
		}
		checkErrorCount(t, mock, 0)
	})
	t.Run("no token limit", func(t *testing.T) {
		mock := &uiMock{TextInputFunc: inputSequence(t, []inputResponse{{}})}
		if got := NewConfigService(mock, "").GetMaxOutputTokens(&domain.AgentConfig{}); got != nil {
			t.Errorf("token limit = %v; want nil", got)
		}
		checkErrorCount(t, mock, 0)
	})
}

func TestDefaultValueLabels(t *testing.T) {
	limit := 2048
	for _, tc := range []struct {
		name  string
		value any
		exist bool
		label string
	}{
		{"nil", nil, false, "Input"},
		{"nil pointer", (*int)(nil), false, "Input"},
		{"empty string", "", false, "Input"},
		{"string", "value", true, "Input (default: value)"},
		{"zero integer", 0, true, "Input (default: 0)"},
		{"zero temperature", 0.0, true, "Input (default: 0)"},
		{"false", false, true, "Input (default: false)"},
		{"integer pointer", &limit, true, "Input (default: 2048)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkDefaultExist(tc.value); got != tc.exist {
				t.Errorf("checkDefaultExist = %v; want %v", got, tc.exist)
			}
			if got := addDefaultIfExist("Input", tc.value); got != tc.label {
				t.Errorf("label = %q; want %q", got, tc.label)
			}
		})
	}

	defaults := defaultAgent("test")
	defaults.Agent.Provider = "ollama"
	defaults.Agent.Model = "qwen3"
	defaults.Agent.Description = "Description"
	defaults.Agent.SystemPrompt = "Prompt"
	defaults.Agent.Temperature = 0
	defaults.Agent.MaxOutputTokens = &limit
	defaults.Agent.WorkDir = "."
	for _, tc := range []struct {
		name string
		read func(*ConfigService, *domain.AgentConfig)
		want string
	}{
		{"provider", func(s *ConfigService, c *domain.AgentConfig) { s.getProvider(c) }, "(default: ollama)"},
		{"model", func(s *ConfigService, c *domain.AgentConfig) { s.getModel(c) }, "(default: qwen3)"},
		{"description", func(s *ConfigService, c *domain.AgentConfig) { s.getDescription(c) }, "(default: Description)"},
		{"temperature", func(s *ConfigService, c *domain.AgentConfig) { s.getTemperature(c) }, "(default: 0)"},
		{"system prompt", func(s *ConfigService, c *domain.AgentConfig) { s.getSystemPrompt(c) }, "(default: Prompt)"},
		{"token limit", func(s *ConfigService, c *domain.AgentConfig) { s.GetMaxOutputTokens(c) }, "(default: 2048)"},
		{"work directory", func(s *ConfigService, c *domain.AgentConfig) { s.getWorkDir(c) }, "(default: .)"},
	} {
		t.Run("field/"+tc.name, func(t *testing.T) {
			respond := inputSequence(t, []inputResponse{{}})
			input := func(messages ...string) (string, error) {
				if len(messages) != 1 || !strings.HasSuffix(messages[0], tc.want) {
					t.Errorf("input label = %v; want suffix %q", messages, tc.want)
				}
				return respond(messages...)
			}
			mock := &uiMock{
				TextInputFunc: input, TextInputMultilineFunc: input,
				InteractiveSelectFunc: func(message string, options ...string) (string, error) { return input(message) },
			}
			tc.read(NewConfigService(mock, ""), defaults)
			checkErrorCount(t, mock, 0)
		})
	}
}

var errInput = errors.New("input interrupted")

type inputResponse struct {
	text string
	err  error
}

type inputTestCase[T comparable] struct {
	name       string
	inputs     []inputResponse
	want       T
	wantErrors int
}

// Fail on unexpected retries instead of letting a broken input loop hang a test.
func inputSequence(t *testing.T, inputs []inputResponse) func(...string) (string, error) {
	t.Helper()
	next := 0
	t.Cleanup(func() {
		if next != len(inputs) {
			t.Errorf("consumed %d inputs; want %d", next, len(inputs))
		}
	})
	return func(...string) (string, error) {
		t.Helper()
		if next >= len(inputs) {
			t.Fatalf("unexpected input request after %d responses", next)
		}
		response := inputs[next]
		next++
		return response.text, response.err
	}
}

func runTextInputTests[T comparable](t *testing.T, defaults *domain.AgentConfig, read func(*ConfigService, *domain.AgentConfig) T, multiline bool, cases []inputTestCase[T]) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &uiMock{}
			if multiline {
				mock.TextInputMultilineFunc = inputSequence(t, tc.inputs)
			} else {
				mock.TextInputFunc = inputSequence(t, tc.inputs)
			}
			got := read(NewConfigService(mock, ""), defaults)
			if got != tc.want {
				t.Errorf("result = %v; want %v", got, tc.want)
			}
			checkErrorCount(t, mock, tc.wantErrors)
		})
	}
}

func checkErrorCount(t *testing.T, mock *uiMock, want int) {
	t.Helper()
	if mock.LogErrorCalledTimes != want {
		t.Errorf("LogErrorCalledTimes = %d; want %d", mock.LogErrorCalledTimes, want)
	}
}
