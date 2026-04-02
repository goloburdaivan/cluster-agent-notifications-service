package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"notifications-service/internal/core/domain"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type ValidationErrorResponse struct {
	Error   string       `json:"error"`
	Details []FieldError `json:"details"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			var appErr *domain.AppError
			if errors.As(err, &appErr) {
				var statusCode int

				switch appErr.Type {
				case domain.TypeNotFound:
					statusCode = http.StatusNotFound
				case domain.TypeValidation:
					statusCode = http.StatusBadRequest
				case domain.TypeAlreadyExists:
					statusCode = http.StatusConflict
				default:
					statusCode = http.StatusInternalServerError
				}

				c.JSON(statusCode, ErrorResponse{Error: appErr.Message})
				return
			}

			var validationErrs validator.ValidationErrors
			if errors.As(err, &validationErrs) {
				var details []FieldError
				for _, fieldErr := range validationErrs {
					details = append(details, FieldError{
						Field:   fieldErr.Field(),
						Message: "failed on the '" + fieldErr.Tag() + "' tag",
					})
				}

				c.JSON(http.StatusBadRequest, ValidationErrorResponse{
					Error:   "validation failed",
					Details: details,
				})
				return
			}

			slog.Error("unknown error from server", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		}
	}
}
