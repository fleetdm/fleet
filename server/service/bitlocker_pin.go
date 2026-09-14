package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
)

// BitLocker startup PIN relay.
//
// Windows only offers PIN setup through Manage BitLocker, which needs UAC elevation, so a standard user cannot satisfy
// a fleet that requires a startup PIN. fleetd runs as SYSTEM and can add the protector on their behalf, but the modal
// the end user types into lives in a browser, and nothing on the device lets that page reach fleetd. So the PIN is
// relayed through the server: the device endpoint below stores it encrypted, the agent collects it exactly once on its
// next config poll, applies it, and reports back. The server never hands the PIN to a user-authenticated caller. The
// PIN is encrypted with the server private key, not the WSTEP certificate that protects the BitLocker recovery key. The
// server normally holds the ciphertext for seconds. A submission the agent never collects is cleared by the hourly
// cleanups cron once its TTL passes, so the worst case is the TTL plus an hour.

// bitLockerPINClientErrorMaxLength matches the width of host_bitlocker_pin_requests.client_error.
const bitLockerPINClientErrorMaxLength = 255

// bitLockerPINLicensed reports whether this instance's license covers the BitLocker PIN flow.
//
// Only a Premium instance can turn on require_bitlocker_pin in the first place (UpdateMDMDiskEncryption refuses
// without a Premium license), so in practice a Free host never reaches these paths. The check is still made on every
// device-facing and agent-facing path so that a queued submission cannot outlive the license that justified it, and so
// the licensing boundary is stated here rather than inferred from a setting three layers away.
func bitLockerPINLicensed(ctx context.Context) bool {
	lic, _ := license.FromContext(ctx)
	return lic != nil && lic.IsPremium()
}

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

// RedactedForDebugLog keeps the submitted PIN out of the server's debug logs, which marshal whole request objects.
func (r *submitDiskEncryptionPINRequest) RedactedForDebugLog() any {
	redacted := *r
	if redacted.PIN != "" {
		redacted.PIN = fleet.MaskedPassword
	}
	return redacted
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

	if !bitLockerPINLicensed(ctx) {
		return fleet.ErrMissingLicense
	}

	if err := microsoft_mdm.ValidateBitLockerPIN(pin); err != nil {
		return ctxerr.Wrap(ctx, err, "validate bitlocker pin")
	}

	// Re-check eligibility on submit rather than trusting the page, which may be showing a stale view of a host whose
	// fleet stopped requiring a PIN, or whose PIN another session already set. Read from the primary: this gates the
	// write below, and a lagging replica would let a stale "needs a PIN" answer queue another secret.
	needsPIN, fleetdCapable, err := svc.bitLockerPINState(ctxdb.RequirePrimary(ctx, true), host)
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

	// Without the license the page must not offer the PIN form, so it falls back to the Manage BitLocker instructions.
	if !bitLockerPINLicensed(ctx) || host.FleetPlatform() != "windows" {
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
	// The cleanups cron retires abandoned submissions, but the page should not have to wait for it: a request past its
	// TTL is already uncollectable, so report it as timed out rather than leaving the modal spinning on "pending".
	if req != nil && req.Expired(time.Now()) {
		req = &fleet.HostBitLockerPINRequest{
			Status:    fleet.BitLockerPINRequestFailed,
			Error:     fleet.BitLockerPINRequestTimedOutError,
			CreatedAt: req.CreatedAt,
		}
	}

	return state.FleetdBitLockerPINCapable, req, nil
}

// bitLockerPINState reports whether the host is currently being asked to create a startup PIN, and whether its fleetd
// can apply one, for the submit endpoint's eligibility check. The orbit notification and the Fleet Desktop flag reach
// the same answer through HostMDMDiskEncryption.NeedsBitLockerPIN, so the three cannot disagree about whether a PIN is wanted.
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

	return de.NeedsBitLockerPIN(), state.FleetdBitLockerPINCapable, nil
}

////////////////////////////////////////////////////////////////////////////////
// Orbit collects the submitted PIN
////////////////////////////////////////////////////////////////////////////////

func getOrbitDiskEncryptionPINEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	pin, requestUUID, err := svc.GetBitLockerPINForHost(ctx)
	if err != nil {
		return fleet.OrbitGetDiskEncryptionPINResponse{Err: err}, nil
	}
	return fleet.OrbitGetDiskEncryptionPINResponse{PIN: pin, RequestUUID: requestUUID}, nil
}

