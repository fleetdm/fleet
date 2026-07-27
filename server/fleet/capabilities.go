package fleet

import (
	"os"
	"runtime"
	"strings"
	"sync"
)

// Capability represents a concrete feature of Fleet.
type Capability string

// CapabilityMap is an utility type to represent a set of capabilities.
type CapabilityMap map[Capability]struct{}

// mu is used to allow for safe access to the capability map.
var mu sync.Mutex

// PopulateFromString populates the CapabilityMap from a comma-separated string.
// Example: "foo,bar,baz" => {"foo": struct{}, "bar": struct{}, "baz": struct{}}
func (c *CapabilityMap) PopulateFromString(s string) {
	mu.Lock()
	defer mu.Unlock()
	*c = make(CapabilityMap)

	if s == "" {
		return
	}

	for _, capability := range strings.Split(s, ",") {
		(*c)[Capability(capability)] = struct{}{}
	}
}

// String returns a comma-separated string with the capabilities in the map.
// Example: {"foo": struct{}, "bar": struct{}, "baz": struct{}} => "foo,bar,baz"
func (c *CapabilityMap) String() string {
	mu.Lock()
	defer mu.Unlock()
	idx := 0
	capabilities := make([]string, len(*c))
	for capability := range *c {
		capabilities[idx] = string(capability)
		idx++
	}
	return strings.Join(capabilities, ",")
}

// Has returns true if the CapabilityMap contains the given capability.
func (c CapabilityMap) Has(capability Capability) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := c[capability]
	return ok
}

// Copy copies the capabilities from another map to this map.
func (c CapabilityMap) Copy(from CapabilityMap) {
	mu.Lock()
	defer mu.Unlock()
	for capability := range from {
		c[capability] = struct{}{}
	}
}

// The following are the capabilities that Fleet supports. These can be used by
// the Fleet server, Orbit or Fleet Desktop to communicate that a given feature
// is supported.
const (
	// CapabilityOrbitEndpoints denotes the presence of server endpoints
	// dedicated to communicating with Orbit. These endpoints start with
	// `/api/fleet/orbit`, and allow enrolling a host through Orbit among other
	// functionality.
	CapabilityOrbitEndpoints Capability = "orbit_endpoints"
	// CapabilityTokenRotation denotes the ability of the server to support
	// periodic rotation of device tokens
	CapabilityTokenRotation  Capability = "token_rotation"
	CapabilityErrorReporting Capability = "error_reporting"
	// CapabilityEndUserEmail denotes the ability of the server to support
	// receiving the end-user email from orbit.
	CapabilityEndUserEmail Capability = "end_user_email"
	// CapabilityEscrowBuddy allows to use Escrow Buddy to rotate FileVault keys
	CapabilityEscrowBuddy Capability = "escrow_buddy"
	// CapabilityLinuxDiskEncryptionEscrow denotes the ability of the server to escrow Ubuntu and Fedora disk
	// encryption LUKS passphrases
	CapabilityLinuxDiskEncryptionEscrow Capability = "linux_disk_encryption_escrow"
	// CapabilityLUKSRecoveryKeyEscrow denotes the ability of the server to
	// accept an OrbitPostLUKSRequest whose KeyType is LUKSKeyTypeRecoveryKey
	// (no Salt, no numeric KeySlot). Servers without this capability reject
	// the payload with a "passphrase, salt, and key_slot must be provided"
	// error, so orbit must gate the snapd/TPM-backed FDE escrow path on it
	// to avoid churning the fleet-escrow key slot on every retry.
	CapabilityLUKSRecoveryKeyEscrow Capability = "luks_recovery_key_escrow"
	// CapabilitySetupExperience denotes the ability of the server to support
	// installing software and running a script during macOS ADE enrollment, and
	// the ability of the client to show the corresponding UI to support that
	// flow.
	CapabilitySetupExperience Capability = "setup_experience"
	// CapabilityWebSetupExperience denotes the ability of the server to support installing software
	// as part of a non-blocking setup experience for Linux and Windows
	CapabilityWebSetupExperience Capability = "web_setup_experience"
	// CapabilityMacOSWebSetupExperience denotes the ability of the server to support
	// a web-based setup experience UI for macOS devices
	CapabilityMacOSWebSetupExperience Capability = "macos_web_setup_experience"
	// CapabilityEndUserAuth denotes the ability of the client to authenticate
	// the end user against the Fleet server (e.g. SSO) before enrolling
	CapabilityEndUserAuth Capability = "end_user_auth"
	// CapabilityWindowsMDMSync denotes the ability of Windows fleetd to start an on-demand OMA-DM session (via deviceenroller) when the server
	// signals that the host has queued Windows MDM commands. This lets the server relax the aggressive Windows MDM poll while keeping command
	// latency low.
	CapabilityWindowsMDMSync Capability = "windows_mdm_sync"
)

func GetServerOrbitCapabilities() CapabilityMap {
	return CapabilityMap{
		CapabilityOrbitEndpoints:            {},
		CapabilityTokenRotation:             {},
		CapabilityEndUserEmail:              {},
		CapabilityEscrowBuddy:               {},
		CapabilityLinuxDiskEncryptionEscrow: {},
		CapabilityLUKSRecoveryKeyEscrow:     {},
		CapabilitySetupExperience:           {},
		CapabilityWebSetupExperience:        {},
		CapabilityMacOSWebSetupExperience:   {},
	}
}

func GetServerDeviceCapabilities() CapabilityMap {
	capabilities := CapabilityMap{}
	if os.Getenv("FLEET_ENABLE_POST_CLIENT_DEBUG_ERRORS") == "1" {
		capabilities[CapabilityErrorReporting] = struct{}{}
	}
	return capabilities
}

func GetOrbitClientCapabilities() CapabilityMap {
	capabilities := CapabilityMap{
		CapabilityEscrowBuddy:     {},
		CapabilitySetupExperience: {},
	}
	// On non-macOS systems, include end user auth capability.
	if runtime.GOOS != "darwin" {
		capabilities[CapabilityEndUserAuth] = struct{}{}
	}
	// Windows fleetd can start an on-demand OMA-DM session (windowsMDMSyncConfigReceiver) when the server signals queued MDM commands.
	if runtime.GOOS == "windows" {
		capabilities[CapabilityWindowsMDMSync] = struct{}{}
	}
	return capabilities
}

// CapabilitiesHeader is the header name used to communicate the capabilities.
const CapabilitiesHeader = "X-Fleet-Capabilities"
