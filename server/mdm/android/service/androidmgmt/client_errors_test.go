package androidmgmt

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"non-google error", errors.New("something"), false},
		{"google 404", &googleapi.Error{Code: http.StatusNotFound, Message: "Not Found"}, true},
		{"google 500 with entity not found in message", &googleapi.Error{Code: http.StatusInternalServerError, Message: "Requested entity was not found"}, true},
		{"google 500 with entity not found in body only", &googleapi.Error{Code: http.StatusInternalServerError, Body: "Requested entity was not found"}, true},
		{"google 500 generic", &googleapi.Error{Code: http.StatusInternalServerError, Message: "internal error"}, false},
		{"google 400", &googleapi.Error{Code: http.StatusBadRequest, Message: "bad request"}, false},
		{"wrapped google 404", fmt.Errorf("outer: %w", &googleapi.Error{Code: http.StatusNotFound}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsNotFoundError(tt.err))
		})
	}
}

func TestFleetErrFromAMAPI(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantNil   bool
		checkFunc func(t *testing.T, err error)
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:    "non-google error",
			err:     errors.New("random"),
			wantNil: true,
		},
		{
			name: "400 bad request",
			err:  &googleapi.Error{Code: http.StatusBadRequest, Message: "invalid command"},
			checkFunc: func(t *testing.T, err error) {
				var brErr *fleet.BadRequestError
				require.ErrorAs(t, err, &brErr)
				assert.Equal(t, "invalid command", brErr.Message)
			},
		},
		{
			name: "404 not found",
			err:  &googleapi.Error{Code: http.StatusNotFound, Message: "device not found"},
			checkFunc: func(t *testing.T, err error) {
				require.True(t, fleet.IsNotFound(err))
				assert.Contains(t, err.Error(), "device not found")
			},
		},
		{
			name: "500 with entity not found in message (AMAPI quirk)",
			err:  &googleapi.Error{Code: http.StatusInternalServerError, Message: "Requested entity was not found"},
			checkFunc: func(t *testing.T, err error) {
				require.True(t, fleet.IsNotFound(err))
			},
		},
		{
			name: "500 with entity not found in body only (AMAPI quirk)",
			err:  &googleapi.Error{Code: http.StatusInternalServerError, Body: "Requested entity was not found"},
			checkFunc: func(t *testing.T, err error) {
				require.True(t, fleet.IsNotFound(err))
			},
		},
		{
			name: "409 conflict",
			err:  &googleapi.Error{Code: http.StatusConflict, Message: "device state incompatible"},
			checkFunc: func(t *testing.T, err error) {
				var cErr *fleet.ConflictError
				require.ErrorAs(t, err, &cErr)
				assert.Equal(t, "device state incompatible", cErr.Message)
			},
		},
		{
			name:    "401 unauthorized falls through",
			err:     &googleapi.Error{Code: http.StatusUnauthorized, Message: "bad creds"},
			wantNil: true,
		},
		{
			name:    "403 forbidden falls through",
			err:     &googleapi.Error{Code: http.StatusForbidden, Message: "no access"},
			wantNil: true,
		},
		{
			name:    "500 generic falls through",
			err:     &googleapi.Error{Code: http.StatusInternalServerError, Message: "server error"},
			wantNil: true,
		},
		{
			name: "wrapped google 404",
			err:  fmt.Errorf("outer: %w", &googleapi.Error{Code: http.StatusNotFound, Message: "gone"}),
			checkFunc: func(t *testing.T, err error) {
				require.True(t, fleet.IsNotFound(err))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FleetErrFromAMAPI(tt.err)
			if tt.wantNil {
				assert.NoError(t, result)
			} else {
				require.Error(t, result)
				tt.checkFunc(t, result)
			}
		})
	}
}
