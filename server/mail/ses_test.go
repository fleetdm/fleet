package mail

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getFromSES(t *testing.T) {
	type args struct {
		e            fleet.Email
		senderDomain string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return properly formatted SMTP from for use in SES",
			args: args{e: fleet.Email{
				ServerURL: "https://foobar.fleetdm.com",
			}},
			want:    "From: do-not-reply@foobar.fleetdm.com\r\n",
			wantErr: assert.NoError,
		},
		{
			name: "should use configured sender domain when provided",
			args: args{
				e: fleet.Email{
					ServerURL: "not-a-url",
				},
				senderDomain: "notifications.example.com",
			},
			want:    "From: do-not-reply@notifications.example.com\r\n",
			wantErr: assert.NoError,
		},
		{
			name: "should error when we fail to parse fleet server url",
			args: args{e: fleet.Email{
				ServerURL: "not-a-url",
			}},
			want:    "",
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFromSES(tt.args.e, tt.args.senderDomain)
			if !tt.wantErr(t, err, fmt.Sprintf("getFromSES(%v)", tt.args.e)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getFromSES(%v)", tt.args.e)
		})
	}
}

type mockSESSender struct {
	shouldErr bool
	input     *ses.SendRawEmailInput
}

func (m *mockSESSender) SendRawEmail(ctx context.Context, input *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error) {
	if m.shouldErr {
		return nil, errors.New("some error")
	}
	m.input = input
	return nil, nil
}

func Test_sesSender_SendEmail(t *testing.T) {
	baseEmail := fleet.Email{
		Subject:   "Hello from Fleet!",
		To:        []string{"foouser@fleetdm.com"},
		ServerURL: "https://foobar.fleetdm.com",
		Mailer: &SMTPTestMailer{
			BaseURL: "https://localhost:8080",
		},
	}

	t.Run("should send email with configured sender domain", func(t *testing.T) {
		client := &mockSESSender{}
		s := &sesSender{
			client:       client,
			sourceArn:    "foo",
			senderDomain: "notifications.example.com",
		}

		email := baseEmail
		email.ServerURL = "not-a-url"
		err := s.SendEmail(context.Background(), email)
		require.NoError(t, err)
		require.NotNil(t, client.input)
		assert.Contains(t, string(client.input.RawMessage.Data), "From: do-not-reply@notifications.example.com\r\n")
	})

	t.Run("should send email with server url host when sender domain is not configured", func(t *testing.T) {
		client := &mockSESSender{}
		s := &sesSender{
			client:    client,
			sourceArn: "foo",
		}

		err := s.SendEmail(context.Background(), baseEmail)
		require.NoError(t, err)
		require.NotNil(t, client.input)
		assert.Contains(t, string(client.input.RawMessage.Data), "From: do-not-reply@foobar.fleetdm.com\r\n")
	})

	t.Run("should error when server url is invalid and sender domain is not configured", func(t *testing.T) {
		s := &sesSender{
			client:    &mockSESSender{},
			sourceArn: "foo",
		}

		email := baseEmail
		email.ServerURL = "not-a-url"
		assert.Error(t, s.SendEmail(context.Background(), email))
	})

	t.Run("should error when ses client is nil", func(t *testing.T) {
		s := &sesSender{sourceArn: "foo"}
		assert.Error(t, s.SendEmail(context.Background(), baseEmail))
	})

	t.Run("should error when ses client returns an error", func(t *testing.T) {
		s := &sesSender{
			client:    &mockSESSender{shouldErr: true},
			sourceArn: "foo",
		}
		assert.Error(t, s.SendEmail(context.Background(), baseEmail))
	})
}
