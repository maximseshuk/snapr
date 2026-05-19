package notifier

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/maximseshuk/snapr/internal/config"
)

type Email struct {
	cfg config.NotifierConfig
}

func NewEmail(c config.NotifierConfig) (*Email, error) {
	return &Email{cfg: c}, nil
}

func (e *Email) GetType() string { return "email" }

func (e *Email) Notify(ctx context.Context, ev Event) error {
	msg := []byte(buildMIME(e.cfg.From, e.cfg.To, ev))

	addr := fmt.Sprintf("%s:%d", e.cfg.SMTPHost, e.cfg.SMTPPort)

	dialer := &net.Dialer{Timeout: 30 * time.Second}
	type result struct{ err error }
	done := make(chan result, 1)

	// net/smtp has no context-aware API: on cancellation we return early while
	// the SMTP exchange keeps running until the dialer/server timeout.
	go func() {
		done <- result{err: e.send(dialer, addr, msg)}
	}()

	select {
	case r := <-done:
		return r.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Email) send(dialer *net.Dialer, addr string, msg []byte) error {
	var conn net.Conn
	var err error

	if e.cfg.UseTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: e.cfg.SMTPHost})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, e.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Quit() }()

	if !e.cfg.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: e.cfg.SMTPHost}); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	if e.cfg.SMTPUser != "" {
		auth := smtp.PlainAuth("", e.cfg.SMTPUser, e.cfg.SMTPPass, e.cfg.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(e.cfg.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, to := range e.cfg.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", to, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return nil
}

func buildMIME(from string, to []string, ev Event) string {
	status, emoji := "OK", "✅"
	if !ev.Success {
		status, emoji = "FAILED", "❌"
	}

	subject := fmt.Sprintf("%s [snapr] %s: %s", emoji, status, ev.JobName)
	preview := fmt.Sprintf("%s in %s", status, ev.Duration)

	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", preview)
	fmt.Fprintf(&body, "%s Backup %s\n", emoji, status)
	body.WriteString(strings.Repeat("─", 32) + "\n")
	fmt.Fprintf(&body, "Job:      %s\n", ev.JobName)
	fmt.Fprintf(&body, "Duration: %s\n", ev.Duration)
	if ev.Error != "" {
		fmt.Fprintf(&body, "\nError:\n%s\n", ev.Error)
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", strings.Join(to, ", "))
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	msg.WriteString(base64.StdEncoding.EncodeToString([]byte(body.String())))
	return msg.String()
}
