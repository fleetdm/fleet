// Package externalauth authenticates API-only users with JWTs from operator-configured
// OIDC issuers. A token's subject must be bound to the user through
// user_external_identities; Fleet stores no secret for this mechanism.
package externalauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/golang-jwt/jwt/v4"
	"github.com/throttled/throttled/v2"
)

const (
	clockSkew              = 60 * time.Second
	defaultRefreshInterval = time.Hour
	httpTimeout            = 10 * time.Second
)

// UserLookup resolves a bound external identity to a Fleet user.
type UserLookup interface {
	UserByExternalIdentity(ctx context.Context, issuer, subject string) (*fleet.User, error)
}

// Authenticator verifies external JWTs. Construct it once at server start.
type Authenticator struct {
	issuers         map[string]*issuer // keyed by the exact `iss` value
	users           UserLookup
	limiter         throttled.RateLimiter
	logger          *slog.Logger
	now             func() time.Time
	client          *http.Client
	refreshInterval time.Duration
}

type Option func(*Authenticator)

func WithClock(now func() time.Time) Option { return func(a *Authenticator) { a.now = now } }
func WithHTTPClient(c *http.Client) Option  { return func(a *Authenticator) { a.client = c } }
func WithRefreshInterval(d time.Duration) Option {
	return func(a *Authenticator) { a.refreshInterval = d }
}

// New builds one key cache per issuer, fetches each issuer's keys once, and refreshes
// them in the background until ctx is done. An unreachable issuer is logged rather than
// fatal so that it cannot stop Fleet from starting; its tokens are rejected until a
// refresh succeeds. limiter may be nil to disable rate limiting.
func New(ctx context.Context, issuers []config.JWTIssuerConfig, users UserLookup, limiter throttled.RateLimiter,
	logger *slog.Logger, opts ...Option,
) (*Authenticator, error) {
	if len(issuers) == 0 {
		return nil, ctxerr.New(ctx, "no jwt issuers configured")
	}
	a := &Authenticator{
		issuers:         make(map[string]*issuer, len(issuers)),
		users:           users,
		limiter:         limiter,
		logger:          logger,
		now:             time.Now,
		client:          fleethttp.NewClient(fleethttp.WithTimeout(httpTimeout)),
		refreshInterval: defaultRefreshInterval,
	}
	for _, o := range opts {
		o(a)
	}
	// Copy the client so the https requirement also applies to every redirect hop.
	client := *a.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxKeyFetchRedirects {
			return ctxerr.New(req.Context(), "too many redirects")
		}
		return config.RequireHTTPSOrLoopback(req.URL.String())
	}
	a.client = &client
	for _, c := range issuers {
		if _, dup := a.issuers[c.Issuer]; dup {
			return nil, ctxerr.Errorf(ctx, "duplicate jwt issuer %q", c.Issuer)
		}
		a.issuers[c.Issuer] = newIssuer(c, a.client)
	}
	_ = a.refreshAll(ctx)
	for _, iss := range a.issuers {
		go a.refreshLoop(ctx, iss)
	}
	return a, nil
}

