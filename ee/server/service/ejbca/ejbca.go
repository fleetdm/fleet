// Package ejbca is Fleet's client for the EJBCA REST API, authenticated with a TLS client certificate.
// https://docs.keyfactor.com/ejbca/latest/ejbca-rest-interface
package ejbca

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/certissue"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	defaultTimeout = 20 * time.Second
	restAPIPrefix  = "/ejbca/ejbca-rest-api"
	// maxResponseSize bounds what Fleet reads from EJBCA. An enrolled certificate is a few KB.
	maxResponseSize = 1 << 20
	// serviceDisabledText is in the HTML EJBCA answers with, with a 403, when the REST protocol is disabled.
	serviceDisabledText = "This service has been disabled"
	// unknownProfileText is in EJBCA's 400 for a certificate profile name it doesn't have.
	unknownProfileText = "unknown certificate profile"
	// maxSerialLength is the size of the managed certificate serial column.
	maxSerialLength = 40
)

// User-facing messages. They end up in the UI, so they say what to check.
const (
	msgInvalidURL          = "The EJBCA URL must be an https URL without credentials, such as https://ejbca.example.com:8443."
	msgCertExpired         = "The TLS client certificate Fleet uses for EJBCA expired on %s. Upload a new client certificate."
	msgServerCertUntrusted = "Couldn't verify EJBCA's server certificate: %s. Add the CA that issued it to the trust CA bundle."
	msgClientCertRejected  = "EJBCA rejected the TLS client certificate (%s). Check that EJBCA trusts the CA that issued it."
	msgConnectionClosed    = "EJBCA closed the connection without responding. Check that the REST Certificate Management protocol is " +
		"enabled in EJBCA, and restart EJBCA if you just enabled it."
	msgUnreachable      = "Couldn't connect to EJBCA at %s: %s"
	msgProtocolDisabled = "EJBCA's REST Certificate Management protocol is disabled. Enable it under System Configuration > " +
		"Protocol Configuration in EJBCA, and restart EJBCA if needed."
	msgNoTrustCerts  = "The EJBCA trust CA bundle has no usable certificates."
	msgNotAuthorized = "EJBCA didn't authorize the Fleet client certificate: %s. Check that EJBCA trusts the CA that issued it, and that " +
		"it is a member of an EJBCA role with the access rules Fleet needs."
	msgNoCAs           = "The Fleet client certificate can't access any CAs in EJBCA. Add the CA to the access rules of its EJBCA role."
	msgCANotFound      = "EJBCA CA %q was not found. The Fleet client certificate can access: %s."
	msgProfileNotFound = "EJBCA certificate profile %q was not found."
	msgEnrollRejected  = "EJBCA rejected the enrollment request: %s. Check the CA, certificate profile, and end entity profile names."
	msgUnexpected      = "Unexpected response from EJBCA, status %d: %s"
	msgNotFound        = "EJBCA answered not found. Check the EJBCA URL: it should be EJBCA's base URL, such as https://ejbca.example.com:8443, " +
		"without a path such as /ejbca."
	msgSerialTooLong   = "EJBCA issued a certificate with a serial number longer than %d characters: %s"
	msgCommonNameEmpty = "EJBCA issued a certificate without a common name. Check that the end entity profile takes the subject from " +
		"the request or fills in a common name."
	msgUPNMissing = "EJBCA left the user principal name %q out of the certificate. Check that the end entity and certificate profiles " +
		"allow it."
)

// Service is the EJBCA REST client.
type Service struct {
	logger  *slog.Logger
	timeout time.Duration
}

// Compile-time check for the EJBCAService interface
var _ fleet.EJBCAService = (*Service)(nil)

func NewService(opts ...Opt) fleet.EJBCAService {
	s := &Service{}
	s.populateOpts(opts)
	return s
}

// Opt is the type for EJBCA integration options.
type Opt func(*Service)

// WithTimeout sets the timeout to use for the HTTP client.
func WithTimeout(t time.Duration) Opt {
	return func(s *Service) {
		s.timeout = t
	}
}

