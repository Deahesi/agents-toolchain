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
	Update(message ...any)
}
