package domain

import "fmt"

type ErrorType string

const (
	TypeNotFound      ErrorType = "NOT_FOUND"
	TypeValidation    ErrorType = "VALIDATION"
	TypeInternal      ErrorType = "INTERNAL"
	TypeAlreadyExists ErrorType = "ALREADY_EXISTS"
)

type AppError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func NewNotFoundError(msg string) error {
	return &AppError{
		Type:    TypeNotFound,
		Message: msg,
	}
}

func NewValidationError(msg string) error {
	return &AppError{
		Type:    TypeValidation,
		Message: msg,
	}
}

func NewAlreadyExistsError(msg string) error {
	return &AppError{
		Type:    TypeAlreadyExists,
		Message: msg,
	}
}
