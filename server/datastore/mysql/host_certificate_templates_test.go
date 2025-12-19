package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestHostCertificateTemplates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ListAndroidHostUUIDsWithDeliverableCertificateTemplates", testListAndroidHostUUIDsWithDeliverableCertificateTemplates},
		{"ListCertificateTemplatesForHosts", testListCertificateTemplatesForHosts},
		{"GetCertificateTemplateForHostNoTeam", testGetCertificateTemplateForHostNoTeam},
		{"BulkInsertAndDeleteHostCertificateTemplates", testBulkInsertAndDeleteHostCertificateTemplates},
		{"DeleteHostCertificateTemplate", testDeleteHostCertificateTemplate},
		{"UpsertHostCertificateTemplateStatus", testUpsertHostCertificateTemplateStatus},
		{"CreatePendingCertificateTemplatesForExistingHosts", testCreatePendingCertificateTemplatesForExistingHosts},
		{"CreatePendingCertificateTemplatesForNewHost", testCreatePendingCertificateTemplatesForNewHost},
		{"ListAndroidHostUUIDsWithPendingCertificateTemplates", testListAndroidHostUUIDsWithPendingCertificateTemplates},
		{"CertificateTemplateFullStateMachine", testCertificateTemplateFullStateMachine},
		{"RevertStaleCertificateTemplates", testRevertStaleCertificateTemplates},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// certTemplateTestSetup contains the common test fixtures for certificate template tests.
type certTemplateTestSetup struct {
	team     *fleet.Team
	ca       *fleet.CertificateAuthority
	template *fleet.CertificateTemplateResponse
}

// createCertTemplateTestSetup creates a team, certificate authority, and certificate template for testing.
// If teamName is empty, a unique name is generated.
func createCertTemplateTestSetup(t *testing.T, ctx context.Context, ds *Datastore, teamName string) certTemplateTestSetup {
	if teamName == "" {
		teamName = fmt.Sprintf("Test Team %s", uuid.New().String()[:8])
	}

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: teamName})
	require.NoError(t, err)

	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String(fmt.Sprintf("Test SCEP CA %s", uuid.New().String()[:8])),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)

	template, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   fmt.Sprintf("Test Cert %s", uuid.New().String()[:8]),
		TeamID:                 team.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Test",
	})
	require.NoError(t, err)

	return certTemplateTestSetup{
		team:     team,
		ca:       ca,
		template: template,
	}
}

// noTeamCertTemplateTestSetup contains test fixtures for "no team" certificate template tests.
type noTeamCertTemplateTestSetup struct {
	ca       *fleet.CertificateAuthority
	template *fleet.CertificateTemplateResponse
}

// createNoTeamCertTemplateTestSetup creates a certificate authority and certificate template
// for "no team" (team_id = 0) for testing.
func createNoTeamCertTemplateTestSetup(t *testing.T, ctx context.Context, ds *Datastore) noTeamCertTemplateTestSetup {
	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String(fmt.Sprintf("Test SCEP CA NoTeam %s", uuid.New().String()[:8])),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)

	template, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   fmt.Sprintf("Test Cert NoTeam %s", uuid.New().String()[:8]),
		TeamID:                 0, // No team uses team_id = 0
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=TestNoTeam",
	})
	require.NoError(t, err)

	return noTeamCertTemplateTestSetup{
		ca:       ca,
		template: template,
	}
}

// createEnrolledAndroidHost creates an enrolled Android host in the given team.
func createEnrolledAndroidHost(t *testing.T, ctx context.Context, ds *Datastore, hostUUID string, teamID *uint) *fleet.Host {
	host, err := ds.NewHost(ctx, &fleet.Host{
		UUID:     hostUUID,
		TeamID:   teamID,
		Platform: "android",
	})
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "", false, "", "", false)
	require.NoError(t, err)
	return host
}

