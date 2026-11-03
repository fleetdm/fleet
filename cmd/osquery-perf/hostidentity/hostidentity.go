package hostidentity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttpsig"
	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/enrollment"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/x509util"
	"github.com/remitly-oss/httpsig-go"
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
	config                       Config
	hostIdentityCert             *x509.Certificate
	hostIdentityKey              *ecdsa.PrivateKey
	httpSigner                   *httpsig.Signer
	useHTTPSignatures            bool
	httpMessageSignatureP384Prob float64
}

// NewClient creates a new host identity client
func NewClient(config Config, useHTTPSignatures bool, httpMessageSignatureP384Prob float64) *Client {
	return &Client{
		config:                       config,
		useHTTPSignatures:            useHTTPSignatures,
		httpMessageSignatureP384Prob: httpMessageSignatureP384Prob,
	}
}

// RequestCertificate requests a host identity certificate from Fleet via SCEP
func (c *Client) RequestCertificate() error {
	if !c.useHTTPSignatures {
		return nil // Not needed if not using HTTP signatures
	}

	// Create ECC private key (randomly choose P384 or P256 with 50-50 probability)
	var curve elliptic.Curve
	if rand.Float64() < c.httpMessageSignatureP384Prob { // nolint:gosec // ignore weak randomizer
		curve = elliptic.P384()
	} else {
		curve = elliptic.P256()
	}
	eccPrivateKey, err := ecdsa.GenerateKey(curve, cryptorand.Reader)
	if err != nil {
		log.Printf("Agent %d: Failed to generate ECC private key: %v", c.config.AgentIndex, err)
		return err
	}

	// Create SCEP client with no-op logger and 30-second timeout
	scepURL := fmt.Sprintf("%s/api/fleet/orbit/host_identity/scep", c.config.ServerAddress)
	timeout := 30 * time.Second
	scepClient, err := scepclient.New(scepURL, slog.New(slog.DiscardHandler),
		scepclient.WithTimeout(&timeout),
		scepclient.Insecure(),
	)
	if err != nil {
		log.Printf("Agent %d: Failed to create SCEP client: %v", c.config.AgentIndex, err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	start := time.Now()
	caCerts, err := enrollment.FetchCACerts(ctx, scepClient)
	if err != nil {
		log.Printf("Agent %d: Failed to get CA cert: %v", c.config.AgentIndex, err)
		return err
	}
	log.Printf("Agent %d: GetCACert duration: %s", c.config.AgentIndex, time.Since(start))

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

	start = time.Now()
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
	log.Printf("Agent %d: create and parse CSR duration: %s", c.config.AgentIndex, time.Since(start))

	// Temporary RSA key and cert for the SCEP envelope, since the CSR key is ECC
	start = time.Now()
	signerKey, signerCert, err := enrollment.NewEphemeralSigner(csr.Subject)
	if err != nil {
		log.Printf("Agent %d: %v", c.config.AgentIndex, err)
		return err
	}
	log.Printf("Agent %d: create temp RSA key and cert: %s", c.config.AgentIndex, time.Since(start))

	start = time.Now()
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	cert, err := enrollment.Enroll(ctx, scepClient, caCerts, enrollment.Request{
		CSR:        csr,
		SignerKey:  signerKey,
		SignerCert: signerCert,
	})
	if err != nil {
		log.Printf("Agent %d: SCEP enrollment failed: %v", c.config.AgentIndex, err)
		return err
	}
	log.Printf("Agent %d: PKIOperation and decrypt duration: %s", c.config.AgentIndex, time.Since(start))

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

	signer, err := fleethttpsig.Signer(fmt.Sprintf("%X", cert.SerialNumber), eccPrivateKey, algo)
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
