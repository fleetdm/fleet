package service

import (
	"context"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
)

// BitLocker startup PIN handoff.
//
// A standard user can't set a startup PIN without UAC elevation, and the browser modal can't reach fleetd, which runs as SYSTEM.
// So the server stores the PIN encrypted with its private key until the agent collects it once on its next config poll, applies
// it, and reports back. Only the agent can collect it. An uncollected PIN is cleared by the hourly cleanups cron after its TTL,
// so the worst case is the TTL plus an hour.

const (
	bitLockerPINNotNeededMessage   = "This host doesn't need a BitLocker PIN."
	bitLockerPINAgentTooOldMessage = "Fleet's agent on this host is too old to set a BitLocker PIN. Update fleetd and try again."
	bitLockerPINUnreadableError    = "Fleet could not read the submitted PIN. Try again."
)

func (svc *Service) SubmitBitLockerPIN(ctx context.Context, host *fleet.Host, pin string) error {
	// The device auth token in the URL is the authorization for this endpoint; there is no Fleet user.
	svc.authz.SkipAuthorization(ctx)

	if err := microsoft_mdm.ValidateBitLockerPIN(pin); err != nil {
		return ctxerr.Wrap(ctx, err, "validate bitlocker pin")
	}

	// Re-check eligibility on submit rather than trusting the page, which may be showing a stale view of a host whose
	// fleet stopped requiring a PIN, or whose PIN another session already set. The capability is checked first because
	// it comes off a cheap row, and an agent that can't apply a PIN makes the BitLocker status irrelevant.
	notNeeded := &fleet.BadRequestError{Message: bitLockerPINNotNeededMessage}
	if host.FleetPlatform() != "windows" {
		return notNeeded
	}
	state, err := svc.ds.GetMDMWindowsHostConfigState(ctx, host.UUID)
	switch {
	case fleet.IsNotFound(err):
		return notNeeded
	case err != nil:
		return ctxerr.Wrap(ctx, err, "get windows mdm config state for bitlocker pin")
	case !state.FleetdBitLockerPINCapable:
		return &fleet.BadRequestError{Message: bitLockerPINAgentTooOldMessage}
	}
	de, err := svc.ds.GetMDMWindowsBitLockerStatus(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get bitlocker status for pin eligibility")
	}
	if !de.NeedsBitLockerPIN() {
		return notNeeded
	}

	if svc.config.Server.PrivateKey == "" {
		return ctxerr.New(ctx, "internal error: missing server private key")
	}
	encryptedPIN, err := mdm.EncryptAndEncode(pin, svc.config.Server.PrivateKey)
	if err != nil {
		// Deliberately not wrapping the underlying error into a user-facing message: it can echo the input.
		return ctxerr.Wrap(ctx, err, "internal error: could not encrypt BitLocker PIN")
	}

	return ctxerr.Wrap(ctx, svc.ds.QueueBitLockerPINRequest(ctx, host, encryptedPIN), "queue bitlocker pin request")
}

// BitLockerPINStateForDevice reports what the My device page needs to decide which modal to show and what to poll for.
func (svc *Service) BitLockerPINStateForDevice(
	ctx context.Context, host *fleet.Host,
) (bool, *fleet.HostBitLockerPINRequest, error) {
	// Device-authenticated.
	svc.authz.SkipAuthorization(ctx)

	if host.FleetPlatform() != "windows" {
		return false, nil, nil
	}

	state, err := svc.ds.GetMDMWindowsHostConfigState(ctx, host.UUID)
	switch {
	case fleet.IsNotFound(err):
		// Not enrolled in Windows MDM, so there is no agent that could apply a PIN.
		return false, nil, nil
	case err != nil:
		return false, nil, ctxerr.Wrap(ctx, err, "get windows mdm config state for bitlocker pin")
	}
	if !state.FleetdBitLockerPINCapable {
		return false, nil, nil
	}

	req, err := svc.ds.GetBitLockerPINRequest(ctx, host.ID)
	switch {
	case fleet.IsNotFound(err):
		req = nil
	case err != nil:
		return false, nil, ctxerr.Wrap(ctx, err, "get bitlocker pin request")
	}
	// The cleanups cron retires abandoned submissions, but the page should not have to wait for it: a request past its
	// TTL is already uncollectable, so report it as timed out rather than leaving the modal spinning on "pending".
	if req != nil && req.Expired(time.Now()) {
		req = &fleet.HostBitLockerPINRequest{
			Status:    fleet.BitLockerPINRequestFailed,
			Error:     fleet.BitLockerPINRequestTimedOutError,
			CreatedAt: req.CreatedAt,
		}
	}

	return true, req, nil
}

