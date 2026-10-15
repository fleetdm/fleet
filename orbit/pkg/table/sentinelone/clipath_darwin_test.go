//go:build darwin

package sentinelone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCLICandidatePathsOrder asserts the installer-created symlink is tried
// first on macOS: it is the only path stable across agent versions.
func TestCLICandidatePathsOrder(t *testing.T) {
	assert.Equal(t, "/usr/local/bin/sentinelctl", cliCandidatePaths[0])
}
