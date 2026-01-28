package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"SetHostCertificateTemplatesToPendingRemove", testSetHostCertificateTemplatesToPendingRemove},
		{"SetHostCertificateTemplatesToPendingRemoveForHost", testSetHostCertificateTemplatesToPendingRemoveForHost},
		{"ListCertificateTemplatesForHostsIncludesRemovalAfterTeamTransfer", testListCertificateTemplatesForHostsIncludesRemovalAfterTeamTransfer},
		{"ListAndroidHostUUIDsWithPendingCertificateTemplatesIncludesRemoval", testListAndroidHostUUIDsWithPendingCertificateTemplatesIncludesRemoval},
		{"GetAndTransitionCertificateTemplatesToDeliveringIncludesRemoval", testGetAndTransitionCertificateTemplatesToDeliveringIncludesRemoval},
		{"CertificateTemplateReinstalledAfterTransferBackToOriginalTeam", testCertificateTemplateReinstalledAfterTransferBackToOriginalTeam},
		{"GetAndroidCertificateTemplatesForRenewal", testGetAndroidCertificateTemplatesForRenewal},
		{"SetAndroidCertificateTemplatesForRenewal", testSetAndroidCertificateTemplatesForRenewal},
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
			err := ds.UpsertCertificateStatus(ctx, &fleet.CertificateStatusUpdate{
				HostUUID:              hostUUID,
				CertificateTemplateID: tc.templateID,
				Status:                fleet.MDMDeliveryStatus(tc.newStatus),
				Detail:                tc.detail,
				OperationType:         tc.operationType,
			})
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
	require.Len(t, certTemplates.Templates, 2)

	// Verify host is no longer in pending list
	hostUUIDs, err = ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 0, 10)
	require.NoError(t, err)
	require.Len(t, hostUUIDs, 0)

	// Second call should return the already-delivering templates (no new pending ones to transition)
	certTemplates, err = ds.GetAndTransitionCertificateTemplatesToDelivering(ctx, "android-host")
	require.NoError(t, err)
	require.Len(t, certTemplates.DeliveringTemplateIDs, 2) // Already delivering from previous call
	require.ElementsMatch(t, []uint{setup.template.ID, templateTwo.ID}, certTemplates.DeliveringTemplateIDs)
	require.Len(t, certTemplates.Templates, 2)

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

