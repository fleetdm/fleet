package s3

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallerExists(t *testing.T) {
	ctx := context.Background()
	store := setupInstallerStore(t, "installers", "random-prefix")
	defer cleanupStore(t, store)

	t.Run("returns true for existing installers", func(t *testing.T) {
		installers := seedInstallerStore(t, store, "abc")

		for _, i := range installers {
			exists, err := store.Exists(ctx, *i)
			require.NoError(t, err)
			require.Equal(t, exists, true)
		}
	})

	t.Run("returns false for non-existing installers", func(t *testing.T) {
		exists, err := store.Exists(ctx, Installer{"non-existent", "pkg", false})
		require.Error(t, err)
		require.Equal(t, exists, false)

		exists, err = store.Exists(ctx, Installer{"non-existent", "pkg", true})
		require.Error(t, err)
		require.Equal(t, exists, false)
	})
}

func TestGetInstaller(t *testing.T) {
	ctx := context.Background()
	store := setupInstallerStore(t, "installers", "random-prefix")
	defer cleanupStore(t, store)

	t.Run("gets a blob with the file contents for each installer", func(t *testing.T) {
		installers := seedInstallerStore(t, store, "abc")

		for _, i := range installers {
			blob, err := store.Get(ctx, *i)
			require.NoError(t, err)
			contents, err := io.ReadAll(blob)
			require.NoError(t, err)
			require.Equal(t, "mock", string(contents))
		}
	})

	t.Run("returns an error for non-existing installers", func(t *testing.T) {
		blob, err := store.Get(ctx, Installer{"non-existent", "pkg", false})
		require.Error(t, err)
		require.Nil(t, blob)

		blob, err = store.Get(ctx, Installer{"non-existent", "pkg", true})
		require.Error(t, err)
		require.Nil(t, blob)
	})
}
