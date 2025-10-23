package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/est_ca"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

const (
	// Go makes it a bit of a pain to generate a CSR with both a SAN email and UPN so the below
	// was generated with the following openSSL commands:
	/*
		UUID="85700036-11ef-11e1-bbda-389239cc2c41"
		UPN="fleetie@example.com"
		USERNAME="${UPN}"
		# USERNAME="badactor@example.com" # uncomment to make "bad" CSR

		# generate the password-protected private key
		openssl genpkey -algorithm RSA -out test.key -pkeyopt rsa_keygen_bits:2048 -aes256 -pass pass:$UUID

		# generate CSR signed with that private key
		openssl req -new -sha256 -key test.key -out test.csr -subj /CN=Test -addext "subjectAltName=DNS:example.com, email:$USERNAME, URI:ID:FleetDM:GUID:$UUID, otherName:msUPN;UTF8:$UPN" -passin pass:$UUID

		# modify CSR to be one-line with \n string literal characters to address API limitations
		sed 's/$/\\n/' test.csr | tr -d '\n' > test-escaped.csr
	*/
	goodCSR = "-----BEGIN CERTIFICATE REQUEST-----\nMIIC8jCCAdoCAQAwDzENMAsGA1UEAwwEVGVzdDCCASIwDQYJKoZIhvcNAQEBBQAD\nggEPADCCAQoCggEBALMrkHOVZWVGv9PqU20NgpWed9MdRtMc8406GGWQJ3Rj9/8J\ncy8LOx1d5/XWLKK5VbN2c1hD/a26qkgHtDMfzRXnv5oFybkhaI5tlc9yhQmJVFI2\nRIBsSkZvIlX+SNWV2RuiyVHyGbjhzi3wZen1s0aOeXMMHdD5FVEngX4Fz3TuTb/Z\n8romrsSmWb32fQyQxola9/xe0IAnXZocrxi4xPjNKQbEN/2+gQ/MRJx+c+xnV3MV\nIrXn+8Av8MMBsXhCDlmT2QrpRezNAwWwRni9yKOb0sZMtTDrsCOgAmWsj0Qxf/AS\nMPh7xbozXK4ubf5ombYxEdwGgYl/IKQUKvBKYMMCAwEAAaCBnTCBmgYJKoZIhvcN\nAQkOMYGMMIGJMIGGBgNVHREEfzB9ggtleGFtcGxlLmNvbYETZmxlZXRpZUBleGFt\ncGxlLmNvbYY0SUQ6RmxlZXRETTpHVUlEOjg1NzAwMDM2LTExZWYtMTFlMS1iYmRh\nLTM4OTIzOWNjMmM0MaAjBgorBgEEAYI3FAIDoBUME2ZsZWV0aWVAZXhhbXBsZS5j\nb20wDQYJKoZIhvcNAQELBQADggEBABSBUwyvH/B4kMi9haabDmXpgjb+I7GN2ibz\nN9xS0D/p1TEPNZ2owMdd71oEUPO+pL4PeOIKkn/TRm5ZjnVHtlwlz9PPtkyg7n0d\n6v1L0PPn17jMu9o5u984oP+PYt/VXjJfqzSv2QY2fuR7u108bnxVfWh03n0w1+is\npDQhM5jT+RmXbeOiMIwLojwsYV78y3IYu9ElskonL2v8HQUD9yP8TKlASEhYOD7N\npPLSre8uKL3+A1nyvhG53Ia5xID9mQR3cMO0g6wOoCMerJ4QYMX9jkfPolteT25m\n3NKghdVqvxjm/Oxp7ZFn7LsbdALjnXDYbnNYl8BQTc1rMInnuOw=\n-----END CERTIFICATE REQUEST-----\n"
	badCSR  = "-----BEGIN CERTIFICATE REQUEST-----\nMIIC9DCCAdwCAQAwDzENMAsGA1UEAwwEVGVzdDCCASIwDQYJKoZIhvcNAQEBBQAD\nggEPADCCAQoCggEBAKUUUwsYGpfCCFZYPFL2KLMtf9QdKTizvv3xGPPh6exUo5tB\nEIyhuifEbVIJwf5BhL3104rAY1uywdcUIHqHtWcmaEzS8G6vn1hE4iOMMh5qG6e2\nzobHTxeRgOSeUKHGXWy93BqS09Nkj5H8zlTJO6NjwD3SKiDYZGQDhljdsHTw9Txt\ndHHrEi+y4Qn4FoAf/ie7x2OmfemhLIqpLpU6BxMmqiEHkGObNNlgFGsHGGC3qs9G\nR+2roK3r+nQouMKbFL2CqDCd6F/dBfSSYgOTeOJeOLoM6mZuYqF7dTC1ZU9xhPIR\nzwi9sodQ6kYj++ZycUGT56s6/0yEc4E2AUHAeB0CAwEAAaCBnzCBnAYJKoZIhvcN\nAQkOMYGOMIGLMIGIBgNVHREEgYAwfoILZXhhbXBsZS5jb22BFGJhZGFjdG9yQGV4\nYW1wbGUuY29thjRJRDpGbGVldERNOkdVSUQ6ODU3MDAwMzYtMTFlZi0xMWUxLWJi\nZGEtMzg5MjM5Y2MyYzQxoCMGCisGAQQBgjcUAgOgFQwTZmxlZXRpZUBleGFtcGxl\nLmNvbTANBgkqhkiG9w0BAQsFAAOCAQEASt8qgOCQTtYYYr7KDMcp90Kw+ZiJAL8k\nyRhJy4OsiO4mCdUVvzkyccfV+n6U/51ktPjYkWc1CVYXa+KNN/Z0prsAKYmonR9/\nJh3VVeZrwyglsw+X2ct/H9neOC433KfstRYAZ5WGCSaBJRN1+SUI23O6fjQN7DaL\ntzBPMXMcfNZoWj8rbM/E0WjTnlgUi6L3Ppys5xq1vupdQCiryE8J8A9kKHnMyEi4\nqkCoKOBajEIT9tyFKg5NDjMbIAHFLoUWpLeEtgrGnq5bqBE+q/gOUFb+uqJmQQQz\nVlzFj30tfmt3uBq79Wne1Hu0S634eaCbHOmbuOmLforQqzKpaHXqPQ==\n-----END CERTIFICATE REQUEST-----\n"
)

