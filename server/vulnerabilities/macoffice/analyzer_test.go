package macoffice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/stretchr/testify/require"
)

func TestAnalyzer(t *testing.T) {
	ctx := context.Background()

	t.Run("Analyze", func(t *testing.T) {
		t.Run("when using wrong path", func(t *testing.T) {
			vulns, err := Analyze(ctx, nil, "some bad path", false)
			require.Empty(t, vulns)
			require.Error(t, err)
		})

		t.Run("when no release notes on path", func(t *testing.T) {
			vulnDir := t.TempDir()
			vulns, err := Analyze(ctx, nil, vulnDir, false)
			require.Empty(t, vulns)
			require.NoError(t, err)
		})
	})

	t.Run("updateVulnsInDB", func(t *testing.T) {
		t.Run("on error when deleting vulns", func(t *testing.T) {
			ds := new(mock.Store)
			ds.DeleteSoftwareVulnerabilitiesFunc = func(ctx context.Context, vulnerabilities []fleet.SoftwareVulnerability) error {
				return errors.New("some error")
			}
			ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, source fleet.VulnerabilitySource) (bool, error) {
				return false, nil
			}

			vulns, err := updateVulnsInDB(ctx, ds, nil, nil)
			require.Empty(t, vulns)
			require.Error(t, err, "some error")
		})

		t.Run("on error when inserting vulns", func(t *testing.T) {
			detected := []fleet.SoftwareVulnerability{
				{SoftwareID: 1, CVE: "123"},
			}

			ds := new(mock.Store)
			ds.DeleteSoftwareVulnerabilitiesFunc = func(ctx context.Context, vulnerabilities []fleet.SoftwareVulnerability) error {
				return nil
			}
			ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, source fleet.VulnerabilitySource) (bool, error) {
				return false, errors.New("some error")
			}

			vulns, err := updateVulnsInDB(ctx, ds, detected, nil)
			require.Empty(t, vulns)
			require.Error(t, err, "some error")
		})
	})

	t.Run("collectVulnerabilities", func(t *testing.T) {
		t.Run("no release notes", func(t *testing.T) {
			software := fleet.Software{}
			var relNotes ReleaseNotes
			vulns := collectVulnerabilities(&software, Word, relNotes)
			require.Empty(t, vulns)
		})
	})

	t.Run("getStoredVulnerabilities", func(t *testing.T) {
		t.Run("on error", func(t *testing.T) {
			ds := new(mock.Store)
			ds.SoftwareByIDFunc = func(ctx context.Context, id uint, teamID *uint, includeCVEScores bool, tmFilter *fleet.TeamFilter) (*fleet.Software, error) {
				return nil, errors.New("some error")
			}

			vulns, err := getStoredVulnerabilities(ctx, ds, uint(0))
			require.Empty(t, vulns)
			require.Error(t, err, "some error")
		})
	})

	t.Run("latestReleaseNotes", func(t *testing.T) {
		t.Run("returns release notes in order", func(t *testing.T) {
			vulnPath := t.TempDir()

			releaseNotes := ReleaseNotes{
				{Version: "1", Date: time.Now().Add(-36 * time.Hour)},
				{Version: "2", Date: time.Now()},
			}

			err := releaseNotes.Serialize(time.Now(), vulnPath)
			require.NoError(t, err)

			actual, err := getLatestReleaseNotes(vulnPath)
			require.NoError(t, err)
			require.Equal(t, releaseNotes[0].Version, actual[1].Version)
			require.Equal(t, releaseNotes[1].Version, actual[0].Version)
		})

		t.Run("when vuln path exists", func(t *testing.T) {
			vulnPath := t.TempDir()

			actual, err := getLatestReleaseNotes(vulnPath)
			require.NoError(t, err)
			require.Empty(t, actual)

			err = ReleaseNotes{{Version: "2"}}.Serialize(time.Now(), vulnPath)
			require.NoError(t, err)

			err = ReleaseNotes{{Version: "1"}}.Serialize(time.Now().Add(-35*time.Hour), vulnPath)
			require.NoError(t, err)

			actual, err = getLatestReleaseNotes(vulnPath)
			require.NoError(t, err)
			require.NotEmpty(t, actual)
			require.Equal(t, "2", actual[0].Version)
		})

		t.Run("when vuln path does not exists", func(t *testing.T) {
			releaseNotes, err := getLatestReleaseNotes("bad path")
			require.Empty(t, releaseNotes)
			require.Error(t, err)
		})

		t.Run("when the JSON file is invalid", func(t *testing.T) {
			vulnPath := t.TempDir()

			fileName := io.MacOfficeRelNotesFileName(time.Now())
			filePath := filepath.Join(vulnPath, fileName)

			f, err := os.Create(filePath)
			require.NoError(t, err)
			defer f.Close()

			_, err = f.WriteString("some bad json")
			require.NoError(t, err)

			_, err = getLatestReleaseNotes(vulnPath)
			require.Error(t, err)
		})
	})
}
