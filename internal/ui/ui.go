package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

var (
	ErrCanceled       = errors.New("input canceled")
	ErrNotInteractive = errors.New("interactive input requires a terminal")
	ErrPromptBusy     = errors.New("another input prompt is active")
)

type Option func(*UIService)

func WithInput(input io.Reader) Option        { return func(s *UIService) { s.input = input } }
func WithOutput(output io.Writer) Option      { return func(s *UIService) { s.output = output } }
func WithErrorOutput(output io.Writer) Option { return func(s *UIService) { s.errorOutput = output } }
func WithContext(ctx context.Context) Option  { return func(s *UIService) { s.ctx = ctx } }

type UIService struct {
	mu                  sync.Mutex
	input               io.Reader
	output, errorOutput io.Writer
	ctx                 context.Context
	cancel              context.CancelFunc
	interactive         bool
	session             *session
}

type session struct {
	program *tea.Program
	done    chan struct{}
	err     error
	refs    int
}

var _ domain.UI = (*UIService)(nil)

func NewUIService(options ...Option) *UIService {
	s := &UIService{input: os.Stdin, output: os.Stdout, errorOutput: os.Stderr, ctx: context.Background()}
	for _, option := range options {
		option(s)
	}
	s.ctx, s.cancel = context.WithCancel(s.ctx)
	s.interactive = isTerminal(s.output)
	return s
}

func (s *UIService) Context() context.Context { return s.ctx }

func isTerminal(value any) bool {
	file, ok := value.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(int(file.Fd()))
}

// All live components share this program and its renderer.
func (s *UIService) acquire(apply func(*model) tea.Cmd) (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	if s.session == nil {
		current := &session{done: make(chan struct{})}
		state := newModel()
		state.cancel = s.cancel
		current.program = tea.NewProgram(state, tea.WithInput(s.input), tea.WithOutput(s.output), tea.WithContext(s.ctx), tea.WithoutSignalHandler())
		s.session = current
		go func() {
			final, err := current.program.Run()
			if state, ok := final.(*model); ok && state.canceled {
				err = ErrCanceled
			}
			current.err = err
			close(current.done)
		}()
	}
	current := s.session
	if err := send(current, apply, false); err != nil {
		if current.refs == 0 {
			s.session = nil
		}
		return nil, err
	}
	current.refs++
	return current, nil
}

func send(current *session, apply func(*model) tea.Cmd, flush bool) error {
	ack := make(chan struct{})
	current.program.Send(request{apply: apply, ack: ack, flush: flush})
	select {
	case <-ack:
		return nil
	case <-current.done:
		if current.err != nil {
			return current.err
		}
		return ErrCanceled
	}
}

func (s *UIService) change(current *session, apply func(*model) tea.Cmd) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return send(current, apply, false)
}

func (s *UIService) release(current *session, apply func(*model) tea.Cmd) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := send(current, apply, true)
	current.refs--
	if current.refs == 0 {
		current.program.Quit()
		<-current.done
		if s.session == current {
			s.session = nil
		}
		if err == nil {
			err = current.err
		}
	}
	return err
}

func message(values ...any) string {
	var text string
	for _, value := range values {
		if value != nil {
			text += fmt.Sprint(value)
		}
	}
	return text
}
