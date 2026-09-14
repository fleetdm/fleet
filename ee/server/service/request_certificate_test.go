package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	"github.com/fleetdm/fleet/v4/ee/server/service/est"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/httpsig"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/smallstep/pkcs7"
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
	// prefixUPNCSR carries email=fleetie@example.com with UPN=fleetie, the documented shorthand
	// form where the UPN is a prefix of the email.
	prefixUPNCSR = "-----BEGIN CERTIFICATE REQUEST-----\nMIIC4zCCAcsCAQAwDzENMAsGA1UEAwwEVGVzdDCCASIwDQYJKoZIhvcNAQEBBQAD\nggEPADCCAQoCggEBAKna6L5GFWTlNjNVpaDWJadyL+xb8VAMSufVQqyBWKi6SUHy\nbyU+rAEk8prdKN1bFHQFVtEBaeaNgyiwJAioLeYeJ8fpiSU5/HVIyrA15hAUQNN3\nHcUGHRwKzGP+gbQmneNUIxdjMQ/f9FSqEbv15EBveFrYO6BSNWBUS8toLX39QC+S\nVaMPd3Wv5u2eVMGUnnrLPVhG52FwsyklO6ZtQ5eZC+fJ1zvZDiM9Pv6zFD/RcZXg\nxJtfUtJRc52rvZKhJ9nWW8Iy6klUdnzIRv9fX++Aaa8xzAx4I4ib939i9GW1vrH7\nvBwfcEX/ySI7OvU3BDQHHxt1ZOtlEByLegHAzVUCAwEAAaCBjjCBiwYJKoZIhvcN\nAQkOMX4wfDB6BgNVHREEczBxggtleGFtcGxlLmNvbYETZmxlZXRpZUBleGFtcGxl\nLmNvbYY0SUQ6RmxlZXRETTpHVUlEOjg1NzAwMDM2LTExZWYtMTFlMS1iYmRhLTM4\nOTIzOWNjMmM0MaAXBgorBgEEAYI3FAIDoAkMB2ZsZWV0aWUwDQYJKoZIhvcNAQEL\nBQADggEBAGYGbqCFKPCeJ1T2xYCm129asYvmvyIcMGAErQUHdmHacJGVEOcJ9eQD\ndPOvHPYpP6X7i8Oqp8hUcnpV64VWwcogojJwicsWvE2jlmIg94iobi8sC3ikyUNI\nThvoLBa3uj/SCQ7On0BDHjGJ7rXDHKIaInNVH7s5JWl1qageGEHkJfGtlbxTpA+8\n/yNzZrAxiWDtMA6pU3KIt0GyUf8lk1rycPUek2Qjf0bEBqcT9AX94Fxk4mVmXEUV\n/r3ygjeVNVVBO8Ocf/tpz0L/2eq0mt/7UThO/j49y7QTEWu3HzWyO5boLK2Mm0aM\nLMpd2rRDUFZwRXTVvQOtNAy5h3yOKz8=\n-----END CERTIFICATE REQUEST-----\n"
	// noUPNCSR carries the SAN email but no UPN othername.
	noUPNCSR = "-----BEGIN CERTIFICATE REQUEST-----\nMIICyDCCAbACAQAwDzENMAsGA1UEAwwEVGVzdDCCASIwDQYJKoZIhvcNAQEBBQAD\nggEPADCCAQoCggEBAKna6L5GFWTlNjNVpaDWJadyL+xb8VAMSufVQqyBWKi6SUHy\nbyU+rAEk8prdKN1bFHQFVtEBaeaNgyiwJAioLeYeJ8fpiSU5/HVIyrA15hAUQNN3\nHcUGHRwKzGP+gbQmneNUIxdjMQ/f9FSqEbv15EBveFrYO6BSNWBUS8toLX39QC+S\nVaMPd3Wv5u2eVMGUnnrLPVhG52FwsyklO6ZtQ5eZC+fJ1zvZDiM9Pv6zFD/RcZXg\nxJtfUtJRc52rvZKhJ9nWW8Iy6klUdnzIRv9fX++Aaa8xzAx4I4ib939i9GW1vrH7\nvBwfcEX/ySI7OvU3BDQHHxt1ZOtlEByLegHAzVUCAwEAAaB0MHIGCSqGSIb3DQEJ\nDjFlMGMwYQYDVR0RBFowWIILZXhhbXBsZS5jb22BE2ZsZWV0aWVAZXhhbXBsZS5j\nb22GNElEOkZsZWV0RE06R1VJRDo4NTcwMDAzNi0xMWVmLTExZTEtYmJkYS0zODky\nMzljYzJjNDEwDQYJKoZIhvcNAQELBQADggEBAIF6R7tJAZPvNmQGKbQ21i/s5IUX\ng7VsF3xcR+8m52Q1RirwkonCZAd9dkSo2JTO2u85YUzO9Sh09FBMtkI6jDtQYCwh\nwPrpWFm8VkZZgiCpKtJXoOE+tH0UU/XsOIku9JrX5KfoSjgzhGWBdfqRMpq8wQHL\n3j1CiRTTnKsFgGMd0VqQJt/G3gK0GIQ7VvzdwanetLzHNTSIe7rryu1PDCKdWFdi\nQW23hRVUArYGVy5XzqFEDZ4IRrMVxNnPdSMBiFOeVP9lObrd+2uA4oKXUhvN2LxU\n1x3+3FhxAYCO+AeulD7bbmLPQVUrocAtyv6zsigbjxqVw5YRyHqchpLo0wo=\n-----END CERTIFICATE REQUEST-----\n"
	// noEmailCSR carries the UPN othername but no SAN email, built with the same commands
	// minus the "email:$USERNAME" entry.
	noEmailCSR = "-----BEGIN CERTIFICATE REQUEST-----\nMIIC2jCCAcICAQAwDzENMAsGA1UEAwwEVGVzdDCCASIwDQYJKoZIhvcNAQEBBQAD\nggEPADCCAQoCggEBAJTyRlK1B5jg/IXbpsOQ24ch92GaPgXYstgHysZowVdnTj++\n3ucs+XpCbVQvLJ0KFI8wi9sNhZeLG4Srz8wNtljkiixodUNo/YGGK0/0iEbWnhJK\nf5tNzv7ZqQqJ9MQBazhsM80d1nYe/QLgUeCGaaZJChUzeaKACNjtWGsA46lLU9//\nd+30qwO8PYu04bDzk/ZfIx8PouxviOLVK1pj87jb930QrzemuhFOpkgOjEN3Ofms\nDwdzDc2ofNU8gCsYP18+oKHHoY+nmFDLOFeV0jjoQmlL28WKsgAnLH6VTieoBUxs\nunTGBt2hBI1dt/KtExHcGgW1HluSrbu+bf7GAFMCAwEAAaCBhTCBggYJKoZIhvcN\nAQkOMXUwczBxBgNVHREEajBoggtleGFtcGxlLmNvbYY0SUQ6RmxlZXRETTpHVUlE\nOjg1NzAwMDM2LTExZWYtMTFlMS1iYmRhLTM4OTIzOWNjMmM0MaAjBgorBgEEAYI3\nFAIDoBUME2ZsZWV0aWVAZXhhbXBsZS5jb20wDQYJKoZIhvcNAQELBQADggEBAI/J\nrnQBLcEVaVt4JOc6zkotRd32Nf8jqmQKwDYKflecVdNm2g1pDR5MzkdKWizSyHcc\n1/gzJHmRelKrvI1tErkT+kO65eYjJue4aadOahzsIJbgK1l5TNJlU2Oc2MrNnk6Y\nzAWqSKNosP83Z0SM9cWVcy2nGvBdaTpYGHkYi/K2gF8aggu2+K3iU3K4r6MZyray\nJoHhyzDdvr46IdcglVG4AhpB7lIPmMq9oOvdPNM2Wj8HUTWLlwYt6MRMP2YYaHMP\nYtmr12RC7c+ZV6Jznax+DUtUTXho5S7z8E19CgLOB0FTyRj+FaQTjdRhRMBUKLLu\noeeC9q4yKvtw0iNUU9k=\n-----END CERTIFICATE REQUEST-----\n"
)

