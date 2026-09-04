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
	pythonFleetVarsMsg = "Fleet couldn't run this script because Fleet variables aren't supported in Python scripts. Use a shell script, or remove the variable."
	// PowerShell requires a param() block to be the first statement.
	powerShellParamBlockMsg = "Fleet couldn't run this script because Fleet variables aren't supported in a PowerShell script that starts with a param() block. Move the param() block, or remove the variable."
	unsupportedInterpMsg    = "Fleet couldn't run this script because its interpreter isn't supported."
	noPlatformMsg           = "There is no platform for this host. Fleet couldn't populate Fleet variables."
)

// maybeExpandScriptFleetVariables resolves supported $FLEET_VAR_* references in
// contents for the given host. Values are defined in a preamble and the body
// keeps its tokens, so a value is never parsed as script source. It returns the
// expanded contents, or a non-empty failureMessage when a variable can't be
// resolved for this host (one line per failing variable). Unsupported variable
// names are left untouched: validation rejects them in new content, and content
// saved before validation shipped must keep working unchanged.
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
		// Python has no $VAR expansion, and no splice into it is context-free
		return 0, pythonFleetVarsMsg
	}
	return variables.DialectPOSIX, ""
}
