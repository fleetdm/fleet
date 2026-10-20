package ejbca

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
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
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/certissue"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

// testCA issues the client certificates that the fake EJBCA trusts, and the certificates it enrolls.
type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newTestCA(t *testing.T, name string) *testCA {
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
	return &testCA{cert: cert, key: key}
}

// sign signs a certificate for the public key.
func (ca *testCA) sign(template *x509.Certificate, pub any) (*x509.Certificate, error) {
	if template.SerialNumber == nil {
		var err error
		if template.SerialNumber, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 159)); err != nil {
			return nil, err
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, pub, ca.key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}

func (ca *testCA) issue(t *testing.T, template *x509.Certificate, pub any) *x509.Certificate {
	cert, err := ca.sign(template, pub)
	require.NoError(t, err)
	return cert
}

// issueClient returns a client certificate and key, in PEM, that expire at notAfter.
func (ca *testCA) issueClient(t *testing.T, notAfter time.Time) (certPEM, keyPEM string) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	cert := ca.issue(t, &x509.Certificate{
		Subject:     pkix.Name{CommonName: "Fleet"},
		NotBefore:   notAfter.Add(-48 * time.Hour),
		NotAfter:    notAfter,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, &key.PublicKey)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}

const (
	testCAName          = "ManagementCA"
	testCertProfile     = "fleetCertProfile"
	testEndEntityProf   = "fleetEEProfile"
	adminNotAuthorized  = `{"error_code":403,"error_message":"Administrator is not authorized to resource /administrator. Msg: ."}`
	serviceDisabledPage = `<html><head><title>Error</title></head><body>This service has been disabled.</body></html>`
)

// fakeEJBCA answers like EJBCA's REST API does, as observed against EJBCA 9.6.3: it requires a TLS client certificate and issues
// certificates from the CSRs it is sent.
type fakeEJBCA struct {
	srv       *httptest.Server
	clientCA  *testCA
	serverPEM string

	// Behavior, set before starting.
	cas               []string
	certProfiles      []string
	endEntityProfiles []string
	unauthorized      bool // the client certificate has no EJBCA role
	protocolDisabled  bool
	v2Disabled        bool
	enrollStatus      int // when set, enrolling fails with this status and an EJBCA error
	wrongKey          bool
	dropSAN           bool     // the issued certificate leaves out the requested subjectAltName
	issuedCommonName  *string  // when set, the issued certificate has this common name, possibly empty
	serial            *big.Int // when set, the issued certificate has this serial number
	redirectTo        string   // when set, every request is redirected there
	onRequest         func(r *http.Request)

	mu       sync.Mutex
	requests []string
	enrolls  []map[string]any
	conns    map[net.Conn]struct{} // the open connections
}

func newFakeEJBCA(t *testing.T, configure func(*fakeEJBCA)) *fakeEJBCA {
	f := &fakeEJBCA{
		clientCA:          newTestCA(t, "Client CA"),
		cas:               []string{testCAName},
		certProfiles:      []string{testCertProfile},
		endEntityProfiles: []string{testEndEntityProf},
		conns:             map[net.Conn]struct{}{},
	}
	if configure != nil {
		configure(f)
	}
	issuer := newTestCA(t, testCAName)

	pool := x509.NewCertPool()
	pool.AddCert(f.clientCA.cert)
	f.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.serve(t, issuer, w, r) }))
	f.srv.Config.ErrorLog = log.New(io.Discard, "", 0) // the handshake errors that the tests cause
	f.srv.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if state == http.StateNew {
			f.conns[conn] = struct{}{}
		} else if state == http.StateClosed {
			delete(f.conns, conn)
		}
	}
	f.srv.TLS = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool, MinVersion: tls.VersionTLS12}
	f.srv.StartTLS()
	t.Cleanup(f.srv.Close)
	f.serverPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.srv.Certificate().Raw}))
	return f
}

// snapshot returns what the fake received, safely against its handler goroutines.
func (f *fakeEJBCA) snapshot() (requests []string, enrolls []map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests), slices.Clone(f.enrolls)
}

func (f *fakeEJBCA) openConns() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.conns)
}

func (f *fakeEJBCA) requestCount() int {
	requests, _ := f.snapshot()
	return len(requests)
}

