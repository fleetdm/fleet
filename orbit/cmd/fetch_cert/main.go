package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/scep"
	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/securehw"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/fleethttpsig"
	"github.com/fleetdm/fleet/v4/server/fleet"
	httpsig "github.com/remitly-oss/httpsig-go"
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

	SecureHW securehw.SecureHW
}

// Close releases key resources.
func (c *Credentials) Close() {
	c.SecureHW.Close()
}

func main() {
	rootDir := flag.String("rootdir", update.DefaultOptions.RootDirectory, "fleetd installation root")
	fleetURL := flag.String("fleeturl", "", "fleet server base URL")
	authorityID := flag.Uint("ca", 0, "certificate authority ID")
	csrPath := flag.String("csr", "", "csr path")
	outPath := flag.String("out", "certificate.pem", "output certificate path")
	debug := flag.Bool("debug", false, "enable debug logging")

	flag.Parse()

	logLevel := zerolog.ErrorLevel
	if *debug {
		logLevel = zerolog.DebugLevel
	}
	logger := zerolog.New(os.Stderr).Level(logLevel)

	if *fleetURL == "" {
		logger.Error().Msg("fleet URL must be set")
		flag.Usage()
		os.Exit(1)
	}

	if *authorityID == 0 {
		logger.Error().Msg("authority ID must be set")
		flag.Usage()
		os.Exit(1)
	}

	if *csrPath == "" {
		logger.Error().Msg("CSR path must be set")
		flag.Usage()
		os.Exit(1)
	}

	csr, err := os.ReadFile(*csrPath)
	if err != nil {
		logger.Err(err).Msg("failed to read CSR")
		os.Exit(1)
	}

	signer, err := getSigner(*rootDir, logger)
	if err != nil {
		logger.Err(err).Msg("failed to get http signer")
		os.Exit(1)
	}

	cert, err := requestCert(signer, *fleetURL, *authorityID, string(csr))
	if err != nil {
		logger.Err(err).Msg("failed to request certificate")
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, []byte(cert), os.FileMode(0644)); err != nil {
		logger.Err(err).Msg("failed to write output certificate")
		os.Exit(1)
	}

	fmt.Printf("Success! Certificate output to %q", *outPath)
}

type requestCertificateResponse struct {
	Certificate string `json:"certificate"`
	Errors      []struct {
		Reason string `json:"reason"`
	} `json:"errors"`
}

func requestCert(signer *httpsig.Signer, fleetURL string, certificateAuthorityID uint, csr string) (string, error) {
	payload := fleet.RequestCertificatePayload{
		ID:  certificateAuthorityID,
		CSR: csr,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode request body: %w", err)
	}

	requestURL := fmt.Sprintf(
		"%s/api/v1/fleet/certificate_authorities/%d/request_certificate",
		strings.TrimRight(fleetURL, "/"),
		certificateAuthorityID,
	)
	reader := bytes.NewReader(bodyBytes)
	req, err := http.NewRequest(http.MethodPost, requestURL, reader)
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}

	if err := signer.Sign(req); err != nil {
		return "", fmt.Errorf("failed to sign request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("making http request: %w", err)
	}
	defer res.Body.Close()

	var certRes requestCertificateResponse
	if err := json.NewDecoder(res.Body).Decode(&certRes); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		var reason string
		if len(certRes.Errors) > 0 {
			reason = ": " + certRes.Errors[0].Reason
		}
		return "", fmt.Errorf("request failed with status code %d%s", res.StatusCode, reason)
	}

	return certRes.Certificate, nil
}

func getSigner(rootDir string, logger zerolog.Logger) (*httpsig.Signer, error) {
	creds, err := loadCredentials(rootDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	cryptoSigner, err := creds.SecureHWKey.HTTPSigner()
	if err != nil {
		return nil, fmt.Errorf("error getting secure HW backed signer: %w", err)
	}

	// Get serial number as hex string
	certSN := strings.ToUpper(creds.Certificate.SerialNumber.Text(16))

	// Get ECC algorithm for signing.
	var signingAlgorithm httpsig.Algorithm
	switch v := cryptoSigner.ECCAlgorithm(); v {
	case securehw.ECCAlgorithmP256:
		signingAlgorithm = httpsig.Algo_ECDSA_P256_SHA256
	case securehw.ECCAlgorithmP384:
		signingAlgorithm = httpsig.Algo_ECDSA_P384_SHA384
	default:
		return nil, fmt.Errorf("invalid ECC algorithm: %v", v)
	}

	httpSigner, err := fleethttpsig.Signer(certSN, cryptoSigner, signingAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP signer: %w", err)
	}

	return httpSigner, nil
}

func loadCredentials(rootDir string, logger zerolog.Logger) (*Credentials, error) {
	secureHWDevice, err := securehw.New(rootDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize secure hardware device: %w", err)
	}

	credentials := &Credentials{
		CertificatePath: filepath.Join(rootDir, constant.FleetHTTPSignatureCertificateFileName),
		SecureHW:        secureHWDevice,
	}

	credentials.SecureHWKey, err = secureHWDevice.LoadKey()
	switch {
	case err == nil:
		// OK
	case errors.As(err, &securehw.ErrKeyNotFound{}):
		return nil, errors.New("a certificate has not yet been issued to this device, try again later")
	default:
		return nil, fmt.Errorf("failed to load secure hardware key: %w", err)
	}

	credentials.Certificate, err = loadSCEPClientCert(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load scep cert: %w", err)
	}

	secureHWPubKey, err := credentials.SecureHWKey.Public()
	if err != nil {
		return nil, fmt.Errorf("error getting public key from secure HW key: %w", err)
	}
	keysEqual, err := scep.PublicKeysEqual(secureHWPubKey, credentials.Certificate.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("error comparing public keys: %w", err)
	}
	if !keysEqual {
		return nil, errors.New("scep public key and tpm public key not equal")
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