func testSetHostCertificateTemplatesToPendingRemove(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("deletes pending and failed rows and updates others to pending remove", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Insert records with various statuses for the same template
		_, err := ds.writer(ctx).ExecContext(ctx, `
			INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name, uuid) VALUES
			(?, ?, ?, ?, ?, ?, UUID_TO_BIN(UUID(), true)),
			(?, ?, ?, ?, ?, ?, UUID_TO_BIN(UUID(), true)),
			(?, ?, ?, ?, ?, ?, UUID_TO_BIN(UUID(), true)),
			(?, ?, ?, ?, ?, ?, UUID_TO_BIN(UUID(), true))
		`,
			"host-pending", setup.template.ID, fleet.CertificateTemplatePending, fleet.MDMOperationTypeInstall, nil, setup.template.Name,
			"host-delivered", setup.template.ID, fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeInstall, "challenge1", setup.template.Name,
			"host-verified", setup.template.ID, fleet.CertificateTemplateVerified, fleet.MDMOperationTypeInstall, "challenge2", setup.template.Name,
			"host-failed", setup.template.ID, fleet.CertificateTemplateFailed, fleet.MDMOperationTypeInstall, "challenge3", setup.template.Name,
		)
		require.NoError(t, err)

		err = ds.SetHostCertificateTemplatesToPendingRemove(ctx, setup.template.ID)
		require.NoError(t, err)

		// Verify the pending row was deleted
		var count int
		err = ds.writer(ctx).GetContext(ctx, &count,
			"SELECT COUNT(*) FROM host_certificate_templates WHERE host_uuid = ?", "host-pending")
		require.NoError(t, err)
		require.Equal(t, 0, count)

		// Verify the failed row was also deleted
		err = ds.writer(ctx).GetContext(ctx, &count,
			"SELECT COUNT(*) FROM host_certificate_templates WHERE host_uuid = ?", "host-failed")
		require.NoError(t, err)
		require.Equal(t, 0, count)

		// Verify remaining rows (delivered, verified) have status=pending and operation_type=remove
		var remaining []struct {
			HostUUID      string                 `db:"host_uuid"`
			Status        string                 `db:"status"`
			OperationType fleet.MDMOperationType `db:"operation_type"`
			UUID          string                 `db:"uuid"`
		}
		err = ds.writer(ctx).SelectContext(ctx, &remaining,
			"SELECT host_uuid, status, operation_type, COALESCE(BIN_TO_UUID(uuid, true), '') AS uuid FROM host_certificate_templates ORDER BY host_uuid")
		require.NoError(t, err)
		require.Len(t, remaining, 2)

		for _, r := range remaining {
			require.Equal(t, string(fleet.CertificateTemplatePending), r.Status)
			require.Equal(t, fleet.MDMOperationTypeRemove, r.OperationType)
			require.NotEmpty(t, r.UUID, "UUID should be set after transition to remove")
		}

		// Capture UUIDs before second call
		uuidsBefore := make(map[string]string)
		for _, r := range remaining {
			uuidsBefore[r.HostUUID] = r.UUID
		}

		// Second call should be idempotent - UUIDs should not change
		err = ds.SetHostCertificateTemplatesToPendingRemove(ctx, setup.template.ID)
		require.NoError(t, err)

		err = ds.writer(ctx).SelectContext(ctx, &remaining,
			"SELECT host_uuid, status, operation_type, COALESCE(BIN_TO_UUID(uuid, true), '') AS uuid FROM host_certificate_templates ORDER BY host_uuid")
		require.NoError(t, err)
		require.Len(t, remaining, 2)

		for _, r := range remaining {
			require.Equal(t, uuidsBefore[r.HostUUID], r.UUID, "UUID should not change when already in remove state")
		}
	})

	t.Run("only affects rows for the specified template", func(t *testing.T) {
		defer TruncateTables(t, ds)
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create a second template
		templateTwo, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert2",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test2",
		})
		require.NoError(t, err)

		// Insert records for both templates
		_, err = ds.writer(ctx).ExecContext(ctx, `
			INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name) VALUES
			(?, ?, ?, ?, ?, ?),
			(?, ?, ?, ?, ?, ?)
		`,
			"host-1", setup.template.ID, fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeInstall, "challenge1", setup.template.Name,
			"host-1", templateTwo.ID, fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeInstall, "challenge2", templateTwo.Name,
		)
		require.NoError(t, err)

		// Call the method for template one only
		err = ds.SetHostCertificateTemplatesToPendingRemove(ctx, setup.template.ID)
		require.NoError(t, err)

		// Verify template one was updated
		var row struct {
			Status        string                 `db:"status"`
			OperationType fleet.MDMOperationType `db:"operation_type"`
		}
		err = ds.writer(ctx).GetContext(ctx, &row,
			"SELECT status, operation_type FROM host_certificate_templates WHERE certificate_template_id = ?",
			setup.template.ID)
		require.NoError(t, err)
		require.Equal(t, string(fleet.CertificateTemplatePending), row.Status)
		require.Equal(t, fleet.MDMOperationTypeRemove, row.OperationType)

		// Verify template two was NOT affected
		err = ds.writer(ctx).GetContext(ctx, &row,
			"SELECT status, operation_type FROM host_certificate_templates WHERE certificate_template_id = ?",
			templateTwo.ID)
		require.NoError(t, err)
		require.Equal(t, string(fleet.CertificateTemplateDelivered), row.Status)
		require.Equal(t, fleet.MDMOperationTypeInstall, row.OperationType)
	})

	t.Run("handles no matching rows gracefully", func(t *testing.T) {
		defer TruncateTables(t, ds)

		// Call with a non-existent template ID
		err := ds.SetHostCertificateTemplatesToPendingRemove(ctx, 99999)
		require.NoError(t, err)
	})
}

