// Package ejbcatest has a fake EJBCA that tests can run the EJBCA client and Fleet against. It answers like the REST API of EJBCA 9.6.3,
// over TLS that requires a client certificate.
package ejbcatest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json/v2"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/certissue"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

// CA issues the client certificates that the fake EJBCA trusts, and the certificates it enrolls.
type CA struct {
	Cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func NewCA(t *testing.T, name string) *CA {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return &CA{Cert: cert, key: key}
}

// Sign signs a certificate for the public key.
func (ca *CA) Sign(template *x509.Certificate, pub any) (*x509.Certificate, error) {
	if template.SerialNumber == nil {
		var err error
		if template.SerialNumber, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 159)); err != nil {
			return nil, err
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, pub, ca.key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}

func (ca *CA) Issue(t *testing.T, template *x509.Certificate, pub any) *x509.Certificate {
	cert, err := ca.Sign(template, pub)
	require.NoError(t, err)
	return cert
}

// clientCert issues a client certificate that expires at notAfter and is valid since an hour ago, or since an hour before it expires.
func (ca *CA) clientCert(t *testing.T, notAfter time.Time) (*x509.Certificate, *ecdsa.PrivateKey) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	notBefore := time.Now().Add(-time.Hour)
	if !notBefore.Before(notAfter) {
		notBefore = notAfter.Add(-time.Hour)
	}
	cert := ca.Issue(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "Fleet"},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, &key.PublicKey)
	return cert, key
}

// IssueClient returns a client certificate and key, in PEM, that expire at notAfter.
func (ca *CA) IssueClient(t *testing.T, notAfter time.Time) (certPEM, keyPEM string) {
	cert, key := ca.clientCert(t, notAfter)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}

const (
	// CAName, CertProfile, and EndEntityProfile are the names the fake knows by default.
	CAName           = "ManagementCA"
	CertProfile      = "fleetCertProfile"
	EndEntityProfile = "fleetEEProfile"
	// RESTAPIPrefix is where EJBCA serves its REST API.
	RESTAPIPrefix       = "/ejbca/ejbca-rest-api"
	adminNotAuthorized  = `{"error_code":403,"error_message":"Administrator is not authorized to resource /administrator. Msg: ."}`
	serviceDisabledPage = `<html><head><title>Error</title></head><body>This service has been disabled.</body></html>`
)

// Server answers like EJBCA's REST API does, as observed against EJBCA 9.6.3: it requires a TLS client certificate and issues
// certificates from the CSRs it is sent.
type Server struct {
	HTTP      *httptest.Server
	ClientCA  *CA
	ServerPEM string

	// Behavior, set before starting.
	CAs               []string
	CertProfiles      []string
	EndEntityProfiles []string
	Unauthorized      bool // the client certificate has no EJBCA role
	ProtocolDisabled  bool
	V2Disabled        bool
	EnrollStatus      int // when set, enrolling fails with this status and an EJBCA error
	WrongKey          bool
	DropSAN           bool     // the issued certificate leaves out the requested subjectAltName
	IssuedCommonName  *string  // when set, the issued certificate has this common name, possibly empty
	Serial            *big.Int // when set, the issued certificate has this serial number
	RedirectTo        string   // when set, every request is redirected there
	OnRequest         func(r *http.Request)

	mu       sync.Mutex
	requests []string
	enrolls  []map[string]any
	conns    map[net.Conn]struct{} // the open connections
}

func New(t *testing.T, configure func(*Server)) *Server {
	f := &Server{
		ClientCA:          NewCA(t, "Client CA"),
		CAs:               []string{CAName},
		CertProfiles:      []string{CertProfile},
		EndEntityProfiles: []string{EndEntityProfile},
		conns:             map[net.Conn]struct{}{},
	}
	if configure != nil {
		configure(f)
	}
	issuer := NewCA(t, CAName)

	pool := x509.NewCertPool()
	pool.AddCert(f.ClientCA.Cert)
	f.HTTP = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.serve(t, issuer, w, r) }))
	f.HTTP.Config.ErrorLog = log.New(io.Discard, "", 0) // the handshake errors that the tests cause
	f.HTTP.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if state == http.StateNew {
			f.conns[conn] = struct{}{}
		} else if state == http.StateClosed {
			delete(f.conns, conn)
		}
	}
	f.HTTP.TLS = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool, MinVersion: tls.VersionTLS12}
	f.HTTP.StartTLS()
	t.Cleanup(f.HTTP.Close)
	f.ServerPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.HTTP.Certificate().Raw}))
	return f
}

// Snapshot returns what the fake received, safely against its handler goroutines.
func (f *Server) Snapshot() (requests []string, enrolls []map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests), slices.Clone(f.enrolls)
}

func (f *Server) OpenConns() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.conns)
}

func (f *Server) RequestCount() int {
	requests, _ := f.Snapshot()
	return len(requests)
}

func (f *Server) serve(t *testing.T, issuer *CA, w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mu.Unlock()

	if f.OnRequest != nil {
		f.OnRequest(r)
	}
	if f.RedirectTo != "" {
		http.Redirect(w, r, f.RedirectTo, http.StatusTemporaryRedirect)
		return
	}
	status, body, err := f.respond(issuer, r)
	if err != nil {
		t.Errorf("fake EJBCA: %v", err) // require can't be used from a handler goroutine
		status, body = http.StatusInternalServerError, err.Error()
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body)) // nolint:gosec // G705: a test server answering with its own fixtures
}

