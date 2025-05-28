package scep

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/rs/zerolog"
	"github.com/smallstep/scep"
	"github.com/smallstep/scep/x509util"
)

const (
	rsaKeySize = 4096
)

// Client fetches a certificate using SCEP protocol.
// SCEP protocol overview: https://www.cisco.com/c/en/us/support/docs/security-vpn/public-key-infrastructure-pki/116167-technote-scep-00.html
type Client struct {
	// commonName is the CN of the certificate request (required)
	commonName string
	// scepChallenge: SCEP challenge password, which could be static or dynamic.
	scepChallenge string
	// scepUrl: The URL of the SCEP server which supports the SCEP protocol (required)
	scepUrl string
	// certDestDir: The destination directory where retrieved cert and its private key will be saved (required)
	certDestDir string
	timeout     time.Duration
	logger      zerolog.Logger
}

// Option is a functional option for configuring a SCEP Client
type Option func(*Client)

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
		c.scepUrl = url
	}
}

// WithCertDestDir sets the directory where certificates will be saved
func WithCertDestDir(dir string) Option {
	return func(c *Client) {
		c.certDestDir = dir
	}
}

// WithCommonName sets the common name for the certificate request
func WithCommonName(commonName string) Option {
	return func(c *Client) {
		c.commonName = commonName
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// NewClient creates a new SCEP client with the provided options
func NewClient(opts ...Option) (*Client, error) {
	// Create client with default options
	c := &Client{
		logger:  zerolog.Nop(),
		timeout: 30 * time.Second,
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Check that required options are set.
	// SCEP challenge is optional since the SCEP server could allow an empty challenge.
	if c.scepUrl == "" || c.certDestDir == "" || c.commonName == "" {
		return nil, errors.New("required SCEP client options not set")
	}

	return c, nil
}

// FetchAndSaveCert fetches a certificate using SCEP protocol and saves it on disk
func (c *Client) FetchAndSaveCert(ctx context.Context) error {
	// We assume the required fields have already been validated by the NewClient factory.

	kitLogger := &zerologAdapter{logger: c.logger}
	scepClient, err := scepclient.New(c.scepUrl, kitLogger, &c.timeout)
	if err != nil {
		return fmt.Errorf("create SCEP client: %w", err)
	}

	resp, _, err := scepClient.GetCACert(ctx, "")
	if err != nil {
		return fmt.Errorf("get CA cert: %w", err)
	}
	caCert, err := x509.ParseCertificates(resp)
	if err != nil {
		return fmt.Errorf("parse CA cert: %w", err)
	}

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return fmt.Errorf("generate RSA private key: %w", err)
	}

	// Generate CSR
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: c.commonName,
			},
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
		ChallengePassword: c.scepChallenge,
	}

	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		return fmt.Errorf("create CSR: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	if err != nil {
		return fmt.Errorf("parse CSR: %w", err)
	}

	// Create a self-signed certificate for client authentication
	deviceCertificateTemplate := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   c.commonName,
			Organization: csr.Subject.Organization,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	deviceCertificateDerBytes, err := x509.CreateCertificate(
		rand.Reader,
		&deviceCertificateTemplate,
		&deviceCertificateTemplate,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return fmt.Errorf("create device certificate: %w", err)
	}

	deviceCertificateForRequest, err := x509.ParseCertificate(deviceCertificateDerBytes)
	if err != nil {
		return fmt.Errorf("parse device certificate: %w", err)
	}

	// Send PKCSReq message to SCEP server
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCert,
		SignerKey:   privateKey,
		SignerCert:  deviceCertificateForRequest,
		CSRReqMessage: &scep.CSRReqMessage{
			ChallengePassword: c.scepChallenge,
		},
	}

	msg, err := scep.NewCSRRequest(csr, pkiMsgReq, scep.WithLogger(kitLogger))
	if err != nil {
		return fmt.Errorf("create CSR request: %w", err)
	}

	respBytes, err := scepClient.PKIOperation(ctx, msg.Raw)
	if err != nil {
		return fmt.Errorf("do CSR request: %w", err)
	}

	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(kitLogger), scep.WithCACerts(msg.Recipients))
	if err != nil {
		return fmt.Errorf("parse PKIMessage response: %w", err)
	}

	if pkiMsgResp.PKIStatus != scep.SUCCESS {
		return fmt.Errorf("PKIMessage CSR request failed with code: %s, fail info: %s", pkiMsgResp.PKIStatus, pkiMsgResp.FailInfo)
	}

	if err := pkiMsgResp.DecryptPKIEnvelope(deviceCertificateForRequest, privateKey); err != nil {
		return fmt.Errorf("decrypt PKI envelope: %w", err)
	}

	// Save the certificate and private key
	cert := pkiMsgResp.CertRepMessage.Certificate
	if err := c.saveCertAndKey(cert, privateKey); err != nil {
		return fmt.Errorf("save cert and key: %w", err)
	}

	c.logger.Info().Msg("SCEP enrollment successful")
	return nil
}

// saveCertAndKey saves the certificate and private key to the certDestDir
func (c *Client) saveCertAndKey(cert *x509.Certificate, key *rsa.PrivateKey) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(c.certDestDir, 0o755); err != nil {
		return fmt.Errorf("create cert directory: %w", err)
	}

	// Save certificate
	certPath := filepath.Join(c.certDestDir, constant.FleetTLSClientCertificateFileName)
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("create cert file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}); err != nil {
		return fmt.Errorf("encode cert: %w", err)
	}

	// Save the private key
	keyPath := filepath.Join(c.certDestDir, constant.FleetTLSClientKeyFileName)
	keyFile, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("create key file: %w", err)
	}
	defer keyFile.Close()

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	if err := pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}); err != nil {
		return fmt.Errorf("encode key: %w", err)
	}

	c.logger.Info().
		Str("cert_path", certPath).
		Str("key_path", keyPath).
		Msg("Saved certificate and private key")

	return nil
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