func (f *fakeEJBCA) serve(t *testing.T, issuer *testCA, w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mu.Unlock()

	if f.onRequest != nil {
		f.onRequest(r)
	}
	if f.redirectTo != "" {
		http.Redirect(w, r, f.redirectTo, http.StatusTemporaryRedirect)
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

func (f *fakeEJBCA) respond(issuer *testCA, r *http.Request) (int, string, error) {
	apiError := func(status int, message string) (int, string, error) {
		body, err := json.Marshal(map[string]any{"error_code": status, "error_message": message})
		return status, string(body), err
	}
	path := strings.TrimPrefix(r.URL.Path, restAPIPrefix)
	switch {
	case f.protocolDisabled, f.v2Disabled && strings.HasPrefix(path, "/v2/"):
		return http.StatusForbidden, serviceDisabledPage, nil
	case path == "/v1/ca/status":
		return http.StatusOK, `{"status":"OK"}`, nil
	case f.unauthorized:
		return http.StatusForbidden, adminNotAuthorized, nil
	case path == "/v1/ca":
		type ca struct {
			Name string `json:"name"`
		}
		list := []ca{}
		for _, name := range f.cas {
			list = append(list, ca{name})
		}
		body, err := json.Marshal(map[string]any{"certificate_authorities": list})
		return http.StatusOK, string(body), err
	case strings.HasPrefix(path, "/v2/certificate/profile/"):
		if slices.Contains(f.certProfiles, strings.TrimPrefix(path, "/v2/certificate/profile/")) {
			return http.StatusOK, `{"certificate_profile_id":1}`, nil
		}
		return apiError(http.StatusBadRequest, "Invalid search criteria, unknown certificate profile.")
	case path == "/v1/certificate/pkcs10enroll":
		return f.enroll(issuer, r)
	default:
		return apiError(http.StatusNotFound, "no such endpoint")
	}
}

func (f *fakeEJBCA) enroll(issuer *testCA, r *http.Request) (int, string, error) {
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
	case f.enrollStatus != 0:
		return fail(f.enrollStatus, "General failure.")
	case !slices.Contains(f.cas, fmt.Sprint(body["certificate_authority_name"])):
		return fail(http.StatusBadRequest, "CA with name %q doesn't exist", body["certificate_authority_name"])
	case !slices.Contains(f.certProfiles, fmt.Sprint(body["certificate_profile_name"])):
		return fail(http.StatusBadRequest, "Certificate profile with name %q doesn't exist", body["certificate_profile_name"])
	case !slices.Contains(f.endEntityProfiles, fmt.Sprint(body["end_entity_profile_name"])):
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
	if f.wrongKey {
		other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return 0, "", err
		}
		pub = &other.PublicKey
	}
	// Like EJBCA with an end entity profile that allows it, keep the subject and subjectAltName of the CSR.
	var extensions []pkix.Extension
	for _, ext := range csr.Extensions {
		if ext.Id.Equal(certissue.OIDSubjectAltName) && !f.dropSAN {
			extensions = append(extensions, ext)
		}
	}
	subject := csr.Subject
	if f.issuedCommonName != nil {
		subject.CommonName = *f.issuedCommonName
	}
	cert, err := issuer.sign(&x509.Certificate{
		SerialNumber:    f.serial,
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

// config is a configuration for the fake EJBCA with a valid client certificate.
func (f *fakeEJBCA) config(t *testing.T) fleet.EJBCA {
	certPEM, keyPEM := f.clientCA.issueClient(t, time.Now().Add(24*time.Hour))
	return fleet.EJBCA{
		Name:                          "EJBCA",
		URL:                           f.srv.URL,
		MTLSCertPEM:                   certPEM,
		MTLSKeyPEM:                    keyPEM,
		TrustCABundlePEM:              f.serverPEM,
		CertificateAuthorityName:      testCAName,
		CertificateProfileName:        testCertProfile,
		EndEntityProfileName:          testEndEntityProf,
		CertificateCommonName:         "alice@example.com",
		UsernameTemplate:              "SERIAL123",
		CertificateUserPrincipalNames: []string{"alice@example.com"},
	}
}

func TestVerifyConnection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		configure func(*fakeEJBCA)
		modify    func(t *testing.T, f *fakeEJBCA, config *fleet.EJBCA)
		wantErr   string
		// wantNoRequests is for failures that must not contact EJBCA.
		wantNoRequests bool
	}{
		// It lists the CAs, which needs authentication, and doesn't rely on the status endpoint, which doesn't.
		{name: "valid"},
		{
			name:      "unknown certificate profile",
			configure: func(f *fakeEJBCA) { f.certProfiles = []string{"other"} },
			wantErr:   fmt.Sprintf(msgProfileNotFound, testCertProfile),
		},
		{
			name:      "certificate profile lookup disabled",
			configure: func(f *fakeEJBCA) { f.certProfiles, f.v2Disabled = []string{"other"}, true },
		},
		{
			name:      "CA the certificate can't use",
			configure: func(f *fakeEJBCA) { f.cas = []string{"SubCA", "AnotherCA"} },
			wantErr:   fmt.Sprintf(msgCANotFound, testCAName, "AnotherCA, SubCA"),
		},
		{name: "no CAs", configure: func(f *fakeEJBCA) { f.cas = nil }, wantErr: msgNoCAs},
		{
			name:      "certificate without an EJBCA role",
			configure: func(f *fakeEJBCA) { f.unauthorized = true },
			wantErr:   fmt.Sprintf(msgNotAuthorized, "Administrator is not authorized to resource /administrator"),
		},
		{name: "protocol disabled", configure: func(f *fakeEJBCA) { f.protocolDisabled = true }, wantErr: msgProtocolDisabled},
		{
			name:    "URL with the /ejbca path of the admin UI",
			modify:  func(_ *testing.T, f *fakeEJBCA, config *fleet.EJBCA) { config.URL = f.srv.URL + "/ejbca/" },
			wantErr: msgNotFound,
		},
		{
			name:    "server certificate not trusted",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCA) { config.TrustCABundlePEM = "" },
			wantErr: strings.Split(msgServerCertUntrusted, "%s")[0],
		},
		{
			name: "client certificate from a CA EJBCA doesn't trust",
			modify: func(t *testing.T, _ *fakeEJBCA, config *fleet.EJBCA) {
				config.MTLSCertPEM, config.MTLSKeyPEM = newTestCA(t, "Other CA").issueClient(t, time.Now().Add(time.Hour))
			},
			wantErr: strings.Split(msgClientCertRejected, "%s")[0],
		},
		{
			name: "client certificate expired",
			modify: func(t *testing.T, f *fakeEJBCA, config *fleet.EJBCA) {
				notAfter := time.Now().Add(-time.Hour)
				config.MTLSCertPEM, config.MTLSKeyPEM = f.clientCA.issueClient(t, notAfter)
			},
			wantErr:        strings.Split(msgCertExpired, "%s")[0],
			wantNoRequests: true,
		},
		{
			name:    "trust bundle without certificates",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCA) { config.TrustCABundlePEM = "not a certificate" },
			wantErr: msgNoTrustCerts,
		},
		{
			name:    "server not running",
			modify:  func(_ *testing.T, f *fakeEJBCA, _ *fleet.EJBCA) { f.srv.Close() },
			wantErr: strings.Split(msgUnreachable, "%s")[0],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeEJBCA(t, tt.configure)
			config := f.config(t)
			if tt.modify != nil {
				tt.modify(t, f, &config)
			}

			err := NewService().VerifyConnection(t.Context(), config)
			if tt.wantErr == "" {
				require.NoError(t, err)
				requests, _ := f.snapshot()
				require.Equal(t, []string{
					"GET " + restAPIPrefix + "/v1/ca",
					"GET " + restAPIPrefix + "/v2/certificate/profile/" + testCertProfile,
				}, requests)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
			if tt.wantNoRequests {
				require.Zero(t, f.requestCount())
			}
		})
	}
}

