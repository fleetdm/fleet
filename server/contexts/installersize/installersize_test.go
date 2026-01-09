package installersize

import (
	"context"
	"testing"

	"github.com/docker/go-units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
