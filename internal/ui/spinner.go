package ui

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinnerState struct {
	spinner  spinner.Model
	text     string
	started  time.Time
	rounding time.Duration
}

func (s *spinnerState) View() string {
	text := s.spinner.View() + " " + s.text
	if s.rounding > 0 {
		text += fmt.Sprintf(" (%s)", time.Since(s.started).Round(s.rounding))
	}
	return text
}

type spinnerHandle struct {
	service *UIService
	session *session
	state   *spinnerState
	once    sync.Once
}

var _ domain.Spinner = (*spinnerHandle)(nil)

func (s *UIService) LogSpinner(values ...any) (domain.Spinner, error) {
	return s.startSpinner(0, values...)
}
func (s *UIService) LogSpinnerTimer(rounding time.Duration, values ...any) (domain.Spinner, error) {
	if rounding <= 0 {
		rounding = time.Second
	}
	return s.startSpinner(rounding, values...)
}

func (s *UIService) startSpinner(rounding time.Duration, values ...any) (domain.Spinner, error) {
	state := &spinnerState{spinner: spinner.New(spinner.WithSpinner(spinner.Dot)), text: message(values...), started: time.Now(), rounding: rounding}
	handle := &spinnerHandle{service: s, state: state}
	if !s.interactive {
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
		return handle, nil
	}
	current, err := s.acquire(func(m *model) tea.Cmd { m.spinners = append(m.spinners, state); return state.spinner.Tick })
	if err != nil {
		return nil, err
	}
	handle.session = current
	return handle, nil
}

func (s *spinnerHandle) Success(values ...any) { s.finish("success", values...) }
func (s *spinnerHandle) Warning(values ...any) { s.finish("warning", values...) }
func (s *spinnerHandle) Fail(values ...any)    { s.finish("error", values...) }

func (s *spinnerHandle) finish(kind string, values ...any) {
	s.once.Do(func() {
		text := message(values...)
		if len(values) == 0 {
			text = s.state.text
		}
		switch kind {
		case "success":
			text = successStyle.Render("✓ " + text)
		case "warning":
			text = warningStyle.Render("! " + text)
		case "error":
			text = errorStyle.Render("✗ " + text)
		}
		if s.session == nil {
			s.service.log(s.service.output, text)
			return
		}
		_ = s.service.release(s.session, func(m *model) tea.Cmd {
			m.spinners = slices.DeleteFunc(m.spinners, func(state *spinnerState) bool { return state == s.state })
			return tea.Println(text)
		})
	})
}
