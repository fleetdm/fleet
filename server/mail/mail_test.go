package mail

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testMailpitSMTPPort  = getTestMailpitSMTPPort()
	testMailpitWebURL    = getTestMailpitWebURL()
	testSMTP4DevSMTPPort = getTestSMTP4DevSMTPPort()
)

func getTestMailpitSMTPPort() uint {
	if port := os.Getenv("FLEET_MAILPIT_SMTP_PORT"); port != "" {
		if p, err := strconv.ParseUint(port, 10, 32); err == nil && p > 0 {
			return uint(p)
		}
	}
	return 1026
}

func getTestMailpitWebURL() string {
	if port := os.Getenv("FLEET_MAILPIT_WEB_PORT"); port != "" {
		return "http://127.0.0.1:" + port
	}
	return "http://127.0.0.1:8026"
}

func getTestSMTP4DevSMTPPort() uint {
	if port := os.Getenv("FLEET_SMTP4DEV_SMTP_PORT"); port != "" {
		if p, err := strconv.ParseUint(port, 10, 32); err == nil && p > 0 {
			return uint(p)
		}
	}
	return 1027
}

var testFunctions = [...]func(*testing.T, fleet.MailService){
	testSMTPPlainAuth,
	testSMTPPlainAuthInvalidCreds,
	testSMTPSkipVerify,
	testSMTPNoAuthWithTLS,
	testSMTPDomain,
	testMailTest,
}

func TestCanSendMail(t *testing.T) {
	settings := fleet.SMTPSettings{
		SMTPConfigured:           true,
		SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
		SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
		SMTPUserName:             "mailpit-username",
		SMTPPassword:             "mailpit-password",
		SMTPEnableTLS:            false,
		SMTPVerifySSLCerts:       false,
		SMTPEnableStartTLS:       false,
		SMTPPort:                 testMailpitSMTPPort,
		SMTPServer:               "localhost",
		SMTPSenderAddress:        "test@example.com",
	}

	r, err := NewService(config.TestConfig())
	require.NoError(t, err)
	require.True(t, r.CanSendEmail(settings))
	require.False(t, r.CanSendEmail(fleet.SMTPSettings{}))
}

func TestMail(t *testing.T) {
	// This mail test requires mailhog and mailpit (ports read from env vars
	// FLEET_MAILPIT_SMTP_PORT, FLEET_SMTP4DEV_SMTP_PORT, FLEET_MAILPIT_WEB_PORT).
	if _, ok := os.LookupEnv("MAIL_TEST"); !ok {
		t.Skip("Mail tests are disabled")
	}

	for _, f := range testFunctions {
		r, err := NewService(config.TestConfig())
		require.NoError(t, err)

		t.Run(test.FunctionName(f), func(t *testing.T) {
			f(t, r)
		})
	}
}

func testSMTPPlainAuth(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "smtp plain auth",
		To:      []string{"john@fleet.co"},
		SMTPSettings: fleet.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
			SMTPUserName:             "mailpit-username",
			SMTPPassword:             "mailpit-password",
			SMTPEnableTLS:            false,
			SMTPVerifySSLCerts:       false,
			SMTPEnableStartTLS:       false,
			SMTPPort:                 testMailpitSMTPPort,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(context.Background(), mail)
	assert.Nil(t, err)
}

func testSMTPPlainAuthInvalidCreds(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "smtp plain auth with invalid credentials",
		To:      []string{"john@fleet.co"},
		SMTPSettings: fleet.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
			SMTPUserName:             "mailpit-username",
			SMTPPassword:             "wrong",
			SMTPEnableTLS:            false,
			SMTPVerifySSLCerts:       false,
			SMTPEnableStartTLS:       false,
			SMTPPort:                 testMailpitSMTPPort,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(context.Background(), mail)
	assert.Error(t, err)
}

func testSMTPSkipVerify(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "skip verify",
		To:      []string{"john@fleet.co"},
		SMTPSettings: fleet.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
			SMTPUserName:             "mailpit-username",
			SMTPPassword:             "mailpit-password",
			SMTPEnableTLS:            true,
			SMTPVerifySSLCerts:       false,
			SMTPEnableStartTLS:       true,
			SMTPPort:                 testSMTP4DevSMTPPort,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(context.Background(), mail)
	assert.Nil(t, err)
}