func testListAndroidHostUUIDsWithDeliverableCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("android host with no host certificate templates is returned", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")
		createEnrolledAndroidHost(t, ctx, ds, "test-host-uuid", &setup.team.ID)

		results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "test-host-uuid", results[0])
	})

	t.Run("android host with existing host certificate templates is not returned", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")
		createEnrolledAndroidHost(t, ctx, ds, "test-host-uuid", &setup.team.ID)

		// Insert host certificate template record (host already has this template)
		_, err := ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, name) VALUES (?, ?, ?, ?, ?)",
			"test-host-uuid", setup.template.ID, "challenge", fleet.MDMDeliveryPending, setup.template.Name,
		)
		require.NoError(t, err)

		results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
		require.NoError(t, err)
		require.Len(t, results, 0)
	})

	t.Run("non-android host is not returned", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create a macOS host instead of Android
		host := &fleet.Host{
			UUID:     "macos-host-uuid",
			TeamID:   &setup.team.ID,
			Platform: "darwin",
		}
		h, err := ds.NewHost(ctx, host)
		require.NoError(t, err)
		nanoEnroll(t, ds, h, false)

		results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
		require.NoError(t, err)
		require.Len(t, results, 0)
	})

	t.Run("unenrolled host is not returned", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create Android host without MDM enrollment
		_, err := ds.NewHost(ctx, &fleet.Host{
			UUID:     "unenrolled-host-uuid",
			TeamID:   &setup.team.ID,
			Platform: "android",
		})
		require.NoError(t, err)
		// Note: not calling SetOrUpdateMDMData, so host is not enrolled

		results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
		require.NoError(t, err)
		require.Len(t, results, 0)
	})

	t.Run("no team android host with no team certificate template is returned", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createNoTeamCertTemplateTestSetup(t, ctx, ds)

		// Create an enrolled Android host with no team (TeamID = nil)
		createEnrolledAndroidHost(t, ctx, ds, "no-team-host-uuid", nil)

		results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
		require.NoError(t, err)
		require.Len(t, results, 1, "no team host should be returned for no team certificate template")
		require.Equal(t, "no-team-host-uuid", results[0])

		// Verify the template exists with team_id = 0
		var templateTeamID uint
		err = ds.writer(ctx).GetContext(ctx, &templateTeamID,
			"SELECT team_id FROM certificate_templates WHERE id = ?", setup.template.ID)
		require.NoError(t, err)
		require.Equal(t, uint(0), templateTeamID, "certificate template should have team_id = 0")
	})
}

func testListCertificateTemplatesForHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("host with no existing host certificate templates", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create a second template for this team
		_, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert2",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test Subject 2",
		})
		require.NoError(t, err)

		_, err = ds.NewHost(ctx, &fleet.Host{
			UUID:     "test-host-uuid",
			TeamID:   &setup.team.ID,
			Platform: "android",
		})
		require.NoError(t, err)

		results, err := ds.ListCertificateTemplatesForHosts(ctx, []string{"test-host-uuid"})
		require.NoError(t, err)
		require.Len(t, results, 2)

		for _, res := range results {
			require.Equal(t, "test-host-uuid", res.HostUUID)
			require.NotEmpty(t, res.CertificateTemplateID)
			require.Nil(t, res.FleetChallenge)
			require.Nil(t, res.Status)
		}
	})

	t.Run("host with existing host certificate templates", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create a second template
		templateTwo, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert2",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test Subject 2",
		})
		require.NoError(t, err)

		_, err = ds.NewHost(ctx, &fleet.Host{
			UUID:     "test-host-uuid",
			TeamID:   &setup.team.ID,
			Platform: "android",
		})
		require.NoError(t, err)

		// Insert host certificate template record for first template only
		_, err = ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, name) VALUES (?, ?, ?, ?, ?)",
			"test-host-uuid", setup.template.ID, "challenge", fleet.CertificateTemplateDelivered, setup.template.Name,
		)
		require.NoError(t, err)

		results, err := ds.ListCertificateTemplatesForHosts(ctx, []string{"test-host-uuid"})
		require.NoError(t, err)
		require.Len(t, results, 2)

		for _, res := range results {
			require.Equal(t, "test-host-uuid", res.HostUUID)
			require.NotEmpty(t, res.CertificateTemplateID)
			if res.CertificateTemplateID == setup.template.ID {
				require.Equal(t, ptr.String("challenge"), res.FleetChallenge)
				require.Equal(t, &fleet.CertificateTemplateDelivered, res.Status)
			} else {
				require.Equal(t, templateTwo.ID, res.CertificateTemplateID)
				require.Nil(t, res.FleetChallenge)
				require.Nil(t, res.Status)
			}
		}
	})

	t.Run("no team host returns no team certificate templates", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createNoTeamCertTemplateTestSetup(t, ctx, ds)

		// Create a host with no team (TeamID = nil)
		_, err := ds.NewHost(ctx, &fleet.Host{
			UUID:     "no-team-host-uuid",
			TeamID:   nil,
			Platform: "android",
		})
		require.NoError(t, err)

		results, err := ds.ListCertificateTemplatesForHosts(ctx, []string{"no-team-host-uuid"})
		require.NoError(t, err)
		require.Len(t, results, 1, "no team host should get no team certificate templates")
		require.Equal(t, "no-team-host-uuid", results[0].HostUUID)
		require.Equal(t, setup.template.ID, results[0].CertificateTemplateID)
	})
}

