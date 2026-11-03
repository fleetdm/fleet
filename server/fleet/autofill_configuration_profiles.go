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

// AutofilledConfigProfile is a generated profile that passed the checks Fleet
// runs when it's added. Name is set only for a .mobileconfig, whose
// PayloadDisplayName is the name Fleet would give it.
type AutofilledConfigProfile struct {
	Type          string  `json:"type"`
	Name          *string `json:"name"`
	Content       string  `json:"content"`
	DeliveryNotes string  `json:"delivery_notes"`
}

// AutofillConfigProfileError is a 422 whose message and field the autofill
// API documents, which neither InvalidArgumentError ("Validation Failed") nor
// AutofillError (always "base") can produce.
type AutofillConfigProfileError struct {
	Message     string
	Field       string
	Reason      string
	InternalErr error
}

func (e *AutofillConfigProfileError) Error() string {
	return e.Reason
}

func (e *AutofillConfigProfileError) StatusCode() int {
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
