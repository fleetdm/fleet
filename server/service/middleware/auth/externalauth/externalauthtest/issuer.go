// Package externalauthtest provides an OIDC issuer for tests of external JWT authentication.
package externalauthtest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

const KeyID = "test-key"

// Issuer serves OIDC discovery and a JWKS with one RSA key.
type Issuer struct {
	srv *httptest.Server
	key *rsa.PrivateKey
}

func NewIssuer(t testing.TB) *Issuer {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	jwk, err := jwkset.NewJWKFromKey(key.Public(), jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{KID: KeyID, ALG: jwkset.AlgRS256},
	})
	require.NoError(t, err)
	iss := &Issuer{key: key}
	iss.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": iss.URL(), "jwks_uri": iss.URL() + "/jwks.json"})
		case "/jwks.json":
			_ = json.NewEncoder(w).Encode(jwkset.JWKSMarshal{Keys: []jwkset.JWKMarshal{jwk.Marshal()}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(iss.srv.Close)
	return iss
}

func (i *Issuer) URL() string { return i.srv.URL }

// Token mints an RS256 token for sub and aud valid for ttl.
func (i *Issuer) Token(t testing.TB, sub, aud string, ttl time.Duration) string {
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": i.URL(), "sub": sub, "aud": aud, "iat": now.Unix(), "exp": now.Add(ttl).Unix(),
	})
	tok.Header["kid"] = KeyID
	s, err := tok.SignedString(i.key)
	require.NoError(t, err)
	return s
}