func testGetCertificateTemplateForHostNoTeam(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("returns certificate template for host with team", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create a host in the team
		_, err := ds.NewHost(ctx, &fleet.Host{
			UUID:     "team-host-uuid",
			TeamID:   &setup.team.ID,
			Platform: "android",
		})
		require.NoError(t, err)

		result, err := ds.GetCertificateTemplateForHost(ctx, "team-host-uuid", setup.template.ID)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "team-host-uuid", result.HostUUID)
		require.Equal(t, setup.template.ID, result.CertificateTemplateID)
	})

	t.Run("returns certificate template for no team host", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createNoTeamCertTemplateTestSetup(t, ctx, ds)

		// Create a host with no team (TeamID = nil)
		_, err := ds.NewHost(ctx, &fleet.Host{
			UUID:     "no-team-host-uuid",
			TeamID:   nil,
			Platform: "android",
		})
		require.NoError(t, err)

		result, err := ds.GetCertificateTemplateForHost(ctx, "no-team-host-uuid", setup.template.ID)
		require.NoError(t, err, "should find certificate template for no team host")
		require.NotNil(t, result)
		require.Equal(t, "no-team-host-uuid", result.HostUUID)
		require.Equal(t, setup.template.ID, result.CertificateTemplateID)
	})

	t.Run("returns not found for non-existent host", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		_, err := ds.GetCertificateTemplateForHost(ctx, "non-existent-host", setup.template.ID)
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
	})
}

func testBulkInsertAndDeleteHostCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("bulk inserts and deletes specific records", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		templateTwo, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert2",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test Subject 2",
		})
		require.NoError(t, err)

		// Insert host certificate templates
		hostCerts := []fleet.HostCertificateTemplate{
			{HostUUID: "host-1", CertificateTemplateID: setup.template.ID, FleetChallenge: ptr.String("challenge-1"), Status: fleet.CertificateTemplateDelivered, OperationType: fleet.MDMOperationTypeInstall},
			{HostUUID: "host-1", CertificateTemplateID: templateTwo.ID, FleetChallenge: ptr.String("challenge-2"), Status: fleet.CertificateTemplateDelivered, OperationType: fleet.MDMOperationTypeInstall},
			{HostUUID: "host-2", CertificateTemplateID: setup.template.ID, FleetChallenge: ptr.String("challenge-3"), Status: fleet.CertificateTemplateVerified, OperationType: fleet.MDMOperationTypeInstall},
		}
		err = ds.BulkInsertHostCertificateTemplates(ctx, hostCerts)
		require.NoError(t, err)

		var count int
		err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates")
		require.NoError(t, err)
		require.Equal(t, 3, count)

		// Delete only host-1's first certificate
		toDelete := []fleet.HostCertificateTemplate{
			{HostUUID: "host-1", CertificateTemplateID: setup.template.ID},
		}
		err = ds.DeleteHostCertificateTemplates(ctx, toDelete)
		require.NoError(t, err)

		// Verify only 2 records remain
		err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates")
		require.NoError(t, err)
		require.Equal(t, 2, count)

		// Verify the correct records remain
		var remaining []struct {
			HostUUID              string `db:"host_uuid"`
			CertificateTemplateID uint   `db:"certificate_template_id"`
		}
		err = ds.writer(ctx).SelectContext(ctx, &remaining,
			"SELECT host_uuid, certificate_template_id FROM host_certificate_templates ORDER BY host_uuid, certificate_template_id")
		require.NoError(t, err)
		require.Len(t, remaining, 2)
		require.Equal(t, "host-1", remaining[0].HostUUID)
		require.Equal(t, templateTwo.ID, remaining[0].CertificateTemplateID)
		require.Equal(t, "host-2", remaining[1].HostUUID)
		require.Equal(t, setup.template.ID, remaining[1].CertificateTemplateID)
	})

	t.Run("deletes multiple records at once", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		templateTwo, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert2",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test Subject 2",
		})
		require.NoError(t, err)

		// Insert host certificate templates
		hostCerts := []fleet.HostCertificateTemplate{
			{HostUUID: "host-1", CertificateTemplateID: setup.template.ID, FleetChallenge: ptr.String("challenge-1"), Status: fleet.CertificateTemplateDelivered, OperationType: fleet.MDMOperationTypeInstall},
			{HostUUID: "host-1", CertificateTemplateID: templateTwo.ID, FleetChallenge: ptr.String("challenge-2"), Status: fleet.CertificateTemplateDelivered, OperationType: fleet.MDMOperationTypeInstall},
		}
		err = ds.BulkInsertHostCertificateTemplates(ctx, hostCerts)
		require.NoError(t, err)

		// Delete both records
		toDelete := []fleet.HostCertificateTemplate{
			{HostUUID: "host-1", CertificateTemplateID: setup.template.ID},
			{HostUUID: "host-1", CertificateTemplateID: templateTwo.ID},
		}
		err = ds.DeleteHostCertificateTemplates(ctx, toDelete)
		require.NoError(t, err)

		var count int
		err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates")
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("no error when deleting non-existent records", func(t *testing.T) {
		defer TruncateTables(t, ds)
		toDelete := []fleet.HostCertificateTemplate{
			{HostUUID: "non-existent-host", CertificateTemplateID: 999},
		}
		err := ds.DeleteHostCertificateTemplates(ctx, toDelete)
		require.NoError(t, err)
	})

	t.Run("no error with empty list", func(t *testing.T) {
		defer TruncateTables(t, ds)
		err := ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{})
		require.NoError(t, err)

		err = ds.DeleteHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{})
		require.NoError(t, err)
	})
}

func testDeleteHostCertificateTemplate(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("deletes single record successfully", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Insert a host certificate template record
		_, err := ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
			"host-1", setup.template.ID, "challenge", fleet.CertificateTemplateVerified, fleet.MDMOperationTypeRemove, setup.template.Name,
		)
		require.NoError(t, err)

		// Verify record exists
		var count int
		err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?", "host-1", setup.template.ID)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		// Delete the record
		err = ds.DeleteHostCertificateTemplate(ctx, "host-1", setup.template.ID)
		require.NoError(t, err)

		// Verify record was deleted
		err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?", "host-1", setup.template.ID)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("no error when deleting non-existent record", func(t *testing.T) {
		defer TruncateTables(t, ds)

		err := ds.DeleteHostCertificateTemplate(ctx, "non-existent-host", 999)
		require.NoError(t, err)
	})

	t.Run("only deletes specified record", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		templateTwo, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert2",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test Subject 2",
		})
		require.NoError(t, err)

		// Insert multiple host certificate template records
		hostCerts := []fleet.HostCertificateTemplate{
			{HostUUID: "host-1", CertificateTemplateID: setup.template.ID, FleetChallenge: ptr.String("challenge-1"), Status: fleet.CertificateTemplateVerified, OperationType: fleet.MDMOperationTypeRemove, Name: setup.template.Name},
			{HostUUID: "host-1", CertificateTemplateID: templateTwo.ID, FleetChallenge: ptr.String("challenge-2"), Status: fleet.CertificateTemplateVerified, OperationType: fleet.MDMOperationTypeInstall, Name: templateTwo.Name},
			{HostUUID: "host-2", CertificateTemplateID: setup.template.ID, FleetChallenge: ptr.String("challenge-3"), Status: fleet.CertificateTemplateVerified, OperationType: fleet.MDMOperationTypeRemove, Name: setup.template.Name},
		}
		err = ds.BulkInsertHostCertificateTemplates(ctx, hostCerts)
		require.NoError(t, err)

		// Delete only host-1's first certificate
		err = ds.DeleteHostCertificateTemplate(ctx, "host-1", setup.template.ID)
		require.NoError(t, err)

		// Verify only 2 records remain
		var count int
		err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates")
		require.NoError(t, err)
		require.Equal(t, 2, count)

		// Verify the correct records remain
		var remaining []struct {
			HostUUID              string `db:"host_uuid"`
			CertificateTemplateID uint   `db:"certificate_template_id"`
		}
		err = ds.writer(ctx).SelectContext(ctx, &remaining,
			"SELECT host_uuid, certificate_template_id FROM host_certificate_templates ORDER BY host_uuid, certificate_template_id")
		require.NoError(t, err)
		require.Len(t, remaining, 2)
		require.Equal(t, "host-1", remaining[0].HostUUID)
		require.Equal(t, templateTwo.ID, remaining[0].CertificateTemplateID)
		require.Equal(t, "host-2", remaining[1].HostUUID)
		require.Equal(t, setup.template.ID, remaining[1].CertificateTemplateID)
	})
}

