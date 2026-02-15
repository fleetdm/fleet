package nvd

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestFmtStrEpochStripping(t *testing.T) {
	item := &IndexedCPEItem{
		Vendor:  "gnu",
		Product: "emacs",
	}

	testCases := []struct {
		name            string
		software        fleet.Software
		expectedVersion string // the version segment in the CPE string
	}{
		{
			name: "deb package with epoch prefix strips epoch",
			software: fleet.Software{
				Name:    "emacs-common",
				Version: "1:29.3+1-1ubuntu2",
				Source:  "deb_packages",
			},
			expectedVersion: "29.3.1-1ubuntu2",
		},
		{
			name: "rpm package with epoch strips epoch",
			software: fleet.Software{
				Name:    "some-package",
				Version: "2:3.28.0-4.el9",
				Source:  "rpm_packages",
			},
			expectedVersion: "3.28.0-4.el9",
		},
		{
			name: "deb package without epoch unchanged",
			software: fleet.Software{
				Name:    "some-package",
				Version: "29.3+1-1ubuntu2",
				Source:  "deb_packages",
			},
			expectedVersion: "29.3.1-1ubuntu2",
		},
		{
			name: "non-deb source does not strip epoch-like prefix",
			software: fleet.Software{
				Name:    "some-app",
				Version: "1:2.3.4",
				Source:  "programs",
			},
			expectedVersion: "1.2.3.4",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := item.FmtStr(&tc.software)
			assert.Contains(t, result, ":"+tc.expectedVersion+":")
		})
	}
}
