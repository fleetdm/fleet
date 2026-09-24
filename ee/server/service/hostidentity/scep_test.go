package hostidentity

import (
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
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
