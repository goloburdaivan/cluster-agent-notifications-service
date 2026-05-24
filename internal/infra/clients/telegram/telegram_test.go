package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/bottest-token/sendMessage")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithMaxRetries(1))

	err := client.SendMessage(context.Background(), "test-token", "12345", "Hello")
	require.NoError(t, err)
}

func TestSendMessage_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Description: "Unauthorized"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithMaxRetries(3))

	err := client.SendMessage(context.Background(), "bad-token", "12345", "Hello")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestSendMessage_RateLimit_Retries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Description: "Too Many Requests"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithMaxRetries(3),
		WithRateLimitBackoff(10*time.Millisecond),
	)

	err := client.SendMessage(context.Background(), "token", "12345", "Hello")
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestSendMessage_ServerError_Retries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Description: "Internal Server Error"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithMaxRetries(5),
		WithRateLimitBackoff(10*time.Millisecond),
	)

	err := client.SendMessage(context.Background(), "token", "12345", "Hello")
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestSendMessage_ClientError_NoRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Description: "Bad Request"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithMaxRetries(3))

	err := client.SendMessage(context.Background(), "token", "12345", "Hello")
	assert.Error(t, err)
	assert.Equal(t, 1, attempts)
}

func TestSendMessage_AllRetriesExhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Description: "Server Error"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithMaxRetries(2))

	err := client.SendMessage(context.Background(), "token", "12345", "Hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 2 retries")
}

func TestSendMessage_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithMaxRetries(1))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.SendMessage(ctx, "token", "12345", "Hello")
	assert.Error(t, err)
}

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient()
	tc := client.(*telegramClient)
	assert.Equal(t, defaultBaseURL, tc.baseURL)
	assert.Equal(t, defaultMaxRetries, tc.maxRetries)
	assert.Equal(t, defaultRateLimitBackoff, tc.rateLimitBackoff)
}

func TestNewClient_WithOptions(t *testing.T) {
	hc := &http.Client{Timeout: 30 * time.Second}
	client := NewClient(
		WithBaseURL("https://custom.api"),
		WithMaxRetries(5),
		WithRateLimitBackoff(10*time.Second),
		WithHTTPClient(hc),
	)

	tc := client.(*telegramClient)
	assert.Equal(t, "https://custom.api", tc.baseURL)
	assert.Equal(t, 5, tc.maxRetries)
	assert.Equal(t, 10*time.Second, tc.rateLimitBackoff)
	assert.Equal(t, hc, tc.httpClient)
}

func TestSendMessage_HTTPDoError(t *testing.T) {
	client := NewClient(
		WithBaseURL("http://127.0.0.1:1"),
		WithMaxRetries(1),
	)

	err := client.SendMessage(context.Background(), "token", "12345", "Hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 1 retries")
}
