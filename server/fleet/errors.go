package fleet

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/rs/zerolog"
)

var (
	ErrNoContext               = errors.New("context key not set")
	ErrPasswordResetRequired   = platform_http.ErrPasswordResetRequired
	ErrMissingLicense          = &licenseError{}
	ErrMDMNotConfigured        = &MDMNotConfiguredError{}
	ErrWindowsMDMNotConfigured = &WindowsMDMNotConfiguredError{}
	ErrAndroidMDMNotConfigured = &AndroidMDMNotConfiguredError{}
	ErrNotConfigured           = &NotConfiguredError{}

	MDMNotConfiguredMessage                      = "MDM features aren't turned on in Fleet. For more information about setting up MDM, please visit https://fleetdm.com/docs/using-fleet"
	WindowsMDMNotConfiguredMessage               = "Windows MDM isn't turned on. For more information about setting up MDM, please visit https://fleetdm.com/learn-more-about/windows-mdm"
	AndroidMDMNotConfiguredMessage               = "Android MDM isn't turned on. For more information about setting up MDM, please visit https://fleetdm.com/learn-more-about/how-to-connect-android-enterprise"
	AppleMDMNotConfiguredMessage                 = "macOS MDM isn't turned on. Visit https://fleetdm.com/docs/using-fleet to learn how to turn on MDM."
	AppleABMDefaultTeamDeprecatedMessage         = "mdm.apple_bm_default_team has been deprecated. Please use the new mdm.apple_business_manager key documented here: https://fleetdm.com/learn-more-about/apple-business-manager-gitops"
	CantTurnOffMDMForWindowsHostsMessage         = "Can't turn off MDM for Windows hosts."
	CantTurnOffMDMForPersonalHostsMessage        = "Couldn't turn off MDM. This command isn't available for personal hosts."
	CantWipePersonalHostsMessage                 = "Couldn't wipe. This command isn't available for personal hosts."
	CantLockPersonalHostsMessage                 = "Couldn't lock. This command isn't available for personal hosts."
	CantLockManualIOSIpadOSHostsMessage          = "Couldn't lock. This command isn't available for manually enrolled iOS/iPadOS hosts."
	CantDisableDiskEncryptionIfPINRequiredErrMsg = "Couldn't disable disk encryption, you need to disable the BitLocker PIN requirement first."
	CantEnablePINRequiredIfDiskEncryptionEnabled = "Couldn't enable BitLocker PIN requirement, you must enable disk encryption first."
	CantResendAppleDeclarationProfilesMessage    = "Can't resend declaration (DDM) profiles. Unlike configuration profiles (.mobileconfig), the host automatically checks in to get the latest DDM profiles."
	CantAddSoftwareConflictMessage               = "Couldn't add software. %s already has an installer available for the %s team."
)

// ErrWithStatusCode is an interface for errors that should set a specific HTTP
// status code.
type ErrWithStatusCode interface {
	error
	StatusCode() int
}

// ErrWithInternal is an alias for platform_http.ErrWithInternal.
type ErrWithInternal = platform_http.ErrWithInternal

// ErrWithLogFields is an alias for platform_http.ErrWithLogFields.
type ErrWithLogFields = platform_http.ErrWithLogFields

// ErrWithRetryAfter is an alias for platform_http.ErrWithRetryAfter.
type ErrWithRetryAfter = platform_http.ErrWithRetryAfter

// ErrWithIsClientError is an alias for platform_http.ErrWithIsClientError.
type ErrWithIsClientError = platform_http.ErrWithIsClientError

type invalidArgWithStatusError struct {
	InvalidArgumentError
	code int
}

func (e invalidArgWithStatusError) Status() int {
	if e.code == 0 {
		// 422 is the default code for invalid args
		return http.StatusUnprocessableEntity
	}
	return e.code
}

// ErrorUUIDer is an alias for platform_http.ErrorUUIDer.
type ErrorUUIDer = platform_http.ErrorUUIDer

// ErrorWithUUID is an alias for platform_http.ErrorWithUUID.
type ErrorWithUUID = platform_http.ErrorWithUUID

