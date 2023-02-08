package msrc

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/stretchr/testify/require"
)

type testData struct {
	remoteList          map[io.MetadataFileName]string
	remoteListError     error
	remoteDownloaded    []string
	remoteDownloadError error
	localList           []io.MetadataFileName
	localListError      error
	localDeleted        []io.MetadataFileName
	localDeleteError    error
}

type ghMock struct{ testData *testData }

func (gh ghMock) Bulletins(ctx context.Context) (map[io.MetadataFileName]string, error) {
	return gh.testData.remoteList, gh.testData.remoteListError
}

func (gh ghMock) Download(url string) (string, error) {
	gh.testData.remoteDownloaded = append(gh.testData.remoteDownloaded, url)
	return "", gh.testData.remoteDownloadError
}

type fsMock struct{ testData *testData }

func (fs fsMock) MSRCBulletins() ([]io.MetadataFileName, error) {
	return fs.testData.localList, fs.testData.localListError
}

func (fs fsMock) Delete(d io.MetadataFileName) error {
	fs.testData.localDeleted = append(fs.testData.localDeleted, d)
	return fs.testData.localDeleteError
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
			remoteList: map[io.MetadataFileName]string{
				io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json"): "http://somebulletin.com",
			},
			localList: []io.MetadataFileName{io.NewMSRCMetadataFileName("Windows_10-2022_09_10.json")},
		}

		err := sync(ctx, os, fsMock{testData: &testData}, ghMock{testData: &testData})
		require.NoError(t, err)
		require.ElementsMatch(t, testData.remoteDownloaded, []string{"http://somebulletin.com"})
		require.ElementsMatch(t, testData.localDeleted, []io.MetadataFileName{io.NewMSRCMetadataFileName("Windows_10-2022_09_10.json")})
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
			}
			t.Run("without remote bulletins", func(t *testing.T) {
				var remote []io.MetadataFileName
				local := []io.MetadataFileName{
					io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json"),
				}
				toDownload, toDelete := bulletinsDelta(os, local, remote)
				require.Empty(t, toDownload)
				require.Empty(t, toDelete)
			})

			t.Run("with remote bulletins", func(t *testing.T) {
				remote := []io.MetadataFileName{
					io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json"),
					io.NewMSRCMetadataFileName("Windows_11-2022_10_10.json"),
					io.NewMSRCMetadataFileName("Windows_Server_2016-2022_10_10.json"),
					io.NewMSRCMetadataFileName("Windows_8.1-2022_10_10.json"),
				}
				t.Run("no local bulletins", func(t *testing.T) {
					var local []io.MetadataFileName
					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []io.MetadataFileName{
						io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json"),
						io.NewMSRCMetadataFileName("Windows_11-2022_10_10.json"),
					})
					require.Empty(t, toDelete)
				})

				t.Run("missing some local bulletin", func(t *testing.T) {
					local := []io.MetadataFileName{
						io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json"),
					}
					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []io.MetadataFileName{
						io.NewMSRCMetadataFileName("Windows_11-2022_10_10.json"),
					})
					require.Empty(t, toDelete)
				})

				t.Run("out of date local bulletin", func(t *testing.T) {
					local := []io.MetadataFileName{
						io.NewMSRCMetadataFileName("Windows_10-2022_09_10.json"),
						io.NewMSRCMetadataFileName("Windows_11-2022_10_10.json"),
					}

					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []io.MetadataFileName{
						io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json"),
					})
					require.ElementsMatch(t, toDelete, []io.MetadataFileName{
						io.NewMSRCMetadataFileName("Windows_10-2022_09_10.json"),
					})
				})

				t.Run("up to date local bulletins", func(t *testing.T) {
					local := []io.MetadataFileName{
						io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json"),
						io.NewMSRCMetadataFileName("Windows_11-2022_10_10.json"),
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
			local := []io.MetadataFileName{io.NewMSRCMetadataFileName("Windows_11-2022_10_10.json")}
			remote := []io.MetadataFileName{io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json")}

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
						io.NewMSRCMetadataFileName("Windows_10-2022_10_10.json"),
						io.NewMSRCMetadataFileName("Windows_11-2022_10_10.json"),
					}

					toDownload, toDelete := bulletinsDelta(os, local, remote)
					require.ElementsMatch(t, toDownload, remote)
					require.Empty(t, toDelete)
				})
			})
		})
	})
}
