package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// BitLocker startup PIN handoff. The Premium implementation is in ee/server/service/bitlocker_pin.go. The stubs below
// skip authorization so the missing license error is not replaced by a forbidden one.

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
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

// BitLockerPINStateForDevice reports nothing on Free, so the My device page shows the Manage BitLocker instructions.
func (svc *Service) BitLockerPINStateForDevice(ctx context.Context, host *fleet.Host) (bool, *fleet.HostBitLockerPINRequest, error) {
	svc.authz.SkipAuthorization(ctx)
	return false, nil, nil
}

////////////////////////////////////////////////////////////////////////////////
// Orbit collects the submitted PIN
////////////////////////////////////////////////////////////////////////////////

func getOrbitDiskEncryptionPINDetailsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	pin, requestUUID, err := svc.GetBitLockerPINForHost(ctx)
	if err != nil {
		return fleet.OrbitGetDiskEncryptionPINDetailsResponse{Err: err}, nil
	}
	return fleet.OrbitGetDiskEncryptionPINDetailsResponse{PIN: pin, RequestUUID: requestUUID}, nil
}

func (svc *Service) GetBitLockerPINForHost(ctx context.Context) (string, string, error) {
	svc.authz.SkipAuthorization(ctx)
	return "", "", fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Orbit reports whether it applied the PIN
////////////////////////////////////////////////////////////////////////////////

func postOrbitDiskEncryptionPINResultEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.OrbitPostDiskEncryptionPINResultRequest)
	if err := svc.SetBitLockerPINOutcome(ctx, req.RequestUUID, req.Outcome, req.ClientError); err != nil {
		return fleet.OrbitPostDiskEncryptionPINResultResponse{Err: err}, nil
	}
	return fleet.OrbitPostDiskEncryptionPINResultResponse{}, nil
}

func (svc *Service) SetBitLockerPINOutcome(
	ctx context.Context, requestUUID string, outcome fleet.BitLockerPINRequestStatus, clientError string,
) error {
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

// setBitLockerPINNotification tells the agent to collect a PIN the end user has submitted, after confirming the host still
// needs it. The caller only calls it while a PIN is waiting for a capable agent, which is rare and short-lived. Turning
// the requirement off, or another session setting a PIN first, drops the submission rather than applying it late.
func (svc *Service) setBitLockerPINNotification(ctx context.Context, notifs *fleet.OrbitConfigNotifications, host *fleet.Host) error {
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() {
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