// InvalidArgumentError is the error returned when invalid data is presented to
// a service method. It is a client error.
type InvalidArgumentError struct {
	Errors []InvalidArgument

	ErrorWithUUID
}

// InvalidArgument is the details about a single invalid argument.
type InvalidArgument struct {
	name   string
	reason string
}

// NewInvalidArgumentError returns a InvalidArgumentError with at least
// one error.
func NewInvalidArgumentError(name, reason string) *InvalidArgumentError {
	var invalid InvalidArgumentError
	invalid.Append(name, reason)
	return &invalid
}

func (e InvalidArgumentError) IsClientError() bool {
	return true
}

func (e *InvalidArgumentError) Append(name, reason string) {
	e.Errors = append(e.Errors, InvalidArgument{
		name:   name,
		reason: reason,
	})
}

func (e *InvalidArgumentError) Appendf(name, reasonFmt string, args ...interface{}) {
	e.Append(name, fmt.Sprintf(reasonFmt, args...))
}

// WithStatus returns an error that combines the InvalidArgumentError
// with a custom status code.
func (e InvalidArgumentError) WithStatus(code int) error {
	return invalidArgWithStatusError{e, code}
}

func (e *InvalidArgumentError) HasErrors() bool {
	return len(e.Errors) != 0
}

// Error implements the error interface.
func (e InvalidArgumentError) Error() string {
	switch len(e.Errors) {
	case 0:
		return "validation failed"
	case 1:
		return fmt.Sprintf("validation failed: %s %s", e.Errors[0].name, e.Errors[0].reason)
	default:
		return fmt.Sprintf("validation failed: %s %s and %d other errors", e.Errors[0].name, e.Errors[0].reason,
			len(e.Errors))
	}
}

func (e InvalidArgumentError) Invalid() []map[string]string {
	var invalid []map[string]string
	for _, i := range e.Errors {
		invalid = append(invalid, map[string]string{"name": i.name, "reason": i.reason})
	}
	return invalid
}

// BadRequestError is an alias for platform_http.BadRequestError.
type BadRequestError = platform_http.BadRequestError

// AuthFailedError is an alias for platform_http.AuthFailedError.
type AuthFailedError = platform_http.AuthFailedError

// NewAuthFailedError is an alias for platform_http.NewAuthFailedError.
var NewAuthFailedError = platform_http.NewAuthFailedError

// AuthRequiredError is an alias for platform_http.AuthRequiredError.
type AuthRequiredError = platform_http.AuthRequiredError

// NewAuthRequiredError is an alias for platform_http.NewAuthRequiredError.
var NewAuthRequiredError = platform_http.NewAuthRequiredError

// AuthHeaderRequiredError is an alias for platform_http.AuthHeaderRequiredError.
type AuthHeaderRequiredError = platform_http.AuthHeaderRequiredError

// NewAuthHeaderRequiredError is an alias for platform_http.NewAuthHeaderRequiredError.
var NewAuthHeaderRequiredError = platform_http.NewAuthHeaderRequiredError

// PermissionError, set when user is authenticated, but not allowed to perform action
type PermissionError struct {
	message string

	ErrorWithUUID
}

func NewPermissionError(message string) *PermissionError {
	return &PermissionError{message: message}
}

func (e PermissionError) Error() string {
	return e.message
}

func (e PermissionError) PermissionError() []map[string]string {
	var forbidden []map[string]string
	return forbidden
}

// OTAForbiddenError is a special kind of forbidden error that intentionally
// exposes information about the error so it can be shown in iPad/iPhone native
// dialogs during OTA enrollment.
//
// I couldn't find any documentation but the way it works is:
//
// - if the response has a status code 403
// - and the body has a `message` field
//
// the content of `message` will be displayed to the end user.
type OTAForbiddenError struct {
	ErrorWithUUID
	InternalErr error
}

func (e OTAForbiddenError) Error() string {
	return "Couldn't install the profile. Invalid enroll secret. Please contact your IT admin."
}

func (e OTAForbiddenError) StatusCode() int {
	return http.StatusForbidden
}

func (e OTAForbiddenError) Internal() string {
	if e.InternalErr == nil {
		return ""
	}
	return e.InternalErr.Error()
}

