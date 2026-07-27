package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/variables"
	"github.com/google/uuid"
)

// reconcileHostDeviceNamesBatchSize bounds how many queued rename rows a
// single cron tick processes; remaining rows are picked up on subsequent
// ticks, amortizing command delivery for large teams.
//
// var (not const) so tests can override it.
var reconcileHostDeviceNamesBatchSize = 500

// secretExpansion memoizes the result of expanding the custom (secret) variables
// in a host name template. err is set (e.g. fleet.MissingSecretsError) when a
// referenced secret is undefined.
type secretExpansion struct {
	value string
	err   error
}

// ReconcileHostDeviceNames runs one pass of host-name template enforcement:
// for each host whose enforcement row is queued (status NULL), it resolves
// the host's team name template and either enqueues a Settings/DeviceName
// command or records the outcome directly (name already matching → verified;
// resolved name unusable → failed).
func ReconcileHostDeviceNames(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger *slog.Logger,
) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading app config")
	}
	if !appConfig.MDM.EnabledAndConfigured {
		return nil
	}

	pending, err := ds.ListHostsPendingDeviceNameCommand(ctx, reconcileHostDeviceNamesBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list hosts pending device name command")
	}
	if len(pending) == 0 {
		return nil
	}

	// Every host in this batch is queued (status NULL), which for a host that
	// previously received a command means the command was reset (resend, template
	// change, transfer/enrollment reconcile) without having executed. Deactivate
	// any such lingering command before enqueuing fresh ones so an out-of-order
	// NotNow retry of a stale command can't rename the device to an old name.
	pendingUUIDs := make([]string, 0, len(pending))
	for _, host := range pending {
		pendingUUIDs = append(pendingUUIDs, host.HostUUID)
	}
	if err := ds.DeactivateHostDeviceNameCommands(ctx, pendingUUIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "deactivate stale device name commands")
	}

	noTeamTemplate := appConfig.MDM.HostNameTemplate.Value
	templates := make(map[uint]string) // team ID → name template
	// secretsExpanded caches the result of expanding $FLEET_SECRET_* custom
	// variables for a given raw template. Secret values are global (name-keyed,
	// host-independent), so the same template expands to the same string for
	// every host — expand once per distinct template rather than per host.
	secretsExpanded := make(map[string]secretExpansion)
	var notify []string // hosts with a freshly-enqueued command to push
	for _, host := range pending {
		var tmpl string
		if host.TeamID == nil {
			tmpl = noTeamTemplate
		} else {
			var ok bool
			tmpl, ok = templates[*host.TeamID]
			if !ok {
				mdmConfig, err := ds.TeamMDMConfig(ctx, *host.TeamID)
				if err != nil {
					if fleet.IsNotFound(err) {
						// team deleted between cron runs
						templates[*host.TeamID] = ""
						continue
					}
					return ctxerr.Wrap(ctx, err, "get team mdm config for device name")
				}
				tmpl = mdmConfig.HostNameTemplate
				templates[*host.TeamID] = tmpl
			}
		}
		if tmpl == "" {
			// Template cleared between cron runs, or the team was deleted (cached
			// as "" above), or a No-team template was cleared. Either way the
			// clear/delete/transfer path removes the rows, so there's nothing to
			// enforce here.
			continue
		}

		// Expand any custom (secret, $FLEET_SECRET_*) variables before resolving
		// the built-in host variables. Secret values are global and
		// host-independent, so this is memoized per distinct template.
		expandedTmpl := tmpl
		if len(fleet.ContainsPrefixVars(tmpl, fleet.ServerSecretPrefix)) > 0 {
			exp, ok := secretsExpanded[tmpl]
			if !ok {
				value, expandErr := ds.ExpandEmbeddedSecrets(ctx, tmpl)
				exp = secretExpansion{value: value, err: expandErr}
				secretsExpanded[tmpl] = exp
			}
			if exp.err != nil {
				if !fleet.IsMissingSecretsError(exp.err) {
					// A transient failure (e.g. a DB error while fetching/decrypting
					// the secret). Abort the batch so the next cron tick retries,
					// exactly like the team-config lookup error above — don't
					// permanently fail rows that a retry would resolve (failed rows
					// aren't re-picked until a manual resend).
					return ctxerr.Wrap(ctx, exp.err, "expand host name template secrets")
				}
				// A referenced secret is genuinely undefined (e.g. deleted). Save-time
				// validation and the delete guard normally prevent this, so it's a
				// defensive path: fail the row with the reason and don't send a
				// command. On a write error, log and move on rather than aborting the
				// batch, consistent with the other outcomes below.
				if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryFailed, nil, "",
					exp.err.Error()); err != nil {
					logger.ErrorContext(ctx, "mark device name row failed for missing secret", "host_uuid", host.HostUUID, "err", err)
				}
				continue
			}
			expandedTmpl = exp.value
		}

		// Expand any custom host vital ($FLEET_HOST_VITAL_<id>) references with this
		// host's stored value. Unlike secrets, vital values are per-host, so this
		// can't be memoized across hosts sharing a template.
		if len(fleet.ContainsCustomHostVitalIDs(expandedTmpl)) > 0 {
			withVitals, vitalErr := ds.ExpandCustomHostVitals(ctx, host.HostID, expandedTmpl)
			if vitalErr != nil {
				if _, ok := errors.AsType[*fleet.MissingCustomHostVitalValueError](vitalErr); !ok {
					return ctxerr.Wrap(ctx, vitalErr, "expand host name template custom host vitals")
				}
				// A referenced vital exists but has no value set for this host.
				if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryFailed, nil, "",
					vitalErr.Error()); err != nil {
					logger.ErrorContext(ctx, "mark device name row failed for missing custom host vital value", "host_uuid", host.HostUUID, "err", err)
				}
				continue
			}
			expandedTmpl = withVitals
		}

		resolved := fleet.ResolveHostNameTemplate(expandedTmpl, &fleet.Host{
			UUID:           host.HostUUID,
			HardwareSerial: host.HardwareSerial,
			Platform:       host.Platform,
		})

		// Resolve IdP end-user variables (if any). These need a per-host datastore
		// lookup and fail the same way configuration profiles do when the data is
		// missing.
		resolvedWithIDP, idpFailDetail, idpErr := resolveHostNameIDPVars(ctx, ds, resolved, host.HostID)
		if idpErr != nil {
			// A datastore failure; abort the batch so the next cron tick retries
			// rather than permanently failing rows a retry would resolve.
			return idpErr
		}
		if idpFailDetail != "" {
			// The host is missing IdP data the template needs; fail the row with the
			// profile-style detail. On a write error, log and move on rather than
			// aborting the batch, consistent with the other outcomes below.
			if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryFailed, nil, "",
				idpFailDetail); err != nil {
				logger.ErrorContext(ctx, "mark device name row failed for idp resolution", "host_uuid", host.HostUUID, "err", err)
			}
			continue
		}
		resolved = resolvedWithIDP

		switch {
		case len(resolved) > fleet.MaxResolvedHostNameBytes:
			// The resolved name is not stored: it can exceed the column width,
			// and failed rows are never compared against reported names. On a
			// write error, log and move on so one bad host doesn't abort the
			// batch (matches the enqueue branch below); the row stays queued and
			// a later cron run retries it.
			if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryFailed, nil, "",
				"Resolved name exceeds 63 bytes."); err != nil {
				logger.ErrorContext(ctx, "mark device name row failed for too-long name", "host_uuid", host.HostUUID, "err", err)
				continue
			}
			logger.InfoContext(ctx, "host name template resolves past the device name limit, not sending command",
				"host_uuid", host.HostUUID, "resolved_bytes", len(resolved))
		case resolved == host.ComputerName:
			// The device already carries the resolved name; no command needed.
			// On a write error, log and move on rather than aborting the batch.
			if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryVerified, nil, resolved, ""); err != nil {
				logger.ErrorContext(ctx, "mark device name row verified for matching name", "host_uuid", host.HostUUID, "err", err)
				continue
			}
		default:
			cmdUUID := fleet.DeviceNameCommandUUIDPrefix + uuid.NewString()
			if err := commander.DeviceNameSettingWithoutNotifications(ctx, host.HostUUID, cmdUUID, resolved); err != nil {
				// The command was not persisted; leave the row queued so the
				// next cron run retries this host, and move on so one bad
				// host doesn't starve the rest of the batch.
				logger.ErrorContext(ctx, "enqueue device name command", "host_uuid", host.HostUUID, "err", err)
				continue
			}
			// The command is persisted; the device will apply it on its next
			// check-in even if the batched push below never reaches it. Collect
			// the UUID so every device in this batch is woken with a single APNs
			// push instead of one request per host.
			notify = append(notify, host.HostUUID)
			if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryPending, &cmdUUID, resolved, ""); err != nil {
				// The command was sent but recording it failed; log and move on
				// rather than aborting the batch, consistent with the
				// enqueue-failure handling above. The row stays queued and a
				// later cron run re-sends; the device resolves to the latest
				// command, and the superseded one's result is dropped as stale.
				logger.ErrorContext(ctx, "mark device name command sent", "host_uuid", host.HostUUID, "command_uuid", cmdUUID, "err", err)
				continue
			}
		}
	}

	if len(notify) > 0 {
		// One batched push wakes every device whose command was just enqueued.
		// Same handling as the iOS/iPadOS revive cron: a per-device APNs failure
		// is tolerable (the command is already persisted, so the device applies
		// it on its next check-in) so it's logged and the run succeeds — retrying
		// would enqueue duplicates. Any other error means the push subsystem
		// itself failed, so surface it.
		if err := commander.SendNotifications(ctx, notify); err != nil {
			var apnsErr *apple_mdm.APNSDeliveryError
			if !errors.As(err, &apnsErr) {
				return ctxerr.Wrap(ctx, err, "push device name commands")
			}
			logger.InfoContext(ctx, "failed to push device name command to some hosts", "err", apnsErr.Error())
		}
	}
	return nil
}

