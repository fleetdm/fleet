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
	"fmt"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
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
	middleware := renewalMiddleware(ds, logger, false, nextSigner)

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

func TestRenewalMiddleware_ProofBoundToCSRKey(t *testing.T) {
	const originalCN = "original-host-identity"
	const serialHex = "0xc"

	oldKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	oldPubKeyRaw, err := types.CreateECDSAPublicKeyRaw(&oldKey.PublicKey)
	require.NoError(t, err)

	ds := new(mock.Store)
	ds.GetHostIdentityCertBySerialNumberFunc = func(_ context.Context, _ uint64) (*types.HostIdentityCertificate, error) {
		return &types.HostIdentityCertificate{
			SerialNumber:  12,
			CommonName:    originalCN,
			HostID:        new(uint),
			NotValidAfter: time.Now().Add(24 * time.Hour),
			PublicKeyRaw:  oldPubKeyRaw,
		}, nil
	}
	ds.UpdateHostIdentityCertHostIDBySerialFunc = func(_ context.Context, _ uint64, _ uint) error { return nil }
	var signed int
	nextSigner := scepserver.CSRSignerContextFunc(func(_ context.Context, _ *scep.CSRReqMessage) (*x509.Certificate, error) {
		signed++
		return &x509.Certificate{SerialNumber: big.NewInt(99999)}, nil
	})

	boundProof := func(key *ecdsa.PrivateKey) []byte {
		pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		require.NoError(t, err)
		return types.RenewalChallenge(serialHex, pubDER)
	}
	legacyProof := []byte(serialHex)

	renewAs := func(t *testing.T, cn string, requireKeyBinding bool, signedMsg []byte, csrKey *ecdsa.PrivateKey, badCSRSignature bool) (string, error) {
		hash := sha256.Sum256(signedMsg)
		sig, err := ecdsa.SignASN1(rand.Reader, oldKey, hash[:])
		require.NoError(t, err)
		renewalJSON, err := json.Marshal(types.RenewalData{SerialNumber: serialHex, Signature: base64.StdEncoding.EncodeToString(sig)})
		require.NoError(t, err)
		csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
			Subject:            pkix.Name{CommonName: cn},
			SignatureAlgorithm: x509.ECDSAWithSHA256,
			ExtraExtensions:    []pkix.Extension{{Id: types.RenewalExtensionOID, Value: renewalJSON}},
		}, csrKey)
		require.NoError(t, err)
		if badCSRSignature {
			csrDER[len(csrDER)-1] ^= 0xff // the signature is the last field of the CSR
		}
		csr, err := x509.ParseCertificateRequest(csrDER)
		require.NoError(t, err)

		var logs bytes.Buffer
		middleware := renewalMiddleware(ds, slog.New(slog.NewTextHandler(&logs, nil)), requireKeyBinding, nextSigner)
		_, err = middleware.SignCSRContext(t.Context(), &scep.CSRReqMessage{CSR: csr})
		return logs.String(), err
	}
	renew := func(t *testing.T, requireKeyBinding bool, signedMsg []byte, csrKey *ecdsa.PrivateKey, badCSRSignature bool) (string, error) {
		return renewAs(t, originalCN, requireKeyBinding, signedMsg, csrKey, badCSRSignature)
	}

	newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	attackerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	for _, requireKeyBinding := range []bool{false, true} {
		t.Run(fmt.Sprintf("requireKeyBinding=%t", requireKeyBinding), func(t *testing.T) {
			logs, err := renew(t, requireKeyBinding, boundProof(newKey), newKey, false)
			require.NoError(t, err, "bound proof for the CSR key is accepted")
			assert.NotContains(t, logs, "not bound")

			_, err = renew(t, requireKeyBinding, boundProof(newKey), newKey, true)
			require.ErrorContains(t, err, "invalid renewal CSR signature", "bound proof in a CSR that doesn't prove possession of its key is rejected")

			_, err = renew(t, requireKeyBinding, boundProof(newKey), attackerKey, false)
			require.ErrorContains(t, err, "invalid renewal signature", "bound proof replayed onto another key is rejected")

			before := signed
			_, err = renewAs(t, "other-host-identity", requireKeyBinding, boundProof(newKey), newKey, false)
			require.ErrorContains(t, err, "common name does not match", "bound proof can't renew into another identity")
			require.Equal(t, before, signed, "no certificate is signed for a mismatched CN")

			logs, err = renew(t, requireKeyBinding, legacyProof, attackerKey, false)
			if requireKeyBinding {
				require.ErrorContains(t, err, "renewal proof is not bound to the CSR key", "legacy proof is rejected when key binding is required")
				assert.Contains(t, logs, "legacy renewal proof rejected, upgrade fleetd")
			} else {
				require.NoError(t, err, "legacy proof is accepted until key binding is required")
				assert.Contains(t, logs, "renewal proof is not bound to the CSR key")
			}
		})
	}
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

