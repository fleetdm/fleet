package macoffice

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/stretchr/testify/require"
)

func newMetadataFile(t *testing.T, name string) io.MetadataFileName {
	mfn, err := io.NewMSRCMetadata(name)
	require.NoError(t, err)
	return mfn
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	t.Run("#sync", func(t *testing.T) {
		remote := newMetadataFile(t, "macoffice-2023_10_10.json")

		t.Run("when there are no local files", func(t *testing.T) {
			var local []io.MetadataFileName

			testData := io.TestData{
				RemoteList: map[io.MetadataFileName]string{{}: "http://someurl.com"},
				LocalList:  local,
			}

			err := sync(ctx, io.FsMock{TestData: &testData}, io.GhMock{TestData: &testData})
			require.NoError(t, err)
			require.Empty(t, testData.LocalDeleted)
			require.Empty(t, testData.RemoteDownloaded)
		})

		t.Run("when there are no remote rel notes", func(t *testing.T) {
			local := []io.MetadataFileName{
				newMetadataFile(t, "macoffice-2022_09_10.json"),
			}

			testData := io.TestData{
				RemoteList: map[io.MetadataFileName]string{{}: "http://someurl.com"},
				LocalList:  local,
			}

			err := sync(ctx, io.FsMock{TestData: &testData}, io.GhMock{TestData: &testData})
			require.NoError(t, err)

			require.Empty(t, testData.LocalDeleted)
			require.Empty(t, testData.RemoteDownloaded)
		})

		t.Run("removes multiple out of date copies", func(t *testing.T) {
			local := []io.MetadataFileName{
				newMetadataFile(t, "macoffice-2022_09_10.json"),
				newMetadataFile(t, "macoffice-2022_08_10.json"),
			}

			testData := io.TestData{
				RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
				LocalList:  local,
			}

			err := sync(ctx, io.FsMock{TestData: &testData}, io.GhMock{TestData: &testData})
			require.NoError(t, err)

			require.ElementsMatch(t, testData.LocalDeleted, local)
			require.Contains(t, testData.RemoteDownloaded, "http://someurl.com")
		})

		t.Run("when local copy is out of date", func(t *testing.T) {
			local := newMetadataFile(t, "macoffice-2022_09_10.json")

			testData := io.TestData{
				RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
				LocalList:  []io.MetadataFileName{local},
			}

			err := sync(ctx, io.FsMock{TestData: &testData}, io.GhMock{TestData: &testData})
			require.NoError(t, err)

			require.ElementsMatch(t, testData.RemoteDownloaded, []string{"http://someurl.com"})
			require.ElementsMatch(t, testData.LocalDeleted, []io.MetadataFileName{local})
		})

		t.Run("when local copy is not out of date", func(t *testing.T) {
			local := []io.MetadataFileName{
				newMetadataFile(t, "macoffice-2023_11_10.json"),
				newMetadataFile(t, "macoffice-2023_01_10.json"),
			}

			testData := io.TestData{
				RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
				LocalList:  local,
			}

			err := sync(ctx, io.FsMock{TestData: &testData}, io.GhMock{TestData: &testData})
			require.NoError(t, err)

			require.Empty(t, testData.RemoteDownloaded)
			require.ElementsMatch(t, testData.LocalDeleted, []io.MetadataFileName{local[1]})
		})
	})
}
