package gdmf

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetLatest(t *testing.T) {
	// test GetLatestOSVersion using a mock server that returns a known response
	// and ensure the response is parsed correctly

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// load the test data from the file
		b, err := os.ReadFile("test_data.json")
		require.NoError(t, err)
		_, err = w.Write(b)
		require.NoError(t, err)
	}))
	os.Setenv("FLEET_DEV_GDMF_URL", srv.URL)
	t.Cleanup(func() {
		srv.Close()
		os.Unsetenv("FLEET_DEV_GDMF_URL")
	})

	// test the function
	d := apple_mdm.MachineInfo{
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
	// latestIOSVersion := "17.6.1"
	// latestIOSBuild := "21G93"

	resp, err := GetLatestOSVersion(d)
	require.NoError(t, err)
	require.Equal(t, latestMacOSVersion, resp.ProductVersion)
	require.Equal(t, latestMacOSBuild, resp.Build)

	// NOTE: GetLatestOSVersion does not depend on the value of MDMCanRequestSoftwareUpdate. It is
	// expected that the caller has already verified this value before calling GetLatestOSVersion.

	tests := []struct {
		name            string
		machineInfo     apple_mdm.MachineInfo
		expectedVersion string
		expectedBuild   string
		expectError     bool
	}{
		{
			name: "macOS matching software update device ID",
			machineInfo: apple_mdm.MachineInfo{
				OSVersion:                "14.4.1",
				Product:                  "Mac15,7", // macOS generally relies on the SoftwareUpdateDeviceID field and not the Product field
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "J516sAP",
				SupplementalBuildVersion: "23E224",
				UDID:                     uuid.New().String(),
				Version:                  "23E224",
			},
			expectedVersion: "14.6.1",
			expectedBuild:   "23G93",
			expectError:     false,
		},
		{
			name: "macOS non-matching software update device ID",
			machineInfo: apple_mdm.MachineInfo{
				OSVersion:                "14.4.1",
				Product:                  "Mac15,7",
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
			name: "non-matching product but matching software update device ID",
			machineInfo: apple_mdm.MachineInfo{
				OSVersion:                "14.4.1",
				Product:                  "INVALID",
				Serial:                   "TESTSERIAL",
				SoftwareUpdateDeviceID:   "J516sAP",
				SupplementalBuildVersion: "23E224",
				UDID:                     uuid.New().String(),
				Version:                  "23E224",
			},
			expectedVersion: "14.6.1",
			expectedBuild:   "23G93",
			expectError:     false,
		},
		{
			name: "non-matching product and software update device ID",
			machineInfo: apple_mdm.MachineInfo{
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