// WithLogger sets the logger to use for the service.
func WithLogger(logger *slog.Logger) Opt {
	return func(s *Service) {
		s.logger = logger
	}
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

// VerifyConnection checks that EJBCA accepts the client certificate and that the configured CA is one the certificate may use. It also
// checks the certificate profile, but only when EJBCA's optional V2 protocol can look it up. EJBCA can't look up end entity profiles.
func (s *Service) VerifyConnection(ctx context.Context, config fleet.EJBCA) error {
	client, base, err := s.newClient(ctx, config)
	if err != nil {
		return err
	}

	status, body, err := s.do(ctx, client, base, http.MethodGet, "/v1/ca", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return statusError(ctx, status, body, false)
	}
	var cas struct {
		CertificateAuthorities []struct {
			Name string `json:"name"`
		} `json:"certificate_authorities"`
	}
	if err := json.Unmarshal(body, &cas); err != nil {
		return ctxerr.Wrap(ctx, err, "decoding EJBCA CA list")
	}
	names := make([]string, 0, len(cas.CertificateAuthorities))
	for _, ca := range cas.CertificateAuthorities {
		names = append(names, ca.Name)
	}
	if len(names) == 0 {
		return ctxerr.New(ctx, msgNoCAs)
	}
	if !slices.Contains(names, config.CertificateAuthorityName) {
		slices.Sort(names)
		return ctxerr.Errorf(ctx, msgCANotFound, config.CertificateAuthorityName, strings.Join(names, ", "))
	}

	return s.verifyCertificateProfile(ctx, client, base, config.CertificateProfileName)
}

// verifyCertificateProfile only fails for a profile EJBCA says doesn't exist. Any other answer, such as the V2 protocol being disabled,
// means the name can't be checked yet, and enrolling reports it.
func (s *Service) verifyCertificateProfile(ctx context.Context, client *http.Client, base, profileName string) error {
	if profileName == "" {
		return nil
	}
	status, body, err := s.do(ctx, client, base, http.MethodGet, "/v2/certificate/profile/"+url.PathEscape(profileName), nil)
	switch {
	case err != nil && ctx.Err() != nil:
		return err // canceled or out of time, which is not the lookup being unavailable
	case err != nil:
		s.logger.DebugContext(ctx, "EJBCA certificate profile lookup unavailable", "err", err)
	case status == http.StatusBadRequest && strings.Contains(errorMessage(body), unknownProfileText):
		return ctxerr.Errorf(ctx, msgProfileNotFound, profileName)
	case status != http.StatusOK:
		s.logger.DebugContext(ctx, "EJBCA certificate profile lookup unavailable", "status", status)
	}
	return nil
}

// GetCertificate enrolls a certificate with a key generated here and returns both in a PKCS#12 bundle.
func (s *Service) GetCertificate(ctx context.Context, config fleet.EJBCA) (*fleet.EJBCACertificate, error) {
	for field, value := range map[string]string{
		"certificate authority name": config.CertificateAuthorityName,
		"certificate profile name":   config.CertificateProfileName,
		"end entity profile name":    config.EndEntityProfileName,
		"common name":                config.CertificateCommonName,
		"username":                   config.UsernameTemplate,
	} {
		if value == "" {
			return nil, ctxerr.Errorf(ctx, "EJBCA %s is empty", field)
		}
	}
	client, base, err := s.newClient(ctx, config)
	if err != nil {
		return nil, err
	}

	privateKey, csr, err := certissue.NewCSR(ctx, config.CertificateCommonName, config.CertificateUserPrincipalNames)
	if err != nil {
		return nil, err
	}

	// EJBCA requires a password in the request. The client certificate authenticates Fleet, so a random one that is never kept will do.
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating EJBCA enrollment password")
	}
	reqBody, err := json.Marshal(map[string]any{
		"certificate_request":        csr,
		"certificate_profile_name":   config.CertificateProfileName,
		"end_entity_profile_name":    config.EndEntityProfileName,
		"certificate_authority_name": config.CertificateAuthorityName,
		"username":                   config.UsernameTemplate,
		"password":                   hex.EncodeToString(passwordBytes),
		"include_chain":              false,
		"response_format":            "DER",
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshaling EJBCA request body")
	}

	status, body, err := s.do(ctx, client, base, http.MethodPost, "/v1/certificate/pkcs10enroll", reqBody)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, statusError(ctx, status, body, true)
	}

	var certResp struct {
		Certificate    string `json:"certificate"`
		ResponseFormat string `json:"response_format"`
	}
	if err := json.Unmarshal(body, &certResp); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding EJBCA enrollment response")
	}
	if certResp.ResponseFormat != "" && certResp.ResponseFormat != "DER" {
		return nil, ctxerr.Errorf(ctx, "unexpected EJBCA certificate format: %s", certResp.ResponseFormat)
	}
	certDER, err := base64.StdEncoding.DecodeString(certResp.Certificate)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding EJBCA certificate")
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing EJBCA certificate")
	}
	if !privateKey.PublicKey.Equal(cert.PublicKey) {
		return nil, ctxerr.New(ctx, "EJBCA returned a certificate for a different key")
	}
	// An end entity profile can drop the common name or the UPNs, and the host would get a certificate that doesn't identify it. A
	// different common name can be deliberate, for example one built from the username, so it is accepted.
	if cert.Subject.CommonName == "" {
		return nil, ctxerr.New(ctx, msgCommonNameEmpty)
	}
	issuedUPNs := certissue.UPNs(cert)
	for _, upn := range certissue.NonBlank(config.CertificateUserPrincipalNames) {
		if !slices.Contains(issuedUPNs, upn) {
			return nil, ctxerr.Errorf(ctx, msgUPNMissing, upn)
		}
	}

	pfxData, pfxPassword, err := certissue.EncodePKCS12(ctx, privateKey, cert)
	if err != nil {
		return nil, err
	}

	// Lowercase hex padded to 40 characters, the format of Fleet's host certificate list, and it has to fit the serial column.
	serial := fmt.Sprintf("%040s", cert.SerialNumber.Text(16))
	if len(serial) > maxSerialLength {
		return nil, ctxerr.Errorf(ctx, msgSerialTooLong, maxSerialLength, serial)
	}
	s.logger.DebugContext(ctx, "EJBCA certificate issued", "serial_number", serial, "not_after", cert.NotAfter)
	return &fleet.EJBCACertificate{
		PfxData:        pfxData,
		Password:       pfxPassword,
		NotValidBefore: cert.NotBefore,
		NotValidAfter:  cert.NotAfter,
		SerialNumber:   serial,
	}, nil
}

