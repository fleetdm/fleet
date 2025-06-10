package mysql

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostCertificates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"UpdateAndList", testUpdateAndListHostCertificates},
		{"Update with host_mdm_managed_certificates to update", testUpdatingHostMDMManagedCertificates},
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

	require.NoError(t, ds.UpdateHostCertificates(context.Background(), 1, "95816502-d8c0-462c-882f-39991cc89a0c", payload))

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
	require.NoError(t, ds.UpdateHostCertificates(context.Background(), 1, "95816502-d8c0-462c-882f-39991cc89a0c", []*fleet.HostCertificateRecord{payload[1]}))
	certs3, _, err := ds.ListHostCertificates(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, certs3, 1)
	require.Equal(t, expected2.Subject.CommonName, certs3[0].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs3[0].SubjectCommonName)
}

func testUpdatingHostMDMManagedCertificates(t *testing.T, ds *Datastore) {
	// test that we can update the host_mdm_managed_certificates table when
	// ingesting the associated certificate from the host
	ctx := context.Background()
	initialCP := storeDummyConfigProfileForTest(t, ds)
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host0-osquery-id"),
		NodeKey:         ptr.String("host0-node-key"),
		UUID:            "host0-test-mdm-profiles",
		Hostname:        "hostname0",
	})
	require.NoError(t, err)

	err = ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       initialCP.ProfileUUID,
			ProfileIdentifier: initialCP.Identifier,
			ProfileName:       initialCP.Name,
			HostUUID:          host.UUID,
			Status:            &fleet.MDMDeliveryPending,
			OperationType:     fleet.MDMOperationTypeInstall,
			CommandUUID:       "command-uuid",
			Checksum:          []byte("checksum"),
			Scope:             fleet.PayloadScopeSystem,
		},
	},
	)
	require.NoError(t, err)

	// Initial certificate state where a host has been requested to install but we have no metadata
	challengeRetrievedAt := time.Now().Add(-time.Hour).UTC().Round(time.Microsecond)
	err = ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
		{
			HostUUID:             host.UUID,
			ProfileUUID:          initialCP.ProfileUUID,
			ChallengeRetrievedAt: &challengeRetrievedAt,
			Type:                 fleet.CAConfigCustomSCEPProxy,
			CAName:               "test-ca",
		},
	})
	require.NoError(t, err)

	expected1 := x509.Certificate{
		Subject: pkix.Name{
			Country:      []string{"US"},
			CommonName:   "MYHWSERIAL fleet-" + initialCP.ProfileUUID,
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
			CommonName:         "MYOTHERHWSERIAL",
			Organization:       []string{"Org 2"},
			OrganizationalUnit: []string{"Engineering"},
		},
		SerialNumber: big.NewInt(31337),
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

	require.NoError(t, ds.UpdateHostCertificates(context.Background(), host.ID, host.UUID, payload))

	// verify that we saved the records correctly
	certs, _, err := ds.ListHostCertificates(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, certs, 2)
	// default ordering is by common name ascending
	require.Equal(t, expected1.Subject.CommonName, certs[0].CommonName)
	require.Equal(t, expected1.Subject.CommonName, certs[0].SubjectCommonName)
	require.Equal(t, expected2.Subject.CommonName, certs[1].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs[1].SubjectCommonName)

	// Check that the managed certificate details were updated correctly
	profile, err := ds.GetHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, "test-ca")
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, host.UUID, profile.HostUUID)
	assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
	require.NotNil(t, profile.ChallengeRetrievedAt)
	assert.Equal(t, &challengeRetrievedAt, profile.ChallengeRetrievedAt)
	assert.Equal(t, fleet.CAConfigCustomSCEPProxy, profile.Type)
	require.NotNil(t, profile.Serial)
	assert.Equal(t, fmt.Sprintf("%040s", expected1.SerialNumber.Text(16)), *profile.Serial)
	require.NotNil(t, profile.NotValidBefore)
	assert.Equal(t, expected1.NotBefore, *profile.NotValidBefore)
	require.NotNil(t, profile.NotValidAfter)
	assert.Equal(t, expected1.NotAfter, *profile.NotValidAfter)
	assert.Equal(t, "test-ca", profile.CAName)

	// simulate removal of a certificate
	require.NoError(t, ds.UpdateHostCertificates(context.Background(), 1, "95816502-d8c0-462c-882f-39991cc89a0c", []*fleet.HostCertificateRecord{payload[1]}))
	certs3, _, err := ds.ListHostCertificates(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, certs3, 1)
	require.Equal(t, expected2.Subject.CommonName, certs3[0].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs3[0].SubjectCommonName)

	// Check that the managed certificate details were not updated
	profile, err = ds.GetHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, "test-ca")
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, host.UUID, profile.HostUUID)
	assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
	require.NotNil(t, profile.ChallengeRetrievedAt)
	assert.Equal(t, &challengeRetrievedAt, profile.ChallengeRetrievedAt)
	assert.Equal(t, fleet.CAConfigCustomSCEPProxy, profile.Type)
	require.NotNil(t, profile.Serial)
	assert.Equal(t, fmt.Sprintf("%040s", expected1.SerialNumber.Text(16)), *profile.Serial)
	require.NotNil(t, profile.NotValidBefore)
	assert.Equal(t, expected1.NotBefore, *profile.NotValidBefore)
	require.NotNil(t, profile.NotValidAfter)
	assert.Equal(t, expected1.NotAfter, *profile.NotValidAfter)
	assert.Equal(t, "test-ca", profile.CAName)
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
