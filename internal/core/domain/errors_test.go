package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error_WithWrappedErr(t *testing.T) {
	inner := errors.New("inner error")
	appErr := &AppError{
		Type:    TypeInternal,
		Message: "something failed",
		Err:     inner,
	}
	assert.Equal(t, "something failed: inner error", appErr.Error())
}

func TestAppError_Error_WithoutWrappedErr(t *testing.T) {
	appErr := &AppError{
		Type:    TypeInternal,
		Message: "something failed",
	}
	assert.Equal(t, "something failed", appErr.Error())
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("not found")
	var appErr *AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, TypeNotFound, appErr.Type)
	assert.Equal(t, "not found", appErr.Message)
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input")
	var appErr *AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, TypeValidation, appErr.Type)
	assert.Equal(t, "invalid input", appErr.Message)
}

func TestNewAlreadyExistsError(t *testing.T) {
	err := NewAlreadyExistsError("duplicate")
	var appErr *AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, TypeAlreadyExists, appErr.Type)
	assert.Equal(t, "duplicate", appErr.Message)
}
