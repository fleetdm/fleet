package scep

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/enrollment"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/rs/zerolog"
	"github.com/smallstep/scep/x509util"
)

// SigningKey are keys that can generate a crypto.Signer type.
type SigningKey interface {
	// Signer returns a crypto.Signer that uses this key for signing operations.
	// The returned Signer is safe for concurrent use.
	Signer() (crypto.Signer, error)
}

// Client fetches a certificate using SCEP protocol.
// SCEP protocol overview: https://www.cisco.com/c/en/us/support/docs/security-vpn/public-key-infrastructure-pki/116167-technote-scep-00.html
type Client struct {
	// signingKey is a key which will hold the private key of the cert.
	signingKey SigningKey
	// commonName is the CN of the certificate request (required)
	commonName string
	// scepChallenge: SCEP challenge password, which could be static or dynamic.
	scepChallenge string
	// scepURL: The URL of the SCEP server which supports the SCEP protocol (required)
	scepURL string
	timeout *time.Duration
	logger  zerolog.Logger

	insecure bool
	rootCA   string

	// extraExtensions allows adding custom extensions to the CSR
	extraExtensions []pkix.Extension
}

// Option is a functional option for configuring a SCEP Client
type Option func(*Client)

// WithSigningKey sets the private key signer for the certificate request.
func WithSigningKey(key SigningKey) Option {
	return func(c *Client) {
		c.signingKey = key
	}
}

// WithRootCA sets the root CA file to use when connecting to the SCEP server.
func WithRootCA(rootCA string) Option {
	return func(c *Client) {
		c.rootCA = rootCA
	}
}

// WithLogger sets the logger for the Client
func WithLogger(logger zerolog.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithChallenge sets the SCEP challenge password
func WithChallenge(challenge string) Option {
	return func(c *Client) {
		c.scepChallenge = challenge
	}
}

// WithURL sets the SCEP server URL
func WithURL(url string) Option {
	return func(c *Client) {
		c.scepURL = url
	}
}

// WithCommonName sets the common name for the certificate request
func WithCommonName(commonName string) Option {
	return func(c *Client) {
		c.commonName = commonName
	}
}

// WithTimeout configures the timeout for SCEP client requests.
func WithTimeout(timeout *time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// Insecure configures the client to not verify server certificates.
// Only used for tests.
func Insecure() Option {
	return func(c *Client) {
		c.insecure = true
	}
}

// WithExtraExtensions adds custom extensions to the CSR
func WithExtraExtensions(extensions []pkix.Extension) Option {
	return func(c *Client) {
		c.extraExtensions = extensions
	}
}

// NewClient creates a new SCEP client with the provided options
func NewClient(opts ...Option) (*Client, error) {
	// Create client with default options
	c := &Client{
		logger: zerolog.Nop(),
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	if c.timeout == nil {
		// Set a sane default for the timeout.
		c.timeout = ptr.Duration(30 * time.Second)
	}

	// Check that required options are set.
	// SCEP challenge is optional since the SCEP server could allow an empty challenge.
	if c.scepURL == "" || c.commonName == "" || c.signingKey == nil {
		return nil, errors.New("required SCEP client options not set")
	}

	// Set up logger with component tag
	c.logger = c.logger.With().Str("component", "scep").Logger()

	return c, nil
}

// FetchCert fetches and returns a certificate using the SCEP protocol.
func (c *Client) FetchCert(ctx context.Context) (*x509.Certificate, error) {
	// We assume the required fields have already been validated by the NewClient factory.

	slogLogger := slog.New(&zerologSlogHandler{logger: c.logger})
	opts := []scepclient.Option{
		scepclient.WithTimeout(c.timeout),
		scepclient.WithRootCA(c.rootCA),
	}
	if c.insecure {
		opts = append(opts, scepclient.Insecure())
	}

	scepClient, err := scepclient.New(c.scepURL, slogLogger, opts...)
	if err != nil {
		return nil, fmt.Errorf("create SCEP client: %w", err)
	}
	caCerts, err := enrollment.FetchCACerts(ctx, scepClient)
	if err != nil {
		return nil, err
	}

	signer, err := c.signingKey.Signer()
	if err != nil {
		return nil, fmt.Errorf("get signer: %w", err)
	}

	// Generate CSR using signing key
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: c.commonName,
			},
			// Currently, signer.Public() will always be of type *ecdsa.PublicKey.
			SignatureAlgorithm: x509.ECDSAWithSHA256,
			ExtraExtensions:    c.extraExtensions,
		},
		ChallengePassword: c.scepChallenge,
	}

	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, signer)
	if err != nil {
		return nil, fmt.Errorf("create CSR: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	if err != nil {
		return nil, fmt.Errorf("parse CSR: %w", err)
	}

	// The SCEP envelope needs an RSA key for signing and decryption, which the secure hardware key
	// is not, so a temporary one signs it; the CSR itself is signed with the secure hardware key.
	signerKey, signerCert, err := enrollment.NewEphemeralSigner(csr.Subject)
	if err != nil {
		return nil, err
	}
	cert, err := enrollment.Enroll(ctx, scepClient, caCerts, enrollment.Request{
		CSR:        csr,
		SignerKey:  signerKey,
		SignerCert: signerCert,
		Logger:     slogLogger,
	})
	if err != nil {
		return nil, err
	}

	c.logger.Info().Msg("SCEP enrollment successful")
	return cert, nil
}

// zerologSlogHandler adapts zerolog.Logger to slog.Handler so it can be used with *slog.Logger.
type zerologSlogHandler struct {
	logger zerolog.Logger
	attrs  []slog.Attr
}

func (h *zerologSlogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *zerologSlogHandler) Handle(_ context.Context, r slog.Record) error {
	var event *zerolog.Event
	switch r.Level {
	case slog.LevelDebug:
		event = h.logger.Debug()
	case slog.LevelWarn:
		event = h.logger.Warn()
	case slog.LevelError:
		event = h.logger.Error()
	default:
		event = h.logger.Info()
	}
	for _, a := range h.attrs {
		event = event.Interface(a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		event = event.Interface(a.Key, a.Value.Any())
		return true
	})
	event.Msg(r.Message)
	return nil
}

func (h *zerologSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &zerologSlogHandler{logger: h.logger, attrs: slices.Concat(h.attrs, attrs)}
}

func (h *zerologSlogHandler) WithGroup(_ string) slog.Handler {
	return h
}

func PublicKeysEqual(a, b crypto.PublicKey) (bool, error) {
	derA, err := x509.MarshalPKIXPublicKey(a)
	if err != nil {
		return false, fmt.Errorf("marshal a: %w", err)
	}
	derB, err := x509.MarshalPKIXPublicKey(b)
	if err != nil {
		return false, fmt.Errorf("marshal b: %w", err)
	}
	return bytes.Equal(derA, derB), nil
}
