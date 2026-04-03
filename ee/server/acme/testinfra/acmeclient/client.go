// Package acmeclient provides a test ACME client for exercising the full
// RFC 8555 flow against a local step-ca or Fleet ACME proxy.
//
// It wraps golang.org/x/crypto/acme with convenience methods for the
// complete certificate issuance flow including challenge completion.
package acmeclient

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/acme"
)

// Client is a test ACME client that exercises the full RFC 8555 flow.
type Client struct {
	acme       *acme.Client
	httpClient *http.Client
	ctx        context.Context
}

// Option configures the test ACME client.
type Option func(*Client)

// WithTLSConfig sets the TLS configuration for the HTTP client.
// Use this to trust the step-ca test CA.
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(c *Client) {
		c.httpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}
}

// WithAccountKey sets a specific account key instead of generating one.
func WithAccountKey(key crypto.Signer) Option {
	return func(c *Client) {
		c.acme.Key = key
	}
}

// New creates a test ACME client pointing at the given directory URL.
func New(directoryURL string, opts ...Option) (*Client, error) {
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating account key: %w", err)
	}

	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		ctx:        context.Background(),
		acme: &acme.Client{
			DirectoryURL: directoryURL,
			Key:          accountKey,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	// Set the HTTP client on the ACME client
	c.acme.HTTPClient = c.httpClient

	return c, nil
}

// RegisterAccount creates a new ACME account.
func (c *Client) RegisterAccount(contacts ...string) (*acme.Account, error) {
	acct := &acme.Account{
		Contact: contacts,
	}
	registered, err := c.acme.Register(c.ctx, acct, acme.AcceptTOS)
	if err != nil {
		return nil, fmt.Errorf("registering account: %w", err)
	}
	return registered, nil
}

// OrderCertificate executes the full ACME certificate issuance flow:
// 1. Create order
// 2. Get authorizations
// 3. Accept challenges (http-01)
// 4. Wait for order to be ready
// 5. Finalize with CSR
// 6. Download certificate
//
// The challengeHandler is called for each http-01 challenge and must make
// the challenge token available at the expected HTTP path. For testing with
// step-ca using the ACME provisioner, challenges may be auto-approved.
//
// Returns the issued certificate chain as DER-encoded certificates.
func (c *Client) OrderCertificate(domain string, challengeHandler func(challenge *acme.Challenge) error) ([]*x509.Certificate, error) {
	// 1. Create order
	order, err := c.acme.AuthorizeOrder(c.ctx, acme.DomainIDs(domain))
	if err != nil {
		return nil, fmt.Errorf("creating order: %w", err)
	}

	// 2. Process each authorization
	for _, authzURL := range order.AuthzURLs {
		authz, err := c.acme.GetAuthorization(c.ctx, authzURL)
		if err != nil {
			return nil, fmt.Errorf("getting authorization %s: %w", authzURL, err)
		}

		if authz.Status == acme.StatusValid {
			continue // Already valid
		}

		// Find a challenge we can handle
		challenge, err := c.selectChallenge(authz.Challenges)
		if err != nil {
			return nil, fmt.Errorf("no suitable challenge for authz %s: %w", authzURL, err)
		}

		// Let the caller set up the challenge response
		if challengeHandler != nil {
			if err := challengeHandler(challenge); err != nil {
				return nil, fmt.Errorf("setting up challenge: %w", err)
			}
		}

		// Accept the challenge
		if _, err := c.acme.Accept(c.ctx, challenge); err != nil {
			return nil, fmt.Errorf("accepting challenge: %w", err)
		}

		// Wait for authorization to become valid
		if _, err := c.acme.WaitAuthorization(c.ctx, authzURL); err != nil {
			return nil, fmt.Errorf("waiting for authorization: %w", err)
		}
	}

	// 3. Wait for order to be ready
	order, err = c.acme.WaitOrder(c.ctx, order.URI)
	if err != nil {
		return nil, fmt.Errorf("waiting for order: %w", err)
	}

	// 4. Generate key and CSR
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating cert key: %w", err)
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: domain,
		},
		DNSNames: []string{domain},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, certKey)
	if err != nil {
		return nil, fmt.Errorf("creating CSR: %w", err)
	}

	// 5. Finalize with CSR
	derChain, _, err := c.acme.CreateOrderCert(c.ctx, order.FinalizeURL, csrDER, true)
	if err != nil {
		return nil, fmt.Errorf("finalizing order: %w", err)
	}

	// 6. Parse certificate chain
	var certs []*x509.Certificate
	for _, der := range derChain {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("parsing certificate: %w", err)
		}
		certs = append(certs, cert)
	}

	return certs, nil
}

// Discover fetches and returns the ACME directory.
func (c *Client) Discover() (acme.Directory, error) {
	return c.acme.Discover(c.ctx)
}

// HTTP01ChallengeResponse returns the key authorization for an http-01 challenge.
func (c *Client) HTTP01ChallengeResponse(challenge *acme.Challenge) (string, error) {
	return c.acme.HTTP01ChallengeResponse(challenge.Token)
}

// HTTP01ChallengePath returns the URL path for an http-01 challenge.
func (c *Client) HTTP01ChallengePath(challenge *acme.Challenge) string {
	return c.acme.HTTP01ChallengePath(challenge.Token)
}

// CertificateToPEM converts DER-encoded certificates to PEM format.
func CertificateToPEM(certs []*x509.Certificate) []byte {
	var pemData []byte
	for _, cert := range certs {
		block := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		pemData = append(pemData, pem.EncodeToMemory(block)...)
	}
	return pemData
}

func (c *Client) selectChallenge(challenges []*acme.Challenge) (*acme.Challenge, error) {
	// Prefer http-01 for testing, fall back to others
	for _, ch := range challenges {
		if ch.Type == "http-01" {
			return ch, nil
		}
	}
	// Accept any challenge type if http-01 not available
	if len(challenges) > 0 {
		return challenges[0], nil
	}
	return nil, fmt.Errorf("no challenges available")
}
