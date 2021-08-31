package certificate

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPEM(t *testing.T) {
	t.Parallel()

	pool, err := LoadPEM(filepath.Join("testdata", "test.crt"))
	require.NoError(t, err)
	assert.True(t, len(pool.Subjects()) > 0)
}

func TestLoadErrorNoCertificates(t *testing.T) {
	t.Parallel()

	_, err := LoadPEM(filepath.Join("testdata", "empty.crt"))
	require.Error(t, err)
}

func TestLoadErrorMissingFile(t *testing.T) {
	t.Parallel()

	_, err := LoadPEM(filepath.Join("testdata", "invalid_path"))
	require.Error(t, err)
}
