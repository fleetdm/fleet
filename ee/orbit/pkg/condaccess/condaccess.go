// Package condaccess handles SCEP enrollment for Fleet's Okta conditional access feature on Linux.
// It generates an ECC key, fetches a client certificate from Fleet's SCEP endpoint, and persists
// both to disk so the certificate can be presented during mTLS at Fleet's SAML SSO endpoint.
package condaccess

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	orbitscep "github.com/fleetdm/fleet/v4/ee/orbit/pkg/scep"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/rs/zerolog"
)

const (
	// commonName is the CN used in the SCEP certificate request, matching the macOS MDM profile.
	commonName = "Fleet conditional access for Okta"
	// sanURIPrefix is the SAN URI prefix used to identify the device UUID in the certificate.
	sanURIPrefix = "urn:device:fleet:uuid:"
	// renewalThreshold is how far from expiry we trigger re-enrollment.
	renewalThreshold = 30 * 24 * time.Hour
)

// scepSigningKey wraps *ecdsa.PrivateKey to satisfy orbitscep.SigningKey.
type scepSigningKey struct {
	key *ecdsa.PrivateKey
}

// Signer returns the underlying ECDSA key as a crypto.Signer.
func (s *scepSigningKey) Signer() (crypto.Signer, error) {
	return s.key, nil
}

// Enroll ensures a valid conditional access certificate exists in metadataDir.
// If no certificate exists or the existing one is expiring within 30 days, it
// generates a new ECC P-256 key, enrolls via SCEP, and saves the cert and key.
// Returns the certificate (existing or newly issued).
func Enroll(
	ctx context.Context,
	metadataDir string,
	scepURL string,
	scepChallenge string,
	hardwareUUID string,
	rootCA string,
	insecure bool,
	logger zerolog.Logger,
) (*x509.Certificate, error) {
	cert, err := loadCert(metadataDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load existing cert: %w", err)
	}

	if cert != nil && !certNeedsRenewal(cert, renewalThreshold) {
		logger.Debug().Str("serial", cert.SerialNumber.String()).Msg("conditional access cert valid, skipping enrollment")
		return cert, nil
	}

	logger.Info().Msg("enrolling conditional access SCEP certificate")

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ECC key: %w", err)
	}

	if err := saveKey(metadataDir, privKey); err != nil {
		return nil, fmt.Errorf("save private key: %w", err)
	}

	sanURI, err := url.Parse(sanURIPrefix + hardwareUUID)
	if err != nil {
		return nil, fmt.Errorf("parse SAN URI: %w", err)
	}

	opts := []orbitscep.Option{
		orbitscep.WithSigningKey(&scepSigningKey{key: privKey}),
		orbitscep.WithURL(scepURL),
		orbitscep.WithChallenge(scepChallenge),
		orbitscep.WithCommonName(commonName),
		orbitscep.WithURIs([]*url.URL{sanURI}),
		orbitscep.WithRootCA(rootCA),
		orbitscep.WithLogger(logger),
	}
	if insecure {
		opts = append(opts, orbitscep.Insecure())
	}

	scepClient, err := orbitscep.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create SCEP client: %w", err)
	}

	newCert, err := scepClient.FetchCert(ctx)
	if err != nil {
		return nil, fmt.Errorf("SCEP enrollment: %w", err)
	}

	if err := saveCert(metadataDir, newCert); err != nil {
		return nil, fmt.Errorf("save cert: %w", err)
	}

	logger.Info().Str("serial", newCert.SerialNumber.String()).Msg("conditional access SCEP enrollment complete")
	return newCert, nil
}

// loadCert reads the conditional access certificate from metadataDir.
// Returns os.ErrNotExist if the file doesn't exist.
func loadCert(metadataDir string) (*x509.Certificate, error) {
	certPath := filepath.Join(metadataDir, constant.ConditionalAccessCertFileName)
	pemBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("failed to decode PEM certificate block")
	}
	return x509.ParseCertificate(block.Bytes)
}

// saveCert writes the certificate as PEM to metadataDir (world-readable).
func saveCert(metadataDir string, cert *x509.Certificate) error {
	certPath := filepath.Join(metadataDir, constant.ConditionalAccessCertFileName)
	f, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, constant.DefaultWorldReadableFileMode)
	if err != nil {
		return fmt.Errorf("open cert file: %w", err)
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

// saveKey encodes privKey as PKCS#8 PEM and writes it to metadataDir with mode 0600.
func saveKey(metadataDir string, privKey *ecdsa.PrivateKey) error {
	keyPath := filepath.Join(metadataDir, constant.ConditionalAccessKeyFileName)
	derBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	f, err := os.OpenFile(keyPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, constant.DefaultFileMode)
	if err != nil {
		return fmt.Errorf("open key file: %w", err)
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: derBytes})
}

// certNeedsRenewal returns true if the certificate expires within threshold.
func certNeedsRenewal(cert *x509.Certificate, threshold time.Duration) bool {
	return time.Until(cert.NotAfter) < threshold
}
