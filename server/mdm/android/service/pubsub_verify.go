package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"google.golang.org/api/androidmanagement/v1"
)

// A notification that would reset, unenroll or rebind a host, or finish a command, is checked
// against AMAPI's current record of the device or operation before Fleet acts on it, so a delayed,
// redelivered or out-of-order notification cannot act on state that has since changed. One that
// does not match is acked (HTTP 200) and dropped, and the reason is only logged.
//
// A dropped DELETED is picked up by the device reconcile cron and a dropped command result by the
// command reconcile cron. A dropped re-enrollment is not recovered, so its checks only reject what
// a current AMAPI notification cannot look like.

func errUnverifiedPubSub() error {
	return fleet.NewInvalidArgumentError("message", "Android PubSub notification was not processed").
		WithStatus(http.StatusOK)
}

func (svc *Service) setAMAPIClientSecret(ctx context.Context) error {
	secret, err := svc.getClientAuthenticationSecret(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting client authentication secret")
	}
	_ = svc.androidAPIClient.SetAuthenticationSecret(secret)
	return nil
}

// enterpriseDevicesPrefix returns the resource-name prefix of this Fleet's enterprise devices.
// Device names from a notification must be checked against it: the proxy answers 404 for an
// enterprise it does not know, which would otherwise read as "device gone".
func (svc *Service) enterpriseDevicesPrefix(ctx context.Context) (string, error) {
	enterprise, err := svc.ds.GetEnterprise(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting android enterprise")
	}
	return enterprise.Name() + "/devices/", nil
}

// getAMAPIDevice returns the device as AMAPI knows it, or nil if AMAPI has no such device.
func (svc *Service) getAMAPIDevice(ctx context.Context, deviceName string) (*androidmanagement.Device, error) {
	if err := svc.setAMAPIClientSecret(ctx); err != nil {
		return nil, err
	}
	// No retry on 429: the retry client waits out the quota window, longer than Pub/Sub's ack
	// deadline, so Pub/Sub would redeliver while this delivery is still running. Failing lets
	// Pub/Sub's own backoff retry.
	device, err := svc.androidAPIClient.EnterprisesDevicesGet(androidmgmt.WithoutRetry(ctx), deviceName)
	switch {
	case androidmgmt.IsNotFoundError(err):
		return nil, nil
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "get android device from AMAPI")
	}
	return device, nil
}

// verifyAMAPIDevice checks that the notification's device is a live device of this Fleet's
// enterprise with the same hardware identity. With checkEnrollmentTime, the enrollment time
// must also match, tying the notification to the device's current enrollment.
func (svc *Service) verifyAMAPIDevice(ctx context.Context, device *androidmanagement.Device, checkEnrollmentTime bool) error {
	prefix, err := svc.enterpriseDevicesPrefix(ctx)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(device.Name, prefix) {
		return svc.rejectPubSub(ctx, device, "device is not in this enterprise")
	}
	amapiDevice, err := svc.getAMAPIDevice(ctx, device.Name)
	if err != nil {
		return err
	}
	switch {
	case amapiDevice == nil:
		return svc.rejectPubSub(ctx, device, "device not found")
	// AMAPI returns the canonical name; a different one means the lookup resolved a name that is
	// not the device's own (e.g. one with extra characters the proxy drops).
	case amapiDevice.Name != device.Name:
		return svc.rejectPubSub(ctx, device, "device name differs")
	case strings.EqualFold(amapiDevice.State, string(android.DeviceStateDeleted)):
		return svc.rejectPubSub(ctx, device, "device is deleted")
	case amapiDevice.HardwareInfo == nil || getAndroidHostKey(amapiDevice) != getAndroidHostKey(device):
		return svc.rejectPubSub(ctx, device, "hardware identity differs")
	case checkEnrollmentTime && !sameTimestamp(amapiDevice.EnrollmentTime, device.EnrollmentTime):
		return svc.rejectPubSub(ctx, device, "enrollment time differs")
	}
	return nil
}

// sameTimestamp compares two RFC3339 timestamps as instants, since AMAPI may render the same
// time with a different number of fractional digits.
func sameTimestamp(a, b string) bool {
	if a == b {
		return true
	}
	ta, errA := time.Parse(time.RFC3339Nano, a)
	tb, errB := time.Parse(time.RFC3339Nano, b)
	return errA == nil && errB == nil && ta.Equal(tb)
}