func (svc *Service) GetBitLockerPINForHost(ctx context.Context) (string, string, error) {
	// Orbit node key authentication, not a Fleet user.
	svc.authz.SkipAuthorization(ctx)

	if !bitLockerPINLicensed(ctx) {
		return "", "", fleet.ErrMissingLicense
	}

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return "", "", newOsqueryError("internal error: missing host from request context")
	}

	// Check this before consuming the request. Collecting is destructive, so discovering a missing key afterwards
	// would burn the user's submission for nothing.
	if svc.config.Server.PrivateKey == "" {
		return "", "", newOsqueryError("internal error: missing server private key")
	}

	encryptedPIN, requestUUID, err := svc.ds.TakeBitLockerPINRequest(ctx, host)
	if err != nil {
		// notFound covers never-submitted, already-collected, already-finished and expired alike. The agent treats
		// them identically: there is nothing to apply on this poll.
		return "", "", ctxerr.Wrap(ctx, err, "take bitlocker pin request")
	}

	pin, err := mdm.DecodeAndDecrypt(encryptedPIN, svc.config.Server.PrivateKey)
	if err != nil {
		// The ciphertext is already gone, so nothing can rescue this submission. Retire it as failed rather than
		// leaving the page waiting on a delivered request that will never produce an outcome.
		if outcomeErr := svc.ds.SetBitLockerPINRequestOutcome(ctx, host, requestUUID,
			fleet.BitLockerPINRequestFailed, "Fleet could not read the submitted PIN. Try again."); outcomeErr != nil {
			svc.logger.ErrorContext(ctx, "retiring undecryptable bitlocker pin request", "err", outcomeErr)
		}
		return "", "", ctxerr.Wrap(ctx, err, "internal error: could not decrypt BitLocker PIN")
	}

	return pin, requestUUID, nil
}

////////////////////////////////////////////////////////////////////////////////
// Orbit reports whether it applied the PIN
////////////////////////////////////////////////////////////////////////////////

func postOrbitDiskEncryptionPINEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.OrbitPostDiskEncryptionPINRequest)
	if err := svc.SetBitLockerPINOutcome(ctx, req.RequestUUID, req.Outcome, req.ClientError); err != nil {
		return fleet.OrbitPostDiskEncryptionPINResponse{Err: err}, nil
	}
	return fleet.OrbitPostDiskEncryptionPINResponse{}, nil
}

func (svc *Service) SetBitLockerPINOutcome(
	ctx context.Context, requestUUID string, outcome fleet.BitLockerPINRequestStatus, clientError string,
) error {
	// Orbit node key authentication, not a Fleet user.
	svc.authz.SkipAuthorization(ctx)

	if !bitLockerPINLicensed(ctx) {
		return fleet.ErrMissingLicense
	}

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return newOsqueryError("internal error: missing host from request context")
	}

	// clientError is untrusted input from fleetd: normalize it before judging whether it says anything. Truncate by
	// characters rather than bytes: a Windows error in a non-English locale is multi-byte, and cutting one in half would
	// produce invalid UTF-8 that MySQL rejects, failing the report outright.
	clientError = str.TruncateRunes(strings.TrimSpace(clientError), bitLockerPINClientErrorMaxLength)

	switch outcome {
	case fleet.BitLockerPINRequestSet:
		// Record the outcome first, and only go on if it actually landed on a submission this host collected. That is
		// what stops a host claiming a PIN it was never given: without it, any Windows MDM host could post a success
		// and have Fleet mark it as having a startup PIN it does not have.
		if err := svc.ds.SetBitLockerPINRequestOutcome(ctx, host, requestUUID, outcome, ""); err != nil {
			return ctxerr.Wrap(ctx, err, "set bitlocker pin request outcome")
		}

		// From here on the outcome is recorded, so nothing below may fail the request. An error would make the agent
		// retry, the retry would find the submission already settled and get a 404, and the activity would never be
		// written. These steps only speed up what osquery reports anyway: tpm_pin_set_verify owns the protector list
		// and sets tpm_pin_set on its own schedule. They are separate calls rather than part of the outcome transaction
		// because both go through the host cache wrappers, which have to invalidate Redis after the write.
		//
		// Set the flag so the end user's banner clears now, and ask for a refetch so osquery confirms it within seconds.
		if err := svc.ds.SetOrUpdateHostDiskTpmPIN(ctx, host.ID, true); err != nil {
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

// setBitLockerPINNotification tells a capable agent to collect a PIN the end user has submitted. An error leaves the
// notification unset; the caller logs it rather than failing the whole config response, since the agent simply tries
// again on its next poll.
//
// Both gates come off the enrollment row the orbit config check-in already read, so an ordinary poll costs nothing
// extra here. Only when a PIN is genuinely waiting, which is rare and short-lived, does this confirm the host still
// needs it, so that turning the requirement off, or another session setting a PIN first, quietly drops the submission
// rather than applying it late.
func (svc *Service) setBitLockerPINNotification(
	ctx context.Context, notifs *fleet.OrbitConfigNotifications, host *fleet.Host,
	state *fleet.MDMWindowsHostConfigState, fleetdCapable bool,
) error {
	// fleetdCapable comes from this request's capability header rather than state, which was read before the header was
	// persisted: on the poll where an agent first advertises the capability the stored value is still false, and gating
	// on it would swallow the notification for that poll.
	if state == nil || !fleetdCapable || !state.BitLockerPINRequestPending || !bitLockerPINLicensed(ctx) {
		return nil
	}

	// Primary-routed: a stale replica read here would delete a submission the user just made and is waiting on.
	de, err := svc.ds.GetMDMWindowsBitLockerStatus(ctxdb.RequirePrimary(ctx, true), host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get bitlocker status for pin notification")
	}
	if !de.NeedsBitLockerPIN() {
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