func testUpsertHostCertificateTemplateStatus(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	setup := createCertTemplateTestSetup(t, ctx, ds, "")

	// Create a second template for testing insert
	templateTwo, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "Cert2",
		TeamID:                 setup.team.ID,
		CertificateAuthorityID: setup.ca.ID,
		SubjectName:            "CN=Test Subject 2",
	})
	require.NoError(t, err)

	// Create a third template for testing insert with operation type
	templateThree, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "Cert3",
		TeamID:                 setup.team.ID,
		CertificateAuthorityID: setup.ca.ID,
		SubjectName:            "CN=Test Subject 3",
	})
	require.NoError(t, err)

	hostUUID := uuid.New().String()
	_, err = ds.NewHost(ctx, &fleet.Host{
		UUID:     hostUUID,
		Platform: "android",
		TeamID:   &setup.team.ID,
	})
	require.NoError(t, err)

	// Create an initial record for the first template
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, fleet_challenge, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
			hostUUID, setup.template.ID, fleet.MDMDeliveryPending, "some_challenge_value", fleet.MDMOperationTypeInstall, setup.template.Name)
		return err
	})

	cases := []struct {
		name                  string
		templateID            uint
		newStatus             string
		expectedErrorMsg      string
		detail                *string
		operationType         fleet.MDMOperationType
		expectedOperationType string
	}{
		{
			name:                  "valid update with install operation type",
			templateID:            setup.template.ID,
			newStatus:             "verified",
			operationType:         fleet.MDMOperationTypeInstall,
			expectedOperationType: "install",
		},
		{
			name:                  "valid update with details",
			templateID:            setup.template.ID,
			newStatus:             "failed",
			detail:                ptr.String("some details"),
			operationType:         fleet.MDMOperationTypeInstall,
			expectedOperationType: "install",
		},
		{
			name:             "invalid status",
			templateID:       setup.template.ID,
			newStatus:        "invalid_status",
			operationType:    fleet.MDMOperationTypeInstall,
			expectedErrorMsg: "Invalid status 'invalid_status'",
		},
		{
			name:                  "creates new record with install operation type",
			templateID:            templateTwo.ID,
			newStatus:             "verified",
			detail:                ptr.String("some details"),
			operationType:         fleet.MDMOperationTypeInstall,
			expectedOperationType: "install",
		},
		{
			name:                  "update operation type to remove",
			templateID:            setup.template.ID,
			newStatus:             "pending",
			operationType:         fleet.MDMOperationTypeRemove,
			expectedOperationType: "remove",
		},
		{
			name:                  "creates new record with remove operation type",
			templateID:            templateThree.ID,
			newStatus:             "pending",
			operationType:         fleet.MDMOperationTypeRemove,
			expectedOperationType: "remove",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ds.UpsertCertificateStatus(ctx, hostUUID, tc.templateID, fleet.MDMDeliveryStatus(tc.newStatus), tc.detail, tc.operationType)
			if tc.expectedErrorMsg == "" {
				require.NoError(t, err)
				var result struct {
					Status        string `db:"status"`
					OperationType string `db:"operation_type"`
				}
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					return sqlx.GetContext(ctx, q, &result,
						"SELECT status, operation_type FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?",
						hostUUID, tc.templateID)
				})
				require.Equal(t, tc.newStatus, result.Status)
				require.Equal(t, tc.expectedOperationType, result.OperationType)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErrorMsg)
			}
		})
	}
}