// verifyReenrollment checks an ENROLLMENT notification for a host Fleet already knows about,
// since processing it clears the host's pending commands, including a queued wipe.
func (svc *Service) verifyReenrollment(ctx context.Context, device *androidmanagement.Device) error {
	return svc.verifyAMAPIDevice(ctx, device, true)
}

// verifyDeviceChange checks a STATUS_REPORT that names a different AMAPI device than the one
// Fleet has for the host. Processing it stores the new device ID, so it is only accepted for a
// device AMAPI has with the same hardware identity.
func (svc *Service) verifyDeviceChange(ctx context.Context, host *fleet.AndroidHost, device *androidmanagement.Device) error {
	deviceID, err := svc.getDeviceID(ctx, device)
	if err != nil {
		return svc.rejectPubSub(ctx, device, "invalid device name")
	}
	if host.Device != nil && host.Device.DeviceID == deviceID {
		return nil
	}
	return svc.verifyAMAPIDevice(ctx, device, false)
}

// verifyNewHostDevice checks an enrollment that would create a host for a device ID Fleet already
// has, since creating the host moves that device record, and with it the device's commands and
// policy, to the new host.
func (svc *Service) verifyNewHostDevice(ctx context.Context, device *androidmanagement.Device) error {
	deviceID, err := svc.getDeviceID(ctx, device)
	if err != nil {
		return svc.rejectPubSub(ctx, device, "invalid device name")
	}
	exists, err := svc.fleetDS.AndroidDeviceExistsByDeviceID(ctxdb.RequirePrimary(ctx, true), deviceID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking for an existing android device")
	}
	if !exists {
		return nil
	}
	return svc.verifyAMAPIDevice(ctx, device, false)
}

// verifyDeviceDeleted checks a DELETED notification before the host is unenrolled. It must be for
// the host's current AMAPI device, and AMAPI must no longer have that device as an active one.
func (svc *Service) verifyDeviceDeleted(ctx context.Context, host *fleet.AndroidHost, device *androidmanagement.Device) error {
	if host.Device == nil || host.Device.DeviceID == "" {
		return svc.rejectPubSub(ctx, device, "host has no current device")
	}
	prefix, err := svc.enterpriseDevicesPrefix(ctx)
	if err != nil {
		return err
	}
	currentName := prefix + host.Device.DeviceID
	if device.Name != currentName {
		return svc.rejectPubSub(ctx, device, "device is not the host's current device")
	}
	amapiDevice, err := svc.getAMAPIDevice(ctx, currentName)
	if err != nil {
		return err
	}
	if amapiDevice != nil && !strings.EqualFold(amapiDevice.State, string(android.DeviceStateDeleted)) {
		return svc.rejectPubSub(ctx, device, "device is still active")
	}
	return nil
}

func (svc *Service) rejectPubSub(ctx context.Context, device *androidmanagement.Device, reason string) error {
	var esID string
	if device.HardwareInfo != nil {
		esID = device.HardwareInfo.EnterpriseSpecificId
	}
	svc.logger.WarnContext(ctx, "dropping Android PubSub notification that does not match AMAPI",
		"reason", reason, "device.name", device.Name, "device.enterpriseSpecificId", esID)
	return errUnverifiedPubSub()
}

// amapiDeviceGetter returns the device as AMAPI knows it, or nil if AMAPI has no such device.
type amapiDeviceGetter func(ctx context.Context, deviceName string) (*androidmanagement.Device, error)

// isCommandForCurrentDevice reports whether a command was issued to the host's current AMAPI
// device. Acknowledged commands survive a re-enrollment, so a redelivered notification for one
// issued to an earlier enrollment must not act on the re-enrolled host.
func isCommandForCurrentDevice(ctx context.Context, ds fleet.Datastore, cmd *android.MDMAndroidCommand) (bool, error) {
	// Read from the primary: a replica that hasn't caught up with a re-enrollment would still
	// return the previous device ID.
	host, err := ds.AndroidHostLiteByHostUUID(ctxdb.RequirePrimary(ctx, true), cmd.HostUUID)
	switch {
	case fleet.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, ctxerr.Wrap(ctx, err, "get android host for command")
	case host == nil || host.Device == nil:
		return false, nil
	}
	return deviceIDFromOperationName(cmd.OperationName) == host.Device.DeviceID, nil
}

