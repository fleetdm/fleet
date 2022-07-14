package s3

import (
	"context"
	"io"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestInstallerExists(t *testing.T) {
	ctx := context.Background()
	store := setupInstallerStore(t, "installers", "random-prefix")

	t.Run("returns true for existing installers", func(t *testing.T) {
		installers := seedInstallerStore(t, store, "enroll-secret")

		for _, i := range installers {
			exists, err := store.Exists(ctx, i)
			require.NoError(t, err)
			require.Equal(t, exists, true)
		}
	})

	t.Run("returns false for non-existing installers", func(t *testing.T) {
		i := fleet.Installer{
			EnrollSecret: "non-existent",
			Kind:         "pkg",
			Desktop:      false,
		}
		exists, err := store.Exists(ctx, i)
		require.Error(t, err)
		require.Equal(t, exists, false)

		i = fleet.Installer{
			EnrollSecret: "non-existent",
			Kind:         "pkg",
			Desktop:      true,
		}
		exists, err = store.Exists(ctx, i)
		require.Error(t, err)
		require.Equal(t, exists, false)
	})
}

func TestGetInstaller(t *testing.T) {
	ctx := context.Background()
	store := setupInstallerStore(t, "installers", "random-prefix")

	t.Run("gets a blob with the file contents for each installer", func(t *testing.T) {
		installers := seedInstallerStore(t, store, "enroll-secret")

		for _, i := range installers {
			blob, err := store.Get(ctx, i)
			require.NoError(t, err)
			contents, err := io.ReadAll(blob)
			require.NoError(t, err)
			require.Equal(t, "mock", string(contents))
		}
	})

	t.Run("returns an error for non-existing installers", func(t *testing.T) {
		i := fleet.Installer{
			EnrollSecret: "non-existent",
			Kind:         "pkg",
			Desktop:      false,
		}
		blob, err := store.Get(ctx, i)
		require.Error(t, err)
		require.Nil(t, blob)

		i = fleet.Installer{
			EnrollSecret: "non-existent",
			Kind:         "pkg",
			Desktop:      true,
		}
		blob, err = store.Get(ctx, i)
		require.Error(t, err)
		require.Nil(t, blob)
	})
}
