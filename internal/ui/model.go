package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type request struct {
	apply func(*model) tea.Cmd
	ack   chan struct{}
	flush bool
}
type acknowledged struct{ ack chan struct{} }
type inputResult struct {
	value string
	err   error
}
type prompt interface {
	Init() tea.Cmd
	Update(tea.Msg) (tea.Cmd, *inputResult)
	View() string
	resize(int, int)
}

type model struct {
	width, height int
	spinners      []*spinnerState
	areas         []*areaState
	prompt        prompt
	result        chan inputResult
	canceled      bool
	cancel        func()
}

func newModel() *model         { return &model{width: 80, height: 24} }
func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if req, ok := msg.(request); ok {
		cmd := req.apply(m)
		m.resize()
		if cmd == nil || !req.flush {
			close(req.ack)
			return m, cmd
		}
		return m, tea.Sequence(cmd, func() tea.Msg { return acknowledged{ack: req.ack} })
	}
	if ack, ok := msg.(acknowledged); ok {
		close(ack.ack)
		return m, nil
	}
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = max(2, size.Width), max(2, size.Height)
		m.resize()
	}
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		m.canceled = true
		if m.result != nil {
			m.result <- inputResult{err: ErrCanceled}
			m.result = nil
		}
		m.prompt = nil
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit
	}
	var commands []tea.Cmd
	for _, state := range m.spinners {
		var cmd tea.Cmd
		state.spinner, cmd = state.spinner.Update(msg)
		commands = append(commands, cmd)
	}
	if m.prompt != nil {
		cmd, result := m.prompt.Update(msg)
		commands = append(commands, cmd)
		if result != nil {
			m.result <- *result
			m.result, m.prompt = nil, nil
			m.resize()
		}
	} else {
		for _, area := range m.areas {
			var cmd tea.Cmd
			area.viewport, cmd = area.viewport.Update(msg)
			commands = append(commands, cmd)
		}
	}
	return m, tea.Batch(commands...)
}

func (m *model) resize() {
	available := max(1, m.height-1-len(m.spinners))
	if m.prompt != nil {
		m.prompt.resize(max(1, m.width-1), available)
		available -= strings.Count(m.prompt.View(), "\n") + 1
	}
	for _, area := range m.areas {
		area.resize(max(1, m.width-1), max(1, available/max(1, len(m.areas))-1))
	}
}

func (m *model) View() string {
	var sections []string
	for _, state := range m.spinners {
		sections = append(sections, state.View())
	}
	for _, area := range m.areas {
		sections = append(sections, area.View())
	}
	if m.prompt != nil {
		sections = append(sections, m.prompt.View())
	}
	if len(sections) == 0 {
		return ""
	}
	lines := strings.Split(strings.Join(sections, "\n"), "\n")
	lines = lines[:min(len(lines), max(1, m.height-1))]
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, max(1, m.width-1), "")
	}
	return strings.Join(lines, "\n")
}

func (s *UIService) runPrompt(value prompt) (string, error) {
	if !isTerminal(s.input) || !s.interactive {
		return "", ErrNotInteractive
	}
	result := make(chan inputResult, 1)
	current, err := s.acquire(func(m *model) tea.Cmd {
		if m.prompt != nil {
			result <- inputResult{err: ErrPromptBusy}
			return nil
		}
		m.prompt, m.result = value, result
		return value.Init()
	})
	if err != nil {
		return "", err
	}
	var answer inputResult
	select {
	case answer = <-result:
	case <-current.done:
		answer.err = current.err
		if answer.err == nil {
			answer.err = ErrCanceled
		}
	}
	finishErr := s.release(current, func(*model) tea.Cmd { return nil })
	if answer.err == nil {
		answer.err = finishErr
	}
	return answer.value, answer.err
}
