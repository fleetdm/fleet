package apple_mdm

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/micromdm/nanodep/tokenpki"
)

const (
	defaultFleetDMAPIURL     = "https://fleetdm.com"
	getSignedAPNSCSRPath     = "/api/v1/deliver-apple-csr"
	depCertificateCommonName = "FleetDM"
	depCertificateExpiryDays = 30
)

// emailAddressOID defined by https://oidref.com/1.2.840.113549.1.9.1
var emailAddressOID = []int{1, 2, 840, 113549, 1, 9, 1}

// GenerateAPNSCSRKey generates a APNS CSR (certificate signing request) and
// returns the CSR and private key.
func GenerateAPNSCSRKey(email, org string) (*x509.CertificateRequest, *rsa.PrivateKey, error) {
	key, err := newPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("generate private key: %w", err)
	}

	subj := pkix.Name{
		Organization: []string{org},
		ExtraNames: []pkix.AttributeTypeAndValue{{
			Type:  emailAddressOID,
			Value: email,
		}},
	}
	template := &x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	b, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, nil, err
	}

	certReq, err := x509.ParseCertificateRequest(b)
	if err != nil {
		return nil, nil, err
	}

	return certReq, key, nil
}

type FleetWebsiteError struct {
	Status  int
	message string
}

func (e FleetWebsiteError) Error() string {
	if e.message != "" {
		return e.message
	}

	return "Unknown Error"
}

type getSignedAPNSCSRRequest struct {
	UnsignedCSRData []byte `json:"unsignedCsrData"`
}

// GetSignedAPNSCSR makes a request to the fleetdm.com API to get a signed APNs
// CSR that is sent to the email provided in the certificate subject.
func GetSignedAPNSCSR(client *http.Client, csr *x509.CertificateRequest) error {
	csrPEM := EncodeCertRequestPEM(csr)

	payload := getSignedAPNSCSRRequest{
		UnsignedCSRData: csrPEM,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// for testing
	baseURL := defaultFleetDMAPIURL
	if x := os.Getenv("TEST_FLEETDM_API_URL"); x != "" {
		baseURL = strings.TrimRight(x, "/")
	}
	u := baseURL + getSignedAPNSCSRPath

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return FleetWebsiteError{Status: resp.StatusCode, message: string(b)}
	}
	return nil
}

// NewSCEPCACertKey creates a self-signed CA certificate for use with SCEP and
// returns the certificate and its private key.
func NewSCEPCACertKey() (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := newPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	caCert := depot.NewCACert(
		depot.WithYears(10),
		depot.WithCommonName("FleetDM"),
	)

	crtBytes, err := caCert.SelfSign(rand.Reader, key.Public(), key)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// NEWDEPKeyPairPEM generates a new public key certificate and private key for downloading the Apple DEP token.
// The public key is returned as a PEM encoded certificate.
func NewDEPKeyPairPEM() ([]byte, []byte, error) {
	// Note, Apple doesn't check the expiry
	key, cert, err := tokenpki.SelfSignedRSAKeypair(depCertificateCommonName, depCertificateExpiryDays)
	if err != nil {
		return nil, nil, fmt.Errorf("generate encryption keypair: %w", err)
	}

	publicKeyPEM := tokenpki.PEMCertificate(cert.Raw)
	privateKeyPEM := tokenpki.PEMRSAPrivateKey(key)

	return publicKeyPEM, privateKeyPEM, nil
}