// The URL is checked before anything else, so these need no EJBCA.
func TestInvalidURLs(t *testing.T) {
	t.Parallel()
	for _, url := range []string{
		"http://ejbca.example.com",
		"https://user:pass@ejbca.example.com",
	} {
		t.Run(url, func(t *testing.T) {
			require.ErrorContains(t, NewService().VerifyConnection(t.Context(), fleet.EJBCA{URL: url}), msgInvalidURL)
		})
	}
}

// A query or fragment pasted with the URL is dropped, not sent on to EJBCA.
func TestURLQueryAndFragmentDropped(t *testing.T) {
	t.Parallel()
	var queries atomic.Int32
	f := newFakeEJBCA(t, func(f *fakeEJBCA) {
		f.onRequest = func(r *http.Request) {
			if r.URL.RawQuery != "" {
				queries.Add(1)
			}
		}
	})
	config := f.config(t)
	config.URL += "?tenant=prod#ca"
	require.NoError(t, NewService().VerifyConnection(t.Context(), config))
	require.NotZero(t, f.requestCount())
	require.Zero(t, queries.Load())
}

func TestGetCertificate(t *testing.T) {
	t.Parallel()
	f := newFakeEJBCA(t, nil)
	config := f.config(t)
	config.CertificateUserPrincipalNames = []string{"alice@example.com", " ", "alice@corp.example.com"}
	s := NewService()

	cert, err := s.GetCertificate(t.Context(), config)
	require.NoError(t, err)

	// What EJBCA was asked for. The CSR common name and the username are separate.
	_, enrolls := f.snapshot()
	require.Len(t, enrolls, 1)
	body := enrolls[0]
	assert.Equal(t, config.UsernameTemplate, body["username"])
	assert.Equal(t, config.CertificateAuthorityName, body["certificate_authority_name"])
	assert.Equal(t, config.CertificateProfileName, body["certificate_profile_name"])
	assert.Equal(t, config.EndEntityProfileName, body["end_entity_profile_name"])
	assert.Equal(t, "DER", body["response_format"])
	assert.Len(t, body["password"], 64)

	// What Fleet got back: the key and the certificate EJBCA issued, in a bundle that the password opens.
	key, issued, err := pkcs12.Decode(cert.PfxData, cert.Password)
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", issued.Subject.CommonName)
	assert.True(t, key.(*rsa.PrivateKey).PublicKey.Equal(issued.PublicKey))
	assert.Equal(t, fmt.Sprintf("%040s", issued.SerialNumber.Text(16)), cert.SerialNumber)
	assert.Len(t, cert.SerialNumber, 40, "the serial is padded to the length of the serial column")
	assert.True(t, issued.NotBefore.Equal(cert.NotValidBefore))
	assert.True(t, issued.NotAfter.Equal(cert.NotValidAfter))
	assert.Equal(t, []string{"alice@example.com", "alice@corp.example.com"}, certissue.UPNs(issued))

	// The enrollment password is not reused, and each certificate gets its own key and password.
	second, err := s.GetCertificate(t.Context(), config)
	require.NoError(t, err)
	_, enrolls = f.snapshot()
	assert.NotEqual(t, enrolls[0]["password"], enrolls[1]["password"])
	assert.NotEqual(t, cert.Password, second.Password)
}

func TestGetCertificateErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		configure func(*fakeEJBCA)
		modify    func(t *testing.T, f *fakeEJBCA, config *fleet.EJBCA)
		wantErr   string
	}{
		{
			name:    "unknown CA",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCA) { config.CertificateAuthorityName = "Missing" },
			wantErr: fmt.Sprintf(msgEnrollRejected, `CA with name "Missing" doesn't exist`),
		},
		{
			name:    "unknown certificate profile",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCA) { config.CertificateProfileName = "Missing" },
			wantErr: fmt.Sprintf(msgEnrollRejected, `Certificate profile with name "Missing" doesn't exist`),
		},
		{
			name:    "unknown end entity profile",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCA) { config.EndEntityProfileName = "Missing" },
			wantErr: fmt.Sprintf(msgEnrollRejected, `End Entity Profile of name "Missing" was not found`),
		},
		{
			name:    "URL with the /ejbca path of the admin UI",
			modify:  func(_ *testing.T, f *fakeEJBCA, config *fleet.EJBCA) { config.URL = f.srv.URL + "/ejbca" },
			wantErr: msgNotFound,
		},
		{
			name:      "certificate with a serial number that doesn't fit",
			configure: func(f *fakeEJBCA) { f.serial = new(big.Int).Lsh(big.NewInt(1), 167) },
			wantErr:   strings.Split(msgSerialTooLong, "%d")[0],
		},
		{
			name:      "EJBCA server error",
			configure: func(f *fakeEJBCA) { f.enrollStatus = http.StatusInternalServerError },
			wantErr:   fmt.Sprintf(msgUnexpected, http.StatusInternalServerError, "General failure"),
		},
		{
			name:      "certificate for another key",
			configure: func(f *fakeEJBCA) { f.wrongKey = true },
			wantErr:   "a certificate for a different key",
		},
		{
			name:      "certificate without the requested UPN",
			configure: func(f *fakeEJBCA) { f.dropSAN = true },
			wantErr:   fmt.Sprintf(msgUPNMissing, "alice@example.com"),
		},
		{
			name:      "certificate without a common name",
			configure: func(f *fakeEJBCA) { f.issuedCommonName = new("") },
			wantErr:   msgCommonNameEmpty,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeEJBCA(t, tt.configure)
			config := f.config(t)
			if tt.modify != nil {
				tt.modify(t, f, &config)
			}
			cert, err := NewService().GetCertificate(t.Context(), config)
			require.ErrorContains(t, err, tt.wantErr)
			require.Nil(t, cert)
		})
	}

	t.Run("required fields", func(t *testing.T) {
		f := newFakeEJBCA(t, nil)
		for field, clear := range map[string]func(*fleet.EJBCA){
			"certificate authority name": func(c *fleet.EJBCA) { c.CertificateAuthorityName = "" },
			"certificate profile name":   func(c *fleet.EJBCA) { c.CertificateProfileName = "" },
			"end entity profile name":    func(c *fleet.EJBCA) { c.EndEntityProfileName = "" },
			"common name":                func(c *fleet.EJBCA) { c.CertificateCommonName = "" },
			"username":                   func(c *fleet.EJBCA) { c.UsernameTemplate = "" },
		} {
			config := f.config(t)
			clear(&config)
			_, err := NewService().GetCertificate(t.Context(), config)
			require.ErrorContains(t, err, "EJBCA "+field+" is empty")
		}
		require.Zero(t, f.requestCount())
	})
}