// resolveHostNameIDPVars substitutes the IdP end-user built-in variables in name
// for the given host, with the same value mapping and fail-hard messages as
// configuration profiles (server/mdm/profiles.ResolveHostEndUserIDPValue). It
// fetches the host's end user once (all IdP variables in a template resolve from
// the same user), so a template using several IdP variables needs a single
// GetEndUsers call. It returns:
//   - the resolved name, when every IdP variable is populated;
//   - a non-empty failDetail (and empty name) when an IdP variable can't be
//     populated for this host — the caller marks the host's row Failed with it,
//     exactly as a profile install would fail;
//   - an error only for a datastore failure, which the caller treats as transient.
func resolveHostNameIDPVars(ctx context.Context, ds fleet.Datastore, name string, hostID uint) (resolved string, failDetail string, err error) {
	// Collect the IdP variables used, longest-first (variables.Find sorts that way)
	// so ..._IDP_USERNAME_LOCAL_PART is substituted before ..._IDP_USERNAME, whose
	// regexps have no trailing boundary — matching the profile processor's order.
	var idpVars []string
	for _, v := range variables.Find(name) {
		if fleet.IsHostNameTemplateIDPVar(v) {
			idpVars = append(idpVars, v)
		}
	}
	if len(idpVars) == 0 {
		return name, "", nil
	}

	users, err := fleet.GetEndUsers(ctx, ds, hostID)
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "get end users for device name")
	}
	var user *fleet.HostEndUser
	if len(users) > 0 && users[0].IdpUserName != "" {
		user = &users[0]
	}

	for _, v := range idpVars {
		value, rx, ok, detail := resolveHostNameIDPValue(user, v)
		if !ok {
			return "", detail, nil
		}
		name = rx.ReplaceAllLiteralString(name, value)
	}
	return name, "", nil
}

