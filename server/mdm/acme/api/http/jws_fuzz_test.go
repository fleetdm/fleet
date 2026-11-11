package http

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const acmeFuzzProtectedHeader = `{"alg":"ES256","kid":"https://example.test/acme/account/1","url":"https://example.test/acme/new-account"}`
const acmeFuzzProtectedHeaderWithJWK = `{"alg":"ES256","jwk":{"kty":"EC","crv":"P-256","x":"axfR8uEsQkf4vOblY6RA8ncDfYEt6zOg9KE5RdiYwpY","y":"T-NC4v4af5uO5-tKfA-eFivOM1drMV7Oy7ZAaDe_UfU"},"url":"https://example.test/acme/new-account"}`

func acmeFuzzBody(protectedHeader []byte) []byte {
	body, _ := json.Marshal(map[string]string{
		"protected": base64.RawURLEncoding.EncodeToString(protectedHeader),
		"payload":   "e30",
		"signature": "AA",
	})
	return body
}

func assertPrecheckSuccessProperties(t *testing.T, body []byte) {
	t.Helper()
	var flattened map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &flattened))
	keys := make(map[string]struct{}, len(flattened))
	for key := range flattened {
		keys[key] = struct{}{}
	}
	_, hasUnprotectedHeader := keys["header"]
	_, hasGeneralSerialization := keys["signatures"]
	require.False(t, hasUnprotectedHeader)
	require.False(t, hasGeneralSerialization)

	var protected string
	require.NoError(t, json.Unmarshal(flattened["protected"], &protected))
	headerJSON, err := base64.RawURLEncoding.DecodeString(protected)
	require.NoError(t, err)
	var header map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(headerJSON, &header))
	if rawJWK, ok := header["jwk"]; ok {
		var jwk map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(rawJWK, &jwk))
		var kty string
		require.NoError(t, json.Unmarshal(jwk["kty"], &kty))
		require.Equal(t, "EC", kty)
		_, hasPrivateKey := jwk["d"]
		require.False(t, hasPrivateKey)
	}
}

func assertDecodeBodySuccessProperties(t *testing.T, req *JWSRequestContainer) {
	t.Helper()
	require.Len(t, req.JWS.Signatures, 1)
	require.NotEmpty(t, req.JWSHeaderURL)
	require.NotEqual(t, req.Key == nil, req.KeyID == nil, "exactly one protected key or key ID is required")
	if req.Key != nil {
		require.NotNil(t, req.Key.Key)
	}
}

func assertP256JWKExtracted(t *testing.T, req *JWSRequestContainer) {
	t.Helper()
	require.NotNil(t, req.Key)
	require.Nil(t, req.KeyID)
	publicKey, ok := req.Key.Key.(*ecdsa.PublicKey)
	require.True(t, ok, "protected JWK must decode to an ECDSA public key")
	require.Equal(t, elliptic.P256(), publicKey.Curve)
}

func FuzzPrecheckJWSBodyRaw(f *testing.F) {
	validBody := acmeFuzzBody([]byte(acmeFuzzProtectedHeader))
	validJWKBody := acmeFuzzBody([]byte(acmeFuzzProtectedHeaderWithJWK))
	f.Add(validBody)
	f.Add(validJWKBody)
	f.Add([]byte(`{"payload":"","signature":"AA"}`))
	f.Add([]byte(`e30.e30.AA`))
	f.Add([]byte(`{"protected":"e30","header":{"jwk":{"kty":"RSA","d":"AQ"}}}`))

	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > 16384 {
			t.Skip()
		}
		err := precheckJWSBody(body)
		if bytes.Equal(body, validBody) {
			require.NoError(t, err, "fixed flattened JWS seed must pass precheck")
		}
		if bytes.Equal(body, validJWKBody) {
			require.NoError(t, err, "fixed flattened JWK JWS seed must pass precheck")
		}
		if err != nil {
			return
		}
		assertPrecheckSuccessProperties(t, body)
	})
}

