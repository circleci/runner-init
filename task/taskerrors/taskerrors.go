package taskerrors

import "fmt"

type RetryableError struct {
	error
}

func RetryableErrorf(format string, a ...any) RetryableError {
	return RetryableError{fmt.Errorf(format, a...)}
}

// HandledError indicates the error has been managed (infra-fail or retry)
// and shouldn't be handled further up the call stack.
type HandledError struct {
	err error
}

func (h HandledError) Error() string {
	return fmt.Sprintf("handled: %v", h.err)
}

func NewHandledError(err error) HandledError {
	return HandledError{err}
}
