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
// before validation shipped must keep working unchanged.
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
