package mysql

import (
	"context"
	"fmt"
	"testing"

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
		{"BulkInsertAndDeleteHostCertificateTemplates", testBulkInsertAndDeleteHostCertificateTemplates},
		{"UpsertHostCertificateTemplateStatus", testUpsertHostCertificateTemplateStatus},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testListAndroidHostUUIDsWithDeliverableCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"android host with no host certificate templates",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID := team.ID

				// Create a test host
				host := &fleet.Host{
					UUID:     "test-host-uuid",
					TeamID:   &teamID,
					Platform: "android",
				}
				_, err = ds.NewHost(ctx, host)
				require.NoError(t, err)
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO host_mdm (host_id, enrolled) VALUES (?, ?)",
					host.ID,
					true,
				)
				require.NoError(t, err)

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID := ca.ID
				// Insert initial certificates
				certificateTemplate := fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificateTemplate.Name,
					certificateTemplate.TeamID,
					certificateTemplate.CertificateAuthorityID,
					certificateTemplate.SubjectName,
				)
				require.NoError(t, err)
			}, func(t *testing.T, ds *Datastore) {
				results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
				require.NoError(t, err)
				require.Len(t, results, 1)
				require.Equal(t, "test-host-uuid", results[0])
			},
		},
		{
			"android host with existing host certificate templates",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID := team.ID

				// Create a test host
				host := &fleet.Host{
					UUID:     "test-host-uuid",
					TeamID:   &teamID,
					Platform: "android",
				}
				_, err = ds.NewHost(ctx, host)
				require.NoError(t, err)
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO host_mdm (host_id, enrolled) VALUES (?, ?)",
					host.ID,
					true,
				)
				require.NoError(t, err)

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID := ca.ID

				// Insert initial certificate
				certificateTemplate := fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				res, err := ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificateTemplate.Name,
					certificateTemplate.TeamID,
					certificateTemplate.CertificateAuthorityID,
					certificateTemplate.SubjectName,
				)
				require.NoError(t, err)
				lastID, err := res.LastInsertId()
				require.NoError(t, err)

				// Insert host certificate template record
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status) VALUES (?, ?, ?, ?)",
					host.UUID,
					uint(lastID), //nolint:gosec
					"challenge",
					fleet.MDMDeliveryPending,
				)
				require.NoError(t, err)
			},
			func(t *testing.T, ds *Datastore) {
				results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
				require.NoError(t, err)
				require.Len(t, results, 0)
			},
		},
		{
			"host not on android platform",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID := team.ID

				// Create a test host
				host := &fleet.Host{
					UUID:     "test-host-uuid",
					TeamID:   &teamID,
					Platform: "macOS",
				}
				_, err = ds.NewHost(ctx, host)
				require.NoError(t, err)
				nanoEnroll(t, ds, host, false)

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID := ca.ID
				// Insert initial certificates
				certificateTemplate := fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificateTemplate.Name,
					certificateTemplate.TeamID,
					certificateTemplate.CertificateAuthorityID,
					certificateTemplate.SubjectName,
				)
				require.NoError(t, err)
			},
			func(t *testing.T, ds *Datastore) {
				results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
				require.NoError(t, err)
				require.Len(t, results, 0)
			},
		},
		{
			"host not enrolled in MDM",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID := team.ID

				// Create a test host
				host := &fleet.Host{
					UUID:     "test-host-uuid",
					TeamID:   &teamID,
					Platform: "android",
				}
				_, err = ds.NewHost(ctx, host)
				require.NoError(t, err)

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID := ca.ID
				// Insert initial certificates
				certificateTemplate := fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificateTemplate.Name,
					certificateTemplate.TeamID,
					certificateTemplate.CertificateAuthorityID,
					certificateTemplate.SubjectName,
				)
				require.NoError(t, err)
			},
			func(t *testing.T, ds *Datastore) {
				results, err := ds.ListAndroidHostUUIDsWithDeliverableCertificateTemplates(ctx, 0, 10)
				require.NoError(t, err)
				require.Len(t, results, 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			tc.before(ds)

			tc.testFunc(t, ds)
		})
	}
}

func testListCertificateTemplatesForHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	var templateWithHostRecordId uint
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"Host with no existing host certificate templates",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID := team.ID

				// Create a test host
				host := &fleet.Host{
					UUID:     "test-host-uuid",
					TeamID:   &teamID,
					Platform: "android",
				}
				_, err = ds.NewHost(context.Background(), host)
				require.NoError(t, err)

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID := ca.ID

				// Create certificate templates for the team
				for i := 0; i < 2; i++ {
					certificateTemplate := fleet.CertificateTemplate{
						Name:                   fmt.Sprintf("Cert%d", i),
						TeamID:                 teamID,
						CertificateAuthorityID: caID,
						SubjectName:            fmt.Sprintf("CN=Test Subject %d", i),
					}
					_, err := ds.writer(ctx).ExecContext(ctx,
						"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
						certificateTemplate.Name,
						certificateTemplate.TeamID,
						certificateTemplate.CertificateAuthorityID,
						certificateTemplate.SubjectName,
					)
					require.NoError(t, err)
				}
			}, func(t *testing.T, ds *Datastore) {
				hostUUIDs := []string{"test-host-uuid"}
				results, err := ds.ListCertificateTemplatesForHosts(context.Background(), hostUUIDs)
				require.NoError(t, err)
				require.Len(t, results, 2)

				for _, res := range results {
					require.Equal(t, "test-host-uuid", res.HostUUID)
					require.NotEmpty(t, res.CertificateTemplateID)
					require.Nil(t, res.FleetChallenge)
					require.Nil(t, res.Status)
				}
			},
		},
		{
			"Host with existing host certificate templates",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID := team.ID

				// Create a test host
				host := &fleet.Host{
					UUID:     "test-host-uuid-2",
					TeamID:   &teamID,
					Platform: "android",
				}
				_, err = ds.NewHost(context.Background(), host)
				require.NoError(t, err)

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID := ca.ID

				// Create certificate templates for the team
				for i := 0; i < 2; i++ {
					certificateTemplate := fleet.CertificateTemplate{
						Name:                   fmt.Sprintf("Cert%d", i),
						TeamID:                 teamID,
						CertificateAuthorityID: caID,
						SubjectName:            fmt.Sprintf("CN=Test Subject %d", i),
					}
					res, err := ds.writer(ctx).ExecContext(ctx,
						"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
						certificateTemplate.Name,
						certificateTemplate.TeamID,
						certificateTemplate.CertificateAuthorityID,
						certificateTemplate.SubjectName,
					)
					require.NoError(t, err)
					lastID, err := res.LastInsertId()
					require.NoError(t, err)

					if i == 0 {
						templateWithHostRecordId = uint(lastID) //nolint:gosec
						_, err = ds.writer(ctx).ExecContext(ctx,
							"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status) VALUES (?, ?, ?, ?)",
							host.UUID,
							lastID,
							"challenge",
							fleet.MDMDeliveryPending,
						)
						require.NoError(t, err)
					}
				}
			},
			func(t *testing.T, ds *Datastore) {
				hostUUIDs := []string{"test-host-uuid-2"}
				results, err := ds.ListCertificateTemplatesForHosts(context.Background(), hostUUIDs)
				require.NoError(t, err)
				require.Len(t, results, 2)

				for _, res := range results {
					require.Equal(t, "test-host-uuid-2", res.HostUUID)
					require.NotEmpty(t, res.CertificateTemplateID)
					if res.CertificateTemplateID == templateWithHostRecordId {
						require.Equal(t, ptr.String("challenge"), res.FleetChallenge)
						require.Equal(t, &fleet.MDMDeliveryPending, res.Status)
					} else {
						require.Nil(t, res.FleetChallenge)
						require.Nil(t, res.Status)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			tc.before(ds)

			tc.testFunc(t, ds)
		})
	}
}

func testBulkInsertAndDeleteHostCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	var certificateTemplateID uint
	var certificateTemplateIDTwo uint

	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"bulk inserts and deletes specific records",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID := team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)

				// Create two certificate templates
				res, err := ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					"Cert1", teamID, ca.ID, "CN=Test Subject 1",
				)
				require.NoError(t, err)
				lastID, err := res.LastInsertId()
				require.NoError(t, err)
				certificateTemplateID = uint(lastID) //nolint:gosec

				res, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					"Cert2", teamID, ca.ID, "CN=Test Subject 2",
				)
				require.NoError(t, err)
				lastID, err = res.LastInsertId()
				require.NoError(t, err)
				certificateTemplateIDTwo = uint(lastID) //nolint:gosec
			},
			func(t *testing.T, ds *Datastore) {
				// Insert host certificate templates
				hostCerts := []fleet.HostCertificateTemplate{
					{HostUUID: "host-1", CertificateTemplateID: certificateTemplateID, FleetChallenge: "challenge-1", Status: fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
					{HostUUID: "host-1", CertificateTemplateID: certificateTemplateIDTwo, FleetChallenge: "challenge-2", Status: fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
					{HostUUID: "host-2", CertificateTemplateID: certificateTemplateID, FleetChallenge: "challenge-3", Status: fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall},
				}
				err := ds.BulkInsertHostCertificateTemplates(ctx, hostCerts)
				require.NoError(t, err)

				var count int
				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 3, count)

				// Delete only host-1's first certificate
				toDelete := []fleet.HostCertificateTemplate{
					{HostUUID: "host-1", CertificateTemplateID: certificateTemplateID},
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
				require.Equal(t, certificateTemplateIDTwo, remaining[0].CertificateTemplateID)
				require.Equal(t, "host-2", remaining[1].HostUUID)
				require.Equal(t, certificateTemplateID, remaining[1].CertificateTemplateID)
			},
		},
		{
			"deletes multiple records at once",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID := team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)

				// Create two certificate templates
				res, err := ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					"Cert1", teamID, ca.ID, "CN=Test Subject 1",
				)
				require.NoError(t, err)
				lastID, err := res.LastInsertId()
				require.NoError(t, err)
				certificateTemplateID = uint(lastID) //nolint:gosec

				res, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					"Cert2", teamID, ca.ID, "CN=Test Subject 2",
				)
				require.NoError(t, err)
				lastID, err = res.LastInsertId()
				require.NoError(t, err)
				certificateTemplateIDTwo = uint(lastID) //nolint:gosec

				// Insert host certificate templates
				hostCerts := []fleet.HostCertificateTemplate{
					{HostUUID: "host-1", CertificateTemplateID: certificateTemplateID, FleetChallenge: "challenge-1", Status: fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
					{HostUUID: "host-1", CertificateTemplateID: certificateTemplateIDTwo, FleetChallenge: "challenge-2", Status: fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
				}
				err = ds.BulkInsertHostCertificateTemplates(ctx, hostCerts)
				require.NoError(t, err)
			},
			func(t *testing.T, ds *Datastore) {
				toDelete := []fleet.HostCertificateTemplate{
					{HostUUID: "host-1", CertificateTemplateID: certificateTemplateID},
					{HostUUID: "host-1", CertificateTemplateID: certificateTemplateIDTwo},
				}
				err := ds.DeleteHostCertificateTemplates(ctx, toDelete)
				require.NoError(t, err)

				var count int
				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 0, count)
			},
		},
		{
			"no error when deleting non-existent records",
			func(ds *Datastore) {},
			func(t *testing.T, ds *Datastore) {
				toDelete := []fleet.HostCertificateTemplate{
					{HostUUID: "non-existent-host", CertificateTemplateID: 999},
				}
				err := ds.DeleteHostCertificateTemplates(ctx, toDelete)
				require.NoError(t, err)
			},
		},
		{
			"no error with empty list",
			func(ds *Datastore) {},
			func(t *testing.T, ds *Datastore) {
				err := ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{})
				require.NoError(t, err)

				err = ds.DeleteHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{})
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			tc.before(ds)

			tc.testFunc(t, ds)
		})
	}
}

func testUpsertHostCertificateTemplateStatus(t *testing.T, ds *Datastore) {
	nodeKey := uuid.New().String()
	uuid := uuid.New().String()
	hostName := "test-update-host-certificate-template"

	ctx := t.Context()

	// Create a test team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
	require.NoError(t, err)
	teamID := team.ID

	// Create a test certificate authority
	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String("Test SCEP CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	ct1, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "Cert1",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 1",
	})
	require.NoError(t, err)
	require.NotNil(t, ct1)

	ct2, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "Cert2",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 1",
	})
	require.NoError(t, err)
	require.NotNil(t, ct2)

	// Create a host
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		NodeKey:  &nodeKey,
		UUID:     uuid,
		Hostname: hostName,
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	// Create a record in host_certificate_templates using ad hoc SQL
	sql := `
INSERT INTO host_certificate_templates (
	host_uuid,
	certificate_template_id,
	status,
	fleet_challenge
) VALUES (?, ?, ?, ?);
	`
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(context.Background(), sql, host.UUID, ct1.ID, "pending", "some_challenge_value")
		require.NoError(t, err)
		return nil
	})

	// Test cases
	cases := []struct {
		name             string
		templateID       uint
		newStatus        string
		expectedErrorMsg string
		detail           *string
	}{
		{
			name:       "Valid Update",
			templateID: ct1.ID,
			newStatus:  "verified",
		},
		{
			name:       "Valid Update with some details",
			templateID: ct1.ID,
			newStatus:  "failed",
			detail:     ptr.String("some details"),
		},
		{
			name:             "Invalid Status",
			templateID:       ct1.ID,
			newStatus:        "invalid_status",
			expectedErrorMsg: fmt.Sprintf("Invalid status '%s'", "invalid_status"),
		},
		{
			name:       "Creates a new status if record does not exist",
			templateID: ct2.ID,
			newStatus:  "verified",
			detail:     ptr.String("some details"),
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("TestUpdateHostCertificateTemplate:%s", tc.name), func(t *testing.T) {
			err := ds.UpsertCertificateStatus(context.Background(), host.UUID, tc.templateID, fleet.MDMDeliveryStatus(tc.newStatus), tc.detail)
			if tc.expectedErrorMsg == "" {
				require.NoError(t, err)
				// Verify the update
				var status string
				query := `
SELECT status FROM host_certificate_templates
WHERE host_uuid = ? AND certificate_template_id = ?;
				`
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					return sqlx.GetContext(context.Background(), q, &status, query, host.UUID, tc.templateID)
				})
				require.NoError(t, err)
				require.Equal(t, tc.newStatus, status)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErrorMsg)
			}
		})
	}
}
