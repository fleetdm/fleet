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
		// wantBodyContains is the AMAPI-supplied text the client should see. Only
		// meaningful for mapped statuses.
		wantBodyContains string
	}{
		{"404", &googleapi.Error{Code: http.StatusNotFound, Message: "Not Found"}, true, http.StatusNotFound, "Not Found"},
		{"404 empty message falls back to body", &googleapi.Error{Code: http.StatusNotFound, Body: "device is gone"}, true, http.StatusNotFound, "device is gone"},
		{"500 entity not found in message", &googleapi.Error{Code: http.StatusInternalServerError, Message: "Requested entity was not found"}, true, http.StatusNotFound, "Requested entity was not found"},
		{"500 entity not found in body", &googleapi.Error{Code: http.StatusInternalServerError, Body: "Requested entity was not found"}, true, http.StatusNotFound, "Requested entity was not found"},
		{"400", &googleapi.Error{Code: http.StatusBadRequest, Message: "invalid command"}, true, http.StatusBadRequest, "invalid command"},
		{"409", &googleapi.Error{Code: http.StatusConflict, Message: "device state incompatible"}, true, http.StatusConflict, "device state incompatible"},

		// Unmapped statuses fall through to the caller's ctxerr.Wrap, which is a 500. These
		// rows pin current behaviour, they do not endorse it: mapping 401/403/429 to their
		// own statuses is a separate decision, and doing so should update these rows.
		{"401 falls through", &googleapi.Error{Code: http.StatusUnauthorized, Message: "bad creds"}, false, http.StatusInternalServerError, ""},
		{"403 falls through", &googleapi.Error{Code: http.StatusForbidden, Message: "no access"}, false, http.StatusInternalServerError, ""},
		{"429 falls through", &googleapi.Error{Code: http.StatusTooManyRequests, Message: "quota exceeded"}, false, http.StatusInternalServerError, ""},
		{"500 generic falls through", &googleapi.Error{Code: http.StatusInternalServerError, Message: "boom"}, false, http.StatusInternalServerError, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := ctxerr.NewContext(t.Context(), ctxerr.MockHandler{StoreImpl: func(error) {}})

			mapped := androidmgmt.FleetErrFromAMAPI(c.amapiErr)
			if !c.wantMapped {
				require.NoError(t, mapped, "status must not be mapped to a Fleet error type")
				// Production never encodes the AMAPI error bare: the caller wraps it. Assert
				// only the shape that actually reaches the HTTP layer.
				wrapped := httptest.NewRecorder()
				encodeError(ctx, ctxerr.Wrap(ctx, c.amapiErr, "issuing android command"), wrapped)
				assert.Equal(t, c.wantStatus, wrapped.Code, "unmapped AMAPI error encoded to the wrong status")
				return
			}
			require.Error(t, mapped)

			bare := httptest.NewRecorder()
			encodeError(ctx, mapped, bare)
			assert.Equal(t, c.wantStatus, bare.Code, "bare error encoded to the wrong status")

			wrapped := httptest.NewRecorder()
			encodeError(ctx, ctxerr.Wrap(ctx, mapped, "issuing android command"), wrapped)
			assert.Equal(t, c.wantStatus, wrapped.Code, "ctxerr-wrapped error encoded to the wrong status")

			// notFoundError implements Internal() so the AMAPI error is logged rather than
			// returned. A mapped response carries the AMAPI message, never the googleapi
			// envelope that the pre-fix 500 exposed.
			for _, body := range []string{bare.Body.String(), wrapped.Body.String()} {
				assert.NotContains(t, body, "googleapi:", "the AMAPI error envelope must not reach the response")
				assert.Contains(t, body, c.wantBodyContains)
			}
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
