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
			"AuthError with matching status and body",
			&depclient.AuthError{StatusCode: http.StatusForbidden, Body: []byte(`"token_rejected"`)},
			true,
		},
		{
			"AuthError with matching status but different body",
			&depclient.AuthError{StatusCode: http.StatusForbidden, Body: []byte(`"signature_invalid"`)},
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
			"AuthError with matching status and body",
			&depclient.AuthError{StatusCode: http.StatusForbidden, Body: []byte(`"signature_invalid"`)},
			true,
		},
		{
			"AuthError with matching status but different body",
			&depclient.AuthError{StatusCode: http.StatusForbidden, Body: []byte(`"token_rejected"`)},
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, IsSignatureInvalid(c.err))
		})
	}
}
