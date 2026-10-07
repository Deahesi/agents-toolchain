package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type textPrompt struct {
	title string
	input textinput.Model
}

func (s *UIService) TextInput(messages ...string) (string, error) {
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 0
	return s.runPrompt(&textPrompt{title: strings.Join(messages, " "), input: input})
}

func (p *textPrompt) Init() tea.Cmd       { return p.input.Focus() }
func (p *textPrompt) resize(width, _ int) { p.input.Width = max(1, width-2) }
func (p *textPrompt) View() string {
	return p.title + "\n" + p.input.View() + "\nEnter: submit · Esc: cancel"
}
func (p *textPrompt) Update(msg tea.Msg) (tea.Cmd, *inputResult) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			return nil, &inputResult{value: p.input.Value()}
		case "esc":
			return nil, &inputResult{err: ErrCanceled}
		}
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return cmd, nil
}
