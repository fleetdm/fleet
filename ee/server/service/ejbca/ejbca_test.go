package ejbca

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/certissue"
	"github.com/fleetdm/fleet/v4/ee/server/service/ejbca/ejbcatest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

func TestVerifyConnection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		configure func(*ejbcatest.Server)
		modify    func(t *testing.T, f *ejbcatest.Server, config *fleet.EJBCA)
		wantErr   string
		// wantNoRequests is for failures that must not contact EJBCA.
		wantNoRequests bool
	}{
		// It lists the CAs, which needs authentication, and doesn't rely on the status endpoint, which doesn't.
		{name: "valid"},
		{
			name:      "unknown certificate profile",
			configure: func(f *ejbcatest.Server) { f.CertProfiles = []string{"other"} },
			wantErr:   fmt.Sprintf(msgProfileNotFound, ejbcatest.CertProfile),
		},
		{
			name:      "certificate profile lookup disabled",
			configure: func(f *ejbcatest.Server) { f.CertProfiles, f.V2Disabled = []string{"other"}, true },
		},
		{
			name:      "CA the certificate can't use",
			configure: func(f *ejbcatest.Server) { f.CAs = []string{"SubCA", "AnotherCA"} },
			wantErr:   fmt.Sprintf(msgCANotFound, ejbcatest.CAName, "AnotherCA, SubCA"),
		},
		{name: "no CAs", configure: func(f *ejbcatest.Server) { f.CAs = nil }, wantErr: msgNoCAs},
		{
			name:      "certificate without an EJBCA role",
			configure: func(f *ejbcatest.Server) { f.Unauthorized = true },
			wantErr:   fmt.Sprintf(msgNotAuthorized, "Administrator is not authorized to resource /administrator"),
		},
		{name: "protocol disabled", configure: func(f *ejbcatest.Server) { f.ProtocolDisabled = true }, wantErr: msgProtocolDisabled},
		{
			name:    "URL with the /ejbca path of the admin UI",
			modify:  func(_ *testing.T, f *ejbcatest.Server, config *fleet.EJBCA) { config.URL = f.HTTP.URL + "/ejbca/" },
			wantErr: msgNotFound,
		},
		{
			name:    "server certificate not trusted",
			modify:  func(_ *testing.T, _ *ejbcatest.Server, config *fleet.EJBCA) { config.TrustCABundlePEM = "" },
			wantErr: strings.Split(msgServerCertUntrusted, "%s")[0],
		},
		{
			name: "client certificate from a CA EJBCA doesn't trust",
			modify: func(t *testing.T, _ *ejbcatest.Server, config *fleet.EJBCA) {
				config.MTLSCertPEM, config.MTLSKeyPEM = ejbcatest.NewCA(t, "Other CA").IssueClient(t, time.Now().Add(time.Hour))
			},
			wantErr: strings.Split(msgClientCertRejected, "%s")[0],
		},
		{
			name: "client certificate expired",
			modify: func(t *testing.T, f *ejbcatest.Server, config *fleet.EJBCA) {
				notAfter := time.Now().Add(-time.Hour)
				config.MTLSCertPEM, config.MTLSKeyPEM = f.ClientCA.IssueClient(t, notAfter)
			},
			wantErr:        strings.Split(msgCertExpired, "%s")[0],
			wantNoRequests: true,
		},
		{
			name: "trust bundle without certificates",
			modify: func(_ *testing.T, _ *ejbcatest.Server, config *fleet.EJBCA) {
				config.TrustCABundlePEM = "not a certificate"
			},
			wantErr: msgNoTrustCerts,
		},
		{
			name:    "server not running",
			modify:  func(_ *testing.T, f *ejbcatest.Server, _ *fleet.EJBCA) { f.HTTP.Close() },
			wantErr: strings.Split(msgUnreachable, "%s")[0],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ejbcatest.New(t, tt.configure)
			config := f.Config(t)
			if tt.modify != nil {
				tt.modify(t, f, &config)
			}

			err := NewService().VerifyConnection(t.Context(), config)
			if tt.wantErr == "" {
				require.NoError(t, err)
				requests, _ := f.Snapshot()
				require.Equal(t, []string{
					"GET " + restAPIPrefix + "/v1/ca",
					"GET " + restAPIPrefix + "/v2/certificate/profile/" + ejbcatest.CertProfile,
				}, requests)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
			if tt.wantNoRequests {
				require.Zero(t, f.RequestCount())
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
	f := ejbcatest.New(t, func(f *ejbcatest.Server) {
		f.OnRequest = func(r *http.Request) {
			if r.URL.RawQuery != "" {
				queries.Add(1)
			}
		}
	})
	config := f.Config(t)
	config.URL += "?tenant=prod#ca"
	require.NoError(t, NewService().VerifyConnection(t.Context(), config))
	require.NotZero(t, f.RequestCount())
	require.Zero(t, queries.Load())
}

func TestGetCertificate(t *testing.T) {
	t.Parallel()
	f := ejbcatest.New(t, nil)
	config := f.Config(t)
	config.CertificateUserPrincipalNames = []string{"alice@example.com", " ", "alice@corp.example.com"}
	s := NewService()

	cert, err := s.GetCertificate(t.Context(), config)
	require.NoError(t, err)

	// What EJBCA was asked for. The CSR common name and the username are separate.
	_, enrolls := f.Snapshot()
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
	_, enrolls = f.Snapshot()
	assert.NotEqual(t, enrolls[0]["password"], enrolls[1]["password"])
	assert.NotEqual(t, cert.Password, second.Password)
}

func TestGetCertificateErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		configure func(*ejbcatest.Server)
		modify    func(t *testing.T, f *ejbcatest.Server, config *fleet.EJBCA)
		wantErr   string
		// wantTransient is whether the error is one that is expected to clear on its own, so the enrollment is worth retrying.
		wantTransient bool
	}{
		{
			name: "unknown CA",
			modify: func(_ *testing.T, _ *ejbcatest.Server, config *fleet.EJBCA) {
				config.CertificateAuthorityName = "Missing"
			},
			wantErr: fmt.Sprintf(msgEnrollRejected, `CA with name "Missing" doesn't exist`),
		},
		{
			name: "unknown certificate profile",
			modify: func(_ *testing.T, _ *ejbcatest.Server, config *fleet.EJBCA) {
				config.CertificateProfileName = "Missing"
			},
			wantErr: fmt.Sprintf(msgEnrollRejected, `Certificate profile with name "Missing" doesn't exist`),
		},
		{
			name:    "unknown end entity profile",
			modify:  func(_ *testing.T, _ *ejbcatest.Server, config *fleet.EJBCA) { config.EndEntityProfileName = "Missing" },
			wantErr: fmt.Sprintf(msgEnrollRejected, `End Entity Profile of name "Missing" was not found`),
		},
		{
			name:    "URL with the /ejbca path of the admin UI",
			modify:  func(_ *testing.T, f *ejbcatest.Server, config *fleet.EJBCA) { config.URL = f.HTTP.URL + "/ejbca" },
			wantErr: msgNotFound,
		},
		{
			name:      "certificate with a serial number that doesn't fit",
			configure: func(f *ejbcatest.Server) { f.Serial = new(big.Int).Lsh(big.NewInt(1), 167) },
			wantErr:   strings.Split(msgSerialTooLong, "%d")[0],
		},
		{
			name:      "EJBCA server error",
			configure: func(f *ejbcatest.Server) { f.EnrollStatus = http.StatusInternalServerError },
			wantErr:   fmt.Sprintf(msgUnexpected, http.StatusInternalServerError, "General failure"), wantTransient: true,
		},
		{
			name:    "EJBCA that is not running",
			modify:  func(_ *testing.T, f *ejbcatest.Server, _ *fleet.EJBCA) { f.HTTP.Close() },
			wantErr: strings.Split(msgUnreachable, "%s")[0], wantTransient: true,
		},
		{
			name:      "certificate for another key",
			configure: func(f *ejbcatest.Server) { f.WrongKey = true },
			wantErr:   "a certificate for a different key",
		},
		{
			name:      "certificate without the requested UPN",
			configure: func(f *ejbcatest.Server) { f.DropSAN = true },
			wantErr:   fmt.Sprintf(msgUPNMissing, "alice@example.com"),
		},
		{
			name:      "certificate without a common name",
			configure: func(f *ejbcatest.Server) { f.IssuedCommonName = new("") },
			wantErr:   msgCommonNameEmpty,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ejbcatest.New(t, tt.configure)
			config := f.Config(t)
			if tt.modify != nil {
				tt.modify(t, f, &config)
			}
			cert, err := NewService().GetCertificate(t.Context(), config)
			require.ErrorContains(t, err, tt.wantErr)
			require.Nil(t, cert)
			_, isTransient := errors.AsType[fleet.CertificateAuthorityTransientError](err)
			require.Equal(t, tt.wantTransient, isTransient)
		})
	}

	t.Run("required fields", func(t *testing.T) {
		f := ejbcatest.New(t, nil)
		for field, clear := range map[string]func(*fleet.EJBCA){
			"certificate authority name": func(c *fleet.EJBCA) { c.CertificateAuthorityName = "" },
			"certificate profile name":   func(c *fleet.EJBCA) { c.CertificateProfileName = "" },
			"end entity profile name":    func(c *fleet.EJBCA) { c.EndEntityProfileName = "" },
			"common name":                func(c *fleet.EJBCA) { c.CertificateCommonName = "" },
			"username":                   func(c *fleet.EJBCA) { c.UsernameTemplate = "" },
		} {
			config := f.Config(t)
			clear(&config)
			_, err := NewService().GetCertificate(t.Context(), config)
			require.ErrorContains(t, err, "EJBCA "+field+" is empty")
		}
		require.Zero(t, f.RequestCount())
	})
}

