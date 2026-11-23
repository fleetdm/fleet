package externalauth

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

// testIssuer is an OIDC issuer serving discovery and a JWKS from httptest.
type testIssuer struct {
	srv      *httptest.Server
	rsaKey   *rsa.PrivateKey
	ecKey    *ecdsa.PrivateKey
	requests atomic.Int32
	down     atomic.Bool
	empty    atomic.Bool // serve a 200 JWKS with no keys

	mu       sync.Mutex
	rsaAlgIn string // "alg" published on the RSA JWK
	rsaUse   string // "use" published on the RSA JWK
}

func newTestIssuer(t *testing.T) *testIssuer {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	iss := &testIssuer{rsaKey: rsaKey, ecKey: ecKey, rsaAlgIn: "RS256"}
	iss.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iss.requests.Add(1)
		if iss.down.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": iss.URL(), "jwks_uri": iss.URL() + "/jwks.json"})
		case "/jwks.json":
			if iss.empty.Load() {
				_, _ = w.Write([]byte(`{"keys":[]}`))
				return
			}
			iss.mu.Lock()
			alg, use := iss.rsaAlgIn, iss.rsaUse
			iss.mu.Unlock()
			var set jwkset.JWKSMarshal
			for _, k := range []struct {
				key any
				kid string
				alg jwkset.ALG
				use jwkset.USE
			}{{rsaKey.Public(), "rsa1", jwkset.ALG(alg), jwkset.USE(use)}, {ecKey.Public(), "ec1", jwkset.AlgES256, ""}} {
				jwk, err := jwkset.NewJWKFromKey(k.key, jwkset.JWKOptions{Metadata: jwkset.JWKMetadataOptions{KID: k.kid, ALG: k.alg, USE: k.use}})
				require.NoError(t, err)
				set.Keys = append(set.Keys, jwk.Marshal())
			}
			_ = json.NewEncoder(w).Encode(set)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(iss.srv.Close)
	return iss
}

func (i *testIssuer) URL() string { return i.srv.URL }

func (i *testIssuer) sign(t *testing.T, method jwt.SigningMethod, kid string, claims jwt.MapClaims) string {
	tok := jwt.NewWithClaims(method, claims)
	if kid != "" {
		tok.Header["kid"] = kid
	}
	var key crypto.Signer = i.rsaKey
	if strings.HasPrefix(method.Alg(), "ES") {
		key = i.ecKey
	}
	s, err := tok.SignedString(key)
	require.NoError(t, err)
	return s
}

type fakeUsers struct {
	mu       sync.Mutex
	bindings map[[2]string]*fleet.User
}

func (f *fakeUsers) UserByExternalIdentity(_ context.Context, issuer, subject string) (*fleet.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.bindings[[2]string{issuer, subject}]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, notFound{}
}

type notFound struct{}

func (notFound) Error() string    { return "not found" }
func (notFound) IsNotFound() bool { return true }

type testEnv struct {
	a     *Authenticator
	iss   *testIssuer
	now   time.Time
	users *fakeUsers
	logs  *bytes.Buffer
	bot   *fleet.User
}

func issuerConfig(iss *testIssuer) config.JWTIssuerConfig {
	return config.JWTIssuerConfig{
		Name: "test", Issuer: iss.URL(), Audience: "fleet", SubjectClaim: "sub",
		AllowedAlgorithms: []string{"RS256", "ES256"}, MaxTokenLifetime: 15 * time.Minute,
	}
}