func testCreatePendingCertificateTemplatesForExistingHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("creates pending records for all enrolled android hosts in team", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create 3 enrolled Android hosts
		hostUUIDs := []string{"android-host-0", "android-host-1", "android-host-2"}
		for _, hostUUID := range hostUUIDs {
			createEnrolledAndroidHost(t, ctx, ds, hostUUID, &setup.team.ID)
		}

		rowsAffected, err := ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, setup.template.ID, setup.team.ID)
		require.NoError(t, err)
		require.Equal(t, int64(3), rowsAffected)

		// Verify records were created with pending status
		records, err := ds.ListCertificateTemplatesForHosts(ctx, hostUUIDs)
		require.NoError(t, err)
		require.Len(t, records, 3)

		for _, r := range records {
			require.NotNil(t, r.Status)
			require.EqualValues(t, fleet.CertificateTemplatePending, *r.Status)
			require.Nil(t, r.FleetChallenge)
		}
	})

	t.Run("does not create records for non-android hosts", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create a macOS host
		_, err := ds.NewHost(ctx, &fleet.Host{
			UUID:     "macos-host",
			TeamID:   &setup.team.ID,
			Platform: "darwin",
		})
		require.NoError(t, err)

		rowsAffected, err := ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, setup.template.ID, setup.team.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), rowsAffected)
	})

	t.Run("does not create records for unenrolled hosts", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create an unenrolled Android host (enrolled=false)
		host, err := ds.NewHost(ctx, &fleet.Host{
			UUID:     "unenrolled-android",
			TeamID:   &setup.team.ID,
			Platform: "android",
		})
		require.NoError(t, err)
		_, err = ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO host_mdm (host_id, enrolled) VALUES (?, ?)",
			host.ID, false,
		)
		require.NoError(t, err)

		rowsAffected, err := ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, setup.template.ID, setup.team.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), rowsAffected)
	})
}

func testListAndroidHostUUIDsWithPendingCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("returns hosts with pending install certificates", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create enrolled Android hosts with pending records
		for _, hostUUID := range []string{"host-1", "host-2"} {
			createEnrolledAndroidHost(t, ctx, ds, hostUUID, &setup.team.ID)
		}
		_, err := ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, setup.template.ID, setup.team.ID)
		require.NoError(t, err)

		results, err := ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 0, 10)
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Contains(t, results, "host-1")
		require.Contains(t, results, "host-2")
	})

	t.Run("does not return hosts with non-pending status", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Insert records with various non-pending statuses
		_, err := ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name) VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)",
			"host-delivering", setup.template.ID, "delivering", "install", nil, setup.template.Name,
			"host-delivered", setup.template.ID, "delivered", "install", "challenge1", setup.template.Name,
			"host-verified", setup.template.ID, "verified", "install", "challenge2", setup.template.Name,
		)
		require.NoError(t, err)

		results, err := ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 0, 10)
		require.NoError(t, err)
		require.Len(t, results, 0)
	})

	t.Run("respects pagination", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Insert pending records for 5 hosts
		for i := range 5 {
			_, err := ds.writer(ctx).ExecContext(ctx,
				"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, name) VALUES (?, ?, ?, ?, ?)",
				fmt.Sprintf("host-%d", i), setup.template.ID, "pending", "install", setup.template.Name,
			)
			require.NoError(t, err)
		}

		// First page
		results, err := ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 0, 2)
		require.NoError(t, err)
		require.Len(t, results, 2)

		// Second page
		results, err = ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 2, 2)
		require.NoError(t, err)
		require.Len(t, results, 2)

		// Third page
		results, err = ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 4, 2)
		require.NoError(t, err)
		require.Len(t, results, 1)
	})
}

