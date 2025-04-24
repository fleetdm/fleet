package digicert

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-json-experiment/json"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"software.sslmate.com/src/go-pkcs12"
)

// REST client for https://one.digicert.com/mpki/docs/swagger-ui/index.html

// defaultTimeout is the timeout for requests.
const defaultTimeout = 20 * time.Second

const (
	errMessageInvalidAPIToken = "The API token configured in %s certificate authority is invalid. " + // nolint:gosec // ignore G101
		"Status code for POST request: %d"
	errMessageInvalidProfile = "The \"profile_id\" configured in %s certificate authority doesn't exist. Status code for POST request: %d"
)

type Service struct {
	logger  kitlog.Logger
	timeout time.Duration
}

// Compile-time check for DigiCertService interface
var _ fleet.DigiCertService = (*Service)(nil)

func NewService(opts ...Opt) fleet.DigiCertService {
	s := &Service{}
	s.populateOpts(opts)
	return s
}

// Opt is the type for DigiCert integration options.
type Opt func(*Service)

// WithTimeout sets the timeout to use for the HTTP client.
func WithTimeout(t time.Duration) Opt {
	return func(s *Service) {
		s.timeout = t
	}
}

// WithLogger sets the logger to use for the service.
func WithLogger(logger kitlog.Logger) Opt {
	return func(s *Service) {
		s.logger = logger
	}
}

func (s *Service) VerifyProfileID(ctx context.Context, config fleet.DigiCertIntegration) error {
	client := fleethttp.NewClient(fleethttp.WithTimeout(s.timeout))

	config.URL = strings.TrimRight(config.URL, "/")
	req, err := http.NewRequest("GET", config.URL+"/mpki/api/v2/profile/"+url.PathEscape(config.ProfileID), nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating DigiCert request")
	}
	req.Header.Set("X-API-key", config.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sending DigiCert request")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// all good
	case http.StatusUnauthorized:
		return ctxerr.Errorf(ctx, "most likely invalid API token; status code: %d", resp.StatusCode)
	case http.StatusForbidden:
		return ctxerr.Errorf(ctx, "most likely invalid profile GUID; status code: %d", resp.StatusCode)
	default:
		return ctxerr.Errorf(ctx, "unexpected DigiCert status code: %d", resp.StatusCode)
	}

	type profile struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	var p profile
	err = json.UnmarshalRead(resp.Body, &p)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshaling DigiCert response")
	}
	if p.Status != "Active" {
		return ctxerr.Errorf(ctx, "DigiCert profile status is not Active: %s", p.Status)
	}
	level.Debug(s.logger).Log("msg", "DigiCert profile verified", "id", p.ID, "name", p.Name, "status", p.Status)
	return nil
}

func (s *Service) populateOpts(opts []Opt) {
	for _, opt := range opts {
		opt(s)
	}
	if s.timeout <= 0 {
		s.timeout = defaultTimeout
	}
	if s.logger == nil {
		s.logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	}
}

