package mysql

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestHostCertificates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"UpdateAndList", testUpdateAndListHostCertificates},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testUpdateAndListHostCertificates(t *testing.T, ds *Datastore) {
	expected1 := x509.Certificate{
		Subject: pkix.Name{
			Country:      []string{"US"},
			CommonName:   "test.example.com",
			Organization: []string{"Org 1"},

			OrganizationalUnit: []string{"Engineering"},
		},
		Issuer: pkix.Name{
			Country:      []string{"US"},
			CommonName:   "issuer.test.example.com",
			Organization: []string{"Issuer 1"},
		},
		SerialNumber: big.NewInt(1337),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now().Add(-time.Hour).Truncate(time.Second).UTC(),
		NotAfter:              time.Now().Add(24 * time.Hour).Truncate(time.Second).UTC(),
		BasicConstraintsValid: true,
	}

	expected2 := x509.Certificate{
		Subject: pkix.Name{
			Country:            []string{"US"},
			CommonName:         "another.test.example.com",
			Organization:       []string{"Org 2"},
			OrganizationalUnit: []string{"Engineering"},
		},
		SerialNumber: big.NewInt(1337),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now().Add(-2 * time.Hour).Truncate(time.Second).UTC(),
		NotAfter:              time.Now().Add(48 * time.Hour).Truncate(time.Second).UTC(),
		BasicConstraintsValid: true,
	}

	payload := []*fleet.HostCertificateRecord{
		generateTestHostCertificateRecord(t, 1, &expected1),
		generateTestHostCertificateRecord(t, 1, &expected2),
	}

	require.NoError(t, ds.UpdateHostCertificates(context.Background(), 1, payload))

	// verify that we saved the records correctly
	certs, _, err := ds.ListHostCertificates(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, certs, 2)
	// default ordering is by common name ascending
	require.Equal(t, expected2.Subject.CommonName, certs[0].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs[0].SubjectCommonName)
	require.Equal(t, expected1.Subject.CommonName, certs[1].CommonName)
	require.Equal(t, expected1.Subject.CommonName, certs[1].SubjectCommonName)

	// order by not_valid_after descending
	certs2, _, err := ds.ListHostCertificates(context.Background(), 1, fleet.ListOptions{OrderKey: "not_valid_after", OrderDirection: fleet.OrderAscending})
	require.NoError(t, err)
	require.Len(t, certs2, 2)
	require.Equal(t, expected1.Subject.CommonName, certs2[0].CommonName)
	require.Equal(t, expected1.Subject.CommonName, certs2[0].SubjectCommonName)
	require.Equal(t, expected2.Subject.CommonName, certs2[1].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs2[1].SubjectCommonName)

	// simulate removal of a certificate
	require.NoError(t, ds.UpdateHostCertificates(context.Background(), 1, []*fleet.HostCertificateRecord{payload[1]}))
	certs3, _, err := ds.ListHostCertificates(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, certs3, 1)
	require.Equal(t, expected2.Subject.CommonName, certs3[0].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs3[0].SubjectCommonName)
}

func generateTestHostCertificateRecord(t *testing.T, hostID uint, template *x509.Certificate) *fleet.HostCertificateRecord {
	b, _, err := GenerateTestCertBytes(template)
	require.NoError(t, err)

	block, _ := pem.Decode(b)

	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	return fleet.NewHostCertificateRecord(hostID, parsed)
}
