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
//
// Substitute returns a *UnresolvableAppConfigVarError wrapping this sentinel,
// so callers can:
//   - check with errors.Is(err, ErrUnresolvableAppConfigVar), and
//   - extract the user-facing detail with errors.As(err, &typed) → typed.Detail.
var ErrUnresolvableAppConfigVar = errors.New("apple_mdm: unresolvable Fleet variable in managed app configuration")

// UnresolvableAppConfigVarError carries the unresolved variable name and a
// user-facing detail message. Detail matches the wording configuration-profile
// delivery uses for the same variable so admins see consistent copy across
// surfaces.
type UnresolvableAppConfigVarError struct {
	// FleetVar is the variable name without the $FLEET_VAR_ prefix
	// (e.g. "HOST_END_USER_IDP_DEPARTMENT").
	FleetVar string
	// Detail is the user-facing reason. May be empty for variables that don't
	// have a specific message (the generic sentinel applies).
	Detail string
}

func (e *UnresolvableAppConfigVarError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return fmt.Sprintf("apple_mdm: unresolvable Fleet variable $FLEET_VAR_%s", e.FleetVar)
}

// Is lets `errors.Is(err, ErrUnresolvableAppConfigVar)` keep working for
// callers that just want the sentinel check.
func (e *UnresolvableAppConfigVarError) Is(target error) bool {
	return target == ErrUnresolvableAppConfigVar
}

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
				return nil, &UnresolvableAppConfigVarError{
					FleetVar: name,
					Detail:   fleet.HostEndUserEmailIDPVariableReplacementFailedError,
				}
			}
			contents = profiles.ReplaceFleetVariableInXML(fleetVarHostEndUserEmailIDPRegexp, contents, emails[0])
		case fleet.FleetVarHostEndUserIDPUsername,
			fleet.FleetVarHostEndUserIDPUsernameLocalPart,
			fleet.FleetVarHostEndUserIDPGroups,
			fleet.FleetVarHostEndUserIDPDepartment,
			fleet.FleetVarHostEndUserIDPFullname:
			// onError in the profile processor writes a per-host failure
			// record; here we just capture the per-variable detail string so
			// we can surface it through UnresolvableAppConfigVarError.
			var detail string
			replaced, ok, err := profiles.ReplaceHostEndUserIDPVariables(
				ctx, ds, name, contents, host.UUID, idpUUIDCache,
				func(d string) error { detail = d; return nil },
			)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "substitute host idp variable in app config")
			}
			if !ok {
				return nil, &UnresolvableAppConfigVarError{FleetVar: name, Detail: detail}
			}
			contents = replaced
		default:
			// The validator restricts to FleetVarsSupportedInAppleAppConfig at
			// write time, so an unknown variable here means the validator and
			// this switch have drifted. Treat it as unresolvable rather than
			// silently leaving the literal token in the device-bound XML.
			return nil, &UnresolvableAppConfigVarError{FleetVar: name}
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

	// IsUserEnrollment indicates this command targets an Account-Driven User
	// Enrolled (BYOD) host. When true, ChangeManagementState is omitted because
	// Apple rejects it on the User Enrollment channel.
	IsUserEnrollment bool
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
`)
	// Apple rejects ChangeManagementState on the Account-Driven User Enrollment
	// channel. Omit it for User Enrollment; include it everywhere else.
	if !params.IsUserEnrollment {
		b.WriteString(`        <key>ChangeManagementState</key>
        <string>Managed</string>
`)
	}
	b.WriteString(`        <key>Options</key>
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