func (svc *Service) GetBitLockerPINForHost(ctx context.Context) (string, string, error) {
	// Orbit node key authentication, not a Fleet user.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return "", "", ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
	}

	if svc.config.Server.PrivateKey == "" {
		return "", "", ctxerr.New(ctx, "internal error: missing server private key")
	}

	encryptedPIN, requestUUID, err := svc.ds.TakeBitLockerPINRequest(ctx, host)
	if err != nil {
		// notFound covers never-submitted, already-collected, already-finished and expired alike. The agent treats
		// them identically: there is nothing to apply on this poll.
		return "", "", ctxerr.Wrap(ctx, err, "take bitlocker pin request")
	}

	pin, err := mdm.DecodeAndDecrypt(encryptedPIN, svc.config.Server.PrivateKey)
	if err != nil {
		// The ciphertext is already gone, so nothing can rescue this submission. Retire it as failed.
		if outcomeErr := svc.ds.SetBitLockerPINRequestOutcome(ctx, host, requestUUID,
			fleet.BitLockerPINRequestFailed, bitLockerPINUnreadableError); outcomeErr != nil {
			svc.logger.ErrorContext(ctx, "retiring undecryptable bitlocker pin request", "err", outcomeErr)
		}
		return "", "", ctxerr.Wrap(ctx, err, "internal error: could not decrypt BitLocker PIN")
	}

	return pin, requestUUID, nil
}

func (svc *Service) SetBitLockerPINOutcome(
	ctx context.Context, requestUUID string, outcome fleet.BitLockerPINRequestStatus, clientError string,
) error {
	// Orbit node key authentication, not a Fleet user.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
	}

	// clientError is untrusted input from fleetd: normalize it before judging whether it says anything.
	clientError = str.TruncateRunes(strings.TrimSpace(clientError), fleet.BitLockerPINClientErrorMaxLength)

	switch outcome {
	case fleet.BitLockerPINRequestSet:
		// Record the outcome first, and only go on if it actually landed on a submission this host collected.
		if err := svc.ds.SetBitLockerPINRequestOutcome(ctx, host, requestUUID, outcome, ""); err != nil {
			return ctxerr.Wrap(ctx, err, "set bitlocker pin request outcome")
		}

		// From here on the outcome is recorded, so nothing below may fail the request.
		// Set the flags so the end user's banner clears now, and ask for a refetch so osquery confirms it within seconds. A
		// TPM and PIN protector can release the volume master key at boot, so it is also a boot protector.
		if err := svc.ds.SetOrUpdateHostDiskBitLockerProtectors(ctx, host.ID, true, true); err != nil {
			svc.logger.ErrorContext(ctx, "recording bitlocker pin set after outcome", "host_id", host.ID, "err", err)
			ctxerr.Handle(ctx, err)
		}
		if err := svc.ds.UpdateHostRefetchRequested(ctx, host.ID, true); err != nil {
			svc.logger.ErrorContext(ctx, "requesting refetch after bitlocker pin set", "host_id", host.ID, "err", err)
			ctxerr.Handle(ctx, err)
		}
		// The end user chose the PIN, so this is deliberately recorded with no actor rather than as Fleet-initiated.
		if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeCreatedDiskEncryptionPIN{
			HostID:          host.ID,
			HostDisplayName: host.DisplayName(),
		}); err != nil {
			// OK: see above, the outcome is already recorded.
			svc.logger.ErrorContext(ctx, "record created disk encryption pin activity", "err", err)
			ctxerr.Handle(ctx, err)
		}

	case fleet.BitLockerPINRequestFailed:
		if clientError == "" {
			return fleet.NewInvalidArgumentError("client_error", "cannot be empty when outcome is failed")
		}
		if err := svc.ds.SetBitLockerPINRequestOutcome(ctx, host, requestUUID, outcome, clientError); err != nil {
			return ctxerr.Wrap(ctx, err, "set bitlocker pin request outcome")
		}

	default:
		return &fleet.BadRequestError{Message: "unknown outcome " + string(outcome)}
	}

	return nil
}
