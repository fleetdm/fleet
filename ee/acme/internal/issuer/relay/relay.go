// Package relay implements a CertificateIssuer that relays certificate
// operations to an upstream ACME CA (e.g., Hydrant, Sectigo, Smallstep).
//
// The relay maintains its own ACME client session with each upstream CA,
// using a dedicated account key. When the ACME server calls IssueCertificate,
// the relay creates an order on the upstream CA, completes any required
// challenges, and forwards the CSR to obtain the certificate.
package relay

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"golang.org/x/crypto/acme"
)

// Backend implements api.CertificateIssuer by relaying to upstream ACME CAs.
type Backend struct {
	upstreams map[string]*upstreamCA
	logger    *slog.Logger
	mu        sync.RWMutex
}

// upstreamCA holds the state for a single upstream CA connection.
type upstreamCA struct {
	config     *api.CAConfig
	acmeClient *acme.Client
	account    *acme.Account // lazily registered
}

// New creates a new relay backend.
func New(logger *slog.Logger) *Backend {
	return &Backend{
		upstreams: make(map[string]*upstreamCA),
		logger:    logger,
	}
}

// AddCA registers an upstream CA configuration.
func (b *Backend) AddCA(cfg *api.CAConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	httpClient, err := buildHTTPClient(cfg)
	if err != nil {
		return fmt.Errorf("building HTTP client for CA %q: %w", cfg.Name, err)
	}

	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating account key for CA %q: %w", cfg.Name, err)
	}

	b.upstreams[cfg.Name] = &upstreamCA{
		config: cfg,
		acmeClient: &acme.Client{
			DirectoryURL: cfg.DirectoryURL,
			Key:          accountKey,
			HTTPClient:   httpClient,
		},
	}

	b.logger.Info("registered upstream CA", "ca", cfg.Name, "directory", cfg.DirectoryURL)
	return nil
}

// ValidateChallenge validates a device's challenge response.
//
// For the relay, this means validating the device attestation locally.
// The upstream challenge (http-01 etc.) is handled separately during
// IssueCertificate when the relay creates the upstream order.
//
// For the POC, this is a no-op that accepts all challenges.
func (b *Backend) ValidateChallenge(ctx context.Context, challenge *api.Challenge, order *api.Order) error {
	b.logger.Info("validating challenge",
		"ca", order.CAName,
		"challenge_type", challenge.Type,
		"challenge_id", challenge.ID,
	)

	// TODO: Implement device attestation validation.
	// For the POC, accept all challenges.
	return nil
}

// IssueCertificate creates an order on the upstream CA, completes any required
// challenges, and forwards the CSR to obtain the certificate.
//
// This is the core of the relay: it manages a complete ACME flow with the
// upstream CA on behalf of the device.
func (b *Backend) IssueCertificate(ctx context.Context, csr *x509.CertificateRequest, order *api.Order) (*api.IssuedCertificate, error) {
	b.mu.RLock()
	upstream, ok := b.upstreams[order.CAName]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown CA: %s", order.CAName)
	}

	// Ensure we have an account with the upstream CA
	if err := b.ensureAccount(ctx, upstream); err != nil {
		return nil, fmt.Errorf("ensuring upstream account: %w", err)
	}

	// Convert identifiers to ACME AuthzIDs
	var authzIDs []acme.AuthzID
	for _, id := range order.Identifiers {
		authzIDs = append(authzIDs, acme.AuthzID{Type: id.Type, Value: id.Value})
	}

	// 1. Create order on upstream CA
	upstreamOrder, err := upstream.acmeClient.AuthorizeOrder(ctx, authzIDs)
	if err != nil {
		return nil, fmt.Errorf("creating upstream order: %w", err)
	}
	b.logger.Info("created upstream order", "ca", order.CAName, "status", upstreamOrder.Status)

	// 2. Complete upstream challenges
	for _, authzURL := range upstreamOrder.AuthzURLs {
		if err := b.completeAuthorization(ctx, upstream, authzURL); err != nil {
			return nil, fmt.Errorf("completing upstream authorization: %w", err)
		}
	}

	// 3. Wait for order to be ready
	upstreamOrder, err = upstream.acmeClient.WaitOrder(ctx, upstreamOrder.URI)
	if err != nil {
		return nil, fmt.Errorf("waiting for upstream order: %w", err)
	}

	// 4. Finalize with CSR
	derChain, _, err := upstream.acmeClient.CreateOrderCert(ctx, upstreamOrder.FinalizeURL, csr.Raw, true)
	if err != nil {
		return nil, fmt.Errorf("finalizing upstream order: %w", err)
	}

	b.logger.Info("certificate issued by upstream CA",
		"ca", order.CAName,
		"chain_length", len(derChain),
	)

	// 5. Parse the leaf certificate for metadata
	result := &api.IssuedCertificate{
		DERChain: derChain,
	}
	if len(derChain) > 0 {
		leaf, err := x509.ParseCertificate(derChain[0])
		if err == nil {
			result.Leaf = leaf
			result.SerialNumber = leaf.SerialNumber
			result.NotBefore = leaf.NotBefore
			result.NotAfter = leaf.NotAfter
		}
	}

	return result, nil
}