func newEnv(t *testing.T, mutate ...func(*config.JWTIssuerConfig)) *testEnv {
	env := &testEnv{iss: newTestIssuer(t), now: time.Now().Truncate(time.Second), logs: &bytes.Buffer{}}
	env.bot = &fleet.User{ID: 7, Name: "bot", APIOnly: true, GlobalRole: new(fleet.RoleObserver)}
	human := &fleet.User{ID: 8, Name: "human", GlobalRole: new(fleet.RoleAdmin)}
	env.users = &fakeUsers{bindings: map[[2]string]*fleet.User{
		{"test", "bot-sub"}:   env.bot,
		{"test", "human-sub"}: human,
	}}
	cfg := issuerConfig(env.iss)
	for _, m := range mutate {
		m(&cfg)
	}
	a, err := New(t.Context(), []config.JWTIssuerConfig{cfg}, env.users, nil,
		slog.New(slog.NewJSONHandler(env.logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		WithClock(func() time.Time { return env.now }))
	require.NoError(t, err)
	env.a = a
	return env
}

func (e *testEnv) claims(sub string) jwt.MapClaims {
	return jwt.MapClaims{
		"iss": e.iss.URL(), "aud": "fleet", "sub": sub, "jti": "j1",
		"iat": e.now.Unix(), "exp": e.now.Add(5 * time.Minute).Unix(),
	}
}

func (e *testEnv) requireRejected(t *testing.T, token, reason string) {
	t.Helper()
	e.logs.Reset()
	v, err := e.a.AuthViewer(t.Context(), token, "192.0.2.1")
	require.Error(t, err)
	require.Nil(t, v)
	require.Same(t, errInvalidToken, err, "every rejection returns the same generic 401")
	require.NotContains(t, e.logs.String(), token, "raw token must never be logged")
	require.Contains(t, e.logs.String(), reason)
}

func TestIsJWTShaped(t *testing.T) {
	assert.True(t, IsJWTShaped("a.b.c"))
	for _, s := range []string{"", "abc", "a.b", "a..c", ".b.c", "a.b.", "a.b.c.d", "aGVsbG8+/w=="} {
		assert.False(t, IsJWTShaped(s), s)
	}
}

// IsJWTShaped runs on every bearer token, authenticated or not, even with no issuers
// configured, so it must not allocate per dot.
func TestIsJWTShapedDoesNotAllocate(t *testing.T) {
	dots := strings.Repeat(".", 256<<10)
	allocs := testing.AllocsPerRun(10, func() { _ = IsJWTShaped(dots) })
	require.Zero(t, allocs)
	require.False(t, IsJWTShaped(dots))
}

// A 200 with an empty or unusable key set is an outage, not a rotation: keep the old keys.
func TestAuthViewerEmptyJWKSKeepsCachedKeys(t *testing.T) {
	e := newEnv(t)
	tok := e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", e.claims("bot-sub"))
	_, err := e.a.AuthViewer(t.Context(), tok, "192.0.2.1")
	require.NoError(t, err)

	e.iss.empty.Store(true)
	require.ErrorContains(t, e.a.refreshAll(t.Context()), "no usable keys")
	_, err = e.a.AuthViewer(t.Context(), tok, "192.0.2.1")
	require.NoError(t, err, "cached keys must survive an empty JWKS response")
}

// The https requirement must hold across redirects, not only for the configured URL.
func TestKeyFetchRefusesRedirectToHTTP(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://keys.example.com/jwks.json", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	cfg := config.JWTIssuerConfig{
		Name: "redirect", Issuer: srv.URL, JWKSURL: srv.URL + "/jwks.json", Audience: "fleet",
		SubjectClaim: "sub", AllowedAlgorithms: []string{"RS256"}, MaxTokenLifetime: time.Minute,
	}
	a, err := New(t.Context(), []config.JWTIssuerConfig{cfg}, &fakeUsers{}, nil,
		slog.New(slog.DiscardHandler), WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	require.ErrorContains(t, a.refreshAll(t.Context()), "must be an https URL")
}

func TestAuthViewer(t *testing.T) {
	t.Run("valid RS256", func(t *testing.T) {
		e := newEnv(t)
		v, err := e.a.AuthViewer(t.Context(), e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", e.claims("bot-sub")), "192.0.2.1")
		require.NoError(t, err)
		require.Nil(t, v.Session)
		require.Equal(t, uint(7), v.User.ID)
		require.Contains(t, e.logs.String(), "external jwt authenticated")
	})

	t.Run("valid ES256", func(t *testing.T) {
		e := newEnv(t)
		v, err := e.a.AuthViewer(t.Context(), e.iss.sign(t, jwt.SigningMethodES256, "ec1", e.claims("bot-sub")), "192.0.2.1")
		require.NoError(t, err)
		require.Equal(t, uint(7), v.User.ID)
	})

	t.Run("time claims", func(t *testing.T) {
		e := newEnv(t)
		c := e.claims("bot-sub")
		c["exp"] = e.now.Add(-30 * time.Second).Unix() // within 60s skew
		c["iat"] = e.now.Add(-2 * time.Minute).Unix()
		_, err := e.a.AuthViewer(t.Context(), e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "192.0.2.1")
		require.NoError(t, err)

		c["exp"] = e.now.Add(-90 * time.Second).Unix()
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "expired")

		c = e.claims("bot-sub")
		c["nbf"] = e.now.Add(2 * time.Minute).Unix()
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "not yet valid")

		c = e.claims("bot-sub")
		delete(c, "iat")
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "missing iat")

		c = e.claims("bot-sub")
		delete(c, "exp")
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "missing exp")

		c = e.claims("bot-sub")
		c["exp"] = e.now.Add(16 * time.Minute).Unix()
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "lifetime exceeds max_token_lifetime")

		c = e.claims("bot-sub")
		c["iat"] = e.now.Add(5 * time.Minute).Unix()
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "issued in the future")
	})

	t.Run("audience", func(t *testing.T) {
		e := newEnv(t)
		c := e.claims("bot-sub")
		c["aud"] = []string{"other", "fleet"}
		_, err := e.a.AuthViewer(t.Context(), e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "192.0.2.1")
		require.NoError(t, err)

		for _, aud := range []any{"other", "fleet-api", "flee", []string{"fleet-api"}, nil} {
			c["aud"] = aud
			e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "audience")
		}
	})

	t.Run("issuer", func(t *testing.T) {
		e := newEnv(t)
		before := e.iss.requests.Load()
		c := e.claims("bot-sub")
		c["iss"] = "https://evil.example.com"
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "unknown issuer")
		c["iss"] = e.iss.URL() + "/"
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "unknown issuer")
		require.Equal(t, before, e.iss.requests.Load(), "unknown issuers must not cause a fetch")
	})

	t.Run("algorithm confusion", func(t *testing.T) {
		e := newEnv(t)
		none, err := jwt.NewWithClaims(jwt.SigningMethodNone, e.claims("bot-sub")).SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)
		e.requireRejected(t, none, "algorithm")

		// HS256 keyed with the public key's bytes: the classic key-confusion attack.
		pub, err := json.Marshal(e.iss.rsaKey.PublicKey)
		require.NoError(t, err)
		hs := jwt.NewWithClaims(jwt.SigningMethodHS256, e.claims("bot-sub"))
		hs.Header["kid"] = "rsa1"
		hsTok, err := hs.SignedString(pub)
		require.NoError(t, err)
		e.requireRejected(t, hsTok, "algorithm")

		// RS512 is permitted globally but not allowed for this issuer.
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS512, "rsa1", e.claims("bot-sub")), "algorithm")
	})

	t.Run("key selection", func(t *testing.T) {
		e := newEnv(t)
		before := e.iss.requests.Load()
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "unknown-kid", e.claims("bot-sub")), "unknown kid")
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "", e.claims("bot-sub")), "missing kid")
		require.Equal(t, before, e.iss.requests.Load(), "an unknown kid must not trigger a refetch")

		// Token signed with the EC key but naming the RSA kid fails verification.
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodES256, "rsa1", e.claims("bot-sub")), "algorithm")
	})

	t.Run("JWK use", func(t *testing.T) {
		e := newEnv(t)
		tok := e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", e.claims("bot-sub"))
		e.iss.mu.Lock()
		e.iss.rsaUse = "sig"
		e.iss.mu.Unlock()
		require.NoError(t, e.a.refreshAll(t.Context()))
		_, err := e.a.AuthViewer(t.Context(), tok, "192.0.2.1")
		require.NoError(t, err, `a "sig" key verifies tokens`)

		// An encryption key must never verify a signature (RFC 7517 section 4.2).
		e.iss.mu.Lock()
		e.iss.rsaUse = "enc"
		e.iss.mu.Unlock()
		require.NoError(t, e.a.refreshAll(t.Context()))
		e.requireRejected(t, tok, "unknown kid")
	})

	t.Run("JWK alg mismatch", func(t *testing.T) {
		e := newEnv(t, func(c *config.JWTIssuerConfig) { c.AllowedAlgorithms = []string{"RS256", "RS384"} })
		e.iss.mu.Lock()
		e.iss.rsaAlgIn = "RS384"
		e.iss.mu.Unlock()
		require.NoError(t, e.a.refreshAll(t.Context()))
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", e.claims("bot-sub")), "does not match key")
	})

	t.Run("subject resolution", func(t *testing.T) {
		e := newEnv(t)
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", e.claims("nobody")), "no binding")
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", e.claims("human-sub")), "not api-only")

		// Demoting the bound user takes effect on the next request; nothing is cached.
		tok := e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", e.claims("bot-sub"))
		_, err := e.a.AuthViewer(t.Context(), tok, "192.0.2.1")
		require.NoError(t, err)
		e.users.mu.Lock()
		e.bot.APIOnly = false
		e.users.mu.Unlock()
		e.requireRejected(t, tok, "not api-only")
	})

	t.Run("subject_claim", func(t *testing.T) {
		e := newEnv(t, func(c *config.JWTIssuerConfig) { c.SubjectClaim = "repository" })
		c := e.claims("ignored")
		c["repository"] = "bot-sub"
		_, err := e.a.AuthViewer(t.Context(), e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "192.0.2.1")
		require.NoError(t, err)
		delete(c, "repository")
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "missing subject claim")
	})

	t.Run("Google service account ID token shape", func(t *testing.T) {
		e := newEnv(t)
		e.users.bindings[[2]string{"test", "112233445566778899"}] = e.bot
		c := e.claims("112233445566778899")
		c["email"] = "clarion-posture@my-project.iam.gserviceaccount.com"
		c["email_verified"] = true
		c["azp"] = "112233445566778899"
		_, err := e.a.AuthViewer(t.Context(), e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "192.0.2.1")
		require.NoError(t, err)

		// Anything in the project can mint an ID token for the same account with another audience.
		c["aud"] = "https://some-other-service.example.com"
		e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "audience")
	})

	t.Run("email subject requires email_verified", func(t *testing.T) {
		e := newEnv(t, func(c *config.JWTIssuerConfig) { c.SubjectClaim = "email" })
		const email = "clarion-posture@my-project.iam.gserviceaccount.com"
		e.users.bindings[[2]string{"test", email}] = e.bot
		c := e.claims("112233445566778899")
		c["email"] = email

		c["email_verified"] = true
		_, err := e.a.AuthViewer(t.Context(), e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "192.0.2.1")
		require.NoError(t, err)

		for _, v := range []any{false, "true", nil} {
			c["email_verified"] = v
			if v == nil {
				delete(c, "email_verified")
			}
			e.requireRejected(t, e.iss.sign(t, jwt.SigningMethodRS256, "rsa1", c), "email not verified")
		}
	})
}

