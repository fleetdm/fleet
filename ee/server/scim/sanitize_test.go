package scim

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elimity-com/scim"
	scimerrors "github.com/elimity-com/scim/errors"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/stretchr/testify/require"
)

// stubHandler returns a fixed error from every ResourceHandler method.
type stubHandler struct {
	scim.ResourceHandler
	err error
}

func (s stubHandler) Create(*http.Request, scim.ResourceAttributes) (scim.Resource, error) {
	return scim.Resource{}, s.err
}

func (s stubHandler) Replace(*http.Request, string, scim.ResourceAttributes) (scim.Resource, error) {
	return scim.Resource{}, s.err
}

func (s stubHandler) Delete(*http.Request, string) error { return s.err }

func TestSanitizedResourceHandler(t *testing.T) {
	t.Parallel()

	driverErr := errors.New("batch insert scim group users: Error 1452 (23000): a foreign key constraint fails (`fleet`.`scim_user_group`)")

	for _, tc := range []struct {
		name     string
		err      error
		want     error
		replaced bool
	}{
		{
			// CheckScimError puts err.Error() straight into the response detail for
			// anything that is not already a ScimError.
			"datastore error is replaced",
			driverErr,
			nil,
			true,
		},
		{
			"library's own error survives",
			scimerrors.ScimErrorResourceNotFound("42"),
			scimerrors.ScimErrorResourceNotFound("42"),
			false,
		},
		{
			"uniqueness error survives",
			scimerrors.ScimErrorUniqueness,
			scimerrors.ScimErrorUniqueness,
			false,
		},
		{
			"nil stays nil",
			nil,
			nil,
			false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newSanitizedResourceHandler(stubHandler{err: tc.err}, slog.New(slog.DiscardHandler))
			ctx := logging.NewContext(t.Context(), &logging.LoggingContext{})
			logCtx, ok := logging.FromContext(ctx)
			require.True(t, ok)
			req := httptest.NewRequest(http.MethodPut, "/Groups/42", nil).WithContext(ctx)

			_, err := h.Replace(req, "42", scim.ResourceAttributes{})

			if tc.replaced {
				// The caller needs something to quote to find the log line.
				var scimErr scimerrors.ScimError
				require.ErrorAs(t, err, &scimErr)
				require.Contains(t, scimErr.Detail, platform_http.GenericErrorMessage)
				require.Contains(t, scimErr.Detail, logCtx.RequestID)
				require.NotContains(t, scimErr.Detail, "scim_user_group")
				return
			}
			require.Equal(t, tc.want, err)
			// Delete returns only an error, so it gets its own pass.
			require.Equal(t, tc.want, h.Delete(req, "42"))
		})
	}
}