// Each operation has its own client, so it has to close its connection or bulk enrollment piles up idle ones.
func TestConnectionsClosed(t *testing.T) {
	t.Parallel()
	f := newFakeEJBCA(t, nil)
	config := f.config(t)
	require.NoError(t, NewService().VerifyConnection(t.Context(), config))
	_, err := NewService().GetCertificate(t.Context(), config)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return f.openConns() == 0 }, 5*time.Second, 10*time.Millisecond)
}

// EJBCA can't send Fleet's client certificate or an enrollment request to another host with a redirect.
func TestRedirectsNotFollowed(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	t.Cleanup(target.Close)
	f := newFakeEJBCA(t, func(f *fakeEJBCA) { f.redirectTo = target.URL })
	config := f.config(t)

	err := NewService().VerifyConnection(t.Context(), config)
	require.ErrorContains(t, err, fmt.Sprintf(msgUnexpected, http.StatusTemporaryRedirect, ""))
	_, err = NewService().GetCertificate(t.Context(), config)
	require.ErrorContains(t, err, fmt.Sprintf(msgUnexpected, http.StatusTemporaryRedirect, ""))
	require.Zero(t, hits.Load())
}

// Running out of time or being canceled while looking up the certificate profile is not that lookup being unavailable.
func TestVerifyConnectionCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	f := newFakeEJBCA(t, func(f *fakeEJBCA) {
		f.onRequest = func(r *http.Request) {
			if strings.HasPrefix(r.URL.Path, restAPIPrefix+"/v2/") {
				cancel()
				// Answer only after the client has given up, or the response can win the race.
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
				}
			}
		}
	})
	require.ErrorContains(t, NewService().VerifyConnection(ctx, f.config(t)), "canceled")
}

// A short serial number is padded to the length of the serial column.
func TestGetCertificatePadsSerial(t *testing.T) {
	t.Parallel()
	f := newFakeEJBCA(t, func(f *fakeEJBCA) { f.serial = big.NewInt(0xABC) })
	cert, err := NewService().GetCertificate(t.Context(), f.config(t))
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("0", 37)+"abc", cert.SerialNumber)
}

// An end entity profile can set the common name itself, for example from the username.
func TestGetCertificateAcceptsOtherCommonName(t *testing.T) {
	t.Parallel()
	f := newFakeEJBCA(t, func(f *fakeEJBCA) { f.issuedCommonName = new("FLEET-USER-1") })
	cert, err := NewService().GetCertificate(t.Context(), f.config(t))
	require.NoError(t, err)
	_, issued, err := pkcs12.Decode(cert.PfxData, cert.Password)
	require.NoError(t, err)
	assert.Equal(t, "FLEET-USER-1", issued.Subject.CommonName)
}
