package mailer

import (
	"crypto/tls"
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

	client, err := smtp.Dial(cfg.Addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		// A local mail server often presents a certificate for the machine
		// hostname, not 127.0.0.1. Only relax verification for loopback.
		config := &tls.Config{ServerName: host}
		if host == "127.0.0.1" || host == "::1" || strings.EqualFold(host, "localhost") {
			config.InsecureSkipVerify = true
		}
		if err := client.StartTLS(config); err != nil {
			return err
		}
	}

	if cfg.User != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(smtp.PlainAuth("", cfg.User, cfg.Pass, host)); err != nil {
				return err
			}
		}
	}

	if err := client.Mail(envelope(from)); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
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
	if _, err := writer.Write([]byte(message)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func envelope(from string) string {
	if start := strings.LastIndex(from, "<"); start >= 0 {
		if end := strings.Index(from[start:], ">"); end > 0 {
			return strings.TrimSpace(from[start+1 : start+end])
		}
	}
	return from
}
