package depot_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/url"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepmock "github.com/fleetdm/fleet/v4/server/mock/scep"
	"github.com/stretchr/testify/require"
)

// Host identity and conditional access sign through Signx509CSR and rely on the CSR's Subject and SANs (conditional
// access reads the device UUID from a URI SAN), so that path must keep copying them.
func TestSignx509CSRCopiesSubjectAndSANs(t *testing.T) {
	caCert, caKey, err := depot.NewSCEPCACertKey()
	require.NoError(t, err)
	signer := depot.NewSigner(&scepmock.Depot{
		CAFunc: func([]byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
			return []*x509.Certificate{caCert}, caKey, nil
		},
		SerialFunc: func() (*big.Int, error) { return big.NewInt(1), nil },
		PutFunc:    func(string, *x509.Certificate) error { return nil },
	})

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	uri := &url.URL{Scheme: "urn", Opaque: "device:apple:uuid:abc"}
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: "host-identifier", Organization: []string{"Some Org"}},
		DNSNames: []string{"host.example.com"},
		URIs:     []*url.URL{uri},
	}, key)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(der)
	require.NoError(t, err)

	cert, err := signer.Signx509CSR(csr)
	require.NoError(t, err)
	require.Equal(t, "host-identifier", cert.Subject.CommonName)
	require.Equal(t, []string{"Some Org"}, cert.Subject.Organization)
	require.Equal(t, []string{"host.example.com"}, cert.DNSNames)
	require.Equal(t, []*url.URL{uri}, cert.URIs)
}
