package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// orbitConfigFieldTriggers declares, for every field of the orbit config, what
// tells connected agents to fetch it when it changes
// (websocket.orbit_config_enabled): a datastore hook, or nothing but the
// fallback poll, with why. Hosts in their first hour after enrolling poll at
// the default interval regardless (see orbitConfigOnboardingWindow).
var orbitConfigFieldTriggers = map[string]string{
	// fleet.OrbitConfig
	"script_execution_timeout":   "hook: orbitconfig_notify SaveAppConfig/SaveTeam/AddHostsToTeam",
	"command_line_startup_flags": "hook: orbitconfig_notify SaveAppConfig/SaveTeam/AddHostsToTeam",
	"extensions":                 "hook: orbitconfig_notify SaveAppConfig/SaveTeam/AddHostsToTeam; fallback-only for label membership changes",
	"nudge_config":               "fallback-only: macOS < 14 only, unsupported",
	"update_channels":            "hook: orbitconfig_notify SaveAppConfig/SaveTeam/AddHostsToTeam",
	"debug_logging":              "fallback-only: expiry is time-based",
	"websocket_transport":        "fallback-only: server config, changes on restart",
	"notifications":              "see the notifications fields",

	// fleet.OrbitConfigNotifications
	"renew_enrollment_profile":                    "hook: EnrollOsquery; fallback-only for host_mdm changes from osquery ingestion",
	"rotate_disk_encryption_key":                  "hook: SetHostsDiskEncryptionKeyStatus",
	"needs_mdm_migration":                         "hook: EnrollOsquery, orbitconfig_notify SaveAppConfig; fallback-only for host_mdm changes from osquery ingestion",
	"needs_programmatic_windows_mdm_enrollment":   "hook: EnrollOsquery, orbitconfig_notify SaveAppConfig; fallback-only for host_mdm changes from osquery ingestion",
	"windows_mdm_discovery_endpoint":              "data of needs_programmatic_windows_mdm_enrollment",
	"needs_programmatic_windows_mdm_unenrollment": "hook: orbitconfig_notify SaveAppConfig",
	"windows_mdm_sync_request":                    "hook: markMDMWindowsHasPendingCommandsByEnrollmentIDs",
	"pending_script_execution_ids":                "hook: activateNextUpcomingActivity",
	"enforce_bitlocker_encryption":                "hook: orbitconfig_notify SaveAppConfig/SaveTeam; fallback-only for disk status from osquery ingestion",
	"enable_bitlocker_protection":                 "fallback-only: disk status from osquery ingestion",
	"bitlocker_pin_request_pending":               "hook: QueueBitLockerPINRequest",
	"pending_software_installer_ids":              "hook: activateNextUpcomingActivity",
	"run_setup_experience":                        "hook: enqueueSetupExperienceItems, SetHostAwaitingConfiguration (macOS); onboarding window (Windows ESP)",
	"create_windows_managed_local_account":        "hook: initiateWindowsManagedLocalAccountRotation, orbitconfig_notify SaveAppConfig/SaveTeam",
	"run_disk_encryption_escrow":                  "hook: QueueEscrow",
}

// TestOrbitConfigFieldTriggers fails when an orbit config field has no
// declared trigger, so that a new field comes with a decision on how agents
// learn it changed.
func TestOrbitConfigFieldTriggers(t *testing.T) {
	fields := make(map[string]struct{})
	for _, typ := range []reflect.Type{reflect.TypeFor[fleet.OrbitConfig](), reflect.TypeFor[fleet.OrbitConfigNotifications]()} {
		for f := range typ.Fields() {
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			require.NotEmpty(t, name, "field %s.%s has no json name", typ.Name(), f.Name)
			fields[name] = struct{}{}
			require.Contains(t, orbitConfigFieldTriggers, name,
				"declare in orbitConfigFieldTriggers how connected agents learn that %s.%s changed", typ.Name(), f.Name)
		}
	}
	for name := range orbitConfigFieldTriggers {
		require.Contains(t, fields, name, "orbitConfigFieldTriggers declares a field that doesn't exist")
	}
}