// testCertificateTemplateFullStateMachine tests the complete certificate template
// status lifecycle: pending -> delivering -> delivered, including revert scenarios.
func testCertificateTemplateFullStateMachine(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	setup := createCertTemplateTestSetup(t, ctx, ds, "")

	// Create a second template
	templateTwo, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "Test Cert 2",
		TeamID:                 setup.team.ID,
		CertificateAuthorityID: setup.ca.ID,
		SubjectName:            "CN=Test2",
	})
	require.NoError(t, err)

	// Create enrolled Android host
	createEnrolledAndroidHost(t, ctx, ds, "android-host", &setup.team.ID)

	// Step 1: Create pending records
	rowsAffected, err := ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, setup.template.ID, setup.team.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)

	rowsAffected, err = ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, templateTwo.ID, setup.team.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)

	// Step 2: List hosts with pending templates
	hostUUIDs, err := ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 0, 10)
	require.NoError(t, err)
	require.Len(t, hostUUIDs, 1)
	require.Equal(t, "android-host", hostUUIDs[0])

	// Step 3: Transition to delivering
	certTemplates, err := ds.GetAndTransitionCertificateTemplatesToDelivering(ctx, "android-host")
	require.NoError(t, err)
	require.Len(t, certTemplates.DeliveringTemplateIDs, 2)
	require.ElementsMatch(t, []uint{setup.template.ID, templateTwo.ID}, certTemplates.DeliveringTemplateIDs)
	require.Empty(t, certTemplates.OtherTemplateIDs) // No existing verified/delivered templates yet

	// Verify host is no longer in pending list
	hostUUIDs, err = ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 0, 10)
	require.NoError(t, err)
	require.Len(t, hostUUIDs, 0)

	// Second call should return the already-delivering templates (no new pending ones to transition)
	certTemplates, err = ds.GetAndTransitionCertificateTemplatesToDelivering(ctx, "android-host")
	require.NoError(t, err)
	require.Len(t, certTemplates.DeliveringTemplateIDs, 2) // Already delivering from previous call
	require.ElementsMatch(t, []uint{setup.template.ID, templateTwo.ID}, certTemplates.DeliveringTemplateIDs)
	require.Empty(t, certTemplates.OtherTemplateIDs) // No delivered/verified/failed yet

	// Verify database shows delivering status
	records, err := ds.ListCertificateTemplatesForHosts(ctx, []string{"android-host"})
	require.NoError(t, err)
	require.Len(t, records, 2)
	for _, r := range records {
		require.NotNil(t, r.Status)
		require.EqualValues(t, fleet.CertificateTemplateDelivering, *r.Status)
	}

	// Step 4: Transition to delivered with challenges
	challenges := map[uint]string{
		setup.template.ID: "challenge-abc",
		templateTwo.ID:    "challenge-xyz",
	}
	err = ds.TransitionCertificateTemplatesToDelivered(ctx, "android-host", challenges)
	require.NoError(t, err)

	// Verify final state
	records, err = ds.ListCertificateTemplatesForHosts(ctx, []string{"android-host"})
	require.NoError(t, err)
	require.Len(t, records, 2)
	for _, r := range records {
		require.NotNil(t, r.Status)
		require.EqualValues(t, fleet.CertificateTemplateDelivered, *r.Status)
		require.NotNil(t, r.FleetChallenge)
		if r.CertificateTemplateID == setup.template.ID {
			require.Equal(t, "challenge-abc", *r.FleetChallenge)
		} else {
			require.Equal(t, "challenge-xyz", *r.FleetChallenge)
		}
	}

	// Test revert scenario: Create new pending records, transition to delivering, then revert
	createEnrolledAndroidHost(t, ctx, ds, "revert-test-host", &setup.team.ID)
	_, err = ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, setup.template.ID, setup.team.ID)
	require.NoError(t, err)

	_, err = ds.GetAndTransitionCertificateTemplatesToDelivering(ctx, "revert-test-host")
	require.NoError(t, err)

	// Revert to pending
	err = ds.RevertHostCertificateTemplatesToPending(ctx, "revert-test-host", []uint{setup.template.ID})
	require.NoError(t, err)

	// Verify reverted state
	records, err = ds.ListCertificateTemplatesForHosts(ctx, []string{"revert-test-host"})
	require.NoError(t, err)
	require.Len(t, records, 2) // Both templates for the team
	for _, r := range records {
		if r.CertificateTemplateID == setup.template.ID {
			require.NotNil(t, r.Status)
			require.EqualValues(t, fleet.CertificateTemplatePending, *r.Status)
		}
	}
}

