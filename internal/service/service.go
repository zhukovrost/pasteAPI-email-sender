package service

import (
	"bytes"
	"crypto/tls"
	"embed"
	"errors"
	"fmt"
	"github.com/go-mail/mail"
	"github.com/sirupsen/logrus"
	"github.com/zhukovrost/pasteAPI-email-sender/internal/models"
	"html/template"
	"time"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
	Config
	Logger *logrus.Logger
	dialer *mail.Dialer
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
	Timeout  time.Duration
}

var ErrInvalidEmail = errors.New("invalid email")

func New(log *logrus.Logger, cfg Config) *Mailer {
	dialer := mail.NewDialer(
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
	)
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return &Mailer{
		dialer: dialer,
		Config: cfg,
		Logger: log,
	}
}

func (m *Mailer) SendEmail(data *models.Email) error {
	switch data.Type {
	case "activation":
		m.Logger.Debug("got activation mail")
		return m.send(data, "welcome.tmpl")
	case "password-reset":
		m.Logger.Debug("got password reset mail")
		return m.send(data, "password_reset.tmpl")
	default:
		return ErrInvalidEmail
	}
}

func (m *Mailer) send(data *models.Email, templateFile string) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return fmt.Errorf("templating error: %v", err)
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return fmt.Errorf("templating error: %v", err)
	}

	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return fmt.Errorf("templating error: %v", err)
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return fmt.Errorf("templating error: %v", err)
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", data.To.Email)
	msg.SetHeader("Sender", m.Sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// trying to send email 3 times
	for i := 1; i <= 3; i++ {
		err = m.dialer.DialAndSend(msg)
		if nil == err {
			return nil
		}
		m.Logger.Debugf("Failed to send email (attempt %d of 3): %v", i, err)
		time.Sleep(time.Second)
	}

	return err
}
