package hostidentity

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/scep"
	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/securehw"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/rs/zerolog"
)

const (
	// certificateRenewalThreshold is the time before certificate expiration
	// when renewal should be initiated (180 days)
	certificateRenewalThreshold = 180 * 24 * time.Hour
)

// Credentials holds a certificate and its corresponding private key handle stored in secure hardware.
type Credentials struct {
	// Certificate holds the public certificate issued via SCEP.
	Certificate *x509.Certificate
	// SecureHWKey holds the private key protected by secure hardware.
	SecureHWKey securehw.Key

	// CertificatePath is the file path to the public certificate issued via SCEP.
	CertificatePath string

	SecureHW securehw.SecureHW
}

// Close releases key resources.
func (c *Credentials) Close() {
	c.SecureHW.Close()
}

// Setup creates a private key using a SecureHW and generates a new client
// certificate using SCEP.
// If there's already a key and certificate in the metadata directory it will return them.
// The returned Credentials needs to be closed after its use.
// The restartFunc will be called to trigger an Orbit restart for certificate renewal
// if the certificate is close to expiration.
func Setup(
	ctx context.Context,
	metadataDir string,
	scepURL string,
	scepChallenge string,
	commonName string,
	rootCA string,
	insecure bool,
	logger zerolog.Logger,
	restartFunc func(reason string),
) (*Credentials, error) {
	secureHWDevice, err := securehw.New(metadataDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize secure hardware device: %w", err)
	}
	credentials := &Credentials{
		CertificatePath: filepath.Join(metadataDir, constant.FleetHTTPSignatureCertificateFileName),
		SecureHW:        secureHWDevice,
	}
	credentials.SecureHWKey, err = secureHWDevice.LoadKey()
	switch {
	case err == nil:
		// OK
	case errors.As(err, &securehw.ErrKeyNotFound{}):
		// Key doesn't exist yet, let's create it.

		// First let's clear any existing certificate in
		// case a user or process deleted the keyfile but not
		// the issued-via-SCEP certificate.
		certPath := filepath.Join(metadataDir, constant.FleetHTTPSignatureCertificateFileName)
		if err := os.RemoveAll(certPath); err != nil {
			return nil, fmt.Errorf("failed to clear the host identity certificate: %w", err)
		}

		credentials.SecureHWKey, err = secureHWDevice.CreateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to create secure hardware key: %w", err)
		}
	default:
		return nil, fmt.Errorf("failed to load secure hardware key: %w", err)
	}

	clientCert, err := loadSCEPClientCert(metadataDir)
	switch {
	case err == nil && certNeedsRenewal(clientCert, certificateRenewalThreshold):
		logger.Info().Msg("Certificate expires within 180 days, initiating renewal")

		// Perform certificate renewal
		credentials.Certificate = clientCert
		renewedCert, err := RenewCertificate(ctx, metadataDir, credentials, scepURL, rootCA, insecure, logger)
		if err != nil {
			logger.Error().Err(err).Msg("Certificate renewal failed, continuing with existing certificate")
		} else {
			clientCert = renewedCert
			logger.Info().Msg("Certificate renewal completed successfully")
		}
	case errors.Is(err, os.ErrNotExist):
		// We don't have a certificate, let's issue one using SCEP.
		opts := []scep.Option{
			scep.WithRootCA(rootCA),
			scep.WithSigningKey(credentials.SecureHWKey),
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
	case err != nil:
		return nil, fmt.Errorf("failed to load host identity certificate: %w", err)
	}
	credentials.Certificate = clientCert

	// Sanity check in case the public key material on the secure HW
	// does not match the certificate public key.
	// This can happen if something or someone deletes the private and public blobs
	// and they are re-generated at startup.

	secureHWPubKey, err := credentials.SecureHWKey.Public()
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

	// Start a goroutine with a timer to trigger restart for certificate renewal
	if restartFunc != nil {
		go func() {
			// Calculate time until certificate expires
			timeUntilExpiry := time.Until(clientCert.NotAfter)

			// Set timer for 180 days before expiry (plus 1 minute buffer)
			// or 1 hour, whichever is longer
			renewalTime := timeUntilExpiry - certificateRenewalThreshold + 1*time.Minute
			if renewalTime < 1*time.Hour {
				renewalTime = 1 * time.Hour
			}

			logger.Info().
				Dur("renewal_in", renewalTime).
				Time("cert_expires", clientCert.NotAfter).
				Msg("Scheduling host identity certificate renewal timer")

			timer := time.NewTimer(renewalTime)
			<-timer.C

			logger.Info().Msg("Certificate renewal timer triggered")
			restartFunc("host identity certificate renewal")
		}()
	}

	return credentials, nil
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

// certNeedsRenewal checks if the certificate expires within the given duration
func certNeedsRenewal(cert *x509.Certificate, renewalThreshold time.Duration) bool {
	return time.Until(cert.NotAfter) < renewalThreshold
}

// RenewCertificate performs certificate renewal with proof-of-possession
func RenewCertificate(
	ctx context.Context,
	metadataDir string,
	credentials *Credentials,
	scepURL string,
	rootCA string,
	insecure bool,
	logger zerolog.Logger,
) (*x509.Certificate, error) {
	// First, backup the existing key file
	keyPath := filepath.Join(metadataDir, constant.FleetHTTPSignatureTPMKeyFileName)
	oldKeyPath := filepath.Join(metadataDir, constant.FleetHTTPSignatureTPMKeyBackupFileName)

	// Clean up any existing old key file
	if err := os.RemoveAll(oldKeyPath); err != nil {
		return nil, fmt.Errorf("failed to clean up existing old key: %w", err)
	}

	// Backup the current key
	if err := os.Rename(keyPath, oldKeyPath); err != nil {
		return nil, fmt.Errorf("failed to backup existing key: %w", err)
	}

	// Ensure we restore the backup if something goes wrong
	defer func() {
		if _, err := os.Stat(oldKeyPath); err == nil {
			// If we still have the old key and no new key exists, restore it
			if _, err := os.Stat(keyPath); err != nil {
				_ = os.Rename(oldKeyPath, keyPath)
			}
		}
	}()

	// Create new key (this will create it at the standard path)
	newKey, err := credentials.SecureHW.CreateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to create renewal key: %w", err)
	}

	// Get the old key's signer for proof-of-possession
	oldSigner, err := credentials.SecureHWKey.Signer()
	if err != nil {
		return nil, fmt.Errorf("failed to get signer from old key: %w", err)
	}

	// Create renewal data with proof-of-possession
	serialHex := fmt.Sprintf("0x%x", credentials.Certificate.SerialNumber.Bytes())
	hash := sha256.Sum256([]byte(serialHex))
	signature, err := oldSigner.Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("failed to sign renewal data: %w", err)
	}

	renewalData := types.RenewalData{
		SerialNumber: serialHex,
		Signature:    base64.StdEncoding.EncodeToString(signature),
	}

	renewalDataJSON, err := json.Marshal(renewalData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal renewal data: %w", err)
	}

	// Create SCEP client with custom CSR that includes the renewal extension
	renewedCert, err := fetchCertWithRenewal(ctx, newKey, scepURL, credentials.Certificate.Subject.CommonName, rootCA, insecure, renewalDataJSON, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch renewed certificate: %w", err)
	}

	// Save the renewed certificate
	if err := saveSCEPClientCert(metadataDir, renewedCert); err != nil {
		return nil, fmt.Errorf("failed to save renewed certificate: %w", err)
	}

	// Remove the old key backup now that renewal was successful
	if err := os.Remove(oldKeyPath); err != nil {
		return nil, fmt.Errorf("failed to remove old key backup: %w", err)
	}

	// Close the old TPM key since it will no longer be used.
	_ = credentials.SecureHWKey.Close()

	credentials.SecureHWKey = newKey

	return renewedCert, nil
}

// fetchCertWithRenewal performs SCEP certificate fetch with renewal extension
func fetchCertWithRenewal(
	ctx context.Context,
	signingKey securehw.Key,
	scepURL string,
	commonName string,
	rootCA string,
	insecure bool,
	renewalDataJSON []byte,
	logger zerolog.Logger,
) (*x509.Certificate, error) {
	// Create the renewal extension
	renewalExtension := pkix.Extension{
		Id:    types.RenewalExtensionOID,
		Value: renewalDataJSON,
	}

	// Create SCEP client with the renewal extension
	opts := []scep.Option{
		scep.WithRootCA(rootCA),
		scep.WithSigningKey(signingKey),
		scep.WithLogger(logger),
		scep.WithURL(scepURL),
		scep.WithCommonName(commonName),
		scep.WithExtraExtensions([]pkix.Extension{renewalExtension}),
	}
	if insecure {
		opts = append(opts, scep.Insecure())
	}

	scepClient, err := scep.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SCEP client: %w", err)
	}

	// Fetch the certificate with the renewal extension in the CSR
	return scepClient.FetchCert(ctx)
}