// resolveHostNameIDPValue mirrors server/mdm/profiles.ResolveHostEndUserIDPValue's
// value mapping and fail-hard detail messages, but works from an already-fetched
// end user (nil when the host has no IdP user) so the caller can resolve every IdP
// variable in a template from a single GetEndUsers call. On success it returns the
// value and the variable's regexp; otherwise ok is false and detail carries the
// profile-style failure message.
func resolveHostNameIDPValue(user *fleet.HostEndUser, fleetVar string) (value string, rx *regexp.Regexp, ok bool, detail string) {
	noGroupsErr := fmt.Sprintf("There are no IdP groups for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPGroups)
	noDepartmentErr := fmt.Sprintf("There is no IdP department for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPDepartment)
	noFullnameErr := fmt.Sprintf("There is no IdP full name for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPFullname)

	if user == nil {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserIDPGroups):
			return "", nil, false, noGroupsErr
		case string(fleet.FleetVarHostEndUserIDPDepartment):
			return "", nil, false, noDepartmentErr
		case string(fleet.FleetVarHostEndUserIDPFullname):
			return "", nil, false, noFullnameErr
		default:
			return "", nil, false, fmt.Sprintf("There is no IdP username for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleetVar)
		}
	}

	switch fleetVar {
	case string(fleet.FleetVarHostEndUserIDPUsername):
		return user.IdpUserName, fleet.FleetVarHostEndUserIDPUsernameRegexp, true, ""
	case string(fleet.FleetVarHostEndUserIDPUsernameLocalPart):
		localPart, _, _ := strings.Cut(user.IdpUserName, "@")
		return localPart, fleet.FleetVarHostEndUserIDPUsernameLocalPartRegexp, true, ""
	case string(fleet.FleetVarHostEndUserIDPGroups):
		if len(user.IdpGroups) == 0 {
			return "", nil, false, noGroupsErr
		}
		return strings.Join(user.IdpGroups, ","), fleet.FleetVarHostEndUserIDPGroupsRegexp, true, ""
	case string(fleet.FleetVarHostEndUserIDPDepartment):
		if user.Department == "" {
			return "", nil, false, noDepartmentErr
		}
		return user.Department, fleet.FleetVarHostEndUserIDPDepartmentRegexp, true, ""
	case string(fleet.FleetVarHostEndUserIDPFullname):
		fullName := strings.TrimSpace(user.IdpFullName)
		if fullName == "" {
			return "", nil, false, noFullnameErr
		}
		return fullName, fleet.FleetVarHostEndUserIDPFullnameRegexp, true, ""
	default:
		return "", nil, false, fmt.Sprintf("Fleet couldn't populate $FLEET_VAR_%s.", fleetVar)
	}
}
