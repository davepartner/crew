package mailer

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type Config struct {
	Addr string
	From string
	User string
	Pass string
}

func Send(cfg Config, to, subject, body string) error {
	to = strings.TrimSpace(to)
	if to == "" || !strings.Contains(to, "@") {
		return fmt.Errorf("enter a valid recipient email")
	}
	subject = strings.ReplaceAll(strings.ReplaceAll(subject, "\r", " "), "\n", " ")
	from := strings.TrimSpace(cfg.From)
	if from == "" {
		return fmt.Errorf("email is not configured on this server")
	}

	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return fmt.Errorf("email is not configured correctly")
	}

	var auth smtp.Auth
	if cfg.User != "" {
		auth = smtp.PlainAuth("", cfg.User, cfg.Pass, host)
	}

	message := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")

	return smtp.SendMail(cfg.Addr, auth, envelope(from), []string{to}, []byte(message))
}

func envelope(from string) string {
	if start := strings.LastIndex(from, "<"); start >= 0 {
		if end := strings.Index(from[start:], ">"); end > 0 {
			return strings.TrimSpace(from[start+1 : start+end])
		}
	}
	return from
}
