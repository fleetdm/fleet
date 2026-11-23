package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/testutils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func loadTestConfig(t *testing.T, yaml string, env ...string) FleetConfig {
	var cmd cobra.Command
	cmd.PersistentFlags().StringP("config", "c", "", "Path to a configuration file")
	man := NewManager(&cmd)
	man.viper.SetConfigType("yaml")
	require.NoError(t, man.viper.ReadConfig(strings.NewReader(yaml)))

	testutils.SaveEnv(t)
	os.Clearenv()
	for _, e := range env {
		kv := strings.SplitN(e, "=", 2)
		t.Setenv(kv[0], kv[1])
	}
	cfg, err := man.LoadConfig()
	require.NoError(t, err)
	return cfg
}

func TestJWTIssuersLoad(t *testing.T) {
	want := []JWTIssuerConfig{{
		Name:              "clarion",
		Issuer:            "https://clarion.example.com",
		JWKSURL:           "https://clarion.example.com/jwks.json",
		Audience:          "https://fleet.example.com",
		SubjectClaim:      "sub",
		AllowedAlgorithms: []string{"RS256", "ES256"},
		MaxTokenLifetime:  15 * time.Minute,
	}}

	t.Run("unset", func(t *testing.T) {
		cfg := loadTestConfig(t, "")
		require.Empty(t, cfg.Auth.JWTIssuers)
		require.Equal(t, 1200, cfg.Auth.JWTRateLimitPerMinute)
	})

	t.Run("YAML list with defaults applied", func(t *testing.T) {
		cfg := loadTestConfig(t, `
auth:
  jwt_issuers:
    - name: clarion
      issuer: https://clarion.example.com
      jwks_url: https://clarion.example.com/jwks.json
      audience: https://fleet.example.com`)
		require.Equal(t, want, cfg.Auth.JWTIssuers)
	})

	t.Run("environment variable JSON matches YAML", func(t *testing.T) {
		cfg := loadTestConfig(t, "",
			`FLEET_AUTH_JWT_ISSUERS=[{"name":"clarion","issuer":"https://clarion.example.com","jwks_url":"https://clarion.example.com/jwks.json","audience":"https://fleet.example.com"}]`)
		require.Equal(t, want, cfg.Auth.JWTIssuers)
	})

	t.Run("explicit values override defaults", func(t *testing.T) {
		cfg := loadTestConfig(t, `
auth:
  jwt_rate_limit_per_minute: 30
  jwt_issuers:
    - name: google
      issuer: https://accounts.google.com
      audience: fleet-api
      subject_claim: email
      allowed_algorithms: [RS256]
      max_token_lifetime: 1h`)
		require.Equal(t, 30, cfg.Auth.JWTRateLimitPerMinute)
		require.Equal(t, []JWTIssuerConfig{{
			Name: "google", Issuer: "https://accounts.google.com", Audience: "fleet-api",
			SubjectClaim: "email", AllowedAlgorithms: []string{"RS256"}, MaxTokenLifetime: time.Hour,
		}}, cfg.Auth.JWTIssuers)
	})

	t.Run("malformed JSON panics", func(t *testing.T) {
		require.Panics(t, func() { loadTestConfig(t, "", `FLEET_AUTH_JWT_ISSUERS=not-json`) })
	})

	t.Run("unknown key panics", func(t *testing.T) {
		// A misspelled key must fail loudly rather than silently fall back to a default.
		require.Panics(t, func() {
			loadTestConfig(t, `
auth:
  jwt_issuers:
    - name: a
      issuer: https://a.example.com
      audience: x
      max_token_lifetme: 1h`)
		})
		require.Panics(t, func() {
			loadTestConfig(t, "",
				`FLEET_AUTH_JWT_ISSUERS=[{"name":"a","issuer":"https://a.example.com","audiance":"x"}]`)
		})
	})

	t.Run("malformed duration panics", func(t *testing.T) {
		require.Panics(t, func() {
			loadTestConfig(t, `
auth:
  jwt_issuers:
    - name: a
      issuer: https://a.example.com
      audience: x
      max_token_lifetime: soon`)
		})
	})
}

