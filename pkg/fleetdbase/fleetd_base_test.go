package fleetdbase

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBaseURL(t *testing.T) {
	t.Run("with env variable", func(t *testing.T) {
		t.Setenv("FLEET_DEV_DOWNLOAD_FLEETDM_URL", "https://download-testing.fleetdm.com")
		require.Equal(t, "https://download-testing.fleetdm.com", getBaseURL())
	})

	t.Run("without env variable", func(t *testing.T) {
		require.Equal(t, "https://download.fleetdm.com", getBaseURL())
	})
}

func TestGetMetadata(t *testing.T) {
	expectedMetadata := &Metadata{
		MSIURL:           "https://download-testing.fleetdm.com/archive/stable/2024-06-25_03-01-17/fleetd-base.msi",
		MSISha256:        "456e4f16c437c54d4cfacb54717450f4be582e572b8a7252a0384ac3118fbd11",
		PKGURL:           "https://download-testing.fleetdm.com/archive/stable/2024-06-25_03-01-17/fleetd-base.pkg",
		PKGSha256:        "4c914def2af5f4e0f5507e397d1d8af5b5991ea23cf606450787b8377e7bcecd",
		ManifestPlistURL: "https://download-testing.fleetdm.com/archive/stable/2024-06-25_03-01-17/fleetd-base-manifest.plist",
		Version:          "2024-06-25_03-01-17",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/stable/meta.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		require.NoError(t, json.NewEncoder(w).Encode(expectedMetadata))
	}))
	t.Cleanup(server.Close)
	t.Setenv("FLEET_DEV_DOWNLOAD_FLEETDM_URL", server.URL)

	meta, err := GetMetadata()
	require.NoError(t, err)
	require.Equal(t, expectedMetadata, meta)
}

func TestGetPKGManifestURL(t *testing.T) {
	t.Run("with env variable", func(t *testing.T) {
		t.Setenv("FLEET_DEV_DOWNLOAD_FLEETDM_URL", "https://download-test.fleetdm.com")
		require.Equal(t, "https://download-test.fleetdm.com/stable/fleetd-base-manifest.plist", GetPKGManifestURL())
	})

	t.Run("without env variable", func(t *testing.T) {
		require.Equal(t, "https://download.fleetdm.com/stable/fleetd-base-manifest.plist", GetPKGManifestURL())
	})
}
