package mysql

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestCertificates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"CreateCertificateTemplate", testCreateCertificateTemplate},
		{"GetCertificateTemplateById", testGetCertificateTemplateByID},
		{"GetCertificateTemplatesByTeamID", testGetCertificateTemplatesByTeamID},
		{"DeleteCertificateTemplate", testDeleteCertificateTemplate},
		{"BatchUpsertCertificates", testBatchUpsertCertificates},
		{"BatchDeleteCertificateTemplates", testBatchDeleteCertificateTemplates},
		{"GetHostCertificateTemplates", testGetHostCertificateTemplates},
		{"GetCertificateTemplateForHost", testGetCertificateTemplateForHost},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testCreateCertificateTemplate(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, caID uint
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"Create certificate template",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID
			},
			func(t *testing.T, ds *Datastore) {
				certTemplate := &fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				savedTemplate, err := ds.CreateCertificateTemplate(ctx, certTemplate)
				require.NoError(t, err)
				require.NotNil(t, savedTemplate)

				require.NotZero(t, savedTemplate.ID)
				require.Equal(t, certTemplate.Name, savedTemplate.Name)
				require.Equal(t, certTemplate.CertificateAuthorityID, savedTemplate.CertificateAuthorityId)
				require.Equal(t, certTemplate.SubjectName, savedTemplate.SubjectName)
			},
		},
		{
			"Certificate template exists, fails to create",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Insert initial certificate
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
				certTemplate := &fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Another Subject ",
				}
				savedTemplate, err := ds.CreateCertificateTemplate(ctx, certTemplate)
				require.Error(t, err)
				require.Nil(t, savedTemplate)
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

