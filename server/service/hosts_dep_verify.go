package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
)

// depDeleteCheck is what Apple says about a host's Apple Business assignment
// when a delete is requested.
type depDeleteCheck int

const (
	// depDeleteNoAssignment means nothing could restore this host after the
	// delete, so there is nothing to verify.
	depDeleteNoAssignment depDeleteCheck = iota
	// depDeleteDisowned means Apple no longer reports the device as assigned to
	// us. Releasing a device from Apple Business never sends the deleted op_type
	// that would clear Fleet's assignment record, so the record outlives the
	// assignment and would otherwise restore the host forever.
	depDeleteDisowned
	// depDeleteAssigned means Apple still reports the device as ours, so
	// restoring the host after the delete is correct.
	depDeleteAssigned
	// depDeleteUnverified means Apple gave no usable answer. Deleting would risk
	// reporting a success that silently reverses, so callers refuse instead.
	depDeleteUnverified
)

// errDEPLookupFailed is the cause recorded when Apple answered but reported that
// it could not look the device up.
var errDEPLookupFailed = errors.New("apple reported a failed device lookup")

// depDeleteResult is what the check concluded for one host, plus the Apple-side
// failure behind an unverified result so callers can surface it rather than
// leaving it in the logs.
type depDeleteResult struct {
	check    depDeleteCheck
	appleErr error
}

// checkDEPAssignmentsForDelete asks Apple whether the given hosts are still
// assigned to this Fleet instance, so a delete is not reported as successful
// when the host is about to be restored from a stale assignment record.
//
// Anything that leaves Fleet unable to get an answer about a host — Apple being
// unreachable, Apple reporting a failed lookup, or its Apple Business token not
// resolving — is reported as depDeleteUnverified for that host rather than failing
// the whole call. Only failures to read the state this check is built on (app
// config, the assignment rows) return an error.
func (svc *Service) checkDEPAssignmentsForDelete(ctx context.Context, hosts []*fleet.Host) (map[uint]depDeleteResult, error) {
	checks := make(map[uint]depDeleteResult, len(hosts))
	for _, h := range hosts {
		checks[h.ID] = depDeleteResult{check: depDeleteNoAssignment}
	}

	// Mirrors the gates the restore path itself applies: if it cannot run, no
	// host can come back and there is nothing worth asking Apple about.
	if !license.IsPremium(ctx) || svc.depStorage == nil {
		return checks, nil
	}
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config for dep delete check")
	}
	if !appCfg.MDM.AppleBMEnabledAndConfigured {
		return checks, nil
	}

	hostIDs := make([]uint, 0, len(hosts))
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}
	assignments, err := svc.ds.GetHostDEPAssignmentsByHostIDs(ctx, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host dep assignments for delete check")
	}
	if len(assignments) == 0 {
		return checks, nil
	}

	// Serials are looked up per Apple Business token, since each token addresses a
	// different organization.
	type pending struct {
		hostID uint
		serial string
	}
	byToken := make(map[uint][]pending)
	for _, a := range assignments {
		if a.ABMTokenID == nil || a.HardwareSerial == "" {
			// An assignment with no token is left over from an Apple Business token
			// that was removed from Fleet: there is no organization left to ask, and
			// nothing can restore the host through it either. Treat it as released so
			// the record is cleared rather than blocking the delete forever.
			checks[a.HostID] = depDeleteResult{check: depDeleteDisowned}
			continue
		}
		byToken[*a.ABMTokenID] = append(byToken[*a.ABMTokenID], pending{hostID: a.HostID, serial: a.HardwareSerial})
	}

	depClient := apple_mdm.NewDEPClient(svc.depStorage, svc.ds, svc.logger)
	for tokenID, entries := range byToken {
		token, err := svc.ds.GetABMTokenByID(ctx, tokenID)
		if err != nil {
			svc.logger.ErrorContext(ctx, "get ABM token for dep delete check", "abm_token_id", tokenID, "err", err)
			for _, e := range entries {
				checks[e.hostID] = depDeleteResult{check: depDeleteUnverified, appleErr: err}
			}
			continue
		}

		for start := 0; start < len(entries); start += apple_mdm.DEPSyncLimit {
			end := min(start+apple_mdm.DEPSyncLimit, len(entries))
			chunk := entries[start:end]

			serials := make([]string, 0, len(chunk))
			for _, e := range chunk {
				serials = append(serials, e.serial)
			}

			details, err := depClient.GetDevicesDetails(ctx, token.OrganizationName, serials...)
			if err != nil {
				svc.logger.ErrorContext(ctx, "get DEP device details for delete check",
					"abm_token_id", tokenID, "org_name", token.OrganizationName, "devices", len(serials), "err", err)
				for _, e := range chunk {
					checks[e.hostID] = depDeleteResult{check: depDeleteUnverified, appleErr: err}
				}
				continue
			}

			for _, e := range chunk {
				d, answered := details[e.serial]
				if !answered || d == nil {
					svc.logger.WarnContext(ctx, "no DEP device details returned for serial",
						"abm_token_id", tokenID, "host_id", e.hostID)
				}
				res := depDeleteResult{check: classifyDEPDeviceDetails(d)}
				if res.check == depDeleteUnverified {
					// Apple answered, so there is no transport error to carry — record
					// why the host could not be verified rather than reporting no cause.
					res.appleErr = errDEPLookupFailed
				}
				checks[e.hostID] = res
			}
		}
	}

	return checks, nil
}

