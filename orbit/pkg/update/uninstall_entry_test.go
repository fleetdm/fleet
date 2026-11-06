package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFleetdUninstallEntry(t *testing.T) {
	for _, tc := range []struct {
		displayName, publisher string
		want                   bool
	}{
		{"Fleet", fleetdPublisher, true},
		{"Fleet osquery", fleetdPublisher, true},
		{"Fleet osquery", "", true},
		{"Fleet", "JetBrains s.r.o.", false},
		{"Fleet", "", false},
		{"fleet", fleetdPublisher, false},
		{"Fleet Desktop", fleetdPublisher, false},
		{"osquery", "osquery", false},
	} {
		assert.Equal(t, tc.want, isFleetdUninstallEntry(tc.displayName, tc.publisher), "%q by %q", tc.displayName, tc.publisher)
	}
}

func TestPackedProductCode(t *testing.T) {
	for _, tc := range []struct {
		code, want string
		ok         bool
	}{
		{"{AFA95216-3541-4281-BD30-8C16737ACBA1}", "61259AFA14531824DB03C86137A7BC1A", true},
		{"{afa95216-3541-4281-bd30-8c16737acba1}", "61259afa14531824db03c86137a7bc1a", true},
		{"Git_is1", "", false},
		{"AFA95216-3541-4281-BD30-8C16737ACBA1", "", false},
		{"{AFA95216-3541-4281-BD308-C16737ACBA1}", "", false},
		{"{AFA95216-3541-4281-BD30-8C16737ACBAZ}", "", false},
	} {
		got, ok := packedProductCode(tc.code)
		assert.Equal(t, tc.ok, ok, tc.code)
		assert.Equal(t, tc.want, got, tc.code)
	}
}