func testSetHostCertificateTemplatesToPendingRemoveForHost(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t.Run("deletes pending and failed installs, updates other installs, leaves removes unchanged", func(t *testing.T) {
		setup := createCertTemplateTestSetup(t, ctx, ds, "")

		// Create additional templates
		templateTwo, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert2",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test2",
		})
		require.NoError(t, err)

		templateThree, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert3",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test3",
		})
		require.NoError(t, err)

		templateFour, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
			Name:                   "Cert4",
			TeamID:                 setup.team.ID,
			CertificateAuthorityID: setup.ca.ID,
			SubjectName:            "CN=Test4",
		})
		require.NoError(t, err)

		// Insert records for host-1 (the target host) and host-2 (should not be affected)
		_, err = ds.writer(ctx).ExecContext(ctx, `
			INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name) VALUES
			(?, ?, ?, ?, ?, ?),
			(?, ?, ?, ?, ?, ?),
			(?, ?, ?, ?, ?, ?),
			(?, ?, ?, ?, ?, ?),
			(?, ?, ?, ?, ?, ?)
		`,
			"host-1", setup.template.ID, fleet.CertificateTemplatePending, fleet.MDMOperationTypeInstall, nil, setup.template.Name,
			"host-1", templateTwo.ID, fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeInstall, "challenge1", templateTwo.Name,
			"host-1", templateThree.ID, fleet.CertificateTemplateFailed, fleet.MDMOperationTypeInstall, nil, templateThree.Name,
			"host-1", templateFour.ID, fleet.CertificateTemplateDelivering, fleet.MDMOperationTypeRemove, nil, templateFour.Name,
			"host-2", setup.template.ID, fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeInstall, "challenge2", setup.template.Name,
		)
		require.NoError(t, err)

		// Call the method for host-1 only
		err = ds.SetHostCertificateTemplatesToPendingRemoveForHost(ctx, "host-1")
		require.NoError(t, err)

		// Verify host-1's pending install was deleted
		var count int
		err = ds.writer(ctx).GetContext(ctx, &count,
			"SELECT COUNT(*) FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?",
			"host-1", setup.template.ID)
		require.NoError(t, err)
		require.Equal(t, 0, count, "pending install should be deleted")

		// Verify host-1's failed install was also deleted
		err = ds.writer(ctx).GetContext(ctx, &count,
			"SELECT COUNT(*) FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?",
			"host-1", templateThree.ID)
		require.NoError(t, err)
		require.Equal(t, 0, count, "failed install should be deleted")

		// Verify host-1's delivered install was updated to pending remove
		var row struct {
			Status        string                 `db:"status"`
			OperationType fleet.MDMOperationType `db:"operation_type"`
		}
		err = ds.writer(ctx).GetContext(ctx, &row,
			"SELECT status, operation_type FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?",
			"host-1", templateTwo.ID)
		require.NoError(t, err)
		require.Equal(t, string(fleet.CertificateTemplatePending), row.Status)
		require.Equal(t, fleet.MDMOperationTypeRemove, row.OperationType)

		// Verify host-1's delivering remove was NOT changed (removal in progress)
		err = ds.writer(ctx).GetContext(ctx, &row,
			"SELECT status, operation_type FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?",
			"host-1", templateFour.ID)
		require.NoError(t, err)
		require.Equal(t, string(fleet.CertificateTemplateDelivering), row.Status, "remove operation should not change status")
		require.Equal(t, fleet.MDMOperationTypeRemove, row.OperationType, "remove operation should stay as remove")

		// Verify host-2 was NOT affected
		err = ds.writer(ctx).GetContext(ctx, &row,
			"SELECT status, operation_type FROM host_certificate_templates WHERE host_uuid = ?", "host-2")
		require.NoError(t, err)
		require.Equal(t, string(fleet.CertificateTemplateDelivered), row.Status)
		require.Equal(t, fleet.MDMOperationTypeInstall, row.OperationType)
	})
}

