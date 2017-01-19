// Package mail provides implementations of the Kolide MailService
package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/pkg/errors"
)

func NewService() kolide.MailService {
	return &mailService{}
}

type mailService struct{}

type sender interface {
	sendMail(e kolide.Email, msg []byte) error
}

func Test(mailer kolide.MailService, e kolide.Email) error {
	mailBody, err := getMessageBody(e)
	if err != nil {
		return errors.Wrap(err, "failed to get message body")
	}

	svc, ok := mailer.(sender)
	if !ok {
		return nil
	}

	err = svc.sendMail(e, mailBody)
	if err != nil {
		return errors.Wrap(err, "sending mail")
	}

	return nil
}

const (
	PortSSL = 465
	PortTLS = 587
)

func getMessageBody(e kolide.Email) ([]byte, error) {
	body, err := e.Mailer.Message()
	if err != nil {
		return nil, errors.Wrap(err, "get mailer message")
	}
	mime := `MIME-version: 1.0;` + "\r\n"
	content := `Content-Type: text/html; charset="UTF-8";` + "\r\n"
	subject := "Subject: " + e.Subject + "\r\n"
	msg := []byte(subject + mime + content + "\r\n" + string(body) + "\r\n")
	return msg, nil
}

func (m mailService) SendEmail(e kolide.Email) error {
	if !e.Config.SMTPConfigured {
		return fmt.Errorf("email not configured")
	}
	msg, err := getMessageBody(e)
	if err != nil {
		return err
	}
	return m.sendMail(e, msg)
}

func smtpAuth(e kolide.Email) (smtp.Auth, error) {
	if e.Config.SMTPAuthenticationType != kolide.AuthTypeUserNamePassword {
		return nil, nil
	}
	var auth smtp.Auth
	switch e.Config.SMTPAuthenticationMethod {
	case kolide.AuthMethodCramMD5:
		auth = smtp.CRAMMD5Auth(e.Config.SMTPUserName, e.Config.SMTPPassword)
	case kolide.AuthMethodPlain:
		auth = smtp.PlainAuth("", e.Config.SMTPUserName, e.Config.SMTPPassword, e.Config.SMTPServer)
	default:
		return nil, fmt.Errorf("unknown SMTP auth type '%d'", e.Config.SMTPAuthenticationMethod)
	}
	return auth, nil
}

func (m mailService) sendMail(e kolide.Email, msg []byte) error {
	smtpHost := fmt.Sprintf("%s:%d", e.Config.SMTPServer, e.Config.SMTPPort)
	auth, err := smtpAuth(e)
	if err != nil {
		return errors.Wrap(err, "failed to get smtp auth")
	}

	if e.Config.SMTPAuthenticationMethod == kolide.AuthMethodCramMD5 {
		err = smtp.SendMail(smtpHost, auth, e.Config.SMTPSenderAddress, e.To, msg)
		if err != nil {
			return errors.Wrap(err, "failed to send mail. cramd5 auth method")
		}
		return nil
	}

	client, err := dialTimeout(smtpHost)
	if err != nil {
		return errors.Wrap(err, "could not dial smtp host")
	}
	defer client.Close()
	if e.Config.SMTPEnableStartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{
				ServerName:         e.Config.SMTPServer,
				InsecureSkipVerify: !e.Config.SMTPVerifySSLCerts,
			}
			if err = client.StartTLS(config); err != nil {
				return errors.Wrap(err, "startTLS error")
			}
		}
	}
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return errors.Wrap(err, "client auth error")
		}
	}
	if err = client.Mail(e.Config.SMTPSenderAddress); err != nil {
		return errors.Wrap(err, "could not issue mail to provided address")
	}
	for _, recip := range e.To {
		if err = client.Rcpt(recip); err != nil {
			return errors.Wrap(err, "failed to get recipient")
		}
	}
	writer, err := client.Data()
	if err != nil {
		return errors.Wrap(err, "getting client data")
	}
	_, err = writer.Write(msg)
	if err = writer.Close(); err != nil {
		return errors.Wrap(err, "failed to close writer")
	}

	if err := client.Quit(); err != nil {
		return errors.Wrap(err, "error on client quit")
	}
	return nil
}

// dialTimeout sets a timeout on net.Dial to prevent email from attempting to
// send indefinitely.
func dialTimeout(addr string) (*smtp.Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "dialing with timeout")
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrap(err, "split host port")
	}
	return smtp.NewClient(conn, host)
}
