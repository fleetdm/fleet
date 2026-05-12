package apple_mdm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/variables"
)


// ErrUnresolvableAppConfigVar signals that one of the $FLEET_VAR_* tokens
// referenced in a host's managed-app-configuration could not be resolved
// (for example, the host is not linked to an end-user IDP account).
// Callers should treat this as a non-retryable failure for the host —
// retrying without first changing host state will keep failing.
var ErrUnresolvableAppConfigVar = errors.New("apple_mdm: unresolvable Fleet variable in managed app configuration")

// AppConfigSubstitutionHost carries the host context needed to substitute the
// host-scoped $FLEET_VAR_* tokens supported in iOS / iPadOS managed app
// configuration. The validator (fleet.ValidateAppleAppConfiguration) restricts
// stored bytes to FleetVarsSupportedInAppleAppConfig, so this type only needs
// to cover that set.
type AppConfigSubstitutionHost struct {
	UUID           string
	HardwareSerial string
	Platform       string
}

// SubstituteFleetVarsInAppConfig replaces every supported $FLEET_VAR_* token
// in config with the resolved value for the given host, returning the
// substituted bytes. End-user IDP fields are looked up via ds; for that
// reason ds must be non-nil whenever the configuration references an IDP
// variable. Returns ErrUnresolvableAppConfigVar (wrapped) if the host can't
// supply a referenced variable — the caller is responsible for surfacing
// that to the user (typically by failing the install for that host).
func SubstituteFleetVarsInAppConfig(
	ctx context.Context,
	ds fleet.Datastore,
	config []byte,
	host AppConfigSubstitutionHost,
) ([]byte, error) {
	if len(config) == 0 {
		return config, nil
	}
	used := variables.Find(string(config))
	if len(used) == 0 {
		return config, nil
	}

	contents := string(config)
	idpUUIDCache := map[string]uint{}

	for _, name := range used {
		switch fleet.FleetVarName(name) {
		case fleet.FleetVarHostUUID:
			contents = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostUUIDRegexp, contents, host.UUID)
		case fleet.FleetVarHostHardwareSerial:
			contents = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostHardwareSerialRegexp, contents, host.HardwareSerial)
		case fleet.FleetVarHostPlatform:
			platform := host.Platform
			if platform == "darwin" {
				platform = "macos"
			}
			contents = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostPlatformRegexp, contents, platform)
		case fleet.FleetVarHostEndUserEmailIDP:
			emails, err := ds.GetHostEmails(ctx, host.UUID, fleet.DeviceMappingMDMIdpAccounts)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get host idp email for app config")
			}
			if len(emails) == 0 {
				return nil, ctxerr.Wrapf(ctx, ErrUnresolvableAppConfigVar, "$FLEET_VAR_%s", name)
			}
			contents = profiles.ReplaceFleetVariableInXML(fleetVarHostEndUserEmailIDPRegexp, contents, emails[0])
		case fleet.FleetVarHostEndUserIDPUsername,
			fleet.FleetVarHostEndUserIDPUsernameLocalPart,
			fleet.FleetVarHostEndUserIDPGroups,
			fleet.FleetVarHostEndUserIDPDepartment,
			fleet.FleetVarHostEndUserIDPFullname:
			replaced, ok, err := profiles.ReplaceHostEndUserIDPVariables(
				ctx, ds, name, contents, host.UUID, idpUUIDCache,
				// onError is only used by the profile processor to write a
				// per-host failure record; for app config we surface the
				// failure to the caller via ErrUnresolvableAppConfigVar.
				func(string) error { return nil },
			)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "substitute host idp variable in app config")
			}
			if !ok {
				return nil, ctxerr.Wrapf(ctx, ErrUnresolvableAppConfigVar, "$FLEET_VAR_%s", name)
			}
			contents = replaced
		default:
			// The validator restricts to FleetVarsSupportedInAppleAppConfig at
			// write time, so an unknown variable here means the validator and
			// this switch have drifted. Treat it as unresolvable rather than
			// silently leaving the literal token in the device-bound XML.
			return nil, ctxerr.Wrapf(ctx, ErrUnresolvableAppConfigVar, "$FLEET_VAR_%s", name)
		}
	}

	return []byte(contents), nil
}

