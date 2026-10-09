package ui

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	stepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

func (s *UIService) LogStep(values ...any) {
	s.log(s.output, stepStyle.Render("• "+message(values...)))
}
func (s *UIService) LogSuccess(values ...any) {
	s.log(s.output, successStyle.Render("✓ "+message(values...)))
}
func (s *UIService) LogError(values ...any) {
	s.log(s.errorOutput, errorStyle.Render("✗ "+message(values...)))
}

func (s *UIService) log(writer io.Writer, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		if err := send(s.session, func(*model) tea.Cmd { return tea.Println(text) }, true); err == nil {
			return
		}
	}
	fmt.Fprintln(writer, text)
}
