package hostidentity

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenewalMiddleware_CNMismatchRejected(t *testing.T) {
	ctx := t.Context()

	// Generate a key pair for the "old" certificate
	oldKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	oldPubKeyRaw, err := types.CreateECDSAPublicKeyRaw(&oldKey.PublicKey)
	require.NoError(t, err)

	const originalCN = "original-host-identity"
	const serialNumber uint64 = 12345

	ds := new(mock.Store)
	ds.GetHostIdentityCertBySerialNumberFunc = func(_ context.Context, _ uint64) (*types.HostIdentityCertificate, error) {
		return &types.HostIdentityCertificate{
			SerialNumber:  serialNumber,
			CommonName:    originalCN,
			HostID:        new(uint),
			NotValidAfter: time.Now().Add(24 * time.Hour),
			PublicKeyRaw:  oldPubKeyRaw,
		}, nil
	}

	// The next signer should NOT be called when CN mismatches
	nextSignerCalled := false
	nextSigner := scepserver.CSRSignerContextFunc(func(_ context.Context, _ *scep.CSRReqMessage) (*x509.Certificate, error) {
		nextSignerCalled = true
		return &x509.Certificate{SerialNumber: big.NewInt(99999)}, nil
	})

	logger := slog.Default()
	middleware := renewalMiddleware(ds, logger, nextSigner)

	// Build renewal data signed by the old key
	serialHex := "0xc"
	hash := sha256.Sum256([]byte(serialHex))
	sig, err := ecdsa.SignASN1(rand.Reader, oldKey, hash[:])
	require.NoError(t, err)

	renewalData := types.RenewalData{
		SerialNumber: serialHex,
		Signature:    base64.StdEncoding.EncodeToString(sig),
	}
	renewalJSON, err := json.Marshal(renewalData)
	require.NoError(t, err)

	t.Run("mismatched CN is rejected", func(t *testing.T) {
		// Create a CSR with a DIFFERENT CN
		attackerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		csrTemplate := &x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "attacker-identity",
			},
			SignatureAlgorithm: x509.ECDSAWithSHA256,
			ExtraExtensions: []pkix.Extension{
				{
					Id:    types.RenewalExtensionOID,
					Value: renewalJSON,
				},
			},
		}
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, attackerKey)
		require.NoError(t, err)
		csr, err := x509.ParseCertificateRequest(csrDER)
		require.NoError(t, err)

		msg := &scep.CSRReqMessage{CSR: csr}
		_, err = middleware.SignCSRContext(ctx, msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "common name does not match")
		assert.False(t, nextSignerCalled, "next signer should not be called when CN mismatches")
	})

	t.Run("matching CN is accepted", func(t *testing.T) {
		nextSignerCalled = false

		newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		csrTemplate := &x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: originalCN,
			},
			SignatureAlgorithm: x509.ECDSAWithSHA256,
			ExtraExtensions: []pkix.Extension{
				{
					Id:    types.RenewalExtensionOID,
					Value: renewalJSON,
				},
			},
		}
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, newKey)
		require.NoError(t, err)
		csr, err := x509.ParseCertificateRequest(csrDER)
		require.NoError(t, err)

		ds.UpdateHostIdentityCertHostIDBySerialFunc = func(_ context.Context, _ uint64, _ uint) error {
			return nil
		}

		msg := &scep.CSRReqMessage{CSR: csr}
		_, err = middleware.SignCSRContext(ctx, msg)
		require.NoError(t, err)
		assert.True(t, nextSignerCalled, "next signer should be called when CN matches")
	})
}

func TestPKIOperationUndecryptableEnvelope(t *testing.T) {
	caCert, caKey, err := scepdepot.NewSCEPCACertKey()
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})

	ds := new(mock.Store)
	ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetHostIdentityCACert: {Name: fleet.MDMAssetHostIdentityCACert, Value: certPEM},
			fleet.MDMAssetHostIdentityCAKey:  {Name: fleet.MDMAssetHostIdentityCAKey, Value: keyPEM},
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

