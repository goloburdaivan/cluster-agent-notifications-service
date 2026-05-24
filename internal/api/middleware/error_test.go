package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/core/domain"
)

func setupRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/test", handler)
	return r
}

func TestErrorHandler_NoError(t *testing.T) {
	r := setupRouter(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestErrorHandler_NotFoundError(t *testing.T) {
	r := setupRouter(func(c *gin.Context) {
		_ = c.Error(domain.NewNotFoundError("resource not found"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "resource not found", resp.Error)
}

func TestErrorHandler_ValidationError(t *testing.T) {
	r := setupRouter(func(c *gin.Context) {
		_ = c.Error(domain.NewValidationError("invalid input"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestErrorHandler_AlreadyExistsError(t *testing.T) {
	r := setupRouter(func(c *gin.Context) {
		_ = c.Error(domain.NewAlreadyExistsError("duplicate"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestErrorHandler_InternalAppError(t *testing.T) {
	r := setupRouter(func(c *gin.Context) {
		_ = c.Error(&domain.AppError{Type: domain.TypeInternal, Message: "internal"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestErrorHandler_UnknownError(t *testing.T) {
	r := setupRouter(func(c *gin.Context) {
		_ = c.Error(errors.New("unknown error"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "internal server error", resp.Error)
}

func TestErrorHandler_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())

	type testReq struct {
		Name string `json:"name" binding:"required"`
	}

	r.POST("/test", func(c *gin.Context) {
		var req testReq
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ValidationErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "validation failed", resp.Error)
	assert.NotEmpty(t, resp.Details)
}
