package ui

import (
	"errors"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type optionItem string

func (i optionItem) Title() string       { return string(i) }
func (i optionItem) Description() string { return "" }
func (i optionItem) FilterValue() string { return string(i) }

type selectPrompt struct{ list list.Model }

func (s *UIService) InteractiveSelect(title string, options ...string) (string, error) {
	if len(options) == 0 {
		return "", errors.New("select requires at least one option")
	}
	items := make([]list.Item, len(options))
	for i, value := range options {
		items[i] = optionItem(value)
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	model := list.New(items, delegate, 79, min(len(options)+5, 20))
	model.Title = title
	model.SetFilteringEnabled(false)
	model.SetShowStatusBar(false)
	model.DisableQuitKeybindings()
	return s.runPrompt(&selectPrompt{list: model})
}

func (p *selectPrompt) Init() tea.Cmd { return nil }
func (p *selectPrompt) resize(width, height int) {
	p.list.SetSize(width, max(1, min(height-1, len(p.list.Items())+7)))
}
func (p *selectPrompt) View() string { return p.list.View() + "\nEnter: select · Esc: cancel" }
func (p *selectPrompt) Update(msg tea.Msg) (tea.Cmd, *inputResult) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			if item, ok := p.list.SelectedItem().(optionItem); ok {
				return nil, &inputResult{value: string(item)}
			}
		case "esc":
			return nil, &inputResult{err: ErrCanceled}
		}
	}
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return cmd, nil
}