func testGetCertificateTemplateByID(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	var teamID, caID uint
	var err error
	var certificateTemplateID uint
	var hostUUID string
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"No existing certificate template",
			func(ds *Datastore) {},
			func(t *testing.T, ds *Datastore) {
				_, err = ds.GetCertificateTemplateById(ctx, 0)
				require.Error(t, err)
			},
		},
		{
			"Get existing certificate template",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

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
				template, err := ds.GetCertificateTemplateById(ctx, certificateTemplateID)
				require.NoError(t, err)
				require.Equal(t, certificateTemplateID, template.ID)
			},
		},
		{
			"Template with pending host certificate template",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Create a test host
				host := &fleet.Host{
					UUID:     "test-host-uuid-2",
					TeamID:   &teamID,
					Platform: "android",
				}
				_, err = ds.NewHost(context.Background(), host)
				require.NoError(t, err)
				hostUUID = host.UUID

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

				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, name) VALUES (?, ?, ?, ?, ?)",
					host.UUID,
					certificateTemplateID,
					"fleet-challenge",
					fleet.CertificateTemplateDelivered,
					certificateTemplate.Name,
				)
				require.NoError(t, err)
			},
			func(t *testing.T, ds *Datastore) {
				// GetCertificateTemplateById should return template data without host-specific fields
				template, err := ds.GetCertificateTemplateById(ctx, certificateTemplateID)
				require.NoError(t, err)
				require.Equal(t, certificateTemplateID, template.ID)

				// GetCertificateTemplateByIdForHost should return host-specific data
				templateForHost, err := ds.GetCertificateTemplateByIdForHost(ctx, certificateTemplateID, hostUUID)
				require.NoError(t, err)
				require.Equal(t, certificateTemplateID, templateForHost.ID)
				require.Equal(t, fleet.CertificateTemplateDelivered, templateForHost.Status)
				require.Equal(t, "fleet-challenge", *templateForHost.FleetChallenge)
				require.Equal(t, "test-challenge", *templateForHost.SCEPChallenge)
			},
		},
		{
			"Template with verified host certificate template",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Create a test host
				host := &fleet.Host{
					UUID:     "test-host-uuid-2",
					TeamID:   &teamID,
					Platform: "android",
				}
				_, err = ds.NewHost(context.Background(), host)
				require.NoError(t, err)
				hostUUID = host.UUID

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

				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, name) VALUES (?, ?, ?, ?, ?)",
					host.UUID,
					certificateTemplateID,
					"challenge",
					fleet.CertificateTemplateVerified,
					certificateTemplate.Name,
				)
				require.NoError(t, err)
			},
			func(t *testing.T, ds *Datastore) {
				// GetCertificateTemplateById should return template data without host-specific fields
				template, err := ds.GetCertificateTemplateById(ctx, certificateTemplateID)
				require.NoError(t, err)
				require.Equal(t, certificateTemplateID, template.ID)

				// GetCertificateTemplateByIdForHost should return host-specific data (challenges nil for verified status)
				templateForHost, err := ds.GetCertificateTemplateByIdForHost(ctx, certificateTemplateID, hostUUID)
				require.NoError(t, err)
				require.Equal(t, certificateTemplateID, templateForHost.ID)
				require.Equal(t, fleet.CertificateTemplateVerified, templateForHost.Status)
				require.Nil(t, templateForHost.FleetChallenge)
				require.Nil(t, templateForHost.SCEPChallenge)
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

func testGetCertificateTemplatesByTeamID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, caID uint
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"No existing certificate templates for team",
			func(ds *Datastore) {},
			func(t *testing.T, ds *Datastore) {
				templates, _, err := ds.GetCertificateTemplatesByTeamID(ctx, 1, fleet.ListOptions{Page: 0, PerPage: 10})
				require.NoError(t, err)
				require.Len(t, templates, 0)
			},
		},
		{
			"Get existing certificate templates for team",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Insert initial certificates
				certificateTemplate1 := fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificateTemplate1.Name,
					certificateTemplate1.TeamID,
					certificateTemplate1.CertificateAuthorityID,
					certificateTemplate1.SubjectName,
				)
				require.NoError(t, err)

				certificateTemplate2 := fleet.CertificateTemplate{
					Name:                   "Cert2",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 2",
				}
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificateTemplate2.Name,
					certificateTemplate2.TeamID,
					certificateTemplate2.CertificateAuthorityID,
					certificateTemplate2.SubjectName,
				)
				require.NoError(t, err)
			},
			func(t *testing.T, ds *Datastore) {
				templates, _, err := ds.GetCertificateTemplatesByTeamID(ctx, teamID, fleet.ListOptions{Page: 0, PerPage: 10})
				require.NoError(t, err)
				require.Len(t, templates, 2)
			},
		},
		{
			"Pagination works",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Insert initial certificates
				for i := 1; i <= 5; i++ {
					certificateTemplate := fleet.CertificateTemplate{
						Name:                   fmt.Sprintf("Cert%d", i),
						TeamID:                 teamID,
						CertificateAuthorityID: caID,
						SubjectName:            fmt.Sprintf("CN=Test Subject %d", i),
					}
					_, err = ds.writer(ctx).ExecContext(ctx,
						"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
						certificateTemplate.Name,
						certificateTemplate.TeamID,
						certificateTemplate.CertificateAuthorityID,
						certificateTemplate.SubjectName,
					)
					require.NoError(t, err)
				}
			},
			func(t *testing.T, ds *Datastore) {
				// First page
				templates, meta, err := ds.GetCertificateTemplatesByTeamID(ctx, teamID, fleet.ListOptions{Page: 0, PerPage: 2, IncludeMetadata: true})
				require.NoError(t, err)
				require.Len(t, templates, 2)
				require.False(t, meta.HasPreviousResults)
				require.True(t, meta.HasNextResults)

				// Second page
				templates, meta, err = ds.GetCertificateTemplatesByTeamID(ctx, teamID, fleet.ListOptions{Page: 1, PerPage: 2, IncludeMetadata: true})
				require.NoError(t, err)
				require.Len(t, templates, 2)
				require.True(t, meta.HasPreviousResults)
				require.True(t, meta.HasNextResults)

				// Third page
				templates, meta, err = ds.GetCertificateTemplatesByTeamID(ctx, teamID, fleet.ListOptions{Page: 2, PerPage: 2, IncludeMetadata: true})
				require.NoError(t, err)
				require.Len(t, templates, 1)
				require.True(t, meta.HasPreviousResults)
				require.False(t, meta.HasNextResults)
			},
		},
		{
			"Get certificate templates for No team (team_id = 0)",
			func(ds *Datastore) {
				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Insert initial certificates
				for i := 0; i < 2; i++ {
					certificateTemplate1 := fleet.CertificateTemplate{
						Name:                   fmt.Sprintf("No Team Cert%d", i),
						TeamID:                 0,
						CertificateAuthorityID: caID,
						SubjectName:            fmt.Sprintf("CN=No Team Subject %d", i),
					}
					_, err = ds.writer(ctx).ExecContext(ctx,
						"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
						certificateTemplate1.Name,
						certificateTemplate1.TeamID,
						certificateTemplate1.CertificateAuthorityID,
						certificateTemplate1.SubjectName,
					)
					require.NoError(t, err)
				}
			},
			func(t *testing.T, ds *Datastore) {
				templates, meta, err := ds.GetCertificateTemplatesByTeamID(ctx, 0, fleet.ListOptions{Page: 0, PerPage: 10, IncludeMetadata: true})
				require.NoError(t, err)
				require.Len(t, templates, 2)
				require.Equal(t, uint(2), meta.TotalResults)

				for _, template := range templates {
					require.Contains(t, []string{"No Team Cert0", "No Team Cert1"}, template.Name)
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

func testDeleteCertificateTemplate(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, caID uint
	var certificateTemplateID uint
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"Delete existing certificate template",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Insert initial certificate
				certificateTemplate := fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				result, err := ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificateTemplate.Name,
					certificateTemplate.TeamID,
					certificateTemplate.CertificateAuthorityID,
					certificateTemplate.SubjectName,
				)
				require.NoError(t, err)
				lastID, err := result.LastInsertId()
				require.NoError(t, err)
				certificateTemplateID = uint(lastID) //nolint:gosec
			},
			func(t *testing.T, ds *Datastore) {
				err := ds.DeleteCertificateTemplate(ctx, certificateTemplateID)
				require.NoError(t, err)

				var count int
				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates WHERE id = ?", certificateTemplateID)
				require.NoError(t, err)
				require.Equal(t, 0, count)
			},
		},
		{
			"Delete non-existing certificate template",
			func(ds *Datastore) {},
			func(t *testing.T, ds *Datastore) {
				err := ds.DeleteCertificateTemplate(ctx, 0)
				require.Error(t, err)
				require.Equal(t, notFound("CertificateTemplate").WithID(0), err)
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

func testBatchUpsertCertificates(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, caID uint
	var certificates []*fleet.CertificateTemplate

	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"Empty slice",
			func(ds *Datastore) {},
			func(t *testing.T, ds *Datastore) {
				// Test with empty slice
				teamsModified, err := ds.BatchUpsertCertificateTemplates(ctx, []*fleet.CertificateTemplate{})
				require.NoError(t, err)

				require.Empty(t, teamsModified)
			},
		},
		{
			"Create certificates",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID
			},
			func(t *testing.T, ds *Datastore) {
				certificates := []*fleet.CertificateTemplate{
					{
						Name:                   "Cert1",
						TeamID:                 teamID,
						CertificateAuthorityID: caID,
						SubjectName:            "CN=Test Subject 1",
					},
					{
						Name:                   "Cert2",
						TeamID:                 teamID,
						CertificateAuthorityID: caID,
						SubjectName:            "CN=Test Subject 2",
					},
				}

				teamsModified, err := ds.BatchUpsertCertificateTemplates(ctx, certificates)
				require.NoError(t, err)

				var count int
				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 2, count)

				require.Len(t, teamsModified, 1)
				require.Equal(t, teamsModified[0], teamID)
			},
		},
		{
			"Upsert existing certificates",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Insert initial certificates
				certificate := fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				_, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificate.Name,
					certificate.TeamID,
					certificate.CertificateAuthorityID,
					certificate.SubjectName,
				)
				require.NoError(t, err)
				certificates = []*fleet.CertificateTemplate{&certificate}
			},
			func(t *testing.T, ds *Datastore) {
				var count int
				err := ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 1, count)

				certificates[0].SubjectName = "Updated Subject"
				teamsModified, err := ds.BatchUpsertCertificateTemplates(ctx, certificates)
				require.NoError(t, err)

				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 1, count)

				require.Len(t, teamsModified, 0)

				var subjectName string
				err = ds.writer(ctx).GetContext(ctx, &subjectName, "SELECT subject_name FROM certificate_templates WHERE name = ?", "Cert1")
				require.NoError(t, err)
				require.Equal(t, "CN=Test Subject 1", subjectName)
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