func TestPKIOperationRejectsDeeplyNestedBER(t *testing.T) {
	svc := NewSCEPService(nil, nil, slog.New(slog.DiscardHandler))

	deep := bytes.Repeat([]byte{0x30, 0x80}, cryptoutil.MaxBERDepth+1)
	_, err := svc.PKIOperation(t.Context(), deep)

	var badReq *fleet.BadRequestError
	require.ErrorAs(t, err, &badReq)
	require.ErrorIs(t, err, cryptoutil.ErrBERTooDeep)
}

func TestChallengeMiddleware(t *testing.T) {
	ctx := t.Context()
	const (
		validSecret = "valid-secret"
		hostName    = "host-identity-name"
	)

	newCSRMessage := func(t *testing.T, commonName, challenge string) *scep.CSRReqMessage {
		t.Helper()
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
			Subject:            pkix.Name{CommonName: commonName},
			SignatureAlgorithm: x509.ECDSAWithSHA256,
		}, key)
		require.NoError(t, err)
		csr, err := x509.ParseCertificateRequest(csrDER)
		require.NoError(t, err)
		return &scep.CSRReqMessage{CSR: csr, ChallengePassword: challenge}
	}

	testCases := []struct {
		name          string
		challenge     string
		existingCert  *types.HostIdentityCertificate
		lookupErr     error
		wantErrSubstr string
		wantSigned    bool
		wantNoLookup  bool
	}{
		{
			name:       "valid secret and no existing cert is signed",
			challenge:  validSecret,
			lookupErr:  &notFoundError{},
			wantSigned: true,
		},
		{
			name:          "valid secret does not replace a valid cert",
			challenge:     validSecret,
			existingCert:  &types.HostIdentityCertificate{SerialNumber: 1, CommonName: hostName},
			wantErrSubstr: "already exists",
		},
		{
			name:          "cert lookup failure fails closed",
			challenge:     validSecret,
			lookupErr:     errors.New("db down"),
			wantErrSubstr: "checking existing host identity certificate",
		},
		{
			name:          "missing challenge",
			challenge:     "",
			wantErrSubstr: "missing challenge",
			wantNoLookup:  true,
		},
		{
			name:          "invalid secret is rejected before the cert lookup",
			challenge:     "wrong",
			wantErrSubstr: "invalid challenge",
			wantNoLookup:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.VerifyEnrollSecretFunc = func(_ context.Context, secret string) (*fleet.EnrollSecret, error) {
				if secret != validSecret {
					return nil, &notFoundError{}
				}
				return &fleet.EnrollSecret{Secret: secret}, nil
			}
			ds.GetHostIdentityCertByNameFunc = func(ctx context.Context, name string) (*types.HostIdentityCertificate, error) {
				require.Equal(t, hostName, name)
				require.True(t, ctxdb.IsPrimaryRequired(ctx), "the existence check must not read from a replica")
				if tc.existingCert != nil {
					return tc.existingCert, nil
				}
				return nil, tc.lookupErr
			}

			signed := false
			next := scepserver.CSRSignerContextFunc(func(_ context.Context, _ *scep.CSRReqMessage) (*x509.Certificate, error) {
				signed = true
				return &x509.Certificate{SerialNumber: big.NewInt(2)}, nil
			})

			_, err := challengeMiddleware(ds, next).SignCSRContext(ctx, newCSRMessage(t, hostName, tc.challenge))
			if tc.wantErrSubstr != "" {
				require.ErrorContains(t, err, tc.wantErrSubstr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantSigned, signed)
			require.Equal(t, !tc.wantNoLookup, ds.GetHostIdentityCertByNameFuncInvoked)
		})
	}
}

type notFoundError struct{}

func (*notFoundError) Error() string    { return "not found" }
func (*notFoundError) IsNotFound() bool { return true }