// licenseError is returned when the application is not properly licensed.
type licenseError struct {
	ErrorWithUUID
}

func (e licenseError) Error() string {
	return "Requires Fleet Premium license"
}

func (e licenseError) StatusCode() int {
	return http.StatusPaymentRequired
}

// MDMNotConfiguredError is used when an MDM endpoint or resource is accessed
// without having MDM correctly configured.
type MDMNotConfiguredError struct{}

// Status implements the kithttp.StatusCoder interface so we can customize the
// HTTP status code of the response returning this error.
func (e *MDMNotConfiguredError) StatusCode() int {
	return http.StatusBadRequest
}

func (e *MDMNotConfiguredError) Error() string {
	return MDMNotConfiguredMessage
}

// WindowsMDMNotConfiguredError is used when an MDM endpoint or resource is accessed
// without having Windows MDM correctly configured.
type WindowsMDMNotConfiguredError struct{}

// Status implements the kithttp.StatusCoder interface so we can customize the
// HTTP status code of the response returning this error.
func (e *WindowsMDMNotConfiguredError) StatusCode() int {
	return http.StatusBadRequest
}

func (e *WindowsMDMNotConfiguredError) Error() string {
	return WindowsMDMNotConfiguredMessage
}

// AndroidMDMNotConfiguredError is used when an MDM endpoint or resource is accessed
// without having Android MDM correctly configured.
type AndroidMDMNotConfiguredError struct{}

// Status implements the kithttp.StatusCoder interface so we can customize the
// HTTP status code of the response returning this error.
func (e *AndroidMDMNotConfiguredError) StatusCode() int {
	return http.StatusBadRequest
}

func (e *AndroidMDMNotConfiguredError) Error() string {
	return AndroidMDMNotConfiguredMessage
}

// NotConfiguredError is a generic "not configured" error that can be used
// when expected configuration is missing.
type NotConfiguredError struct{}

func (e *NotConfiguredError) Error() string {
	return "not configured"
}

// GatewayError is an error type that generates a 502 or 504 status code.
type GatewayError struct {
	Message string
	err     error
	code    int

	ErrorWithUUID
}

// NewBadGatewayError returns a GatewayError with the message and
// error specified and that returns a 502 status code.
func NewBadGatewayError(message string, err error) *GatewayError {
	return &GatewayError{
		Message: message,
		err:     err,
		code:    http.StatusBadGateway,
	}
}

// NewGatewayTimeoutError returns a GatewayError with the message and
// error specified and that returns a 504 status code.
func NewGatewayTimeoutError(message string, err error) *GatewayError {
	return &GatewayError{
		Message: message,
		err:     err,
		code:    http.StatusGatewayTimeout,
	}
}

// StatusCode implements the kithttp.StatusCoder interface so we can customize the
// HTTP status code of the response returning this error.
func (e *GatewayError) StatusCode() int {
	return e.code
}

// Error returns the error message.
func (e *GatewayError) Error() string {
	msg := e.Message
	if e.err != nil {
		msg += ": " + e.err.Error()
	}
	return msg
}

// Error is an alias for platform_http.Error.
// It's meant to be used for errors that are related to fleet logic specifically.
type Error = platform_http.Error

const (
	// ErrNoRoleNeeded is the error number for valid role needed
	ErrNoRoleNeeded = 1
	// ErrNoOneAdminNeeded is the error number when all admins are about to be removed
	ErrNoOneAdminNeeded = 2
	// ErrNoUnknownTranslate is returned when an item type in the translate payload is unknown
	ErrNoUnknownTranslate = 3
	// ErrAPIOnlyRole is returned when a selected role for a user is for API only users.
	ErrAPIOnlyRole = 4
)

// NewError returns a fleet error with the code and message specified
func NewError(code int, message string) error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorf returns a fleet error with the code, and message formatted
// based on the format string and args specified
func NewErrorf(code int, format string, args ...interface{}) error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// UserMessageError is an alias for platform_http.UserMessageError.
type UserMessageError = platform_http.UserMessageError

