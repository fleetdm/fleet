package service

import (
	"context"
	"fmt"
	"regexp"
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
	found := slices.DeleteFunc(variables.Find(contents), func(v string) bool {
		return !slices.Contains(fleet.FleetVarsSupportedInScripts, fleet.FleetVarName(v))
	})
	if len(found) == 0 {
		return contents, nil, "", nil
	}

	// defensive re-check in case variable-bearing content slipped past upload
	// validation (e.g. saved before validation shipped, or the license expired)
	if !license.IsPremium(ctx) {
		return "", nil, "Fleet couldn't run this script because it uses variables, which require a Fleet Premium license.", nil
	}

	// A shebang naming a non-shell interpreter runs the file directly, so
	// nothing expands $FLEET_VAR_* from the script text.
	if kind, direct, err := fleet.ShebangInfo(contents); err == nil && direct && kind != fleet.ShebangShell {
		return "", nil, "Fleet couldn't run this script because it uses variables but isn't run by a shell. Read the FLEET_VAR_* environment variables from the script instead.", nil
	}

	// The platform decides how the tokens are delivered, so without it there is
	// no safe choice (a Windows host would read them as empty PowerShell variables).
	if host.Platform == "" {
		return "", nil, "There is no platform for this host. Fleet couldn't deliver variables to this script.", nil
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
				return "", nil, "", ctxerr.Wrap(ctx, err, "resolve IdP variable for script")
			}
			if !ok {
				// the fail callback recorded the reason
				continue
			}
			value = idpValue
		}

		// A NUL byte can't be represented in a process environment entry, so it
		// would fail the exec with an opaque error. Reject it here with a clear,
		// per-variable message instead, reusing the resolution-failure path.
		if strings.IndexByte(value, 0) >= 0 {
			_ = fail(fmt.Sprintf("The value for $FLEET_VAR_%s contains an invalid character and can't be used in a script.", v))
			continue
		}

		// Deliver the value via the environment instead of splicing it into the
		// script body, so the interpreter expands it without re-parsing it.
		resolved["FLEET_VAR_"+v] = value
	}

	if len(failures) > 0 {
		return "", nil, strings.Join(failures, "\n"), nil
	}
	if isWindows {
		contents = rewriteWindowsTokens(contents, resolved)
	}
	return contents, resolved, "", nil
}

// unbracedFleetVarToken matches $FLEET_VAR_NAME up to the end of the name, so
// a suffix such as .log stays outside the token and a longer unsupported name
// (e.g. $FLEET_VAR_HOST_UUID_OLD) is matched whole and left alone.
var unbracedFleetVarToken = regexp.MustCompile(`\$FLEET_VAR_\w+`)

// rewriteWindowsTokens turns each resolved reference into PowerShell's braced
// environment syntax, since $FLEET_VAR_NAME there names a PowerShell variable
// rather than an environment variable. The braces keep an adjacent suffix out
// of the variable name.
func rewriteWindowsTokens(contents string, resolved map[string]string) string {
	for name := range resolved {
		contents = strings.ReplaceAll(contents, "${"+name+"}", "${env:"+name+"}")
	}
	return unbracedFleetVarToken.ReplaceAllStringFunc(contents, func(token string) string {
		name := token[1:]
		if _, ok := resolved[name]; !ok {
			return token
		}
		return "${env:" + name + "}"
	})
}
