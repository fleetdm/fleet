package mail

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMailer struct{}

func (m *mockMailer) SendEmail(e kolide.Email) error {
	return nil
}

func getMailer() kolide.MailService {

	if os.Getenv("MAIL_TEST") == "" {
		return &mockMailer{}
	}
	return NewService()
}

var testFunctions = [...]func(*testing.T, kolide.MailService){
	testSMTPPlainAuth,
	testSMTPSkipVerify,
	testSMTPNoAuth,
	testMailTest,
}

func TestMail(t *testing.T) {
	for _, f := range testFunctions {
		r := getMailer()

		t.Run(test.FunctionName(f), func(t *testing.T) {
			f(t, r)
		})
	}
}

func testSMTPPlainAuth(t *testing.T, mailer kolide.MailService) {
	mail := kolide.Email{
		Subject: "smtp plain auth",
		To:      []string{"john@kolide.co"},
		Config: &kolide.AppConfig{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   kolide.AuthTypeUserNamePassword,
			SMTPAuthenticationMethod: kolide.AuthMethodPlain,
			SMTPUserName:             "bob",
			SMTPPassword:             "secret",
			SMTPEnableTLS:            true,
			SMTPVerifySSLCerts:       true,
			SMTPEnableStartTLS:       true,
			SMTPPort:                 1025,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "kolide@kolide.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testSMTPSkipVerify(t *testing.T, mailer kolide.MailService) {
	mail := kolide.Email{
		Subject: "skip verify",
		To:      []string{"john@kolide.co"},
		Config: &kolide.AppConfig{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   kolide.AuthTypeUserNamePassword,
			SMTPAuthenticationMethod: kolide.AuthMethodPlain,
			SMTPUserName:             "bob",
			SMTPPassword:             "secret",
			SMTPEnableTLS:            true,
			SMTPVerifySSLCerts:       false,
			SMTPEnableStartTLS:       true,
			SMTPPort:                 1025,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "kolide@kolide.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testSMTPNoAuth(t *testing.T, mailer kolide.MailService) {
	mail := kolide.Email{
		Subject: "no auth",
		To:      []string{"bob@foo.com"},
		Config: &kolide.AppConfig{
			SMTPConfigured:         true,
			SMTPAuthenticationType: kolide.AuthTypeNone,
			SMTPEnableTLS:          true,
			SMTPVerifySSLCerts:     true,
			SMTPPort:               1025,
			SMTPServer:             "localhost",
			SMTPSenderAddress:      "kolide@kolide.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testMailTest(t *testing.T, mailer kolide.MailService) {
	mail := kolide.Email{
		Subject: "test tester",
		To:      []string{"bob@foo.com"},
		Config: &kolide.AppConfig{
			SMTPConfigured:         true,
			SMTPAuthenticationType: kolide.AuthTypeNone,
			SMTPEnableTLS:          true,
			SMTPVerifySSLCerts:     true,
			SMTPPort:               1025,
			SMTPServer:             "localhost",
			SMTPSenderAddress:      "kolide@kolide.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}
	err := Test(mailer, mail)
	assert.Nil(t, err)

}

func TestTemplateProcessor(t *testing.T) {
	mailer := PasswordResetMailer{
		BaseURL: "https://localhost.com:8080",
		Token:   "12345",
	}

	out, err := mailer.Message()
	require.Nil(t, err)
	assert.NotNil(t, out)
}
