package godep

import (
	"errors"
	"net/http"
	"testing"

	depclient "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/stretchr/testify/assert"
)

func TestIsTokenRejected(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unrelated error", errors.New("boom"), false},
		{
			"HTTPError with matching status and body",
			&HTTPError{StatusCode: http.StatusForbidden, Body: []byte(`"token_rejected"`)},
			true,
		},
		{
			"HTTPError with matching body but wrong status",
			&HTTPError{StatusCode: http.StatusUnauthorized, Body: []byte(`"token_rejected"`)},
			false,
		},
		{
			"AuthError from do() with matching status (401) and body",
			&depclient.AuthError{StatusCode: http.StatusUnauthorized, Body: []byte(`"token_rejected"`)},
			true,
		},
		{
			"AuthError with matching status but different body",
			&depclient.AuthError{StatusCode: http.StatusUnauthorized, Body: []byte(`"signature_invalid"`)},
			false,
		},
		{
			// DoAuth's /session handshake (client/auth.go) constructs
			// AuthError with whatever status Apple actually returns, often
			// 403, unlike do() which always uses 401.
			"AuthError from DoAuth's /session handshake with matching status (403) and body",
			&depclient.AuthError{StatusCode: http.StatusForbidden, Body: []byte(`"token_rejected"`)},
			true,
		},
		{
			"AuthError with matching body but a status neither do() nor DoAuth would use for this error",
			&depclient.AuthError{StatusCode: http.StatusInternalServerError, Body: []byte(`"token_rejected"`)},
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, IsTokenRejected(c.err))
		})
	}
}

func TestIsSignatureInvalid(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unrelated error", errors.New("boom"), false},
		{
			"HTTPError with matching status and body",
			&HTTPError{StatusCode: http.StatusForbidden, Body: []byte(`"signature_invalid"`)},
			true,
		},
		{
			"HTTPError with matching body but wrong status",
			&HTTPError{StatusCode: http.StatusUnauthorized, Body: []byte(`"signature_invalid"`)},
			false,
		},
		{
			"AuthError from do() with matching status (401) and body",
			&depclient.AuthError{StatusCode: http.StatusUnauthorized, Body: []byte(`"signature_invalid"`)},
			true,
		},
		{
			"AuthError with matching status but different body",
			&depclient.AuthError{StatusCode: http.StatusUnauthorized, Body: []byte(`"token_rejected"`)},
			false,
		},
		{
			// DoAuth's /session handshake (client/auth.go) constructs
			// AuthError with whatever status Apple actually returns, often
			// 403, unlike do() which always uses 401.
			"AuthError from DoAuth's /session handshake with matching status (403) and body",
			&depclient.AuthError{StatusCode: http.StatusForbidden, Body: []byte(`"signature_invalid"`)},
			true,
		},
		{
			"AuthError with matching body but a status neither do() nor DoAuth would use for this error",
			&depclient.AuthError{StatusCode: http.StatusInternalServerError, Body: []byte(`"signature_invalid"`)},
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, IsSignatureInvalid(c.err))
		})
	}
}
