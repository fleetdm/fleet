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

// ClientCertificate holds a certificate and its corresponding private key handle stored in a TEE.
type ClientCertificate struct {
	C      *x509.Certificate
	TEEKey securehw.Key

	teeDevice securehw.TEE
}

func (c *ClientCertificate) Close() {
	c.teeDevice.Close()
}

// CreateOrLoadClientCertificate creates a private key using a TEE and generates a new client
// certificate using SCEP.
//
// If there's already a key and certificate in the metadata directory it will return them.
func CreateOrLoadClientCertificate(
	ctx context.Context,
	metadataDir string,
	scepURL string,
	scepChallenge string,
	commonName string,
	rootCA string,
	insecure bool,
	logger zerolog.Logger,
) (*ClientCertificate, error) {
	teeDevice, err := securehw.New(metadataDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TEE device: %w", err)
	}
	teeKey, err := teeDevice.LoadKey()
	switch {
	case err == nil:
		// OK
	case errors.As(err, &securehw.ErrKeyNotFound{}):
		// Key doesn't exist yet, let's create it.
		teeKey, err = teeDevice.CreateKey()
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
			scep.WithSigningKey(teeKey),
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
	return &ClientCertificate{
		C:      clientCert,
		TEEKey: teeKey,

		teeDevice: teeDevice,
	}, nil
}

func loadSCEPClientCert(rootDir string) (*x509.Certificate, error) {
	certPath := filepath.Join(rootDir, constant.FleetHTTPSignatureCertificateFileName)
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

func saveSCEPClientCert(rootDir string, cert *x509.Certificate) error {
	certPath := filepath.Join(rootDir, constant.FleetHTTPSignatureCertificateFileName)
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
