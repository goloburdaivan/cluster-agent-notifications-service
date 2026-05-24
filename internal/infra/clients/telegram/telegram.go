package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var (
	ErrRateLimit        = errors.New("telegram: rate limit exceeded")
	ErrUnexpectedStatus = errors.New("telegram: unexpected status code")
	ErrUnauthorized     = errors.New("telegram: invalid token")
)

type Client interface {
	SendMessage(ctx context.Context, botToken string, chatID string, text string) error
}

type telegramClient struct {
	baseURL          string
	httpClient       *http.Client
	maxRetries       int
	rateLimitBackoff time.Duration
}

const (
	defaultBaseURL          = "https://api.telegram.org"
	defaultMaxRetries       = 3
	defaultRateLimitBackoff = 5 * time.Second
)

type Option func(*telegramClient)

func WithBaseURL(url string) Option {
	return func(c *telegramClient) {
		c.baseURL = url
	}
}

func WithMaxRetries(n int) Option {
	return func(c *telegramClient) {
		c.maxRetries = n
	}
}

func WithRateLimitBackoff(d time.Duration) Option {
	return func(c *telegramClient) {
		c.rateLimitBackoff = d
	}
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *telegramClient) {
		c.httpClient = hc
	}
}

func NewClient(opts ...Option) Client {
	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	c := &telegramClient{
		baseURL:          defaultBaseURL,
		maxRetries:       defaultMaxRetries,
		rateLimitBackoff: defaultRateLimitBackoff,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: t,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func (c *telegramClient) SendMessage(
	ctx context.Context,
	botToken string,
	chatID string,
	text string,
) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, botToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	var lastErr error

	for i := 0; i < c.maxRetries; i++ {
		lastErr = c.doSendAttempt(ctx, url, body)
		if lastErr == nil {
			return nil
		}

		if errors.Is(lastErr, ErrUnauthorized) {
			return lastErr
		}

		waitDuration := c.retryDelay(ctx, lastErr, i)
		if waitErr := c.waitBackoff(ctx, waitDuration); waitErr != nil {
			return waitErr
		}

		if isClientError(lastErr) {
			break
		}
	}

	return fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
}

func (c *telegramClient) doSendAttempt(ctx context.Context, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	_, apiErr := c.readResponse(resp)
	return apiErr
}

func (c *telegramClient) retryDelay(ctx context.Context, err error, attempt int) time.Duration {
	if errors.Is(err, ErrRateLimit) {
		return c.rateLimitBackoff
	}
	return time.Duration(1<<attempt) * time.Second
}

type clientError struct {
	err error
}

func (e *clientError) Error() string { return e.err.Error() }
func (e *clientError) Unwrap() error { return e.err }

func isClientError(err error) bool {
	var ce *clientError
	return errors.As(err, &ce)
}

func (c *telegramClient) readResponse(resp *http.Response) (statusCode int, err error) {
	defer resp.Body.Close() //nolint:errcheck

	statusCode = resp.StatusCode
	respBody, _ := io.ReadAll(resp.Body)

	var apiResp apiResponse
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &apiResp)
	}

	switch statusCode {
	case http.StatusOK:
		return statusCode, nil
	case http.StatusTooManyRequests:
		return statusCode, fmt.Errorf("%w: %s", ErrRateLimit, apiResp.Description)
	case http.StatusUnauthorized:
		return statusCode, fmt.Errorf("%w: %s", ErrUnauthorized, apiResp.Description)
	default:
		if statusCode >= 400 && statusCode < 500 {
			return statusCode, &clientError{err: fmt.Errorf("%w: %d %s", ErrUnexpectedStatus, statusCode, apiResp.Description)}
		}
		return statusCode, fmt.Errorf("%w: %d %s", ErrUnexpectedStatus, statusCode, apiResp.Description)
	}
}

func (c *telegramClient) waitBackoff(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