func testBatchDeleteCertificateTemplates(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, caID uint
	var certificateTemplateIDs []uint

	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"Empty slice",
			func(ds *Datastore) {},
			func(t *testing.T, ds *Datastore) {
				// Test with empty slice
				generateActivity, err := ds.BatchDeleteCertificateTemplates(ctx, []uint{})
				require.NoError(t, err)

				require.False(t, generateActivity)
			},
		},
		{
			"Delete existing certificates",
			func(ds *Datastore) {
				// Create a test team
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
				require.NoError(t, err)
				teamID = team.ID

				// Create a test certificate authority
				ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
					Type:      string(fleet.CATypeCustomSCEPProxy),
					Name:      ptr.String("Test SCEP CA"),
					URL:       ptr.String("http://localhost:8080/scep"),
					Challenge: ptr.String("test-challenge"),
				})
				require.NoError(t, err)
				caID = ca.ID

				// Insert initial certificates
				certificate1 := fleet.CertificateTemplate{
					Name:                   "Cert1",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 1",
				}
				res, err := ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificate1.Name,
					certificate1.TeamID,
					certificate1.CertificateAuthorityID,
					certificate1.SubjectName,
				)
				require.NoError(t, err)
				lastID1, err := res.LastInsertId()
				require.NoError(t, err)
				certificateTemplateIDs = append(certificateTemplateIDs, uint(lastID1)) //nolint:gosec

				certificate2 := fleet.CertificateTemplate{
					Name:                   "Cert2",
					TeamID:                 teamID,
					CertificateAuthorityID: caID,
					SubjectName:            "CN=Test Subject 2",
				}
				res, err = ds.writer(ctx).ExecContext(ctx,
					"INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name) VALUES (?, ?, ?, ?)",
					certificate2.Name,
					certificate2.TeamID,
					certificate2.CertificateAuthorityID,
					certificate2.SubjectName,
				)
				require.NoError(t, err)
				lastID2, err := res.LastInsertId()
				require.NoError(t, err)
				certificateTemplateIDs = append(certificateTemplateIDs, uint(lastID2)) //nolint:gosec
			},
			func(t *testing.T, ds *Datastore) {
				var count int
				err := ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 2, count)

				generateActivity, err := ds.BatchDeleteCertificateTemplates(ctx, certificateTemplateIDs)
				require.NoError(t, err)

				require.True(t, generateActivity)

				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 0, count)
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