// testListCertificateTemplatesForHostsIncludesRemovalAfterTeamTransfer verifies that
// ListCertificateTemplatesForHosts returns removal entries for templates from a previous team
// after the host has transferred to a new team.
func testListCertificateTemplatesForHostsIncludesRemovalAfterTeamTransfer(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Setup: Create two teams with certificate templates using existing helpers
	setupA := createCertTemplateTestSetup(t, ctx, ds, "Team A for removal test")
	setupB := createCertTemplateTestSetup(t, ctx, ds, "Team B for removal test")

	// Create host initially in Team A
	host := createEnrolledAndroidHost(t, ctx, ds, uuid.New().String(), &setupA.team.ID)

	// Insert verified certificate from Team A using BulkInsertHostCertificateTemplates
	challenge := "challenge-a"
	err := ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{{
		HostUUID:              host.UUID,
		CertificateTemplateID: setupA.template.ID,
		Status:                fleet.CertificateTemplateVerified,
		OperationType:         fleet.MDMOperationTypeInstall,
		FleetChallenge:        &challenge,
		Name:                  setupA.template.Name,
	}})
	require.NoError(t, err)

	// Simulate team transfer: move host to Team B using UpdateHost
	host.TeamID = &setupB.team.ID
	err = ds.UpdateHost(ctx, host)
	require.NoError(t, err)

	// Mark Team A template for removal using the datastore method
	err = ds.SetHostCertificateTemplatesToPendingRemoveForHost(ctx, host.UUID)
	require.NoError(t, err)

	// Insert pending install for Team B template
	err = ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{{
		HostUUID:              host.UUID,
		CertificateTemplateID: setupB.template.ID,
		Status:                fleet.CertificateTemplatePending,
		OperationType:         fleet.MDMOperationTypeInstall,
		Name:                  setupB.template.Name,
	}})
	require.NoError(t, err)

	// Act: List certificate templates for host
	results, err := ds.ListCertificateTemplatesForHosts(ctx, []string{host.UUID})
	require.NoError(t, err)

	require.Len(t, results, 2, "should include both install and removal templates")

	templatesByID := make(map[uint]fleet.CertificateTemplateForHost)
	for _, r := range results {
		templatesByID[r.CertificateTemplateID] = r
	}

	// Team A template should be marked for removal
	require.Contains(t, templatesByID, setupA.template.ID, "should include Team A template marked for removal")
	require.NotNil(t, templatesByID[setupA.template.ID].OperationType)
	require.Equal(t, fleet.MDMOperationTypeRemove, *templatesByID[setupA.template.ID].OperationType)
	require.NotNil(t, templatesByID[setupA.template.ID].Status)
	require.Equal(t, fleet.CertificateTemplatePending, *templatesByID[setupA.template.ID].Status)

	// Team B template should be pending install
	require.Contains(t, templatesByID, setupB.template.ID, "should include Team B template")
	require.NotNil(t, templatesByID[setupB.template.ID].OperationType)
	require.Equal(t, fleet.MDMOperationTypeInstall, *templatesByID[setupB.template.ID].OperationType)
}

// testListAndroidHostUUIDsWithPendingCertificateTemplatesIncludesRemoval verifies that
// ListAndroidHostUUIDsWithPendingCertificateTemplates returns hosts with pending removal templates.
func testListAndroidHostUUIDsWithPendingCertificateTemplatesIncludesRemoval(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	setup := createCertTemplateTestSetup(t, ctx, ds, "")
	host := createEnrolledAndroidHost(t, ctx, ds, uuid.New().String(), &setup.team.ID)

	// Insert a pending removal template using BulkInsertHostCertificateTemplates
	err := ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{{
		HostUUID:              host.UUID,
		CertificateTemplateID: setup.template.ID,
		Status:                fleet.CertificateTemplatePending,
		OperationType:         fleet.MDMOperationTypeRemove,
		Name:                  setup.template.Name,
	}})
	require.NoError(t, err)

	// Act: List hosts with pending templates
	results, err := ds.ListAndroidHostUUIDsWithPendingCertificateTemplates(ctx, 0, 100)
	require.NoError(t, err)

	require.Len(t, results, 1, "should include host with pending removal")
	require.Equal(t, host.UUID, results[0])
}

