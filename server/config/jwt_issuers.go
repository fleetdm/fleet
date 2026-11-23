package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"time"
)

const (
	jwtIssuersKey           = "auth.jwt_issuers"
	defaultJWTSubjectClaim  = "sub"
	defaultJWTTokenLifetime = 15 * time.Minute
	maxJWTTokenLifetime     = 24 * time.Hour
	// maxJWTIssuerNameLen matches user_external_identities.issuer.
	maxJWTIssuerNameLen          = 255
	DefaultJWTRateLimitPerMinute = 1200
	jwtRateLimitPerMinuteKey     = "auth.jwt_rate_limit_per_minute"
)

var (
	defaultJWTAlgorithms = []string{"RS256", "ES256"}
	// PermittedJWTAlgorithms is the asymmetric-only set an issuer may allow. Symmetric
	// algorithms are excluded so a published JWKS key can never be used as an HMAC secret.
	PermittedJWTAlgorithms = []string{"RS256", "RS384", "RS512", "ES256", "ES384"}
)

// JWTIssuerConfig is an external OIDC issuer whose tokens may authenticate API-only
// users that have an external identity bound to them.
type JWTIssuerConfig struct {
	// Name is the stable handle stored in user_external_identities.issuer.
	Name   string `json:"name" yaml:"name"`
	Issuer string `json:"issuer" yaml:"issuer"`
	// JWKSURL is optional; when empty it is discovered from the issuer's
	// /.well-known/openid-configuration.
	JWKSURL           string        `json:"jwks_url" yaml:"jwks_url"`
	Audience          string        `json:"audience" yaml:"audience"`
	SubjectClaim      string        `json:"subject_claim" yaml:"subject_claim"`
	AllowedAlgorithms []string      `json:"allowed_algorithms" yaml:"allowed_algorithms"`
	MaxTokenLifetime  time.Duration `json:"max_token_lifetime" yaml:"max_token_lifetime"`
}

// ValidateJWTIssuers reports the first invalid jwt_issuers entry, if any.
func (a AuthConfig) ValidateJWTIssuers() error {
	names := make(map[string]struct{}, len(a.JWTIssuers))
	issuers := make(map[string]struct{}, len(a.JWTIssuers))
	for i, c := range a.JWTIssuers {
		prefix := fmt.Sprintf("%s[%d]", jwtIssuersKey, i)
		if c.Name == "" || len(c.Name) > maxJWTIssuerNameLen {
			return fmt.Errorf("%s: name must be 1 to %d characters", prefix, maxJWTIssuerNameLen)
		}
		prefix = fmt.Sprintf("%s (%s)", prefix, c.Name)
		if _, dup := names[c.Name]; dup {
			return fmt.Errorf("%s: duplicate name", prefix)
		}
		names[c.Name] = struct{}{}
		if err := requireHTTPS(c.Issuer); err != nil {
			return fmt.Errorf("%s: issuer %w", prefix, err)
		}
		if _, dup := issuers[c.Issuer]; dup {
			return fmt.Errorf("%s: duplicate issuer %q; bind all its subjects under one entry", prefix, c.Issuer)
		}
		issuers[c.Issuer] = struct{}{}
		if c.JWKSURL != "" {
			if err := requireHTTPS(c.JWKSURL); err != nil {
				return fmt.Errorf("%s: jwks_url %w", prefix, err)
			}
		}
		if c.Audience == "" {
			return fmt.Errorf("%s: audience is required", prefix)
		}
		if c.SubjectClaim == "" {
			return fmt.Errorf("%s: subject_claim is required", prefix)
		}
		if len(c.AllowedAlgorithms) == 0 {
			return fmt.Errorf("%s: allowed_algorithms is required", prefix)
		}
		for _, alg := range c.AllowedAlgorithms {
			if !slices.Contains(PermittedJWTAlgorithms, alg) {
				return fmt.Errorf("%s: algorithm %q is not permitted; use one of %v", prefix, alg, PermittedJWTAlgorithms)
			}
		}
		if c.MaxTokenLifetime <= 0 || c.MaxTokenLifetime > maxJWTTokenLifetime {
			return fmt.Errorf("%s: max_token_lifetime must be between 1s and %s", prefix, maxJWTTokenLifetime)
		}
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// RequireHTTPSOrLoopback reports an error unless raw is an https URL, or an http URL
// whose host is loopback.
func RequireHTTPSOrLoopback(raw string) error { return requireHTTPS(raw) }

func requireHTTPS(raw string) error {
	if raw == "" {
		return errors.New("is required")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%q must be an https URL", raw)
	}
	switch {
	case u.Scheme == "https":
		return nil
	case u.Scheme == "http" && isLoopbackHost(u.Hostname()):
		// Plain http is allowed only where traffic cannot leave the host.
		return nil
	}
	return fmt.Errorf("%q must be an https URL", raw)
}

// decodeJSONStrict rejects unknown keys, so a misspelled option such as
// "max_token_lifetme" fails at startup instead of silently using the default.
func decodeJSONStrict(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// getConfigJWTIssuers accepts a YAML list from a config file or a JSON array from
// an environment variable or flag, like server.endpoint_request_size_overrides.
func (man Manager) getConfigJWTIssuers() []JWTIssuerConfig {
	var raw []struct {
		JWTIssuerConfig
		MaxTokenLifetime string `json:"max_token_lifetime"`
	}
	switch v := man.getInterfaceVal(jwtIssuersKey).(type) {
	case string:
		if v == "" {
			return nil
		}
		if err := decodeJSONStrict([]byte(v), &raw); err != nil {
			panic(fmt.Sprintf("Unable to parse %s: %s", jwtIssuersKey, err))
		}
	case []any:
		b, err := json.Marshal(v)
		if err != nil {
			panic(fmt.Sprintf("Unable to encode value for key %s: %s", jwtIssuersKey, err))
		}
		if err := decodeJSONStrict(b, &raw); err != nil {
			panic(fmt.Sprintf("Unable to parse %s: %s", jwtIssuersKey, err))
		}
	default:
		panic(fmt.Sprintf("Unable to parse %s: unexpected type %T", jwtIssuersKey, v))
	}
	if len(raw) == 0 {
		return nil
	}

	out := make([]JWTIssuerConfig, 0, len(raw))
	for _, r := range raw {
		c := r.JWTIssuerConfig
		c.MaxTokenLifetime = defaultJWTTokenLifetime
		if r.MaxTokenLifetime != "" {
			d, err := time.ParseDuration(r.MaxTokenLifetime)
			if err != nil {
				panic(fmt.Sprintf("Unable to parse %s max_token_lifetime %q: %s", jwtIssuersKey, r.MaxTokenLifetime, err))
			}
			c.MaxTokenLifetime = d
		}
		if c.SubjectClaim == "" {
			c.SubjectClaim = defaultJWTSubjectClaim
		}
		if len(c.AllowedAlgorithms) == 0 {
			c.AllowedAlgorithms = slices.Clone(defaultJWTAlgorithms)
		}
		out = append(out, c)
	}
	return out
}
