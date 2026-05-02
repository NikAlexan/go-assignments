package email

import (
	"fmt"
	"net/smtp"
)

type Sender struct {
	from     string
	password string
	host     string
	port     string
}

// NewSender returns nil if credentials are not provided (email sending disabled).
func NewSender(from, password string) *Sender {
	if from == "" || password == "" {
		return nil
	}
	return &Sender{from: from, password: password, host: "smtp.gmail.com", port: "587"}
}

func (s *Sender) Send(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.from, s.password, s.host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.from, to, subject, body)
	return smtp.SendMail(s.host+":"+s.port, auth, s.from, []string{to}, []byte(msg))
}
