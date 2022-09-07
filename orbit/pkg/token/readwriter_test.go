package token

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadOrGenerate(t *testing.T) {
	t.Run("creates file if it doesn't exist", func(t *testing.T) {
		dir := os.TempDir()
		file := filepath.Join(dir, "identifier")
		defer os.Remove(file)

		rw := NewReadWriter(file)
		rw.LoadOrGenerate()
		token, err := rw.Read()
		require.NoError(t, err)
		require.NotEmpty(t, token)
	})

	t.Run("returns the file value if it exists", func(t *testing.T) {
		file, err := os.CreateTemp("", "identifier")
		require.NoError(t, err)
		_, err = file.WriteString("test")
		require.NoError(t, err)
		defer os.Remove(file.Name())

		rw := NewReadWriter(file.Name())
		rw.LoadOrGenerate()
		token, err := rw.Read()
		require.NoError(t, err)
		require.Equal(t, "test", token)
	})

	t.Run("errors for other reasons", func(t *testing.T) {
		file, err := os.CreateTemp("", "identifier")
		require.NoError(t, err)
		_, err = file.WriteString("test")
		require.NoError(t, err)
		file.Chmod(0x600)
		defer os.Remove(file.Name())

		rw := NewReadWriter(file.Name())
		rw.LoadOrGenerate()
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

	rw.Rotate()
	token, err = rw.Read()
	require.NoError(t, err)
	require.NotEmpty(t, token)

	rw.Rotate()
	newToken, err := rw.Read()
	require.NoError(t, err)
	require.NotEmpty(t, newToken)
	require.NotEqual(t, token, newToken)
}
