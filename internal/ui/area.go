package ui

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type areaState struct {
	title, content string
	viewport       viewport.Model
}

func (a *areaState) resize(width, height int) {
	bottom := a.viewport.AtBottom()
	a.viewport.Width, a.viewport.Height = width, height
	a.viewport.SetContent(ansi.Hardwrap(strings.ReplaceAll(a.content, "\r\n", "\n"), width, true))
	if bottom {
		a.viewport.GotoBottom()
	}
}

func (a *areaState) View() string {
	if a.title == "" {
		return a.viewport.View()
	}
	return a.title + "\n" + a.viewport.View()
}

type areaHandle struct {
	mu      sync.Mutex
	service *UIService
	session *session
	state   *areaState
	closed  bool
	err     error
}

var _ domain.Area = (*areaHandle)(nil)

func (s *UIService) Area(values ...any) (domain.Area, error) {
	state := &areaState{title: message(values...), viewport: viewport.New(79, 20)}
	handle := &areaHandle{service: s, state: state}
	if !s.interactive {
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
		return handle, nil
	}
	current, err := s.acquire(func(m *model) tea.Cmd { m.areas = append(m.areas, state); return nil })
	if err != nil {
		return nil, err
	}
	handle.session = current
	return handle, nil
}

func (a *areaHandle) Update(values ...any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.err != nil {
		return
	}
	content := message(values...)
	if a.session == nil {
		a.state.content = content
		return
	}
	a.err = a.service.change(a.session, func(*model) tea.Cmd {
		a.state.content = content
		a.state.resize(a.state.viewport.Width, a.state.viewport.Height)
		return nil
	})
}

func (a *areaHandle) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return a.err
	}
	a.closed = true
	if a.session == nil {
		a.service.mu.Lock()
		defer a.service.mu.Unlock()
		if a.state.content != "" {
			_, a.err = fmt.Fprint(a.service.output, a.state.content)
			if a.err == nil && !strings.HasSuffix(a.state.content, "\n") {
				_, a.err = fmt.Fprintln(a.service.output)
			}
		}
		return a.err
	}
	err := a.service.release(a.session, func(m *model) tea.Cmd {
		m.areas = slices.DeleteFunc(m.areas, func(state *areaState) bool { return state == a.state })
		if a.state.content != "" {
			return tea.Println(strings.TrimSuffix(a.state.content, "\n"))
		}
		return nil
	})
	if a.err == nil {
		a.err = err
	}
	return a.err
}