func TestAuthViewerKeyAvailability(t *testing.T) {
	t.Run("issuer down at startup recovers on refresh", func(t *testing.T) {
		iss := newTestIssuer(t)
		iss.down.Store(true)
		users := &fakeUsers{bindings: map[[2]string]*fleet.User{{"test", "bot-sub"}: {ID: 7, APIOnly: true}}}
		a, err := New(t.Context(), []config.JWTIssuerConfig{issuerConfig(iss)}, users, nil, slog.New(slog.DiscardHandler),
			WithRefreshInterval(20*time.Millisecond))
		require.NoError(t, err, "an unreachable issuer must not stop the server from starting")

		now := time.Now()
		tok := iss.sign(t, jwt.SigningMethodRS256, "rsa1", jwt.MapClaims{
			"iss": iss.URL(), "aud": "fleet", "sub": "bot-sub", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
		})
		_, err = a.AuthViewer(t.Context(), tok, "192.0.2.1")
		require.Error(t, err)

		iss.down.Store(false)
		require.Eventually(t, func() bool {
			_, err := a.AuthViewer(t.Context(), tok, "192.0.2.1")
			return err == nil
		}, 2*time.Second, 10*time.Millisecond)

		// Cached keys survive a later outage.
		iss.down.Store(true)
		time.Sleep(60 * time.Millisecond)
		_, err = a.AuthViewer(t.Context(), tok, "192.0.2.1")
		require.NoError(t, err)
	})

	t.Run("explicit jwks_url skips discovery", func(t *testing.T) {
		iss := newTestIssuer(t)
		cfg := issuerConfig(iss)
		cfg.JWKSURL = iss.URL() + "/jwks.json"
		_, err := New(t.Context(), []config.JWTIssuerConfig{cfg}, &fakeUsers{}, nil, slog.New(slog.DiscardHandler))
		require.NoError(t, err)
		require.Equal(t, int32(1), iss.requests.Load())
	})
}

