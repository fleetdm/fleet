package gdmf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// testUpdateAssets loads the GDMF fixture and converts it to the platform-keyed map of cached
// assets that GetLatestOSVersion takes. In production the same shape is produced by the OS updates
// cron: it fetches the public asset sets from GDMF, upserts them into
// apple_software_update_assets, and reads them back with Datastore.ListAppleOSUpdateAssets.
func testUpdateAssets(t *testing.T) map[string][]fleet.AppleSoftwareUpdateAsset {
	t.Helper()

	b, err := os.ReadFile("./testdata/gdmf.json")
	require.NoError(t, err)

	var am AssetMetadata
	require.NoError(t, json.Unmarshal(b, &am))

	convert := func(assets []fleet.OSUpdateAsset) []fleet.AppleSoftwareUpdateAsset {
		out := make([]fleet.AppleSoftwareUpdateAsset, 0, len(assets))
		for _, a := range assets {
			// the dates are stored in DATE columns, so they come back as time.Time
			postingDate, err := time.Parse(time.DateOnly, a.PostingDate)
			require.NoError(t, err)
			expirationDate, err := time.Parse(time.DateOnly, a.ExpirationDate)
			require.NoError(t, err)
			out = append(out, fleet.AppleSoftwareUpdateAsset{
				ProductVersion:   a.ProductVersion,
				Build:            a.Build,
				PostingDate:      postingDate,
				ExpirationDate:   expirationDate,
				SupportedDevices: a.SupportedDevices,
			})
		}
		return out
	}

	return map[string][]fleet.AppleSoftwareUpdateAsset{
		"macos": convert(am.PublicAssetSets.MacOS),
		"ios":   convert(am.PublicAssetSets.IOS),
	}
}

