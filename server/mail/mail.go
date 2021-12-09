// Package mail provides implementations of the Fleet MailService
package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/bindata"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
		return fmt.Errorf("failed to get message body: %w", err)
	}

	svc, ok := mailer.(sender)
	if !ok {
		return nil
	}

	err = svc.sendMail(e, mailBody)
	if err != nil {
		return fmt.Errorf("sending mail: %w", err)
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
		return nil, fmt.Errorf("get mailer message: %w", err)
	}
	mime := `MIME-version: 1.0;` + "\r\n"
	content := `Content-Type: text/html; charset="UTF-8";` + "\r\n"
	subject := "Subject: " + e.Subject + "\r\n"
	from := "From: " + e.Config.SMTPSettings.SMTPSenderAddress + "\r\n"
	msg := []byte(subject + from + mime + content + "\r\n" + string(body) + "\r\n")
	return msg, nil
}

func (m mailService) SendEmail(e fleet.Email) error {
	if !e.Config.SMTPSettings.SMTPConfigured {
		return errors.New("email not configured")
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
	if e.Config.SMTPSettings.SMTPAuthenticationType != fleet.AuthTypeNameUserNamePassword {
		return nil, nil
	}

	username := e.Config.SMTPSettings.SMTPUserName
	password := e.Config.SMTPSettings.SMTPPassword
	server := e.Config.SMTPSettings.SMTPServer
	authMethod := e.Config.SMTPSettings.SMTPAuthenticationMethod

	var auth smtp.Auth
	switch authMethod {
	case fleet.AuthMethodNameCramMD5:
		auth = smtp.CRAMMD5Auth(username, password)
	case fleet.AuthMethodNamePlain:
		auth = smtp.PlainAuth("", username, password, server)
	case fleet.AuthMethodNameLogin:
		auth = LoginAuth(username, password, server)
	default:
		return nil, fmt.Errorf("unknown SMTP auth type '%s'", authMethod)
	}
	return auth, nil
}

func (m mailService) sendMail(e fleet.Email, msg []byte) error {
	smtpHost := fmt.Sprintf(
		"%s:%d", e.Config.SMTPSettings.SMTPServer, e.Config.SMTPSettings.SMTPPort)
	auth, err := smtpAuth(e)
	if err != nil {
		return fmt.Errorf("failed to get smtp auth: %w", err)
	}

	if e.Config.SMTPSettings.SMTPAuthenticationMethod == fleet.AuthMethodNameCramMD5 {
		err = smtp.SendMail(smtpHost, auth, e.Config.SMTPSettings.SMTPSenderAddress, e.To, msg)
		if err != nil {
			return fmt.Errorf("failed to send mail. crammd5 auth method: %w", err)
		}
		return nil
	}

	client, err := dialTimeout(smtpHost)
	if err != nil {
		return fmt.Errorf("could not dial smtp host: %w", err)
	}
	defer client.Close()

	if e.Config.SMTPSettings.SMTPEnableStartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{
				ServerName:         e.Config.SMTPSettings.SMTPServer,
				InsecureSkipVerify: !e.Config.SMTPSettings.SMTPVerifySSLCerts,
			}
			if err = client.StartTLS(config); err != nil {
				return fmt.Errorf("startTLS error: %w", err)
			}
		}
	}
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("client auth error: %w", err)
		}
	}
	if err = client.Mail(e.Config.SMTPSettings.SMTPSenderAddress); err != nil {
		return fmt.Errorf("could not issue mail to provided address: %w", err)
	}
	for _, recip := range e.To {
		if err = client.Rcpt(recip); err != nil {
			return fmt.Errorf("failed to get recipient: %w", err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("getting client data: %w", err)
	}

	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("error on client quit: %w", err)
	}
	return nil
}

// dialTimeout sets a timeout on net.Dial to prevent email from attempting to
// send indefinitely.
func dialTimeout(addr string) (client *smtp.Client, err error) {
	// Ensure that errors are always returned after at least 5s to
	// eliminate (some) timing attacks (in which a malicious user tries to
	// port scan using the email functionality in Fleet)
	c := time.After(30 * time.Second)
	defer func() {
		if err != nil {
			// Wait until timer has elapsed to return anything
			<-c
		}
	}()

	conn, err := net.DialTimeout("tcp", addr, 28*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dialing with timeout: %w", err)
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split host port: %w", err)
	}

	// Set a deadline to ensure we time out quickly when there is a TCP
	// server listening but it's not an SMTP server (otherwise this seems
	// to time out in 20s)
	_ = conn.SetDeadline(time.Now().Add(28 * time.Second))
	client, err = smtp.NewClient(conn, host)
	if err != nil {
		return nil, fmt.Errorf("SMTP connection error: %w", err)
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
