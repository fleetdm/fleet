package fleet

import "net/http"

// MaxAutofillConfigProfileInstructionsLength is the same limit fleetdm.com
// enforces, checked here so an oversized request never leaves Fleet.
const MaxAutofillConfigProfileInstructionsLength = 2000

const (
	AutofillConfigProfilePlatformApple   = "apple"
	AutofillConfigProfilePlatformWindows = "windows"
)

const (
	AutofillConfigProfileTypeMobileconfig = "mobileconfig"
	AutofillConfigProfileTypeDeclaration  = "declaration"
	AutofillConfigProfileTypeWindows      = "windows"
)

const (
	AutofillConfigProfileNameDescriptionErrorMessage = "Couldn't generate configuration profile name and description"
	AutofillConfigProfileContentErrorMessage         = "Couldn't generate configuration profile"
)

const (
	AutofillConfigProfileLimitReachedReason      = "You have reached the monthly limit for AI-generated configuration profiles on Fleet Free."
	AutofillConfigProfileMissingPrivateKeyReason = "Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key"
)

// AutofilledConfigProfile is a generated profile that passed the checks Fleet
// runs when it's added. Name is set only for a .mobileconfig, whose
// PayloadDisplayName is the name Fleet would give it.
type AutofilledConfigProfile struct {
	Type          string  `json:"type"`
	Name          *string `json:"name"`
	Content       string  `json:"content"`
	DeliveryNotes string  `json:"delivery_notes"`
}

// AutofillConfigProfileError is an error whose message and field the autofill
// API documents, which neither InvalidArgumentError ("Validation Failed") nor
// AutofillError (always "base") can produce.
type AutofillConfigProfileError struct {
	Message string
	Field   string
	Reason  string
	// MoreReasons are reported under Field after Reason, one error each.
	MoreReasons []string
	InternalErr error
	// Status defaults to 422.
	Status int
	// RetryAfterSeconds, when set, is sent as the Retry-After header.
	RetryAfterSeconds int
}

func (e *AutofillConfigProfileError) Error() string {
	return e.Reason
}

func (e *AutofillConfigProfileError) StatusCode() int {
	if e.Status != 0 {
		return e.Status
	}
	return http.StatusUnprocessableEntity
}

func (e *AutofillConfigProfileError) Internal() string {
	if e.InternalErr == nil {
		return ""
	}
	return e.InternalErr.Error()
}

// IsClientError keeps declines and failed checks out of error-level logs,
// while a fleetdm.com failure, which carries its cause, still reaches them.
func (e *AutofillConfigProfileError) IsClientError() bool {
	return e.InternalErr == nil
}
