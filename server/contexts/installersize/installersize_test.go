package installersize

import (
	"context"
	"testing"

	"github.com/docker/go-units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		// SI units are shorter
		{"1 GB (SI shorter)", 1000000000, "1GB"},
		{"500 MB (SI shorter)", 500000000, "500MB"},
		{"1.5 GB (SI shorter)", 1500000000, "1.5GB"},

		// Binary units are shorter
		{"1 GiB (binary shorter)", 1073741824, "1GiB"},
		{"512 MiB (binary shorter)", 536870912, "512MiB"},
		{"1.5 GiB (binary shorter)", 1610612736, "1.5GiB"},
		{"1 TiB (binary shorter)", 1099511627776, "1TiB"},

		// Small values
		{"1000 bytes", 1000, "1kB"},
		{"1024 bytes", 1024, "1KiB"},

		// Default max installer size (10 GiB)
		{"default max (10 GiB)", DefaultMaxInstallerSize, "10GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Human(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultMaxInstallerSize(t *testing.T) {
	// Verify the default is 10 GiB
	assert.Equal(t, int64(10*units.GiB), DefaultMaxInstallerSize)
}

func TestDefaultMaxInstallerSizeStrMatchesInt(t *testing.T) {
	// Verify that the string constant parses to the same value as the int64 constant
	parsed, err := units.RAMInBytes(DefaultMaxInstallerSizeStr)
	require.NoError(t, err)
	assert.Equal(t, DefaultMaxInstallerSize, parsed)
}

func TestFromContextWithValue(t *testing.T) {
	ctx := context.Background()
	customSize := int64(5 * units.GiB)
	ctx = NewContext(ctx, customSize)

	result := FromContext(ctx)
	assert.Equal(t, customSize, result)
}

func TestFromContextWithoutValue(t *testing.T) {
	ctx := context.Background()

	result := FromContext(ctx)
	assert.Equal(t, DefaultMaxInstallerSize, result)
}

func TestNewContextOverwrite(t *testing.T) {
	ctx := context.Background()

	// Set first value
	ctx = NewContext(ctx, int64(5*units.GiB))
	assert.Equal(t, int64(5*units.GiB), FromContext(ctx))

	// Overwrite with second value
	ctx = NewContext(ctx, int64(20*units.GiB))
	assert.Equal(t, int64(20*units.GiB), FromContext(ctx))
}
