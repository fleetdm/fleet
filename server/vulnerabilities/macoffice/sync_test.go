package macoffice

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/stretchr/testify/require"
)

func TestSync(t *testing.T) {
	ctx := context.Background()
	t.Run("#sync", func(t *testing.T) {
		remote := io.NewMacOfficeRelNotesMetadata("macoffice-2023_10_10.json")

		t.Run("removes multiple out of date copies", func(t *testing.T) {
			local := []io.MetadataFileName{
				io.NewMacOfficeRelNotesMetadata("macoffice-2022_09_10.json"),
				io.NewMacOfficeRelNotesMetadata("macoffice-2022_08_10.json"),
			}

			testData := io.TestData{
				RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
				LocalList:  local,
			}

			err := sync(ctx, io.FsMock{TestData: &testData}, io.GhMock{TestData: &testData})
			require.NoError(t, err)

			require.ElementsMatch(t, testData.LocalDeleted, local)
		})

		t.Run("when local copy is out of date", func(t *testing.T) {
			local := io.NewMacOfficeRelNotesMetadata("macoffice-2022_09_10.json")

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
			local := io.NewMacOfficeRelNotesMetadata("macoffice-2023_10_10.json")

			testData := io.TestData{
				RemoteList: map[io.MetadataFileName]string{remote: "http://someurl.com"},
				LocalList:  []io.MetadataFileName{local},
			}

			err := sync(ctx, io.FsMock{TestData: &testData}, io.GhMock{TestData: &testData})
			require.NoError(t, err)

			require.Empty(t, testData.RemoteDownloaded)
			require.Empty(t, testData.LocalDeleted)
		})
	})
}
