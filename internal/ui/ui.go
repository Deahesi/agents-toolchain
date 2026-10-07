package ui

import (
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/pterm/pterm"
)

// LogStep(messages ...any)
// LogSuccess(messages ...any)
// LogError(messages ...any)

type UIService struct{}

func NewUIService() *UIService {
	return &UIService{}
}

func (s *UIService) LogStep(message ...any) {
	endColor := pterm.NewRGB(255, 255, 255)
	endColor.Println(message...)
}

func (s *UIService) LogSuccess(message ...any) {
	pterm.Success.Println(message...)
}

func (s *UIService) LogError(message ...any) {
	pterm.Error.Println(message...)
}
func (s *UIService) LogSpinner(message ...any) (domain.Spinner, error) {
	spinnerInfo, err := pterm.DefaultSpinner.Start(message...)

	return spinnerInfo, err
}
