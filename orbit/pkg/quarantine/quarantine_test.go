// +build windows

package quarantine

// TODO: implement tests for all items below:

// IsQuarantined():
// [x] True if "i am quarantined" file exists, false otherwise
// - Quarantined host has an "i am quarantined" file
// - Quarantined host has a file specifying all added firewall rules
// - Unquarantined host does not have this file
// - All rules are in the firewall for a quarantined host, and not duplicated
// - All rules are removed for a non-quarantined host
//
// markQuarantined(), markUnquarantined():
// - Test using isQuarantined
//
// Rule generation:
// - Test that these rules do not fail for all IPv4 and IPv6 layers
//   - Block IP addresses
//   - Allow fleet server
//   - Allow DNS
//   - Allow OSquery and others maybe
//
// Test communication:
// - External communication (disabled)
// - Fleet server communication (allowed)
// - DNS (allowed)
// - OSquery (allowed)
// - Other traffic that might be necessary

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestIsQuarantinedWhenQuarantined(t *testing.T) {
	// In a typical case, all tests should be run in parallel
	// but in this specific test the file creation would create
	// a race condition.
	// t.Parallel()

	filename := filepath.Join(".", "I_am_quarantined")
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	f.Close()
	defer os.Remove(filename)

	assert.True(t, isQuarantined(), "expected isQuarantined to return true")
}

func TestIsQuarantinedWhenNotQuarantined(t *testing.T) {
	// In a typical case, all tests should be run in parallel
	// but in this specific test the file creation would create
	// a race condition.
	// t.Parallel()

	// Removing the file just in case it exists
	filename := filepath.Join(".", "I_am_quarantined")
	os.Remove(filename)

	assert.False(t, isQuarantined(), "expected isQuarantined to return true")
}
