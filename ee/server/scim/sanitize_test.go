package scim

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elimity-com/scim"
	scimerrors "github.com/elimity-com/scim/errors"
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
		name string
		err  error
		want error
	}{
		{
			// CheckScimError puts err.Error() straight into the response detail for
			// anything that is not already a ScimError.
			"datastore error is replaced",
			driverErr,
			scimerrors.ScimError{Status: http.StatusInternalServerError, Detail: platform_http.GenericErrorMessage},
		},
		{
			"library's own error survives",
			scimerrors.ScimErrorResourceNotFound("42"),
			scimerrors.ScimErrorResourceNotFound("42"),
		},
		{
			"uniqueness error survives",
			scimerrors.ScimErrorUniqueness,
			scimerrors.ScimErrorUniqueness,
		},
		{
			"nil stays nil",
			nil,
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newSanitizedResourceHandler(stubHandler{err: tc.err}, slog.New(slog.DiscardHandler))
			req := httptest.NewRequest(http.MethodPut, "/Groups/42", nil)

			_, err := h.Replace(req, "42", scim.ResourceAttributes{})
			require.Equal(t, tc.want, err)

			if tc.err != nil {
				require.NotContains(t, err.Error(), "scim_user_group")
			}
			// Delete returns only an error, so it gets its own pass.
			require.Equal(t, tc.want, h.Delete(req, "42"))
		})
	}
}
