package hostidentity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"time"

	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	kitlog "github.com/go-kit/log"
	"github.com/remitly-oss/httpsig-go"
	"github.com/smallstep/scep"
)

// Config holds the configuration needed for host identity certificate requests
type Config struct {
	ServerAddress string
	EnrollSecret  string
	HostUUID      string
	AgentIndex    int
}

// Client manages host identity certificates and HTTP message signing
type Client struct {
	config            Config
	hostIdentityCert  *x509.Certificate
	hostIdentityKey   *ecdsa.PrivateKey
	httpSigner        *httpsig.Signer
	useHTTPSignatures bool
}

// NewClient creates a new host identity client
func NewClient(config Config, useHTTPSignatures bool) *Client {
	return &Client{
		config:            config,
		useHTTPSignatures: useHTTPSignatures,
	}
}

// createTempRSAKeyAndCert creates a temporary RSA key and certificate for SCEP protocol
func createTempRSAKeyAndCert(hostIdentifier string) (*rsa.PrivateKey, *x509.Certificate, error) {
	// Generate temporary RSA key for SCEP protocol
	tempRSAKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate temp RSA key: %w", err)
	}

	// Create temporary certificate for SCEP
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: hostIdentifier,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(cryptorand.Reader, &certTemplate, &certTemplate, &tempRSAKey.PublicKey, tempRSAKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse temp certificate: %w", err)
	}

	return tempRSAKey, cert, nil
}

