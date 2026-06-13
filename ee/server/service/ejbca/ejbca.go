// Package ejbca implements Fleet's REST client for the EJBCA Certificate
// Authority. It speaks the EJBCA REST API over mutual TLS, using a client
// certificate the customer's EJBCA admin enrolls and binds to an
// administrator role with appropriate access rules.
//
// REST reference: https://docs.keyfactor.com/ejbca/latest/ejbca-rest-interface
package ejbca

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-json-experiment/json"
	"software.sslmate.com/src/go-pkcs12"
)

const (
	defaultTimeout = 20 * time.Second
	restAPIPrefix  = "/ejbca/ejbca-rest-api/v1"
)

var (
	// Microsoft User Principal Name OID for the SubjectAltName otherName extension.
	oidMicrosoftUPN = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 20, 2, 3}
	// X.509 SubjectAltName extension OID.
	oidSubjectAltName = asn1.ObjectIdentifier{2, 5, 29, 17}
)

// Service is the EJBCA REST API client.
type Service struct {
	logger  *slog.Logger
	timeout time.Duration
}

// Compile-time check that Service satisfies fleet.EJBCAService.
var _ fleet.EJBCAService = (*Service)(nil)

// NewService constructs a Service with the given options.
func NewService(opts ...Opt) fleet.EJBCAService {
	s := &Service{}
	s.populateOpts(opts)
	return s
}

// Opt configures the Service.
type Opt func(*Service)

// WithTimeout sets the per-request HTTP timeout.
func WithTimeout(t time.Duration) Opt {
	return func(s *Service) { s.timeout = t }
}

// WithLogger sets the slog.Logger used by the service.
func WithLogger(logger *slog.Logger) Opt {
	return func(s *Service) { s.logger = logger }
}

func (s *Service) populateOpts(opts []Opt) {
	for _, opt := range opts {
		opt(s)
	}
	if s.timeout <= 0 {
		s.timeout = defaultTimeout
	}
	if s.logger == nil {
		s.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
}

// buildClient returns an HTTP client configured for mTLS against the supplied
// EJBCA configuration. The client presents cfg.ClientCertPEM / ClientKeyPEM and
// trusts cfg.TrustCABundlePEM (plus the system root store) for server cert
// verification. When TrustCABundlePEM is empty, the system root store alone is
// used — appropriate only when EJBCA's HTTPS cert is signed by a publicly
// trusted CA, which is uncommon for self-hosted EJBCA deployments.
func (s *Service) buildClient(ctx context.Context, cfg fleet.EJBCACA) (*http.Client, error) {
	cert, err := tls.X509KeyPair([]byte(cfg.ClientCertPEM), []byte(cfg.ClientKeyPEM))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading EJBCA client cert/key pair")
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if cfg.TrustCABundlePEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(cfg.TrustCABundlePEM)) {
			return nil, ctxerr.Errorf(ctx, "EJBCA trust_ca_bundle did not contain any usable certificates")
		}
		tlsCfg.RootCAs = pool
	}
	return fleethttp.NewClient(
		fleethttp.WithTimeout(s.timeout),
		fleethttp.WithTLSClientConfig(tlsCfg),
	), nil
}

