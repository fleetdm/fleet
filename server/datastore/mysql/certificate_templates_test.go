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
		{"BatchUpsertCertificates", testBatchUpsertCertificates},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testBatchUpsertCertificates(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, caID uint
	var certificates []*fleet.Certificate
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
				err = ds.BatchUpsertCertificateTemplates(ctx, []*fleet.Certificate{})
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
				certificates := []*fleet.Certificate{
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
				certificate := fleet.Certificate{
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
				certificates = []*fleet.Certificate{&certificate}
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
