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

	// 403-class reasons: certificate or topic problems that retrying a
	// device token cannot fix.
	APNSReasonBadCertificate            = "BadCertificate"
	APNSReasonBadCertificateEnvironment = "BadCertificateEnvironment"
	APNSReasonExpiredProviderToken      = "ExpiredProviderToken"
	APNSReasonForbidden                 = "Forbidden"
	APNSReasonInvalidProviderToken      = "InvalidProviderToken"
	APNSReasonMissingProviderToken      = "MissingProviderToken"
	APNSReasonBadTopic                  = "BadTopic"
	APNSReasonTopicDisallowed           = "TopicDisallowed"
)

// isPermanentAPNSRejection reports whether a per-device push error is a
// rejection that retrying cannot fix: dead or mismatched tokens, or
// certificate/topic problems. Everything else (429, 5xx, transport errors,
// GOAWAY) is treated as transient.
func isPermanentAPNSRejection(err error) bool {
	switch APNSReason(err) {
	case APNSReasonUnregistered, APNSReasonExpiredToken, APNSReasonBadDeviceToken, APNSReasonDeviceTokenNotForTopic,
		APNSReasonBadCertificate, APNSReasonBadCertificateEnvironment, APNSReasonExpiredProviderToken,
		APNSReasonForbidden, APNSReasonInvalidProviderToken, APNSReasonMissingProviderToken,
		APNSReasonBadTopic, APNSReasonTopicDisallowed:
		return true
	}
	return false
}

// APNSReason extracts the APNs rejection reason (e.g. "Unregistered",
// "BadDeviceToken") from a push response error, or "" if the error does not
// carry one (transport failures, non-APNs errors).
func APNSReason(err error) string {
	if jsonErr, ok := errors.AsType[*nanopush.JSONPushError](err); ok {
		return jsonErr.Reason
	}
	return ""
}