// testGetAndTransitionCertificateTemplatesToDeliveringIncludesRemoval verifies that
// GetAndTransitionCertificateTemplatesToDelivering handles both install and remove operations.
func testGetAndTransitionCertificateTemplatesToDeliveringIncludesRemoval(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	setup := createCertTemplateTestSetup(t, ctx, ds, "")
	host := createEnrolledAndroidHost(t, ctx, ds, uuid.New().String(), &setup.team.ID)

	// Create second template for removal
	templateForRemoval, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "Template for Removal",
		TeamID:                 setup.team.ID,
		CertificateAuthorityID: setup.ca.ID,
		SubjectName:            "CN=Template for Removal",
	})
	require.NoError(t, err)

	// Insert both pending install and pending removal using BulkInsertHostCertificateTemplates
	err = ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{
		{
			HostUUID:              host.UUID,
			CertificateTemplateID: setup.template.ID,
			Status:                fleet.CertificateTemplatePending,
			OperationType:         fleet.MDMOperationTypeInstall,
			Name:                  setup.template.Name,
		},
		{
			HostUUID:              host.UUID,
			CertificateTemplateID: templateForRemoval.ID,
			Status:                fleet.CertificateTemplatePending,
			OperationType:         fleet.MDMOperationTypeRemove,
			Name:                  templateForRemoval.Name,
		},
	})
	require.NoError(t, err)

	// Act: Transition to delivering
	result, err := ds.GetAndTransitionCertificateTemplatesToDelivering(ctx, host.UUID)
	require.NoError(t, err)

	// Assert: Should include BOTH install and removal templates
	// Currently only install is included
	require.Len(t, result.Templates, 2, "should include both install and removal")
	require.Len(t, result.DeliveringTemplateIDs, 2, "should transition both to delivering")

	// Verify both templates are in delivering state in the database
	var statuses []struct {
		CertificateTemplateID uint   `db:"certificate_template_id"`
		Status                string `db:"status"`
		OperationType         string `db:"operation_type"`
	}
	err = ds.writer(ctx).SelectContext(ctx, &statuses,
		`SELECT certificate_template_id, status, operation_type
		 FROM host_certificate_templates WHERE host_uuid = ?`, host.UUID)
	require.NoError(t, err)
	require.Len(t, statuses, 2)

	for _, s := range statuses {
		require.Equal(t, string(fleet.CertificateTemplateDelivering), s.Status,
			"template %d should be in delivering status", s.CertificateTemplateID)
	}

	// Verify the result contains templates with correct operation types
	hasInstall := false
	hasRemove := false
	for _, tmpl := range result.Templates {
		if tmpl.OperationType == fleet.MDMOperationTypeInstall {
			hasInstall = true
		}
		if tmpl.OperationType == fleet.MDMOperationTypeRemove {
			hasRemove = true
		}
	}
	require.True(t, hasInstall, "should include install operation")
	require.True(t, hasRemove, "should include remove operation")
}

// testCertificateTemplateReinstalledAfterTransferBackToOriginalTeam verifies that when a host
// transfers back to its original team, the certificate template that was marked for removal
// is correctly transitioned back to pending install.
func testCertificateTemplateReinstalledAfterTransferBackToOriginalTeam(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	setupA := createCertTemplateTestSetup(t, ctx, ds, "Team A")
	setupB := createCertTemplateTestSetup(t, ctx, ds, "Team B")
	host := createEnrolledAndroidHost(t, ctx, ds, uuid.New().String(), &setupA.team.ID)

	// Host starts with verified cert in Team A
	challenge := "challenge-a"
	err := ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{{
		HostUUID:              host.UUID,
		CertificateTemplateID: setupA.template.ID,
		Status:                fleet.CertificateTemplateVerified,
		OperationType:         fleet.MDMOperationTypeInstall,
		FleetChallenge:        &challenge,
		Name:                  setupA.template.Name,
	}})
	require.NoError(t, err)

	// Capture initial UUID
	initialResults, err := ds.ListCertificateTemplatesForHosts(ctx, []string{host.UUID})
	require.NoError(t, err)
	var initialUUID string
	for _, r := range initialResults {
		if r.CertificateTemplateID == setupA.template.ID {
			require.NotNil(t, r.UUID, "initial UUID should be set")
			initialUUID = *r.UUID
			break
		}
	}
	require.NotEmpty(t, initialUUID, "initial UUID should not be empty")

	// Transfer to Team B: mark Team A cert for removal, create pending install for Team B
	host.TeamID = &setupB.team.ID
	require.NoError(t, ds.UpdateHost(ctx, host))
	require.NoError(t, ds.SetHostCertificateTemplatesToPendingRemoveForHost(ctx, host.UUID))
	_, err = ds.CreatePendingCertificateTemplatesForNewHost(ctx, host.UUID, setupB.team.ID)
	require.NoError(t, err)

	// Capture UUID after marking for removal (should be different from initial)
	removeResults, err := ds.ListCertificateTemplatesForHosts(ctx, []string{host.UUID})
	require.NoError(t, err)
	var uuidAfterRemove string
	for _, r := range removeResults {
		if r.CertificateTemplateID == setupA.template.ID {
			require.NotNil(t, r.UUID, "UUID after remove should be set")
			uuidAfterRemove = *r.UUID
			break
		}
	}
	require.NotEqual(t, initialUUID, uuidAfterRemove, "UUID should change when marked for removal")

	// Transfer back to Team A: mark Team B cert for removal, re-create pending install for Team A
	host.TeamID = &setupA.team.ID
	require.NoError(t, ds.UpdateHost(ctx, host))
	require.NoError(t, ds.SetHostCertificateTemplatesToPendingRemoveForHost(ctx, host.UUID))
	_, err = ds.CreatePendingCertificateTemplatesForNewHost(ctx, host.UUID, setupA.team.ID)
	require.NoError(t, err)

	// Team A's cert should now be pending install (not pending remove) with a new UUID
	results, err := ds.ListCertificateTemplatesForHosts(ctx, []string{host.UUID})
	require.NoError(t, err)

	var certA *fleet.CertificateTemplateForHost
	for _, r := range results {
		if r.CertificateTemplateID == setupA.template.ID {
			certA = &r
			break
		}
	}
	require.NotNil(t, certA, "Team A cert should exist")
	require.Equal(t, fleet.MDMOperationTypeInstall, *certA.OperationType)
	require.Equal(t, fleet.CertificateTemplatePending, *certA.Status)
	require.NotNil(t, certA.UUID, "UUID should be set")
	require.NotEqual(t, uuidAfterRemove, *certA.UUID, "UUID should change when reinstalled")
}