// Each operation has its own client, so it has to close its connection or bulk enrollment piles up idle ones.
func TestConnectionsClosed(t *testing.T) {
	t.Parallel()
	f := ejbcatest.New(t, nil)
	config := f.Config(t)
	require.NoError(t, NewService().VerifyConnection(t.Context(), config))
	_, err := NewService().GetCertificate(t.Context(), config)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return f.OpenConns() == 0 }, 5*time.Second, 10*time.Millisecond)
}

// EJBCA can't send Fleet's client certificate or an enrollment request to another host with a redirect.
func TestRedirectsNotFollowed(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	t.Cleanup(target.Close)
	f := ejbcatest.New(t, func(f *ejbcatest.Server) { f.RedirectTo = target.URL })
	config := f.Config(t)

	err := NewService().VerifyConnection(t.Context(), config)
	require.ErrorContains(t, err, fmt.Sprintf(msgUnexpected, http.StatusTemporaryRedirect, ""))
	_, err = NewService().GetCertificate(t.Context(), config)
	require.ErrorContains(t, err, fmt.Sprintf(msgUnexpected, http.StatusTemporaryRedirect, ""))
	require.Zero(t, hits.Load())
}

func TestErrorMessage(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, body, want string }{
		{"EJBCA error", `{"error_code":400,"error_message":"CA doesn't exist"}`, "CA doesn't exist"},
		{"plain text", "  Gateway timeout  ", "Gateway timeout"},
		{
			"HTML page of a proxy",
			"<!DOCTYPE HTML>\n<html><head>\n<title>503 Service Unavailable</title>\n</head><body>\n<h1>Service Unavailable</h1>\n</body></html>",
			"503 Service Unavailable Service Unavailable",
		},
		{"long text is cut", strings.Repeat("a", 300), strings.Repeat("a", 200)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, errorMessage([]byte(tt.body)))
		})
	}
}

