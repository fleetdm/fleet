package profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// ErrUnresolvableAndroidAppConfigVar signals that one of the $FLEET_VAR_*
// tokens referenced in an Android managed app configuration could not be
// resolved for the target host.
var ErrUnresolvableAndroidAppConfigVar = errors.New("android: unresolvable Fleet variable in managed app configuration")

// UnresolvableAndroidAppConfigVarError carries the unresolved variable name
// and a user-facing detail message.
type UnresolvableAndroidAppConfigVarError struct {
	FleetVar string
	Detail   string
}

func (e *UnresolvableAndroidAppConfigVarError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return fmt.Sprintf("android: unresolvable Fleet variable $FLEET_VAR_%s", e.FleetVar)
}

func (e *UnresolvableAndroidAppConfigVarError) Is(target error) bool {
	return target == ErrUnresolvableAndroidAppConfigVar
}

// AndroidAppConfigSubstitutionHost carries the host context needed to
// substitute host-scoped $FLEET_VAR_* tokens and $FLEET_HOST_VITAL_<id>
// custom host vitals in Android managed app configuration.
type AndroidAppConfigSubstitutionHost struct {
	HostID         uint
	UUID           string
	HardwareSerial string
	Platform       string
}

// SubstituteFleetVarsInAndroidAppConfig replaces every supported $FLEET_VAR_*
// token in config with the resolved value for the given host, returning the
// substituted bytes. End-user IDP fields are looked up via ds.
// Returns ErrUnresolvableAndroidAppConfigVar (wrapped) if the host can't
// supply a referenced variable.
func SubstituteFleetVarsInAndroidAppConfig(
	ctx context.Context,
	ds fleet.Datastore,
	config []byte,
	host AndroidAppConfigSubstitutionHost,
) ([]byte, error) {
	if len(config) == 0 {
		return config, nil
	}
	contents := string(config)
	used := variables.Find(contents)
	hasHostVitals := len(fleet.FindCustomHostVitalIDs(contents)) > 0
	if len(used) == 0 && !hasHostVitals {
		return config, nil
	}

	idpUUIDCache := map[string]uint{}

	for _, name := range used {
		switch fleet.FleetVarName(name) {
		case fleet.FleetVarHostUUID:
			contents = replaceJSONSafe(contents, name, host.UUID)

		case fleet.FleetVarHostHardwareSerial:
			if host.HardwareSerial == "" {
				return nil, &UnresolvableAndroidAppConfigVarError{
					FleetVar: name,
					Detail:   fmt.Sprintf("There is no serial number for this host. Fleet couldn't populate $FLEET_VAR_%s.", name),
				}
			}
			contents = replaceJSONSafe(contents, name, host.HardwareSerial)

		case fleet.FleetVarHostPlatform:
			contents = replaceJSONSafe(contents, name, host.Platform)

		case fleet.FleetVarHostEndUserEmailIDP:
			emails, err := ds.GetHostEmails(ctx, host.UUID, fleet.DeviceMappingMDMIdpAccounts)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get host idp email for android app config")
			}
			if len(emails) == 0 {
				return nil, &UnresolvableAndroidAppConfigVarError{
					FleetVar: name,
					Detail:   fmt.Sprintf("There is no IdP email for this host. Fleet couldn't populate $FLEET_VAR_%s.", name),
				}
			}
			contents = replaceJSONSafe(contents, name, emails[0])

		case fleet.FleetVarHostEndUserIDPUsername,
			fleet.FleetVarHostEndUserIDPUsernameLocalPart,
			fleet.FleetVarHostEndUserIDPGroups,
			fleet.FleetVarHostEndUserIDPDepartment,
			fleet.FleetVarHostEndUserIDPFullname:
			var detail string
			value, _, ok, err := ResolveHostEndUserIDPValue(
				ctx, ds, name, host.UUID, idpUUIDCache,
				func(d string) error { detail = d; return nil },
			)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "resolve host idp variable for android app config")
			}
			if !ok {
				return nil, &UnresolvableAndroidAppConfigVarError{FleetVar: name, Detail: detail}
			}
			contents = replaceJSONSafe(contents, name, value)

		default:
			return nil, &UnresolvableAndroidAppConfigVarError{FleetVar: name}
		}
	}

	if hasHostVitals {
		expanded, err := ds.ExpandCustomHostVitals(ctx, host.HostID, contents)
		if err != nil {
			return nil, err
		}
		contents = expanded
	}

	return []byte(contents), nil
}

// ContainsFleetVarOrCustomHostVital reports whether content has a $FLEET_VAR_*
// token or a $FLEET_HOST_VITAL_<id> token. Checks bytes for the vital prefix
// before falling back to fleet.FindCustomHostVitalIDs, which needs a string, to
// avoid that conversion's allocation in the common case where content has neither.
func ContainsFleetVarOrCustomHostVital(content []byte) bool {
	if variables.ContainsBytes(content) {
		return true
	}
	if !bytes.Contains(content, []byte(fleet.CustomHostVitalPrefix)) {
		return false
	}
	return len(fleet.FindCustomHostVitalIDs(string(content))) > 0
}

// replaceJSONSafe replaces a Fleet variable in contents with a JSON-safe value.
func replaceJSONSafe(contents, variableName, value string) string {
	return variables.Replace(contents, variableName, jsonEscapeString(value))
}

// jsonEscapeString returns value with JSON special characters escaped,
// suitable for embedding inside a JSON string literal.
func jsonEscapeString(s string) string {
	b, _ := json.Marshal(s) // json.Marshal for strings never errors
	return strings.TrimSuffix(strings.TrimPrefix(string(b), `"`), `"`)
}
