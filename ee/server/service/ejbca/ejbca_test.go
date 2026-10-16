package ejbca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json/v2"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/cryptobyte"
	cryptobyteasn1 "golang.org/x/crypto/cryptobyte/asn1"
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
	var err error
	if template.SerialNumber, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 159)); err != nil {
		return nil, err
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

	mu       sync.Mutex
	requests []string
	enrolls  []map[string]any
}

func newFakeEJBCA(t *testing.T, configure func(*fakeEJBCA)) *fakeEJBCA {
	f := &fakeEJBCA{
		clientCA:          newTestCA(t, "Client CA"),
		cas:               []string{testCAName},
		certProfiles:      []string{testCertProfile},
		endEntityProfiles: []string{testEndEntityProf},
	}
	if configure != nil {
		configure(f)
	}
	issuer := newTestCA(t, testCAName)

	pool := x509.NewCertPool()
	pool.AddCert(f.clientCA.cert)
	f.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.serve(t, issuer, w, r) }))
	f.srv.Config.ErrorLog = log.New(io.Discard, "", 0) // the handshake errors that the tests cause
	f.srv.TLS = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool, MinVersion: tls.VersionTLS12}
	f.srv.StartTLS()
	t.Cleanup(f.srv.Close)
	f.serverPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.srv.Certificate().Raw}))
	return f
}

func (f *fakeEJBCA) requestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeEJBCA) serve(t *testing.T, issuer *testCA, w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mu.Unlock()

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
		if ext.Id.Equal(oidSubjectAltName) {
			extensions = append(extensions, ext)
		}
	}
	cert, err := issuer.sign(&x509.Certificate{
		Subject:         csr.Subject,
		NotBefore:       time.Now().Add(-time.Minute),
		NotAfter:        time.Now().Add(365 * 24 * time.Hour),
		ExtraExtensions: extensions,
	}, pub)
	if err != nil {
		return 0, "", err
	}
	resp, err := json.Marshal(map[string]any{
		"certificate":     base64.StdEncoding.EncodeToString(cert.Raw),
		"serial_number":   strings.ToUpper(cert.SerialNumber.Text(16)),
		"response_format": "DER",
	})
	return http.StatusCreated, string(resp), err
}