// VerifyConnection probes GET /v1/ca/status over mTLS to confirm the supplied
// EJBCA configuration reaches the API and authenticates successfully.
func (s *Service) VerifyConnection(ctx context.Context, cfg fleet.EJBCACA) error {
	client, err := s.buildClient(ctx, cfg)
	if err != nil {
		return err
	}

	base := strings.TrimRight(cfg.URL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+restAPIPrefix+"/ca/status", nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating EJBCA status request")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sending EJBCA status request")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through to body decode
	case http.StatusUnauthorized, http.StatusForbidden:
		return ctxerr.Errorf(ctx,
			"EJBCA rejected the Fleet client certificate (likely revoked, expired, or not bound to a role with sufficient access); status code: %d",
			resp.StatusCode,
		)
	default:
		return ctxerr.Errorf(ctx, "unexpected EJBCA status code on /ca/status: %d", resp.StatusCode)
	}

	var statusResp struct {
		Status   string `json:"status"`
		Version  string `json:"version"`
		Revision string `json:"revision"`
	}
	if err := json.UnmarshalRead(resp.Body, &statusResp); err != nil {
		return ctxerr.Wrap(ctx, err, "decoding EJBCA status response")
	}
	if statusResp.Status != "OK" {
		return ctxerr.Errorf(ctx, "EJBCA reports non-OK status: %q", statusResp.Status)
	}
	s.logger.DebugContext(ctx, "EJBCA connection verified",
		"version", statusResp.Version,
		"revision", statusResp.Revision,
	)
	return nil
}

// GetCertificate enrolls a certificate against the supplied EJBCA configuration.
// Fleet generates the RSA keypair locally, sends the CSR, and wraps the issued
// X.509 cert and key in a PKCS#12 bundle for delivery via MDM profile variable
// substitution. The caller is responsible for expanding any Fleet variables in
// cfg fields before invoking.
func (s *Service) GetCertificate(ctx context.Context, cfg fleet.EJBCACA) (*fleet.EJBCACertificate, error) {
	client, err := s.buildClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating RSA private key for EJBCA enrollment")
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: cfg.UsernameTemplate},
	}
	if len(cfg.CertificateUserPrincipalNames) > 0 {
		ext, err := buildUPNSANExtension(cfg.CertificateUserPrincipalNames)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "building UPN subjectAltName extension")
		}
		csrTemplate.ExtraExtensions = []pkix.Extension{ext}
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating CSR for EJBCA enrollment")
	}
	csrPEM := strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})))

	// EJBCA's backend rejects a null `password` for any CA with
	// useUserStorage=true (essentially every production deployment, verified
	// in SignSessionBean.java). Under the standard auto-create-EE +
	// permissive-password configuration we ship against, the supplied value
	// isn't authenticating anything; the mTLS client cert + role scope is the
	// actual security boundary (see research.md). We satisfy the requirement
	// with a per-call random value and discard it — no persistent shared
	// secret on Fleet's side.
	pwBytes := make([]byte, 32)
	if _, err := rand.Read(pwBytes); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating per-enrollment EJBCA password")
	}
	enrollmentPassword := hex.EncodeToString(pwBytes)

	reqBody := map[string]any{
		"certificate_request":        csrPEM,
		"certificate_profile_name":   cfg.CertificateProfileName,
		"end_entity_profile_name":    cfg.EndEntityProfileName,
		"certificate_authority_name": cfg.CertificateAuthorityNameEJBCA,
		"username":                   cfg.UsernameTemplate,
		"password":                   enrollmentPassword,
		"include_chain":              false,
		"response_format":            "DER",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshaling EJBCA pkcs10enroll body")
	}

	base := strings.TrimRight(cfg.URL, "/")
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		base+restAPIPrefix+"/certificate/pkcs10enroll", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating EJBCA pkcs10enroll request")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sending EJBCA pkcs10enroll request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, mapEnrollError(ctx, resp)
	}

	var certResp struct {
		Certificate    string `json:"certificate"`
		SerialNumber   string `json:"serial_number"`
		ResponseFormat string `json:"response_format"`
	}
	if err := json.UnmarshalRead(resp.Body, &certResp); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding EJBCA pkcs10enroll response")
	}
	if certResp.Certificate == "" {
		return nil, ctxerr.Errorf(ctx, "EJBCA returned an empty certificate")
	}

	derBytes, err := base64.StdEncoding.DecodeString(certResp.Certificate)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "base64-decoding EJBCA certificate")
	}
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing EJBCA certificate DER")
	}

	pfxPassword, err := server.GenerateRandomText(10)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating PKCS#12 wrapper password")
	}
	pfxData, err := pkcs12.Legacy.Encode(privateKey, cert, nil, pfxPassword)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "encoding EJBCA certificate as PKCS#12")
	}

	s.logger.DebugContext(ctx, "EJBCA certificate issued",
		"serial_number", cert.SerialNumber.String(),
		"not_after", cert.NotAfter,
	)

	return &fleet.EJBCACertificate{
		PfxData:        pfxData,
		Password:       pfxPassword,
		NotValidBefore: cert.NotBefore,
		NotValidAfter:  cert.NotAfter,
		SerialNumber:   cert.SerialNumber.String(),
	}, nil
}

