package domain

import (
	"time"
)

type UI interface {
	LogStep(messages ...any)
	LogSuccess(messages ...any)
	LogError(messages ...any)
	LogSpinner(message ...any) (Spinner, error)
	LogSpinnerTimer(roundingFactor time.Duration, message ...any) (Spinner, error)
	TextInput(message ...string) (string, error)
	TextInputMultiline(message ...string) (string, error)
	InteractiveSelect(message string, options ...string) (string, error)
	Area(message ...any) (Area, error)
}

type Spinner interface {
	Success(message ...any)
	Warning(message ...any)
	Fail(message ...any)
}

type Area interface {
	Stop() error
	// ChangeColor sets the default color for future fragments; empty resets it.
	ChangeColor(color string) error
	// Update appends text. An optional ANSI index or HEX color applies only to
	// this fragment; an explicit empty color uses the terminal's default color.
	Update(text string, color ...string)
}
