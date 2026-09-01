package scepserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/stretchr/testify/require"
)

func TestEncodeSCEPError(t *testing.T) {
	t.Parallel()

	internal := errors.New("depot: could not open /var/lib/fleet/scep.db: permission denied")

	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			"internal detail is replaced",
			internal,
			http.StatusInternalServerError,
			platform_http.GenericErrorMessage,
		},
		{
			"explicit 4xx keeps its message",
			&BadRequestError{Message: "missing PKIOperation message"},
			http.StatusBadRequest,
			"missing PKIOperation message",
		},
		{
			"timeout keeps its message",
			&TimeoutError{Message: "request timed out"},
			http.StatusRequestTimeout,
			"request timed out",
		},
		{
			// Carries no StatusCode, so it lands on the 500 branch, but its message
			// is curated and the caller needs it.
			"curated message without a status code survives",
			&platform_http.BadRequestError{Message: "missing data for PKIOperation"},
			http.StatusInternalServerError,
			"missing data for PKIOperation",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			encodeSCEPError(t.Context(), tc.err, recorder)

			require.Equal(t, tc.wantStatus, recorder.Code)
			require.Contains(t, recorder.Body.String(), tc.wantBody)
			require.NotContains(t, recorder.Body.String(), "permission denied")
		})
	}
}