func (s *Service) GetCertificate(ctx context.Context, config fleet.DigiCertIntegration) (*fleet.DigiCertCertificate, error) {
	client := fleethttp.NewClient(fleethttp.WithTimeout(s.timeout))

	// Generate a CSR (Certificate Signing Request).
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating RSA private key")
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: config.CertificateCommonName,
		},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating CSR")
	}

	csr := strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})))

	reqBody := map[string]interface{}{
		"profile": map[string]string{
			"id": config.ProfileID,
		},
		"seat": map[string]string{
			"seat_id": config.CertificateSeatID,
		},
		"delivery_format": "x509",
		"attributes": map[string]interface{}{
			"subject": map[string]string{
				"common_name": config.CertificateCommonName,
			},
		},
		"csr": csr,
	}
	// UPN (User Principal Names) is only supported by User seat type (2025/03/10)
	// https://docs.digicert.com/fr/trust-lifecycle-manager/inventory/certificate-attributes-and-extensions/subject-alternative-name--san--attributes.html
	// Check that UPNs are present and not empty (we only support 1 as of 2025/03/27)
	if len(config.CertificateUserPrincipalNames) > 0 && len(strings.TrimSpace(config.CertificateUserPrincipalNames[0])) > 0 {
		attributes, ok := reqBody["attributes"].(map[string]interface{})
		if !ok {
			return nil, ctxerr.Errorf(ctx, "unexpected DigiCert attributes type: %T", reqBody["attributes"])
		}
		attributes["extensions"] = map[string]interface{}{
			"san": map[string]interface{}{
				"user_principal_names": config.CertificateUserPrincipalNames,
			},
		}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshaling request body")
	}

	config.URL = strings.TrimRight(config.URL, "/")
	req, err := http.NewRequest("POST", config.URL+"/mpki/api/v1/certificate", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating DigiCert POST request")
	}

	req.Header.Set("X-API-key", config.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sending DigiCert POST request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Try to see if errors are present in body
		type errorResponse struct {
			Errors []struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"errors"`
		}
		var errResp errorResponse
		err = json.UnmarshalRead(resp.Body, &errResp)
		if err != nil || len(errResp.Errors) == 0 {
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				return nil, ctxerr.Errorf(ctx, errMessageInvalidAPIToken, config.Name, resp.StatusCode)
			case http.StatusForbidden:
				return nil, ctxerr.Errorf(ctx, errMessageInvalidProfile, config.Name, resp.StatusCode)
			}
			return nil, ctxerr.Errorf(ctx, "unexpected DigiCert status code for POST request: %d", resp.StatusCode)
		}

		combinedErrorMessages := make([]string, len(errResp.Errors))
		for i, e := range errResp.Errors {
			combinedErrorMessages[i] = e.Message
		}
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, ctxerr.Errorf(ctx, errMessageInvalidAPIToken+", errors: %s", config.Name, resp.StatusCode,
				strings.Join(combinedErrorMessages, "; "))
		case http.StatusForbidden:
			return nil, ctxerr.Errorf(ctx, errMessageInvalidProfile+", errors: %s", config.Name, resp.StatusCode,
				strings.Join(combinedErrorMessages, "; "))
		}
		return nil, ctxerr.Errorf(ctx, "unexpected DigiCert status code for POST request: %d, errors: %s", resp.StatusCode,
			strings.Join(combinedErrorMessages, "; "))
	}

	type certificateResponse struct {
		SerialNumber   string `json:"serial_number"`
		DeliveryFormat string `json:"delivery_format"`
		Certificate    string `json:"certificate"`
	}

	var certResp certificateResponse
	err = json.UnmarshalRead(resp.Body, &certResp)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshaling DigiCert POST response")
	}

	if certResp.DeliveryFormat != "x509" {
		return nil, ctxerr.Errorf(ctx, "unexpected DigiCert delivery format: %s", certResp.DeliveryFormat)
	}

	// Serial number is an up to 20-byte(40 char) hex string
	_, err = hex.DecodeString(certResp.SerialNumber)
	if err != nil || certResp.SerialNumber == "" || len(certResp.SerialNumber) > 40 {
		level.Error(s.logger).Log("msg", "DigiCert certificate returned with invalid serial number", "serial_number", certResp.SerialNumber, "decode_err", err)
		return nil, ctxerr.Errorf(ctx, "invalid DigiCert serial number: %s", certResp.SerialNumber)
	}

	if len(certResp.Certificate) == 0 {
		return nil, ctxerr.Errorf(ctx, "did not receive DigiCert certificate")
	}

	level.Debug(s.logger).Log("msg", "DigiCert certificate created", "serial_number", certResp.SerialNumber)

	// Decode the certificate from PEM format
	certBlock, _ := pem.Decode([]byte(certResp.Certificate))
	if certBlock == nil {
		return nil, ctxerr.Errorf(ctx, "failed to decode certificate PEM block")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing certificate from PEM")
	}

	// Encode the private key and certificate into PKCS12
	password, err := server.GenerateRandomText(10)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating password for PKCS12 bundle")
	}
	pkcs12Data, err := pkcs12.Legacy.Encode(privateKey, cert, nil, password)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating PKCS12 bundle")
	}

	return &fleet.DigiCertCertificate{
		PfxData:        pkcs12Data,
		Password:       password,
		NotValidBefore: cert.NotBefore,
		NotValidAfter:  cert.NotAfter,
		SerialNumber:   certResp.SerialNumber,
	}, nil
}
