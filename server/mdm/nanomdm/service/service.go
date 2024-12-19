// Package service defines an MDM service
package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

// DeclarativeManagement is the interface for handling the Apple
// Declarative Management protocol via MDM "v1."
type DeclarativeManagement interface {
	DeclarativeManagement(*mdm.Request, *mdm.DeclarativeManagement) ([]byte, error)
}

// UserAuthenticate is an interface for processing the UserAuthenticate MDM check-in message.
type UserAuthenticate interface {
	UserAuthenticate(*mdm.Request, *mdm.UserAuthenticate) ([]byte, error)
}

// GetToken is the interface for handling a GetToken check-in message.
// See https://developer.apple.com/documentation/devicemanagement/get_token
type GetToken interface {
	GetToken(*mdm.Request, *mdm.GetToken) (*mdm.GetTokenResponse, error)
}

// Checkin represents the various check-in requests.
// See https://developer.apple.com/documentation/devicemanagement/check-in
type Checkin interface {
	Authenticate(*mdm.Request, *mdm.Authenticate) error
	TokenUpdate(*mdm.Request, *mdm.TokenUpdate) error
	CheckOut(*mdm.Request, *mdm.CheckOut) error
	SetBootstrapToken(*mdm.Request, *mdm.SetBootstrapToken) error
	GetBootstrapToken(*mdm.Request, *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error)
	UserAuthenticate
	DeclarativeManagement
	GetToken
}

// CommandAndReportResults represents the command report and next-command request.
// See https://developer.apple.com/documentation/devicemanagement/implementing_device_management/sending_mdm_commands_to_a_device
type CommandAndReportResults interface {
	CommandAndReportResults(*mdm.Request, *mdm.CommandResults) (*mdm.Command, error)
}

// CheckinAndCommandService faciliates check-ins and commands.
type CheckinAndCommandService interface {
	Checkin
	CommandAndReportResults
}

// ProfileService represents the interface to call specific functions from Fleet's main services.
type ProfileService interface {
	SignAndEncodeInstallProfile(ctx context.Context, profile []byte, commandUUID string) (string, error)
}