func testGetHostCertificateTemplates(t *testing.T, ds *Datastore) {
	defer TruncateTables(t, ds)

	ctx := context.Background()

	h1 := test.NewHost(t, ds, "host_1", "127.0.0.1", "1", "1", time.Now())
	h2 := test.NewHost(t, ds, "host_2", "127.0.0.2", "2", "2", time.Now())

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
	require.NoError(t, err)

	h2.TeamID = &team.ID
	err = ds.UpdateHost(ctx, h2)
	require.NoError(t, err)

	// Create a test certificate authority
	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String("Test SCEP CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)

	// Create some certificate templates
	ct1, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "AAA",
		TeamID:                 team.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Test Subject 1",
	})
	require.NoError(t, err)

	ct2, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "BBB",
		TeamID:                 team.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Test Subject 2",
	})
	require.NoError(t, err)

	// Set the installation status on the certificate templates
	_, err = ds.writer(ctx).ExecContext(ctx,
		"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
		h2.UUID, ct1.ID, "test-challenge", fleet.OSSettingsVerified, fleet.MDMOperationTypeInstall, ct1.Name,
	)

	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, detail, operation_type, name) VALUES (?, ?, ?, ?, ?, ?, ?)",
		h2.UUID, ct2.ID, "test-challenge", fleet.OSSettingsFailed, "some error yooo", fleet.MDMOperationTypeInstall, ct2.Name,
	)
	require.NoError(t, err)

	testCases := []struct {
		name string
		do   func(*testing.T, *Datastore)
	}{
		{
			"hostUUID is not provided",
			func(t *testing.T, ds *Datastore) {
				_, err := ds.GetHostCertificateTemplates(ctx, "")
				require.Error(t, err)
			},
		},

		{
			"No certificate templates found",
			func(t *testing.T, ds *Datastore) {
				templates, err := ds.GetHostCertificateTemplates(ctx, h1.UUID)
				require.NoError(t, err)
				require.Empty(t, templates)
			},
		},
		{
			"Returns the certificates available for the host",
			func(t *testing.T, datastore *Datastore) {
				templates, err := ds.GetHostCertificateTemplates(ctx, h2.UUID)
				require.NoError(t, err)
				require.Len(t, templates, 2)

				// Sort the templates by name to make results deterministic
				sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })

				require.Equal(t, ct1.Name, templates[0].Name)
				require.Equal(t, fleet.CertificateTemplateVerified, templates[0].Status)
				require.Equal(t, fleet.MDMOperationTypeInstall, templates[0].OperationType)

				require.Equal(t, ct2.Name, templates[1].Name)
				require.Equal(t, fleet.CertificateTemplateFailed, templates[1].Status)
				require.Equal(t, "some error yooo", *templates[1].Detail)
				require.Equal(t, fleet.MDMOperationTypeInstall, templates[1].OperationType)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.do(t, ds)
		})
	}
}

