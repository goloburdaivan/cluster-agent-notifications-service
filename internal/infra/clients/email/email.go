package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
)

var (
	ErrAuth          = errors.New("email: authentication failed")
	ErrSend          = errors.New("email: send failed")
	ErrInvalidConfig = errors.New("email: invalid configuration")
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type Client interface {
	Send(ctx context.Context, cfg SMTPConfig, to string, subject string, body string) error
}

type emailClient struct {
	backoffCfg  BackoffConfig
	dialTimeout time.Duration
}

type BackoffConfig struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
}

var defaultBackoffConfig = BackoffConfig{
	InitialInterval: 1 * time.Second,
	MaxInterval:     5 * time.Second,
	MaxElapsedTime:  2 * time.Minute,
}

type Option func(*emailClient)

func WithBackoffConfig(cfg BackoffConfig) Option {
	return func(c *emailClient) { c.backoffCfg = cfg }
}

func WithDialTimeout(d time.Duration) Option {
	return func(c *emailClient) { c.dialTimeout = d }
}

func NewClient(opts ...Option) Client {
	c := &emailClient{
		backoffCfg:  defaultBackoffConfig,
		dialTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *emailClient) Send(ctx context.Context, cfg SMTPConfig, to string, subject string, body string) error {
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" || len(to) == 0 {
		return fmt.Errorf("%w: host, port, from, and to are required", ErrInvalidConfig)
	}

	msg := buildMessage(cfg.From, to, subject, body)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = c.backoffCfg.InitialInterval
	b.MaxInterval = c.backoffCfg.MaxInterval
	b.MaxElapsedTime = c.backoffCfg.MaxElapsedTime

	operation := func() error {
		if err := ctx.Err(); err != nil {
			return backoff.Permanent(err)
		}

		err := c.sendMail(addr, cfg, to, msg)
		if err != nil {
			if isAuthError(err) {
				return backoff.Permanent(fmt.Errorf("%w: %v", ErrAuth, err))
			}
			return fmt.Errorf("%w: %v", ErrSend, err)
		}

		return nil
	}

	return backoff.Retry(operation, backoff.WithContext(b, ctx))
}

func (c *emailClient) sendMail(addr string, cfg SMTPConfig, to string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)

	conn, err := net.DialTimeout("tcp", addr, c.dialTimeout)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close() //nolint:errcheck,gosec
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	if ok, _ := client.Extension("STARTTLS"); ok {
		if tlsErr := client.StartTLS(&tls.Config{ServerName: host}); tlsErr != nil {
			return fmt.Errorf("STARTTLS: %w", tlsErr)
		}
	}

	if cfg.Username != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, host)
		if authErr := client.Auth(auth); authErr != nil {
			return fmt.Errorf("auth: %w", authErr)
		}
	}

	if mailErr := client.Mail(cfg.From); mailErr != nil {
		return fmt.Errorf("mail from: %w", mailErr)
	}

	if rcptErr := client.Rcpt(to); rcptErr != nil {
		return fmt.Errorf("rcpt to %s: %w", to, rcptErr)
	}

	w, dataErr := client.Data()
	if dataErr != nil {
		return fmt.Errorf("data: %w", dataErr)
	}

	if _, writeErr := w.Write(msg); writeErr != nil {
		return fmt.Errorf("write body: %w", writeErr)
	}

	if closeErr := w.Close(); closeErr != nil {
		return fmt.Errorf("close data: %w", closeErr)
	}

	return client.Quit()
}

func buildMessage(from string, to string, subject string, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}

func isAuthError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "auth") ||
		strings.Contains(s, "535") ||
		strings.Contains(s, "authentication")
}
