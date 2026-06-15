package service

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPSSORegistrationRequestDecodeBody(t *testing.T) {
	pem := "-----BEGIN PUBLIC KEY-----\nMFkw+abc/def=\n-----END PUBLIC KEY-----"
	form := url.Values{}
	form.Set("device_uuid", "A72B07D0-2E08-45CE-9423-1FCAFFAEC390")
	form.Set("device_signing_key", pem)
	form.Set("device_encryption_key", pem)
	form.Set("signing_key_id", "sign-kid")
	form.Set("encryption_key_id", "enc-kid")

	var req pssoRegistrationRequest
	err := req.DecodeBody(t.Context(), strings.NewReader(form.Encode()), nil, nil)
	require.NoError(t, err)
	require.Equal(t, "A72B07D0-2E08-45CE-9423-1FCAFFAEC390", req.DeviceUUID)
	// PEM survives urlencoding round trip: '+', '/', '=' and newlines intact.
	require.Equal(t, pem, req.DeviceSigningKey)
	require.Equal(t, pem, req.DeviceEncryptionKey)
	require.Equal(t, "sign-kid", req.SigningKeyID)
	require.Equal(t, "enc-kid", req.EncryptionKeyID)
}

func TestPSSOTokenRequestDecodeBody(t *testing.T) {
	t.Run("extracts assertion", func(t *testing.T) {
		form := url.Values{}
		form.Set("assertion", "eyJhbGciOiJFUzI1NiJ9.payload.sig")

		var req pssoTokenRequest
		err := req.DecodeBody(t.Context(), strings.NewReader(form.Encode()), nil, nil)
		require.NoError(t, err)
		require.Equal(t, "eyJhbGciOiJFUzI1NiJ9.payload.sig", req.Assertion)
	})

	t.Run("missing assertion rejected", func(t *testing.T) {
		var req pssoTokenRequest
		err := req.DecodeBody(t.Context(), strings.NewReader("other=value"), nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing assertion")
	})

	t.Run("empty body rejected", func(t *testing.T) {
		var req pssoTokenRequest
		err := req.DecodeBody(t.Context(), strings.NewReader(""), nil, nil)
		require.Error(t, err)
	})

	t.Run("malformed form rejected", func(t *testing.T) {
		var req pssoTokenRequest
		err := req.DecodeBody(t.Context(), strings.NewReader("a=%zz"), nil, nil)
		require.Error(t, err)
	})
}
