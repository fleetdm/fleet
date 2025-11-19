package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestCertificates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetCertificateTemplateById", testGetCertificateTemplateByID},
		{"GetCertificateTemplatesByTeamID", testGetCertificateTemplatesByTeamID},
		{"BatchUpsertCertificates", testBatchUpsertCertificates},
		{"BatchDeleteCertificateTemplates", testBatchDeleteCertificateTemplates},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testGetCertificateTemplateByID(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	var teamID, caID uint
	var err error
	var certificateTemplateID uint
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
				templates, err := ds.GetCertificateTemplatesByTeamID(ctx, 1)
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
				templates, err := ds.GetCertificateTemplatesByTeamID(ctx, teamID)
				require.NoError(t, err)
				require.Len(t, templates, 2)
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
	var err error
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
				err = ds.BatchUpsertCertificateTemplates(ctx, []*fleet.CertificateTemplate{})
				require.NoError(t, err)
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

				err = ds.BatchUpsertCertificateTemplates(ctx, certificates)
				require.NoError(t, err)

				var count int
				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 2, count)
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
				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 1, count)

				certificates[0].SubjectName = "CN=Updated Subject 1"
				err = ds.BatchUpsertCertificateTemplates(ctx, certificates)
				require.NoError(t, err)

				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 1, count)

				var updatedSubject string
				err = ds.writer(ctx).GetContext(ctx, &updatedSubject, "SELECT subject_name FROM certificate_templates WHERE name = ?", "Cert1")
				require.NoError(t, err)
				require.Equal(t, "CN=Updated Subject 1", updatedSubject)
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
	var err error
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
				err = ds.BatchDeleteCertificateTemplates(ctx, []uint{})
				require.NoError(t, err)
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
				err = ds.writer(ctx).GetContext(ctx, &count, "SELECT COUNT(*) FROM certificate_templates")
				require.NoError(t, err)
				require.Equal(t, 2, count)

				err = ds.BatchDeleteCertificateTemplates(ctx, certificateTemplateIDs)
				require.NoError(t, err)

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
