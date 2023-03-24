package mail

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFunctions = [...]func(*testing.T, fleet.MailService){
	testSMTPPlainAuth,
	testSMTPPlainAuthInvalidCreds,
	testSMTPSkipVerify,
	testSMTPNoAuth,
	testMailTest,
}

func TestMail(t *testing.T) {
	// This mail test requires mailhog unauthenticated running on localhost:1025
	// and mailpit running on localhost:1026.
	if _, ok := os.LookupEnv("MAIL_TEST"); !ok {
		t.Skip("Mail tests are disabled")
	}

	for _, f := range testFunctions {
		r := NewService()

		t.Run(test.FunctionName(f), func(t *testing.T) {
			f(t, r)
		})
	}
}

func testSMTPPlainAuth(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "smtp plain auth",
		To:      []string{"john@fleet.co"},
		Config: &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
				SMTPConfigured:           true,
				SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
				SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
				SMTPUserName:             "mailpit-username",
				SMTPPassword:             "mailpit-password",
				SMTPEnableTLS:            true,
				SMTPVerifySSLCerts:       true,
				SMTPEnableStartTLS:       true,
				SMTPPort:                 1026,
				SMTPServer:               "localhost",
				SMTPSenderAddress:        "test@example.com",
			},
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testSMTPPlainAuthInvalidCreds(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "smtp plain auth with invalid credentials",
		To:      []string{"john@fleet.co"},
		Config: &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
				SMTPConfigured:           true,
				SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
				SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
				SMTPUserName:             "mailpit-username",
				SMTPPassword:             "wrong",
				SMTPEnableTLS:            true,
				SMTPVerifySSLCerts:       true,
				SMTPEnableStartTLS:       true,
				SMTPPort:                 1026,
				SMTPServer:               "localhost",
				SMTPSenderAddress:        "test@example.com",
			},
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Error(t, err)
}

func testSMTPSkipVerify(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "skip verify",
		To:      []string{"john@fleet.co"},
		Config: &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
				SMTPConfigured:           true,
				SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
				SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
				SMTPUserName:             "mailpit-username",
				SMTPPassword:             "mailpit-password",
				SMTPEnableTLS:            true,
				SMTPVerifySSLCerts:       false,
				SMTPEnableStartTLS:       true,
				SMTPPort:                 1025,
				SMTPServer:               "localhost",
				SMTPSenderAddress:        "test@example.com",
			},
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testSMTPNoAuth(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "no auth",
		To:      []string{"bob@foo.com"},
		Config: &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
				SMTPConfigured:         true,
				SMTPAuthenticationType: fleet.AuthTypeNameNone,
				SMTPEnableTLS:          true,
				SMTPVerifySSLCerts:     true,
				SMTPPort:               1025,
				SMTPServer:             "localhost",
				SMTPSenderAddress:      "test@example.com",
			},
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(mail)
	assert.Nil(t, err)
}

func testMailTest(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "test tester",
		To:      []string{"bob@foo.com"},
		Config: &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{
				SMTPConfigured:           true,
				SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
				SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
				SMTPUserName:             "mailpit-username",
				SMTPPassword:             "mailpit-password",
				SMTPEnableTLS:            true,
				SMTPVerifySSLCerts:       true,
				SMTPPort:                 1026,
				SMTPServer:               "localhost",
				SMTPSenderAddress:        "test@example.com",
			},
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
