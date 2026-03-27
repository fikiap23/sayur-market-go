package message

import (
	"crypto/tls"
	"notification-service/config"

	"github.com/go-mail/mail"
	"github.com/labstack/gommon/log"
)

type MessageEmailInterface interface {
	SendEmailNotif(to, subject, body string) error
}

type emailAttribute struct {
	Username string
	Password string
	Host     string
	Port     int
	From     string
	IsTls    bool
}

// SendEmailNotif implements MessageEmailInterface.
func (e *emailAttribute) SendEmailNotif(to string, subject string, body string) error {
	m := mail.NewMessage()
	m.SetHeader("From", e.From)
	m.SetHeader("To", to)

	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := mail.NewDialer(e.Host, e.Port, e.Username, e.Password)
	d.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	if err := d.DialAndSend(m); err != nil {
		log.Errorf("[SendEmailNotif-1] Error: %v", err)
		return err
	}

	return nil
}

func NewMessageEmail(cfg *config.Config) MessageEmailInterface {
	return &emailAttribute{
		Username: cfg.EmailConf.Username,
		Password: cfg.EmailConf.Password,
		Host:     cfg.EmailConf.Host,
		Port:     cfg.EmailConf.Port,
		From:     cfg.EmailConf.Sending,
		IsTls:    cfg.EmailConf.IsTLS,
	}
}
