package condaccess

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// Test certificates used across multiple tests
var (
	testCertPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIDKzCCAhOgAwIBAgIUPnw4bYCIKmtfincrNjFPtbMpiUUwDQYJKoZIhvcNAQEL
BQAwJTEjMCEGA1UEAwwadGVzdC1pZHAuZmxlZXQuZXhhbXBsZS5jb20wHhcNMjUx
MTA0MDAyNDUwWhcNMjYxMTA0MDAyNDUwWjAlMSMwIQYDVQQDDBp0ZXN0LWlkcC5m
bGVldC5leGFtcGxlLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AIw2AGrO/ORUiWUVwMjgoAKQvAhoAaEYWw21Zv238ELUCDAYEivg2FjgE7irzgvn
mcO4EH2vXwdcfOsd9/5wVl/za+ejIjTrFG/NjxZe66PF+WoQGx0/mONUZy9A1jx0
9rOVOrM4XIjU2AhUDWADDJZnrFLfAOYjmkjMxjA0deLaMyenl5SP0ta9x4gMAKTO
hEW10ofw/ByiVNWprctsvfAlUrXcqzyzgfPZ6niNEdyknxGr7JqVzI1H/6F3dGMY
s1H4iwk0Mbrv3jbxva3EzElvhHqvSq5yxlHuuom2CtGEC76YzsYcoDYBfcc9C0Er
6EijfhZHmjXkyLNxicktO7kCAwEAAaNTMFEwHQYDVR0OBBYEFDfEC9al2Qpx+QJg
3Lf8bR3PSUxPMB8GA1UdIwQYMBaAFDfEC9al2Qpx+QJg3Lf8bR3PSUxPMA8GA1Ud
EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAIc72uzfpcDv7Y6xD/9Tfpty
Pjk2v2WfkQlezVah4bEZqtSPs1jrJiIrM/N0Bx3Lja0xMRVDxfaDRZubaTLBOQZV
I67n/dpOXOkQE1tFkQiT5wzqtinIxCXZ6qFy3vG0Fg0g51UHkXOL1oNr7/5ylEAc
ws2RdnfUCvnssTOrm3mxykaxF4V6YfQ4m1CT067orQa4K/cjUwDtZeB+EfiQmAdr
vHzA6x70Z46zQUJ4vUolJIGZ0sJGHbJITMXMYqH/uVJtWhZ+ysVpcLtIN8es+Fcs
ZiaQxA4W43jZdy0XykSmsyVCrkmz4wbJ6y0soTft3/ohPK/d8+KBrlMB3px70lo=
-----END CERTIFICATE-----`)

	testKeyPEM = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAupRgzsVGpWkK3F3lz1+S440CECLFFud7GdsK1cO9wM1iKy0e
AhSA2JiEB6Yv0Cj5HN/Duy/itzaTi4Nj2zCUQpVy9vUraT5HZ+xOwV2DJVj6D0OG
dUfkqXsfZuZSnEdDTnmheAhm+KWhQo1e5ZJumbZ44oMeo1JuHNOf5h6e4wcdP1TU
2uuE6BPy5Bsi50GPo6UyihAnXXbqctLxHplYyNhgtsikEhXXwbXGWd/GdeT2eHVr
1STf2lpcD2SbSaK3USNElHNNflu1x4Ixkz2zA6WMgh4zbgjbCB4Zk8vsFFZSWwtZ
XTqL4YCSVQKlDmlShj+O44kKZEh+xIAfU5xizQIDAQABAoIBAAf59WyJiwRhyfzT
R9KWfadcPSEW93GL+luH3X33hQpzzVVWs7B3k22PEZ/pFySxR7sYBtxfBvR5sROX
DaMOf9wb2wMbRpyUdMWI2PITzxo+5EvYQWyMowYq1RQXVyNGuaYmdYR16X8KR6ta
c1rhqHhKUH8wh1QIn1v8oRqbpwPCFJkZOpZf/44e2e6TMeVKjsKeYFiPT9Q6Ue/k
M7yQc3/CFB0//slI9ZfPV5JeyLTAXhbM8SeTGEA52GIOdWFz11LRyNDWOPUPK4kB
ThQEvgfHHkYY06kiBUKibA8uiIqAjIeQ6hJFeV5u5wXHLX2NhMPDA1IQ30ia4JnF
+Ngki7ECgYEA7f3xt6nPKJKzTwYlOlznGDmWrVUQe3qtGE5DNMpU2uU1dZ23pCh7
Ip+Up11zVLJ1EDC5ggKoZtfbz2ShlKNoa42cNZNig9g6VBzfovlIjknHhqavvIq3
eneIUwCBwRHfsQ8WK1eQG1xS6G8np6NyveDIj1Zb+ZOe4zbDG6TMi90CgYEAyLKL
L69g2ZfKTnEC6Jw4hGsp3dIBk9oAyeCRgXYsC0pO2G2ZDGeg+kbjtRcGoHN1H02x
dJdSwTWnUGTJJ1KgGq7A6EBnxPC/FHzCHka8Df7cq6GJ2jCIdgu223eSpqHIv009
96l3wuCSG5faAYgEvK/myKFIcffHFDgAnldm+7ECgYAVGTg+og09eZPv44mVXPsX
yLM09p+Zcsy5pOaMXYucREmy/aJ0KSqRbThOhhhdX9zE7Kzle7rWMzjHcBJrDPmK
32kDzuci7R5uqoig+ByYkK3hoBFgU6PkdYheY2MdbKo6Fi5O9VpPMqYe+Qu47uKT
NsRRAMTyoUWquwYdA0Um+QKBgD5xxvq8P48UOl7zrKsBSFhzG2CoIdOF5e7qD3vP
b97HbQbL+u2wJJcajWjf1DECG3P08XzMRHRXJErQQQIaJDSJIP5iY6cUHO/b7W4M
JiDYpoJETab0qNDJzkg0yQ1Nky9qchhnwxqAUxWAxtTpJEgtFspf3DGRnYB9+DtM
CH/RAoGAMXkTVX5MuMCOVjgoZMSdjszkl0P3AJJ3P7zVwk1hpa6CPoLzg8HCtQkg
7RuA0Ls438qdFtuBDAFVDhT4wMU7hRX/NPM0vgYB1HREAg+TunyMA1y01uecSXpH
b1ctZeF7HaWwFdTC8GqWI6zzRFn+YA3f/yYibhowuEypPQeSjlI=
-----END RSA PRIVATE KEY-----`)
)

