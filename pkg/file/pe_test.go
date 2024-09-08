package file

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPEMetadata(t *testing.T) {
	file, err := os.Open("testdata/installers/hello-world-installer.exe")
	require.NoError(t, err)
	meta, err := ExtractPEMetadata(file)
	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, "Hello world", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, []string{"Hello world"}, meta.PackageIDs)
}
