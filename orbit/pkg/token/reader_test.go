package token

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) *os.File {
	t.Helper()
	file, err := os.CreateTemp("", "")
	require.NoError(t, err)

	_, err = file.Write([]byte("test"))
	require.NoError(t, err)

	t.Cleanup(func() { os.Remove(file.Name()) })
	return file
}

func TestTokenRead(t *testing.T) {
	t.Run("returns an error if can't be found", func(t *testing.T) {
		tr := Reader{Path: "does-not-exist"}
		token, err := tr.Read()
		require.Error(t, err)
		require.Empty(t, token)
	})

	t.Run("reads the token from disk if the cached value is empty", func(t *testing.T) {
		tokenFile := setup(t)
		tr := Reader{Path: tokenFile.Name()}
		token, err := tr.Read()
		require.NoError(t, err)
		require.Equal(t, "test", token)
	})

	t.Run("only reads the file again when the mtime changes", func(t *testing.T) {
		tokenFile := setup(t)
		tr := Reader{Path: tokenFile.Name()}
		token, err := tr.Read()
		require.NoError(t, err)
		require.Equal(t, "test", token)

		// change the value of the token, but set the mtime to the same value
		stat, err := tokenFile.Stat()
		require.NoError(t, err)
		oldmtime := stat.ModTime()
		_, err = tokenFile.WriteAt([]byte("new-value"), 0)
		require.NoError(t, err)
		err = os.Chtimes(tokenFile.Name(), oldmtime, oldmtime)
		require.NoError(t, err)

		// the token should still be the same
		token, err = tr.Read()
		require.NoError(t, err)
		require.Equal(t, "test", token)

		// the token should be updated when the mtime changes
		err = os.Chtimes(tokenFile.Name(), oldmtime, oldmtime.Add(time.Second))
		require.NoError(t, err)
		token, err = tr.Read()
		require.NoError(t, err)
		require.Equal(t, "new-value", token)
	})
}

func TestTokenHasChanged(t *testing.T) {
	tokenFile := setup(t)
	tr := Reader{Path: tokenFile.Name()}
	// perform an initial read
	token, err := tr.Read()
	require.NoError(t, err)
	require.Equal(t, "test", token)

	// token has not changed
	changed, err := tr.HasChanged()
	require.NoError(t, err)
	require.False(t, changed)

	// change the value of the token
	err = os.Chtimes(tokenFile.Name(), time.Now(), time.Now())
	require.NoError(t, err)

	changed, err = tr.HasChanged()
	require.NoError(t, err)
	require.True(t, changed)
}

func TestTokenHasExpired(t *testing.T) {
	tokenFile := setup(t)
	tr := Reader{Path: tokenFile.Name()}
	// perform an initial read
	_, err := tr.Read()
	require.NoError(t, err)
	exp, remain := tr.HasExpired()
	require.False(t, exp)
	require.Greater(t, remain, time.Minute)

	// change the mtime of the file to an expired value
	oldmtime := time.Now().Add(-3 * time.Hour)
	err = os.Chtimes(tokenFile.Name(), oldmtime, oldmtime)
	require.NoError(t, err)

	_, err = tr.Read()
	require.NoError(t, err)
	exp, remain = tr.HasExpired()
	require.True(t, exp)
	require.Zero(t, remain)
}

func TestGetCached(t *testing.T) {
	tokenFile := setup(t)
	tr := Reader{Path: tokenFile.Name()}
	// perform an initial read
	token, err := tr.Read()
	require.NoError(t, err)
	require.Equal(t, "test", token)

	// change the value of the token
	_, err = tokenFile.WriteAt([]byte("new-value"), 0)
	require.NoError(t, err)

	// function should return the cached value
	cached := tr.GetCached()
	require.Equal(t, "test", cached)
}
