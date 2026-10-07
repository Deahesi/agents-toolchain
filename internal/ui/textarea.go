package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type multilinePrompt struct {
	title string
	input textarea.Model
}

func (s *UIService) TextInputMultiline(messages ...string) (string, error) {
	input := textarea.New()
	input.CharLimit = 0
	input.ShowLineNumbers = false
	return s.runPrompt(&multilinePrompt{title: strings.Join(messages, " "), input: input})
}

func (p *multilinePrompt) Init() tea.Cmd { return p.input.Focus() }
func (p *multilinePrompt) resize(width, height int) {
	p.input.SetWidth(width)
	p.input.SetHeight(max(1, min(8, height-2)))
}
func (p *multilinePrompt) View() string {
	return p.title + "\n" + p.input.View() + "\nEnter: new line · Tab: submit · Esc: cancel"
}
func (p *multilinePrompt) Update(msg tea.Msg) (tea.Cmd, *inputResult) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "ctrl+enter":
			return nil, &inputResult{value: p.input.Value()}
		case "esc":
			return nil, &inputResult{err: ErrCanceled}
		}
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return cmd, nil
}