func TestAuthViewerTwoIssuers(t *testing.T) {
	a1, a2 := newTestIssuer(t), newTestIssuer(t)
	c1, c2 := issuerConfig(a1), issuerConfig(a2)
	c2.Name, c2.Audience = "second", "fleet-2"
	users := &fakeUsers{bindings: map[[2]string]*fleet.User{
		{"test", "s"}:   {ID: 1, APIOnly: true},
		{"second", "s"}: {ID: 2, APIOnly: true},
	}}
	a, err := New(t.Context(), []config.JWTIssuerConfig{c1, c2}, users, nil, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	now := time.Now()
	claims := func(iss, aud string) jwt.MapClaims {
		return jwt.MapClaims{"iss": iss, "aud": aud, "sub": "s", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix()}
	}

	v, err := a.AuthViewer(t.Context(), a1.sign(t, jwt.SigningMethodRS256, "rsa1", claims(a1.URL(), "fleet")), "")
	require.NoError(t, err)
	require.Equal(t, uint(1), v.User.ID)
	v, err = a.AuthViewer(t.Context(), a2.sign(t, jwt.SigningMethodRS256, "rsa1", claims(a2.URL(), "fleet-2")), "")
	require.NoError(t, err)
	require.Equal(t, uint(2), v.User.ID)

	// Issuer 2's key cannot sign for issuer 1.
	_, err = a.AuthViewer(t.Context(), a2.sign(t, jwt.SigningMethodRS256, "rsa1", claims(a1.URL(), "fleet")), "")
	require.Error(t, err)
}

func TestAuthViewerRateLimit(t *testing.T) {
	iss := newTestIssuer(t)
	store, err := memstore.New(100)
	require.NoError(t, err)
	limiter, err := throttled.NewGCRARateLimiter(store, throttled.RateQuota{MaxRate: throttled.PerMin(60), MaxBurst: 1})
	require.NoError(t, err)
	logs := &bytes.Buffer{}
	users := &fakeUsers{bindings: map[[2]string]*fleet.User{{"test", "bot-sub"}: {ID: 7, APIOnly: true}}}
	a, err := New(t.Context(), []config.JWTIssuerConfig{issuerConfig(iss)}, users, limiter, slog.New(slog.NewJSONHandler(logs, nil)))
	require.NoError(t, err)
	now := time.Now()
	tok := iss.sign(t, jwt.SigningMethodRS256, "rsa1", jwt.MapClaims{
		"iss": iss.URL(), "aud": "fleet", "sub": "bot-sub", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})

	for range 2 {
		_, err = a.AuthViewer(t.Context(), tok, "192.0.2.1")
		require.NoError(t, err)
	}
	_, err = a.AuthViewer(t.Context(), tok, "192.0.2.1")
	// Rate limiting is a 429, distinguishable from an invalid token.
	var rl *RateLimitedError
	require.ErrorAs(t, err, &rl)
	require.Equal(t, http.StatusTooManyRequests, rl.StatusCode())
	require.Positive(t, rl.RetryAfter())
	require.Contains(t, logs.String(), "rate_limited")

	// Buckets are per client IP.
	_, err = a.AuthViewer(t.Context(), tok, "192.0.2.2")
	require.NoError(t, err)
}

func TestDiscoveredJWKSURIMustBeHTTPS(t *testing.T) {
	var jwksURI string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"issuer": "https://" + r.Host, "jwks_uri": jwksURI})
	}))
	t.Cleanup(srv.Close)
	cfg := config.JWTIssuerConfig{
		Name: "tls", Issuer: srv.URL, Audience: "fleet", SubjectClaim: "sub",
		AllowedAlgorithms: []string{"RS256"}, MaxTokenLifetime: time.Minute,
	}

	// An https issuer must not hand Fleet a plaintext key URL that a network attacker could answer.
	jwksURI = "http://keys.example.com/jwks.json"
	a, err := New(t.Context(), []config.JWTIssuerConfig{cfg}, &fakeUsers{}, nil, slog.New(slog.DiscardHandler),
		WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	err = a.refreshAll(t.Context())
	require.ErrorContains(t, err, "jwks_uri")
	require.False(t, a.issuers[srv.URL].hasKeys())
}
