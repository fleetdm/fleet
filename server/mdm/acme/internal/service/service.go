// Package service provides the service implementation for the ACME service module.
package service

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/redis_nonces_store"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
	"go.opentelemetry.io/otel"
)

// tracer is an OTEL tracer. It has no-op behavior when OTEL is not enabled.
var tracer = otel.Tracer("github.com/fleetdm/fleet/v4/server/mdm/acme/internal/service")

// Service is the ACME bounded context service implementation.
type Service struct {
	store     types.Datastore
	nonces    *redis_nonces_store.RedisNoncesStore
	providers acme.DataProviders
	logger    *slog.Logger

	// Field to set for testing, if not set it will use the hardcoded Apple Enterprise Attestation Root CA
	TestAppleRootCAs *x509.CertPool
}

type ServiceOption func(*Service)

// NewService creates a new ACME service.
func NewService(
	store types.Datastore,
	redisPool acme.RedisPool,
	providers acme.DataProviders,
	logger *slog.Logger,
	opts ...ServiceOption,
) *Service {
	noncesStore := redis_nonces_store.New(redisPool)
	svc := &Service{
		store:     store,
		nonces:    noncesStore,
		providers: providers,
		logger:    logger,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)

func (s *Service) NoncesStore() *redis_nonces_store.RedisNoncesStore {
	return s.nonces
}

func (s *Service) getACMEBaseURL(ctx context.Context) (string, error) {
	return s.providers.ServerURL(ctx)
}

func (s *Service) getACMEURL(ctx context.Context, pathIdentifier string, suffixes ...string) (string, error) {
	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return "", err
	}

	return s.getACMEURLWithBaseURL(ctx, baseURL, pathIdentifier, suffixes...)
}

func (s *Service) getACMEURLWithBaseURL(_ context.Context, baseURL, pathIdentifier string, suffixes ...string) (string, error) {
	return commonmdm.ResolveURL(baseURL, fmt.Sprintf("/api/mdm/acme/%s/%s", pathIdentifier, strings.Join(suffixes, "/")), true)
}