// isCommandDeviceGone reports whether AMAPI no longer has the command's device as an active one.
func isCommandDeviceGone(ctx context.Context, getDevice amapiDeviceGetter, cmd *android.MDMAndroidCommand) (bool, error) {
	deviceName, _, _ := strings.Cut(cmd.OperationName, "/operations/")
	amapiDevice, err := getDevice(ctx, deviceName)
	if err != nil {
		return false, err
	}
	return amapiDevice == nil || strings.EqualFold(amapiDevice.State, string(android.DeviceStateDeleted)), nil
}

// isWipeOfGoneCurrentDevice reports whether cmd is a WIPE of the host's current device and AMAPI no
// longer has that device. It is what shows a wipe finished: AMAPI never marks a WIPE operation done,
// because the device is erased before it can report back. A wipe of an earlier device doesn't count,
// so it can't unenroll the host from the device it has moved to.
func isWipeOfGoneCurrentDevice(ctx context.Context, ds fleet.Datastore, getDevice amapiDeviceGetter, cmd *android.MDMAndroidCommand) (bool, error) {
	if cmd.CommandType != string(android.MDMAndroidCommandTypeWipe) {
		return false, nil
	}
	current, err := isCommandForCurrentDevice(ctx, ds, cmd)
	if err != nil || !current {
		return false, err
	}
	return isCommandDeviceGone(ctx, getDevice, cmd)
}

// verifiedOperation returns AMAPI's record of a finished command operation, to be used in place of
// the notification's copy. It returns nil (and the caller acks without changing the command) when
// AMAPI reports the operation is not done, unless it is a WIPE whose device is gone, which is
// returned as done.
//
// AMAPI drops operations it has finished with, and has none once the device is gone, so a missing
// operation can't be checked and the notification's copy is used, with two exceptions:
//   - when the device is gone, only for a command issued to the host's current device, since a
//     result for an earlier device must not act on the re-enrolled host;
//   - a WIPE can't have finished while its device is still active, so an error is returned and
//     Pub/Sub redelivers once AMAPI's state settles.
func (svc *Service) verifiedOperation(ctx context.Context, cmd *android.MDMAndroidCommand, op *androidmanagement.Operation) (*androidmanagement.Operation, error) {
	if err := svc.setAMAPIClientSecret(ctx); err != nil {
		return nil, err
	}
	amapiOp, err := svc.androidAPIClient.EnterprisesDevicesOperationsGet(ctx, op.Name)
	switch {
	case androidmgmt.IsNotFoundError(err):
		gone, err := isCommandDeviceGone(ctx, svc.getAMAPIDevice, cmd)
		if err != nil {
			return nil, err
		}
		if gone {
			current, err := isCommandForCurrentDevice(ctx, svc.fleetDS, cmd)
			if err != nil {
				return nil, err
			}
			if !current {
				svc.logger.WarnContext(ctx, "dropping Android PubSub COMMAND for a device that is gone and is not the host's current one",
					"operation_name", op.Name)
				return nil, nil
			}
			return op, nil
		}
		if cmd.CommandType == string(android.MDMAndroidCommandTypeWipe) {
			return nil, ctxerr.Errorf(ctx, "android wipe operation %s not found in AMAPI while its device is active, retry", op.Name)
		}
		return op, nil
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "get android command operation from AMAPI")
	case !amapiOp.Done:
		wiped, err := isWipeOfGoneCurrentDevice(ctx, svc.fleetDS, svc.getAMAPIDevice, cmd)
		if err != nil {
			return nil, err
		}
		if wiped {
			return &androidmanagement.Operation{Name: op.Name, Done: true, Metadata: amapiOp.Metadata}, nil
		}
		svc.logger.WarnContext(ctx, "dropping Android PubSub COMMAND for an operation AMAPI reports as not done",
			"operation_name", op.Name)
		return nil, nil
	}
	return amapiOp, nil
}
