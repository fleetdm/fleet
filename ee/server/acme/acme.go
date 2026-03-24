// Package acme provides the ACME certificate bounded context for Fleet.
//
// It implements an ACME server (RFC 8555) that faces devices and delegates
// certificate issuance to pluggable backends via the CertificateIssuer interface.
package acme

import (
	"log/slog"

	"github.com/fleetdm/fleet/v4/ee/server/acme/api"
	"github.com/fleetdm/fleet/v4/ee/server/acme/internal/issuer/relay"
	"github.com/fleetdm/fleet/v4/ee/server/acme/internal/server"
)

// PathPrefix is the URL path prefix for all ACME server endpoints.
const PathPrefix = server.PathPrefix

// Server is the ACME server that faces devices.
type Server = server.Server

// NewServer creates a new ACME server.
func NewServer(baseURL string, logger *slog.Logger) *Server {
	return server.New(baseURL, logger)
}

// RelayBackend implements CertificateIssuer by relaying to upstream ACME CAs.
type RelayBackend = relay.Backend

// NewRelayBackend creates a new relay backend.
func NewRelayBackend(logger *slog.Logger) *RelayBackend {
	return relay.New(logger)
}

// CAConfig is re-exported for convenience.
type CAConfig = api.CAConfig