func TestValidateJWTIssuers(t *testing.T) {
	valid := func() JWTIssuerConfig {
		return JWTIssuerConfig{
			Name: "a", Issuer: "https://a.example.com", Audience: "fleet",
			SubjectClaim: "sub", AllowedAlgorithms: []string{"RS256"}, MaxTokenLifetime: 15 * time.Minute,
		}
	}
	with := func(f func(*JWTIssuerConfig)) JWTIssuerConfig {
		c := valid()
		f(&c)
		return c
	}

	require.NoError(t, AuthConfig{}.ValidateJWTIssuers())
	require.NoError(t, AuthConfig{JWTIssuers: []JWTIssuerConfig{valid()}}.ValidateJWTIssuers())
	require.NoError(t, AuthConfig{JWTIssuers: []JWTIssuerConfig{
		with(func(c *JWTIssuerConfig) { c.MaxTokenLifetime = 24 * time.Hour; c.Name = strings.Repeat("n", 255) }),
	}}.ValidateJWTIssuers())
	require.NoError(t, AuthConfig{JWTIssuers: []JWTIssuerConfig{
		valid(), with(func(c *JWTIssuerConfig) { c.Name = "b"; c.Issuer = "https://b.example.com" }),
	}}.ValidateJWTIssuers())

	// Loopback issuers may use plain http: traffic never leaves the host.
	for _, u := range []string{"http://127.0.0.1:8081", "http://localhost:8081", "http://[::1]:8081"} {
		require.NoError(t, AuthConfig{JWTIssuers: []JWTIssuerConfig{
			with(func(c *JWTIssuerConfig) { c.Issuer = u; c.JWKSURL = u + "/jwks.json" }),
		}}.ValidateJWTIssuers(), u)
	}

	cases := map[string][]JWTIssuerConfig{
		"http lookalike":   {with(func(c *JWTIssuerConfig) { c.Issuer = "http://localhost.example.com" })},
		"empty name":       {with(func(c *JWTIssuerConfig) { c.Name = "" })},
		"empty issuer":     {with(func(c *JWTIssuerConfig) { c.Issuer = "" })},
		"http issuer":      {with(func(c *JWTIssuerConfig) { c.Issuer = "http://a.example.com" })},
		"http jwks_url":    {with(func(c *JWTIssuerConfig) { c.JWKSURL = "http://a.example.com/keys" })},
		"empty audience":   {with(func(c *JWTIssuerConfig) { c.Audience = "" })},
		"HS256":            {with(func(c *JWTIssuerConfig) { c.AllowedAlgorithms = []string{"HS256"} })},
		"none":             {with(func(c *JWTIssuerConfig) { c.AllowedAlgorithms = []string{"none"} })},
		"PS256":            {with(func(c *JWTIssuerConfig) { c.AllowedAlgorithms = []string{"RS256", "PS256"} })},
		"no algorithms":    {with(func(c *JWTIssuerConfig) { c.AllowedAlgorithms = nil })},
		"zero lifetime":    {with(func(c *JWTIssuerConfig) { c.MaxTokenLifetime = 0 })},
		"negative life":    {with(func(c *JWTIssuerConfig) { c.MaxTokenLifetime = -time.Minute })},
		"lifetime > 24h":   {with(func(c *JWTIssuerConfig) { c.MaxTokenLifetime = 25 * time.Hour })},
		"name too long":    {with(func(c *JWTIssuerConfig) { c.Name = strings.Repeat("n", 256) })},
		"duplicate name":   {valid(), with(func(c *JWTIssuerConfig) { c.Issuer = "https://b.example.com" })},
		"duplicate issuer": {valid(), with(func(c *JWTIssuerConfig) { c.Name = "b" })},
	}
	for name, issuers := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, AuthConfig{JWTIssuers: issuers}.ValidateJWTIssuers())
		})
	}
}
