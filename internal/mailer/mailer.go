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

// Mailer je mali interfejs kako bi kasnije bilo lako zameniti implementaciju (mock u testovima itd.)
type Mailer interface {
	Send(to, subject, htmlBody, textBody string) error
}

type Config struct {
	Host               string
	Port               int
	Username           string
	Password           string
	From               string
	FromName           string
	Encryption         string // "none" | "starttls" | "ssl"
	Timeout            time.Duration
	InsecureSkipVerify bool
}

type SimpleMailer struct {
	cfg Config
}

// NewFromEnv konstruiše mailer iz .env / okruženja.
// Podržani ključevi:
//
//	SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD,
//	SMTP_FROM, SMTP_FROM_NAME, SMTP_ENCRYPTION (none|starttls|ssl),
//	SMTP_TIMEOUT (npr "10s"), SMTP_TLS_INSECURE_SKIP_VERIFY (true|false)
func NewFromEnv() (*SimpleMailer, error) {
	enc := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_ENCRYPTION")))

	// Port fallback logika:
	// - ako SMTP_PORT postoji, koristi njega
	// - ako nije, a encryption == ssl -> 465
	// - ako nije, a encryption == starttls -> 587
	// - else -> 25
	port := 0
	if v := strings.TrimSpace(os.Getenv("SMTP_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			port = p
		}
	}
	if port == 0 {
		switch enc {
		case "ssl":
			port = 465
		case "starttls":
			port = 587
		default:
			port = 25
		}
	}

	timeout := 10 * time.Second
	if v := strings.TrimSpace(os.Getenv("SMTP_TIMEOUT")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}

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

	// Ako FROM nije eksplicitno zadat, koristi username (najčešći zahtev kod provajdera)
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

	// Osnovna validacija
	if cfg.Host == "" {
		return nil, fmt.Errorf("mailer config invalid: missing SMTP_HOST")
	}
	// Većina provajdera zahteva auth (a ti si naglasio da je "SMTP requires authentication")
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("mailer config invalid: missing SMTP_USERNAME or SMTP_PASSWORD")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("mailer config invalid: missing SMTP_FROM (or SMTP_USERNAME)")
	}

	return &SimpleMailer{cfg: cfg}, nil
}

func (m *SimpleMailer) Send(to, subject, htmlBody, textBody string) error {
	server := mail.NewSMTPClient()
	server.Host = m.cfg.Host
	server.Port = m.cfg.Port
	server.Username = m.cfg.Username
	server.Password = m.cfg.Password
	server.ConnectTimeout = m.cfg.Timeout
	server.SendTimeout = m.cfg.Timeout
	server.KeepAlive = false

	// TLS/STARTTLS podešavanje
	switch m.cfg.Encryption {
	case "ssl": // implicit TLS (465)
		server.Encryption = mail.EncryptionSSLTLS
		server.TLSConfig = &tls.Config{
			ServerName:         m.cfg.Host, // bitno za verifikaciju CN/SAN
			InsecureSkipVerify: m.cfg.InsecureSkipVerify,
		}
	case "starttls": // STARTTLS nad plain konekcijom (587)
		server.Encryption = mail.EncryptionSTARTTLS
		server.TLSConfig = &tls.Config{
			ServerName:         m.cfg.Host,
			InsecureSkipVerify: m.cfg.InsecureSkipVerify,
		}
	default: // "none"
		server.Encryption = mail.EncryptionNone
	}

	smtpClient, err := server.Connect()
	if err != nil {
		return fmt.Errorf("smtp connect failed: %w", err)
	}

	msg := mail.NewMSG()
	if m.cfg.FromName != "" {
		msg.SetFrom(fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.From))
	} else {
		msg.SetFrom(m.cfg.From)
	}
	msg.AddTo(to)
	msg.SetSubject(subject)

	if htmlBody != "" {
		msg.SetBody(mail.TextHTML, htmlBody)
		if textBody != "" {
			msg.AddAlternative(mail.TextPlain, textBody)
		}
	} else {
		if textBody == "" {
			textBody = subject
		}
		msg.SetBody(mail.TextPlain, textBody)
	}

	if msg.Error != nil {
		return fmt.Errorf("compose mail failed: %w", msg.Error)
	}

	if err := msg.Send(smtpClient); err != nil {
		return fmt.Errorf("send mail failed: %w", err)
	}
	return nil
}
