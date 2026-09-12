package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
)

// BitLocker startup PIN relay.
//
// Windows only offers PIN setup through Manage BitLocker, which needs UAC elevation, so a standard user cannot satisfy
// a fleet that requires a startup PIN. fleetd runs as SYSTEM and can add the protector on their behalf, but the modal
// the end user types into lives in a browser, and nothing on the device lets that page reach fleetd. So the PIN is
// relayed through the server: the device endpoint below stores it encrypted, the agent collects it exactly once on its
// next config poll, applies it, and reports back. The server holds the secret for seconds, and never hands it to a
// user-authenticated caller.

// bitLockerPINClientErrorMaxLength matches the width of host_bitlocker_pin_requests.client_error.
const bitLockerPINClientErrorMaxLength = 255

////////////////////////////////////////////////////////////////////////////////
// Submit a BitLocker PIN from the My device page
////////////////////////////////////////////////////////////////////////////////

type submitDiskEncryptionPINRequest struct {
	Token string `url:"token"`
	PIN   string `json:"pin"`
}

func (r *submitDiskEncryptionPINRequest) deviceAuthToken() string {
	return r.Token
}

type submitDiskEncryptionPINResponse struct {
	Err error `json:"error,omitempty"`
}

func (r submitDiskEncryptionPINResponse) Error() error { return r.Err }

func (r submitDiskEncryptionPINResponse) Status() int { return http.StatusNoContent }

func submitDiskEncryptionPINEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*submitDiskEncryptionPINRequest)
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return submitDiskEncryptionPINResponse{Err: err}, nil
	}

	if err := svc.SubmitBitLockerPIN(ctx, host, req.PIN); err != nil {
		return submitDiskEncryptionPINResponse{Err: err}, nil
	}
	return submitDiskEncryptionPINResponse{}, nil
}

func (svc *Service) SubmitBitLockerPIN(ctx context.Context, host *fleet.Host, pin string) error {
	// The device auth token in the URL is the authorization for this endpoint; there is no Fleet user.
	svc.authz.SkipAuthorization(ctx)

	if err := fleet.ValidateBitLockerPIN(pin); err != nil {
		return ctxerr.Wrap(ctx, err, "validate bitlocker pin")
	}

	// Re-check eligibility on submit rather than trusting the page, which may be showing a stale view of a host whose
	// fleet stopped requiring a PIN, or whose PIN another session already set.
	needsPIN, fleetdCapable, err := svc.bitLockerPINState(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "check bitlocker pin eligibility")
	}
	switch {
	case !needsPIN:
		return &fleet.BadRequestError{Message: "This host doesn't need a BitLocker PIN."}
	case !fleetdCapable:
		return &fleet.BadRequestError{Message: "Fleet's agent on this host is too old to set a BitLocker PIN. Update fleetd and try again."}
	}

	if svc.config.Server.PrivateKey == "" {
		return newOsqueryError("internal error: missing server private key")
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
	// Device-authenticated, same as SubmitBitLockerPIN.
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

	req, err := svc.ds.GetBitLockerPINRequest(ctx, host.ID)
	switch {
	case fleet.IsNotFound(err):
		req = nil
	case err != nil:
		return false, nil, ctxerr.Wrap(ctx, err, "get bitlocker pin request")
	}

	return state.FleetdBitLockerPINCapable, req, nil
}

// bitLockerPINState reports whether the host is currently being asked to create a startup PIN, and whether its fleetd
// can apply one. It is the single source of truth behind the submit endpoint's eligibility check, the orbit
// notification, and the Fleet Desktop toast, so those three can never disagree.
//
// "Needs a PIN" reuses the action_required derivation rather than re-deriving it: that is what already accounts for a
// PIN being required, not yet set, and actually settable on this volume right now.
func (svc *Service) bitLockerPINState(ctx context.Context, host *fleet.Host) (needsPIN bool, fleetdCapable bool, err error) {
	if host.FleetPlatform() != "windows" {
		return false, false, nil
	}

	state, err := svc.ds.GetMDMWindowsHostConfigState(ctx, host.UUID)
	switch {
	case fleet.IsNotFound(err):
		return false, false, nil
	case err != nil:
		return false, false, ctxerr.Wrap(ctx, err, "get windows mdm config state for bitlocker pin")
	}

	de, err := svc.ds.GetMDMWindowsBitLockerStatus(ctx, host)
	if err != nil {
		return false, false, ctxerr.Wrap(ctx, err, "get bitlocker status for pin eligibility")
	}

	return fleet.HostNeedsBitLockerPIN(de), state.FleetdBitLockerPINCapable, nil
}

////////////////////////////////////////////////////////////////////////////////
// Orbit collects the submitted PIN
////////////////////////////////////////////////////////////////////////////////

func getOrbitDiskEncryptionPINEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	pin, err := svc.GetBitLockerPINForHost(ctx)
	if err != nil {
		return fleet.OrbitGetDiskEncryptionPINResponse{Err: err}, nil
	}
	return fleet.OrbitGetDiskEncryptionPINResponse{PIN: pin}, nil
}

