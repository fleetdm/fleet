package scepserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/smallstep/scep"
	"github.com/stretchr/testify/require"
)

func FuzzPKIOperation(f *testing.F) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(f, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "SCEP fuzz CA"},
		NotBefore:             time.Unix(0, 0),
		NotAfter:              time.Unix(4102444800, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(f, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(f, err)
	fixture, err := os.ReadFile("../testdata/PKCSReq.der")
	require.NoError(f, err)
	f.Add(fixture, uint8(0), false)
	f.Add([]byte{}, uint8(0), false)
	f.Add([]byte("device"), uint8(1), false)
	f.Add([]byte("device"), uint8(1), true)
	f.Add([]byte{0x30, 0x00}, uint8(2), false)

	f.Fuzz(func(t *testing.T, data []byte, mode uint8, reject bool) {
		if len(data) > 4096 {
			t.Skip()
		}
		mode %= 3
		called := false
		svc, err := NewService(cert, key, CSRSignerContextFunc(func(_ context.Context, msg *scep.CSRReqMessage) (*x509.Certificate, error) {
			called = true
			require.NotNil(t, msg)
			require.NotNil(t, msg.CSR)
			if mode == 1 {
				require.Equal(t, base64.RawURLEncoding.EncodeToString(data), msg.CSR.Subject.CommonName)
				require.NoError(t, msg.CSR.CheckSignature())
			}
			if reject {
				return nil, errors.New("fuzz signer rejected CSR")
			}
			return cert, nil
		}))
		require.NoError(t, err)

		request := data
		if mode != 0 {
			csr := &x509.CertificateRequest{PublicKey: &key.PublicKey, Raw: data}
			if mode == 1 {
				csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
					Subject: pkix.Name{CommonName: base64.RawURLEncoding.EncodeToString(data)},
				}, key)
				require.NoError(t, err)
				csr, err = x509.ParseCertificateRequest(csrDER)
				require.NoError(t, err)
			}
			msg, err := scep.NewCSRRequest(csr, &scep.PKIMessage{
				MessageType: scep.PKCSReq,
				Recipients:  []*x509.Certificate{cert},
				SignerKey:   key,
				SignerCert:  cert,
			})
			require.NoError(t, err)
			request = msg.Raw
		}
		reply, err := svc.PKIOperation(t.Context(), request)
		if mode == 1 {
			require.NoError(t, err)
			require.True(t, called)
		}
		if err != nil {
			return
		}
		msg, err := scep.ParsePKIMessage(reply)
		require.NoError(t, err)
		require.Equal(t, scep.CertRep, msg.MessageType)
		if mode == 1 && !reject {
			require.Equal(t, scep.SUCCESS, msg.PKIStatus)
			require.NoError(t, msg.DecryptPKIEnvelope(cert, key))
			require.Equal(t, cert.Raw, msg.Certificate.Raw)
		} else if reject || !called {
			require.Equal(t, scep.FAILURE, msg.PKIStatus)
		}
	})
}
