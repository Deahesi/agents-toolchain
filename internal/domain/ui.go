package domain

type UI interface {
	LogStep(messages ...any)
	LogSuccess(messages ...any)
	LogError(messages ...any)
	LogSpinner(messages ...any) (Spinner, error)
}

type Spinner interface {
	Success(message ...any)
	Warning(message ...any)
	Fail(message ...any)
}