func TestGetLatest(t *testing.T) {
	// test GetLatestOSVersion against a known set of cached assets and ensure the latest matching
	// asset is returned for each device

	updateAssets := testUpdateAssets(t)

	// test the function
	d := fleet.MDMAppleMachineInfo{
		MDMCanRequestSoftwareUpdate: true,
		OSVersion:                   "14.4.1",
		Product:                     "Mac15,7",
		Serial:                      "TESTSERIAL",
		SoftwareUpdateDeviceID:      "J516sAP",
		SupplementalBuildVersion:    "23E224",
		UDID:                        uuid.New().String(),
		Version:                     "23E224",
	}

	latestMacOSVersion := "14.6.1"
	latestMacOSBuild := "23G93"
	latestIOSVersion := "17.6.1"
	latestIOSBuild := "21G93"

	resp, err := GetLatestOSVersion(d, updateAssets)
	require.NoError(t, err)
	require.Equal(t, latestMacOSVersion, resp.ProductVersion)
	require.Equal(t, latestMacOSBuild, resp.Build)

	// NOTE: GetLatestOSVersion does not depend on the value of MDMCanRequestSoftwareUpdate. It is
	// expected that the caller has already verified this value before calling GetLatestOSVersion.

	tests := []struct {
		name        string
		machineInfo fleet.MDMAppleMachineInfo
		// updateAssets defaults to the assets loaded from the fixture when nil
		updateAssets    map[string][]fleet.AppleSoftwareUpdateAsset
		expectedVersion string
		expectedBuild   string
		expectError     bool
	}{
		{
			name: "macOS matching software update device ID",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:                "14.4.1",
				Product:                  "Mac15,7",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "J516sAP",
				SupplementalBuildVersion: "23E224",
				UDID:                     uuid.New().String(),
				Version:                  "23E224",
			},
			expectedVersion: latestMacOSVersion,
			expectedBuild:   latestMacOSBuild,
			expectError:     false,
		},
		{
			// macOS generally relies on the SoftwareUpdateDeviceID field and not the Product field
			name: "macOS non-matching software update device ID",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:                "14.4.1",
				Product:                  "Mac15,7",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "INVALID",
				SupplementalBuildVersion: "23E224",
				UDID:                     uuid.New().String(),
				Version:                  "23E224",
			},
			expectedVersion: latestMacOSVersion,
			expectedBuild:   latestMacOSBuild,
			expectError:     true,
		},
		{
			// this should never happen in practice, but by default we still check macOS assets to
			// match the software update device ID
			name: "non-matching product but matching software update device ID",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:                "14.4.1",
				Product:                  "INVALID",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "J516sAP",
				SupplementalBuildVersion: "23E224",
				UDID:                     uuid.New().String(),
				Version:                  "23E224",
			},
			expectedVersion: latestMacOSVersion,
			expectedBuild:   latestMacOSBuild,
			expectError:     false,
		},
		{
			name: "non-matching product and software update device ID",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:                "14.4.1",
				Product:                  "INVALID",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "INVALID",
				SupplementalBuildVersion: "23E224",
				UDID:                     uuid.New().String(),
				Version:                  "23E224",
			},
			expectedVersion: "",
			expectedBuild:   "",
			expectError:     true,
		},
		{
			// missing other fields is not an error, this function always returns the latest
			// version and only depends on the Product and SoftwareUpdateDeviceID fields
			name: "missing other fields",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:              "",
				Product:                "Mac15,7",
				SoftwareUpdateDeviceID: "J516sAP",
			},
			expectedVersion: latestMacOSVersion,
			expectedBuild:   latestMacOSBuild,
			expectError:     false,
		},
		{
			name: "iphone matching product and software update device ID",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:                "17.5.1",
				Product:                  "iPhone14,6",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "iPhone14,6",
				SupplementalBuildVersion: "21F90",
				UDID:                     uuid.New().String(),
				Version:                  "21F90",
			},
			expectedVersion: latestIOSVersion,
			expectedBuild:   latestIOSBuild,
			expectError:     false,
		},
		{
			// iOS generally relies on the Product field and not the SoftwareUpdateDeviceID field so
			// this won't error even though the SoftwareUpdateDeviceID is invalid
			name: "iphone non-matching software update device ID",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:                "17.5.1",
				Product:                  "iPhone14,6",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "INVALID",
				SupplementalBuildVersion: "21F90",
				UDID:                     uuid.New().String(),
				Version:                  "21F90",
			},
			expectedVersion: latestIOSVersion,
			expectedBuild:   latestIOSBuild,
			expectError:     false,
		},
		{
			// this should never happen in practice, but we'll still try to match iOS assets if the
			// software update device ID starts with "iPhone" or "iPad"
			name: "missing product but valid iphone software update device ID",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:                "17.5.1",
				Product:                  "",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "iPhone14,6",
				SupplementalBuildVersion: "21F90",
				UDID:                     uuid.New().String(),
				Version:                  "21F90",
			},
			expectedVersion: latestIOSVersion,
			expectedBuild:   latestIOSBuild,
			expectError:     false,
		},
		{
			// we don't support other Apple products yet, so this should always error
			// because we we default to the macOS asset set and we won't find a matching asset there
			name: "unsupported product",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:                "8.8.1",
				Product:                  "Watch3,1",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "Watch3,1",
				SupplementalBuildVersion: "19U512",
				UDID:                     uuid.New().String(),
				Version:                  "19U512",
			},
			expectedVersion: "",
			expectedBuild:   "",
			expectError:     true,
		},
		{
			// the cached assets are empty until the OS updates cron has run at least once
			name: "no cached assets",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:              "14.4.1",
				Product:                "Mac15,7",
				SoftwareUpdateDeviceID: "J516sAP",
			},
			updateAssets: map[string][]fleet.AppleSoftwareUpdateAsset{},
			expectError:  true,
		},
		{
			// only the asset set for the device's platform matters, so a macOS device errors when
			// only iOS assets are cached
			name: "cached assets for the other platform only",
			machineInfo: fleet.MDMAppleMachineInfo{
				OSVersion:              "14.4.1",
				Product:                "Mac15,7",
				SoftwareUpdateDeviceID: "J516sAP",
			},
			updateAssets: map[string][]fleet.AppleSoftwareUpdateAsset{"ios": updateAssets["ios"]},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assets := tt.updateAssets
			if assets == nil {
				assets = updateAssets
			}
			resp, err := GetLatestOSVersion(tt.machineInfo, assets)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedVersion, resp.ProductVersion)
				require.Equal(t, tt.expectedBuild, resp.Build)
			}
		})
	}
}

func TestGetAssetMetadata(t *testing.T) {
	// test GetAssetMetadata using a mock server that returns a known response and ensure the
	// response is parsed correctly; this is the fetch the OS updates cron performs before caching
	// the assets in the datastore

	// load the test data from the file
	b, err := os.ReadFile("./testdata/gdmf.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(b); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	dev_mode.SetOverride("FLEET_DEV_GDMF_URL", srv.URL, t)

	am, err := GetAssetMetadata()
	require.NoError(t, err)
	require.NotEmpty(t, am.PublicAssetSets.MacOS)
	require.NotEmpty(t, am.PublicAssetSets.IOS)
	require.True(t, am.IsSupportedMacOSVersion("14.6.1", true))
	require.True(t, am.IsSupportedIOSVersion("17.6.1", "iPhone", true))
}

func TestRetries(t *testing.T) {
	retryCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"error": "bad request"}`))
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)
	dev_mode.SetOverride("FLEET_DEV_GDMF_URL", srv.URL, t)

	am, err := GetAssetMetadata()

	require.Error(t, err)
	require.ErrorContains(t, err, "calling gdmf endpoint failed with status 400")
	require.Nil(t, am)
	require.Equal(t, 4, retryCount)
}
