//go:build linux

package sentinelone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCLICandidatePathsOrder asserts the .deb/.rpm install directory is tried
// first on Linux, ahead of the symlinks a package may or may not create.
func TestCLICandidatePathsOrder(t *testing.T) {
	assert.Equal(t, "/opt/sentinelone/bin/sentinelctl", cliCandidatePaths[0])
}
