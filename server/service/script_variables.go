package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// maybeExpandScriptFleetVariables resolves supported $FLEET_VAR_* references
// in contents for the given host. It returns the expanded contents, or a
// non-empty failureMessage when a variable exists but can't be resolved for
// this host (one line per failing variable). Unsupported variable names are
// left untouched: validation rejects them in new content, and content saved
// before validation shipped must keep working unchanged. Known limit of
// variables.Replace, accepted because validation rejects unsupported names
// going forward: in pre-validation content, an unsupported name that extends
// a supported one (e.g. $FLEET_VAR_HOST_UUID_SUFFIX) has its prefix replaced
// along with the supported variable. Supported names that extend each other
// (e.g. ..._IDP_USERNAME and ..._IDP_USERNAME_LOCAL_PART) are safe because
// variables.Find returns names longest-first and each is replaced in turn.
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

	// collect all failures instead of stopping at the first one so the admin
	// can fix everything in one pass
	var failures []string
	fail := func(errMsg string) error {
		failures = append(failures, errMsg)
		return nil
	}

	hostIDForUUIDCache := map[string]uint{host.UUID: host.ID}
	for _, v := range fleetVars {
		if !slices.Contains(fleet.FleetVarsSupportedInScripts, fleet.FleetVarName(v)) {
			continue
		}

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
			if value == "" {
				_ = fail(fmt.Sprintf("There is no platform for this host. Fleet couldn't populate $FLEET_VAR_%s.", v))
				continue
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

		contents = variables.Replace(contents, v, value)
	}

	if len(failures) > 0 {
		return "", strings.Join(failures, "\n"), nil
	}
	return contents, "", nil
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
func (svc *Service) expandNotificationURL(ctx context.Context, host *fleet.Host, script *fleet.HostScriptResult) (expanded string, failureMessage string) {
	notificationUUID, err := svc.notificationsSvc.NotificationUUIDForExecution(ctx, script.ExecutionID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to find the end user notification a script belongs to",
			"execution_id", script.ExecutionID, "err", err)
		return "", "Fleet couldn't find the notification this script belongs to."
	}

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
