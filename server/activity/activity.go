package activity

import "context"

// Service defines the public interface for the activity bounded context.
// Other bounded contexts should use this interface to interact with activities.
//
// TODO: Remove or move this? Consumers should define their own interface for interacting with activities.
type Service interface {
	// Ping verifies the service is healthy.
	// This is a placeholder method for the scaffold phase.
	Ping(ctx context.Context) error
}

// /////////////////////////////////////////////
// Activity API request and response structs

// DefaultResponse is the base response type for activity endpoints.
type DefaultResponse struct {
	Err error `json:"error,omitempty"`
}

// Error implements the fleet.Errorer interface.
func (r DefaultResponse) Error() error { return r.Err }

// PingResponse is the response for the ping endpoint.
type PingResponse struct {
	Message string `json:"message"`
	DefaultResponse
}
