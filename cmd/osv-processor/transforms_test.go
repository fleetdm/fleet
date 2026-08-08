package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransformVuln(t *testing.T) {
	tests := []struct {
		name             string
		packageName      string
		cveID            string
		inputVuln        ProcessedVuln
		expectedPackages []string
		expectModified   bool
	}{
		{
			name:        "emacs maps to emacs, emacs-common, and emacs-el",
			packageName: "emacs",
			cveID:       "CVE-2024-39331",
			inputVuln: ProcessedVuln{
				CVE:        "CVE-2024-39331",
				Published:  "2024-07-01T00:00:00Z",
				Modified:   "2024-07-15T00:00:00Z",
				Fixed:      "1:26.3+1-1ubuntu2.1",
				Introduced: "0",
			},
			expectedPackages: []string{"emacs", "emacs-common", "emacs-el"},
			expectModified:   false,
		},
		{
			name:        "curl returns only curl (no transform)",
			packageName: "curl",
			cveID:       "CVE-2024-1234",
			inputVuln: ProcessedVuln{
				CVE:       "CVE-2024-1234",
				Published: "2024-01-01T00:00:00Z",
				Modified:  "2024-01-15T00:00:00Z",
			},
			expectedPackages: []string{"curl"},
			expectModified:   false,
		},
		{
			name:        "linux returns only linux (no transform)",
			packageName: "linux",
			cveID:       "CVE-2024-5678",
			inputVuln: ProcessedVuln{
				CVE:       "CVE-2024-5678",
				Published: "2024-03-01T00:00:00Z",
				Modified:  "2024-03-15T00:00:00Z",
			},
			expectedPackages: []string{"linux"},
			expectModified:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages, modifiedVuln := transformVuln(tt.packageName, tt.cveID, &tt.inputVuln)
			require.ElementsMatch(t, tt.expectedPackages, packages)

			if tt.expectModified {
				require.NotNil(t, modifiedVuln, "expected modified vulnerability")
			} else {
				require.Nil(t, modifiedVuln, "expected no modification")
			}
		})
	}
}

func TestAppendUbuntuBinaryPackages(t *testing.T) {
	affected := Affected{
		Package: Package{
			Ecosystem: "Ubuntu:26.04:LTS",
			Name:      "libssh2",
		},
		EcosystemSpecific: &EcosystemSpecific{
			Binaries: []UbuntuBinary{
				{
					BinaryName:    "libssh2-1t64",
					BinaryVersion: "1.11.1-1ubuntu0.26.04.2",
				},
			},
		},
	}

	packages := appendUbuntuBinaryPackages(
		[]string{"libssh2", "libssh2"},
		affected,
	)

	require.Equal(
		t,
		[]string{"libssh2", "libssh2-1t64"},
		packages,
	)
}

func TestAppendUbuntuBinaryPackagesWithNilEcosystemSpecific(t *testing.T) {
	affected := Affected{
		Package: Package{
			Name: "libssh2",
		},
	}

	packages := appendUbuntuBinaryPackages(
		[]string{"libssh2"},
		affected,
	)

	require.Equal(
		t,
		[]string{"libssh2"},
		packages,
	)
}

func TestAppendUbuntuBinaryPackagesDeduplicatesNames(t *testing.T) {
	affected := Affected{
		Package: Package{
			Name: "libssh2",
		},
		EcosystemSpecific: &EcosystemSpecific{
			Binaries: []UbuntuBinary{
				{BinaryName: "libssh2"},
				{BinaryName: "libssh2-1t64"},
				{BinaryName: "libssh2-1t64"},
				{BinaryName: ""},
			},
		},
	}

	packages := appendUbuntuBinaryPackages(
		[]string{"libssh2"},
		affected,
	)

	require.Equal(
		t,
		[]string{"libssh2", "libssh2-1t64"},
		packages,
	)
}