// Helper functions for test setup

func newTestService() (*idpService, *mock.Store) {
	ds := new(mock.Store)
	logger := kitlog.NewNopLogger()
	return &idpService{ds: ds, logger: logger}, ds
}

func mockAppConfigFunc(serverURL string) func(context.Context) (*fleet.AppConfig, error) {
	return func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: serverURL,
			},
			ConditionalAccess: &fleet.ConditionalAccessSettings{},
		}, nil
	}
}

func mockCertAssetsFunc(includeCerts bool) func(context.Context, []fleet.MDMAssetName, sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
	return func(ctx context.Context, assetNames []fleet.MDMAssetName, queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		if !includeCerts {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, nil
		}
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetConditionalAccessIDPCert: {
				Name:  fleet.MDMAssetConditionalAccessIDPCert,
				Value: testCertPEM,
			},
			fleet.MDMAssetConditionalAccessIDPKey: {
				Name:  fleet.MDMAssetConditionalAccessIDPKey,
				Value: testKeyPEM,
			},
		}, nil
	}
}

func TestRegisterIdP(t *testing.T) {
	ds := new(mock.Store)
	logger := kitlog.NewNopLogger()
	cfg := &config.FleetConfig{}

	ds.AppConfigFunc = mockAppConfigFunc("https://fleet.example.com")
	ds.GetAllMDMConfigAssetsByNameFunc = mockCertAssetsFunc(false)

	mux := http.NewServeMux()
	err := RegisterIdP(mux, ds, logger, cfg)
	require.NoError(t, err)

	t.Run("metadata endpoint registered", func(t *testing.T) {
		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("SSO endpoint registered", func(t *testing.T) {
		req := httptest.NewRequest("POST", idpSSOPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusSeeOther, w.Code)
		require.Equal(t, "https://fleetdm.com/okta-conditional-access-error", w.Header().Get("Location"))
	})
}

func TestRegisterIdP_NilConfig(t *testing.T) {
	ds := new(mock.Store)
	logger := kitlog.NewNopLogger()
	mux := http.NewServeMux()

	err := RegisterIdP(mux, ds, logger, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fleet config is nil")
}

func TestServeMetadata(t *testing.T) {
	t.Run("returns SAML metadata when configured", func(t *testing.T) {
		svc, ds := newTestService()
		ds.AppConfigFunc = mockAppConfigFunc("https://fleet.example.com")
		ds.GetAllMDMConfigAssetsByNameFunc = mockCertAssetsFunc(true)

		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()

		svc.serveMetadata(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "application/samlmetadata+xml", w.Header().Get("Content-Type"))

		body := w.Body.String()
		require.Contains(t, body, "EntityDescriptor")
		require.Contains(t, body, idpSSOPath)
		require.Contains(t, body, "IDPSSODescriptor")
		require.Contains(t, body, "okta.fleet.example.com")
	})

	t.Run("properly appends paths to server URL with existing path", func(t *testing.T) {
		svc, ds := newTestService()
		ds.AppConfigFunc = mockAppConfigFunc("https://fleet.example.com/go/here/for/fleet")
		ds.GetAllMDMConfigAssetsByNameFunc = mockCertAssetsFunc(true)

		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()

		svc.serveMetadata(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		require.Contains(t, body, "https://fleet.example.com/go/here/for/fleet"+idpMetadataPath)
		require.Contains(t, body, "https://okta.fleet.example.com/go/here/for/fleet"+idpSSOPath)
	})

	t.Run("returns error when certificate assets not found", func(t *testing.T) {
		svc, ds := newTestService()
		ds.AppConfigFunc = mockAppConfigFunc("https://fleet.example.com")
		ds.GetAllMDMConfigAssetsByNameFunc = mockCertAssetsFunc(false)

		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()

		svc.serveMetadata(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns error when server URL not configured", func(t *testing.T) {
		svc, ds := newTestService()
		ds.AppConfigFunc = mockAppConfigFunc("")

		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()

		svc.serveMetadata(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Contains(t, w.Body.String(), "Server URL not configured")
	})
}

func TestParseSerialNumber(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  uint64
		expectErr bool
	}{
		{
			name:     "simple hex number",
			input:    "1A2B3C",
			expected: 0x1A2B3C,
		},
		{
			name:     "hex with colons",
			input:    "1A:2B:3C",
			expected: 0x1A2B3C,
		},
		{
			name:     "hex with spaces",
			input:    "1A 2B 3C",
			expected: 0x1A2B3C,
		},
		{
			name:     "mixed colons and spaces",
			input:    "1A:2B 3C",
			expected: 0x1A2B3C,
		},
		{
			name:     "large serial number",
			input:    "DEADBEEF12345678",
			expected: 0xDEADBEEF12345678,
		},
		{
			name:     "lowercase hex",
			input:    "abcdef123456",
			expected: 0xABCDEF123456,
		},
		{
			name:      "invalid hex characters",
			input:     "GHIJKL",
			expectErr: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
		{
			name:      "overflow uint64",
			input:     "FFFFFFFFFFFFFFFF1",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSerialNumber(tt.input)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestServeSSO(t *testing.T) {
	t.Run("missing certificate serial header", func(t *testing.T) {
		svc, _ := newTestService()

		req := httptest.NewRequest("POST", idpSSOPath, nil)
		w := httptest.NewRecorder()

		svc.serveSSO(w, req)

		require.Equal(t, http.StatusSeeOther, w.Code)
		require.Equal(t, "https://fleetdm.com/okta-conditional-access-error", w.Header().Get("Location"))
	})

	t.Run("invalid certificate serial format", func(t *testing.T) {
		svc, _ := newTestService()

		req := httptest.NewRequest("POST", idpSSOPath, nil)
		req.Header.Set("X-Client-Cert-Serial", "INVALID_HEX")
		w := httptest.NewRecorder()

		svc.serveSSO(w, req)

		require.Equal(t, http.StatusSeeOther, w.Code)
		require.Equal(t, "https://fleetdm.com/okta-conditional-access-error", w.Header().Get("Location"))
	})

	t.Run("certificate not found in database", func(t *testing.T) {
		svc, ds := newTestService()

		ds.GetConditionalAccessCertHostIDBySerialNumberFunc = func(ctx context.Context, serial uint64) (uint, error) {
			return 0, common_mysql.NotFound("certificate")
		}

		req := httptest.NewRequest("POST", idpSSOPath, nil)
		req.Header.Set("X-Client-Cert-Serial", "DEADBEEF")
		w := httptest.NewRecorder()

		svc.serveSSO(w, req)

		require.Equal(t, http.StatusSeeOther, w.Code)
		require.Equal(t, "https://fleetdm.com/okta-conditional-access-error", w.Header().Get("Location"))
		require.True(t, ds.GetConditionalAccessCertHostIDBySerialNumberFuncInvoked)
	})

	t.Run("valid certificate with different serial formats", func(t *testing.T) {
		tests := []struct {
			name   string
			serial string
		}{
			{"plain hex", "DEADBEEF"},
			{"hex with colons", "DE:AD:BE:EF"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, ds := newTestService()

				ds.GetConditionalAccessCertHostIDBySerialNumberFunc = func(ctx context.Context, serial uint64) (uint, error) {
					require.Equal(t, uint64(0xDEADBEEF), serial)
					return 123, nil
				}
				ds.AppConfigFunc = mockAppConfigFunc("https://fleet.example.com")
				ds.GetAllMDMConfigAssetsByNameFunc = mockCertAssetsFunc(false)

				req := httptest.NewRequest("POST", idpSSOPath, nil)
				req.Header.Set("X-Client-Cert-Serial", tt.serial)
				w := httptest.NewRecorder()

				svc.serveSSO(w, req)

				require.Equal(t, http.StatusSeeOther, w.Code)
				require.True(t, ds.GetConditionalAccessCertHostIDBySerialNumberFuncInvoked)
				require.True(t, ds.AppConfigFuncInvoked)
			})
		}
	})

	t.Run("infrastructure errors return 500", func(t *testing.T) {
		tests := []struct {
			name      string
			setupMock func(*mock.Store)
		}{
			{
				name: "certificate lookup error",
				setupMock: func(ds *mock.Store) {
					ds.GetConditionalAccessCertHostIDBySerialNumberFunc = func(ctx context.Context, serial uint64) (uint, error) {
						return 0, errors.New("database connection failed")
					}
				},
			},
			{
				name: "AppConfig load error",
				setupMock: func(ds *mock.Store) {
					ds.GetConditionalAccessCertHostIDBySerialNumberFunc = func(ctx context.Context, serial uint64) (uint, error) {
						return 123, nil
					}
					ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
						return nil, errors.New("database connection failed")
					}
				},
			},
			{
				name: "server URL not configured",
				setupMock: func(ds *mock.Store) {
					ds.GetConditionalAccessCertHostIDBySerialNumberFunc = func(ctx context.Context, serial uint64) (uint, error) {
						return 123, nil
					}
					ds.AppConfigFunc = mockAppConfigFunc("")
				},
			},
			{
				name: "IdP build error",
				setupMock: func(ds *mock.Store) {
					ds.GetConditionalAccessCertHostIDBySerialNumberFunc = func(ctx context.Context, serial uint64) (uint, error) {
						return 123, nil
					}
					ds.AppConfigFunc = mockAppConfigFunc("https://fleet.example.com")
					ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
						return nil, errors.New("database connection failed")
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc, ds := newTestService()
				tt.setupMock(ds)

				req := httptest.NewRequest("POST", idpSSOPath, nil)
				req.Header.Set("X-Client-Cert-Serial", "DEADBEEF")
				w := httptest.NewRecorder()

				svc.serveSSO(w, req)

				require.Equal(t, http.StatusInternalServerError, w.Code)
			})
		}
	})
}

func TestBuildSSOServerURL(t *testing.T) {
	t.Run("uses dev override when FLEET_DEV_OKTA_SSO_SERVER_URL is set", func(t *testing.T) {
		t.Setenv("FLEET_DEV_OKTA_SSO_SERVER_URL", "https://dev.example.com")

		result, err := buildSSOServerURL("https://fleet.example.com")
		require.NoError(t, err)
		require.Equal(t, "https://dev.example.com", result)
	})

	t.Run("prepends okta subdomain to hostname", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "simple hostname",
				input:    "https://bozo.example.com",
				expected: "https://okta.bozo.example.com",
			},
			{
				name:     "hostname with port",
				input:    "https://bozo.example.com:8080",
				expected: "https://okta.bozo.example.com:8080",
			},
			{
				name:     "hostname with path",
				input:    "https://bozo.example.com/path",
				expected: "https://okta.bozo.example.com/path",
			},
			{
				name:     "hostname with port and path",
				input:    "https://bozo.example.com:8080/path",
				expected: "https://okta.bozo.example.com:8080/path",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := buildSSOServerURL(tt.input)
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		_, err := buildSSOServerURL("://invalid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse server URL")
	})
}

func TestParseCertAndKeyBytes(t *testing.T) {
	t.Run("parses valid certificate and key", func(t *testing.T) {
		cert, key, err := parseCertAndKeyBytes(testCertPEM, testKeyPEM)
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.NotNil(t, key)
		require.Equal(t, "test-idp.fleet.example.com", cert.Subject.CommonName)
	})

	t.Run("returns error when certificate is invalid", func(t *testing.T) {
		invalidCert := []byte("not a certificate")
		_, _, err := parseCertAndKeyBytes(invalidCert, testKeyPEM)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode certificate PEM")
	})

	t.Run("returns error when private key is invalid", func(t *testing.T) {
		invalidKey := []byte("not a private key")
		_, _, err := parseCertAndKeyBytes(testCertPEM, invalidKey)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode RSA private key PEM")
	})
}

func TestDeviceHealthSessionProvider(t *testing.T) {
	logger := kitlog.NewNopLogger()

	t.Run("returns session for compliant device", func(t *testing.T) {
		ds := new(mock.Store)

		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			require.Equal(t, uint(123), hostID)
			teamID := uint(1)
			return &fleet.Host{ID: 123, TeamID: &teamID}, nil
		}

		ds.GetPoliciesForConditionalAccessFunc = func(ctx context.Context, teamID uint) ([]uint, error) {
			require.Equal(t, uint(1), teamID)
			return []uint{10, 20}, nil
		}

		ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
			return []*fleet.HostPolicy{
				{PolicyData: fleet.PolicyData{ID: 10}, Response: "pass"},
				{PolicyData: fleet.PolicyData{ID: 20}, Response: "pass"},
				{PolicyData: fleet.PolicyData{ID: 30}, Response: "fail"}, // Not conditional access
			}, nil
		}

		provider := &deviceHealthSessionProvider{ds: ds, logger: logger, hostID: 123}

		req := httptest.NewRequest("POST", idpSSOPath, nil)
		w := httptest.NewRecorder()

		// Pass nil for SAML request to test fallback behavior
		session := provider.GetSession(w, req, nil)

		require.NotNil(t, session)
		// When no NameID is provided in the SAML request, should fall back to host-based identifier
		require.Equal(t, "host-123", session.NameID)
	})

	t.Run("uses NameID from SAML request when provided", func(t *testing.T) {
		ds := new(mock.Store)

		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			require.Equal(t, uint(123), hostID)
			teamID := uint(1)
			return &fleet.Host{ID: 123, TeamID: &teamID, Platform: "darwin"}, nil
		}

		ds.GetPoliciesForConditionalAccessFunc = func(ctx context.Context, teamID uint) ([]uint, error) {
			require.Equal(t, uint(1), teamID)
			return []uint{10, 20}, nil
		}

		ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
			return []*fleet.HostPolicy{
				{PolicyData: fleet.PolicyData{ID: 10}, Response: "pass"},
				{PolicyData: fleet.PolicyData{ID: 20}, Response: "pass"},
			}, nil
		}

		provider := &deviceHealthSessionProvider{ds: ds, logger: logger, hostID: 123}

		req := httptest.NewRequest("POST", idpSSOPath, nil)
		w := httptest.NewRecorder()

		// Create a SAML request with a NameID (simulating what Okta sends)
		samlReq := &saml.IdpAuthnRequest{
			Request: saml.AuthnRequest{
				Subject: &saml.Subject{
					NameID: &saml.NameID{
						Value: "user@example.com",
					},
				},
			},
		}

		session := provider.GetSession(w, req, samlReq)

		require.NotNil(t, session)
		// Should use the NameID from the SAML request (what Okta sent)
		require.Equal(t, "user@example.com", session.NameID)
	})

	t.Run("redirects to remediate for failing conditional access policies", func(t *testing.T) {
		ds := new(mock.Store)

		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			teamID := uint(1)
			return &fleet.Host{ID: 456, TeamID: &teamID}, nil
		}

		ds.GetPoliciesForConditionalAccessFunc = func(ctx context.Context, teamID uint) ([]uint, error) {
			return []uint{10, 20}, nil
		}

		ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
			return []*fleet.HostPolicy{
				{PolicyData: fleet.PolicyData{ID: 10}, Response: "fail"},
				{PolicyData: fleet.PolicyData{ID: 20}, Response: "pass"},
				{PolicyData: fleet.PolicyData{ID: 30}, Response: "fail"}, // Not conditional access
			}, nil
		}

		provider := &deviceHealthSessionProvider{ds: ds, logger: logger, hostID: 456}

		req := httptest.NewRequest("POST", idpSSOPath, nil)
		w := httptest.NewRecorder()

		session := provider.GetSession(w, req, nil)

		require.Nil(t, session)
		require.Equal(t, http.StatusSeeOther, w.Code)
		require.Equal(t, "https://fleetdm.com/remediate", w.Header().Get("Location"))
	})

	t.Run("returns 500 when HostLite fails", func(t *testing.T) {
		ds := new(mock.Store)

		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return nil, errors.New("database error")
		}

		provider := &deviceHealthSessionProvider{ds: ds, logger: logger, hostID: 789}

		req := httptest.NewRequest("POST", idpSSOPath, nil)
		w := httptest.NewRecorder()

		session := provider.GetSession(w, req, nil)

		require.Nil(t, session)
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