// mapEnrollError translates a non-2xx pkcs10enroll response into a wrapped
// error with a user-actionable message. EJBCA returns
// {"error_code": N, "error_message": "..."} on most failures.
func mapEnrollError(ctx context.Context, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	_ = json.Unmarshal(body, &errResp)
	msg := errResp.ErrorMessage
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ctxerr.Errorf(ctx,
			"EJBCA rejected the Fleet client certificate (likely revoked, expired, or not bound to a role with sufficient access); status %d: %s",
			resp.StatusCode, msg)
	case http.StatusNotFound:
		return ctxerr.Errorf(ctx,
			"EJBCA reports the CA or profile name does not exist; status %d: %s",
			resp.StatusCode, msg)
	case http.StatusUnprocessableEntity:
		return ctxerr.Errorf(ctx,
			"EJBCA end-entity profile rejected the CSR; status %d: %s",
			resp.StatusCode, msg)
	default:
		return ctxerr.Errorf(ctx,
			"unexpected EJBCA pkcs10enroll status %d: %s", resp.StatusCode, msg)
	}
}

// buildUPNSANExtension constructs an X.509 subjectAltName extension that
// carries one or more Microsoft User Principal Names as otherName entries.
//
// ASN.1 (RFC 5280 + Microsoft UPN OID 1.3.6.1.4.1.311.20.2.3):
//
//	SubjectAltName ::= SEQUENCE OF GeneralName
//	GeneralName    ::= CHOICE { otherName [0] OtherName, ... }
//	OtherName      ::= SEQUENCE {
//	                     type-id OBJECT IDENTIFIER,
//	                     value   [0] EXPLICIT ANY DEFINED BY type-id
//	                   }
//
// Go's encoding/asn1 marshals the OtherName struct with the universal SEQUENCE
// tag (0x30); under GeneralName.otherName [0] CHOICE the outermost tag must be
// the context-specific constructed [0] (0xA0), so we rewrite the leading byte
// after marshaling.
//
// EJBCA preserves SAN exactly as supplied in the CSR only when the customer's
// Certificate Profile has "Allow Extension Override" enabled — documented in
// the dev guide alongside this code.
func buildUPNSANExtension(upns []string) (pkix.Extension, error) {
	type upnValue struct {
		Value string `asn1:"utf8"`
	}
	type otherName struct {
		TypeID asn1.ObjectIdentifier
		Value  upnValue `asn1:"explicit,tag:0"`
	}

	var generalNames []byte
	for _, upn := range upns {
		on := otherName{
			TypeID: oidMicrosoftUPN,
			Value:  upnValue{Value: upn},
		}
		b, err := asn1.Marshal(on)
		if err != nil {
			return pkix.Extension{}, err
		}
		if len(b) < 1 || b[0] != 0x30 {
			return pkix.Extension{}, errors.New("unexpected ASN.1 prefix for otherName SEQUENCE")
		}
		// Rewrite the universal SEQUENCE tag to context-specific [0] constructed.
		b[0] = 0xA0
		generalNames = append(generalNames, b...)
	}

	sanBytes, err := asn1.Marshal(asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagSequence,
		IsCompound: true,
		Bytes:      generalNames,
	})
	if err != nil {
		return pkix.Extension{}, err
	}

	return pkix.Extension{
		Id:    oidSubjectAltName,
		Value: sanBytes,
	}, nil
}
