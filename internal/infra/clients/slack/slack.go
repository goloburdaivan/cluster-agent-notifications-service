package slack

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

	"github.com/cenkalti/backoff/v4"
)

var (
	ErrRateLimit      = errors.New("slack: rate limit exceeded")
	ErrUnexpected     = errors.New("slack: unexpected status")
	ErrAuth           = errors.New("slack: authentication failed")
	ErrChannelMissing = errors.New("slack: channel not found")
)

type Client interface {
	PostMessage(ctx context.Context, botToken string, channelID string, text string) error
}

type slackClient struct {
	baseURL    string
	httpClient *http.Client
	backoffCfg BackoffConfig
}

type BackoffConfig struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
}

const defaultBaseURL = "https://slack.com/api"

var defaultBackoffConfig = BackoffConfig{
	InitialInterval: 500 * time.Millisecond,
	MaxInterval:     30 * time.Second,
	MaxElapsedTime:  2 * time.Minute,
}

type Option func(*slackClient)

func WithBaseURL(url string) Option {
	return func(c *slackClient) { c.baseURL = url }
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *slackClient) { c.httpClient = hc }
}

func WithBackoffConfig(cfg BackoffConfig) Option {
	return func(c *slackClient) { c.backoffCfg = cfg }
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

	c := &slackClient{
		baseURL:    defaultBaseURL,
		backoffCfg: defaultBackoffConfig,
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
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func (c *slackClient) PostMessage(ctx context.Context, botToken string, channelID string, text string) error {
	url := fmt.Sprintf("%s/chat.postMessage", c.baseURL)

	payload := map[string]string{
		"channel": channelID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = c.backoffCfg.InitialInterval
	b.MaxInterval = c.backoffCfg.MaxInterval
	b.MaxElapsedTime = c.backoffCfg.MaxElapsedTime

	operation := func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return backoff.Permanent(fmt.Errorf("create request: %w", err))
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Authorization", "Bearer "+botToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("execute request: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("%w", ErrRateLimit)
		}

		if resp.StatusCode >= 500 {
			return fmt.Errorf("%w: %d", ErrUnexpected, resp.StatusCode)
		}

		if resp.StatusCode != http.StatusOK {
			return backoff.Permanent(fmt.Errorf("%w: %d %s", ErrUnexpected, resp.StatusCode, string(respBody)))
		}

		var apiResp apiResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return backoff.Permanent(fmt.Errorf("unmarshal response: %w", err))
		}

		if !apiResp.OK {
			if isRetryableSlackError(apiResp.Error) {
				return fmt.Errorf("%w: %s", ErrUnexpected, apiResp.Error)
			}
			if isAuthSlackError(apiResp.Error) {
				return backoff.Permanent(fmt.Errorf("%w: %s", ErrAuth, apiResp.Error))
			}
			if apiResp.Error == "channel_not_found" {
				return backoff.Permanent(fmt.Errorf("%w: %s", ErrChannelMissing, apiResp.Error))
			}
			return backoff.Permanent(fmt.Errorf("%w: %s", ErrUnexpected, apiResp.Error))
		}

		return nil
	}

	return backoff.Retry(operation, backoff.WithContext(b, ctx))
}

func isRetryableSlackError(code string) bool {
	switch code {
	case "timeout", "service_unavailable", "request_timeout", "fatal_error":
		return true
	}
	return false
}

func isAuthSlackError(code string) bool {
	switch code {
	case "invalid_auth", "not_authed", "account_inactive", "token_revoked", "token_expired":
		return true
	}
	return false
}