// RequestCertificate requests a host identity certificate from Fleet via SCEP
func (c *Client) RequestCertificate() error {
	if !c.useHTTPSignatures {
		return nil // Not needed if not using HTTP signatures
	}

	// Create ECC private key (randomly choose P384 or P256 with 50-50 probability)
	var curve elliptic.Curve
	if rand.Float64() < 0.5 { // nolint:gosec // ignore weak randomizer
		curve = elliptic.P256()
	} else {
		curve = elliptic.P384()
	}
	eccPrivateKey, err := ecdsa.GenerateKey(curve, cryptorand.Reader)
	if err != nil {
		log.Printf("Agent %d: Failed to generate ECC private key: %v", c.config.AgentIndex, err)
		return err
	}

	// Create SCEP client with no-op logger and 30-second timeout
	scepURL := fmt.Sprintf("%s/api/fleet/orbit/host_identity/scep", c.config.ServerAddress)
	timeout := 30 * time.Second
	scepClient, err := scepclient.New(scepURL, kitlog.NewNopLogger(), &timeout)
	if err != nil {
		log.Printf("Agent %d: Failed to create SCEP client: %v", c.config.AgentIndex, err)
		return err
	}

	// Get CA certificate with 30-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, _, err := scepClient.GetCACert(ctx, "")
	if err != nil {
		log.Printf("Agent %d: Failed to get CA cert: %v", c.config.AgentIndex, err)
		return err
	}
	caCerts, err := x509.ParseCertificates(resp)
	if err != nil {
		log.Printf("Agent %d: Failed to parse CA cert: %v", c.config.AgentIndex, err)
		return err
	}
	if len(caCerts) == 0 {
		log.Printf("Agent %d: No CA certificates received", c.config.AgentIndex)
		return errors.New("no CA certificates received")
	}

	// Create host identifier using UUID
	hostIdentifier := c.config.HostUUID

	// Create CSR using ECC key
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: hostIdentifier,
			},
			SignatureAlgorithm: x509.ECDSAWithSHA256,
		},
		ChallengePassword: c.config.EnrollSecret,
	}

	csrDerBytes, err := x509util.CreateCertificateRequest(cryptorand.Reader, &csrTemplate, eccPrivateKey)
	if err != nil {
		log.Printf("Agent %d: Failed to create CSR: %v", c.config.AgentIndex, err)
		return err
	}
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	if err != nil {
		log.Printf("Agent %d: Failed to parse CSR: %v", c.config.AgentIndex, err)
		return err
	}

	// Create temporary RSA key and cert for SCEP protocol
	tempRSAKey, deviceCert, err := createTempRSAKeyAndCert(hostIdentifier)
	if err != nil {
		log.Printf("Agent %d: %v", c.config.AgentIndex, err)
		return err
	}

	// Create SCEP PKI message
	pkiMsgReq := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  caCerts,
		SignerKey:   tempRSAKey, // Use RSA key for SCEP protocol
		SignerCert:  deviceCert,
	}

	msg, err := scep.NewCSRRequest(csr, pkiMsgReq)
	if err != nil {
		log.Printf("Agent %d: Failed to create SCEP message: %v", c.config.AgentIndex, err)
		return err
	}

	// Send PKI operation request
	respBytes, err := scepClient.PKIOperation(ctx, msg.Raw)
	if err != nil {
		log.Printf("Agent %d: SCEP PKI operation failed: %v", c.config.AgentIndex, err)
		return err
	}

	// Parse response
	pkiMsgResp, err := scep.ParsePKIMessage(respBytes, scep.WithCACerts(msg.Recipients))
	if err != nil {
		log.Printf("Agent %d: Failed to parse SCEP response: %v", c.config.AgentIndex, err)
		return err
	}

	// Verify successful response
	if pkiMsgResp.PKIStatus != scep.SUCCESS {
		log.Printf("Agent %d: SCEP request failed with status: %v", c.config.AgentIndex, pkiMsgResp.PKIStatus)
		return fmt.Errorf("SCEP request failed with status: %v", pkiMsgResp.PKIStatus)
	}

	// Decrypt PKI envelope using RSA key
	err = pkiMsgResp.DecryptPKIEnvelope(deviceCert, tempRSAKey)
	if err != nil {
		log.Printf("Agent %d: Failed to decrypt SCEP response: %v", c.config.AgentIndex, err)
		return err
	}

	// Extract the certificate
	certRepMsg := pkiMsgResp.CertRepMessage
	if certRepMsg == nil {
		log.Printf("Agent %d: No certificate in SCEP response", c.config.AgentIndex)
		return errors.New("no certificate in SCEP response")
	}

	cert := certRepMsg.Certificate
	if cert == nil {
		log.Printf("Agent %d: No certificate in CertRepMessage", c.config.AgentIndex)
		return errors.New("no certificate in CertRepMessage")
	}

	// Store the certificate and private key
	c.hostIdentityCert = cert
	c.hostIdentityKey = eccPrivateKey

	// Create an HTTP signer
	var algo httpsig.Algorithm
	switch eccPrivateKey.Curve {
	case elliptic.P256():
		algo = httpsig.Algo_ECDSA_P256_SHA256
	case elliptic.P384():
		algo = httpsig.Algo_ECDSA_P384_SHA384
	default:
		log.Printf("Agent %d: Unsupported curve: %v", c.config.AgentIndex, eccPrivateKey.Curve)
		return fmt.Errorf("unsupported curve: %v", eccPrivateKey.Curve)
	}

	signer, err := httpsig.NewSigner(httpsig.SigningProfile{
		Algorithm: algo,
		Fields:    httpsig.Fields("@method", "@authority", "@path", "@query", "content-digest"),
		Metadata:  []httpsig.Metadata{httpsig.MetaKeyID, httpsig.MetaCreated, httpsig.MetaNonce},
	}, httpsig.SigningKey{
		Key:       eccPrivateKey,
		MetaKeyID: fmt.Sprintf("%X", cert.SerialNumber),
	})
	if err != nil {
		log.Printf("Agent %d: Failed to create HTTP signer: %v", c.config.AgentIndex, err)
		return err
	}

	c.httpSigner = signer

	log.Printf("Agent %d: Successfully obtained host identity certificate with serial %X", c.config.AgentIndex, cert.SerialNumber)
	return nil
}

// SignRequest signs an HTTP request with the host identity certificate if available
func (c *Client) SignRequest(req *http.Request) error {
	if c.httpSigner == nil {
		return nil // No signer available
	}

	return c.httpSigner.Sign(req)
}

// IsEnabled returns whether HTTP message signatures are enabled for this client
func (c *Client) IsEnabled() bool {
	return c.useHTTPSignatures
}

// HasSigner returns whether the client has a valid HTTP signer
func (c *Client) HasSigner() bool {
	return c.httpSigner != nil
}

// GetSigner returns the HTTP signer for use with httpsig.NewHTTPClient
func (c *Client) GetSigner() *httpsig.Signer {
	return c.httpSigner
}
