package mysql

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand/v2"
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
		{"Update certificate sources isolation", testUpdateHostCertificatesSourcesIsolation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testUpdateAndListHostCertificates(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	createX509Cert := func(commonName string, notAfter time.Duration) x509.Certificate {
		return x509.Certificate{
			Subject: pkix.Name{
				Country:            []string{"US"},
				CommonName:         commonName,
				Organization:       []string{"Org"},
				OrganizationalUnit: []string{"Engineering"},
			},
			Issuer: pkix.Name{
				Country:      []string{"US"},
				CommonName:   "issuer.test.example.com",
				Organization: []string{"Issuer"},
			},
			SerialNumber: big.NewInt(rand.Int64()), // nolint:gosec
			KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

			SignatureAlgorithm:    x509.SHA256WithRSA,
			NotBefore:             time.Now().Add(-time.Hour).Truncate(time.Second).UTC(),
			NotAfter:              time.Now().Add(notAfter).Truncate(time.Second).UTC(),
			BasicConstraintsValid: true,
		}
	}

	expected1 := createX509Cert("test.example.com", 24*time.Hour)
	expected2 := createX509Cert("another.test.example.com", 48*time.Hour)

	payload := []*fleet.HostCertificateRecord{
		generateTestHostCertificateRecord(t, 1, &expected1),
		generateTestHostCertificateRecord(t, 1, &expected2),
	}

	require.NoError(t, ds.UpdateHostCertificates(ctx, 1, "95816502-d8c0-462c-882f-39991cc89a0c", payload))

	// verify that we saved the records correctly
	certs, _, err := ds.ListHostCertificates(ctx, 1, fleet.ListOptions{OrderKey: "common_name"})
	require.NoError(t, err)
	require.Len(t, certs, 2)
	require.Equal(t, expected2.Subject.CommonName, certs[0].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs[0].SubjectCommonName)
	require.Equal(t, fleet.SystemHostCertificate, certs[0].Source)
	require.Equal(t, expected1.Subject.CommonName, certs[1].CommonName)
	require.Equal(t, expected1.Subject.CommonName, certs[1].SubjectCommonName)
	require.Equal(t, fleet.SystemHostCertificate, certs[1].Source)

	// order by not_valid_after descending
	certs, _, err = ds.ListHostCertificates(ctx, 1, fleet.ListOptions{OrderKey: "not_valid_after", OrderDirection: fleet.OrderAscending})
	require.NoError(t, err)
	require.Len(t, certs, 2)
	require.Equal(t, expected1.Subject.CommonName, certs[0].CommonName)
	require.Equal(t, expected1.Subject.CommonName, certs[0].SubjectCommonName)
	require.Equal(t, expected2.Subject.CommonName, certs[1].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs[1].SubjectCommonName)

	// simulate removal of a certificate
	require.NoError(t, ds.UpdateHostCertificates(ctx, 1, "95816502-d8c0-462c-882f-39991cc89a0c", []*fleet.HostCertificateRecord{payload[1]}))
	certs, _, err = ds.ListHostCertificates(ctx, 1, fleet.ListOptions{OrderKey: "common_name"})
	require.NoError(t, err)
	require.Len(t, certs, 1)
	require.Equal(t, expected2.Subject.CommonName, certs[0].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs[0].SubjectCommonName)

	// re-add first certificate but as a "user" source
	payload[0].Source = fleet.UserHostCertificate
	payload[0].Username = "A"
	require.NoError(t, ds.UpdateHostCertificates(ctx, 1, "95816502-d8c0-462c-882f-39991cc89a0c", []*fleet.HostCertificateRecord{payload[0], payload[1]}))
	certs, _, err = ds.ListHostCertificates(ctx, 1, fleet.ListOptions{OrderKey: "common_name"})
	require.NoError(t, err)
	require.Len(t, certs, 2)
	require.Equal(t, expected2.Subject.CommonName, certs[0].CommonName)
	require.Equal(t, expected2.Subject.CommonName, certs[0].SubjectCommonName)
	require.Equal(t, fleet.SystemHostCertificate, certs[0].Source)
	require.Equal(t, "", certs[0].Username)
	require.Equal(t, expected1.Subject.CommonName, certs[1].CommonName)
	require.Equal(t, expected1.Subject.CommonName, certs[1].SubjectCommonName)
	require.Equal(t, fleet.UserHostCertificate, certs[1].Source)
	require.Equal(t, "A", certs[1].Username)

	hostCert1SrcUserA := payload[0]
	hostCert2SrcSys := payload[1]
	expected3 := createX509Cert("multi.test.example.com", 24*time.Hour)
	hostCert3SrcUserB := generateTestHostCertificateRecord(t, 1, &expected3)
	hostCert3SrcUserB.Source = fleet.UserHostCertificate
	hostCert3SrcUserB.Username = "B"
	cloneC := *hostCert3SrcUserB // copy to create a new record
	hostCert3SrcUserC := &cloneC
	hostCert3SrcUserC.Source = fleet.UserHostCertificate
	hostCert3SrcUserC.Username = "C"
	cloneD := *hostCert3SrcUserB // copy to create a new record
	hostCert3SrcUserD := &cloneD
	hostCert3SrcUserD.Source = fleet.UserHostCertificate
	hostCert3SrcUserD.Username = "D"
	cases := []struct {
		desc   string
		ingest []*fleet.HostCertificateRecord
	}{
		{desc: "nil slice", ingest: nil},
		{desc: "cert 1 and 2", ingest: []*fleet.HostCertificateRecord{hostCert2SrcSys, hostCert1SrcUserA}},
		{desc: "cert 2 and 3 (B, C)", ingest: []*fleet.HostCertificateRecord{hostCert2SrcSys, hostCert3SrcUserB, hostCert3SrcUserC}},
		{desc: "no change", ingest: []*fleet.HostCertificateRecord{hostCert2SrcSys, hostCert3SrcUserB, hostCert3SrcUserC}},
		{desc: "added cert 3 source (D)", ingest: []*fleet.HostCertificateRecord{hostCert2SrcSys, hostCert3SrcUserB, hostCert3SrcUserC, hostCert3SrcUserD}},
		{desc: "removed cert3 source (B)", ingest: []*fleet.HostCertificateRecord{hostCert2SrcSys, hostCert3SrcUserC, hostCert3SrcUserD}},
		{desc: "removed cert3 source (C)", ingest: []*fleet.HostCertificateRecord{hostCert2SrcSys, hostCert3SrcUserB, hostCert3SrcUserD}},
		{desc: "cleared, added cert 1", ingest: []*fleet.HostCertificateRecord{hostCert1SrcUserA}},
		{desc: "all cleared", ingest: []*fleet.HostCertificateRecord{}},
	}
	for _, c := range cases {
		t.Log(c.desc)

		err := ds.UpdateHostCertificates(ctx, 1, "95816502-d8c0-462c-882f-39991cc89a0c", c.ingest)
		require.NoError(t, err)
		certs, _, err := ds.ListHostCertificates(ctx, 1, fleet.ListOptions{OrderKey: "common_name", TestSecondaryOrderKey: "username"})
		require.NoError(t, err)

		require.Len(t, certs, len(c.ingest))
		for i, cert := range certs {
			require.Equal(t, c.ingest[i].CommonName, cert.CommonName, "index %d", i)
			require.Equal(t, c.ingest[i].Source, cert.Source, "index %d", i)
			require.Equal(t, c.ingest[i].Username, cert.Username, "index %d", i)
		}
	}
}

func testUpdatingHostMDMManagedCertificates(t *testing.T, ds *Datastore) {
	// test that we can update the host_mdm_managed_certificates table when
	// ingesting the associated certificate from the host
	ctx := context.Background()
	initialCP := storeDummyConfigProfilesForTest(t, ds, 1)[0]
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
	certs, _, err := ds.ListHostCertificates(context.Background(), 1, fleet.ListOptions{OrderKey: "common_name"})
	require.NoError(t, err)
	require.Len(t, certs, 2)
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
	certs3, _, err := ds.ListHostCertificates(context.Background(), 1, fleet.ListOptions{OrderKey: "common_name"})
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

func testUpdateHostCertificatesSourcesIsolation(t *testing.T, ds *Datastore) {
	// regression test for #30574
	ctx := context.Background()

	host1, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host1-osquery-id"),
		NodeKey:         ptr.String("host1-node-key"),
		UUID:            "host1-uuid",
		Hostname:        "host1",
	})
	require.NoError(t, err)

	host2, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host2-osquery-id"),
		NodeKey:         ptr.String("host2-node-key"),
		UUID:            "host2-uuid",
		Hostname:        "host2",
	})
	require.NoError(t, err)

	// Create identical certificates for both hosts (same SHA1 sum)
	// This simulates the real-world scenario where multiple hosts have the same certificate
	// installed (e.g., a company root CA certificate)
	sharedCert := x509.Certificate{
		Subject: pkix.Name{
			Country:            []string{"US"},
			CommonName:         "shared.example.com",
			Organization:       []string{"Shared Org"},
			OrganizationalUnit: []string{"Engineering"},
		},
		Issuer: pkix.Name{
			Country:      []string{"US"},
			CommonName:   "issuer.example.com",
			Organization: []string{"Issuer"},
		},
		SerialNumber: big.NewInt(12345),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now().Add(-time.Hour).Truncate(time.Second).UTC(),
		NotAfter:              time.Now().Add(24 * time.Hour).Truncate(time.Second).UTC(),
		BasicConstraintsValid: true,
	}

	// Generate certificate records for both hosts using the same certificate data
	// We need to create the certificate bytes once and reuse them to ensure same SHA1
	certBytes, _, err := GenerateTestCertBytes(&sharedCert)
	require.NoError(t, err)

	block, _ := pem.Decode(certBytes)
	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	host1Cert := fleet.NewHostCertificateRecord(host1.ID, parsed)
	host1Cert.Source = fleet.UserHostCertificate
	host1Cert.Username = "jdoe"

	host2Cert := fleet.NewHostCertificateRecord(host2.ID, parsed)
	host2Cert.Source = fleet.UserHostCertificate
	host2Cert.Username = "jsmith"

	// Add the same certificate to both hosts
	require.NoError(t, ds.UpdateHostCertificates(ctx, host1.ID, host1.UUID, []*fleet.HostCertificateRecord{host1Cert}))
	require.NoError(t, ds.UpdateHostCertificates(ctx, host2.ID, host2.UUID, []*fleet.HostCertificateRecord{host2Cert}))

	// Verify both hosts have the correct certs, with the correct sources
	host1Certs, _, err := ds.ListHostCertificates(ctx, host1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, host1Certs, 1)
	require.Equal(t, fleet.UserHostCertificate, host1Certs[0].Source)
	require.Equal(t, "jdoe", host1Certs[0].Username)

	host2Certs, _, err := ds.ListHostCertificates(ctx, host2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, host2Certs, 1)
	require.Equal(t, fleet.UserHostCertificate, host2Certs[0].Source)
	require.Equal(t, "jsmith", host2Certs[0].Username)

	// Trigger a change in host 2's cert sources to force delete/recreate of source records. Prior to the fix this
	// wouldn't modify the correct certs because the DB query would return host 1's cert for the given hash.
	host2CertUpdated := fleet.NewHostCertificateRecord(host2.ID, parsed)
	host2CertUpdated.Source = fleet.UserHostCertificate
	host2CertUpdated.Username = "janesmith"

	require.NoError(t, ds.UpdateHostCertificates(ctx, host2.ID, host2.UUID, []*fleet.HostCertificateRecord{host2CertUpdated}))

	// Verify host1's certificate source was *not* updated
	host1CertsAfter, _, err := ds.ListHostCertificates(ctx, host1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, host1CertsAfter, 1)
	require.Equal(t, "jdoe", host1CertsAfter[0].Username)

	// Verify host2's certificate source *was* updated
	host2CertsAfter, _, err := ds.ListHostCertificates(ctx, host2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, host2CertsAfter, 1)
	require.Equal(t, "janesmith", host2CertsAfter[0].Username)

	// Verify no-op case
	err = ds.UpdateHostCertificates(ctx, host2.ID, host2.UUID, []*fleet.HostCertificateRecord{host2CertUpdated})
	require.NoError(t, err)

	// Verify host2's certificate source was updated
	host2CertsNoop, _, err := ds.ListHostCertificates(ctx, host2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, host2CertsNoop, 1)
	require.Equal(t, fleet.UserHostCertificate, host2CertsNoop[0].Source)
	require.Equal(t, "janesmith", host2CertsNoop[0].Username)

	// Confirm that adding the cert to the system store gets picked up properly
	systemCertOnHost2 := fleet.NewHostCertificateRecord(host2.ID, parsed)
	systemCertOnHost2.Source = fleet.SystemHostCertificate

	require.NoError(t, ds.UpdateHostCertificates(ctx, host2.ID, host2.UUID, []*fleet.HostCertificateRecord{host2CertUpdated, systemCertOnHost2}))

	// Verify host2 now has the certificate with both sources
	host2CertsMultiSource, _, err := ds.ListHostCertificates(ctx, host2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, host2CertsMultiSource, 2)
	var hasUserCert, hasSystemCert bool
	for _, cert := range host2CertsMultiSource {
		if cert.Source == fleet.UserHostCertificate {
			require.Equal(t, "janesmith", cert.Username)
			hasUserCert = true
		} else {
			require.Empty(t, cert.Username)
			require.Equal(t, fleet.SystemHostCertificate, cert.Source)
			hasSystemCert = true
		}
	}
	require.True(t, hasUserCert)
	require.True(t, hasSystemCert)

	// Verify host1 still has only its original certificate source
	host1CertsMultiSource, _, err := ds.ListHostCertificates(ctx, host1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, host1CertsMultiSource, 1)
	require.Equal(t, fleet.UserHostCertificate, host1CertsMultiSource[0].Source)
	require.Equal(t, "jdoe", host1CertsMultiSource[0].Username)
}
