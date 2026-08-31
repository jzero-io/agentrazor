package mailer

import (
	"crypto/tls"

	gomail "gopkg.in/gomail.v2"
)

type SMTPConfig struct {
	From       string
	Host       string
	Port       int
	Username   string
	Password   string
	EnableSSL  bool
	VerifyCert bool
}

func Send(config SMTPConfig, recipient, subject, body string) error {
	message := gomail.NewMessage()
	message.SetHeader("From", config.From)
	message.SetHeader("To", recipient)
	message.SetHeader("Subject", subject)
	message.SetBody("text/plain", body)

	dialer := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)
	dialer.SSL = config.EnableSSL
	if !config.VerifyCert {
		dialer.TLSConfig = &tls.Config{ //nolint:gosec // Explicit administrator-controlled compatibility option.
			InsecureSkipVerify: true,
		}
	}

	return dialer.DialAndSend(message)
}