// An otherName carrying trailing bytes after its OID/value pair used to restart the scan from the
// beginning of the content on every iteration, so the cursor never advanced and the call never
// returned. Any input the caller controls reaching this was a free CPU burn.
func TestExtractCSRUPNTerminatesOnTrailingBytes(t *testing.T) {
	t.Parallel()

	mustMarshal := func(v any) []byte {
		b, err := asn1.Marshal(v)
		require.NoError(t, err)
		return b
	}
	content := slices.Concat(
		mustMarshal(asn1.ObjectIdentifier{1, 2, 3, 4}),
		mustMarshal("hello"),
		mustMarshal(asn1.ObjectIdentifier{1, 2, 3, 5}), // trailing
	)
	otherName := mustMarshal(asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: content})
	san := mustMarshal(asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSequence, IsCompound: true, Bytes: otherName})
	csr := &x509.CertificateRequest{Extensions: []pkix.Extension{
		{Id: asn1.ObjectIdentifier{2, 5, 29, 17}, Value: san},
	}}

	// Closing the channel after the write orders it before the read below, so upnErr is safe to
	// assert on once done fires.
	var upnErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, upnErr = extractCSRUPN(t.Context(), csr)
	}()
	select {
	case <-done:
		require.Error(t, upnErr)
	case <-time.After(10 * time.Second):
		t.Fatal("extractCSRUPN did not return; the othername scan is not advancing")
	}
}

