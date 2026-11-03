package http

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.step.sm/crypto/jose"
)

func signTestJWS(t *testing.T, key any, alg jose.SignatureAlgorithm, embedJWK bool) []byte {
	t.Helper()
	opts := &jose.SignerOptions{
		EmbedJWK:     embedJWK,
		ExtraHeaders: map[jose.HeaderKey]any{"url": "https://example.com/acme/new-account", "nonce": "n"},
	}
	if !embedJWK {
		opts.ExtraHeaders["kid"] = "https://example.com/acme/account/1"
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: alg, Key: key}, opts)
	require.NoError(t, err)
	jws, err := signer.Sign([]byte(`{}`))
	require.NoError(t, err)
	return []byte(jws.FullSerialize())
}

func flattenedWithHeader(header string) []byte {
	b, _ := json.Marshal(map[string]string{
		"protected": base64.RawURLEncoding.EncodeToString([]byte(header)),
		"payload":   "",
		"signature": "AA",
	})
	return b
}

func TestPrecheckJWSBody(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	cases := []struct {
		name    string
		body    []byte
		wantErr bool
	}{
		{"EC embedded jwk", signTestJWS(t, ecKey, jose.ES256, true), false},
		{"kid", signTestJWS(t, ecKey, jose.ES256, false), false},
		{"RSA embedded jwk", signTestJWS(t, rsaKey, jose.RS256, true), true},
		{"private EC jwk", flattenedWithHeader(`{"alg":"ES256","jwk":{"kty":"EC","crv":"P-256","x":"AA","y":"AA","d":"AA"}}`), true},
		{"private RSA jwk", flattenedWithHeader(`{"alg":"RS256","jwk":{"kty":"RSA","n":"AQ","e":"AQAB","d":"AQ","p":"AQ","q":"AQ"}}`), true},
		{"jwk without kty", flattenedWithHeader(`{"alg":"ES256","jwk":{"crv":"P-256"}}`), true},
		{"case-variant protected key", []byte(`{"protected":"` + base64.RawURLEncoding.EncodeToString([]byte(`{"jwk":{"kty":"RSA","d":"AQ"}}`)) +
			`","PROTECTED":"` + base64.RawURLEncoding.EncodeToString([]byte(`{"kid":"x"}`)) + `","payload":"","signature":"AA"}`), true},
		{"deeply nested body", []byte(`{"protected":"e30","header":` + strings.Repeat("[", 20000) + strings.Repeat("]", 20000) + `}`), true},
		{"compact serialization", []byte("e30.e30.AA"), true},
		{"missing protected", []byte(`{"payload":"","signature":"AA"}`), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := precheckJWSBody(c.body)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestJWSRequestContainerDecodeBodyRejectsRSAJWK(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	var req JWSRequestContainer
	err = req.DecodeBody(context.Background(), strings.NewReader(string(signTestJWS(t, rsaKey, jose.RS256, true))), nil, nil)
	require.ErrorContains(t, err, "Failed to parse JWS body")

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	req = JWSRequestContainer{}
	require.NoError(t, req.DecodeBody(context.Background(), strings.NewReader(string(signTestJWS(t, ecKey, jose.ES256, true))), nil, nil))
	require.NotNil(t, req.Key)
	require.Equal(t, "https://example.com/acme/new-account", req.JWSHeaderURL)
}