// config is a configuration for the fake EJBCA with a valid client certificate.
func (f *fakeEJBCA) config(t *testing.T) fleet.EJBCACA {
	certPEM, keyPEM := f.clientCA.issueClient(t, time.Now().Add(24*time.Hour))
	return fleet.EJBCACA{
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
		modify    func(t *testing.T, f *fakeEJBCA, config *fleet.EJBCACA)
		wantErr   string
		// wantNoRequests is for failures that must not contact EJBCA.
		wantNoRequests bool
	}{
		// It lists the CAs, which needs authentication, and doesn't rely on the status endpoint, which doesn't.
		{name: "valid"},
		{name: "certificate profile lookup disabled", configure: func(f *fakeEJBCA) { f.v2Disabled = true }},
		{
			name:      "unknown certificate profile",
			configure: func(f *fakeEJBCA) { f.certProfiles = []string{"other"} },
			wantErr:   fmt.Sprintf(msgProfileNotFound, testCertProfile),
		},
		{
			name:      "unknown certificate profile, but the lookup is disabled",
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
			name:    "server certificate not trusted",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCACA) { config.TrustCABundlePEM = "" },
			wantErr: strings.Split(msgServerCertUntrusted, "%s")[0],
		},
		{
			name: "client certificate from a CA EJBCA doesn't trust",
			modify: func(t *testing.T, _ *fakeEJBCA, config *fleet.EJBCACA) {
				config.MTLSCertPEM, config.MTLSKeyPEM = newTestCA(t, "Other CA").issueClient(t, time.Now().Add(time.Hour))
			},
			wantErr: strings.Split(msgClientCertRejected, "%s")[0],
		},
		{
			name: "client certificate expired",
			modify: func(t *testing.T, f *fakeEJBCA, config *fleet.EJBCACA) {
				notAfter := time.Now().Add(-time.Hour)
				config.MTLSCertPEM, config.MTLSKeyPEM = f.clientCA.issueClient(t, notAfter)
			},
			wantErr:        strings.Split(msgCertExpired, "%s")[0],
			wantNoRequests: true,
		},
		{
			name: "URL without https",
			modify: func(_ *testing.T, f *fakeEJBCA, config *fleet.EJBCACA) {
				config.URL = strings.Replace(f.srv.URL, "https", "http", 1)
			},
			wantErr: msgInsecureURL,
		},
		{
			name:    "trust bundle without certificates",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCACA) { config.TrustCABundlePEM = "not a certificate" },
			wantErr: "no usable certificates",
		},
		{
			name:    "server not running",
			modify:  func(_ *testing.T, f *fakeEJBCA, _ *fleet.EJBCACA) { f.srv.Close() },
			wantErr: strings.Split(msgUnreachable, "%s")[0],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFakeEJBCA(t, tt.configure)
			config := f.config(t)
			if tt.modify != nil {
				tt.modify(t, f, &config)
			}

			err := NewService().VerifyConnection(t.Context(), config)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, []string{
					"GET " + restAPIPrefix + "/v1/ca",
					"GET " + restAPIPrefix + "/v2/certificate/profile/" + testCertProfile,
				}, f.requests)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
			if tt.wantNoRequests {
				require.Zero(t, f.requestCount())
			}
		})
	}
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
	require.Len(t, f.enrolls, 1)
	body := f.enrolls[0]
	assert.Equal(t, config.UsernameTemplate, body["username"])
	assert.Equal(t, config.CertificateAuthorityName, body["certificate_authority_name"])
	assert.Equal(t, config.CertificateProfileName, body["certificate_profile_name"])
	assert.Equal(t, config.EndEntityProfileName, body["end_entity_profile_name"])
	assert.Equal(t, "DER", body["response_format"])
	assert.Equal(t, false, body["include_chain"])
	assert.Len(t, body["password"], 64)

	// What Fleet got back: the key and the certificate EJBCA issued, in a bundle that the password opens.
	key, issued, err := pkcs12.Decode(cert.PfxData, cert.Password)
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", issued.Subject.CommonName)
	assert.True(t, key.(*rsa.PrivateKey).PublicKey.Equal(issued.PublicKey))
	assert.Equal(t, strings.ToUpper(issued.SerialNumber.Text(16)), cert.SerialNumber)
	assert.LessOrEqual(t, len(cert.SerialNumber), 40, "the serial has to fit the managed certificate record")
	assert.True(t, issued.NotBefore.Equal(cert.NotValidBefore))
	assert.True(t, issued.NotAfter.Equal(cert.NotValidAfter))
	assert.Equal(t, []string{"alice@example.com", "alice@corp.example.com"}, upnsInCertificate(t, issued))

	// The enrollment password is not reused, and each certificate gets its own key and password.
	second, err := s.GetCertificate(t.Context(), config)
	require.NoError(t, err)
	assert.NotEqual(t, f.enrolls[0]["password"], f.enrolls[1]["password"])
	assert.NotEqual(t, cert.Password, second.Password)
	assert.NotEqual(t, cert.PfxData, second.PfxData)
}

func TestGetCertificateErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		configure func(*fakeEJBCA)
		modify    func(t *testing.T, f *fakeEJBCA, config *fleet.EJBCACA)
		wantErr   string
	}{
		{
			name:    "unknown CA",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCACA) { config.CertificateAuthorityName = "Missing" },
			wantErr: fmt.Sprintf(msgEnrollRejected, `CA with name "Missing" doesn't exist`),
		},
		{
			name:    "unknown certificate profile",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCACA) { config.CertificateProfileName = "Missing" },
			wantErr: fmt.Sprintf(msgEnrollRejected, `Certificate profile with name "Missing" doesn't exist`),
		},
		{
			name:    "unknown end entity profile",
			modify:  func(_ *testing.T, _ *fakeEJBCA, config *fleet.EJBCACA) { config.EndEntityProfileName = "Missing" },
			wantErr: fmt.Sprintf(msgEnrollRejected, `End Entity Profile of name "Missing" was not found`),
		},
		{
			name:      "certificate without an EJBCA role",
			configure: func(f *fakeEJBCA) { f.unauthorized = true },
			wantErr:   fmt.Sprintf(msgNotAuthorized, "Administrator is not authorized to resource /administrator"),
		},
		{name: "protocol disabled", configure: func(f *fakeEJBCA) { f.protocolDisabled = true }, wantErr: msgProtocolDisabled},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

	t.Run("client certificate expired", func(t *testing.T) {
		t.Parallel()
		f := newFakeEJBCA(t, nil)
		config := f.config(t)
		config.MTLSCertPEM, config.MTLSKeyPEM = f.clientCA.issueClient(t, time.Now().Add(-time.Hour))
		_, err := NewService().GetCertificate(t.Context(), config)
		require.ErrorContains(t, err, strings.Split(msgCertExpired, "%s")[0])
		require.Zero(t, f.requestCount(), "EJBCA is not contacted with an expired client certificate")
	})

	t.Run("required fields", func(t *testing.T) {
		t.Parallel()
		f := newFakeEJBCA(t, nil)
		for field, clear := range map[string]func(*fleet.EJBCACA){
			"certificate authority name": func(c *fleet.EJBCACA) { c.CertificateAuthorityName = "" },
			"certificate profile name":   func(c *fleet.EJBCACA) { c.CertificateProfileName = "" },
			"end entity profile name":    func(c *fleet.EJBCACA) { c.EndEntityProfileName = "" },
			"common name":                func(c *fleet.EJBCACA) { c.CertificateCommonName = "" },
			"username":                   func(c *fleet.EJBCACA) { c.UsernameTemplate = "" },
		} {
			config := f.config(t)
			clear(&config)
			_, err := NewService().GetCertificate(t.Context(), config)
			require.ErrorContains(t, err, "EJBCA "+field+" is empty")
		}
		require.Zero(t, f.requestCount())
	})
}

func TestUPNExtension(t *testing.T) {
	// The encoding OpenSSL produces for these two UPNs, which EJBCA accepts:
	// openssl req -addext "subjectAltName=otherName:1.3.6.1.4.1.311.20.2.3;UTF8:user@example.com,otherName:1.3.6.1.4.1.311.20.2.3;UTF8:b@c.d"
	const openSSLEncoding = "3039a020060a2b060104018237140203a0120c1075736572406578616d706c652e636f6da015060a2b060104018237140203a0070c" +
		"056240632e64"

	ext, ok := upnExtension([]string{"user@example.com", "", "b@c.d"})
	require.True(t, ok)
	assert.True(t, ext.Id.Equal(oidSubjectAltName))
	assert.Equal(t, openSSLEncoding, hex.EncodeToString(ext.Value))

	_, ok = upnExtension([]string{"", " "})
	assert.False(t, ok)
}

// upnsInCertificate returns the UPNs in the subjectAltName of the certificate, read independently of how they are written.
func upnsInCertificate(t *testing.T, cert *x509.Certificate) []string {
	var upns []string
	for _, ext := range cert.Extensions {
		if !ext.Id.Equal(oidSubjectAltName) {
			continue
		}
		names := cryptobyte.String(ext.Value)
		require.True(t, names.ReadASN1(&names, cryptobyteasn1.SEQUENCE))
		for !names.Empty() {
			var name, oid, value, utf8 cryptobyte.String
			require.True(t, names.ReadASN1(&name, cryptobyteasn1.Tag(0).ContextSpecific().Constructed()))
			require.True(t, name.ReadASN1(&oid, cryptobyteasn1.OBJECT_IDENTIFIER))
			require.True(t, name.ReadASN1(&value, cryptobyteasn1.Tag(0).ContextSpecific().Constructed()))
			require.True(t, value.ReadASN1(&utf8, cryptobyteasn1.UTF8String))
			upns = append(upns, string(utf8))
		}
	}
	return upns
}
