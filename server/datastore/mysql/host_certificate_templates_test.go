package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestHostCertificateTemplates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ListAndroidHostUUIDsWithCertificateTemplates", testListAndroidHostUUIDsWithCertificateTemplates},
		{"ListCertificateTemplatesForHosts", testListCertificateTemplatesForHosts},
		{"BulkInsertHostCertificateTemplates", testBulkInsertHostCertificateTemplates},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testListAndroidHostUUIDsWithCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := context.Background()

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
				results, err := ds.ListAndroidHostUUIDsWithCertificateTemplates(ctx, 0, 10)
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
					"pending",
				)
				require.NoError(t, err)
			},
			func(t *testing.T, ds *Datastore) {
				results, err := ds.ListAndroidHostUUIDsWithCertificateTemplates(ctx, 0, 10)
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
				results, err := ds.ListAndroidHostUUIDsWithCertificateTemplates(ctx, 0, 10)
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
				results, err := ds.ListAndroidHostUUIDsWithCertificateTemplates(ctx, 0, 10)
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
	ctx := context.Background()

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
							"pending",
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
						require.Equal(t, ptr.String("pending"), res.Status)
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

func testBulkInsertHostCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var certificateTemplateID uint
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"bulk inserts host certificate templates",
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
				caID := ca.ID

				// Insert initial certificates
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
				certificateTemplateID = uint(lastID) //nolint:gosec
			},
			func(t *testing.T, ds *Datastore) {
				hostCertTemplates := []fleet.HostCertificateTemplate{
					{
						HostUUID:              "host-uuid-1",
						CertificateTemplateID: certificateTemplateID,
						FleetChallenge:        "challenge-1",
						Status:                "pending",
					},
					{
						HostUUID:              "host-uuid-2",
						CertificateTemplateID: certificateTemplateID,
						FleetChallenge:        "challenge-2",
						Status:                "issued",
					},
				}
				err := ds.BulkInsertHostCertificateTemplates(ctx, hostCertTemplates)
				require.NoError(t, err)

				var count int
				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM host_certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 2, count)
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
