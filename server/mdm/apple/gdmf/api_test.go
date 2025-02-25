package gdmf

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetLatest(t *testing.T) {
	// test GetLatestOSVersion using a mock server that returns a known response
	// and ensure the response is parsed correctly

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// load the test data from the file
		b, err := os.ReadFile("./testdata/gdmf.json")
		require.NoError(t, err)
		_, err = w.Write(b)
		require.NoError(t, err)
	}))
	defer srv.Close()
	t.Setenv("FLEET_DEV_GDMF_URL", srv.URL)

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

	resp, err := GetLatestOSVersion(d)
	require.NoError(t, err)
	require.Equal(t, latestMacOSVersion, resp.ProductVersion)
	require.Equal(t, latestMacOSBuild, resp.Build)

	// NOTE: GetLatestOSVersion does not depend on the value of MDMCanRequestSoftwareUpdate. It is
	// expected that the caller has already verified this value before calling GetLatestOSVersion.

	tests := []struct {
		name            string
		machineInfo     fleet.MDMAppleMachineInfo
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := GetLatestOSVersion(tt.machineInfo)
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

func TestRetries(t *testing.T) {
	retryCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"error": "bad request"}`))
		require.NoError(t, err)
	}))
	os.Setenv("FLEET_DEV_GDMF_URL", srv.URL)
	t.Cleanup(func() {
		srv.Close()
		os.Unsetenv("FLEET_DEV_GDMF_URL")
	})

	latest, err := GetLatestOSVersion(fleet.MDMAppleMachineInfo{
		OSVersion:                "14.4.1",
		Product:                  "Mac15,7",
		Serial:                   "TESTSERIAL",
		SoftwareUpdateDeviceID:   "J516sAP",
		SupplementalBuildVersion: "23E224",
		UDID:                     uuid.New().String(),
		Version:                  "23E224",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "calling gdmf endpoint failed with status 400")
	require.Nil(t, latest)
	require.Equal(t, 4, retryCount)
}