// classifyDEPDeviceDetails turns Apple's per-device answer into a check. A nil
// details means Apple replied without mentioning the serial at all.
//
// Blocking is reserved for Apple failing to answer — an unreachable API or a
// FAILED lookup. Anything else is acted on: NOT_ACCESSIBLE means the device is no
// longer ours, and both a readable answer and a silent one mean it still is.
// Apple documents a per-serial status for everything asked about, so silence is
// unexpected — but blocking on it would let one Apple quirk stop deletions
// fleet-wide, a wider blast radius than the bug this guards against.
func classifyDEPDeviceDetails(d *godep.DeviceDetails) depDeleteCheck {
	if d == nil {
		return depDeleteAssigned
	}
	switch godep.DeviceStatus(d.ResponseStatus) {
	case godep.DeviceStatusNotAccessible:
		return depDeleteDisowned
	case godep.DeviceStatusFailed:
		return depDeleteUnverified
	}
	return depDeleteAssigned
}

// clearDisownedDEPAssignments marks the assignment records Apple has disowned as
// deleted — the same thing the DEP syncer does on a deleted op_type, which Apple
// never sends on release. The restore path declines to recreate a host whose
// assignment is marked deleted, so this is what makes the delete stick.
//
// Marks by host rather than DeleteHostDEPAssignments, which is keyed by Apple
// Business token (so it cannot express an assignment that has lost its token) and
// additionally deletes pending host rows — a side effect the delete path doesn't
// want, since the host it was asked to delete is about to go anyway.
//
// Deliberately not in the same transaction as the host delete that follows. The
// record is marked because Apple said the device is not ours, which holds whether
// or not that delete then succeeds: a failed delete leaves the record more
// accurate than it was, and the retry succeeds because no live assignment is left
// to restore from.
func (svc *Service) clearDisownedDEPAssignments(ctx context.Context, checks map[uint]depDeleteResult) error {
	var hostIDs []uint
	for hostID, c := range checks {
		if c.check == depDeleteDisowned {
			hostIDs = append(hostIDs, hostID)
		}
	}
	if len(hostIDs) == 0 {
		return nil
	}
	if err := svc.ds.MarkHostDEPAssignmentsDeleted(ctx, hostIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "clearing disowned dep assignments")
	}
	return nil
}

// unverifiedABMHostsError reports the hosts a bulk delete left in place because
// Apple could not be asked about them, carrying one of the underlying Apple
// failures so the cause is not lost.
//
// deleted is how many hosts the batch did remove, so a caller can tell a partial
// delete from one that removed nothing and decide whether retrying the whole
// request is safe.
func unverifiedABMHostsError(checks map[uint]depDeleteResult, names []string, deleted int) error {
	var appleErr error
	for _, c := range checks {
		if c.check == depDeleteUnverified && c.appleErr != nil {
			appleErr = c.appleErr
			break
		}
	}
	msg := fmt.Sprintf("%s Hosts: %s.", fleet.CantDeleteHostUnverifiedABMMessage, strings.Join(names, ", "))
	if deleted > 0 {
		msg = fmt.Sprintf("%s The other %d host(s) were deleted.", msg, deleted)
	}
	return fleet.NewBadGatewayError(msg, appleErr)
}