func TestRequestCertificate(t *testing.T) {
	t.Parallel()

	// Setup mock Oauth server
	defaultOauthIntrospectResponse := map[string]interface{}{
		"active":   true,
		"username": "fleetie@example.com",
	}
	oauthIntrospectResponse := defaultOauthIntrospectResponse
	oauthIntrospectStatus := http.StatusOK
	mockOauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth2/v1/introspect" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if oauthIntrospectStatus != http.StatusOK {
			w.WriteHeader(oauthIntrospectStatus)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(oauthIntrospectResponse)
		require.NoError(t, err)
	}))
	defer mockOauthServer.Close()

	// Setup mock hydrant server
	defaultHydrantSimpleEnrollResponse := "abc123"
	hydrantSimpleEnrollResponse := defaultHydrantSimpleEnrollResponse
	hydrantSimpleEnrollStatus := http.StatusOK

	mockHydrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Path != "/cacerts" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/pkcs7-mime")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Imagine if there was actually CA cert data here..."))
			require.NoError(t, err)
			return
		}

		if r.Method != http.MethodPost || r.URL.Path != "/simpleenroll" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if hydrantSimpleEnrollStatus != http.StatusOK {
			w.WriteHeader(hydrantSimpleEnrollStatus)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(hydrantSimpleEnrollResponse))
		require.NoError(t, err)
	}))
	defer mockHydrantServer.Close()

	hydrantCA := &fleet.CertificateAuthority{
		ID:           1,
		Name:         ptr.String("TestHydrantCA"),
		Type:         string(fleet.CATypeHydrant),
		URL:          &mockHydrantServer.URL,
		ClientID:     ptr.String("test-client-id"),
		ClientSecret: ptr.String("test-client-secret"),
	}
	digicertCA := &fleet.CertificateAuthority{
		ID:        2,
		Name:      ptr.String("TestDigiCertCA"),
		Type:      string(fleet.CATypeDigiCert),
		URL:       ptr.String("https://api.digicert.com"),
		APIToken:  ptr.String("test-api-token"),
		ProfileID: ptr.String("test-profile-id"),
	}

	baseSetupForTests := func() (*Service, context.Context) {
		ds := new(mock.Store)

		// Setup DS mocks
		ds.GetCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
			require.True(t, includeSecrets, "RequestCertificate should always fetch secrets")
			for _, ca := range []*fleet.CertificateAuthority{hydrantCA, digicertCA} {
				if ca.ID == id {
					return ca, nil
				}
			}
			return nil, common_mysql.NotFound("certificate authority")
		}

		ds.GetCertificateAuthorityByIDFuncInvoked = false
		authorizer, err := authz.NewAuthorizer()
		require.NoError(t, err)

		logger := log.NewLogfmtLogger(os.Stdout)
		svc := &Service{
			logger: logger,
			ds:     ds,
			authz:  authorizer,
			estService: est_ca.NewService(
				est_ca.WithTimeout(2*time.Second),
				est_ca.WithLogger(logger),
			),
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		oauthIntrospectResponse = defaultOauthIntrospectResponse
		oauthIntrospectStatus = http.StatusOK
		hydrantSimpleEnrollResponse = defaultHydrantSimpleEnrollResponse
		hydrantSimpleEnrollStatus = http.StatusOK

		return svc, ctx
	}

	invalidCSR := InvalidCSRError{}
	invalidIDP := InvalidIDPTokenError{}

	t.Run("Request a certificate - Happy path", func(t *testing.T) {
		svc, ctx := baseSetupForTests()

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN CERTIFICATE-----\n"+hydrantSimpleEnrollResponse+"\n-----END CERTIFICATE-----\n", *cert)
	})

	t.Run("Request a certificate - Happy path, no IDP", func(t *testing.T) {
		svc, ctx := baseSetupForTests()

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN CERTIFICATE-----\n"+hydrantSimpleEnrollResponse+"\n-----END CERTIFICATE-----\n", *cert)
	})

	t.Run("Request a certificate - Happy path, no IDP, UPN does not match IDP info(should pass)", func(t *testing.T) {
		svc, ctx := baseSetupForTests()

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: badCSR,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN CERTIFICATE-----\n"+hydrantSimpleEnrollResponse+"\n-----END CERTIFICATE-----\n", *cert)
	})

	t.Run("Request a certificate - CA returns error", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		hydrantSimpleEnrollResponse = "Oh no! Something bad happened"
		hydrantSimpleEnrollStatus = http.StatusInternalServerError
		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.ErrorContains(t, err, "Hydrant certificate request failed")
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - IDP introspection reports non-active token", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		oauthIntrospectResponse = map[string]interface{}{
			"active": false,
		}
		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.ErrorAs(t, err, &invalidIDP)
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - IDP introspection does not return a username", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		oauthIntrospectResponse = map[string]interface{}{
			"active": true,
		}
		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.ErrorAs(t, err, &invalidIDP)
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - IDP introspection returns an error", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		oauthIntrospectResponse = map[string]interface{}{
			"error": "something bad happened",
		}
		oauthIntrospectStatus = http.StatusInternalServerError
		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.ErrorAs(t, err, &invalidIDP)
		require.Nil(t, cert)
	})

	t.Run("Request certificate - non-Hydrant CA", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          digicertCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-idp-client-id"),
		})
		require.ErrorContains(t, err, "This API currently only supports Hydrant Certificate Authorities.")
	})

	t.Run("Request certificate - nonexistent CA", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          999,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-idp-client-id"),
		})
		require.ErrorContains(t, err, "certificate authority was not found in the datastore")
	})

	t.Run("Request certificate - missing IDP client ID", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: nil, // Missing client ID
		})
		require.ErrorContains(t, err, "IDP Client ID, Token, and OAuth URL all must be provided, if any are provided when requesting a certificate.")
	})

	t.Run("Request certificate - missing IDP token", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    nil, // Missing IDP token
			IDPClientID: ptr.String("test-client-id"),
		})
		require.ErrorContains(t, err, "IDP Client ID, Token, and OAuth URL all must be provided, if any are provided when requesting a certificate.")
	})

	t.Run("Request certificate - missing IDP oauth URL", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: nil,
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.ErrorContains(t, err, "IDP Client ID, Token, and OAuth URL all must be provided, if any are provided when requesting a certificate.")
	})

	t.Run("Request certificate - CSR email and UPN do not match", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         badCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.ErrorAs(t, err, &invalidCSR)
	})

	t.Run("Request certificate - CSR is not a CSR, IDP provided", func(t *testing.T) {
		svc, ctx := baseSetupForTests()
		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         "I'm not a CSR at all",
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.ErrorAs(t, err, &invalidCSR)
	})

	t.Run("Request a certificate - CSR is not a CSR, no IDP provided", func(t *testing.T) {
		svc, ctx := baseSetupForTests()

		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: "I am not a CSR",
		})
		require.ErrorAs(t, err, &invalidCSR)
	})
}
