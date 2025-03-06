package digicert

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-json-experiment/json"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// REST client for https://one.digicert.com/mpki/docs/swagger-ui/index.html

// defaultTimeout is the timeout for requests.
const defaultTimeout = 20 * time.Second

type integrationOpts struct {
	timeout time.Duration
}

// Opt is the type for DigiCert integration options.
type Opt func(o *integrationOpts)

// WithTimeout sets the timeout to use for the HTTP client.
func WithTimeout(t time.Duration) Opt {
	return func(o *integrationOpts) {
		o.timeout = t
	}
}

func VerifyProfileID(ctx context.Context, logger kitlog.Logger, config fleet.DigiCertIntegration, opts ...Opt) error {

	client := fleethttp.NewClient(fleethttp.WithTimeout(populateOpts(opts).timeout))

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

	if resp.StatusCode != http.StatusOK {
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
	level.Debug(logger).Log("msg", "DigiCert profile verified", "id", p.ID, "name", p.Name, "status", p.Status)
	return nil
}

func populateOpts(opts []Opt) integrationOpts {
	o := integrationOpts{
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func GetCertificate(ctx context.Context, logger kitlog.Logger, config fleet.DigiCertIntegration, opts ...Opt) error {
	client := fleethttp.NewClient(fleethttp.WithTimeout(populateOpts(opts).timeout))

	// Generate a CSR (Certificate Signing Request) as a string.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generating RSA private key")
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: config.CertificateCommonName,
			// TODO: Add organization from appConfig
		},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privateKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating CSR")
	}

	csr := strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})))
	logger.Log("msg", "CSR generated", "csr", csr)

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

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling request body")
	}

	config.URL = strings.TrimRight(config.URL, "/")
	req, err := http.NewRequest("POST", config.URL+"/mpki/api/v1/certificate", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating DigiCert POST request")
	}

	req.Header.Set("X-API-key", config.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sending DigiCert POST request")
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
			return ctxerr.Errorf(ctx, "unexpected DigiCert status code for POST request: %d", resp.StatusCode)
		}

		combinedErrorMessages := make([]string, len(errResp.Errors))
		for i, e := range errResp.Errors {
			combinedErrorMessages[i] = e.Message
		}
		return ctxerr.Errorf(ctx, "unexpected DigiCert status code for POST request: %d, errors: %s", resp.StatusCode,
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
		return ctxerr.Wrap(ctx, err, "unmarshaling DigiCert POST response")
	}

	level.Debug(logger).Log("msg", "DigiCert certificate created", "serial_number", certResp.SerialNumber)
	fmt.Printf("Certificate:\n%s\n", certResp.Certificate)
	return nil
}
