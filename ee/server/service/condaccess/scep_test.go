package condaccess

import (
	"context"
	"crypto/x509"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallengeMiddleware(t *testing.T) {
	teamAID := uint(1)
	teamBID := uint(2)

	cases := []struct {
		name           string
		challenge      string
		wantErr        string
		wantSignCalled bool
	}{
		{
			name:      "empty challenge is rejected",
			challenge: "",
			wantErr:   "missing challenge",
		},
		{
			name:      "unknown secret is rejected",
			challenge: "unknown-secret",
			wantErr:   "invalid challenge",
		},
		{
			name:      "team-scoped secret is rejected",
			challenge: "secret-team-a",
			wantErr:   "invalid challenge",
		},
		{
			name:      "different team-scoped secret is also rejected",
			challenge: "secret-team-b",
			wantErr:   "invalid challenge",
		},
		{
			name:           "global secret is accepted",
			challenge:      "global-secret",
			wantSignCalled: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.DataStore)
			ds.VerifyEnrollSecretFunc = func(_ context.Context, secret string) (*fleet.EnrollSecret, error) {
				switch secret {
				case "secret-team-a":
					return &fleet.EnrollSecret{Secret: secret, TeamID: &teamAID}, nil
				case "secret-team-b":
					return &fleet.EnrollSecret{Secret: secret, TeamID: &teamBID}, nil
				case "global-secret":
					return &fleet.EnrollSecret{Secret: secret, TeamID: nil}, nil
				default:
					return nil, common_mysql.NotFound("enroll_secret")
				}
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
		})
	}
}
