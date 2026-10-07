package ui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestLiveComponentsShareRendererAndFlushFullResponse(t *testing.T) {
	var output bytes.Buffer
	s := NewUIService(WithInput(nil), WithOutput(&output))
	s.interactive = true // Exercise the real program without requiring a console.
	loading, err := s.LogSpinnerTimer(time.Second, "Generating response")
	if err != nil {
		t.Fatal(err)
	}
	area, err := s.Area("Output")
	if err != nil {
		t.Fatal(err)
	}
	current := s.session
	if current.refs != 2 {
		t.Fatalf("got %d live components, want 2", current.refs)
	}
	full := strings.Repeat("Длинный абзац с Unicode 🙂 и переносами. ", 100) + "\n\nПоследняя строка.\n"
	area.Update(full)
	s.LogStep("A log while both components are active")
	loading.Success("Completed")
	loading.Fail("Must not appear")
	if s.session != current || current.refs != 1 {
		t.Fatal("spinner stopped the area's renderer")
	}
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
	if s.session != nil {
		t.Fatal("renderer was not released")
	}
	if !strings.Contains(output.String(), "Completed") {
		t.Fatal("completion log was lost at shutdown")
	}
	if strings.Contains(output.String(), "Must not appear") {
		t.Fatal("spinner completed twice")
	}
	normalized := strings.ReplaceAll(output.String(), "\r\n", "\n")
	if count := strings.Count(normalized, strings.TrimSuffix(full, "\n")); count != 1 {
		t.Fatalf("full response printed %d times, want 1", count)
	}
	// A subsequent component must be able to start a new program.
	next, err := s.LogSpinner("Next step")
	if err != nil {
		t.Fatal(err)
	}
	next.Success()
}

func TestViewportFitsResizedTerminalAndPreservesContent(t *testing.T) {
	full := strings.Repeat("Очень длинный русский текст 你好 🙂 e\u0301\t ", 100)
	m := newModel()
	m.spinners = []*spinnerState{{spinner: spinner.New(), text: "Generating response", started: time.Now(), rounding: time.Second}}
	area := &areaState{title: "Output", content: full, viewport: viewport.New(79, 20)}
	m.areas = []*areaState{area}
	for _, size := range []tea.WindowSizeMsg{{Width: 40, Height: 10}, {Width: 15, Height: 6}, {Width: 100, Height: 30}} {
		m.Update(size)
		lines := strings.Split(m.View(), "\n")
		if len(lines) >= size.Height {
			t.Fatalf("view has %d rows, terminal has %d", len(lines), size.Height)
		}
		for _, line := range lines {
			if ansi.StringWidth(line) >= size.Width {
				t.Fatalf("line would wrap beyond %d columns: %q", size.Width, line)
			}
		}
		if area.content != full {
			t.Fatal("resizing changed response content")
		}
		if !area.viewport.AtBottom() {
			t.Fatal("resizing lost the live tail")
		}
	}
}

func TestInputComponentsSubmitAndCancel(t *testing.T) {
	input := &textPrompt{input: textinput.New()}
	input.Init()
	input.resize(30, 10)
	value := strings.Repeat("Текст ", 80)
	input.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value), Paste: true})
	_, result := input.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if result == nil || result.value != value {
		t.Fatal("single-line input truncated submitted text")
	}
	_, result = input.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if result == nil || !errors.Is(result.err, ErrCanceled) {
		t.Fatal("Escape did not cancel input")
	}

	multiline := &multilinePrompt{input: textarea.New()}
	multiline.input.CharLimit = 0
	multiline.Init()
	multiline.resize(40, 12)
	multiline.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Первый абзац"), Paste: true})
	if _, result := multiline.Update(tea.KeyMsg{Type: tea.KeyEnter}); result != nil {
		t.Fatal("Enter submitted multiline input")
	}
	multiline.Update(tea.KeyMsg{Type: tea.KeyEnter})
	multiline.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Второй абзац"), Paste: true})
	_, result = multiline.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if result == nil || result.value != "Первый абзац\n\nВторой абзац" {
		t.Fatalf("multiline paragraphs were not preserved: %+v", result)
	}

	selection := &selectPrompt{list: list.New([]list.Item{optionItem("openai"), optionItem("ollama")}, list.NewDefaultDelegate(), 40, 10)}
	selection.resize(79, 24)
	if selection.list.Height() >= 23 {
		t.Fatal("short selection occupies the entire terminal")
	}
	selection.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, result = selection.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if result == nil || result.value != "ollama" {
		t.Fatal("selected option was not returned")
	}
}

func TestRedirectedAreaWritesLatestContentOnce(t *testing.T) {
	var output bytes.Buffer
	s := NewUIService(WithInput(nil), WithOutput(&output))
	area, err := s.Area("Output")
	if err != nil {
		t.Fatal(err)
	}
	area.Update("partial")
	area.Update("\nFull response\n\nSecond paragraph\r\n")
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
	area.Update("late update")
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "\nFull response\n\nSecond paragraph\r\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := s.TextInput("Input"); !errors.Is(err, ErrNotInteractive) {
		t.Fatalf("noninteractive input error: %v", err)
	}
	if _, err := s.InteractiveSelect("Select"); err == nil {
		t.Fatal("empty selection accepted")
	}
}

func TestCancellationReleasesLiveProgram(t *testing.T) {
	var output bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	s := NewUIService(WithInput(nil), WithOutput(&output), WithContext(ctx))
	s.interactive = true
	area, err := s.Area("Output")
	if err != nil {
		t.Fatal(err)
	}
	current := s.session
	cancel()
	select {
	case <-current.done:
	case <-time.After(5 * time.Second):
		t.Fatal("renderer ignored cancellation")
	}
	if err := area.Stop(); !errors.Is(err, context.Canceled) {
		t.Fatalf("Stop error: %v", err)
	}
	if s.session != nil {
		t.Fatal("canceled renderer not released")
	}
}

func TestConcurrentAreaUpdatesAndLogging(t *testing.T) {
	var output bytes.Buffer
	s := NewUIService(WithInput(nil), WithOutput(&output))
	s.interactive = true
	area, err := s.Area("Output")
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 3 {
		group.Go(func() {
			for range 10 {
				area.Update("Updated response")
				s.LogStep("Progress")
			}
		})
	}
	group.Wait()
	area.Update("Final response")
	if err := area.Stop(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Final response") {
		t.Fatal("final update lost")
	}
}
