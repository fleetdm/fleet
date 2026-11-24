package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAndroidCACertificates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"CRUD", testAndroidCACertificatesCRUD},
		{"LabelScoping", testAndroidCACertificatesLabelScoping},
		{"HostStatusTracking", testAndroidCAHostStatusTracking},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testAndroidCACertificatesCRUD(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_Team"})
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)
	notBefore := now.Add(-1 * time.Hour)
	notAfter := now.Add(24 * time.Hour)

	cert := &fleet.AndroidCACertificate{
		TeamID:            team.ID,
		Name:              "Corporate Root CA",
		CertificatePEM:    "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
		SHA256Fingerprint: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Subject:           "CN=Corporate Root CA, O=Corp",
		Issuer:            "CN=Corporate Root CA, O=Corp",
		NotValidBefore:    &notBefore,
		NotValidAfter:     &notAfter,
		LabelsIncludeAll:  []string{"LabelA"},
		LabelsIncludeAny:  []string{"LabelB"},
		LabelsExcludeAny:  []string{"LabelC"},
	}

	created, err := ds.CreateAndroidCACertificate(ctx, cert)
	require.NoError(t, err)
	require.NotZero(t, created.ID)
	assert.Equal(t, cert.Name, created.Name)
	assert.Equal(t, cert.TeamID, created.TeamID)
	assert.ElementsMatch(t, []string{"LabelA"}, created.LabelsIncludeAll)
	assert.ElementsMatch(t, []string{"LabelB"}, created.LabelsIncludeAny)
	assert.ElementsMatch(t, []string{"LabelC"}, created.LabelsExcludeAny)

	// Duplicate create should fail
	_, err = ds.CreateAndroidCACertificate(ctx, cert)
	require.Error(t, err)

	// Get by ID
	got, err := ds.GetAndroidCACertificateByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.Name, got.Name)
	assert.Equal(t, cert.CertificatePEM, got.CertificatePEM)
	assert.ElementsMatch(t, []string{"LabelA"}, got.LabelsIncludeAll)
	assert.ElementsMatch(t, []string{"LabelB"}, got.LabelsIncludeAny)
	assert.ElementsMatch(t, []string{"LabelC"}, got.LabelsExcludeAny)

	// Get by Name
	gotByName, err := ds.GetAndroidCACertificateByName(ctx, team.ID, "Corporate Root CA")
	require.NoError(t, err)
	assert.Equal(t, created.ID, gotByName.ID)

	// List
	list, meta, err := ds.ListAndroidCACertificates(ctx, team.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, uint(1), meta.TotalResults)
	assert.Equal(t, created.ID, list[0].ID)

	// Delete
	err = ds.DeleteAndroidCACertificate(ctx, created.ID)
	require.NoError(t, err)

	// Should not be found after delete
	_, err = ds.GetAndroidCACertificateByID(ctx, created.ID)
	require.Error(t, err)
}

