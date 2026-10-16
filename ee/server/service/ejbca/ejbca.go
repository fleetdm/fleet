// Package ejbca is Fleet's client for the EJBCA REST API, authenticated with a TLS client certificate.
// https://docs.keyfactor.com/ejbca/latest/ejbca-rest-interface
package ejbca

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/crypto/cryptobyte"
	cryptobyteasn1 "golang.org/x/crypto/cryptobyte/asn1"
	"software.sslmate.com/src/go-pkcs12"
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
)

// User-facing messages. They end up in the UI, so they say what to check.
const (
	msgInvalidURL = "The EJBCA URL must be an https URL, such as https://ejbca.example.com:8443, without credentials, " +
		"a query, or a fragment."
	msgCertExpired         = "The TLS client certificate Fleet uses for EJBCA expired on %s. Upload a new client certificate."
	msgServerCertUntrusted = "Couldn't verify EJBCA's server certificate: %s. Add the CA that issued it to the trust CA bundle."
	msgClientCertRejected  = "EJBCA rejected the TLS client certificate (%s). Check that EJBCA trusts the CA that issued it."
	msgConnectionClosed    = "EJBCA closed the connection without responding. Check that the REST Certificate Management protocol is " +
		"enabled in EJBCA, and restart EJBCA if you just enabled it."
	msgUnreachable      = "Couldn't connect to EJBCA at %s: %s"
	msgProtocolDisabled = "EJBCA's REST Certificate Management protocol is disabled. Enable it under System Configuration > " +
		"Protocol Configuration in EJBCA, and restart EJBCA if needed."
	msgNotAuthorized = "EJBCA didn't authorize the Fleet client certificate: %s. Check that EJBCA trusts the CA that issued it, and that " +
		"it is a member of an EJBCA role with the access rules Fleet needs."
	msgNoCAs              = "The Fleet client certificate can't access any CAs in EJBCA. Add the CA to the access rules of its EJBCA role."
	msgCANotFound         = "EJBCA CA %q was not found. The Fleet client certificate can access: %s."
	msgProfileNotFound    = "EJBCA certificate profile %q was not found."
	msgEnrollRejected     = "EJBCA rejected the enrollment request: %s. Check the CA, certificate profile, and end entity profile names."
	msgUnexpected         = "Unexpected response from EJBCA, status %d: %s"
	msgCommonNameMismatch = "EJBCA issued a certificate with the common name %q, not the requested %q. Check that the end entity profile " +
		"takes the subject from the request."
	msgUPNMissing = "EJBCA left the user principal name %q out of the certificate. Check that the end entity and certificate profiles " +
		"allow it."
)

