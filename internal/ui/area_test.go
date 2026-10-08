package ui

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestAreaUpdateKeepsFragmentColors(t *testing.T) {
	var output bytes.Buffer
	s := NewUIService(WithInput(nil), WithOutput(&output))
	area, err := s.Area("Output")
	if err != nil {
		t.Fatal(err)
	}
	handle := area.(*areaHandle)
	renderer := lipgloss.NewRenderer(&output)
	renderer.SetColorProfile(termenv.TrueColor)
	handle.state.renderer = renderer
	area.Update("Answer", "#FFFFFF")
	area.Update(" tool ", "#FF0000")
	area.Update("continued", "2")
	area.Update(" plain", "")
	want := "\x1b[38;2;255;255;255mAnswer\x1b[0m" +
		"\x1b[38;2;255;0;0m tool \x1b[0m" +
		"\x1b[32mcontinued\x1b[0m plain"
	if got := handle.state.renderedContent(); got != want {
		t.Fatalf("fragment colors changed: got %q, want %q", got, want)
	}
	handle.state.resize(40, 1)
	if view := handle.state.View(); !strings.HasPrefix(view, "Output\n") || !strings.Contains(view, want) {
		t.Fatalf("unexpected view: %q", view)
	}
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "Answer tool continued plain\n"; got != want {
		t.Fatalf("redirected output: got %q, want %q", got, want)
	}
}

func TestAreaChangeColorOnlyAffectsFutureFragments(t *testing.T) {
	var output bytes.Buffer
	s := NewUIService(WithInput(nil), WithOutput(&output))
	area, err := s.Area()
	if err != nil {
		t.Fatal(err)
	}
	handle := area.(*areaHandle)
	renderer := lipgloss.NewRenderer(&output)
	renderer.SetColorProfile(termenv.TrueColor)
	handle.state.renderer = renderer
	if err := area.ChangeColor("2"); err != nil {
		t.Fatal(err)
	}
	area.Update("green")
	area.Update("plain", "")
	area.Update("red", "#FF0000")
	area.Update("green again")
	if err := area.ChangeColor(""); err != nil {
		t.Fatal(err)
	}
	area.Update("default")
	want := "\x1b[32mgreen\x1b[0mplain\x1b[38;2;255;0;0mred\x1b[0m" +
		"\x1b[32mgreen again\x1b[0mdefault"
	if got := handle.state.renderedContent(); got != want {
		t.Fatalf("default color leaked: got %q, want %q", got, want)
	}
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestAreaColorsSurviveWrappingAndScrolling(t *testing.T) {
	var output bytes.Buffer
	s := NewUIService(WithInput(nil), WithOutput(&output))
	area, err := s.Area()
	if err != nil {
		t.Fatal(err)
	}
	handle := area.(*areaHandle)
	renderer := lipgloss.NewRenderer(&output)
	renderer.SetColorProfile(termenv.TrueColor)
	handle.state.renderer = renderer
	area.Update("ABCDEFGHIJ", "2")
	area.Update("klmno", "#FF0000")
	handle.state.resize(5, 1)
	for _, row := range []struct {
		offset      int
		text, color string
	}{
		{0, "ABCDE", "\x1b[32m"},
		{1, "FGHIJ", "\x1b[32m"},
		{2, "klmno", "\x1b[38;2;255;0;0m"},
	} {
		handle.state.viewport.SetYOffset(row.offset)
		view := handle.state.View()
		if ansi.Strip(view) != row.text || !strings.Contains(view, row.color) {
			t.Fatalf("row %d lost its text or color: %q", row.offset, view)
		}
	}
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestAreaStopPreservesFragmentColors(t *testing.T) {
	var output bytes.Buffer
	s := NewUIService(WithInput(nil), WithOutput(&output))
	s.interactive = true
	area, err := s.Area()
	if err != nil {
		t.Fatal(err)
	}
	handle := area.(*areaHandle)
	renderer := lipgloss.NewRenderer(&output)
	renderer.SetColorProfile(termenv.TrueColor)
	if err := s.change(handle.session, func(*model) tea.Cmd {
		handle.state.renderer = renderer
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	area.Update("green", "2")
	area.Update("red\n", "#FF0000")
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
	want := "\x1b[32mgreen\x1b[0m\x1b[38;2;255;0;0mred\x1b[0m"
	if count := strings.Count(strings.ReplaceAll(output.String(), "\r\n", "\n"), want+"\n"); count != 1 {
		t.Fatalf("colored content printed %d times: %q", count, output.String())
	}
}
