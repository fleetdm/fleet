package condaccess

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
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

func TestPKIOperationUndecryptableEnvelope(t *testing.T) {
	caCert, caKey, err := scepdepot.NewSCEPCACertKey()
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})

	ds := new(mock.DataStore)
	ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetConditionalAccessCACert: {Name: fleet.MDMAssetConditionalAccessCACert, Value: certPEM},
			fleet.MDMAssetConditionalAccessCAKey:  {Name: fleet.MDMAssetConditionalAccessCAKey, Value: keyPEM},
		}, nil
	}

	signerCalled := false
	signer := scepserver.CSRSignerContextFunc(func(context.Context, *scep.CSRReqMessage) (*x509.Certificate, error) {
		signerCalled = true
		return &x509.Certificate{}, nil
	})
	svc := NewSCEPService(ds, signer, slog.New(slog.DiscardHandler))

	req, err := mdmtest.NewPKCSReqUndecryptableBy(caCert)
	require.NoError(t, err)

	respBytes, err := svc.PKIOperation(t.Context(), req.Raw)
	require.NoError(t, err)

	certRep, err := scep.ParsePKIMessage(respBytes)
	require.NoError(t, err)
	assert.Equal(t, scep.FAILURE, certRep.PKIStatus)
	assert.Equal(t, scep.BadRequest, certRep.FailInfo)
	assert.Equal(t, req.TransactionID, certRep.TransactionID)
	assert.False(t, signerCalled, "signer must not run when the envelope cannot be decrypted")
}
