package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
)

// TestAMAPIErrorHTTPStatus pins the HTTP status every AMAPI error encodes to.
//
// Asserting the mapped error type (or fleet.IsNotFound) is not enough: encodeError
// type-switches on ctxerr.Cause, so an error that satisfies IsNotFound can still encode
// as a 500 if it defines a single-error Unwrap. Each case is checked both bare (as
// UnenrollAndroidHost returns it) and ctxerr-wrapped (as ee/server/service/hosts.go
// wraps it for lock/wipe/clear-passcode).
func TestAMAPIErrorHTTPStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		amapiErr   *googleapi.Error
		wantMapped bool
		wantStatus int
	}{
		{"404", &googleapi.Error{Code: http.StatusNotFound, Message: "Not Found"}, true, http.StatusNotFound},
		{"404 empty message falls back to body", &googleapi.Error{Code: http.StatusNotFound, Body: "device is gone"}, true, http.StatusNotFound},
		{"500 entity not found in message", &googleapi.Error{Code: http.StatusInternalServerError, Message: "Requested entity was not found"}, true, http.StatusNotFound},
		{"500 entity not found in body", &googleapi.Error{Code: http.StatusInternalServerError, Body: "Requested entity was not found"}, true, http.StatusNotFound},
		{"400", &googleapi.Error{Code: http.StatusBadRequest, Message: "invalid command"}, true, http.StatusBadRequest},
		{"409", &googleapi.Error{Code: http.StatusConflict, Message: "device state incompatible"}, true, http.StatusConflict},

		// Unmapped statuses fall through to the caller's ctxerr.Wrap, which is a 500.
		{"401 falls through", &googleapi.Error{Code: http.StatusUnauthorized, Message: "bad creds"}, false, http.StatusInternalServerError},
		{"403 falls through", &googleapi.Error{Code: http.StatusForbidden, Message: "no access"}, false, http.StatusInternalServerError},
		{"429 falls through", &googleapi.Error{Code: http.StatusTooManyRequests, Message: "quota exceeded"}, false, http.StatusInternalServerError},
		{"500 generic falls through", &googleapi.Error{Code: http.StatusInternalServerError, Message: "boom"}, false, http.StatusInternalServerError},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := ctxerr.NewContext(t.Context(), ctxerr.MockHandler{StoreImpl: func(error) {}})

			mapped := androidmgmt.FleetErrFromAMAPI(c.amapiErr)
			if !c.wantMapped {
				require.NoError(t, mapped, "status must not be mapped to a Fleet error type")
				// The caller falls back to wrapping the raw AMAPI error.
				mapped = c.amapiErr
			}
			require.Error(t, mapped)

			bare := httptest.NewRecorder()
			encodeError(ctx, mapped, bare)
			assert.Equal(t, c.wantStatus, bare.Code, "bare error encoded to the wrong status")

			wrapped := httptest.NewRecorder()
			encodeError(ctx, ctxerr.Wrap(ctx, mapped, "issuing android command"), wrapped)
			assert.Equal(t, c.wantStatus, wrapped.Code, "ctxerr-wrapped error encoded to the wrong status")
		})
	}
}

// TestAMAPIMappedErrorsSurviveCtxerrCause guards the specific mechanism that broke:
// encodeError type-switches on ctxerr.Cause, so a mapped error whose Unwrap is the
// single-error form is replaced by the *googleapi.Error before the switch runs and can
// no longer match any status interface.
func TestAMAPIMappedErrorsSurviveCtxerrCause(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		code         int
		carriesAMAPI bool // whether the mapped type retains the AMAPI error for errors.As
	}{
		{http.StatusNotFound, true},
		{http.StatusBadRequest, true},
		{http.StatusConflict, false}, // fleet.ConflictError has no field for the cause
	} {
		amapiErr := &googleapi.Error{Code: c.code, Message: "amapi failure"}
		mapped := androidmgmt.FleetErrFromAMAPI(amapiErr)
		require.Error(t, mapped, "status %d must map to a Fleet error", c.code)

		_, causeIsAMAPI := ctxerr.Cause(mapped).(*googleapi.Error)
		assert.False(t, causeIsAMAPI,
			"ctxerr.Cause walked past the mapped error for status %d; encodeError will render it as a 500", c.code)

		if c.carriesAMAPI {
			// Unwrapping must still work for callers that inspect the AMAPI error.
			var target *googleapi.Error
			require.ErrorAs(t, mapped, &target, "errors.As must still reach the AMAPI error for status %d", c.code)
			require.NotNil(t, target)
			assert.Equal(t, c.code, target.Code)
		}
	}
}
