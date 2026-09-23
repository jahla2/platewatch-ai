package application

import "fmt"

type ValidationError struct {
	message string
}

func NewValidationError(message string) error {
	return ValidationError{message: message}
}

func (e ValidationError) Error() string {
	return e.message
}

func WrapValidationError(format string, args ...any) error {
	return ValidationError{message: fmt.Sprintf(format, args...)}
}
