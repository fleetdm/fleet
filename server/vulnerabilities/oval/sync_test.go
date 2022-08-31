package oval

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestSync(t *testing.T) {
	t.Run("#removeOldDefs", func(t *testing.T) {
		t.Run("with empty dir", func(t *testing.T) {
			path := t.TempDir()
			date := time.Now()

			result, err := removeOldDefs(date, path)
			require.Empty(t, result)
			require.Nil(t, err)
		})

		t.Run("with old definitions", func(t *testing.T) {
			hostPlatform := "ubuntu"
			hostOsVersion := "Ubuntu 20.4.0"
			ovalPlatform := NewPlatform(hostPlatform, hostOsVersion)

			today := time.Now()
			yesterday := today.Add(-24 * time.Hour)

			path := t.TempDir()

			otherFile1 := filepath.Join(path, "my_lyrics.json")
			newDef := filepath.Join(path, ovalPlatform.ToFilename(today, "json"))
			oldDef := filepath.Join(path, ovalPlatform.ToFilename(yesterday, "json"))

			f1, err := os.Create(newDef)
			require.NoError(t, err)
			f1.Close()

			f2, err := os.Create(oldDef)
			require.NoError(t, err)
			f2.Close()

			f3, err := os.Create(otherFile1)
			require.NoError(t, err)
			f3.Close()

			r, err := removeOldDefs(today, path)
			require.NoError(t, err)
			require.Contains(t, r, filepath.Base(newDef))

			_, err = os.Stat(oldDef)
			require.True(t, os.IsNotExist(err))

			_, err = os.Stat(otherFile1)
			require.NoError(t, err)
		})
	})

	t.Run("#whatToDownload", func(t *testing.T) {
		today := time.Now()

		osVersions := fleet.OSVersions{
			CountsUpdatedAt: time.Now(),
			OSVersions: []fleet.OSVersion{
				{
					HostsCount: 1,
					Platform:   "ubuntu",
					Name:       "Ubuntu 20.4.0",
				},
				{
					HostsCount: 1,
					Platform:   "ubuntu",
					Name:       "Ubuntu 18.4.0",
				},
				{
					HostsCount: 1,
					Platform:   "rhle",
					Name:       "CentOS Linux 8.3.2011",
				},
			},
		}

		existing := map[string]bool{
			NewPlatform("ubuntu", "Ubuntu 20.4.0").ToFilename(today, "json"): true,
		}

		r := whatToDownload(&osVersions, existing, today)
		require.Len(t, r, 1)
		require.Contains(t, r, NewPlatform("ubuntu", "Ubuntu 18.4.0"))
		require.NotContains(t, r, NewPlatform("rhle", "CentOS Linux 8.3.2011"))
	})
}
