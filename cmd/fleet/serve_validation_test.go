package main

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/require"
)

func TestValidateServerPrivateKeyExclusive(t *testing.T) {
	for _, tc := range []struct {
		name             string
		privateKey       string
		privateKeyArn    string
		wantErrSubstring string
	}{
		{"both empty", "", "", ""},
		{"direct key only", "some-direct-key-value", "", ""},
		{"arn only", "", "arn:aws:secretsmanager:us-east-1:123:secret:foo", ""},
		{
			"both set rejected",
			"some-direct-key-value",
			"arn:aws:secretsmanager:us-east-1:123:secret:foo",
			"cannot specify both private_key and private_key_secret_arn",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.FleetConfig{
				Server: config.ServerConfig{
					PrivateKey:          tc.privateKey,
					PrivateKeySecretArn: tc.privateKeyArn,
				},
			}
			err := validateServerPrivateKeyExclusive(cfg)
			if tc.wantErrSubstring == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrSubstring)
		})
	}
}

func TestValidateServerPrivateKeyLength(t *testing.T) {
	for _, tc := range []struct {
		name             string
		privateKey       string
		wantErrSubstring string
	}{
		{"empty key", "", ""},
		{"key under minimum length", strings.Repeat("x", 16), "at least 32 bytes long"},
		{"key one byte under minimum", strings.Repeat("x", 31), "at least 32 bytes long"},
		{"key at exactly minimum length", strings.Repeat("x", 32), ""},
		{"key over minimum length", strings.Repeat("x", 64), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.FleetConfig{
				Server: config.ServerConfig{
					PrivateKey: tc.privateKey,
				},
			}
			err := validateServerPrivateKeyLength(cfg)
			if tc.wantErrSubstring == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrSubstring)
		})
	}
}

func TestValidateOsqueryHostIdentifier(t *testing.T) {
	for _, tc := range []struct {
		name             string
		identifier       string
		wantErrSubstring string
	}{
		{"provided is allowed", "provided", ""},
		{"instance is allowed", "instance", ""},
		{"uuid is allowed", "uuid", ""},
		{"hostname is allowed", "hostname", ""},
		{"empty string rejected", "", "is not a valid value for osquery_host_identifier"},
		{"unknown value rejected", "serial", "is not a valid value for osquery_host_identifier"},
		{"case-sensitive: UUID rejected", "UUID", "is not a valid value for osquery_host_identifier"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.FleetConfig{
				Osquery: config.OsqueryConfig{HostIdentifier: tc.identifier},
			}
			err := validateOsqueryHostIdentifier(cfg)
			if tc.wantErrSubstring == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrSubstring)
		})
	}
}

func TestValidateOTELLoggingConfig(t *testing.T) {
	for _, tc := range []struct {
		name             string
		otelLogs         bool
		tracing          bool
		wantErrSubstring string
	}{
		{"both disabled", false, false, ""},
		{"tracing only", false, true, ""},
		{"both enabled", true, true, ""},
		{"otel logs without tracing rejected", true, false, "logging.otel_logs_enabled requires logging.tracing_enabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.FleetConfig{
				Logging: config.LoggingConfig{
					OtelLogsEnabled: tc.otelLogs,
					TracingEnabled:  tc.tracing,
				},
			}
			err := validateOTELLoggingConfig(cfg)
			if tc.wantErrSubstring == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrSubstring)
		})
	}
}

func TestNormalizeAndValidateServerURLPrefix(t *testing.T) {
	for _, tc := range []struct {
		name             string
		input            string
		wantNormalized   string
		wantErrSubstring string
	}{
		{"empty prefix is left alone", "", "", ""},
		{"slash-prefixed value passes unchanged", "/fleet", "/fleet", ""},
		{"leading slash added", "fleet", "/fleet", ""},
		{"trailing slash trimmed", "/fleet/", "/fleet", ""},
		{"both fixes applied", "fleet/", "/fleet", ""},
		{"nested path is allowed", "/api/v1", "/api/v1", ""},
		{"single slash trims to empty and is rejected by regex", "/", "", "must match regexp"},
		{"invalid character rejected", "/fleet space", "", "must match regexp"},
		{"query-style suffix rejected", "/fleet?x=1", "", "must match regexp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.FleetConfig{
				Server: config.ServerConfig{URLPrefix: tc.input},
			}
			err := normalizeAndValidateServerURLPrefix(&cfg)
			if tc.wantErrSubstring == "" {
				require.NoError(t, err)
				require.Equal(t, tc.wantNormalized, cfg.Server.URLPrefix)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrSubstring)
		})
	}
}