// NewUserMessageError is an alias for platform_http.NewUserMessageError.
var NewUserMessageError = platform_http.NewUserMessageError

// IsJSONUnknownFieldError returns true if err is a JSON unknown field error.
// There is no exported type or value for this error, so we have to match the
// error message.
func IsJSONUnknownFieldError(err error) bool {
	return platform_http.IsJSONUnknownFieldError(err)
}

// GetJSONUnknownField returns the unknown field name from a JSON unknown field error.
func GetJSONUnknownField(err error) *string {
	return platform_http.GetJSONUnknownField(err)
}

// Cause returns the root error in err's chain.
var Cause = platform_http.Cause

// FleetdError is an error that can be reported by any of the fleetd
// components.
type FleetdError struct {
	ErrorSource         string         `json:"error_source"`
	ErrorSourceVersion  string         `json:"error_source_version"`
	ErrorTimestamp      time.Time      `json:"error_timestamp"`
	ErrorMessage        string         `json:"error_message"`
	ErrorAdditionalInfo map[string]any `json:"error_additional_info"`
	// Vital errors are always reported to Fleet server.
	Vital bool `json:"vital"`
}

// Error implements the error interface
func (fe FleetdError) Error() string {
	return fe.ErrorMessage
}

// MarshalZerologObject implements `zerolog.LogObjectMarshaler` so all details
// about the error can be logged by the components that use zerolog (Orbit,
// Fleet Desktop)
func (fe FleetdError) MarshalZerologObject(e *zerolog.Event) {
	e.Bool("vital", fe.Vital)
	e.Str("error_source", fe.ErrorSource)
	e.Str("error_source_version", fe.ErrorSourceVersion)
	e.Time("error_timestamp", fe.ErrorTimestamp)
	e.Str("error_message", fe.ErrorMessage)
	e.Interface("error_additional_info", fe.ErrorAdditionalInfo)
}

// ToMap returns a map representation of the error
func (fe FleetdError) ToMap() map[string]any {
	return map[string]any{
		"vital":                 fe.Vital,
		"error_source":          fe.ErrorSource,
		"error_source_version":  fe.ErrorSourceVersion,
		"error_timestamp":       fe.ErrorTimestamp,
		"error_message":         fe.ErrorMessage,
		"error_additional_info": fe.ErrorAdditionalInfo,
	}
}

// OrbitError is used for orbit endpoints, to return an error message along
// with a failed request's response.
type OrbitError struct {
	Message string
	code    int
}

// Error implements the error interface for the OrbitError.
func (e OrbitError) Error() string {
	return e.Message
}

// StatusCode implements the ErrWithStatusCode interface for the OrbitError.
func (e OrbitError) StatusCode() int {
	if e.code == 0 {
		return http.StatusInternalServerError
	}
	return e.code
}

func NewOrbitIDPAuthRequiredError() *OrbitError {
	return &OrbitError{
		Message: "END_USER_AUTH_REQUIRED",
		code:    http.StatusUnauthorized,
	}
}

