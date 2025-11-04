package condaccess

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestRegisterIdP(t *testing.T) {
	ds := new(mock.Store)
	logger := kitlog.NewNopLogger()
	cfg := &config.FleetConfig{}

	// Mock AppConfig to return unconfigured Okta settings
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ConditionalAccess: &fleet.ConditionalAccessSettings{},
		}, nil
	}

	mux := http.NewServeMux()
	err := RegisterIdP(mux, ds, logger, cfg)
	require.NoError(t, err)

	// Verify all three endpoints are registered
	t.Run("metadata endpoint registered", func(t *testing.T) {
		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Should return error since certificate not configured
		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("SSO endpoint registered", func(t *testing.T) {
		req := httptest.NewRequest("POST", idpSSOPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Should return 501 Not Implemented (handler stub)
		require.Equal(t, http.StatusNotImplemented, w.Code)
	})

	t.Run("signing cert endpoint registered", func(t *testing.T) {
		req := httptest.NewRequest("GET", idpSigningCertPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Should return 501 Not Implemented (handler stub)
		require.Equal(t, http.StatusNotImplemented, w.Code)
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
	// Test certificate
	testCertPEM := []byte(`-----BEGIN CERTIFICATE-----
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

	testKeyPEM := []byte(`-----BEGIN RSA PRIVATE KEY-----
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

	t.Run("returns SAML metadata when Okta configured", func(t *testing.T) {
		ds := new(mock.Store)
		logger := kitlog.NewNopLogger()

		// Mock AppConfig
		appConfig := &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: "https://fleet.example.com",
			},
			ConditionalAccess: &fleet.ConditionalAccessSettings{},
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appConfig, nil
		}

		// Mock MDM config assets
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
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

		svc := &idpService{
			ds:     ds,
			logger: logger,
		}

		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()

		svc.serveMetadata(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "application/samlmetadata+xml", w.Header().Get("Content-Type"))

		// Verify XML contains expected elements
		body := w.Body.String()
		require.Contains(t, body, "EntityDescriptor")
		require.Contains(t, body, idpSSOPath)
		require.Contains(t, body, "IDPSSODescriptor")
		// Verify SSO URL uses okta.* subdomain
		require.Contains(t, body, "okta.fleet.example.com")
	})

	t.Run("properly appends paths to server URL with existing path", func(t *testing.T) {
		ds := new(mock.Store)
		logger := kitlog.NewNopLogger()

		// Mock AppConfig with server URL that includes a path
		appConfig := &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: "https://fleet.example.com/go/here/for/fleet",
			},
			ConditionalAccess: &fleet.ConditionalAccessSettings{},
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appConfig, nil
		}

		// Mock MDM config assets
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
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

		svc := &idpService{
			ds:     ds,
			logger: logger,
		}

		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()

		svc.serveMetadata(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "application/samlmetadata+xml", w.Header().Get("Content-Type"))

		body := w.Body.String()
		// Verify metadata URL preserves the original path and appends the metadata path
		require.Contains(t, body, "https://fleet.example.com/go/here/for/fleet"+idpMetadataPath)
		// Verify SSO URL preserves the path when prepending okta. subdomain
		require.Contains(t, body, "https://okta.fleet.example.com/go/here/for/fleet"+idpSSOPath)
	})

	t.Run("returns error when certificate assets not found", func(t *testing.T) {
		ds := new(mock.Store)
		logger := kitlog.NewNopLogger()

		// Mock AppConfig
		appConfig := &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: "https://fleet.example.com",
			},
			ConditionalAccess: &fleet.ConditionalAccessSettings{},
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appConfig, nil
		}

		// Mock MDM config assets - return empty map (assets not found)
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, nil
		}

		svc := &idpService{
			ds:     ds,
			logger: logger,
		}

		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()

		svc.serveMetadata(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns error when server URL not configured", func(t *testing.T) {
		ds := new(mock.Store)
		logger := kitlog.NewNopLogger()

		// Mock AppConfig with no server URL
		appConfig := &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: "",
			},
			ConditionalAccess: &fleet.ConditionalAccessSettings{},
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appConfig, nil
		}

		svc := &idpService{
			ds:     ds,
			logger: logger,
		}

		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()

		svc.serveMetadata(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Contains(t, w.Body.String(), "Server URL not configured")
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

// parseCertAndKey parses a PEM-encoded certificate and private key.
// The PEM string should contain both the certificate and private key.
// Test helper function.
func parseCertAndKey(pemData string) (*x509.Certificate, crypto.PrivateKey, error) {
	var cert *x509.Certificate
	var key crypto.PrivateKey

	rest := []byte(pemData)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}

		switch block.Type {
		case "CERTIFICATE":
			var err error
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("parse certificate: %w", err)
			}
		case "RSA PRIVATE KEY":
			var err error
			key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("parse RSA private key: %w", err)
			}
		}
	}

	if cert == nil {
		return nil, nil, errors.New("no certificate found in PEM data")
	}
	if key == nil {
		return nil, nil, errors.New("no private key found in PEM data")
	}

	return cert, key, nil
}

