package msrc

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/stretchr/testify/require"
)

func newMetadataFile(t *testing.T, name string) io.MetadataFileName {
	mfn, err := io.NewMSRCMetadata(name)
	require.NoError(t, err)
	return mfn
}

type testData struct {
	RemoteList       map[io.MetadataFileName]string
	RemoteDownloaded []string
	LocalList        []io.MetadataFileName
	LocalDeleted     []io.MetadataFileName
}

type ghMock struct{ TestData *testData }

func (gh ghMock) MSRCBulletins(ctx context.Context) (map[io.MetadataFileName]string, error) {
	return gh.TestData.RemoteList, nil
}

func (gh ghMock) MacOfficeReleaseNotes(ctx context.Context) (io.MetadataFileName, string, error) {
	for k, v := range gh.TestData.RemoteList {
		return k, v, nil
	}
	return io.MetadataFileName{}, "", nil
}

func (gh ghMock) Download(url string) (string, error) {
	gh.TestData.RemoteDownloaded = append(gh.TestData.RemoteDownloaded, url)
	return "", nil
}

type fsMock struct{ TestData *testData }

func (fs fsMock) MSRCBulletins() ([]io.MetadataFileName, error) {
	return fs.TestData.LocalList, nil
}

func (fs fsMock) MacOfficeReleaseNotes() ([]io.MetadataFileName, error) {
	return fs.TestData.LocalList, nil
}

func (fs fsMock) Delete(d io.MetadataFileName) error {
	fs.TestData.LocalDeleted = append(fs.TestData.LocalDeleted, d)
	return nil
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	t.Run("#sync", func(t *testing.T) {
		os := []fleet.OperatingSystem{
			{
				Name:          "Microsoft Windows 11 Enterprise",
				Version:       "21H2",
				Arch:          "64-bit",
				KernelVersion: "10.0.22000.795",
			},
			{
				Name:          "Microsoft Windows 10 Pro",
				Version:       "10.0.19044",
				Arch:          "64-bit",
				KernelVersion: "10.0.19044",
			},
		}

		testData := testData{
			RemoteList: map[io.MetadataFileName]string{
				newMetadataFile(t, "Windows_10-2022_10_10.json"): "http://somebulletin.com",
			},
			LocalList: []io.MetadataFileName{newMetadataFile(t, "Windows_10-2022_09_10.json")},
		}

		err := sync(ctx, os, fsMock{TestData: &testData}, ghMock{TestData: &testData})
		require.NoError(t, err)
		require.ElementsMatch(t, testData.RemoteDownloaded, []string{"http://somebulletin.com"})

		expectedMfn, err := io.NewMSRCMetadata("Windows_10-2022_09_10.json")
		require.NoError(t, err)
		require.ElementsMatch(t, testData.LocalDeleted, []io.MetadataFileName{expectedMfn})
	})

	t.Run("#bulletinsDelta", func(t *testing.T) {
		t.Run("win OS provided", func(t *testing.T) {
			os := []fleet.OperatingSystem{
				{
					Name:          "Microsoft Windows 11 Enterprise",
					Version:       "21H2",
					Arch:          "64-bit",
					KernelVersion: "10.0.22000.795",
				},
				{
					Name:          "Microsoft Windows 10 Pro",
					Version:       "10.0.19044",
					Arch:          "64-bit",
					KernelVersion: "10.0.19044",
				},
				{ // multiple versions of the same OS map to one file, should only be downloaded/deleted once
					Name:          "Microsoft Windows 10 Pro",
					Version:       "10.0.19045",
					Arch:          "64-bit",
					KernelVersion: "10.0.19045",
				},
			}
			t.Run("without remote bulletins", func(t *testing.T) {
				var remote []io.MetadataFileName
				local := []io.MetadataFileName{
					newMetadataFile(t, "Windows_10-2022_10_10.json"),
				}
				toDownload, toDelete := bulletinsDelta(os, local, remote)
				require.Empty(t, toDownload)
				require.Empty(t, toDelete)
			})

			t.Run("with remote bulletins", func(t *testing.T) {
				remote := []io.MetadataFileName{
					newMetadataFile(t, "Windows_10-2022_10_10.json"),
					newMetadataFile(t, "Windows_11-2022_10_10.json"),
					newMetadataFile(t, "Windows_Server_2016-2022_10_10.json"),
					newMetadataFile(t, "Windows_8.1-2022_10_10.json"),
				}
				t.Run("no local bulletins", func(t *testing.T) {
					var local []io.MetadataFileName
					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []io.MetadataFileName{
						newMetadataFile(t, "Windows_10-2022_10_10.json"),
						newMetadataFile(t, "Windows_11-2022_10_10.json"),
					})
					require.Empty(t, toDelete)
				})

				t.Run("missing some local bulletin", func(t *testing.T) {
					local := []io.MetadataFileName{
						newMetadataFile(t, "Windows_10-2022_10_10.json"),
					}
					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []io.MetadataFileName{
						newMetadataFile(t, "Windows_11-2022_10_10.json"),
					})
					require.Empty(t, toDelete)
				})

				t.Run("out of date local bulletin", func(t *testing.T) {
					local := []io.MetadataFileName{
						newMetadataFile(t, "Windows_10-2022_09_10.json"),
						newMetadataFile(t, "Windows_11-2022_10_10.json"),
					}

					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []io.MetadataFileName{
						newMetadataFile(t, "Windows_10-2022_10_10.json"),
					})
					require.ElementsMatch(t, toDelete, []io.MetadataFileName{
						newMetadataFile(t, "Windows_10-2022_09_10.json"),
					})
				})

				t.Run("up to date local bulletins", func(t *testing.T) {
					local := []io.MetadataFileName{
						newMetadataFile(t, "Windows_10-2022_10_10.json"),
						newMetadataFile(t, "Windows_11-2022_10_10.json"),
					}

					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.Empty(t, toDownload)
					require.Empty(t, toDelete)
				})
			})
		})

		t.Run("no Win OS provided", func(t *testing.T) {
			os := []fleet.OperatingSystem{
				{
					Name:          "CentOS",
					Version:       "8.0.0",
					Platform:      "rhel",
					KernelVersion: "5.10.76-linuxkit",
				},
			}
			local := []io.MetadataFileName{newMetadataFile(t, "Windows_11-2022_10_10.json")}
			remote := []io.MetadataFileName{newMetadataFile(t, "Windows_10-2022_10_10.json")}

			t.Run("nothing to download, nothing to delete", func(t *testing.T) {
				toDownload, toDelete := bulletinsDelta(os, local, remote)
				require.Empty(t, toDownload)
				require.Empty(t, toDelete)
			})
		})

		t.Run("no OS provided", func(t *testing.T) {
			var os []fleet.OperatingSystem
			t.Run("no local bulletins", func(t *testing.T) {
				var local []io.MetadataFileName

				t.Run("returns all remote", func(t *testing.T) {
					remote := []io.MetadataFileName{
						newMetadataFile(t, "Windows_10-2022_10_10.json"),
						newMetadataFile(t, "Windows_11-2022_10_10.json"),
					}

					toDownload, toDelete := bulletinsDelta(os, local, remote)
					require.ElementsMatch(t, toDownload, remote)
					require.Empty(t, toDelete)
				})
			})
		})
	})
}
