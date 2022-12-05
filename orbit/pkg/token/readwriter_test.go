package token

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
)

func TestLoadOrGenerate(t *testing.T) {
	t.Run("creates file if it doesn't exist", func(t *testing.T) {
		dir := os.TempDir()
		file := filepath.Join(dir, "identifier")
		defer os.Remove(file)

		rw := NewReadWriter(file)
		require.NoError(t, rw.LoadOrGenerate())
		token, err := rw.Read()
		require.NoError(t, err)
		require.NotEmpty(t, token)

		stat, err := os.Stat(file)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())
	})

	t.Run("returns the file value if it exists", func(t *testing.T) {
		file, err := os.CreateTemp("", "identifier")
		require.NoError(t, err)
		_, err = file.WriteString("test")
		require.NoError(t, err)
		defer os.Remove(file.Name())
		stat, err := file.Stat()
		require.NoError(t, err)
		oldMtime := stat.ModTime()

		rw := NewReadWriter(file.Name())
		err = rw.LoadOrGenerate()
		require.NoError(t, err)
		token, err := rw.Read()
		require.NoError(t, err)
		require.Equal(t, "test", token)

		stat, err = os.Stat(file.Name())
		require.NoError(t, err)
		require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())
		require.Equal(t, oldMtime, stat.ModTime())
	})

	t.Run("sets the file mode to DefaultWorldReadableFileMode if exists", func(t *testing.T) {
		file, err := os.CreateTemp("", "identifier")
		require.NoError(t, err)
		_, err = file.WriteString("test")
		require.NoError(t, err)
		defer os.Remove(file.Name())

		err = file.Chmod(constant.DefaultFileMode)
		require.NoError(t, err)
		stat, err := file.Stat()
		require.NoError(t, err)
		require.Equal(t, os.FileMode(constant.DefaultFileMode), stat.Mode())

		rw := NewReadWriter(file.Name())
		err = rw.LoadOrGenerate()
		require.NoError(t, err)
		token, err := rw.Read()
		require.NoError(t, err)
		require.Equal(t, "test", token)

		stat, err = file.Stat()
		require.NoError(t, err)
		require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())
	})

	t.Run("errors for other reasons", func(t *testing.T) {
		file, err := os.CreateTemp("", "identifier")
		require.NoError(t, err)
		_, err = file.WriteString("test")
		require.NoError(t, err)
		require.NoError(t, file.Chmod(0x600))
		defer os.Remove(file.Name())

		rw := NewReadWriter(file.Name())
		token, err := rw.Read()
		require.Error(t, err)
		require.Empty(t, token)
	})
}

func TestRotate(t *testing.T) {
	file, err := os.CreateTemp("", t.Name())
	require.NoError(t, err)
	defer os.Remove(file.Name())
	rw := NewReadWriter(file.Name())

	token, err := rw.Read()
	require.NoError(t, err)
	require.Empty(t, token)

	err = rw.Rotate()
	require.NoError(t, err)
	token, err = rw.Read()
	require.NoError(t, err)
	require.NotEmpty(t, token)
	stat, err := file.Stat()
	require.NoError(t, err)
	require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())

	err = rw.Rotate()
	require.NoError(t, err)
	newToken, err := rw.Read()
	require.NoError(t, err)
	require.NotEmpty(t, newToken)
	require.NotEqual(t, token, newToken)
	stat, err = file.Stat()
	require.NoError(t, err)
	require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())
}
