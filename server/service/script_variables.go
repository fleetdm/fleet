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

// maybeExpandScriptFleetVariables resolves supported $FLEET_VAR_* references in
// contents for the given host. Resolved values are returned in fleetVars, keyed
// by their full FLEET_VAR_* name, for the agent to expose as environment
// variables at execution time; they are NOT substituted into the returned
// contents. The values are end-user-influenced (IdP data), so delivering them
// out-of-band lets the interpreter expand them without re-parsing the value
// (e.g. a department of "Engineering`id`" can never reach a shell as code).
//
// The reference token is left as $FLEET_VAR_NAME for POSIX shells, which expand
// it straight from the environment. For Windows hosts the token is rewritten to
// PowerShell's environment syntax $env:FLEET_VAR_NAME, since $FLEET_VAR_NAME
// there names a PowerShell variable rather than an environment variable.
//
// A non-empty failureMessage is returned when a variable exists but can't be
// resolved for this host (one line per failing variable). Unsupported variable
// names are left untouched: validation rejects them in new content, and content
// saved before validation shipped must keep working unchanged. Supported names
// that extend each other (e.g. ..._IDP_USERNAME and ..._IDP_USERNAME_LOCAL_PART)
// are handled correctly because variables.Find returns names longest-first, so
// each token is rewritten before its prefix is considered.
func (svc *Service) maybeExpandScriptFleetVariables(ctx context.Context, host *fleet.Host, contents string) (expanded string, fleetVars map[string]string, failureMessage string, err error) {
	found := variables.Find(contents)
	if len(found) == 0 {
		return contents, nil, "", nil
	}

	// defensive re-check in case variable-bearing content slipped past upload
	// validation (e.g. saved before validation shipped, or the license expired)
	if !license.IsPremium(ctx) {
		return "", nil, "Fleet couldn't run this script because it uses variables, which require a Fleet Premium license.", nil
	}

	// collect all failures instead of stopping at the first one so the admin
	// can fix everything in one pass
	var failures []string
	fail := func(errMsg string) error {
		failures = append(failures, errMsg)
		return nil
	}

	resolved := make(map[string]string)
	isWindows := fleet.IsWindowsPlatform(host.Platform)
	hostIDForUUIDCache := map[string]uint{host.UUID: host.ID}
	for _, v := range found {
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
				return "", nil, "", ctxerr.Wrap(ctx, err, "resolve IdP variable for script")
			}
			if !ok {
				// the fail callback recorded the reason
				continue
			}
			value = idpValue
		}

		// Deliver the value via the environment instead of splicing it into the
		// script body, so the interpreter expands it without re-parsing it.
		resolved["FLEET_VAR_"+v] = value
		if isWindows {
			// Rewrite to PowerShell's environment syntax. Keep the braced form
			// braced ($env: -> ${env:...}) so a reference adjacent to other
			// identifier characters, e.g. ${FLEET_VAR_HOST_UUID}_backup.log,
			// stays a single delimited token instead of swallowing the suffix.
			contents = strings.ReplaceAll(contents, "${FLEET_VAR_"+v+"}", "${env:FLEET_VAR_"+v+"}")
			contents = strings.ReplaceAll(contents, "$FLEET_VAR_"+v, "$env:FLEET_VAR_"+v)
		}
	}

	if len(failures) > 0 {
		return "", nil, strings.Join(failures, "\n"), nil
	}
	return contents, resolved, "", nil
}