func (svc *Service) GetBitLockerPINForHost(ctx context.Context) (string, error) {
	// Orbit node key authentication, not a Fleet user.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return "", newOsqueryError("internal error: missing host from request context")
	}

	encryptedPIN, err := svc.ds.TakeBitLockerPINRequest(ctx, host)
	if err != nil {
		// notFound covers never-submitted, already-collected, already-finished and expired alike. The agent treats
		// them identically: there is nothing to apply on this poll.
		return "", ctxerr.Wrap(ctx, err, "take bitlocker pin request")
	}

	if svc.config.Server.PrivateKey == "" {
		return "", newOsqueryError("internal error: missing server private key")
	}
	pin, err := mdm.DecodeAndDecrypt(encryptedPIN, svc.config.Server.PrivateKey)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "internal error: could not decrypt BitLocker PIN")
	}

	return pin, nil
}

////////////////////////////////////////////////////////////////////////////////
// Orbit reports whether it applied the PIN
////////////////////////////////////////////////////////////////////////////////

func postOrbitDiskEncryptionPINEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.OrbitPostDiskEncryptionPINRequest)
	if err := svc.SetBitLockerPINOutcome(ctx, req.Outcome, req.ClientError); err != nil {
		return fleet.OrbitPostDiskEncryptionPINResponse{Err: err}, nil
	}
	return fleet.OrbitPostDiskEncryptionPINResponse{}, nil
}

func (svc *Service) SetBitLockerPINOutcome(
	ctx context.Context, outcome fleet.BitLockerPINRequestStatus, clientError string,
) error {
	// Orbit node key authentication, not a Fleet user.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return newOsqueryError("internal error: missing host from request context")
	}

	// clientError is untrusted input from fleetd: normalize it before judging whether it says anything.
	clientError = strings.TrimSpace(clientError)
	if len(clientError) > bitLockerPINClientErrorMaxLength {
		clientError = clientError[:bitLockerPINClientErrorMaxLength]
	}

	switch outcome {
	case fleet.BitLockerPINRequestSet:
		// The agent's report is a claim, not an observation of record: tpm_pin_set_verify owns the protector list. Set
		// the flag so the end user's banner clears now, and ask for a refetch so osquery confirms it within seconds.
		if err := svc.ds.SetOrUpdateHostDiskTpmPIN(ctx, host.ID, true); err != nil {
			return ctxerr.Wrap(ctx, err, "recording bitlocker pin set")
		}
		if err := svc.ds.SetBitLockerPINRequestOutcome(ctx, host, outcome, ""); err != nil {
			return ctxerr.Wrap(ctx, err, "set bitlocker pin request outcome")
		}
		if err := svc.ds.UpdateHostRefetchRequested(ctx, host.ID, true); err != nil {
			return ctxerr.Wrap(ctx, err, "requesting refetch after setting bitlocker pin")
		}
		// The end user chose the PIN, so this is deliberately recorded with no actor rather than as Fleet-initiated.
		if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeCreatedDiskEncryptionPIN{
			HostID:          host.ID,
			HostDisplayName: host.DisplayName(),
		}); err != nil {
			// OK: losing the audit entry must not fail the endpoint, which would make the agent retry an apply that
			// already succeeded.
			svc.logger.ErrorContext(ctx, "record created disk encryption pin activity", "err", err)
			ctxerr.Handle(ctx, err)
		}

	case fleet.BitLockerPINRequestFailed:
		if clientError == "" {
			return fleet.NewInvalidArgumentError("client_error", "cannot be empty when outcome is failed")
		}
		if err := svc.ds.SetBitLockerPINRequestOutcome(ctx, host, outcome, clientError); err != nil {
			return ctxerr.Wrap(ctx, err, "set bitlocker pin request outcome")
		}

	default:
		return &fleet.BadRequestError{Message: "unknown outcome " + string(outcome)}
	}

	return nil
}

// setBitLockerPINNotification tells a capable agent to collect a PIN the end user has submitted.
//
// Both gates come off the enrollment row the orbit config check-in already read, so an ordinary poll costs nothing
// extra here. Only when a PIN is genuinely waiting, which is rare and short-lived, does this confirm the host still
// needs it, so that turning the requirement off, or another session setting a PIN first, quietly drops the submission
// rather than applying it late.
func (svc *Service) setBitLockerPINNotification(
	ctx context.Context, notifs *fleet.OrbitConfigNotifications, host *fleet.Host, state *fleet.MDMWindowsHostConfigState,
) error {
	if state == nil || !state.FleetdBitLockerPINCapable || !state.BitLockerPINRequestPending {
		return nil
	}

	de, err := svc.ds.GetMDMWindowsBitLockerStatus(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get bitlocker status for pin notification")
	}
	if !fleet.HostNeedsBitLockerPIN(de) {
		// The host no longer needs a PIN, so the submission is moot. Drop it rather than leaving it to expire, so the
		// page stops waiting and the secret is not held for the rest of the TTL.
		if err := svc.ds.DeleteBitLockerPINRequest(ctx, host); err != nil {
			return ctxerr.Wrap(ctx, err, "discard stale bitlocker pin request")
		}
		return nil
	}

	notifs.BitLockerPINRequestPending = true
	return nil
}

// applyBitLockerPINDeviceFields populates the My device response with the PIN state. Absent on every other response,
// because the fields are pointers and only this path sets them.
func applyBitLockerPINDeviceFields(resp *fleet.HostDetailResponse, canSetPIN bool, req *fleet.HostBitLockerPINRequest) {
	if resp == nil || resp.MDM.OSSettings == nil {
		return
	}
	resp.MDM.OSSettings.DiskEncryption.FleetdCanSetPIN = new(canSetPIN)
	resp.MDM.OSSettings.DiskEncryption.PINRequest = req
}
