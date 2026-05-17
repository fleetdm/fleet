package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
)

// validateServerPrivateKeyExclusive enforces that the server private key may
// be supplied either directly via private_key or via private_key_arn (AWS
// Secrets Manager), but not both.
func validateServerPrivateKeyExclusive(cfg config.FleetConfig) error {
	if cfg.Server.PrivateKey != "" && cfg.Server.PrivateKeySecretArn != "" {
		return errors.New("cannot specify both private_key and private_key_secret_arn")
	}
	return nil
}

// validateServerPrivateKeyLength enforces a 32-byte minimum length on the
// server private key when one is configured. The key is truncated to 32
// bytes after this check because AES-256 requires a 32-byte key; rejecting
// shorter inputs prevents a silently invalid AES-256 setup.
func validateServerPrivateKeyLength(cfg config.FleetConfig) error {
	if len(cfg.Server.PrivateKey) > 0 && len(cfg.Server.PrivateKey) < 32 {
		return errors.New("private key must be at least 32 bytes long")
	}
	return nil
}

// validateOsqueryHostIdentifier rejects osquery_host_identifier values
// outside the supported set. The osquery agent uses this to determine which
// identifier is reported as the host UUID.
func validateOsqueryHostIdentifier(cfg config.FleetConfig) error {
	allowed := map[string]struct{}{
		"provided": {},
		"instance": {},
		"uuid":     {},
		"hostname": {},
	}
	if _, ok := allowed[cfg.Osquery.HostIdentifier]; !ok {
		return fmt.Errorf("%s is not a valid value for osquery_host_identifier", cfg.Osquery.HostIdentifier)
	}
	return nil
}

// validateOTELLoggingConfig enforces the dependency between OTEL logs and
// tracing: log records carry trace IDs only when tracing is enabled, so
// enabling logs without tracing produces correlation-broken telemetry.
func validateOTELLoggingConfig(cfg config.FleetConfig) error {
	if cfg.Logging.OtelLogsEnabled && !cfg.Logging.TracingEnabled {
		return errors.New("logging.otel_logs_enabled requires logging.tracing_enabled to be true")
	}
	return nil
}

// normalizeAndValidateServerURLPrefix trims a trailing slash, ensures a
// leading slash, and validates the resulting prefix against
// allowedURLPrefixRegexp. The mutation is done in place because the
// normalized value is consumed downstream as part of route registration.
func normalizeAndValidateServerURLPrefix(cfg *config.FleetConfig) error {
	if len(cfg.Server.URLPrefix) == 0 {
		return nil
	}
	cfg.Server.URLPrefix = strings.TrimSuffix(cfg.Server.URLPrefix, "/")
	if len(cfg.Server.URLPrefix) > 0 && !strings.HasPrefix(cfg.Server.URLPrefix, "/") {
		cfg.Server.URLPrefix = "/" + cfg.Server.URLPrefix
	}
	if !allowedURLPrefixRegexp.MatchString(cfg.Server.URLPrefix) {
		return fmt.Errorf("prefix must match regexp %q", allowedURLPrefixRegexp.String())
	}
	return nil
}
