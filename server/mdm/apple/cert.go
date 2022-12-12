package apple_mdm

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/micromdm/scep/v2/depot"
)

const defaultFleetDMAPIURL = "https://fleetdm.com"

const getSignedAPNSCSRPath = "/api/v1/get_signed_apns_csr"

// emailAddressOID defined by https://oidref.com/1.2.840.113549.1.9.1
var emailAddressOID = []int{1, 2, 840, 113549, 1, 9, 1}

// GenerateAPNSCSRKey generates a APNS csr to be sent to fleetdm.com and returns a csr and key.
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

type getSignedAPNSCSRRequest struct {
	// CSR is the pem encoded certificate request.
	CSR []byte `json:"csr"`
}

// GetSignedAPNSCSR makes a request to the fleetdm.com API to get a signed apns csr that is sent to the email provided in the certificate subject.
func GetSignedAPNSCSR(client *http.Client, csr *x509.CertificateRequest) error {
	csrPEM := EncodeCertRequestPEM(csr)

	payload := getSignedAPNSCSRRequest{
		CSR: csrPEM,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// for testing
	baseURL := defaultFleetDMAPIURL
	if x := os.Getenv("TEST_FLEETDM_API_URL"); x != "" {
		baseURL = x
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
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("api responded with %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// NewSCEPCACertKey creates a self-signed CA certificate for use with SCEP and returns the certificate and its private key.
func NewSCEPCACertKey() (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := newPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	caCert := depot.NewCACert(
		depot.WithYears(10),
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