func testGetAndroidCertificateTemplatesForRenewal(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create test data
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test team"})
	require.NoError(t, err)

	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Name: ptr.String("test ca"),
		Type: string(fleet.CAConfigCustomSCEPProxy),
		URL:  ptr.String("http://localhost:8080/scep"),
	})
	require.NoError(t, err)

	template, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		TeamID:                 team.ID,
		Name:                   "test template",
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=test",
	})
	require.NoError(t, err)

	now := time.Now().UTC()

	// Table-driven test cases for renewal logic:
	// - Validity > 30 days: renew 30 days before expiration
	// - Validity > 2 days and <= 30 days: renew at half validity period before expiration
	// - Validity <= 2 days: does NOT auto-renew
	testCases := []struct {
		name          string
		validityDays  int // total validity period in days
		daysToExpiry  int // days until expiration
		status        fleet.CertificateTemplateStatus
		operationType fleet.MDMOperationType
		shouldRenew   bool
		description   string
	}{
		// Validity > 30 days cases
		{
			name:          "validity_365d_expires_7d",
			validityDays:  365,
			daysToExpiry:  7,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   true,
			description:   "1 year cert expiring in 7 days should renew (< 30 day threshold)",
		},
		{
			name:          "validity_365d_expires_29d",
			validityDays:  365,
			daysToExpiry:  29,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   true,
			description:   "1 year cert expiring in 29 days should renew (< 30 day threshold)",
		},
		{
			name:          "validity_365d_expires_30d",
			validityDays:  365,
			daysToExpiry:  30,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "1 year cert expiring in exactly 30 days should NOT renew (threshold is < 30)",
		},
		{
			name:          "validity_365d_expires_60d",
			validityDays:  365,
			daysToExpiry:  60,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "1 year cert expiring in 60 days should NOT renew",
		},
		{
			name:          "validity_31d_expires_29d",
			validityDays:  31,
			daysToExpiry:  29,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   true,
			description:   "31 day cert expiring in 29 days should renew (validity > 30, uses 30 day threshold)",
		},

		// Validity > 2 days and <= 30 days cases (uses half validity threshold)
		{
			name:          "validity_30d_expires_14d",
			validityDays:  30,
			daysToExpiry:  14,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   true,
			description:   "30 day cert expiring in 14 days should renew (< 15 day threshold = half of 30)",
		},
		{
			name:          "validity_30d_expires_16d",
			validityDays:  30,
			daysToExpiry:  16,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "30 day cert expiring in 16 days should NOT renew (> 15 day threshold)",
		},
		{
			name:          "validity_20d_expires_9d",
			validityDays:  20,
			daysToExpiry:  9,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   true,
			description:   "20 day cert expiring in 9 days should renew (< 10 day threshold = half of 20)",
		},
		{
			name:          "validity_20d_expires_11d",
			validityDays:  20,
			daysToExpiry:  11,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "20 day cert expiring in 11 days should NOT renew (> 10 day threshold)",
		},
		{
			name:          "validity_14d_expires_5d",
			validityDays:  14,
			daysToExpiry:  5,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   true,
			description:   "14 day cert expiring in 5 days should renew (< 7 day threshold = half of 14)",
		},
		{
			name:          "validity_3d_expires_1d",
			validityDays:  3,
			daysToExpiry:  1,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   true,
			description:   "3 day cert expiring in 1 day should renew (validity > 2, < 1.5 day threshold)",
		},

		// Validity <= 2 days cases (should NOT auto-renew)
		{
			name:          "validity_2d_expires_1d",
			validityDays:  2,
			daysToExpiry:  1,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "2 day cert should NOT auto-renew (validity <= 2 days)",
		},
		{
			name:          "validity_1d_expires_0d",
			validityDays:  1,
			daysToExpiry:  0,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "1 day cert should NOT auto-renew (validity <= 2 days)",
		},

		// Wrong status cases
		{
			name:          "pending_status",
			validityDays:  365,
			daysToExpiry:  7,
			status:        fleet.CertificateTemplatePending,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "pending status should NOT renew",
		},
		{
			name:          "delivered_status",
			validityDays:  365,
			daysToExpiry:  7,
			status:        fleet.CertificateTemplateDelivered,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "delivered status should NOT renew (only verified)",
		},
		{
			name:          "failed_status",
			validityDays:  365,
			daysToExpiry:  7,
			status:        fleet.CertificateTemplateFailed,
			operationType: fleet.MDMOperationTypeInstall,
			shouldRenew:   false,
			description:   "failed status should NOT renew",
		},

		// Wrong operation type
		{
			name:          "remove_operation",
			validityDays:  365,
			daysToExpiry:  7,
			status:        fleet.CertificateTemplateVerified,
			operationType: fleet.MDMOperationTypeRemove,
			shouldRenew:   false,
			description:   "remove operation should NOT renew",
		},
	}

	// Create hosts and insert certificate records for each test case
	expectedRenewals := make(map[string]bool)
	for i, tc := range testCases {
		hostUUID := fmt.Sprintf("host-%s-%d", tc.name, i)
		_ = test.NewHost(t, ds, fmt.Sprintf("host%d", i), fmt.Sprintf("192.168.1.%d", i+1),
			fmt.Sprintf("host%d_key", i), hostUUID, now,
			test.WithPlatform("android"), test.WithTeamID(team.ID))

		notValidBefore := now.Add(-time.Duration(tc.validityDays-tc.daysToExpiry) * 24 * time.Hour)
		notValidAfter := now.Add(time.Duration(tc.daysToExpiry) * 24 * time.Hour)
		insertHostCertTemplate(t, ds, hostUUID, template.ID, tc.status, tc.operationType, &notValidBefore, &notValidAfter)

		if tc.shouldRenew {
			expectedRenewals[hostUUID] = true
		}
	}

	// Execute the renewal query
	results, err := ds.GetAndroidCertificateTemplatesForRenewal(ctx, 100)
	require.NoError(t, err)

	// Build map of actual renewals
	actualRenewals := make(map[string]bool)
	for _, r := range results {
		actualRenewals[r.HostUUID] = true
		require.Equal(t, template.ID, r.CertificateTemplateID)
	}

	// Verify each test case
	for i, tc := range testCases {
		hostUUID := fmt.Sprintf("host-%s-%d", tc.name, i)
		if tc.shouldRenew {
			require.True(t, actualRenewals[hostUUID], "Test case %q: %s", tc.name, tc.description)
		} else {
			require.False(t, actualRenewals[hostUUID], "Test case %q: %s", tc.name, tc.description)
		}
	}

	// Verify total count matches expected
	require.Len(t, results, len(expectedRenewals), "Total renewal count should match expected")

	// Test limit functionality
	if len(expectedRenewals) > 1 {
		results, err = ds.GetAndroidCertificateTemplatesForRenewal(ctx, 1)
		require.NoError(t, err)
		require.Len(t, results, 1, "Limit should be respected")
	}

	// Verify results are ordered by not_valid_after ASC (most urgent first)
	results, err = ds.GetAndroidCertificateTemplatesForRenewal(ctx, 100)
	require.NoError(t, err)
	for i := 1; i < len(results); i++ {
		require.True(t, !results[i].NotValidAfter.Before(results[i-1].NotValidAfter),
			"Results should be ordered by expiration date ascending")
	}
}