func FuzzPrecheckJWSProtectedHeader(f *testing.F) {
	validHeader := []byte(acmeFuzzProtectedHeader)
	validJWKHeader := []byte(acmeFuzzProtectedHeaderWithJWK)
	f.Add(validHeader)
	f.Add(validJWKHeader)
	f.Add([]byte(`{"alg":"ES256","jwk":{"kty":"EC","crv":"P-256","x":"BGsX0fLhLEJH-Lzm5WOkQ_J3A32BLeszoPShOUXYmMKWT-NC4v4af5uO5-tKfA-eFivOM1drMV7Oy7ZAaDe_UfU","y":"e45-AtUiRYM44JywWse-19nc17EEPpVZx6y7Zvdy-qE"},"url":"https://example.test/acme"}`))
	f.Add([]byte(`{"alg":"ES256","jwk":{"kty":"EC","d":"AQ"},"url":"https://example.test/acme"}`))

	f.Fuzz(func(t *testing.T, protectedHeader []byte) {
		if len(protectedHeader) > 8192 {
			t.Skip()
		}
		body := acmeFuzzBody(protectedHeader)
		err := precheckJWSBody(body)
		if bytes.Equal(protectedHeader, validHeader) {
			require.NoError(t, err, "fixed protected-header seed must pass precheck")
		}
		if bytes.Equal(protectedHeader, validJWKHeader) {
			require.NoError(t, err, "fixed public-JWK protected-header seed must pass precheck")
		}
		if err != nil {
			return
		}
		assertPrecheckSuccessProperties(t, body)
	})
}

func FuzzJWSRequestContainerDecodeBodyRaw(f *testing.F) {
	validBody := acmeFuzzBody([]byte(acmeFuzzProtectedHeader))
	validJWKBody := acmeFuzzBody([]byte(acmeFuzzProtectedHeaderWithJWK))
	f.Add(validBody)
	f.Add(validJWKBody)
	f.Add([]byte(`{"payload":"","signature":"AA"}`))
	f.Add([]byte(`{"protected":"e30","payload":"e30","signature":"AA"}`))

	f.Fuzz(func(t *testing.T, body []byte) {
		if len(body) > 16384 {
			t.Skip()
		}
		var req JWSRequestContainer
		err := req.DecodeBody(t.Context(), strings.NewReader(string(body)), nil, nil)
		if bytes.Equal(body, validBody) {
			require.NoError(t, err, "fixed flattened JWS seed must decode")
			require.Equal(t, "https://example.test/acme/account/1", *req.KeyID)
		}
		if bytes.Equal(body, validJWKBody) {
			require.NoError(t, err, "fixed flattened public-JWK seed must decode")
			assertP256JWKExtracted(t, &req)
		}
		if err != nil {
			return
		}
		assertDecodeBodySuccessProperties(t, &req)
	})
}

func FuzzJWSRequestContainerDecodeBodyProtectedHeader(f *testing.F) {
	validHeader := []byte(acmeFuzzProtectedHeader)
	validJWKHeader := []byte(acmeFuzzProtectedHeaderWithJWK)
	f.Add(validHeader)
	f.Add(validJWKHeader)
	f.Add([]byte(`{"alg":"ES256","kid":"acct","url":"https://example.test/acme","crit":["unknown"]}`))
	f.Add([]byte(`{"alg":"ES256","jwk":{"kty":"RSA","n":"AQ","e":"AQAB"},"url":"https://example.test/acme"}`))

	f.Fuzz(func(t *testing.T, protectedHeader []byte) {
		if len(protectedHeader) > 8192 {
			t.Skip()
		}
		body := acmeFuzzBody(protectedHeader)
		var req JWSRequestContainer
		err := req.DecodeBody(t.Context(), strings.NewReader(string(body)), nil, nil)
		if bytes.Equal(protectedHeader, validHeader) {
			require.NoError(t, err, "fixed protected-header seed must decode")
			require.Equal(t, "https://example.test/acme/account/1", *req.KeyID)
		}
		if bytes.Equal(protectedHeader, validJWKHeader) {
			require.NoError(t, err, "fixed public-JWK protected-header seed must decode")
			assertP256JWKExtracted(t, &req)
		}
		if err != nil {
			return
		}
		assertDecodeBodySuccessProperties(t, &req)
	})
}