func (a *Authenticator) refreshAll(ctx context.Context) error {
	var errs []error
	for _, iss := range a.issuers {
		if err := a.refreshIssuer(ctx, iss); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (a *Authenticator) refreshIssuer(ctx context.Context, iss *issuer) error {
	err := iss.refresh(ctx)
	if err != nil {
		a.logger.WarnContext(ctx, "external jwt issuer key refresh failed",
			"issuer_name", iss.cfg.Name, "err", err, "serving_cached_keys", iss.hasKeys())
	}
	return err
}

// refreshLoop is the only thing that fetches keys after startup, so request input can
// never make Fleet contact an issuer.
func (a *Authenticator) refreshLoop(ctx context.Context, iss *issuer) {
	failing := !iss.hasKeys()
	for {
		wait := a.refreshInterval
		if failing {
			// Retry sooner after a failed refresh; cached keys keep serving meanwhile.
			wait = min(wait, maxRetryInterval)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			failing = a.refreshIssuer(ctx, iss) != nil
		}
	}
}

// IsJWTShaped reports whether s has the three non-empty dot-separated segments of a
// JWS compact serialization. Fleet session keys are standard base64 and never contain a dot.
//
// It runs on every request's bearer token, so it scans without allocating.
func IsJWTShaped(s string) bool {
	header, rest, ok := strings.Cut(s, ".")
	if !ok || header == "" {
		return false
	}
	payload, sig, ok := strings.Cut(rest, ".")
	return ok && payload != "" && sig != "" && !strings.Contains(sig, ".")
}

var errInvalidToken = fleet.NewAuthRequiredError("invalid token")

// AuthViewer verifies token and returns a viewer for the bound API-only user. The viewer
// has no Session. Every failure returns the same generic error; the reason is logged.
func (a *Authenticator) AuthViewer(ctx context.Context, token, clientIP string) (*viewer.Viewer, error) {
	rj := rejection{a: a, ctx: ctx, clientIP: clientIP}

	if a.limiter != nil {
		limited, res, err := a.limiter.RateLimit("external_jwt:"+clientIP, 1)
		if err != nil {
			return rj.reject("rate_limit_error", err)
		}
		if limited {
			_, _ = rj.reject("rate_limited", nil)
			return nil, &RateLimitedError{RetryAfterSeconds: max(1, int(math.Ceil(res.RetryAfter.Seconds())))}
		}
	}

	unverified := jwt.MapClaims{}
	parsed, _, err := jwt.NewParser().ParseUnverified(token, unverified)
	if err != nil {
		return rj.reject("malformed token", err)
	}
	rj.claims = unverified
	alg, _ := parsed.Header["alg"].(string)
	if !slices.Contains(config.PermittedJWTAlgorithms, alg) {
		return rj.reject("algorithm not permitted", nil)
	}
	issValue, _ := unverified["iss"].(string)
	iss, ok := a.issuers[issValue]
	if !ok {
		return rj.reject("unknown issuer", nil)
	}
	rj.issuerName = iss.cfg.Name
	if _, ok := iss.algs[alg]; !ok {
		return rj.reject("algorithm not allowed for issuer", nil)
	}
	kid, _ := parsed.Header["kid"].(string)
	if kid == "" {
		return rj.reject("missing kid", nil)
	}
	key, ok := iss.key(kid)
	if !ok {
		return rj.reject("unknown kid", nil)
	}
	if keyAlg := key.Marshal().ALG.String(); keyAlg != "" && keyAlg != alg {
		return rj.reject("token algorithm does not match key", nil)
	}
	if !keyFitsAlg(key.Key(), alg) {
		return rj.reject("token algorithm does not match key type", nil)
	}

	claims := jwt.MapClaims{}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{alg}), jwt.WithoutClaimsValidation())
	if _, err := parser.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return key.Key(), nil }); err != nil {
		return rj.reject("invalid signature", err)
	}
	rj.claims = claims

	if reason := a.validateClaims(iss.cfg, claims); reason != "" {
		return rj.reject(reason, nil)
	}
	subject, _ := claims[iss.cfg.SubjectClaim].(string)
	if subject == "" {
		return rj.reject("missing subject claim", nil)
	}
	// OIDC Core 5.1: an email is only an identifier once the issuer asserts it's verified.
	if iss.cfg.SubjectClaim == "email" {
		if verified, _ := claims["email_verified"].(bool); !verified {
			return rj.reject("email not verified", nil)
		}
	}
	rj.subject = subject

	user, err := a.users.UserByExternalIdentity(ctx, iss.cfg.Name, subject)
	switch {
	case fleet.IsNotFound(err):
		return rj.reject("no binding for subject", nil)
	case err != nil:
		return rj.reject("user lookup failed", err)
	case !user.APIOnly:
		// A binding must never authenticate as a human account.
		return rj.reject("bound user is not api-only", nil)
	}

	// Debug, not Info: an automation authenticates on every request it makes.
	a.logger.DebugContext(ctx, "external jwt authenticated", "issuer_name", iss.cfg.Name, "sub", subject,
		"jti", claims["jti"], "user_id", user.ID, "client_ip", clientIP)
	return &viewer.Viewer{User: user}, nil
}

