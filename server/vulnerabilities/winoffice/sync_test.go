package winoffice

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/stretchr/testify/require"
)

func newMetadataFile(t *testing.T, name string) io.MetadataFileName {
	mfn, err := io.NewWinOfficeMetadata(name)
	require.NoError(t, err)
	return mfn
}

type testData struct {
	RemoteList          map[io.MetadataFileName]string
	RemoteListError     error
	RemoteDownloaded    []string
	RemoteDownloadError error
	LocalList           []io.MetadataFileName
	LocalListError      error
	LocalDeleted        []io.MetadataFileName
	LocalDeleteError    error
}

type ghMock struct{ TestData *testData }

func (gh ghMock) MSRCBulletins(ctx context.Context) (map[io.MetadataFileName]string, error) {
	return gh.TestData.RemoteList, gh.TestData.RemoteListError
}

func (gh ghMock) WinOfficeBulletin(ctx context.Context) (io.MetadataFileName, string, error) {
	for k, v := range gh.TestData.RemoteList {
		return k, v, gh.TestData.RemoteListError
	}
	return io.MetadataFileName{}, "", gh.TestData.RemoteListError
}

func (gh ghMock) MacOfficeReleaseNotes(ctx context.Context) (io.MetadataFileName, string, error) {
	for k, v := range gh.TestData.RemoteList {
		return k, v, gh.TestData.RemoteListError
	}
	return io.MetadataFileName{}, "", gh.TestData.RemoteListError
}

func (gh ghMock) Download(url string) (string, error) {
	gh.TestData.RemoteDownloaded = append(gh.TestData.RemoteDownloaded, url)
	return "", gh.TestData.RemoteDownloadError
}

type fsMock struct{ TestData *testData }

func (fs fsMock) MSRCBulletins() ([]io.MetadataFileName, error) {
	return fs.TestData.LocalList, fs.TestData.LocalListError
}

func (fs fsMock) WinOfficeBulletin() ([]io.MetadataFileName, error) {
	return fs.TestData.LocalList, fs.TestData.LocalListError
}

func (fs fsMock) MacOfficeReleaseNotes() ([]io.MetadataFileName, error) {
	return fs.TestData.LocalList, fs.TestData.LocalListError
}

func (fs fsMock) Delete(d io.MetadataFileName) error {
	fs.TestData.LocalDeleted = append(fs.TestData.LocalDeleted, d)
	return fs.TestData.LocalDeleteError
}

func TestSyncBulletin(t *testing.T) {
	ctx := t.Context()

	remote := newMetadataFile(t, "fleet_winoffice_bulletin-2023_10_10.json")

	t.Run("on GH error", func(t *testing.T) {
		td := testData{
			RemoteListError: errors.New("some error"),
		}
		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.Error(t, err)
	})

	t.Run("when nothing published on GH", func(t *testing.T) {
		td := testData{}
		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.NoError(t, err)
	})

	t.Run("on FS error", func(t *testing.T) {
		td := testData{
			RemoteList:     map[io.MetadataFileName]string{{}: "http://someurl.com"},
			LocalListError: errors.New("some error"),
		}
		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.Error(t, err)
	})

	t.Run("on error when downloading GH asset", func(t *testing.T) {
		td := testData{
			RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
			LocalList: []io.MetadataFileName{
				newMetadataFile(t, "fleet_winoffice_bulletin-2020_10_10.json"),
			},
			RemoteDownloadError: errors.New("some error"),
		}
		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.Error(t, err)
	})

	t.Run("when there are no local files", func(t *testing.T) {
		td := testData{
			RemoteList: map[io.MetadataFileName]string{{}: "http://someurl.com"},
		}

		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.NoError(t, err)
		require.Empty(t, td.LocalDeleted)
		require.Contains(t, td.RemoteDownloaded, "http://someurl.com")
	})

	t.Run("when local is newer than remote", func(t *testing.T) {
		td := testData{
			RemoteList: map[io.MetadataFileName]string{{}: "http://someurl.com"},
			LocalList: []io.MetadataFileName{
				newMetadataFile(t, "fleet_winoffice_bulletin-2024_09_10.json"),
			},
		}

		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.NoError(t, err)

		require.Empty(t, td.LocalDeleted)
		require.Empty(t, td.RemoteDownloaded)
	})

	t.Run("removes multiple out of date copies", func(t *testing.T) {
		local := []io.MetadataFileName{
			newMetadataFile(t, "fleet_winoffice_bulletin-2022_09_10.json"),
			newMetadataFile(t, "fleet_winoffice_bulletin-2022_08_10.json"),
		}

		td := testData{
			RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
			LocalList:  local,
		}

		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.NoError(t, err)

		require.ElementsMatch(t, td.LocalDeleted, local)
		require.Contains(t, td.RemoteDownloaded, "http://someurl.com")
	})

	t.Run("on error when deleting", func(t *testing.T) {
		local := []io.MetadataFileName{
			newMetadataFile(t, "fleet_winoffice_bulletin-2022_09_10.json"),
			newMetadataFile(t, "fleet_winoffice_bulletin-2022_08_10.json"),
		}

		td := testData{
			RemoteList:       map[io.MetadataFileName]string{remote: "http://someurl.com"},
			LocalList:        local,
			LocalDeleteError: errors.New("some error"),
		}

		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.Error(t, err)
	})

	t.Run("when local copy is out of date", func(t *testing.T) {
		local := newMetadataFile(t, "fleet_winoffice_bulletin-2022_09_10.json")

		td := testData{
			RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
			LocalList:  []io.MetadataFileName{local},
		}

		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.NoError(t, err)

		require.ElementsMatch(t, td.RemoteDownloaded, []string{"http://someurl.com"})
		require.ElementsMatch(t, td.LocalDeleted, []io.MetadataFileName{local})
	})

	t.Run("when local copy is not out of date", func(t *testing.T) {
		local := []io.MetadataFileName{
			newMetadataFile(t, "fleet_winoffice_bulletin-2023_11_10.json"),
			newMetadataFile(t, "fleet_winoffice_bulletin-2023_01_10.json"),
		}

		td := testData{
			RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
			LocalList:  local,
		}

		err := syncBulletin(ctx, fsMock{TestData: &td}, ghMock{TestData: &td})
		require.NoError(t, err)

		require.Empty(t, td.RemoteDownloaded)
		require.ElementsMatch(t, td.LocalDeleted, []io.MetadataFileName{local[1]})
	})
}