func testSMTPNoAuthWithTLS(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "no auth",
		To:      []string{"bob@foo.com"},
		SMTPSettings: fleet.SMTPSettings{
			SMTPConfigured:         true,
			SMTPAuthenticationType: fleet.AuthTypeNameNone,
			SMTPEnableTLS:          true,
			SMTPVerifySSLCerts:     true,
			SMTPEnableStartTLS:     true,
			SMTPPort:               testSMTP4DevSMTPPort,
			SMTPServer:             "localhost",
			SMTPSenderAddress:      "test@example.com",
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(context.Background(), mail)
	assert.Nil(t, err)
}

func testSMTPDomain(t *testing.T, mailer fleet.MailService) {
	randomAddress := uuid.NewString() + "@example.com"

	mail := fleet.Email{
		Subject: "custom client hello",
		To:      []string{"bob@foo.com"},
		SMTPSettings: fleet.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
			SMTPUserName:             "mailpit-username",
			SMTPPassword:             "mailpit-password",
			SMTPEnableTLS:            false,
			SMTPVerifySSLCerts:       false,
			SMTPEnableStartTLS:       false,
			SMTPPort:                 testMailpitSMTPPort,
			SMTPServer:               "localhost",
			SMTPDomain:               "custom.domain.example.com",
			SMTPSenderAddress:        randomAddress,
		},
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	err := mailer.SendEmail(context.Background(), mail)
	assert.Nil(t, err)

	rawMsg := getLastRawMailpitMessageFrom(t, randomAddress)

	require.Contains(t, rawMsg, "Received: from custom.domain.example.com")
}

// Only what we need for the current test. If you need more, fill the struct out.
// https://mailpit.axllent.org/docs/api-v1/view.html#get-/api/v1/messages
type MailpitMessages struct {
	Messages []struct {
		Created time.Time
		From    struct {
			Address string `json:"Address"`
			Name    string `json:"Name"`
		}
		ID string `json:"ID"`
		To []struct {
			Address string `json:"Address"`
			Name    string `json:"Name"`
		} `json:"To"`
		BCC []struct {
			Address string `json:"Address"`
			Name    string `json:"Name"`
		} `json:"Bcc"`
	} `json:"messages"`
}

func getLastRawMailpitMessageFrom(t *testing.T, address string) string {
	res, err := http.Get(testMailpitWebURL + "/api/v1/messages")
	require.NoError(t, err)

	var messages MailpitMessages
	err = json.NewDecoder(res.Body).Decode(&messages)
	require.NoError(t, err)

	var messageID string
	for _, message := range messages.Messages {
		if message.From.Address == address {
			messageID = message.ID
		}
	}
	require.NotNilf(t, messageID, "could not find message from %s in mailpit", address)

	res, err = http.Get(fmt.Sprintf("%s/api/v1/message/%s/raw", testMailpitWebURL, messageID))
	require.NoError(t, err)

	rawMail, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return string(rawMail)
}

func testMailTest(t *testing.T, mailer fleet.MailService) {
	mail := fleet.Email{
		Subject: "test tester",
		To:      []string{"bob@foo.com"},
		SMTPSettings: fleet.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
			SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
			SMTPUserName:             "foo",
			SMTPPassword:             "bar",
			SMTPEnableTLS:            true,
			SMTPVerifySSLCerts:       true,
			SMTPEnableStartTLS:       true,
			SMTPPort:                 testSMTP4DevSMTPPort,
			SMTPServer:               "localhost",
			SMTPSenderAddress:        "test@example.com",
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

func startFakeSMTPServerFailingSTARTTLS(t *testing.T) (host string, port uint) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))
				r := bufio.NewReader(c)
				_, _ = io.WriteString(c, "220 fake.smtp.test ESMTP\r\n")
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd := strings.ToUpper(strings.TrimSpace(line))
					switch {
					case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
						_, _ = io.WriteString(c, "250-fake.smtp.test\r\n250 STARTTLS\r\n")
					case cmd == "STARTTLS":
						_, _ = io.WriteString(c, "220 Ready to start TLS\r\n")
						// Close immediately so the client's TLS handshake fails.
						return
					case cmd == "QUIT":
						_, _ = io.WriteString(c, "221 Bye\r\n")
						return
					default:
						_, _ = io.WriteString(c, "500 unrecognized\r\n")
					}
				}
			}(conn)
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		<-done
	})

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", uint(addr.Port) //nolint:gosec // dismiss G115 — TCPAddr.Port is in [0, 65535]
}

// fakeMailer satisfies fleet.Mailer without depending on bindata templates.
type fakeMailer struct{}

func (fakeMailer) Message() ([]byte, error) { return []byte("Subject: test\r\n\r\nbody\r\n"), nil }

func TestSendMailSTARTTLSFailureWithoutSSLTLS(t *testing.T) {
	host, port := startFakeSMTPServerFailingSTARTTLS(t)

	err := Test(mailService{}, fleet.Email{
		Subject: "test",
		To:      []string{"to@example.com"},
		SMTPSettings: fleet.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   fleet.AuthTypeNameNone,
			SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
			SMTPVerifySSLCerts:       true,
			SMTPEnableTLS:            false,
			SMTPEnableStartTLS:       true,
			SMTPPort:                 port,
			SMTPServer:               host,
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: fakeMailer{},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSTARTTLSWithoutSSLTLS)
}

func TestSendMailSTARTTLSFailureWithVerifyOff(t *testing.T) {
	host, port := startFakeSMTPServerFailingSTARTTLS(t)

	err := Test(mailService{}, fleet.Email{
		Subject: "test",
		To:      []string{"to@example.com"},
		SMTPSettings: fleet.SMTPSettings{
			SMTPConfigured:           true,
			SMTPAuthenticationType:   fleet.AuthTypeNameNone,
			SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
			SMTPVerifySSLCerts:       false, // user opted into skip-verify
			SMTPEnableTLS:            false,
			SMTPEnableStartTLS:       true,
			SMTPPort:                 port,
			SMTPServer:               host,
			SMTPSenderAddress:        "test@example.com",
		},
		Mailer: fakeMailer{},
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrSTARTTLSWithoutSSLTLS)
	require.Contains(t, err.Error(), "startTLS error")
}

func Test_getFrom(t *testing.T) {
	type args struct {
		e fleet.Email
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return SMTP formatted From string",
			args: args{
				e: fleet.Email{
					SMTPSettings: fleet.SMTPSettings{
						SMTPSenderAddress: "foo@bar.com",
					},
				},
			},
			want:    "From: foo@bar.com\r\n",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFrom(tt.args.e)
			if !tt.wantErr(t, err, fmt.Sprintf("getFrom(%v)", tt.args.e)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getFrom(%v)", tt.args.e)
		})
	}
}