func (a *Authenticator) validateClaims(cfg config.JWTIssuerConfig, c jwt.MapClaims) string {
	if c["iss"] != cfg.Issuer {
		return "unknown issuer"
	}
	if !audienceContains(c["aud"], cfg.Audience) {
		return "audience mismatch"
	}
	now := a.now()
	exp, ok := numericDate(c["exp"])
	if !ok {
		return "missing exp"
	}
	iat, ok := numericDate(c["iat"])
	if !ok {
		return "missing iat"
	}
	switch {
	case now.After(exp.Add(clockSkew)):
		return "expired"
	case iat.After(now.Add(clockSkew)):
		return "issued in the future"
	case exp.Sub(iat) > cfg.MaxTokenLifetime:
		return "lifetime exceeds max_token_lifetime"
	}
	if _, present := c["nbf"]; present {
		nbf, ok := numericDate(c["nbf"])
		if !ok {
			return "malformed nbf"
		}
		if nbf.After(now.Add(clockSkew)) {
			return "not yet valid"
		}
	}
	return ""
}

// audienceContains requires the configured audience to appear exactly; `aud` may be a
// string or an array (RFC 7519 section 4.1.3).
func audienceContains(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, s := range v {
			if s == want {
				return true
			}
		}
	}
	return false
}

func numericDate(v any) (time.Time, bool) {
	switch n := v.(type) {
	case float64:
		return time.Unix(int64(n), 0), true
	case json.Number:
		i, err := n.Int64()
		return time.Unix(i, 0), err == nil
	}
	return time.Time{}, false
}

func keyFitsAlg(key any, alg string) bool {
	switch key.(type) {
	case *rsa.PublicKey:
		return strings.HasPrefix(alg, "RS")
	case *ecdsa.PublicKey:
		return strings.HasPrefix(alg, "ES")
	}
	return false
}

type rejection struct {
	a          *Authenticator
	ctx        context.Context
	clientIP   string
	claims     jwt.MapClaims
	issuerName string
	subject    string
}

func (r rejection) reject(reason string, err error) (*viewer.Viewer, error) {
	attrs := []any{"reason", reason, "issuer_name", r.issuerName, "client_ip", r.clientIP}
	if r.claims != nil {
		attrs = append(attrs, "iss", r.claims["iss"], "sub", r.claims["sub"], "jti", r.claims["jti"])
	}
	if err != nil {
		attrs = append(attrs, "err", err.Error())
	}
	r.a.logger.InfoContext(r.ctx, "external jwt rejected", attrs...)
	return nil, errInvalidToken
}

type ctxKey struct{}

// NewContext marks ctx as authenticated by an external JWT.
func NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, true)
}

// FromContext reports whether ctx was authenticated by an external JWT.
func FromContext(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKey{}).(bool)
	return v
}

// RateLimitedError is returned when a client IP exceeds the JWT authentication rate.
// It is a 429, so callers can tell throttling apart from an invalid token.
type RateLimitedError struct {
	RetryAfterSeconds int
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("external jwt authentication rate limit exceeded, retry after: %ds", e.RetryAfterSeconds)
}
func (e *RateLimitedError) StatusCode() int     { return http.StatusTooManyRequests }
func (e *RateLimitedError) RetryAfter() int     { return e.RetryAfterSeconds }
func (e *RateLimitedError) IsClientError() bool { return true }