// InstallApplicationParams carries the per-host inputs needed to build an
// `InstallApplication` MDM command plist for either a VPP or in-house
// (`.ipa`) Apple app.
//
// Configuration is the managed-app-configuration <dict>...</dict> bytes for
// the host. It is included only for iOS / iPadOS — macOS VPP installs always
// drop it. Empty or nil omits the `<key>Configuration>` entry, which Apple
// treats as "clear any managed config for this app on next apply."
type InstallApplicationParams struct {
	// CommandUUID is the MDM command UUID (== upcoming_activities.execution_id).
	CommandUUID string

	// HostPlatform is the host's OS family ("ios" / "ipados" / "darwin").
	// Determines ManagementFlags and whether Configuration is included.
	HostPlatform string

	// ITunesStoreID is the App Store / VPP adam id. Mutually exclusive with
	// ManifestURL.
	ITunesStoreID string

	// ManifestURL is the in-house `.ipa` manifest URL. Mutually exclusive with
	// ITunesStoreID.
	ManifestURL string

	// Configuration is the stored managed-app-configuration <dict>...</dict>
	// bytes. Caller is responsible for any per-host substitution before
	// passing it in.
	Configuration []byte
}

// BuildInstallApplicationCommand returns the XML plist for the given host's
// `InstallApplication` MDM command. Caller inserts the bytes directly into
// `nano_commands.command`.
//
// For iOS / iPadOS hosts, the `Configuration` dict is injected when params
// supplies non-empty configuration bytes. For macOS, configuration is always
// omitted regardless of the input — that's intentional and matches the
// service-layer silent-drop behavior.
func BuildInstallApplicationCommand(params InstallApplicationParams) []byte {
	var managementFlags int
	if fleet.IsAppleMobilePlatform(params.HostPlatform) {
		// Mobile: remove the app when MDM is removed.
		managementFlags = 1
	}
	// macOS keeps the app on MDM removal (flag 0).

	var b strings.Builder
	b.Grow(1024 + len(params.Configuration))

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>InstallAsManaged</key>
        <true/>
        <key>ManagementFlags</key>
        <integer>`)
	fmt.Fprintf(&b, "%d", managementFlags)
	b.WriteString(`</integer>
        <key>ChangeManagementState</key>
        <string>Managed</string>
        <key>Options</key>
        <dict>
            <key>PurchaseMethod</key>
            <integer>1</integer>
        </dict>
`)

	// Configuration is iOS/iPadOS-only. Strip any outer plist wrapper so we
	// inline only the bare <dict>...</dict>.
	if fleet.IsAppleMobilePlatform(params.HostPlatform) && len(params.Configuration) > 0 {
		b.WriteString("        <key>Configuration</key>\n        ")
		b.Write(stripPlistWrapper(params.Configuration))
		b.WriteString("\n")
	}

	b.WriteString(`        <key>RequestType</key>
        <string>InstallApplication</string>
`)
	switch {
	case params.ITunesStoreID != "":
		fmt.Fprintf(&b, "        <key>iTunesStoreID</key>\n        <integer>%s</integer>\n", params.ITunesStoreID)
	case params.ManifestURL != "":
		fmt.Fprintf(&b, "        <key>ManifestURL</key>\n        <string>%s</string>\n", params.ManifestURL)
	}

	b.WriteString(`    </dict>
    <key>CommandUUID</key>
    <string>`)
	b.WriteString(params.CommandUUID)
	b.WriteString(`</string>
</dict>
</plist>`)

	return []byte(b.String())
}

// stripPlistWrapper removes <?xml ...?>, <!DOCTYPE ...>, and <plist>...</plist>
// wrapping, returning just the bare <dict>...</dict>. No-op on bare fragments.
func stripPlistWrapper(b []byte) []byte {
	s := strings.TrimSpace(string(b))
	if strings.HasPrefix(s, "<?xml") {
		if idx := strings.Index(s, "?>"); idx >= 0 {
			s = strings.TrimSpace(s[idx+2:])
		}
	}
	if strings.HasPrefix(s, "<!DOCTYPE") {
		if idx := strings.Index(s, ">"); idx >= 0 {
			s = strings.TrimSpace(s[idx+1:])
		}
	}
	if strings.HasPrefix(s, "<plist") {
		if idx := strings.Index(s, ">"); idx >= 0 {
			s = strings.TrimSpace(s[idx+1:])
		}
		if strings.HasSuffix(s, "</plist>") {
			s = strings.TrimSpace(s[:len(s)-len("</plist>")])
		}
	}
	return []byte(s)
}
