package fleet

import (
	"fmt"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostStatus(t *testing.T) {
	mockClock := clock.NewMockClock()

	testCases := []struct {
		seenTime            time.Time
		distributedInterval uint
		configTLSRefresh    uint
		status              HostStatus
	}{
		{mockClock.Now().Add(-30 * time.Second), 10, 3600, StatusOnline},
		{mockClock.Now().Add(-75 * time.Second), 10, 3600, StatusOffline},
		{mockClock.Now().Add(-30 * time.Second), 3600, 10, StatusOnline},
		{mockClock.Now().Add(-75 * time.Second), 3600, 10, StatusOffline},

		{mockClock.Now().Add(-60 * time.Second), 60, 60, StatusOnline},
		{mockClock.Now().Add(-121 * time.Second), 60, 60, StatusOffline},

		{mockClock.Now().Add(-1 * time.Second), 10, 10, StatusOnline},
		{mockClock.Now().Add(-2 * time.Minute), 10, 10, StatusOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 10, 10, StatusOffline}, // As of Fleet 4.15, StatusMIA is deprecated in favor of StatusOffline

		// Ensure behavior is reasonable if we don't have the values
		{mockClock.Now().Add(-1 * time.Second), 0, 0, StatusOnline},
		{mockClock.Now().Add(-2 * time.Minute), 0, 0, StatusOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 0, 0, StatusOffline}, // As of Fleet 4.15, StatusMIA is deprecated in favor of StatusOffline
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			// Save interval values
			h := Host{
				DistributedInterval: tt.distributedInterval,
				ConfigTLSRefresh:    tt.configTLSRefresh,
				SeenTime:            tt.seenTime,
			}

			assert.Equal(t, tt.status, h.Status(mockClock.Now()))
		})
	}
}