func TestChallengeMiddlewareExistingCert(t *testing.T) {
	newKey := func() *ecdsa.PrivateKey {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		return key
	}
	existingKey := newKey()
	existingKeyRaw, err := types.CreateECDSAPublicKeyRaw(&existingKey.PublicKey)
	require.NoError(t, err)
	unenrolledCert := &types.HostIdentityCertificate{SerialNumber: 1, PublicKeyRaw: existingKeyRaw}
	enrolledCert := &types.HostIdentityCertificate{SerialNumber: 1, HostID: new(uint(7)), PublicKeyRaw: existingKeyRaw}

	testCases := []struct {
		name         string
		cert         *types.HostIdentityCertificate
		lookupErr    error
		csrKey       *ecdsa.PrivateKey
		badSignature bool
		wantErr      string
	}{
		{name: "no existing cert", lookupErr: common_mysql.NotFound("host identity certificate"), csrKey: newKey()},
		{name: "existing cert, different key", cert: unenrolledCert, csrKey: newKey(), wantErr: "already exists"},
		{name: "existing cert, same key", cert: unenrolledCert, csrKey: existingKey},
		{name: "existing cert, same key but invalid signature", cert: unenrolledCert, csrKey: existingKey, badSignature: true, wantErr: "already exists"},
		{name: "enrolled cert, same key", cert: enrolledCert, csrKey: existingKey, wantErr: "already exists"},
		{name: "lookup failure fails closed", lookupErr: errors.New("db down"), csrKey: newKey(), wantErr: "checking existing host identity"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.VerifyEnrollSecretFunc = func(_ context.Context, secret string) (*fleet.EnrollSecret, error) {
				return &fleet.EnrollSecret{Secret: secret}, nil
			}
			ds.GetHostIdentityCertByNameFunc = func(_ context.Context, _ string) (*types.HostIdentityCertificate, error) {
				return tc.cert, tc.lookupErr
			}
			signed := false
			next := scepserver.CSRSignerContextFunc(func(_ context.Context, _ *scep.CSRReqMessage) (*x509.Certificate, error) {
				signed = true
				return &x509.Certificate{SerialNumber: big.NewInt(2)}, nil
			})

			csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
				Subject: pkix.Name{CommonName: "host-identity-name"}, SignatureAlgorithm: x509.ECDSAWithSHA256,
			}, tc.csrKey)
			require.NoError(t, err)
			if tc.badSignature {
				csrDER[len(csrDER)-1] ^= 0xff // the signature is the last field of the CSR
			}
			csr, err := x509.ParseCertificateRequest(csrDER)
			require.NoError(t, err)

			_, err = challengeMiddleware(ds, next).SignCSRContext(t.Context(), &scep.CSRReqMessage{CSR: csr, ChallengePassword: "secret"})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantErr == "", signed)
		})
	}
}
