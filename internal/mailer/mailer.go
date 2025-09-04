package mailer

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

// Mailer is a tiny interface to keep the rest of the app decoupled from the
// concrete SMTP implementation (easy to mock in tests).
type Mailer interface {
	// Send delivers an email to the given recipient(s).
	// The `to` argument may contain a single address or multiple addresses
	// separated by comma/semicolon/whitespace (e.g. "a@x.com, b@y.com").
	// If htmlBody is empty, a plain-text email is sent.
	// If textBody is empty while htmlBody is provided, a text alternative is omitted.
	Send(to, subject, htmlBody, textBody string) error
}

// SMTP encryption modes (normalized to lowercase).
const (
	encNone     = "none"
	encStartTLS = "starttls"
	encSSL      = "ssl"
)

// Config holds SMTP connection details and sensible timeouts.
type Config struct {
	Host               string
	Port               int
	Username           string
	Password           string
	From               string
	FromName           string
	Encryption         string        // "none" | "starttls" | "ssl" (case-insensitive)
	Timeout            time.Duration // both connect + send timeouts
	InsecureSkipVerify bool          // allow invalid cert (AVOID in production)
}

// SimpleMailer is an SMTP-backed mailer using go-simple-mail.
type SimpleMailer struct {
	cfg Config
}

// Ensure SimpleMailer implements Mailer at compile time.
var _ Mailer = (*SimpleMailer)(nil)

// NewFromEnv builds a SimpleMailer from environment variables.
// Supported variables:
//
//	SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD,
//	SMTP_FROM, SMTP_FROM_NAME, SMTP_ENCRYPTION (none|starttls|ssl),
//	SMTP_TIMEOUT (e.g. "10s"), SMTP_TLS_INSECURE_SKIP_VERIFY (true|false)
//
// Port selection fallback if SMTP_PORT is not set:
//   - ssl       -> 465
//   - starttls  -> 587
//   - otherwise -> 25
func NewFromEnv() (*SimpleMailer, error) {
	enc := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_ENCRYPTION")))

	// Port fallback logic described above.
	port := 0
	if v := strings.TrimSpace(os.Getenv("SMTP_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			port = p
		}
	}
	if port == 0 {
		switch enc {
		case encSSL:
			port = 465
		case encStartTLS:
			port = 587
		default:
			port = 25
		}
	}

	// Timeouts: keep the same value for connect/send to avoid surprises.
	timeout := 10 * time.Second
	if v := strings.TrimSpace(os.Getenv("SMTP_TIMEOUT")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}

	// Insecure TLS (handy for local dev with self-signed certs).
	insecure := false
	if v := strings.TrimSpace(os.Getenv("SMTP_TLS_INSECURE_SKIP_VERIFY")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			insecure = b
		}
	}

	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	username := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	password := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	fromName := strings.TrimSpace(os.Getenv("SMTP_FROM_NAME"))

	// If FROM is not specified, fall back to username (common requirement).
	if from == "" {
		from = username
	}

	cfg := Config{
		Host:               host,
		Port:               port,
		Username:           username,
		Password:           password,
		From:               from,
		FromName:           fromName,
		Encryption:         enc,
		Timeout:            timeout,
		InsecureSkipVerify: insecure,
	}

	// Minimal validation to fail early with a clear error.
	if cfg.Host == "" {
		return nil, fmt.Errorf("mailer config invalid: missing SMTP_HOST")
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("mailer config invalid: missing SMTP_USERNAME or SMTP_PASSWORD")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("mailer config invalid: missing SMTP_FROM (or SMTP_USERNAME)")
	}

	return &SimpleMailer{cfg: cfg}, nil
}

// Send opens a short-lived SMTP connection, composes the message, and sends it.
// A connection per send is perfectly fine for low/medium traffic; if you later
// need higher throughput, consider pooling or keeping a persistent connection.
func (m *SimpleMailer) Send(to, subject, htmlBody, textBody string) error {
	// Build SMTP client config.
	server := mail.NewSMTPClient()
	server.Host = m.cfg.Host
	server.Port = m.cfg.Port
	server.Username = m.cfg.Username
	server.Password = m.cfg.Password
	server.ConnectTimeout = m.cfg.Timeout
	server.SendTimeout = m.cfg.Timeout
	server.KeepAlive = false // short-lived connection per Send

	// TLS / STARTTLS settings based on configured encryption mode.
	switch strings.ToLower(strings.TrimSpace(m.cfg.Encryption)) {
	case encSSL: // Implicit TLS (port 465)
		server.Encryption = mail.EncryptionSSLTLS
		server.TLSConfig = &tls.Config{
			ServerName:         m.cfg.Host,
			InsecureSkipVerify: m.cfg.InsecureSkipVerify,
		}
	case encStartTLS: // STARTTLS upgrade (port 587)
		server.Encryption = mail.EncryptionSTARTTLS
		server.TLSConfig = &tls.Config{
			ServerName:         m.cfg.Host,
			InsecureSkipVerify: m.cfg.InsecureSkipVerify,
		}
	default: // Plain (port 25 or custom)
		server.Encryption = mail.EncryptionNone
	}

	// Establish the connection.
	smtpClient, err := server.Connect()
	if err != nil {
		return fmt.Errorf("smtp connect failed: %w", err)
	}

	// Compose message.
	msg := mail.NewMSG()

	// From: support optional friendly name.
	if m.cfg.FromName != "" {
		msg.SetFrom(fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.From))
	} else {
		msg.SetFrom(m.cfg.From)
	}

	// To: allow multiple recipients in a single string (comma/semicolon/space separated).
	recipients := splitEmails(to)
	if len(recipients) == 0 {
		return fmt.Errorf("no valid recipient address provided")
	}
	msg.AddTo(recipients...)

	// Subject: trim to avoid accidental leading/trailing whitespace.
	msg.SetSubject(strings.TrimSpace(subject))

	// Prefer HTML if provided, otherwise send plain text.
	htmlBody = strings.TrimSpace(htmlBody)
	textBody = strings.TrimSpace(textBody)

	if htmlBody != "" {
		msg.SetBody(mail.TextHTML, htmlBody)
		if textBody != "" {
			msg.AddAlternative(mail.TextPlain, textBody)
		}
	} else {
		// Fallback: ensure we send something useful even if textBody is empty.
		if textBody == "" {
			textBody = subject
		}
		msg.SetBody(mail.TextPlain, textBody)
	}

	// go-simple-mail exposes msg.Error for early validation issues.
	if msg.Error != nil {
		return fmt.Errorf("compose mail failed: %w", msg.Error)
	}

	// Send it.
	if err := msg.Send(smtpClient); err != nil {
		return fmt.Errorf("send mail failed: %w", err)
	}
	return nil
}

// splitEmails normalizes a string that may contain one or multiple addresses
// separated by common delimiters (comma, semicolon, or whitespace).
func splitEmails(s string) []string {
	// Replace semicolons with commas to simplify splitting.
	s = strings.ReplaceAll(s, ";", ",")
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
