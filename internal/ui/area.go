package ui

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
)

type areaState struct {
	title, content string
	coloredContent string
	color          string
	viewport       viewport.Model
	renderer       *lipgloss.Renderer
}

func (a *areaState) resize(width, height int) {
	bottom := a.viewport.AtBottom()
	a.viewport.Width, a.viewport.Height = width, height
	content := ansi.Hardwrap(strings.ReplaceAll(a.renderedContent(), "\r\n", "\n"), width, true)
	// Each viewport line must restore its own style when scrolled into view.
	var wrapped strings.Builder
	pen := cellbuf.NewPenWriter(&wrapped)
	_, _ = pen.Write([]byte(content))
	_ = pen.Close()
	a.viewport.SetContent(wrapped.String())
	if bottom {
		a.viewport.GotoBottom()
	}
}

func (a *areaState) View() string {
	text := a.viewport.View()
	if a.title != "" {
		text = a.title + "\n" + text
	}
	return text
}

func (a *areaState) renderedContent() string {
	if a.coloredContent == "" {
		return a.content
	}
	return a.coloredContent
}

func (a *areaState) append(text string, colors []string) {
	color := a.color
	if len(colors) > 0 {
		color = colors[0]
	}
	a.content += text
	if color == "" {
		a.coloredContent += text
		return
	}
	profile := a.renderer.ColorProfile()
	a.coloredContent += profile.String(text).Foreground(profile.Color(color)).String()
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
	state := &areaState{title: message(values...), viewport: viewport.New(79, 20), renderer: lipgloss.DefaultRenderer()}
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

func (a *areaHandle) Update(text string, color ...string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.err != nil {
		return
	}
	if text == "" {
		return
	}
	if a.session == nil {
		a.state.append(text, color)
		return
	}
	a.err = a.service.change(a.session, func(*model) tea.Cmd {
		a.state.append(text, color)
		a.state.resize(a.state.viewport.Width, a.state.viewport.Height)
		return nil
	})
}

func (a *areaHandle) ChangeColor(color string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.err != nil {
		return a.err
	}
	apply := func(*model) tea.Cmd {
		a.state.color = color
		return nil
	}
	if a.session == nil {
		apply(nil)
		return nil
	}
	a.err = a.service.change(a.session, apply)
	return a.err
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
			content := a.state.renderedContent()
			if strings.HasSuffix(a.state.content, "\n") {
				// A fragment's reset sequence may follow its trailing newline.
				index := strings.LastIndexByte(content, '\n')
				content = content[:index] + content[index+1:]
			}
			return tea.Println(content)
		}
		return nil
	})
	if a.err == nil {
		a.err = err
	}
	return a.err
}