// newClient returns an HTTP client that presents the client certificate, and EJBCA's REST base URL. It fails without contacting EJBCA
// when the client certificate has expired, since EJBCA would only reject the handshake.
func (s *Service) newClient(ctx context.Context, config fleet.EJBCA) (*http.Client, string, error) {
	base, err := url.Parse(strings.TrimRight(config.URL, "/"))
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil {
		return nil, "", ctxerr.New(ctx, msgInvalidURL)
	}
	// EJBCA's REST API takes no query here, and a fragment is never sent.
	base.RawQuery, base.ForceQuery, base.Fragment = "", false, ""

	pair, err := tls.X509KeyPair([]byte(config.MTLSCertPEM), []byte(config.MTLSKeyPEM))
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "loading EJBCA client certificate and key")
	}
	if time.Now().After(pair.Leaf.NotAfter) {
		return nil, "", ctxerr.Errorf(ctx, msgCertExpired, pair.Leaf.NotAfter.UTC().Format(time.DateOnly))
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
	if config.TrustCABundlePEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(config.TrustCABundlePEM)) {
			return nil, "", ctxerr.New(ctx, msgNoTrustCerts)
		}
		tlsConfig.RootCAs = pool
	}
	// Redirects are not followed, so an EJBCA response can't send Fleet's client certificate or the enrollment request elsewhere.
	client := fleethttp.NewClient(fleethttp.WithTimeout(s.timeout), fleethttp.WithTLSClientConfig(tlsConfig), fleethttp.WithFollowRedir(false))
	return client, base.JoinPath(restAPIPrefix).String(), nil
}

// do sends a request to EJBCA and returns the status and body. Failing to get a response is an error, with a message that says what to
// check, and any response is returned for the caller to interpret.
func (s *Service) do(ctx context.Context, client *http.Client, base, method, path string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, base+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "creating EJBCA request")
	}
	req.Header.Set("Accept", "application/json")
	// Each operation builds its own client, so don't leave its connection idle until the timeout, which adds up during bulk enrollment.
	req.Close = true
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, transportError(ctx, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return 0, nil, transportError(ctx, err)
	}
	return resp.StatusCode, respBody, nil
}

// transportError explains a request that got no response.
func transportError(ctx context.Context, err error) error {
	// Name the EJBCA address once instead of repeating the request URL in the error.
	target := "EJBCA"
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		if u, parseErr := url.Parse(urlErr.URL); parseErr == nil {
			target = u.Scheme + "://" + u.Host
		}
		err = urlErr.Err
	}
	var verifyErr *tls.CertificateVerificationError
	var opErr *net.OpError
	switch {
	case errors.As(err, &verifyErr):
		return ctxerr.Errorf(ctx, msgServerCertUntrusted, verifyErr.Err)
	case errors.As(err, &opErr) && opErr.Op == "remote error":
		// A TLS alert from EJBCA, such as bad certificate, which is how it refuses a client certificate it doesn't trust.
		return ctxerr.Errorf(ctx, msgClientCertRejected, opErr.Err)
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return certissue.Transient(ctx, ctxerr.New(ctx, msgConnectionClosed))
	default:
		return certissue.Transient(ctx, ctxerr.Errorf(ctx, msgUnreachable, target, err))
	}
}

// statusError explains an unsuccessful response from EJBCA, which includes EJBCA's own message.
func statusError(ctx context.Context, status int, body []byte, enrolling bool) error {
	// EJBCA ends its authorization errors with an empty " Msg: .".
	message := strings.TrimRight(strings.TrimSuffix(strings.TrimSpace(errorMessage(body)), "Msg: ."), ". ")
	switch {
	case status == http.StatusForbidden && bytes.Contains(body, []byte(serviceDisabledText)):
		return ctxerr.New(ctx, msgProtocolDisabled)
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ctxerr.Errorf(ctx, msgNotAuthorized, message)
	case status == http.StatusNotFound:
		return ctxerr.New(ctx, msgNotFound)
	case status == http.StatusBadRequest && enrolling:
		return ctxerr.Errorf(ctx, msgEnrollRejected, message)
	default:
		return certissue.TransientStatus(ctxerr.Errorf(ctx, msgUnexpected, status, message), status)
	}
}

// errorMessage is EJBCA's error_message, or the start of the body when it isn't the usual JSON error.
func errorMessage(body []byte) string {
	var errResp struct {
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.ErrorMessage != "" {
		return errResp.ErrorMessage
	}
	text := strings.TrimSpace(string(body))
	if len(text) > 200 {
		text = text[:200]
	}
	return text
}
