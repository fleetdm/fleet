package s3

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestGetInstaller(t *testing.T) {
	ctx := context.Background()
	store := SetupTestInstallerStore(t, "installers-unit-test", "get-prefix")

	t.Run("gets a blob with the file contents for each installer", func(t *testing.T) {
		installers := SeedTestInstallerStore(t, store, "enroll-secret")

		for _, i := range installers {
			blob, length, err := store.Get(ctx, i)
			require.NoError(t, err)
			contents, err := io.ReadAll(blob)
			require.NoError(t, err)
			require.Equal(t, "mock", string(contents))
			require.EqualValues(t, length, len(contents))
		}
	})

	t.Run("returns an error for non-existing installers", func(t *testing.T) {
		i := fleet.Installer{
			EnrollSecret: "non-existent",
			Kind:         "pkg",
			Desktop:      false,
		}
		blob, length, err := store.Get(ctx, i)
		require.Error(t, err)
		require.Nil(t, blob)
		require.Zero(t, length)

		i = fleet.Installer{
			EnrollSecret: "non-existent",
			Kind:         "pkg",
			Desktop:      true,
		}
		blob, length, err = store.Get(ctx, i)
		require.Error(t, err)
		require.Nil(t, blob)
		require.Zero(t, length)
	})
}

func TestInstallerPut(t *testing.T) {
	store := SetupTestInstallerStore(t, "installers-unit-test", "put-prefix")

	i := fleet.Installer{
		EnrollSecret: "xyz",
		Kind:         "pkg",
		Desktop:      false,
		Content:      aws.ReadSeekCloser(strings.NewReader(mockInstallerContents)),
	}
	key, err := store.Put(context.Background(), i)
	require.NoError(t, err)
	require.Equal(t, store.keyForInstaller(i), key)

	ri, l, err := store.Get(context.Background(), i)
	require.NoError(t, err)

	rc, err := io.ReadAll(ri)
	require.NoError(t, err)
	require.EqualValues(t, len(mockInstallerContents), l)

	require.Equal(t, mockInstallerContents, string(rc))
}
