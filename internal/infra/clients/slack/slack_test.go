package slack

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

func fastBackoff() BackoffConfig {
	return BackoffConfig{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
		MaxElapsedTime:  500 * time.Millisecond,
	}
}

func TestPostMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json; charset=utf-8", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithBackoffConfig(fastBackoff()))

	err := client.PostMessage(context.Background(), "test-token", "C123", "Hello")
	require.NoError(t, err)
}

func TestPostMessage_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Error: "invalid_auth"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithBackoffConfig(fastBackoff()))

	err := client.PostMessage(context.Background(), "bad-token", "C123", "Hello")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAuth)
}

func TestPostMessage_ChannelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Error: "channel_not_found"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithBackoffConfig(fastBackoff()))

	err := client.PostMessage(context.Background(), "token", "C999", "Hello")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrChannelMissing)
}

func TestPostMessage_RateLimit_Retries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithBackoffConfig(BackoffConfig{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			MaxElapsedTime:  2 * time.Second,
		}),
	)

	err := client.PostMessage(context.Background(), "token", "C123", "Hello")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, attempts, 2)
}

func TestPostMessage_ServerError_Retries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithBackoffConfig(BackoffConfig{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			MaxElapsedTime:  2 * time.Second,
		}),
	)

	err := client.PostMessage(context.Background(), "token", "C123", "Hello")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, attempts, 3)
}

func TestPostMessage_ClientError_NoPermanent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad_request"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithBackoffConfig(fastBackoff()))

	err := client.PostMessage(context.Background(), "token", "C123", "Hello")
	assert.Error(t, err)
}

func TestPostMessage_RetryableSlackError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Error: "timeout"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithBackoffConfig(BackoffConfig{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			MaxElapsedTime:  2 * time.Second,
		}),
	)

	err := client.PostMessage(context.Background(), "token", "C123", "Hello")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, attempts, 2)
}

func TestPostMessage_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithBackoffConfig(fastBackoff()))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := client.PostMessage(ctx, "token", "C123", "Hello")
	assert.Error(t, err)
}

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
}

func TestNewClient_WithHTTPClient(t *testing.T) {
	hc := &http.Client{Timeout: 30 * time.Second}
	client := NewClient(WithHTTPClient(hc))
	assert.NotNil(t, client)

	sc := client.(*slackClient)
	assert.Equal(t, hc, sc.httpClient)
}

func TestPostMessage_HTTPDoError(t *testing.T) {
	client := NewClient(
		WithBaseURL("http://127.0.0.1:1"),
		WithBackoffConfig(fastBackoff()),
	)

	err := client.PostMessage(context.Background(), "token", "C123", "Hello")
	assert.Error(t, err)
}

func TestPostMessage_InvalidResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithBackoffConfig(fastBackoff()))

	err := client.PostMessage(context.Background(), "token", "C123", "Hello")
	assert.Error(t, err)
}

func TestPostMessage_NonRetryableAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Error: "some_unknown_error"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithBackoffConfig(fastBackoff()))

	err := client.PostMessage(context.Background(), "token", "C123", "Hello")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnexpected)
}

func TestPostMessage_TokenExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Error: "token_expired"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithBackoffConfig(fastBackoff()))

	err := client.PostMessage(context.Background(), "token", "C123", "Hello")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAuth)
}
