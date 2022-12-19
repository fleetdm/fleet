package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	"github.com/micromdm/nanodep/godep"
	nanohttp "github.com/micromdm/nanomdm/http"
	"github.com/micromdm/nanomdm/mdm"
)

// MDMAppleEnrollmentType is the type for Apple MDM enrollments.
type MDMAppleEnrollmentType string

const (
	// MDMAppleEnrollmentTypeAutomatic is the value for automatic enrollments.
	MDMAppleEnrollmentTypeAutomatic MDMAppleEnrollmentType = "automatic"
	// MDMAppleEnrollmentTypeManual is the value for manual enrollments.
	MDMAppleEnrollmentTypeManual MDMAppleEnrollmentType = "manual"
)

// MDMAppleEnrollmentProfilePayload contains the data necessary to create
// an enrollment profile in Fleet.
type MDMAppleEnrollmentProfilePayload struct {
	// Type is the type of the enrollment.
	Type MDMAppleEnrollmentType `json:"type"`
	// DEPProfile is the JSON object with the following Apple-defined fields:
	// https://developer.apple.com/documentation/devicemanagement/profile
	//
	// DEPProfile is nil when Type is MDMAppleEnrollmentTypeManual.
	DEPProfile *json.RawMessage `json:"dep_profile"`
	// Token should be auto-generated.
	Token string `json:"-"`
}

// MDMAppleEnrollmentProfile represents an Apple MDM enrollment profile in Fleet.
// Such enrollment profiles are used to enroll Apple devices to Fleet.
type MDMAppleEnrollmentProfile struct {
	// ID is the unique identifier of the enrollment in Fleet.
	ID uint `json:"id" db:"id"`
	// Token is a random identifier for an enrollment. Currently as the authentication
	// token to protect access to the enrollment.
	Token string `json:"token" db:"token"`
	// Type is the type of the enrollment.
	Type MDMAppleEnrollmentType `json:"type" db:"type"`
	// DEPProfile is the JSON object with the following Apple-defined fields:
	// https://developer.apple.com/documentation/devicemanagement/profile
	//
	// DEPProfile is nil when Type is MDMAppleEnrollmentTypeManual.
	DEPProfile *json.RawMessage `json:"dep_profile" db:"dep_profile"`
	// EnrollmentURL is the URL where an enrollement is served.
	EnrollmentURL string `json:"enrollment_url" db:"-"`

	UpdateCreateTimestamps
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleEnrollmentProfile) AuthzType() string {
	return "mdm_apple_enrollment_profile"
}

// MDMAppleCommandResult holds the result of a command execution provided by the target device.
type MDMAppleCommandResult struct {
	// ID is the enrollment ID. This should be the same as the device ID.
	ID string `json:"id" db:"id"`
	// CommandUUID is the unique identifier of the command.
	CommandUUID string `json:"command_uuid" db:"command_uuid"`
	// Status is the command status. One of Acknowledged, Error, or NotNow.
	Status string `json:"status" db:"status"`
	// Result is the original command result XML plist. If the status is Error, it will include the
	// ErrorChain key with more information.
	Result []byte `json:"result" db:"result"`
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleCommandResult) AuthzType() string {
	return "mdm_apple_command_result"
}

// MDMAppleInstaller holds installer packages for Apple devices.
type MDMAppleInstaller struct {
	// ID is the unique identifier of the installer in Fleet.
	ID uint `json:"id" db:"id"`
	// Name is the name of the installer (usually the package file name).
	Name string `json:"name" db:"name"`
	// Size is the size of the installer package.
	Size int64 `json:"size" db:"size"`
	// Manifest is the manifest of the installer. Generated from the installer
	// contents and ready to use in `InstallEnterpriseApplication` commands.
	Manifest string `json:"manifest" db:"manifest"`
	// Installer is the actual installer contents.
	Installer []byte `json:"-" db:"installer"`
	// URLToken is a random token used for authentication to protect access to installers.
	// Applications deployede via InstallEnterpriseApplication must be publicly accessible,
	// this hard to guess token provides some protection.
	URLToken string `json:"url_token" db:"url_token"`
	// URL is the full URL where the installer is served.
	URL string `json:"url"`
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleInstaller) AuthzType() string {
	return "mdm_apple_installer"
}

// MDMAppleDevice represents an MDM enrolled Apple device.
type MDMAppleDevice struct {
	// ID is the device hardware UUID.
	ID string `json:"id" db:"id"`
	// SerialNumber is the serial number of the Apple device.
	SerialNumber string `json:"serial_number" db:"serial_number"`
	// Enabled indicates whether the device is currently enrolled.
	// It's set to false when a device unenrolls from Fleet.
	Enabled bool `json:"enabled" db:"enabled"`
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleDevice) AuthzType() string {
	return "mdm_apple_device"
}

