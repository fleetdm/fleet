package hostidentity

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/scep"
	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/securehw"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/rs/zerolog"
)

// Credentials holds a certificate and its corresponding private key handle stored in secure hardware.
type Credentials struct {
	// Certificate holds the public certificate issued via SCEP.
	Certificate *x509.Certificate
	// SecureHWKey holds the private key protected by secure hardware.
	SecureHWKey securehw.Key

	// CertificatePath is the file path to the public certificate issued via SCEP.
	CertificatePath string

	secureHW securehw.TEE
}

// Close releases key resources.
func (c *Credentials) Close() {
	c.secureHW.Close()
}

// Setup creates a private key using a TEE and generates a new client
// certificate using SCEP.
// If there's already a key and certificate in the metadata directory it will return them.
// The returned Credentials needs to be closed after its use.
func Setup(
	ctx context.Context,
	metadataDir string,
	scepURL string,
	scepChallenge string,
	commonName string,
	rootCA string,
	insecure bool,
	logger zerolog.Logger,
) (*Credentials, error) {
	teeDevice, err := securehw.New(metadataDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TEE device: %w", err)
	}
	secureHWKey, err := teeDevice.LoadKey()
	switch {
	case err == nil:
		// OK
	case errors.As(err, &securehw.ErrKeyNotFound{}):
		// Key doesn't exist yet, let's create it.
		secureHWKey, err = teeDevice.CreateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to create TEE key: %w", err)
		}
	default:
		return nil, fmt.Errorf("failed to load TEE key: %w", err)
	}

	clientCert, err := loadSCEPClientCert(metadataDir)
	switch {
	case err == nil:
		// OK, we have a certificate already, let's use it.
	case errors.Is(err, os.ErrNotExist):
		// We don't have a certificate, let's issue one using SCEP.
		opts := []scep.Option{
			scep.WithRootCA(rootCA),
			scep.WithSigningKey(secureHWKey),
			scep.WithLogger(logger),
			scep.WithURL(scepURL),
			scep.WithChallenge(scepChallenge),
			scep.WithCommonName(commonName),
		}
		if insecure {
			opts = append(opts, scep.Insecure())
		}
		scepClient, err := scep.NewClient(opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create SCEP client: %w", err)
		}
		clientCert, err = scepClient.FetchCert(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch certificate using SCEP: %w", err)
		}
		if err := saveSCEPClientCert(metadataDir, clientCert); err != nil {
			return nil, fmt.Errorf("failed to save certificate: %w", err)
		}
	}

	// Sanity check in case the public key material on the secure HW
	// does not match the certificate public key.
	// This can happen if something or someone deletes the private and public blobs
	// and they are re-generated at startup.

	secureHWPubKey, err := secureHWKey.Public()
	if err != nil {
		return nil, fmt.Errorf("error getting public key from secure HW key: %w", err)
	}
	keysEqual, err := scep.PublicKeysEqual(secureHWPubKey, clientCert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("error comparing public keys: %w", err)
	}
	if !keysEqual {
		// Cleanup the certificate in the metadata directory so that on next start up it will re-issue
		// a new certificate.
		certPath := filepath.Join(metadataDir, constant.FleetHTTPSignatureCertificateFileName)
		if err := os.Remove(certPath); err != nil {
			return nil, fmt.Errorf("error cleaning up %s: %w", certPath, err)
		}
		return nil, fmt.Errorf("secure HW key does not match certificate public key, deleted %q to re-issue a new certificate in the next restart", certPath)
	}
	logger.Debug().Msg("secure HW key matches certificate public key")

	return &Credentials{
		Certificate:     clientCert,
		SecureHWKey:     secureHWKey,
		CertificatePath: filepath.Join(metadataDir, constant.FleetHTTPSignatureCertificateFileName),

		secureHW: teeDevice,
	}, nil
}

func loadSCEPClientCert(metadataDir string) (*x509.Certificate, error) {
	certPath := filepath.Join(metadataDir, constant.FleetHTTPSignatureCertificateFileName)
	certPEMBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", certPath, err)
	}
	block, _ := pem.Decode(certPEMBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("failed to decode PEM block containing certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

func saveSCEPClientCert(metadataDir string, cert *x509.Certificate) error {
	certPath := filepath.Join(metadataDir, constant.FleetHTTPSignatureCertificateFileName)
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
	return nil
}