func testGetCertificateTemplateForHost(t *testing.T, ds *Datastore) {
	defer TruncateTables(t, ds)

	ctx := context.Background()

	// Create teams
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 1"})
	require.NoError(t, err)

	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 2"})
	require.NoError(t, err)

	// Create hosts
	h1 := test.NewHost(t, ds, "host_1", "127.0.0.1", "1", "1", time.Now())
	h1.TeamID = &team1.ID
	err = ds.UpdateHost(ctx, h1)
	require.NoError(t, err)

	h2 := test.NewHost(t, ds, "host_2", "127.0.0.2", "2", "2", time.Now())
	h2.TeamID = &team2.ID
	err = ds.UpdateHost(ctx, h2)
	require.NoError(t, err)

	// Create certificate authority
	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String("Test SCEP CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)

	// Create certificate templates
	ct1, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "Template1",
		TeamID:                 team1.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Test Subject 1",
	})
	require.NoError(t, err)

	ct2, err := ds.CreateCertificateTemplate(ctx, &fleet.CertificateTemplate{
		Name:                   "Template2",
		TeamID:                 team2.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Test Subject 2",
	})
	require.NoError(t, err)

	// Create host_certificate_template record for h1 with ct1
	err = ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{
		{
			HostUUID:              h1.UUID,
			CertificateTemplateID: ct1.ID,
			FleetChallenge:        ptr.String("challenge-123"),
			Status:                fleet.CertificateTemplateDelivered,
			OperationType:         fleet.MDMOperationTypeInstall,
		},
	})
	require.NoError(t, err)

	testCases := []struct {
		name string
		do   func(*testing.T, *Datastore)
	}{
		{
			"Returns certificate template for host with host_certificate_template record",
			func(t *testing.T, ds *Datastore) {
				result, err := ds.GetCertificateTemplateForHost(ctx, h1.UUID, ct1.ID)
				require.NoError(t, err)
				require.NotNil(t, result)

				require.Equal(t, h1.UUID, result.HostUUID)
				require.Equal(t, ct1.ID, result.CertificateTemplateID)
				require.NotNil(t, result.FleetChallenge)
				require.Equal(t, "challenge-123", *result.FleetChallenge)
				require.NotNil(t, result.Status)
				require.Equal(t, fleet.CertificateTemplateDelivered, *result.Status)
				require.Equal(t, fleet.CAConfigAssetType(fleet.CATypeCustomSCEPProxy), result.CAType)
				require.Equal(t, "Test SCEP CA", result.CAName)
			},
		},
		{
			"Returns certificate template for host without host_certificate_template record",
			func(t *testing.T, ds *Datastore) {
				// h2 is in team2 which has ct2, but no host_certificate_template record exists
				result, err := ds.GetCertificateTemplateForHost(ctx, h2.UUID, ct2.ID)
				require.NoError(t, err)
				require.NotNil(t, result)

				require.Equal(t, h2.UUID, result.HostUUID)
				require.Equal(t, ct2.ID, result.CertificateTemplateID)
				require.Nil(t, result.FleetChallenge)
				require.Nil(t, result.Status)
				require.Equal(t, fleet.CAConfigAssetType(fleet.CATypeCustomSCEPProxy), result.CAType)
				require.Equal(t, "Test SCEP CA", result.CAName)
			},
		},
		{
			"Returns error when certificate template doesn't belong to host's team",
			func(t *testing.T, ds *Datastore) {
				// h1 is in team1, ct2 is in team2
				_, err := ds.GetCertificateTemplateForHost(ctx, h1.UUID, ct2.ID)
				require.Error(t, err)
			},
		},
		{
			"Returns error for non-existent host",
			func(t *testing.T, ds *Datastore) {
				_, err := ds.GetCertificateTemplateForHost(ctx, "non-existent-uuid", ct1.ID)
				require.Error(t, err)
			},
		},
		{
			"Returns error for non-existent certificate template",
			func(t *testing.T, ds *Datastore) {
				_, err := ds.GetCertificateTemplateForHost(ctx, h1.UUID, 99999)
				require.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.do(t, ds)
		})
	}
}