// RevokeCertificate revokes a certificate on the upstream CA.
func (b *Backend) RevokeCertificate(ctx context.Context, cert *x509.Certificate, reason int) error {
	// TODO: Determine which upstream CA issued this cert and revoke it.
	return fmt.Errorf("revocation not yet implemented")
}

// HTTP01ChallengeResponse returns the key authorization for an http-01 challenge
// token for the given CA. This is needed for http-01 challenge validation where
// the relay must serve the correct response at /.well-known/acme-challenge/<token>.
func (b *Backend) HTTP01ChallengeResponse(caName, token string) (string, error) {
	b.mu.RLock()
	upstream, ok := b.upstreams[caName]
	b.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("unknown CA: %s", caName)
	}
	return upstream.acmeClient.HTTP01ChallengeResponse(token)
}

// completeAuthorization handles a single upstream authorization by finding
// and completing a supported challenge.
func (b *Backend) completeAuthorization(ctx context.Context, upstream *upstreamCA, authzURL string) error {
	authz, err := upstream.acmeClient.GetAuthorization(ctx, authzURL)
	if err != nil {
		return fmt.Errorf("getting authorization: %w", err)
	}

	if authz.Status == acme.StatusValid {
		return nil // Already valid
	}

	// Find a challenge to complete
	challenge, err := selectChallenge(authz.Challenges)
	if err != nil {
		return err
	}

	// Accept the challenge (tells upstream to validate)
	if _, err := upstream.acmeClient.Accept(ctx, challenge); err != nil {
		return fmt.Errorf("accepting challenge: %w", err)
	}

	// Wait for authorization to become valid
	if _, err := upstream.acmeClient.WaitAuthorization(ctx, authzURL); err != nil {
		return fmt.Errorf("waiting for authorization: %w", err)
	}

	return nil
}

func (b *Backend) ensureAccount(ctx context.Context, upstream *upstreamCA) error {
	if upstream.account != nil {
		return nil
	}

	acct, err := upstream.acmeClient.Register(ctx, &acme.Account{
		Contact: []string{"mailto:fleet-acme-relay@fleet.local"},
	}, acme.AcceptTOS)
	if err != nil {
		return fmt.Errorf("registering upstream account: %w", err)
	}

	upstream.account = acct
	b.logger.Info("registered upstream account", "ca", upstream.config.Name, "uri", acct.URI)
	return nil
}

func selectChallenge(challenges []*acme.Challenge) (*acme.Challenge, error) {
	// Prefer http-01 for testing
	for _, ch := range challenges {
		if ch.Type == "http-01" {
			return ch, nil
		}
	}
	if len(challenges) > 0 {
		return challenges[0], nil
	}
	return nil, fmt.Errorf("no challenges available")
}

func buildHTTPClient(cfg *api.CAConfig) (*http.Client, error) {
	tlsConfig := &tls.Config{}

	if len(cfg.CACert) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(cfg.CACert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = pool
	}

	if len(cfg.ClientCert) > 0 && len(cfg.ClientKey) > 0 {
		cert, err := tls.X509KeyPair(cfg.ClientCert, cfg.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("loading client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 30 * time.Second,
	}, nil
}
