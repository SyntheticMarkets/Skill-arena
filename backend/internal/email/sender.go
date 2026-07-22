package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"skill-arena/internal/config"
)

type Message struct {
	To       string
	Subject  string
	Template string
	Link     string
}

type Sender interface {
	Send(context.Context, Message) error
	Health(context.Context) error
}

type OutboxSender struct {
	Root string
}

func (s OutboxSender) Send(ctx context.Context, message Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Root == "" {
		return errors.New("development email outbox directory is required")
	}
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return err
	}
	name := fmt.Sprintf("%d-%s.eml", time.Now().UTC().UnixNano(), sanitize(message.To))
	return os.WriteFile(filepath.Join(s.Root, name), buildMIME("Skill Arena <no-reply@localhost>", message), 0o600)
}

func (s OutboxSender) Health(context.Context) error {
	if s.Root == "" {
		return errors.New("development email outbox directory is required")
	}
	return os.MkdirAll(s.Root, 0o755)
}

type SMTPSender struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func (s SMTPSender) Health(ctx context.Context) error {
	if s.Host == "" || s.Port <= 0 || s.From == "" {
		return errors.New("SMTP host, port, and from address are required")
	}
	if _, err := mail.ParseAddress(s.From); err != nil {
		return fmt.Errorf("invalid SMTP from address: %w", err)
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.Host, strconv.Itoa(s.Port)))
	if err != nil {
		return err
	}
	return conn.Close()
}

func (s SMTPSender) Send(ctx context.Context, message Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	from, err := mail.ParseAddress(s.From)
	if err != nil {
		return err
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil {
		return err
	}
	address := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	var client *smtp.Client
	if s.Port == 465 {
		dialer := net.Dialer{Timeout: 10 * time.Second}
		conn, err := tls.DialWithDialer(&dialer, "tcp", address, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.Host})
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, s.Host)
		if err != nil {
			_ = conn.Close()
			return err
		}
	} else {
		client, err = smtp.Dial(address)
		if err != nil {
			return err
		}
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return errors.New("SMTP server does not offer STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.Host}); err != nil {
			_ = client.Close()
			return err
		}
	}
	defer client.Close()
	if s.User != "" {
		if err := client.Auth(smtp.PlainAuth("", s.User, s.Password, s.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(buildMIME(s.From, message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func NewSender(settings config.EmailSettings, dataDir string) Sender {
	if settings.OutboxOnly {
		return OutboxSender{Root: filepath.Join(dataDir, "email_outbox")}
	}
	return SMTPSender{Host: settings.SMTPHost, Port: settings.SMTPPort, User: settings.SMTPUser, Password: settings.SMTPPass, From: settings.From}
}

func buildMIME(from string, message Message) []byte {
	title := "Continue to Skill Arena"
	intro := "Use the secure link below to continue."
	if message.Template == "email_verification" {
		title = "Verify your email"
		intro = "Confirm your email address to protect your competitor identity."
	} else if message.Template == "password_reset" {
		title = "Reset your password"
		intro = "A password reset was requested for your Skill Arena account."
	}
	body := "<!doctype html><html><body style=\"font-family:Arial,sans-serif;background:#080b10;color:#f5f7fb;padding:32px\">" +
		"<main style=\"max-width:560px;margin:auto\"><p style=\"color:#53e0bd\">SKILL ARENA</p><h1>" + html.EscapeString(title) + "</h1><p>" + html.EscapeString(intro) + "</p>" +
		"<p><a style=\"display:inline-block;background:#f4ff62;color:#080b10;padding:14px 20px;text-decoration:none;font-weight:bold\" href=\"" + html.EscapeString(message.Link) + "\">Continue securely</a></p>" +
		"<p style=\"color:#9aa4b2\">If you did not request this, you can ignore this email.</p></main></body></html>"
	headers := []string{
		"From: " + from,
		"To: " + message.To,
		"Subject: " + message.Subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body + "\r\n")
}

func sanitize(value string) string {
	value = strings.ToLower(value)
	var result strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteByte('_')
		}
	}
	return strings.Trim(result.String(), "_")
}
