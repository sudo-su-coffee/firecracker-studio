package notifications

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	Enabled    bool
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	Recipients []string
}

type Notifier struct{ cfg Config }

func New(cfg Config) *Notifier { return &Notifier{cfg: cfg} }

func (n *Notifier) Send(subject, body string) error {
	if n == nil || !n.cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(n.cfg.Host) == "" || len(n.cfg.Recipients) == 0 {
		return fmt.Errorf("SMTP notifications require host and recipients")
	}
	port := n.cfg.Port
	if port == 0 {
		port = 587
	}
	from := n.cfg.From
	if from == "" {
		from = n.cfg.Username
	}
	message := []byte("From: " + from + "\r\nTo: " + strings.Join(n.cfg.Recipients, ", ") + "\r\nSubject: " + subject + "\r\n\r\n" + body)
	var auth smtp.Auth
	if n.cfg.Username != "" {
		auth = smtp.PlainAuth("", n.cfg.Username, n.cfg.Password, n.cfg.Host)
	}
	return smtp.SendMail(fmt.Sprintf("%s:%d", n.cfg.Host, port), auth, from, n.cfg.Recipients, message)
}
