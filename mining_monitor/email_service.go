package mining_monitor

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type EmailService interface {
	SendEmail(subject, body string) error

	SetMaxEmails(max int, interval time.Duration)
}

type GMailService struct {
	host     string
	username string
	password string
	port     int

	from string
	to   []string

	interval time.Duration
	max      int
	sent     int
	lastSent time.Time
}

func NewGMailService(host, from string, to []string, username, password string, port int) EmailService {
	return &GMailService{
		host:     host,
		from:     from,
		username: username,
		password: password,
		port:     port,
		to:       to,
		sent:     0,
		lastSent: time.Now(),
		max:      -1,
	}
}

func (e *GMailService) SendEmail(subject, msg string) error {
	if e.max > 0 {
		if time.Since(e.lastSent) > e.interval {
			e.sent = 0
		} else if e.sent >= e.max {
			return fmt.Errorf("maximum of %d emails have been sent within %+v interval", e.max, e.interval)
		} else {
			e.sent++
		}
	}
	body := fmt.Sprintf(
		"To: %s\r\n"+
			"Subject: %s\r\n"+
			"\r\n"+
			"%s",
		strings.Join(e.to, ","), subject, msg,
	)
	auth := smtp.PlainAuth("", e.from, e.password, e.host)
	if err := smtp.SendMail(fmt.Sprintf("%s:%d", e.host, e.port), auth, e.from, e.to, []byte(body)); err != nil {
		return err
	}
	return nil
}

func (e *GMailService) SetMaxEmails(max int, interval time.Duration) {
	e.max = max
	e.interval = interval
}