// Running out of time or being canceled while looking up the certificate profile is not that lookup being unavailable.
func TestVerifyConnectionCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	f := ejbcatest.New(t, func(f *ejbcatest.Server) {
		f.OnRequest = func(r *http.Request) {
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
	require.ErrorContains(t, NewService().VerifyConnection(ctx, f.Config(t)), "canceled")
}

// A short serial number is padded to the length of the serial column.
func TestGetCertificatePadsSerial(t *testing.T) {
	t.Parallel()
	f := ejbcatest.New(t, func(f *ejbcatest.Server) { f.Serial = big.NewInt(0xABC) })
	cert, err := NewService().GetCertificate(t.Context(), f.Config(t))
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("0", 37)+"abc", cert.SerialNumber)
}

// An end entity profile can set the common name itself, for example from the username.
func TestGetCertificateAcceptsOtherCommonName(t *testing.T) {
	t.Parallel()
	f := ejbcatest.New(t, func(f *ejbcatest.Server) { f.IssuedCommonName = new("FLEET-USER-1") })
	cert, err := NewService().GetCertificate(t.Context(), f.Config(t))
	require.NoError(t, err)
	_, issued, err := pkcs12.Decode(cert.PfxData, cert.Password)
	require.NoError(t, err)
	assert.Equal(t, "FLEET-USER-1", issued.Subject.CommonName)
}
