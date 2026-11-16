//go:build darwin

package sentinelone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCLICandidatePathsOrder asserts only the root-owned agent bundle is used
// on macOS, never a path under a possibly user-writable directory.
func TestCLICandidatePathsOrder(t *testing.T) {
	assert.Equal(t, []string{"/Library/Sentinel/sentinel-agent.bundle/Contents/MacOS/sentinelctl"}, cliCandidatePaths)
}