func TestRequestCertificate(t *testing.T) {
	t.Parallel()

	// Counts hits on the mock introspection endpoint, so tests can assert no outbound call.
	// Atomic because the handler runs on the test server's goroutine, not the test's.
	var oauthIntrospectCalls atomic.Int64

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
		oauthIntrospectCalls.Add(1)
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
	customESTCA := &fleet.CertificateAuthority{
		ID:       3,
		Name:     ptr.String("TestCustomESTCA"),
		Type:     string(fleet.CATypeCustomESTProxy),
		URL:      &mockHydrantServer.URL,
		Username: ptr.String("test-username"),
		Password: ptr.String("test-password"),
	}

	useDefaultAuthContext := true

	// Reset by baseSetupForTests; mutated by the allowlist and host-binding subtests.
	var appConfig *fleet.AppConfig
	baseSetupForTests := func() (*Service, *mock.Store, context.Context) {
		ds := new(mock.Store)

		// Setup DS mocks
		ds.GetCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
			require.True(t, includeSecrets, "RequestCertificate should always fetch secrets")
			for _, ca := range []*fleet.CertificateAuthority{hydrantCA, digicertCA, customESTCA} {
				if ca.ID == id {
					return ca, nil
				}
			}
			return nil, common_mysql.NotFound("certificate authority")
		}
		appConfig = &fleet.AppConfig{}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appConfig, nil
		}
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return nil, common_mysql.NotFound("scim user")
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}

		ds.GetCertificateAuthorityByIDFuncInvoked = false
		authorizer, err := authz.NewAuthorizer()
		require.NoError(t, err)

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		svc := &Service{
			logger: logger,
			ds:     ds,
			authz:  authorizer,
			estService: est.NewService(
				est.WithTimeout(2*time.Second),
				est.WithLogger(logger),
			),
		}

		authCtx := &authz_ctx.AuthorizationContext{}
		ctx := authz_ctx.NewContext(context.Background(), authCtx)
		if useDefaultAuthContext {
			authCtx.SetAuthnMethod(authz_ctx.AuthnUserToken)
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
		}

		oauthIntrospectResponse = defaultOauthIntrospectResponse
		oauthIntrospectStatus = http.StatusOK
		oauthIntrospectCalls.Store(0)
		hydrantSimpleEnrollResponse = defaultHydrantSimpleEnrollResponse
		hydrantSimpleEnrollStatus = http.StatusOK

		return svc, ds, ctx
	}

	// signatureAuthSetup authenticates the context by HTTP message signature. A non-nil cert is
	// placed in context as the middleware would; nil exercises the service's own presence check.
	signatureAuthSetup := func(t *testing.T, cert *types.HostIdentityCertificate) (*Service, *mock.Store, context.Context) {
		t.Helper()
		useDefaultAuthContext = false
		t.Cleanup(func() { useDefaultAuthContext = true })

		svc, ds, ctx := baseSetupForTests()
		authCtx, ok := authz_ctx.FromContext(ctx)
		require.True(t, ok)
		authCtx.SetAuthnMethod(authz_ctx.AuthnHTTPMessageSignature)
		if cert != nil {
			ctx = httpsig.NewContext(ctx, *cert)
		}
		return svc, ds, ctx
	}

	// deviceSetup is signatureAuthSetup with a valid certificate for the given host.
	deviceSetup := func(t *testing.T, hostID *uint) (*Service, *mock.Store, context.Context) {
		t.Helper()
		return signatureAuthSetup(t, &types.HostIdentityCertificate{HostID: hostID, NotValidAfter: time.Now().Add(24 * time.Hour)})
	}

	invalidCSR := InvalidCSRError{}
	invalidIDP := InvalidIDPTokenError{}

	t.Run("Request a certificate - Happy path", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN PKCS7-----\n"+hydrantSimpleEnrollResponse+"\n-----END PKCS7-----\n", *cert)
	})

	t.Run("Request a certificate - Happy path, no IDP", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN PKCS7-----\n"+hydrantSimpleEnrollResponse+"\n-----END PKCS7-----\n", *cert)
	})

	t.Run("Request a certificate - Happy path, no IDP, http sig auth", func(t *testing.T) {
		svc, _, ctx := deviceSetup(t, new(uint(1)))

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN PKCS7-----\n"+hydrantSimpleEnrollResponse+"\n-----END PKCS7-----\n", *cert)
	})

	t.Run("Request a certificate - Happy path, no IDP, UPN does not match IDP info(should pass)", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: badCSR,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN PKCS7-----\n"+hydrantSimpleEnrollResponse+"\n-----END PKCS7-----\n", *cert)
	})

	t.Run("Request a certificate - CA returns error", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()
		hydrantSimpleEnrollResponse = "Oh no! Something bad happened"
		hydrantSimpleEnrollStatus = http.StatusInternalServerError
		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-client-id"), // Missing client ID
		})
		require.ErrorContains(t, err, "EST certificate request failed")
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - IDP introspection reports non-active token", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()
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
		svc, _, ctx := baseSetupForTests()
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
		svc, _, ctx := baseSetupForTests()
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

	t.Run("Request a certificate - Custom EST CA", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()
		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  customESTCA.ID,
			CSR: goodCSR,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN PKCS7-----\n"+hydrantSimpleEnrollResponse+"\n-----END PKCS7-----\n", *cert)
	})

	t.Run("Request certificate - non-Hydrant and non-EST CA", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()
		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          digicertCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: ptr.String(mockOauthServer.URL + "/oauth2/v1/introspect"),
			IDPToken:    ptr.String("test-idp-token"),
			IDPClientID: ptr.String("test-idp-client-id"),
		})
		require.ErrorContains(t, err, "This API currently only supports Hydrant and EST Certificate Authorities.")
	})

	t.Run("Request certificate - nonexistent CA", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()
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
		svc, _, ctx := baseSetupForTests()
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
		svc, _, ctx := baseSetupForTests()
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
		svc, _, ctx := baseSetupForTests()
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
		svc, _, ctx := baseSetupForTests()
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
		svc, _, ctx := baseSetupForTests()
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
		svc, _, ctx := baseSetupForTests()

		_, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: "I am not a CSR",
		})
		require.ErrorAs(t, err, &invalidCSR)
	})

	t.Run("Request a certificate - return_pem_certificate true", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()

		issuedDER := generateTestCertDER(t)
		envelope, err := pkcs7.DegenerateCertificate(issuedDER)
		require.NoError(t, err)
		hydrantSimpleEnrollResponse = base64.StdEncoding.EncodeToString(envelope)

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:                   hydrantCA.ID,
			CSR:                  goodCSR,
			ReturnPEMCertificate: true,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)

		block, rest := pem.Decode([]byte(*cert))
		require.NotNil(t, block)
		require.Equal(t, "CERTIFICATE", block.Type)
		require.Equal(t, issuedDER, block.Bytes)
		require.Empty(t, rest)

		parsed, err := x509.ParseCertificate(block.Bytes)
		require.NoError(t, err)
		require.Equal(t, "fleetie@example.com", parsed.Subject.CommonName)
	})

	t.Run("Request a certificate - return_pem_certificate true with whitespace in EST response", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()

		issuedDER := generateTestCertDER(t)
		envelope, err := pkcs7.DegenerateCertificate(issuedDER)
		require.NoError(t, err)
		// EST servers may insert line breaks in the base64 body; ensure we tolerate them.
		b64 := base64.StdEncoding.EncodeToString(envelope)
		var withNewlines strings.Builder
		for i := 0; i < len(b64); i += 64 {
			end := min(i+64, len(b64))
			withNewlines.WriteString(b64[i:end])
			withNewlines.WriteByte('\n')
		}
		hydrantSimpleEnrollResponse = withNewlines.String()

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:                   hydrantCA.ID,
			CSR:                  goodCSR,
			ReturnPEMCertificate: true,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)

		block, _ := pem.Decode([]byte(*cert))
		require.NotNil(t, block)
		require.Equal(t, "CERTIFICATE", block.Type)
		require.Equal(t, issuedDER, block.Bytes)

		parsed, err := x509.ParseCertificate(block.Bytes)
		require.NoError(t, err)
		require.Equal(t, "fleetie@example.com", parsed.Subject.CommonName)
	})

	t.Run("Request a certificate - return_pem_certificate true, malformed PKCS7 returns error", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()
		// hydrantSimpleEnrollResponse defaults to "abc123" which is not a valid PKCS7 envelope.

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:                   hydrantCA.ID,
			CSR:                  goodCSR,
			ReturnPEMCertificate: true,
		})
		require.Error(t, err)
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - return_pem_certificate true rejects envelope with multiple certs", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()

		// Build a PKCS7 SignedData containing two certificates to verify we reject anything
		// that doesn't match RFC 7030's single-issued-certificate response shape.
		signed, err := pkcs7.NewSignedData(nil)
		require.NoError(t, err)
		signed.AddCertificate(parseDERCert(t, generateTestCertDER(t)))
		signed.AddCertificate(parseDERCert(t, generateTestCertDER(t)))
		envelope, err := signed.Finish()
		require.NoError(t, err)
		hydrantSimpleEnrollResponse = base64.StdEncoding.EncodeToString(envelope)

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:                   hydrantCA.ID,
			CSR:                  goodCSR,
			ReturnPEMCertificate: true,
		})
		require.ErrorContains(t, err, "expected exactly 1 certificate")
		require.Nil(t, cert)
	})

	// setAllowlist populates both allowlists with one entry each.
	introspectURL := mockOauthServer.URL + "/oauth2/v1/introspect"
	setAllowlist := func(t *testing.T, introspectionURL, clientID string) {
		t.Helper()
		appConfig.Integrations.CertificatesIdPIntrospectionURLs = optjson.SetSlice([]string{introspectionURL})
		appConfig.Integrations.CertificatesIdPClientIDs = optjson.SetSlice([]string{clientID})
	}

	// The regression test: a signed request with no IdP fields succeeds with an empty allowlist
	// (see "Happy path, no IDP, http sig auth") and must be rejected once one is configured.
	t.Run("Request a certificate - allowlist populated, signed request with no IDP fields is rejected", func(t *testing.T) {
		svc, _, ctx := deviceSetup(t, new(uint(1)))
		setAllowlist(t, introspectURL, "test-client-id")

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.ErrorContains(t, err, "IdP verification is required by this Fleet server.")
		require.Nil(t, cert)
		require.Zero(t, oauthIntrospectCalls.Load())
	})

	// Each populated list constrains its field. The rest of the matrix is covered by
	// TestCheckCertIdPIntrospection.
	t.Run("Request a certificate - allowlist populated, listed URL with unlisted client ID is rejected", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()
		setAllowlist(t, introspectURL, "test-client-id")

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: new(introspectURL),
			IDPToken:    new("test-idp-token"),
			IDPClientID: new("attacker-client-id"),
		})
		// Distinct from an invalid token, so a config problem reads differently to a credential one.
		require.ErrorContains(t, err, "IdP client ID is not permitted.")
		require.NotErrorAs(t, err, &invalidIDP)
		require.Nil(t, cert)
		require.Zero(t, oauthIntrospectCalls.Load())
	})

	t.Run("Request a certificate - allowlist populated, listed URL and client ID succeed", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()
		setAllowlist(t, introspectURL, "test-client-id")

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: new(introspectURL),
			IDPToken:    new("test-idp-token"),
			IDPClientID: new("test-client-id"),
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.EqualValues(t, 1, oauthIntrospectCalls.Load())
	})

	t.Run("Request a certificate - device request with nil host ID is rejected by the service", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, nil)

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.ErrorContains(t, err, "not associated with an enrolled host")
		require.Nil(t, cert)
		// Rejected before the CA is even loaded.
		require.False(t, ds.GetCertificateAuthorityByIDFuncInvoked)
	})

	t.Run("Request a certificate - device request with no host identity certificate is rejected", func(t *testing.T) {
		svc, _, ctx := signatureAuthSetup(t, nil)

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.ErrorContains(t, err, "Missing host identity certificate")
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - host binding on, matching IdP username succeeds regardless of case", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{{HostID: hostID, Email: "Fleetie@Example.com", Source: fleet.DeviceMappingIDP}}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		// With an empty allowlist, host binding is a pure DB lookup and makes no outbound call.
		require.Zero(t, oauthIntrospectCalls.Load())
	})

	t.Run("Request a certificate - host binding on, mismatched IdP username is rejected", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{{HostID: hostID, Email: "someone-else@example.com", Source: fleet.DeviceMappingIDP}}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.ErrorContains(t, err, "does not match the end user identity recorded for this host")
		require.Nil(t, cert)
		require.Zero(t, oauthIntrospectCalls.Load())
	})

	// The CSR email and UPN are independent SAN entries, so binding the email alone still lets a
	// caller name a victim in the UPN, which is the field 802.1X and AD-backed mTLS authenticate on.
	t.Run("Request a certificate - host binding on, UPN naming another user is rejected", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		// badCSR: email=badactor@example.com, UPN=fleetie@example.com.
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{{HostID: hostID, Email: "badactor@example.com", Source: fleet.DeviceMappingIDP}}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: badCSR,
		})
		require.ErrorContains(t, err, "does not match the end user identity recorded for this host")
		require.Nil(t, cert)
	})

	// The documented shorthand: an IdP username of bob@example.com may appear as UPN bob.
	t.Run("Request a certificate - host binding on, UPN that is a prefix of the email succeeds", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{{HostID: hostID, Email: "fleetie@example.com", Source: fleet.DeviceMappingIDP}}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: prefixUPNCSR,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
	})

	// Fail closed, as the missing-email case does. Every documented CSR recipe includes a UPN and
	// the binding is opt-in, so a CSR without one cannot be bound and is refused.
	t.Run("Request a certificate - host binding on, CSR without a UPN is rejected", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{{HostID: hostID, Email: "fleetie@example.com", Source: fleet.DeviceMappingIDP}}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: noUPNCSR,
		})
		require.ErrorAs(t, err, &invalidCSR)
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - host binding on, host with no recorded identity is rejected", func(t *testing.T) {
		// Fail-closed: stops a host enrolled with a leaked secret even holding a valid token.
		svc, _, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		setAllowlist(t, introspectURL, "test-client-id")

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: new(introspectURL),
			IDPToken:    new("test-idp-token"),
			IDPClientID: new("test-client-id"),
		})
		require.ErrorContains(t, err, "does not match the end user identity recorded for this host")
		require.Nil(t, cert)
	})

	// A host can have device mapping while still having no IdP identity: GetEndUsers returns one
	// record whose other emails are populated but whose IdP username is empty. That is not an
	// identity to bind to, so it is refused like a host with no record at all.
	t.Run("Request a certificate - host binding on, host with only non-IdP emails is rejected", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{
				{HostID: hostID, Email: "fleetie@example.com", Source: fleet.DeviceMappingCustomOverride},
			}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.ErrorContains(t, err, "does not match the end user identity recorded for this host")
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - host binding on, CSR without an email address is rejected", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		// The recorded identity matches the CSR's UPN, so only the missing SAN email can fail this.
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{{HostID: hostID, Email: "fleetie@example.com", Source: fleet.DeviceMappingIDP}}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: noEmailCSR,
		})
		require.ErrorAs(t, err, &invalidCSR)
		require.Nil(t, cert)
	})

	// GetEndUsers only falls back to device mapping when there is no SCIM record, so the SCIM
	// username is the only candidate once one exists.
	t.Run("Request a certificate - host binding on, a SCIM record supersedes device mapping", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return &fleet.ScimUser{UserName: "someone-else@example.com"}, nil
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{{HostID: hostID, Email: "fleetie@example.com", Source: fleet.DeviceMappingIDP}}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:  hydrantCA.ID,
			CSR: goodCSR,
		})
		require.ErrorContains(t, err, "does not match the end user identity recorded for this host")
		require.Nil(t, cert)
	})

	t.Run("Request a certificate - host binding on, both controls apply", func(t *testing.T) {
		svc, ds, ctx := deviceSetup(t, new(uint(1)))
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		setAllowlist(t, introspectURL, "test-client-id")
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return &fleet.ScimUser{UserName: "fleetie@example.com"}, nil
		}

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: new(introspectURL),
			IDPToken:    new("test-idp-token"),
			IDPClientID: new("test-client-id"),
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
	})

	t.Run("Request a certificate - host binding on, token-authenticated request falls back to the allowlist", func(t *testing.T) {
		// No host on the token path, so binding cannot apply; the Linux flow must keep working.
		svc, _, ctx := baseSetupForTests()
		appConfig.Integrations.CertificatesRequireHostEndUserBinding = optjson.SetBool(true)
		setAllowlist(t, introspectURL, "test-client-id")

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:          hydrantCA.ID,
			CSR:         goodCSR,
			IDPOauthURL: new(introspectURL),
			IDPToken:    new("test-idp-token"),
			IDPClientID: new("test-client-id"),
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
	})

	t.Run("Request a certificate - return_pem_certificate false preserves PKCS7 wrapping", func(t *testing.T) {
		svc, _, ctx := baseSetupForTests()

		cert, err := svc.RequestCertificate(ctx, fleet.RequestCertificatePayload{
			ID:                   hydrantCA.ID,
			CSR:                  goodCSR,
			ReturnPEMCertificate: false,
		})
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.Equal(t, "-----BEGIN PKCS7-----\n"+hydrantSimpleEnrollResponse+"\n-----END PKCS7-----\n", *cert)
	})
}

// parseDERCert parses DER-encoded certificate bytes for use as input to PKCS7 SignedData.
func parseDERCert(t *testing.T, der []byte) *x509.Certificate {
	t.Helper()
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

// generateTestCertDER returns DER-encoded bytes of a freshly generated self-signed certificate
// for use in tests that need a realistic PKCS7 envelope payload.
func generateTestCertDER(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "fleetie@example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return der
}
