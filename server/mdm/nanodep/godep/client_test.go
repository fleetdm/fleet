package godep

import (
	"errors"
	"net/http"
	"testing"

	depclient "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/stretchr/testify/assert"
)

func TestIsServerError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unrelated error", errors.New("boom"), false},
		{
			"HTTPError with 5xx status",
			&HTTPError{StatusCode: http.StatusInternalServerError, Body: []byte(`"SERVER_ERROR"`)},
			true,
		},
		{
			"HTTPError with 4xx status",
			&HTTPError{StatusCode: http.StatusForbidden, Body: []byte(`"token_rejected"`)},
			false,
		},
		{
			// do() only ever constructs AuthError with StatusCode 401, but
			// DoAuth's /session handshake (client/auth.go) constructs
			// AuthError with whatever status Apple actually returns, which
			// can be a genuine 5xx if Apple's auth endpoint is down.
			"AuthError from DoAuth's /session handshake with 5xx status",
			&depclient.AuthError{StatusCode: http.StatusServiceUnavailable, Body: []byte(`"SERVER_ERROR"`)},
			true,
		},
		{
			"AuthError from do() with 401 status",
			&depclient.AuthError{StatusCode: http.StatusUnauthorized, Body: []byte(`"token_rejected"`)},
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, IsServerError(c.err))
		})
	}
}
