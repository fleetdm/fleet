//go:build !windows

// Windows is disabled because the TPM simulator requires CGO, which causes lint failures on Windows.

package hostidentity

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/securehw"
	"github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/google/go-tpm/tpm2/transport/simulator"
	"github.com/rs/zerolog"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/require"
)

// capsService overrides GetCACaps to simulate servers with and without key-bound renewal support, and failed caps requests.
type capsService struct {
	scepserver.Service
	caps string
	err  error
}

func (s capsService) GetCACaps(context.Context) ([]byte, error) { return []byte(s.caps), s.err }

func TestRenewCertificateProofFormat(t *testing.T) {
	const legacyCaps = "SHA-256\nAES\nPOSTPKIOperation"
	cases := []struct {
		name         string
		caps         string
		capsErr      error
		accept       string // proof format the server verifies; "" rejects every request
		wantAttempts []string
	}{
		{"server verifies bound proofs", legacyCaps + "\n" + types.RenewalKeyBindingCapability, nil, "bound", []string{"bound"}},
		{"server predates bound proofs", legacyCaps, nil, "legacy", []string{"bound", "legacy"}},
		{"server with bound proofs rejects for another reason", legacyCaps + "\n" + types.RenewalKeyBindingCapability, nil, "", []string{"bound"}},
		{"caps request fails", "", errors.New("caps unavailable"), "", []string{"bound"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			logger := zerolog.New(zerolog.NewTestWriter(t))
			sim, err := simulator.OpenSimulator()
			require.NoError(t, err)
			metadataDir := t.TempDir()
			hw, err := securehw.NewTestSecureHW(sim, metadataDir, logger)
			require.NoError(t, err)
			t.Cleanup(func() { _ = hw.Close() })
			oldKey, err := hw.CreateKey()
			require.NoError(t, err)
			oldPub, err := oldKey.Public()
			require.NoError(t, err)

			oldCert := &x509.Certificate{SerialNumber: big.NewInt(12), Subject: pkix.Name{CommonName: "test-host"}}
			serialHex := "0x0c"

			caCert, caKey, err := scepdepot.NewSCEPCACertKey()
			require.NoError(t, err)
			var attempts []string
			signer := scepserver.CSRSignerContextFunc(func(_ context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
				var data types.RenewalData
				for _, ext := range m.CSR.Extensions {
					if ext.Id.Equal(types.RenewalExtensionOID) {
						if err := json.Unmarshal(ext.Value, &data); err != nil {
							return nil, err
						}
					}
				}
				sig, err := base64.StdEncoding.DecodeString(data.Signature)
				if err != nil || data.SerialNumber != serialHex {
					return nil, errors.New("bad renewal extension")
				}
				format := "invalid"
				for name, msg := range map[string][]byte{
					"bound":  types.RenewalChallenge(serialHex, m.CSR.RawSubjectPublicKeyInfo),
					"legacy": []byte(serialHex),
				} {
					hash := sha256.Sum256(msg)
					if ecdsa.VerifyASN1(oldPub.(*ecdsa.PublicKey), hash[:], sig) {
						format = name
					}
				}
				attempts = append(attempts, format)
				if format != c.accept {
					return nil, errors.New("rejected")
				}
				certDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
					SerialNumber: big.NewInt(13),
					Subject:      m.CSR.Subject,
					NotBefore:    time.Now(),
					NotAfter:     time.Now().Add(time.Hour),
				}, caCert, m.CSR.PublicKey, caKey)
				if err != nil {
					return nil, err
				}
				return x509.ParseCertificate(certDER)
			})
			svc, err := scepserver.NewService(caCert, caKey, signer)
			require.NoError(t, err)
			svc = capsService{Service: svc, caps: c.caps, err: c.capsErr}
			srv := httptest.NewServer(scepserver.MakeHTTPHandler(scepserver.MakeServerEndpoints(svc), svc, slog.New(slog.DiscardHandler)))
			t.Cleanup(srv.Close)

			credentials := &Credentials{Certificate: oldCert, SecureHWKey: oldKey, SecureHW: hw}
			cert, err := RenewCertificate(t.Context(), metadataDir, credentials, srv.URL, "", false, logger)
			require.Equal(t, c.wantAttempts, attempts)
			if c.accept == "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "test-host", cert.Subject.CommonName)
		})
	}
}
