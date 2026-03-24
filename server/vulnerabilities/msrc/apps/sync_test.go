package msrcapps

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/stretchr/testify/require"
)

func newMetadataFile(t *testing.T, name string) io.MetadataFileName {
	mfn, err := io.NewMSRCAppMetadata(name)
	require.NoError(t, err)
	return mfn
}

type testData struct {
	RemoteFile          io.MetadataFileName
	RemoteURL           string
	RemoteError         error
	RemoteDownloaded    []string
	RemoteDownloadError error
	LocalList           []io.MetadataFileName
	LocalListError      error
	LocalDeleted        []io.MetadataFileName
	LocalDeleteError    error
}

type ghMock struct{ TestData *testData }

func (gh ghMock) MSRCBulletins(ctx context.Context) (map[io.MetadataFileName]string, error) {
	return nil, nil
}

func (gh ghMock) MSRCAppBulletin(ctx context.Context) (io.MetadataFileName, string, error) {
	return gh.TestData.RemoteFile, gh.TestData.RemoteURL, gh.TestData.RemoteError
}

func (gh ghMock) MacOfficeReleaseNotes(ctx context.Context) (io.MetadataFileName, string, error) {
	return io.MetadataFileName{}, "", nil
}

func (gh ghMock) Download(url string) (string, error) {
	gh.TestData.RemoteDownloaded = append(gh.TestData.RemoteDownloaded, url)
	return "", gh.TestData.RemoteDownloadError
}

type fsMock struct{ TestData *testData }

func (fs fsMock) MSRCBulletins() ([]io.MetadataFileName, error) {
	return nil, nil
}

func (fs fsMock) MSRCAppBulletin() ([]io.MetadataFileName, error) {
	return fs.TestData.LocalList, fs.TestData.LocalListError
}

func (fs fsMock) MacOfficeReleaseNotes() ([]io.MetadataFileName, error) {
	return nil, nil
}

func (fs fsMock) Delete(d io.MetadataFileName) error {
	fs.TestData.LocalDeleted = append(fs.TestData.LocalDeleted, d)
	return fs.TestData.LocalDeleteError
}

func TestSync(t *testing.T) {
	ctx := t.Context()

	t.Run("#sync", func(t *testing.T) {
		remote := newMetadataFile(t, "fleet_msrc_app_apps-2023_10_10.json")

		t.Run("on GH error", func(t *testing.T) {
			testData := testData{
				RemoteError: errors.New("some error"),
			}
			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.Error(t, err, "some error")
		})

		t.Run("when nothing published on GH", func(t *testing.T) {
			testData := testData{}
			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.NoError(t, err)
		})

		t.Run("on FS error", func(t *testing.T) {
			testData := testData{
				RemoteFile:     remote,
				RemoteURL:      "http://someurl.com",
				LocalListError: errors.New("some error"),
			}
			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.Error(t, err, "some error")
		})

		t.Run("on error when downloading GH asset", func(t *testing.T) {
			testData := testData{
				RemoteFile: remote,
				RemoteURL:  "http://someurl.com",
				LocalList: []io.MetadataFileName{
					newMetadataFile(t, "fleet_msrc_app_apps-2020_10_10.json"),
				},
				RemoteDownloadError: errors.New("some error"),
			}
			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.Error(t, err, "some error")
		})

		t.Run("when there are no local files", func(t *testing.T) {
			testData := testData{
				RemoteFile: remote,
				RemoteURL:  "http://someurl.com",
			}

			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.NoError(t, err)
			require.Empty(t, testData.LocalDeleted)
			require.Contains(t, testData.RemoteDownloaded, "http://someurl.com")
		})

		t.Run("removes multiple out of date copies", func(t *testing.T) {
			local := []io.MetadataFileName{
				newMetadataFile(t, "fleet_msrc_app_apps-2022_09_10.json"),
				newMetadataFile(t, "fleet_msrc_app_apps-2022_08_10.json"),
			}

			testData := testData{
				RemoteFile: remote,
				RemoteURL:  "http://someurl.com",
				LocalList:  local,
			}

			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.NoError(t, err)

			require.ElementsMatch(t, testData.LocalDeleted, local)
			require.Contains(t, testData.RemoteDownloaded, "http://someurl.com")
		})

		t.Run("on error when deleting", func(t *testing.T) {
			local := []io.MetadataFileName{
				newMetadataFile(t, "fleet_msrc_app_apps-2022_09_10.json"),
				newMetadataFile(t, "fleet_msrc_app_apps-2022_08_10.json"),
			}

			testData := testData{
				RemoteFile:       remote,
				RemoteURL:        "http://someurl.com",
				LocalList:        local,
				LocalDeleteError: errors.New("some error"),
			}

			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.Error(t, err, "some error")
		})

		t.Run("when local copy is out of date", func(t *testing.T) {
			local := newMetadataFile(t, "fleet_msrc_app_apps-2022_09_10.json")

			testData := testData{
				RemoteFile: remote,
				RemoteURL:  "http://someurl.com",
				LocalList:  []io.MetadataFileName{local},
			}

			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.NoError(t, err)

			require.ElementsMatch(t, testData.RemoteDownloaded, []string{"http://someurl.com"})
			require.ElementsMatch(t, testData.LocalDeleted, []io.MetadataFileName{local})
		})

		t.Run("when local copy is not out of date", func(t *testing.T) {
			local := []io.MetadataFileName{
				newMetadataFile(t, "fleet_msrc_app_apps-2023_11_10.json"),
				newMetadataFile(t, "fleet_msrc_app_apps-2023_01_10.json"),
			}

			testData := testData{
				RemoteFile: remote,
				RemoteURL:  "http://someurl.com",
				LocalList:  local,
			}

			err := sync(ctx, fsMock{TestData: &testData}, ghMock{TestData: &testData})
			require.NoError(t, err)

			require.Empty(t, testData.RemoteDownloaded)
			require.ElementsMatch(t, testData.LocalDeleted, []io.MetadataFileName{local[1]})
		})
	})
}
