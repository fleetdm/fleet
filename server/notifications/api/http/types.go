// Package http provides HTTP request and response types for the
// notifications bounded context. These types are used exclusively by the
// notifications endpoint handlers.
package http

import (
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
)

// GetNotificationRequest is the HTTP request type for a device fetching one
// of its notifications.
type GetNotificationRequest struct {
	Token string `url:"token"`
	UUID  string `url:"uuid"`
}

// DeviceAuthToken implements the interface the device auth middleware uses to
// find the token before the request is authenticated.
func (r *GetNotificationRequest) DeviceAuthToken() string {
	return r.Token
}

// GetNotificationResponse is the HTTP response type for a device fetching one
// of its notifications.
type GetNotificationResponse struct {
	Payload json.RawMessage `json:"payload"`
	Err     error           `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r GetNotificationResponse) Error() error { return r.Err }

// NotificationActionRequest is the HTTP request type for a device reporting
// what it did with one of its notifications.
type NotificationActionRequest struct {
	Token string `url:"token"`
	UUID  string `url:"uuid"`
	api.EndUserNotificationAction
}

// DeviceAuthToken implements the interface the device auth middleware uses to
// find the token before the request is authenticated.
func (r *NotificationActionRequest) DeviceAuthToken() string {
	return r.Token
}

// NotificationActionResponse is the HTTP response type for a device reporting
// what it did with one of its notifications.
type NotificationActionResponse struct {
	Err error `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r NotificationActionResponse) Error() error { return r.Err }

// Status implements the interface that overrides the default response status.
func (r NotificationActionResponse) Status() int { return 204 }
