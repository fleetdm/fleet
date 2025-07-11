package scep

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
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

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/tee"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/rs/zerolog"
	"github.com/smallstep/scep"
	"github.com/smallstep/scep/x509util"
)

// Client fetches a certificate using SCEP protocol.
// SCEP protocol overview: https://www.cisco.com/c/en/us/support/docs/security-vpn/public-key-infrastructure-pki/116167-technote-scep-00.html
type Client struct {
	// teeDevice is a TPM/TEE device which will hold the private key of the cert
	teeDevice tee.TEE
	// commonName is the CN of the certificate request (required)
	commonName string
	// scepChallenge: SCEP challenge password, which could be static or dynamic.
	scepChallenge string
	// scepURL: The URL of the SCEP server which supports the SCEP protocol (required)
	scepURL string
	// certDestDir: The destination directory where retrieved cert will be saved (required)
	certDestDir string
	timeout     time.Duration
	logger      zerolog.Logger
}

// Option is a functional option for configuring a SCEP Client
type Option func(*Client)

func WithTEE(tee tee.TEE) Option {
	return func(c *Client) {
		c.teeDevice = tee
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
	if c.scepURL == "" || c.certDestDir == "" || c.commonName == "" || c.teeDevice == nil {
		return nil, errors.New("required SCEP client options not set")
	}

	// Set up logger with component tag
	c.logger = c.logger.With().Str("component", "scep").Logger()

	return c, nil
}

// FetchAndSaveCert fetches a certificate using SCEP protocol and saves it on disk
func (c *Client) FetchAndSaveCert(ctx context.Context) error {
	// We assume the required fields have already been validated by the NewClient factory.

	kitLogger := &zerologAdapter{logger: c.logger}
	scepClient, err := scepclient.New(c.scepURL, kitLogger, &c.timeout)
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

	// Initialize TEE (TPM 2.0 on Linux)
	// TODO: These paths should be configurable
	// publicBlobPath := filepath.Join(c.certDestDir, "tpm_public.blob")
	// privateBlobPath := filepath.Join(c.certDestDir, "tpm_private.blob")
	// teeDevice, err := tee.NewTPM2(
	// 	tee.WithLogger(c.logger),
	// 	tee.WithPublicBlobPath(publicBlobPath),
	// 	tee.WithPrivateBlobPath(privateBlobPath),
	// )
	// if err != nil {
	// 	return fmt.Errorf("initialize TEE: %w", err)
	// }
	// defer teeDevice.Close()

	// Create a key in TEE for signing
	teeKey, err := c.teeDevice.CreateKey(ctx)
	if err != nil {
		return fmt.Errorf("create TEE key: %w", err)
	}
	defer teeKey.Close()

	// Get signer from TEE key
	teeSigner, err := teeKey.Signer()
	if err != nil {
		return fmt.Errorf("get TEE signer: %w", err)
	}

	// Get public key
	publicKey := teeSigner.Public()

	// Create a temporary RSA key pair in memory for SCEP envelope decryption
	// ECC keys cannot be used for decryption, so we need RSA for this purpose
	tempRSAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate temporary RSA key: %w", err)
	}

	// Determine signature algorithm based on the key type
	var sigAlg x509.SignatureAlgorithm
	switch publicKey.(type) {
	case *ecdsa.PublicKey:
		sigAlg = x509.ECDSAWithSHA256
	case *rsa.PublicKey:
		sigAlg = x509.SHA256WithRSA
	default:
		return fmt.Errorf("unsupported key type: %T", publicKey)
	}

	// Generate CSR using TEE key
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: c.commonName,
			},
			SignatureAlgorithm: sigAlg,
		},
		ChallengePassword: c.scepChallenge,
	}

	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, teeSigner)
	if err != nil {
		return fmt.Errorf("create CSR: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	if err != nil {
		return fmt.Errorf("parse CSR: %w", err)
	}

	// Create a self-signed certificate for SCEP protocol using the temporary RSA key
	// The SCEP protocol requires RSA for both signing and decryption
	// The actual CSR will be signed with the ECC key from TEE
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
		&tempRSAKey.PublicKey,
		tempRSAKey,
	)
	if err != nil {
		return fmt.Errorf("create device certificate: %w", err)
	}

	deviceCertificateForRequest, err := x509.ParseCertificate(deviceCertificateDerBytes)
	if err != nil {
		return fmt.Errorf("parse device certificate: %w", err)
	}

	// Send PKCSReq message to SCEP server
	// Use RSA key for SCEP protocol (signing and decryption)
	// The CSR itself was already signed with the ECC key from TEE
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

	// Use the temporary RSA key for decryption (ECC keys don't support decryption)
	if err := pkiMsgResp.DecryptPKIEnvelope(deviceCertificateForRequest, tempRSAKey); err != nil {
		return fmt.Errorf("decrypt PKI envelope: %w", err)
	}

	if err := c.saveCert(pkiMsgResp.CertRepMessage.Certificate); err != nil {
		return fmt.Errorf("save cert and TEE key: %w", err)
	}

	c.logger.Info().Msg("SCEP enrollment successful")
	return nil
}

// saveCert saves the certificate and TEE key context to the certDestDir
func (c *Client) saveCert(cert *x509.Certificate) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(c.certDestDir, 0o755); err != nil {
		return fmt.Errorf("create cert directory: %w", err)
	}

	// Save certificate
	certPath := filepath.Join(c.certDestDir, constant.FleetHTTPSignatureCertificateFileName)
	certFile, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
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

	c.logger.Info().
		Str("cert_path", certPath).
		Msg("Saved certificate")

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

func GetCert(path string) (*x509.Certificate, error) {
	// Read certificate file
	certPEMBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read certificate file: %w", err)
	}

	// Decode PEM block
	block, _ := pem.Decode(certPEMBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("failed to decode PEM block containing certificate")
	}

	return x509.ParseCertificate(block.Bytes)
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
