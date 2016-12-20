// Package mail provides implementations of the Kolide MailService
package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/kolide/kolide-ose/server/kolide"
)

func NewService() kolide.MailService {
	return &mailService{}
}

func NewDevService() kolide.MailService {
	return &devMailService{}
}

type mailService struct{}
type devMailService struct{}

const (
	PortSSL = 465
	PortTLS = 587
)

func getMessageBody(e kolide.Email) ([]byte, error) {
	body, err := e.Mailer.Message()
	if err != nil {
		return nil, err
	}
	mime := `MIME-version: 1.0;` + "\r\n"
	content := `Content-Type: text/html; charset="UTF-8";` + "\r\n"
	subject := "Subject: " + e.Subject + "\r\n"
	msg := []byte(subject + mime + content + "\r\n" + string(body) + "\r\n")
	return msg, nil
}

func (dm devMailService) SendEmail(e kolide.Email) error {
	if !e.Config.SMTPDisabled {
		if e.Config.SMTPConfigured {
			msg, err := getMessageBody(e)
			if err != nil {
				return err
			}
			fmt.Printf(string(msg))
		}
	}
	return nil
}

func (m mailService) SendEmail(e kolide.Email) error {
	if !e.Config.SMTPDisabled && e.Config.SMTPConfigured {
		msg, err := getMessageBody(e)
		if err != nil {
			return err
		}
		return m.sendMail(e, msg)
	}
	return nil
}

func (m mailService) sendMail(e kolide.Email, msg []byte) error {
	smtpHost := fmt.Sprintf("%s:%d", e.Config.SMTPServer, e.Config.SMTPPort)
	var auth smtp.Auth
	if e.Config.SMTPAuthenticationType == kolide.AuthTypeUserNamePassword {
		switch e.Config.SMTPAuthenticationMethod {
		case kolide.AuthMethodCramMD5:
			auth = smtp.CRAMMD5Auth(e.Config.SMTPUserName, e.Config.SMTPPassword)
			return smtp.SendMail(smtpHost, auth, e.Config.SMTPSenderAddress, e.To, msg)
		case kolide.AuthMethodPlain:
			auth = smtp.PlainAuth("", e.Config.SMTPUserName, e.Config.SMTPPassword, e.Config.SMTPServer)

		default:
			return fmt.Errorf("Unknown SMTP auth type '%d'", e.Config.SMTPAuthenticationMethod)
		}
	} else {
		auth = nil
	}
	client, err := smtp.Dial(smtpHost)
	if err != nil {
		return err
	}
	defer client.Close()
	if err = client.Hello(""); err != nil {
		return err
	}
	if e.Config.SMTPEnableStartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{
				ServerName:         e.Config.SMTPServer,
				InsecureSkipVerify: !e.Config.SMTPVerifySSLCerts,
			}
			if err = client.StartTLS(config); err != nil {
				return err
			}
		}
	}
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(e.Config.SMTPSenderAddress); err != nil {
		return err
	}
	for _, recip := range e.To {
		if err = client.Rcpt(recip); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return nil
	}
	_, err = writer.Write(msg)
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
