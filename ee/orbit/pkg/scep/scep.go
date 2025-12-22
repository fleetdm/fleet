package scep

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"time"

	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/rs/zerolog"
	"github.com/smallstep/scep"
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

	kitLogger := &zerologAdapter{logger: c.logger}
	opts := []scepclient.Option{
		scepclient.WithTimeout(c.timeout),
		scepclient.WithRootCA(c.rootCA),
	}
	if c.insecure {
		opts = append(opts, scepclient.Insecure())
	}

	scepClient, err := scepclient.New(c.scepURL, kitLogger, opts...)
	if err != nil {
		return nil, fmt.Errorf("create SCEP client: %w", err)
	}
	resp, _, err := scepClient.GetCACert(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("get CA cert: %w", err)
	}
	caCert, err := x509.ParseCertificates(resp)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	signer, err := c.signingKey.Signer()
	if err != nil {
		return nil, fmt.Errorf("get signer: %w", err)
	}

	// Create a temporary RSA key pair in memory for SCEP envelope decryption
	// ECC keys cannot be used for decryption, so we need RSA for this purpose
	tempRSAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate temporary RSA key: %w", err)
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

	// Create a self-signed certificate for SCEP protocol using the temporary RSA key
	// The SCEP protocol requires RSA for both signing and decryption
	// The actual CSR will be signed with the ECC key.
	deviceCertificateTemplate := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   c.commonName,
			Organization: csr.Subject.Organization,
		},

		// The server will set these on the final certificate,
		// but we need to set them otherwise the CSR is rejected.
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	deviceCertificateDerBytes, err := x509.CreateCertificate(
		rand.Reader,
		&deviceCertificateTemplate,
		&deviceCertificateTemplate,
		&tempRSAKey.PublicKey,
		tempRSAKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create device certificate: %w", err)
	}

	deviceCertificateForRequest, err := x509.ParseCertificate(deviceCertificateDerBytes)
	if err != nil {
		return nil, fmt.Errorf("parse device certificate: %w", err)
	}

	// Send PKCSReq message to SCEP server
	// Use RSA key for SCEP protocol (signing and decryption)
	// The CSR itself was already signed with the signing key.
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCert,
		SignerKey:   tempRSAKey, // Use RSA key for SCEP protocol
		SignerCert:  deviceCertificateForRequest,
		CSRReqMessage: &scep.CSRReqMessage{
			ChallengePassword: c.scepChallenge,
		},
	}

	msg, err := scep.NewCSRRequest(csr, pkiMsgReq, scep.WithLogger(kitLogger))
	if err != nil {
		return nil, fmt.Errorf("create CSR request: %w", err)
	}

	respBytes, err := scepClient.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return nil, fmt.Errorf("do CSR request: %w", err)
	}

	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(kitLogger), scep.WithCACerts(msg.Recipients))
	if err != nil {
		return nil, fmt.Errorf("parse PKIMessage response: %w", err)
	}

	if pkiMsgResp.PKIStatus != scep.SUCCESS {
		return nil, fmt.Errorf("PKIMessage CSR request failed with code: %s, fail info: %s", pkiMsgResp.PKIStatus, pkiMsgResp.FailInfo)
	}

	// Use the temporary RSA key for decryption (ECC keys don't support decryption)
	if err := pkiMsgResp.DecryptPKIEnvelope(deviceCertificateForRequest, tempRSAKey); err != nil {
		return nil, fmt.Errorf("decrypt PKI envelope: %w", err)
	}

	c.logger.Info().Msg("SCEP enrollment successful")
	return pkiMsgResp.CertRepMessage.Certificate, nil
}

// zerologAdapter adapts zerolog.Logger to kit/log.Logger
type zerologAdapter struct {
	logger zerolog.Logger
}

// Log implements the kit/log.Logger interface
func (a *zerologAdapter) Log(keyvals ...interface{}) error {
	// Convert key-value pairs to a map
	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key, ok := keyvals[i].(string)
			if ok {
				fields[key] = keyvals[i+1]
			}
		}
	}

	// Extract message if present
	msg := ""
	if msgVal, ok := fields["msg"]; ok {
		if msgStr, ok := msgVal.(string); ok {
			msg = msgStr
			delete(fields, "msg")
		}
	}

	// Log with zerolog
	event := a.logger.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)

	return nil
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