// Messages that may be surfaced by the server or the fleetctl client.
const (
	// Hosts, general
	HostNotFoundErrMsg           = "Host doesn't exist. Make sure you provide a valid hostname, UUID, or serial number. Learn more about host identifiers: https://fleetdm.com/learn-more-about/host-identifiers"
	NoHostsTargetedErrMsg        = "No hosts targeted. Make sure you provide a valid hostname, UUID, or serial number. Learn more about host identifiers: https://fleetdm.com/learn-more-about/host-identifiers"
	TargetedHostsDontExistErrMsg = "One or more targeted hosts don't exist. Make sure you provide a valid hostname, UUID, or serial number. Learn more about host identifiers: https://fleetdm.com/learn-more-about/host-identifiers"

	// Scripts
	RunScriptInvalidTypeErrMsg             = "File type not supported. Only .sh (Bash) and .ps1 (PowerShell) file types are allowed."
	RunScriptHostOfflineErrMsg             = "Script can't run on offline host."
	RunScriptForbiddenErrMsg               = "You don't have the right permissions in Fleet to run the script."
	RunScriptAlreadyRunningErrMsg          = "A script is already running on this host. Please wait about 5 minutes to let it finish."
	RunScriptHostTimeoutErrMsg             = "Fleet didn't hear back from the host in under 5 minutes (timeout for live scripts). Fleet doesn't know if the script ran because it didn't receive the result. Go to Fleet and check Host details > Activities to see script results."
	RunScriptScriptsDisabledGloballyErrMsg = "Running scripts is disabled in organization settings."
	RunScriptDisabledErrMsg                = "Scripts are disabled for this host. To run scripts, deploy the fleetd agent with scripts enabled."
	RunScriptsOrbitDisabledErrMsg          = "Couldn't run script. To run a script, deploy the fleetd agent with --enable-scripts."
	RunScriptAsyncScriptEnqueuedMsg        = "Script is running or will run when the host comes online."
	RunScripSavedMaxLenErrMsg              = "Script is too large. It's limited to 500,000 characters (approximately 10,000 lines)."
	RunScripUnsavedMaxLenErrMsg            = "Script is too large. It's limited to 10,000 characters (approximately 125 lines)."
	RunScriptGatewayTimeoutErrMsg          = "Gateway timeout. Fleet didn't hear back from the host and doesn't know if the script ran. Please make sure your load balancer timeout isn't shorter than the Fleet server timeout."

	// Software
	InstallSoftwarePersonalAppleDeviceErrMsg = "Couldn't install. Currently, software install isn't supported on personal (BYOD) iOS and iPadOS hosts."

	// End user authentication
	EndUserAuthDEPWebURLConfiguredErrMsg = `End user authentication can't be configured when the configured automatic enrollment (DEP) profile specifies a configuration_web_url.` // #nosec G101

	// Labels
	InvalidLabelSpecifiedErrMsg = "Invalid label name(s):"

	// Config
	InvalidServerURLMsg = `Fleet server URL must use “https” or “http”.`

	// macOS setup experience
	BootstrapPkgNotDistributionErrMsg = "Couldn’t add. Bootstrap package must be a distribution package. Learn more at: https://fleetdm.com/learn-more-about/macos-distribution-packages"

	// NDES/SCEP validation
	MultipleSCEPPayloadsErrMsg          = "Add only one SCEP payload."
	SCEPVariablesNotInSCEPPayloadErrMsg = "Variables prefixed with \"$FLEET_VAR_SCEP_\", \"$FLEET_VAR_CUSTOM_SCEP_\", \"$FLEET_VAR_NDES_SCEP\" and \"$FLEET_VAR_SMALLSTEP_\" must only be in the SCEP payload."

	// Invalid list options combinations
	FilterTitlesByPlatformNeedsTeamIdErrMsg = "The 'platform' and 'team_id' parameters must be used together to filter the software available for install."
)

// Error message variables
var (
	NDESSCEPVariablesMissingErrMsg         = fmt.Sprintf("SCEP profile for NDES certificate authority requires: $FLEET_VAR_%s, $FLEET_VAR_%s, and $FLEET_VAR_%s variables.", FleetVarNDESSCEPChallenge, FleetVarNDESSCEPProxyURL, FleetVarSCEPRenewalID)
	SCEPRenewalIDWithoutURLChallengeErrMsg = "Variable \"$FLEET_VAR_" + string(FleetVarSCEPRenewalID) + "\" can't be used if variables for SCEP URL and Challenge are not specified."
)

// ConflictError is used to indicate a conflict, such as a UUID conflict in the DB.
type ConflictError struct {
	Message string
}

// Error implements the error interface for the ConflictError.
func (e ConflictError) Error() string {
	return e.Message
}

// StatusCode implements the kithttp.StatusCoder interface.
func (e ConflictError) StatusCode() int {
	return http.StatusConflict
}

// IsConflict implements the conflict interface for middleware compatibility
func (e ConflictError) IsConflict() bool {
	return true
}

// Errorer is an alias for platform_http.Errorer.
type Errorer = platform_http.Errorer

type VPPIconAvailable struct {
	IconURL string
}

func (e *VPPIconAvailable) Error() string {
	return fmt.Sprintf("VPP icon available at: %s", e.IconURL)
}
