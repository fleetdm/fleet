package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/stretchr/testify/require"
)

func TestGetCertPath(t *testing.T) {
	validRoot := t.TempDir()
	invalidRoot := t.TempDir()

	srcCertPath := filepath.Join("..", "..", "pkg", "cryptoinfo", "testdata", "test_crt.pem")
	srcCert, err := os.ReadFile(srcCertPath)
	require.NoError(t, err)

	validCertPath := filepath.Join(validRoot, "certs.pem")
	require.NoError(t, os.WriteFile(validCertPath, srcCert, 0644))

	invalidCertPath := filepath.Join(invalidRoot, "invalid_cert.pem")
	require.NoError(t, os.WriteFile(invalidCertPath, []byte(`INVALID_CERT_CONTENT`), 0644))

	cases := []struct {
		name         string
		rootDir      string
		fleetCert    string
		expectedPath string
		expectError  error
	}{
		{
			name:         "Default cert path exists",
			rootDir:      validRoot,
			fleetCert:    "",
			expectedPath: validCertPath,
		},
		{
			name:         "Provided cert path exists",
			rootDir:      validRoot,
			fleetCert:    srcCertPath,
			expectedPath: srcCertPath,
		},
		{
			name:         "Default cert does not exist",
			rootDir:      invalidRoot,
			fleetCert:    "",
			expectedPath: "",
			expectError:  fmt.Errorf("cert not found at %s", filepath.Join(invalidRoot, "certs.pem")),
		},
		{
			name:         "Invalid cert path provided",
			rootDir:      "",
			fleetCert:    filepath.Join(validRoot, "blah.pem"),
			expectedPath: "",
			expectError:  fmt.Errorf("cert not found at %s", filepath.Join(validRoot, "blah.pem")),
		},
		{
			name:         "Invalid PEM format",
			rootDir:      "",
			fleetCert:    invalidCertPath,
			expectedPath: "",
			expectError:  fmt.Errorf("invalid PEM format %s: no valid certificates found in %s", invalidCertPath, invalidCertPath),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			path, err := getCertPath(tt.rootDir, tt.fleetCert)

			if tt.expectError != nil {
				require.Error(t, err)
				require.Empty(t, path)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedPath, path)
			}
		})
	}
}

func TestGetUpdater(t *testing.T) {
	cases := []struct {
		name           string
		disableUpdates bool
		expectDisabled bool
	}{
		{"updates enabled", false, false},
		{"updates disabled", true, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			updater, err := getUpdater(c.disableUpdates, update.Options{})
			// A 'disabled' updater should never fail, even with invalid options.
			if c.expectDisabled {
				require.NoError(t, err)
				require.NotNil(t, updater)
			} else {
				require.Error(t, err)
				require.Nil(t, updater)
			}
		})
	}
}