func testAndroidCACertificatesLabelScoping(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_Team"})
	require.NoError(t, err)

	// Create labels
	lbl1, err := ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "_Label1"})
	require.NoError(t, err)
	lbl2, err := ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "_Label2"})
	require.NoError(t, err)
	lbl3, err := ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "_Label3"})
	require.NoError(t, err)

	// Create 2 Android hosts in the team
	h1 := test.NewHost(t, ds, "android-h1", "1.1.1.1", "key1", "uuid1", time.Now(), test.WithPlatform("android"), test.WithTeamID(team.ID))
	h2 := test.NewHost(t, ds, "android-h2", "1.1.1.2", "key2", "uuid2", time.Now(), test.WithPlatform("android"), test.WithTeamID(team.ID))

	// Assign h1 to lbl1 and lbl2; h2 to lbl2 and lbl3
	err = ds.RecordLabelQueryExecutions(ctx, h1, map[uint]*bool{lbl1.ID: ptr.Bool(true), lbl2.ID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	err = ds.RecordLabelQueryExecutions(ctx, h2, map[uint]*bool{lbl2.ID: ptr.Bool(true), lbl3.ID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)

	// CA1: no labels -> matches both h1 and h2
	ca1, err := ds.CreateAndroidCACertificate(ctx, &fleet.AndroidCACertificate{
		TeamID:            team.ID,
		Name:              "Global Team CA",
		CertificatePEM:    "pem1",
		SHA256Fingerprint: "hash1",
		Subject:           "CN=Global",
		Issuer:            "CN=Global",
	})
	require.NoError(t, err)

	// CA2: include_all [Label1, Label2] -> matches only h1
	ca2, err := ds.CreateAndroidCACertificate(ctx, &fleet.AndroidCACertificate{
		TeamID:            team.ID,
		Name:              "Include All CA",
		CertificatePEM:    "pem2",
		SHA256Fingerprint: "hash2",
		Subject:           "CN=IncludeAll",
		Issuer:            "CN=IncludeAll",
		LabelsIncludeAll:  []string{lbl1.Name, lbl2.Name},
	})
	require.NoError(t, err)

	// CA3: exclude_any [Label3] -> excludes h2, matches h1
	ca3, err := ds.CreateAndroidCACertificate(ctx, &fleet.AndroidCACertificate{
		TeamID:            team.ID,
		Name:              "Exclude Label3 CA",
		CertificatePEM:    "pem3",
		SHA256Fingerprint: "hash3",
		Subject:           "CN=Exclude3",
		Issuer:            "CN=Exclude3",
		LabelsExcludeAny:  []string{lbl3.Name},
	})
	require.NoError(t, err)

	// Check scoped CAs for h1
	h1Certs, err := ds.GetHostScopedAndroidCACertificates(ctx, h1.ID)
	require.NoError(t, err)
	h1IDs := make([]uint, len(h1Certs))
	for i, c := range h1Certs {
		h1IDs[i] = c.ID
	}
	assert.Contains(t, h1IDs, ca1.ID)
	assert.Contains(t, h1IDs, ca2.ID)
	assert.Contains(t, h1IDs, ca3.ID)

	// Check scoped CAs for h2
	h2Certs, err := ds.GetHostScopedAndroidCACertificates(ctx, h2.ID)
	require.NoError(t, err)
	h2IDs := make([]uint, len(h2Certs))
	for i, c := range h2Certs {
		h2IDs[i] = c.ID
	}
	assert.Contains(t, h2IDs, ca1.ID)
	assert.NotContains(t, h2IDs, ca2.ID)
	assert.NotContains(t, h2IDs, ca3.ID)
}

func testAndroidCAHostStatusTracking(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_Team"})
	require.NoError(t, err)

	h := test.NewHost(t, ds, "android-host-status", "1.1.1.1", "status-key", "status-uuid", time.Now(), test.WithPlatform("android"), test.WithTeamID(team.ID))

	ca, err := ds.CreateAndroidCACertificate(ctx, &fleet.AndroidCACertificate{
		TeamID:            team.ID,
		Name:              "Status Test CA",
		CertificatePEM:    "pem",
		SHA256Fingerprint: "status-hash",
		Subject:           "CN=Status CA",
		Issuer:            "CN=Status CA",
	})
	require.NoError(t, err)

	// Upsert status pending
	err = ds.UpsertHostAndroidCACertificateStatus(ctx, h.ID, h.UUID, ca.ID, fleet.MDMDeliveryPending, "")
	require.NoError(t, err)

	pendingUUIDs, err := ds.ListAndroidHostUUIDsWithPendingCACertificates(ctx, 0, 10)
	require.NoError(t, err)
	assert.Contains(t, pendingUUIDs, h.UUID)

	// Upsert status verified
	err = ds.UpsertHostAndroidCACertificateStatus(ctx, h.ID, h.UUID, ca.ID, fleet.MDMDeliveryVerified, "")
	require.NoError(t, err)

	// List host CA certs
	list, err := ds.ListHostAndroidCACertificates(ctx, h.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, ca.ID, list[0].CACertificateID)
	assert.Equal(t, ca.Name, list[0].CertificateName)
	assert.Equal(t, fleet.MDMDeliveryVerified, list[0].Status)

	// List by UUID
	listByUUID, err := ds.ListHostAndroidCACertificatesByUUID(ctx, h.UUID)
	require.NoError(t, err)
	require.Len(t, listByUUID, 1)
	assert.Equal(t, ca.ID, listByUUID[0].CACertificateID)

	// Update to failed with detail
	err = ds.UpsertHostAndroidCACertificateStatus(ctx, h.ID, h.UUID, ca.ID, fleet.MDMDeliveryFailed, "install failed: corrupt cert")
	require.NoError(t, err)

	list, err = ds.ListHostAndroidCACertificates(ctx, h.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, fleet.MDMDeliveryFailed, list[0].Status)
	assert.Equal(t, "install failed: corrupt cert", list[0].Detail)
}