func testCreatePendingCertificateTemplatesForNewHost(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("creates pending records for newly enrolled host", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create a second template
		_, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert2",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test2",
		})
		require.NoError(t, err)

		// Create newly enrolled host
		hostUUID := "new-android-host"
		createEnrolledAndroidHost(t, ctx, ds, hostUUID, &setup.team.ID)

		// Create pending templates for this new host
		rowsAffected, err := ds.CreatePendingCertificateTemplatesForNewHost(ctx, hostUUID, setup.team.ID)
		require.NoError(t, err)
		require.Equal(t, int64(2), rowsAffected)

		// Verify the records were created
		records, err := ds.ListCertificateTemplatesForHosts(ctx, []string{hostUUID})
		require.NoError(t, err)
		require.Len(t, records, 2)

		for _, r := range records {
			require.Equal(t, hostUUID, r.HostUUID)
			require.NotNil(t, r.Status)
			require.EqualValues(t, fleet.CertificateTemplatePending, *r.Status)
		}
	})

	t.Run("no-op when team has no certificate templates", func(t *testing.T) {
		defer TruncateTables(t, ds)
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team No Templates"})
		require.NoError(t, err)

		hostUUID := "host-no-templates"

		rowsAffected, err := ds.CreatePendingCertificateTemplatesForNewHost(ctx, hostUUID, team.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), rowsAffected)
	})
}

func testRevertStaleCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("reverts stale delivering templates", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Insert a record in 'delivering' status with updated_at set to 7 hours ago
		_, err := ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, updated_at, name) VALUES (?, ?, ?, ?, DATE_SUB(NOW(), INTERVAL 7 HOUR), ?)",
			"stale-host", setup.template.ID, fleet.CertificateTemplateDelivering, fleet.MDMOperationTypeInstall, setup.template.Name,
		)
		require.NoError(t, err)

		// Insert a record in 'delivering' status that is recent (1 hour ago)
		_, err = ds.writer(ctx).ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, updated_at, name) VALUES (?, ?, ?, ?, DATE_SUB(NOW(), INTERVAL 1 HOUR), ?)",
			"recent-host", setup.template.ID, fleet.CertificateTemplateDelivering, fleet.MDMOperationTypeInstall, setup.template.Name,
		)
		require.NoError(t, err)

		// Revert with 6-hour threshold
		affected, err := ds.RevertStaleCertificateTemplates(ctx, 6*time.Hour)
		require.NoError(t, err)
		require.Equal(t, int64(1), affected)

		// Verify only the stale one was reverted
		var statuses []struct {
			HostUUID string                          `db:"host_uuid"`
			Status   fleet.CertificateTemplateStatus `db:"status"`
		}
		err = ds.writer(ctx).SelectContext(ctx, &statuses,
			"SELECT host_uuid, status FROM host_certificate_templates ORDER BY host_uuid")
		require.NoError(t, err)
		require.Len(t, statuses, 2)

		for _, s := range statuses {
			if s.HostUUID == "stale-host" {
				require.EqualValues(t, fleet.CertificateTemplatePending, s.Status)
			} else {
				require.EqualValues(t, fleet.CertificateTemplateDelivering, s.Status)
			}
		}
	})

	t.Run("does not revert non-delivering statuses", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Insert records with various non-delivering statuses, all old
		for _, status := range []fleet.CertificateTemplateStatus{
			fleet.CertificateTemplatePending,
			fleet.CertificateTemplateDelivered,
			fleet.CertificateTemplateVerified,
			fleet.CertificateTemplateFailed,
		} {
			_, err := ds.writer(ctx).ExecContext(ctx,
				"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, updated_at, name) VALUES (?, ?, ?, ?, DATE_SUB(NOW(), INTERVAL 7 HOUR), ?)",
				fmt.Sprintf("host-%s", status), setup.template.ID, status, fleet.MDMOperationTypeInstall, setup.template.Name,
			)
			require.NoError(t, err)
		}

		// Revert should not affect any of them
		affected, err := ds.RevertStaleCertificateTemplates(ctx, 6*time.Hour)
		require.NoError(t, err)
		require.Equal(t, int64(0), affected)
	})

	t.Run("returns zero when no stale templates", func(t *testing.T) {
		defer TruncateTables(t, ds)
		affected, err := ds.RevertStaleCertificateTemplates(ctx, 6*time.Hour)
		require.NoError(t, err)
		require.Equal(t, int64(0), affected)
	})
}