func TestParseCertAndKey(t *testing.T) {
	validPEM := `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`

	t.Run("parses valid PEM with cert and key", func(t *testing.T) {
		cert, key, err := parseCertAndKey(validPEM)
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.NotNil(t, key)
		require.Equal(t, "test-idp.fleet.example.com", cert.Subject.CommonName)
	})

	t.Run("returns error when no certificate", func(t *testing.T) {
		onlyKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAhL8AyvTQZUtMdae59NmD2xaMmESQRrAuhJG0w9mElE+L/ysq
wqTR+JgrpgoemJj6IviSV7Q0dUwBCGTq8LjCaJoozLyIRnEkHaBYs79eTpv5VprU
W7jGD7BZvrW57jocZGG+Ecbd4e3Pw04tqjjYNEKNuCEK/6pENx+B6L8iutNSZIkN
Qxc0y1pBuD504x5BFuB+TPbHosj9N4EqCEgYchhErN5twfqSvVe2BVqkb5fo1xyM
vgWYi98hOk4rr4HfB2zMQfkdSJwx1kS3/zIs7TSQlmHFFUK8rQtvNgQcXHl+0jIE
IHqLAhn14QJ/inztTE597QAPQ0B2b8FH52vWmwIDAQABAoIBAC8dNxhSb8fq8PFc
4cBcBlbEXF5dpbRXPMHt7RuKiI0eSMlkxB/cMH5LoA/P0OLgCJSHSl7Iy+L3e4FE
XVEGvB1Gr1EQm5IVU3K3cIqLkShzUx6C7EcFbQPLXbCVOwTDyLNXKEYHKl+k8qcz
aLKglBlnDkiWlJe0JzrqMjKPLz0OBK7iuDzO9c7yT8AaVTLxZsJRx7LkQkNPp5KJ
BjF4+GKBqLjRN3L3gKRJYL3k9YsS8b7qQSLYBHmHx1K3qFH1dTKgT9+Y0gN7Z8DP
E1B2xCp7MJc5KQGUqV6GXN9W4b6I3GFqKl8+Oq2sL1dQ4wI9x4X4Q8h9b1FZpGjO
C3qQFGECgYEAwMV7K5xR8Q+LPqI4RPQU9LkmRQd8aLdQn2w8X1+cE7jnO3FNqR7K
2VqOKpjJ7P5aZ3kqX0oWJKo8fX8B7q5qPqJNHDGFNYRCFq2mC7CvL+8N1QGY8OJ1
fQ8qL5v1K2W8x9k9MFqG2mH7O1xMm0sZqA8qG9C5Dq9yKhJDqYY7RXECgYEAsVX2
y3RBH5E8L+DqCX8Q0p3fX9aBqC7F7Wvzdt3lKCY4ux7QJH9xR8QPO5qN+5qG7YQu
3z5Dq8Vq4L5q1p8F3x7fL9q7K5XJPY5q7L5Q8L+DqCX8Q0p3fX9aBqC7F7Wvzdt3
lKCY4ux7QJH9xR8QPO5qN+5qG7YQu3z5Dq8Vq4JsCgYEAmC9K8L+DqCX8Q0p3fX9a
BqC7F7Wvzdt3lKCY4ux7QJH9xR8QPO5qN+5qG7YQu3z5Dq8Vq4L5q1p8F3x7fL9q
7K5XJPY5q7L5Q8L+DqCX8Q0p3fX9aBqC7F7Wvzdt3lKCY4ux7QJH9xR8QPO5qN+5
qG7YQu3z5Dq8Vq4L5q1p8F3x7fL9q7K5XJPY5q7L5Q8ECgYEAqC9K8L+DqCX8Q0p3
fX9aBqC7F7Wvzdt3lKCY4ux7QJH9xR8QPO5qN+5qG7YQu3z5Dq8Vq4L5q1p8F3x7
fL9q7K5XJPY5q7L5Q8L+DqCX8Q0p3fX9aBqC7F7Wvzdt3lKCY4ux7QJH9xR8QPO5
qN+5qG7YQu3z5Dq8Vq4L5q1p8F3x7fL9q7K5XJPY5q7L5Q8ECgYBmC9K8L+DqCX8
Q0p3fX9aBqC7F7Wvzdt3lKCY4ux7QJH9xR8QPO5qN+5qG7YQu3z5Dq8Vq4L5q1p8
F3x7fL9q7K5XJPY5q7L5Q8L+DqCX8Q0p3fX9aBqC7F7Wvzdt3lKCY4ux7QJH9xR8
QPO5qN+5qG7YQu3z5Dq8Vq4L5q1p8F3x7fL9q7K5XJPY5q7L5Q8=
-----END RSA PRIVATE KEY-----`

		_, _, err := parseCertAndKey(onlyKey)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no certificate found")
	})

	t.Run("returns error when no private key", func(t *testing.T) {
		onlyCert := `-----BEGIN CERTIFICATE-----
MIIDqDCCApCgAwIBAgIGAZXsT7aXMA0GCSqGSIb3DQEBCwUAMIGUMQswCQYDVQQG
EwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNj
bzENMAsGA1UECgwET2t0YTEUMBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMM
DGRldi01Nzk4ODEyOTEcMBoGCSqGSIb3DQEJARYNaW5mb0Bva3RhLmNvbTAeFw0y
NTAzMzExMzA1NDFaFw0zNTAzMzExMzA2NDFaMIGUMQswCQYDVQQGEwJVUzETMBEG
A1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UE
CgwET2t0YTEUMBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMMDGRldi01Nzk4
ODEyOTEcMBoGCSqGSIb3DQEJARYNaW5mb0Bva3RhLmNvbTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAIS/AMr00GVLTHWnufTZg9sWjJhEkEawLoSRtMPZ
hJRPi/8rKsKk0fiYK6YKHpiY+iL4kle0NHVMAQhk6vC4wmiaKMy8iEZxJB2gWLO/
Xk6b+Vaa1Fu4xg+wWb61ue46HGRhvhHG3eHtz8NOLao42DRCjbghCv+qRDcfgei/
IrrTUmSJDUMXNMtaQbg+dOMeQRbgfkz2x6LI/TeBKghIGHIYRKzebcH6kr1XtgVa
pG+X6NccjL4FmIvfITpOK6+B3wdszEH5HUicMdZEt/8yLO00kJZhxRVCvK0LbzYE
HFx5ftIyBCB6iwIZ9eECf4p87UxOfe0AD0NAdm/BR+dr1psCAwEAATANBgkqhkiG
9w0BAQsFAAOCAQEAWzh9U6/I5G/Uy/BoMTv3lBsbS6h7OGUE2kOTX5YF3+t4EKlG
NHNHx1CcOa7kKb1Cpagnu3UfThlynMVWcUemsnhjN+6DeTGpqX/GGpQ22YKIZbqF
m90jS+CtLQQsi0ciU7w4d981T2I7oRs9yDk+A2ZF9yf8wGi6ocy4EC00dCJ7DoSu
i6HdYiQWk60K4w7LPqtvx2bPPK9j+pmAbuLmHPAQ4qyccDZVDOaPumSer90UyfV6
FkY8/nfrqDk6tE8RyabI3o48Q4m12RoYcA3sZ3Ba3A4CzP7Q0uUFD6nMTqgq4ZeV
FqU+KJOed6qlzj7qy+u5l6CQeajLGdjUxFlFyw==
-----END CERTIFICATE-----`

		_, _, err := parseCertAndKey(onlyCert)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no private key found")
	})
}