func TestHostStatusIsValid(t *testing.T) {
	for _, tt := range []struct {
		name     string
		status   HostStatus
		expected bool
	}{
		{"online", StatusOnline, true},
		{"offline", StatusOffline, true},
		{"new", StatusNew, true},
		{"missing", StatusMissing, true},
		{"mia", StatusMIA, true}, // As of Fleet 4.15, StatusMIA is deprecated in favor of StatusOffline
		{"empty", HostStatus(""), false},
		{"invalid", HostStatus("invalid"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestHostIsNew(t *testing.T) {
	mockClock := clock.NewMockClock()

	host := Host{}

	host.CreatedAt = mockClock.Now().AddDate(0, 0, -1)
	assert.True(t, host.IsNew(mockClock.Now()))

	host.CreatedAt = mockClock.Now().AddDate(0, 0, -2)
	assert.False(t, host.IsNew(mockClock.Now()))
}

func TestPlatformFromHost(t *testing.T) {
	for _, tc := range []struct {
		host        string
		expPlatform string
	}{
		{
			host:        "unknown",
			expPlatform: "",
		},
		{
			host:        "",
			expPlatform: "",
		},
		{
			host:        "linux",
			expPlatform: "linux",
		},
		{
			host:        "ubuntu",
			expPlatform: "linux",
		},
		{
			host:        "debian",
			expPlatform: "linux",
		},
		{
			host:        "rhel",
			expPlatform: "linux",
		},
		{
			host:        "centos",
			expPlatform: "linux",
		},
		{
			host:        "sles",
			expPlatform: "linux",
		},
		{
			host:        "kali",
			expPlatform: "linux",
		},
		{
			host:        "gentoo",
			expPlatform: "linux",
		},
		{
			host:        "tuxedo",
			expPlatform: "linux",
		},
		{
			host:        "darwin",
			expPlatform: "darwin",
		},
		{
			host:        "windows",
			expPlatform: "windows",
		},
	} {
		fleetPlatform := PlatformFromHost(tc.host)
		require.Equal(t, tc.expPlatform, fleetPlatform)

	}
}

func TestHostDisplayName(t *testing.T) {
	const (
		computerName   = "K0mpu73rN4M3"
		hostname       = "h0s7N4ME"
		hardwareModel  = "M0D3l"
		hardwareSerial = "53r14l"
	)
	for _, tc := range []struct {
		host     Host
		expected string
	}{
		{
			host:     Host{ComputerName: computerName, Hostname: hostname, HardwareModel: hardwareModel, HardwareSerial: hardwareSerial},
			expected: computerName, // If ComputerName is present, DisplayName is ComputerName
		},
		{
			host:     Host{ComputerName: "", Hostname: "h0s7N4ME", HardwareModel: "M0D3l", HardwareSerial: "53r14l"},
			expected: hostname, // If ComputerName is empty, DisplayName is Hostname (if present)
		},
		{
			host:     Host{ComputerName: "", Hostname: "", HardwareModel: "M0D3l", HardwareSerial: "53r14l"},
			expected: fmt.Sprintf("%s (%s)", hardwareModel, hardwareSerial), // If ComputerName and Hostname are empty, DisplayName is composite of HardwareModel and HardwareSerial (if both are present)
		},
		{
			host:     Host{ComputerName: "", Hostname: "", HardwareModel: "", HardwareSerial: hardwareSerial},
			expected: "", // If HarwareModel and/or HardwareSerial are empty, DisplayName is also empty
		},
		{
			host:     Host{ComputerName: "", Hostname: "", HardwareModel: hardwareModel, HardwareSerial: ""},
			expected: "", // If HarwareModel and/or HardwareSerial are empty, DisplayName is also empty
		},
		{
			host:     Host{ComputerName: "", Hostname: "", HardwareModel: "", HardwareSerial: ""},
			expected: "", // If HarwareModel and/or HardwareSerial are empty, DisplayName is also empty
		},
	} {
		require.Equal(t, tc.expected, tc.host.DisplayName())
	}
}

func TestMDMEnrollmentStatus(t *testing.T) {
	for _, tc := range []struct {
		hostMDM  HostMDM
		expected string
	}{
		{
			hostMDM:  HostMDM{Enrolled: true, InstalledFromDep: true},
			expected: "On (automatic)",
		},
		{
			hostMDM:  HostMDM{Enrolled: true, InstalledFromDep: false},
			expected: "On (manual)",
		},
		{
			hostMDM:  HostMDM{Enrolled: false, InstalledFromDep: true},
			expected: "Pending",
		},
		{
			hostMDM:  HostMDM{Enrolled: false, InstalledFromDep: false},
			expected: "Off",
		},
	} {
		require.Equal(t, tc.expected, tc.hostMDM.EnrollmentStatus())
	}
}

func TestIsEligibleForDEPMigration(t *testing.T) {
	testCases := []struct {
		name                    string
		osqueryHostID           *string
		depAssignedToFleet      *bool
		depProfileResponse      DEPAssignProfileResponseStatus
		enrolledInThirdPartyMDM bool
		expected                bool
		expectedManual          bool
		hostOS                  string
	}{
		{
			name:                    "Eligible for DEP migration",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseSuccess,
			enrolledInThirdPartyMDM: true,
			expected:                true,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - osqueryHostID nil",
			osqueryHostID:           nil,
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseSuccess,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - not DEP assigned to Fleet",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(false),
			depProfileResponse:      DEPAssignProfileResponseSuccess,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - not enrolled in third-party MDM",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseSuccess,
			enrolledInThirdPartyMDM: false,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - not DEP assigned and DEP profile failed",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(false),
			depProfileResponse:      DEPAssignProfileResponseNotAccessible,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          true,
			hostOS:                  "macOS 14.5",
		},
		{
			name:                    "Not eligible - DEP assigned and DEP profile failed",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseFailed,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - DEP assigned but not response yet",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      "",
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Not eligible - DEP assigned but not accessible",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(true),
			depProfileResponse:      DEPAssignProfileResponseNotAccessible,
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
		},
		{
			name:                    "Manual migration eligible - enrolled in 3rd party, but not DEP",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(false),
			depProfileResponse:      "",
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          true,
			hostOS:                  "macOS 14.5",
		},
		{
			name:                    "Manual migration ineligible - enrolled in 3rd party, not DEP, but OS version too low",
			osqueryHostID:           ptr.String("some-id"),
			depAssignedToFleet:      ptr.Bool(false),
			depProfileResponse:      "",
			enrolledInThirdPartyMDM: true,
			expected:                false,
			expectedManual:          false,
			hostOS:                  "macOS 13.9",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			host := &Host{
				OsqueryHostID:      tc.osqueryHostID,
				DEPAssignedToFleet: tc.depAssignedToFleet,
				OSVersion:          tc.hostOS,
			}

			mdmInfo := &HostMDM{
				Enrolled:               tc.enrolledInThirdPartyMDM,
				Name:                   "Some MDM",
				DEPProfileAssignStatus: ptr.String(string(tc.depProfileResponse)),
			}

			require.Equal(t, tc.expected, IsEligibleForDEPMigration(host, mdmInfo, false))
			manual, err := IsEligibleForManualMigration(host, mdmInfo, false)
			require.NoError(t, err)
			require.Equal(t, tc.expectedManual, manual)
		})
	}
}

func TestHasJSONProfileAssigned(t *testing.T) {
	testCases := []struct {
		name     string
		hostMDM  *HostMDM
		expected bool
	}{
		{
			name:     "nil HostMDM",
			hostMDM:  nil,
			expected: false,
		},
		{
			name: "nil DEPProfileAssignStatus",
			hostMDM: &HostMDM{
				DEPProfileAssignStatus: nil,
			},
			expected: false,
		},
		{
			name: "DEPProfileAssignStatus not successful",
			hostMDM: &HostMDM{
				DEPProfileAssignStatus: new(string),
			},
			expected: false,
		},
		{
			name: "DEPProfileAssignStatus successful",
			hostMDM: &HostMDM{
				DEPProfileAssignStatus: ptr.String(string(DEPAssignProfileResponseSuccess)),
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.hostMDM.HasJSONProfileAssigned()
			require.Equal(t, tc.expected, result)
		})
	}
}
