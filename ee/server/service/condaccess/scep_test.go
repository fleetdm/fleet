package condaccess

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"

	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallengeMiddleware(t *testing.T) {
	cases := []struct {
		name           string
		challenge      string
		consumeErr     error
		wantErr        string
		wantSignCalled bool
	}{
		{
			name:      "empty challenge is rejected",
			challenge: "",
			wantErr:   "missing challenge",
		},
		{
			name:       "unknown challenge is rejected",
			challenge:  "unknown-challenge",
			consumeErr: common_mysql.NotFound("challenge"),
			wantErr:    "invalid challenge",
		},
		{
			name:       "expired or consumed challenge is rejected",
			challenge:  "expired-challenge",
			consumeErr: common_mysql.NotFound("challenge"),
			wantErr:    "invalid challenge",
		},
		{
			name:       "datastore error is propagated",
			challenge:  "boom",
			consumeErr: errors.New("db is down"),
			wantErr:    "consuming SCEP challenge",
		},
		{
			name:           "valid challenge is accepted",
			challenge:      "valid-challenge",
			wantSignCalled: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.DataStore)
			consumeCalled := false
			ds.ConsumeChallengeFunc = func(_ context.Context, challenge string) error {
				consumeCalled = true
				assert.Equal(t, tc.challenge, challenge, "middleware should consume the challenge it received")
				return tc.consumeErr
			}

			signCalled := false
			dummySigner := scepserver.CSRSignerContextFunc(
				func(_ context.Context, _ *scep.CSRReqMessage) (*x509.Certificate, error) {
					signCalled = true
					return &x509.Certificate{}, nil
				},
			)

			mw := challengeMiddleware(ds, dummySigner)
			cert, err := mw.SignCSRContext(t.Context(), &scep.CSRReqMessage{
				ChallengePassword: tc.challenge,
			})

			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Nil(t, cert)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cert)
			}
			assert.Equal(t, tc.wantSignCalled, signCalled, "unexpected signer invocation")
			// The challenge is only consumed when a non-empty challenge is provided.
			assert.Equal(t, tc.challenge != "", consumeCalled, "unexpected ConsumeChallenge invocation")
		})
	}
}