var (
	// oidMicrosoftUPN is the otherName type for a User Principal Name in a subjectAltName.
	oidMicrosoftUPN = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 20, 2, 3}
	// oidSubjectAltName is the subjectAltName extension.
	oidSubjectAltName = asn1.ObjectIdentifier{2, 5, 29, 17}
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
func (s *Service) VerifyConnection(ctx context.Context, config fleet.EJBCACA) error {
	client, base, err := s.newClient(ctx, config)
	if err != nil {
		return err
	}

	// /ca/status needs no authentication, so it would prove nothing. Listing the CAs does.
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
func (s *Service) GetCertificate(ctx context.Context, config fleet.EJBCACA) (*fleet.EJBCACertificate, error) {
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

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating RSA private key")
	}
	csrTemplate := &x509.CertificateRequest{Subject: pkix.Name{CommonName: config.CertificateCommonName}}
	if ext, ok := upnExtension(config.CertificateUserPrincipalNames); ok {
		csrTemplate.ExtraExtensions = []pkix.Extension{ext}
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating CSR")
	}

	// EJBCA requires a password in the request. The client certificate authenticates Fleet, so a random one that is never kept will do.
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating EJBCA enrollment password")
	}
	reqBody, err := json.Marshal(map[string]any{
		"certificate_request":        strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes}))),
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
	// An end entity profile can change the subject or drop the UPNs, and the host would get a certificate that doesn't identify it.
	if cert.Subject.CommonName != config.CertificateCommonName {
		return nil, ctxerr.Errorf(ctx, msgCommonNameMismatch, cert.Subject.CommonName, config.CertificateCommonName)
	}
	issuedUPNs := upnsOf(cert)
	for _, upn := range nonBlank(config.CertificateUserPrincipalNames) {
		if !slices.Contains(issuedUPNs, upn) {
			return nil, ctxerr.Errorf(ctx, msgUPNMissing, upn)
		}
	}

	pfxPassword, err := server.GenerateRandomText(10)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating password for PKCS12 bundle")
	}
	pfxData, err := pkcs12.Legacy.Encode(privateKey, cert, nil, pfxPassword)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating PKCS12 bundle")
	}

	// The serial is hex, like EJBCA shows it and like the managed certificate record stores it.
	serial := strings.ToUpper(cert.SerialNumber.Text(16))
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
func (s *Service) newClient(ctx context.Context, config fleet.EJBCACA) (*http.Client, string, error) {
	base, err := url.Parse(strings.TrimRight(config.URL, "/"))
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.ForceQuery ||
		base.Fragment != "" {
		return nil, "", ctxerr.New(ctx, msgInvalidURL)
	}

	pair, err := tls.X509KeyPair([]byte(config.MTLSCertPEM), []byte(config.MTLSKeyPEM))
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "loading EJBCA client certificate and key")
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "parsing EJBCA client certificate")
	}
	if time.Now().After(leaf.NotAfter) {
		return nil, "", ctxerr.Errorf(ctx, msgCertExpired, leaf.NotAfter.UTC().Format(time.DateOnly))
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
	if config.TrustCABundlePEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(config.TrustCABundlePEM)) {
			return nil, "", ctxerr.New(ctx, "the EJBCA trust CA bundle has no usable certificates")
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
		return ctxerr.New(ctx, msgConnectionClosed)
	default:
		return ctxerr.Errorf(ctx, msgUnreachable, target, err)
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
	case status == http.StatusBadRequest && enrolling:
		return ctxerr.Errorf(ctx, msgEnrollRejected, message)
	default:
		return ctxerr.Errorf(ctx, msgUnexpected, status, message)
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

// upnExtension builds a subjectAltName extension with a Microsoft UPN otherName for each non-blank name:
//
//	GeneralName ::= CHOICE { otherName [0] OtherName, ... }
//	OtherName   ::= SEQUENCE { type-id OBJECT IDENTIFIER, value [0] EXPLICIT ANY DEFINED BY type-id }
//
// where the value is a UTF8String. EJBCA keeps a UPN from the CSR only if the end entity profile allows it.
func upnExtension(upns []string) (pkix.Extension, bool) {
	names := nonBlank(upns)
	if len(names) == 0 {
		return pkix.Extension{}, false
	}
	contextConstructed := cryptobyteasn1.Tag(0).ContextSpecific().Constructed()
	var b cryptobyte.Builder
	b.AddASN1(cryptobyteasn1.SEQUENCE, func(b *cryptobyte.Builder) {
		for _, upn := range names {
			b.AddASN1(contextConstructed, func(b *cryptobyte.Builder) {
				b.AddASN1ObjectIdentifier(oidMicrosoftUPN)
				b.AddASN1(contextConstructed, func(b *cryptobyte.Builder) {
					b.AddASN1(cryptobyteasn1.UTF8String, func(b *cryptobyte.Builder) { b.AddBytes([]byte(upn)) })
				})
			})
		}
	})
	value, err := b.Bytes()
	if err != nil {
		return pkix.Extension{}, false
	}
	return pkix.Extension{Id: oidSubjectAltName, Value: value}, true
}

func nonBlank(values []string) []string {
	return slices.DeleteFunc(slices.Clone(values), func(v string) bool { return strings.TrimSpace(v) == "" })
}

// upnsOf returns the Microsoft UPNs in the subjectAltName of the certificate.
func upnsOf(cert *x509.Certificate) []string {
	otherName := cryptobyteasn1.Tag(0).ContextSpecific().Constructed()
	var upns []string
	for _, ext := range cert.Extensions {
		names := cryptobyte.String(ext.Value)
		if !ext.Id.Equal(oidSubjectAltName) || !names.ReadASN1(&names, cryptobyteasn1.SEQUENCE) {
			continue
		}
		for !names.Empty() {
			var name cryptobyte.String
			var tag cryptobyteasn1.Tag
			if !names.ReadAnyASN1(&name, &tag) {
				break
			}
			var oid asn1.ObjectIdentifier
			var value, upn cryptobyte.String
			if tag == otherName && name.ReadASN1ObjectIdentifier(&oid) && oid.Equal(oidMicrosoftUPN) &&
				name.ReadASN1(&value, otherName) && value.ReadASN1(&upn, cryptobyteasn1.UTF8String) {
				upns = append(upns, string(upn))
			}
		}
	}
	return upns
}
