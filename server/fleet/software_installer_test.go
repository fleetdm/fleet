package fleet

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTempFileReader(t *testing.T) {
	content1And2 := "Hello, World!"
	tfr1, err := NewTempFileReader(strings.NewReader(content1And2), t.TempDir)
	require.NoError(t, err)
	tfr2, err := NewTempFileReader(strings.NewReader(content1And2), t.TempDir)
	require.NoError(t, err)

	content3 := "Hello, Temp!"
	keepFile, err := os.CreateTemp(t.TempDir(), "test")
	require.NoError(t, err)
	_, err = io.Copy(keepFile, strings.NewReader(content3))
	require.NoError(t, err)
	err = keepFile.Close()
	require.NoError(t, err)
	tfr3, err := NewKeepFileReader(keepFile.Name())
	require.NoError(t, err)

	b, err := io.ReadAll(tfr1)
	require.NoError(t, err)
	require.Equal(t, content1And2, string(b))
	b, err = io.ReadAll(tfr2)
	require.NoError(t, err)
	require.Equal(t, content1And2, string(b))

	// rewind and read again gets the same content
	err = tfr1.Rewind()
	require.NoError(t, err)
	b, err = io.ReadAll(tfr1)
	require.NoError(t, err)
	require.Equal(t, content1And2, string(b))

	// tfr2 is at EOF, so it reads nothing
	b, err = io.ReadAll(tfr2)
	require.NoError(t, err)
	require.Equal(t, "", string(b))

	b, err = io.ReadAll(tfr3)
	require.NoError(t, err)
	require.Equal(t, content3, string(b))

	// closing deletes the file
	err = tfr1.Close()
	require.NoError(t, err)
	_, err = os.Stat(tfr1.Name())
	require.True(t, os.IsNotExist(err))

	// tfr2 still exists
	_, err = os.Stat(tfr2.Name())
	require.False(t, os.IsNotExist(err))

	// tfr3 still exists even after Close
	err = tfr3.Close()
	require.NoError(t, err)
	_, err = os.Stat(tfr3.Name())
	require.False(t, os.IsNotExist(err))
}