// MDMAppleDEPDevice represents an Apple device in Apple Business Manager (ABM).
type MDMAppleDEPDevice struct {
	godep.Device
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleDEPDevice) AuthzType() string {
	return "mdm_apple_dep_device"
}

// These following types are copied from nanomdm.

// EnrolledAPIResult is a per-enrollment API result.
type EnrolledAPIResult struct {
	PushError    string `json:"push_error,omitempty"`
	PushResult   string `json:"push_result,omitempty"`
	CommandError string `json:"command_error,omitempty"`
}

// EnrolledAPIResults is a map of enrollments to a per-enrollment API result.
type EnrolledAPIResults map[string]*EnrolledAPIResult

// CommandEnqueueResult is the result of a command execution on enrolled Apple devices.
type CommandEnqueueResult struct {
	// Status is the status of the command.
	Status EnrolledAPIResults `json:"status,omitempty"`
	// NoPush indicates whether the command was issued with no_push.
	// If this is true, then Fleet won't send a push notification to devices.
	NoPush bool `json:"no_push,omitempty"`
	// PushError indicates the error when trying to send push notification
	// to target devices.
	PushError string `json:"push_error,omitempty"`
	// CommandError holds the error when enqueueing the command.
	CommandError string `json:"command_error,omitempty"`
	// CommandUUID is the unique identifier for the command.
	CommandUUID string `json:"command_uuid,omitempty"`
	// RequestType is the name of the command.
	RequestType string `json:"request_type,omitempty"`
}

// MDMAppleCommand represents an Apple MDM command.
type MDMAppleCommand struct {
	*mdm.Command
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleCommand) AuthzType() string {
	return "mdm_apple_command"
}

// MDMAppleHostDetails represents the device identifiers used to ingest an MDM device as a Fleet
// host pending enrollment.
// See also https://developer.apple.com/documentation/devicemanagement/authenticaterequest.
type MDMAppleHostDetails struct {
	SerialNumber string
	UDID         string
	Model        string
}

// MDMHostIngester represents the interface used to ingest an MDM device as a Fleet host pending enrollment.
type MDMHostIngester interface {
	Ingest(context.Context, *http.Request) error
}

// MDMAppleHostIngester implements the MDMHostIngester interface in connection with Apple MDM services.
type MDMAppleHostIngester struct {
	ds     Datastore
	logger kitlog.Logger
}

// NewMDMAppleHostIngester returns a new instance of an MDMAppleHostIngester.
func NewMDMAppleHostIngester(ds Datastore, logger kitlog.Logger) *MDMAppleHostIngester {
	return &MDMAppleHostIngester{ds: ds, logger: logger}
}

// Ingest handles incoming http requests that follow Apple's MDM checkin protocol. For valid
// checkin requests, Ingest decodes the XML body and ingests new host details into the associated
// datastore. See also https://developer.apple.com/documentation/devicemanagement/check-in.
func (ingester *MDMAppleHostIngester) Ingest(ctx context.Context, r *http.Request) error {
	if isMDMAppleCheckinReq(r) {
		host := MDMAppleHostDetails{}
		if err := decodeMDMAppleCheckinReq(r, &host); err != nil {
			return fmt.Errorf("decode checkin request: %w", err)
		}
		if err := ingester.ds.IngestMDMAppleDeviceFromCheckin(ctx, host); err != nil {
			return err
		}
	}
	return nil
}

func isMDMAppleCheckinReq(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	fmt.Println("content type in func", contentType)
	if strings.HasPrefix(contentType, "application/x-apple-aspen-mdm-checkin") {
		return true
	}
	return false
}

func decodeMDMAppleCheckinReq(r *http.Request, dest *MDMAppleHostDetails) error {
	req := *r
	bodyBytes, err := nanohttp.ReadAllAndReplaceBody(&req) // TODO: dev test
	if err != nil {
		return err
	}
	msg, err := mdm.DecodeCheckin(bodyBytes)
	if err != nil {
		return err
	}
	switch m := msg.(type) {
	case *mdm.Authenticate:
		dest.SerialNumber = m.SerialNumber
		dest.UDID = m.UDID
		// dest.Model = m.Model
		fmt.Println(m.SerialNumber, m.UDID) // TODO: add model to the struct
		return nil
	default:
		// these aren't the requests you're looking for, move along
		fmt.Println("wrong message type")
		return nil
	}
}
