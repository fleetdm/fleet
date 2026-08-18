// Package http provides the request and response types for the notifications
// endpoints.
package http

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
)

type GetNotificationRequest struct {
	Token string `url:"token"`
	UUID  string `url:"uuid"`
}

// DeviceAuthToken is where the device auth middleware reads the token from.
func (r *GetNotificationRequest) DeviceAuthToken() string {
	return r.Token
}

type GetNotificationResponse struct {
	Payload json.RawMessage `json:"payload"`
	Err     error           `json:"error,omitempty"`
}

func (r GetNotificationResponse) Error() error { return r.Err }

type NotificationActionRequest struct {
	Token string `url:"token"`
	UUID  string `url:"uuid"`
	api.EndUserNotificationAction
}

// DeviceAuthToken is where the device auth middleware reads the token from.
func (r *NotificationActionRequest) DeviceAuthToken() string {
	return r.Token
}

type NotificationActionResponse struct {
	Err error `json:"error,omitempty"`
}

func (r NotificationActionResponse) Error() error { return r.Err }

// Status is read by the response encoder; acting on a notification returns
// no body.
func (r NotificationActionResponse) Status() int { return nethttp.StatusNoContent }