func (f *Server) respond(issuer *CA, r *http.Request) (int, string, error) {
	apiError := func(status int, message string) (int, string, error) {
		body, err := json.Marshal(map[string]any{"error_code": status, "error_message": message})
		return status, string(body), err
	}
	path := strings.TrimPrefix(r.URL.Path, RESTAPIPrefix)
	switch {
	case f.ProtocolDisabled, f.V2Disabled && strings.HasPrefix(path, "/v2/"):
		return http.StatusForbidden, serviceDisabledPage, nil
	case path == "/v1/ca/status":
		return http.StatusOK, `{"status":"OK"}`, nil
	case f.Unauthorized:
		return http.StatusForbidden, adminNotAuthorized, nil
	case path == "/v1/ca":
		type ca struct {
			Name string `json:"name"`
		}
		list := []ca{}
		for _, name := range f.CAs {
			list = append(list, ca{name})
		}
		body, err := json.Marshal(map[string]any{"certificate_authorities": list})
		return http.StatusOK, string(body), err
	case strings.HasPrefix(path, "/v2/certificate/profile/"):
		if slices.Contains(f.CertProfiles, strings.TrimPrefix(path, "/v2/certificate/profile/")) {
			return http.StatusOK, `{"certificate_profile_id":1}`, nil
		}
		return apiError(http.StatusBadRequest, "Invalid search criteria, unknown certificate profile.")
	case path == "/v1/certificate/pkcs10enroll":
		return f.enroll(issuer, r)
	default:
		return apiError(http.StatusNotFound, "no such endpoint")
	}
}

func (f *Server) enroll(issuer *CA, r *http.Request) (int, string, error) {
	var body map[string]any
	if err := json.UnmarshalRead(r.Body, &body); err != nil {
		return 0, "", err
	}
	f.mu.Lock()
	f.enrolls = append(f.enrolls, body)
	f.mu.Unlock()

	fail := func(status int, format string, args ...any) (int, string, error) {
		resp, err := json.Marshal(map[string]any{"error_code": status, "error_message": fmt.Sprintf(format, args...)})
		return status, string(resp), err
	}
	switch {
	case f.EnrollStatus != 0:
		return fail(f.EnrollStatus, "General failure.")
	case !slices.Contains(f.CAs, fmt.Sprint(body["certificate_authority_name"])):
		return fail(http.StatusBadRequest, "CA with name %q doesn't exist", body["certificate_authority_name"])
	case !slices.Contains(f.CertProfiles, fmt.Sprint(body["certificate_profile_name"])):
		return fail(http.StatusBadRequest, "Certificate profile with name %q doesn't exist", body["certificate_profile_name"])
	case !slices.Contains(f.EndEntityProfiles, fmt.Sprint(body["end_entity_profile_name"])):
		return fail(http.StatusBadRequest, "End Entity Profile of name %q was not found", body["end_entity_profile_name"])
	}

	block, _ := pem.Decode([]byte(fmt.Sprint(body["certificate_request"])))
	if block == nil {
		return 0, "", errors.New("the CSR is not PEM")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return 0, "", err
	}
	if err := csr.CheckSignature(); err != nil {
		return 0, "", err
	}
	pub := csr.PublicKey
	if f.WrongKey {
		other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return 0, "", err
		}
		pub = &other.PublicKey
	}
	// Like EJBCA with an end entity profile that allows it, keep the subject and subjectAltName of the CSR.
	var extensions []pkix.Extension
	for _, ext := range csr.Extensions {
		if ext.Id.Equal(certissue.OIDSubjectAltName) && !f.DropSAN {
			extensions = append(extensions, ext)
		}
	}
	subject := csr.Subject
	if f.IssuedCommonName != nil {
		subject.CommonName = *f.IssuedCommonName
	}
	cert, err := issuer.Sign(&x509.Certificate{
		SerialNumber:    f.Serial,
		Subject:         subject,
		NotBefore:       time.Now().Add(-time.Minute),
		NotAfter:        time.Now().Add(365 * 24 * time.Hour),
		ExtraExtensions: extensions,
	}, pub)
	if err != nil {
		return 0, "", err
	}
	resp, err := json.Marshal(map[string]any{
		"certificate":     base64.StdEncoding.EncodeToString(cert.Raw),
		"serial_number":   cert.SerialNumber.Text(16),
		"response_format": "DER",
	})
	return http.StatusCreated, string(resp), err
}

// Config is a configuration for the fake EJBCA with a valid client certificate.
func (f *Server) Config(t *testing.T) fleet.EJBCA {
	certPEM, keyPEM := f.ClientCA.IssueClient(t, time.Now().Add(24*time.Hour))
	return fleet.EJBCA{
		Name:                          "EJBCA",
		URL:                           f.HTTP.URL,
		MTLSCertPEM:                   certPEM,
		MTLSKeyPEM:                    keyPEM,
		TrustCABundlePEM:              f.ServerPEM,
		CertificateAuthorityName:      CAName,
		CertificateProfileName:        CertProfile,
		EndEntityProfileName:          EndEntityProfile,
		CertificateCommonName:         "alice@example.com",
		UsernameTemplate:              "SERIAL123",
		CertificateUserPrincipalNames: []string{"alice@example.com"},
	}
}

// Requests returns the method and path of each request the fake received.
func (f *Server) Requests() []string {
	requests, _ := f.Snapshot()
	return requests
}

// ClientP12 returns a base64 PKCS#12 of a client certificate that the fake trusts, which expires at notAfter.
func (f *Server) ClientP12(t *testing.T, notAfter time.Time, password string) string {
	t.Helper()
	return ClientP12(t, f.ClientCA, notAfter, password)
}

// ClientP12 returns a base64 PKCS#12 of a client certificate issued by the CA.
func ClientP12(t *testing.T, ca *CA, notAfter time.Time, password string) string {
	t.Helper()
	cert, key := ca.clientCert(t, notAfter)
	p12, err := pkcs12.Modern.Encode(key, cert, []*x509.Certificate{ca.Cert}, password)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(p12)
}
