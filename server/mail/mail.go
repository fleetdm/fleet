// Package mail provides implementations of the Fleet MailService
package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/bindata"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func NewService() fleet.MailService {
	return &mailService{}
}

type mailService struct{}

type sender interface {
	sendMail(e fleet.Email, msg []byte) error
}

func Test(mailer fleet.MailService, e fleet.Email) error {
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

func getMessageBody(e fleet.Email) ([]byte, error) {
	body, err := e.Mailer.Message()
	if err != nil {
		return nil, errors.Wrap(err, "get mailer message")
	}
	mime := `MIME-version: 1.0;` + "\r\n"
	content := `Content-Type: text/html; charset="UTF-8";` + "\r\n"
	subject := "Subject: " + e.Subject + "\r\n"
	from := "From: " + e.Config.SMTPSenderAddress + "\r\n"
	msg := []byte(subject + from + mime + content + "\r\n" + string(body) + "\r\n")
	return msg, nil
}

func (m mailService) SendEmail(e fleet.Email) error {
	if !e.Config.SMTPConfigured {
		return fmt.Errorf("email not configured")
	}
	msg, err := getMessageBody(e)
	if err != nil {
		return err
	}
	return m.sendMail(e, msg)
}

type loginauth struct {
	username string
	password string
	host     string
}

func LoginAuth(username, password, host string) smtp.Auth {
	return &loginauth{username: username, password: password, host: host}
}

func isLocalhost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

func (l *loginauth) Start(server *smtp.ServerInfo) (proto string, toServer []byte, err error) {
	if !server.TLS && !isLocalhost(server.Name) {
		return "", nil, errors.New("unencrypted connection")
	}

	if server.Name != l.host {
		return "", nil, errors.New("wrong host name")
	}

	return "LOGIN", nil, nil
}

func (l *loginauth) Next(fromServer []byte, more bool) (toServer []byte, err error) {
	if !more {
		return nil, nil
	}

	prompt := strings.TrimSpace(string(fromServer))
	switch prompt {
	case "Username:":
		return []byte(l.username), nil
	case "Password:":
		return []byte(l.password), nil
	default:
		return nil, errors.New("unexpected LOGIN prompt from server")
	}
}

func smtpAuth(e fleet.Email) (smtp.Auth, error) {
	if e.Config.SMTPAuthenticationType != fleet.AuthTypeUserNamePassword {
		return nil, nil
	}
	var auth smtp.Auth
	switch e.Config.SMTPAuthenticationMethod {
	case fleet.AuthMethodCramMD5:
		auth = smtp.CRAMMD5Auth(e.Config.SMTPUserName, e.Config.SMTPPassword)
	case fleet.AuthMethodPlain:
		auth = smtp.PlainAuth("", e.Config.SMTPUserName, e.Config.SMTPPassword, e.Config.SMTPServer)
	case fleet.AuthMethodLogin:
		auth = LoginAuth(e.Config.SMTPUserName, e.Config.SMTPPassword, e.Config.SMTPServer)
	default:
		return nil, fmt.Errorf("unknown SMTP auth type '%d'", e.Config.SMTPAuthenticationMethod)
	}
	return auth, nil
}

func (m mailService) sendMail(e fleet.Email, msg []byte) error {
	smtpHost := fmt.Sprintf("%s:%d", e.Config.SMTPServer, e.Config.SMTPPort)
	auth, err := smtpAuth(e)
	if err != nil {
		return errors.Wrap(err, "failed to get smtp auth")
	}

	if e.Config.SMTPAuthenticationMethod == fleet.AuthMethodCramMD5 {
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
	if err != nil {
		return errors.Wrap(err, "failed to write")
	}

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
func dialTimeout(addr string) (client *smtp.Client, err error) {
	// Ensure that errors are always returned after at least 5s to
	// eliminate (some) timing attacks (in which a malicious user tries to
	// port scan using the email functionality in Fleet)
	c := time.After(5 * time.Second)
	defer func() {
		if err != nil {
			// Wait until timer has elapsed to return anything
			<-c
		}
	}()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "dialing with timeout")
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.Wrap(err, "split host port")
	}

	// Set a deadline to ensure we time out quickly when there is a TCP
	// server listening but it's not an SMTP server (otherwise this seems
	// to time out in 20s)
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	client, err = smtp.NewClient(conn, host)
	if err != nil {
		return nil, errors.New("SMTP connection error")
	}
	// Clear deadlines
	_ = conn.SetDeadline(time.Time{})

	return client, nil
}

// SMTPTestMailer is used to build an email message that will be used as
// a test message when testing SMTP configuration
type SMTPTestMailer struct {
	BaseURL  template.URL
	AssetURL template.URL
}

func (m *SMTPTestMailer) Message() ([]byte, error) {
	t, err := getTemplate("server/mail/templates/smtp_setup.html")
	if err != nil {
		return nil, err
	}

	var msg bytes.Buffer
	if err = t.Execute(&msg, m); err != nil {
		return nil, err
	}

	return msg.Bytes(), nil
}

func getTemplate(templatePath string) (*template.Template, error) {
	templateData, err := bindata.Asset(templatePath)
	if err != nil {
		return nil, err
	}

	t, err := template.New("email_template").Parse(string(templateData))
	if err != nil {
		return nil, err
	}

	return t, nil
}
