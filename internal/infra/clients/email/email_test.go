package email

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMessage(t *testing.T) {
	msg := buildMessage("from@example.com", "to@example.com", "Test Subject", "<p>Body</p>")
	msgStr := string(msg)

	assert.Contains(t, msgStr, "From: from@example.com")
	assert.Contains(t, msgStr, "To: to@example.com")
	assert.Contains(t, msgStr, "Subject:")
	assert.Contains(t, msgStr, "MIME-Version: 1.0")
	assert.Contains(t, msgStr, "Content-Type: text/html")
	assert.Contains(t, msgStr, "<p>Body</p>")
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{"contains auth", "authentication failed", true},
		{"535 code", "535 Authentication Credentials Invalid", true},
		{"auth word", "auth required", true},
		{"not auth", "connection refused", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			assert.Equal(t, tt.expected, isAuthError(err))
		})
	}
}

func TestSend_InvalidConfig_EmptyHost(t *testing.T) {
	client := NewClient()

	err := client.Send(context.Background(), SMTPConfig{}, "to@example.com", "Subject", "Body")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestSend_InvalidConfig_EmptyTo(t *testing.T) {
	client := NewClient()

	cfg := SMTPConfig{Host: "smtp.example.com", Port: 587, From: "from@example.com"}
	err := client.Send(context.Background(), cfg, "", "Subject", "Body")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestSend_InvalidConfig_ZeroPort(t *testing.T) {
	client := NewClient()

	cfg := SMTPConfig{Host: "smtp.example.com", Port: 0, From: "from@example.com"}
	err := client.Send(context.Background(), cfg, "to@example.com", "Subject", "Body")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestSend_InvalidConfig_EmptyFrom(t *testing.T) {
	client := NewClient()

	cfg := SMTPConfig{Host: "smtp.example.com", Port: 587, From: ""}
	err := client.Send(context.Background(), cfg, "to@example.com", "Subject", "Body")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestSend_ContextCancelled(t *testing.T) {
	client := NewClient(
		WithDialTimeout(50*time.Millisecond),
		WithBackoffConfig(BackoffConfig{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			MaxElapsedTime:  200 * time.Millisecond,
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := SMTPConfig{Host: "smtp.example.com", Port: 587, From: "from@example.com"}
	err := client.Send(ctx, cfg, "to@example.com", "Subject", "Body")
	assert.Error(t, err)
}

func TestSend_ConnectionRefused(t *testing.T) {
	client := NewClient(
		WithDialTimeout(100*time.Millisecond),
		WithBackoffConfig(BackoffConfig{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			MaxElapsedTime:  200 * time.Millisecond,
		}),
	)

	cfg := SMTPConfig{Host: "127.0.0.1", Port: 1, From: "from@example.com"}
	err := client.Send(context.Background(), cfg, "to@example.com", "Subject", "Body")
	assert.Error(t, err)
}

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient()
	ec := client.(*emailClient)
	assert.Equal(t, 10*time.Second, ec.dialTimeout)
	assert.Equal(t, defaultBackoffConfig, ec.backoffCfg)
}

func TestNewClient_WithOptions(t *testing.T) {
	client := NewClient(
		WithDialTimeout(30*time.Second),
		WithBackoffConfig(BackoffConfig{
			InitialInterval: 2 * time.Second,
			MaxInterval:     10 * time.Second,
			MaxElapsedTime:  5 * time.Minute,
		}),
	)

	ec := client.(*emailClient)
	assert.Equal(t, 30*time.Second, ec.dialTimeout)
	assert.Equal(t, 2*time.Second, ec.backoffCfg.InitialInterval)
}

func TestBuildMessage_UTF8Subject(t *testing.T) {
	msg := buildMessage("a@b.com", "c@d.com", "Тест Юнікод", "body")
	msgStr := string(msg)
	require.Contains(t, msgStr, "Subject:")
	assert.Contains(t, msgStr, "body")
}

func startFakeSMTP(t *testing.T) (addr string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleSMTPConn(conn)
		}
	}()

	return ln.Addr().String()
}

func handleSMTPConn(conn net.Conn) {
	defer conn.Close()
	fmt.Fprintf(conn, "220 fake SMTP ready\r\n")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		cmd := strings.ToUpper(strings.SplitN(line, " ", 2)[0])

		switch cmd {
		case "EHLO", "HELO":
			fmt.Fprintf(conn, "250-hello\r\n250 OK\r\n")
		case "MAIL":
			fmt.Fprintf(conn, "250 OK\r\n")
		case "RCPT":
			fmt.Fprintf(conn, "250 OK\r\n")
		case "DATA":
			fmt.Fprintf(conn, "354 Go ahead\r\n")
			for scanner.Scan() {
				if scanner.Text() == "." {
					break
				}
			}
			fmt.Fprintf(conn, "250 OK\r\n")
		case "QUIT":
			fmt.Fprintf(conn, "221 Bye\r\n")
			return
		default:
			fmt.Fprintf(conn, "250 OK\r\n")
		}
	}
}

func TestSend_Success_FakeSMTP(t *testing.T) {
	addr := startFakeSMTP(t)
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	client := NewClient(
		WithDialTimeout(2*time.Second),
		WithBackoffConfig(BackoffConfig{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			MaxElapsedTime:  2 * time.Second,
		}),
	)

	cfg := SMTPConfig{Host: host, Port: port, From: "from@test.com"}
	err := client.Send(context.Background(), cfg, "to@test.com", "Test Subject", "<p>Hello</p>")
	require.NoError(t, err)
}

func TestSend_WithAuth_FakeSMTP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprintf(conn, "220 fake SMTP\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			cmd := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
			switch cmd {
			case "EHLO", "HELO":
				fmt.Fprintf(conn, "250-hello\r\n250 AUTH PLAIN\r\n")
			case "AUTH":
				fmt.Fprintf(conn, "235 OK\r\n")
			case "MAIL":
				fmt.Fprintf(conn, "250 OK\r\n")
			case "RCPT":
				fmt.Fprintf(conn, "250 OK\r\n")
			case "DATA":
				fmt.Fprintf(conn, "354 Go ahead\r\n")
				for scanner.Scan() {
					if scanner.Text() == "." {
						break
					}
				}
				fmt.Fprintf(conn, "250 OK\r\n")
			case "QUIT":
				fmt.Fprintf(conn, "221 Bye\r\n")
				return
			default:
				fmt.Fprintf(conn, "250 OK\r\n")
			}
		}
	}()

	addr := ln.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	client := NewClient(
		WithDialTimeout(2*time.Second),
		WithBackoffConfig(BackoffConfig{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			MaxElapsedTime:  2 * time.Second,
		}),
	)

	cfg := SMTPConfig{Host: host, Port: port, From: "from@test.com", Username: "user", Password: "pass"}
	err = client.Send(context.Background(), cfg, "to@test.com", "Subject", "Body")
	require.NoError(t, err)
}