// insertHostCertTemplate is a helper to insert a host_certificate_templates record with validity data
func insertHostCertTemplate(t *testing.T, ds *Datastore, hostUUID string, templateID uint, status fleet.CertificateTemplateStatus, opType fleet.MDMOperationType, notValidBefore, notValidAfter *time.Time) {
	t.Helper()
	_, err := ds.writer(t.Context()).ExecContext(
		t.Context(),
		`INSERT INTO host_certificate_templates
			(host_uuid, certificate_template_id, status, operation_type, name, uuid, not_valid_before, not_valid_after)
		VALUES (?, ?, ?, ?, 'test', UUID_TO_BIN(UUID(), true), ?, ?)`,
		hostUUID, templateID, status, opType, notValidBefore, notValidAfter,
	)
	require.NoError(t, err)
}

func testSetAndroidCertificateTemplatesForRenewal(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create test team, CA, and certificate template
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test team renewal set"})
	require.NoError(t, err)

	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Name: ptr.String("test ca renewal set"),
		Type: string(fleet.CAConfigCustomSCEPProxy),
		URL:  ptr.String("http://localhost:8080/scep"),
	})
	require.NoError(t, err)

	template, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		TeamID:                 team.ID,
		Name:                   "test template set",
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=test",
	})
	require.NoError(t, err)
	templateID := template.ID

	// Create test hosts
	now := time.Now().UTC()
	host1 := test.NewHost(t, ds, "host1-set", "192.168.1.1", "host1_key_set", uuid.NewString(), now, test.WithPlatform("android"), test.WithTeamID(team.ID))
	host2 := test.NewHost(t, ds, "host2-set", "192.168.1.2", "host2_key_set", uuid.NewString(), now, test.WithPlatform("android"), test.WithTeamID(team.ID))
	notValidBefore := now.AddDate(-1, 0, 7)
	notValidAfter := now.Add(7 * 24 * time.Hour)

	// Insert certificate records
	insertHostCertTemplate(t, ds, host1.UUID, templateID, fleet.CertificateTemplateVerified, fleet.MDMOperationTypeInstall, &notValidBefore, &notValidAfter)
	insertHostCertTemplate(t, ds, host2.UUID, templateID, fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeInstall, &notValidBefore, &notValidAfter)

	// Get the original UUIDs
	var originalUUIDs []struct {
		HostUUID string `db:"host_uuid"`
		UUID     string `db:"uuid"`
	}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &originalUUIDs,
		`SELECT host_uuid, COALESCE(BIN_TO_UUID(uuid, true), '') AS uuid FROM host_certificate_templates WHERE host_uuid IN (?, ?) ORDER BY host_uuid`,
		host1.UUID, host2.UUID)
	require.NoError(t, err)
	require.Len(t, originalUUIDs, 2)

	originalUUID1 := originalUUIDs[0].UUID
	originalUUID2 := originalUUIDs[1].UUID

	// Set templates for renewal
	templates := []fleet.HostCertificateTemplateForRenewal{
		{HostUUID: host1.UUID, CertificateTemplateID: templateID, NotValidAfter: notValidAfter},
		{HostUUID: host2.UUID, CertificateTemplateID: templateID, NotValidAfter: notValidAfter},
	}
	err = ds.SetAndroidCertificateTemplatesForRenewal(ctx, templates)
	require.NoError(t, err)

	// Verify the records were updated
	var updatedRecords []struct {
		HostUUID       string  `db:"host_uuid"`
		Status         string  `db:"status"`
		UUID           string  `db:"uuid"`
		NotValidBefore *string `db:"not_valid_before"`
		NotValidAfter  *string `db:"not_valid_after"`
		Serial         *string `db:"serial"`
	}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &updatedRecords,
		`SELECT host_uuid, status, COALESCE(BIN_TO_UUID(uuid, true), '') AS uuid, not_valid_before, not_valid_after, serial
		 FROM host_certificate_templates WHERE host_uuid IN (?, ?) ORDER BY host_uuid`,
		host1.UUID, host2.UUID)
	require.NoError(t, err)
	require.Len(t, updatedRecords, 2)

	for _, r := range updatedRecords {
		// Status should be pending
		require.Equal(t, string(fleet.CertificateTemplatePending), r.Status, "Status should be updated to pending")

		// UUID should be different (new one generated)
		if r.HostUUID == host1.UUID {
			require.NotEqual(t, originalUUID1, r.UUID, "UUID should be regenerated for host1")
		} else {
			require.NotEqual(t, originalUUID2, r.UUID, "UUID should be regenerated for host2")
		}

		// Validity fields should be cleared
		require.Nil(t, r.NotValidBefore, "not_valid_before should be cleared")
		require.Nil(t, r.NotValidAfter, "not_valid_after should be cleared")
		require.Nil(t, r.Serial, "serial should be cleared")
	}

	// Test empty slice doesn't error
	err = ds.SetAndroidCertificateTemplatesForRenewal(ctx, []fleet.HostCertificateTemplateForRenewal{})
	require.NoError(t, err)
}
