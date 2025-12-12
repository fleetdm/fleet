package swiftdialog

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeIconSize(t *testing.T) {
	// test logo-wide.png
	b, err := os.ReadFile("testdata/logo-wide.png")
	require.NoError(t, err)
	size, err := decodeIconSize(bytes.NewReader(b))
	require.NoError(t, err)
	require.Equal(t, uint(200), size)

	// test logo-narrow.png
	b, err = os.ReadFile("testdata/logo-narrow.png")
	require.NoError(t, err)
	size, err = decodeIconSize(bytes.NewReader(b))
	require.NoError(t, err)
	require.Equal(t, uint(80), size)

	// test logo-square.png
	b, err = os.ReadFile("testdata/logo-square.png")
	require.NoError(t, err)
	size, err = decodeIconSize(bytes.NewReader(b))
	require.NoError(t, err)
	require.Equal(t, uint(80), size)

	// test unsupported image format
	size, err = decodeIconSize(bytes.NewReader([]byte("invalid image format")))
	require.ErrorContains(t, err, "image: unknown format")
	require.Equal(t, uint(0), size)

	// test empty image
	size, err = decodeIconSize(bytes.NewReader([]byte{}))
	require.Error(t, err)
	require.Equal(t, uint(0), size)
}
