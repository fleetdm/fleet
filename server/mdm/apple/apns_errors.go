package apple_mdm

import (
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/nanopush"
)

// APNs rejection reasons from the JSON body of a failed push request, as
// documented in Apple's "Handling notification responses from APNs".
const (
	APNSReasonUnregistered           = "Unregistered"
	APNSReasonExpiredToken           = "ExpiredToken"
	APNSReasonBadDeviceToken         = "BadDeviceToken"
	APNSReasonDeviceTokenNotForTopic = "DeviceTokenNotForTopic"
	APNSReasonTooManyRequests        = "TooManyRequests"
)

// APNSReason extracts the APNs rejection reason (e.g. "Unregistered",
// "BadDeviceToken") from a push response error, or "" if the error does not
// carry one (transport failures, non-APNs errors).
func APNSReason(err error) string {
	if jsonErr, ok := errors.AsType[*nanopush.JSONPushError](err); ok {
		return jsonErr.Reason
	}
	return ""
}
