package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/variables"
)

const (
	powerShellParamBlockMsg = "Fleet couldn't run this script because Fleet variables aren't supported inside a PowerShell param() block. Use the variable in the script body instead."
	unsupportedInterpMsg    = "Fleet couldn't run this script because its interpreter isn't supported."
	noPlatformMsg           = "There is no platform for this host. Fleet couldn't populate Fleet variables."
)

// maybeExpandScriptFleetVariables resolves supported $FLEET_VAR_* references in
// contents for the given host. Values are defined in a preamble, or escaped and
// substituted for Python, so a value is never parsed as script source. It
// returns the expanded contents, or a non-empty failureMessage when a variable
// can't be resolved for this host (one line per failing variable).
//
// Unsupported variable names are left untouched, since validation rejects them
// in new content and content saved before validation shipped must keep working.
// On the Python path that holds except for an unsupported name that extends a
// supported one (e.g. $FLEET_VAR_HOST_UUID_LEGACY), whose prefix
// variables.Replace rewrites along with the supported variable.
func (svc *Service) maybeExpandScriptFleetVariables(ctx context.Context, host *fleet.Host, contents string) (expanded string, failureMessage string, err error) {
	fleetVars := variables.Find(contents)
	if len(fleetVars) == 0 {
		return contents, "", nil
	}

	// defensive re-check in case variable-bearing content slipped past upload
	// validation (e.g. saved before validation shipped, or the license expired)
	if !license.IsPremium(ctx) {
		return "", "Fleet couldn't run this script because it uses variables, which require a Fleet Premium license.", nil
	}

	supported := make([]string, 0, len(fleetVars))
	for _, v := range fleetVars {
		if slices.Contains(fleet.FleetVarsSupportedInScripts, fleet.FleetVarName(v)) {
			supported = append(supported, v)
		}
	}
	if len(supported) == 0 {
		return contents, "", nil
	}

	dialect, dialectFailure := scriptFleetVarDialect(host, contents)
	if dialectFailure != "" {
		return "", dialectFailure, nil
	}

	// collect all failures instead of stopping at the first one so the admin
	// can fix everything in one pass
	var failures []string
	fail := func(errMsg string) error {
		failures = append(failures, errMsg)
		return nil
	}

	resolved := make(map[string]string, len(supported))
	hostIDForUUIDCache := map[string]uint{host.UUID: host.ID}
	for _, v := range supported {
		var value string
		switch fleet.FleetVarName(v) {
		case fleet.FleetVarHostUUID:
			value = host.UUID
			if value == "" {
				_ = fail(fmt.Sprintf("There is no UUID for this host. Fleet couldn't populate $FLEET_VAR_%s.", v))
				continue
			}
		case fleet.FleetVarHostHardwareSerial:
			value = host.HardwareSerial
			if value == "" {
				_ = fail(fmt.Sprintf("There is no hardware serial for this host. Fleet couldn't populate $FLEET_VAR_%s.", v))
				continue
			}
		case fleet.FleetVarHostPlatform:
			value = host.Platform
			if value == "darwin" {
				value = "macos"
			}
		default: // the IdP variables
			idpValue, _, ok, err := profiles.ResolveHostEndUserIDPValue(ctx, svc.ds, v, host.UUID, hostIDForUUIDCache, fail)
			if err != nil {
				return "", "", ctxerr.Wrap(ctx, err, "resolve IdP variable for script")
			}
			if !ok {
				// the fail callback recorded the reason
				continue
			}
			value = idpValue
		}

		// a NUL silently truncates the line for the interpreter
		if strings.ContainsRune(value, 0) {
			_ = fail(fmt.Sprintf("The value for $FLEET_VAR_%s contains an invalid character. Fleet couldn't populate it.", v))
			continue
		}
		resolved[v] = value
	}

	if len(failures) > 0 {
		return "", strings.Join(failures, "\n"), nil
	}
	if len(resolved) == 0 {
		return contents, "", nil
	}

	if dialect == variables.DialectPython {
		// supported is longest-first from variables.Find, which keeps a shorter
		// name from matching inside a longer token that contains it
		for _, name := range supported {
			value, ok := resolved[name]
			if !ok {
				continue
			}
			contents = variables.Replace(contents, name, variables.PythonEscape(value))
		}
		return contents, "", nil
	}

	expanded, err = variables.InsertPreamble(contents, variables.Preamble(resolved, dialect), dialect)
	switch {
	case errors.Is(err, variables.ErrPowerShellLeadingParamBlock):
		return "", powerShellParamBlockMsg, nil
	case err != nil:
		return "", "", ctxerr.Wrap(ctx, err, "insert fleet variable preamble")
	}
	return expanded, "", nil
}

// scriptFleetVarDialect returns the interpreter to write the preamble for, or a
// message explaining why variables can't be delivered to it. Platform decides
// first: on Windows fleetd runs the script through PowerShell whatever shebang
// it carries.
func scriptFleetVarDialect(host *fleet.Host, contents string) (variables.Dialect, string) {
	if host.Platform == "" {
		return 0, noPlatformMsg
	}
	if fleet.IsWindowsPlatform(host.Platform) {
		return variables.DialectPowerShell, ""
	}
	kind, _, err := fleet.ShebangInfo(contents)
	switch {
	case err != nil:
		return 0, unsupportedInterpMsg
	case kind == fleet.ShebangPython:
		return variables.DialectPython, ""
	}
	return variables.DialectPOSIX, ""
}

// isNotificationScript identifies an end user notification by the script Fleet
// queued for it. Its stored contents always hold the notification URL variable,
// since Fleet only ever expands that into the copy fleetd fetches, so this holds
// whatever the notification's execution_id points at by now. Both the fetch and
// the result path ask this, and they have to agree.
func isNotificationScript(script *fleet.HostScriptResult) bool {
	// admin-written scripts are never internal, so this skips them without a scan
	if !script.IsInternal {
		return false
	}
	return slices.Contains(variables.Find(script.ScriptContents), string(fleet.FleetVarPatchNotificationURL))
}

// expandNotificationURL resolves $FLEET_VAR_PATCH_NOTIFICATION_URL to the
// notification's device page URL. It resolves here rather than when the script
// is queued so script_contents never holds a live credential.
func (svc *Service) expandNotificationURL(ctx context.Context, host *fleet.Host, script *fleet.HostScriptResult, notificationUUID string) (expanded string, failureMessage string) {
	// orbit generates this token and sends it on check-in, so Fleet waits for one
	// rather than minting it here
	token, err := svc.ds.GetDeviceAuthTokenIfFresh(ctx, host.ID, hostDeviceAuthTokenTTL)
	switch {
	case err == nil:
		// OK
	case fleet.IsNotFound(err):
		svc.logger.InfoContext(ctx, "host has no fresh device auth token to notify against, waiting for it to send one",
			"host_id", host.ID)
		return "", "Fleet is waiting for this host to send a current authentication token."
	default:
		svc.logger.ErrorContext(ctx, "failed to check a host's device auth token", "host_id", host.ID, "err", err)
		return "", "Fleet couldn't check this host's authentication token."
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to load app config to build a notification url", "err", err)
		return "", "Fleet couldn't load its configuration to build the notification URL."
	}

	notificationURL := fmt.Sprintf("%s/device/%s/notifications/%s",
		strings.TrimRight(appConfig.ServerSettings.ServerURL, "/"), token, notificationUUID)

	return variables.Replace(script.ScriptContents, string(fleet.FleetVarPatchNotificationURL), notificationURL), ""
}
