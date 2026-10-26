package fleet

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestValidateCertificateAuthoritiesSpec(t *testing.T) {
	// TODO(hca): placeholder for additional tests to the extent not otherwise covered by client_test
}

func TestEffectiveEJBCAUsernameTemplate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ca       CertificateAuthority
		expected string
	}{
		{"username template set", CertificateAuthority{EJBCAUsernameTemplate: new("user"), CertificateCommonName: new("cn")}, "user"},
		{"username template unset", CertificateAuthority{CertificateCommonName: new("cn")}, "cn"},
		{"username template empty", CertificateAuthority{EJBCAUsernameTemplate: new(""), CertificateCommonName: new("cn")}, "cn"},
		{"nothing set", CertificateAuthority{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.ca.EffectiveEJBCAUsernameTemplate())
		})
	}
}

func TestGroupedCertificateAuthorities(t *testing.T) {
	grouped := GroupedCertificateAuthorities{
		NDESSCEP: &NDESSCEPProxyCA{ // should not be included
			URL:      "https://some-ndes-scep-proxy-url.com",
			AdminURL: "https://some-ndes-admin-url.com",
			Username: "some-ndes-username",
			Password: "some-ndes-password",
		},
		CustomScepProxy: []CustomSCEPProxyCA{ // should both be included
			{
				Name:      "some-custom-scep-proxy-name",
				URL:       "https://some-custom-scep-proxy-url.com",
				Challenge: "some-custom-scep-proxy-challenge",
			},
			{
				Name:      "another-custom-scep-proxy-name",
				URL:       "https://another-custom-scep-proxy-url.com",
				Challenge: "another-custom-scep-proxy-challenge",
			},
		},
	}

	mapped := grouped.ToCustomSCEPProxyCAMap()
	require.Len(t, mapped, 2)
	require.Equal(t, "some-custom-scep-proxy-challenge", mapped["some-custom-scep-proxy-name"].Challenge)
	require.Equal(t, "another-custom-scep-proxy-challenge", mapped["another-custom-scep-proxy-name"].Challenge)
}

func TestGroupCertificateAuthoritiesByTypeEJBCA(t *testing.T) {
	grouped, err := GroupCertificateAuthoritiesByType([]*CertificateAuthority{
		{
			ID:                            1,
			Type:                          string(CATypeEJBCA),
			Name:                          new("full"),
			URL:                           new("https://ejbca.example.com"),
			MTLSCertPEM:                   new("cert"),
			MTLSKeyPEM:                    new("key"),
			TrustCABundlePEM:              new("bundle"),
			EJBCACAName:                   new("ManagementCA"),
			EJBCACertificateProfileName:   new("certProfile"),
			EJBCAEndEntityProfileName:     new("eeProfile"),
			CertificateCommonName:         new("cn"),
			EJBCAUsernameTemplate:         new("username"),
			CertificateUserPrincipalNames: &[]string{"upn"},
		},
		// The optional fields are not set.
		{ID: 2, Type: string(CATypeEJBCA), Name: new("minimal"), URL: new("https://ejbca.example.com")},
	})
	require.NoError(t, err)
	require.False(t, grouped.IsEmpty())
	require.Equal(t, []EJBCA{
		{
			ID: 1, Name: "full", URL: "https://ejbca.example.com", MTLSCertPEM: "cert", MTLSKeyPEM: "key", TrustCABundlePEM: "bundle",
			CertificateAuthorityName: "ManagementCA", CertificateProfileName: "certProfile", EndEntityProfileName: "eeProfile",
			CertificateCommonName: "cn", UsernameTemplate: "username", CertificateUserPrincipalNames: []string{"upn"},
		},
		{ID: 2, Name: "minimal", URL: "https://ejbca.example.com"},
	}, grouped.EJBCA)
}

func TestPreprocessCAFields(t *testing.T) {
	t.Run("DigiCert CA fields are trimmed and normalized", func(t *testing.T) {
		digiCertCA := &DigiCertCA{
			Name:      "  DigiCert CA  ",
			URL:       "  https://digicert.com  ",
			ProfileID: "  profile_id  ",
		}

		digiCertCA.Preprocess()

		require.Equal(t, "DigiCert CA", digiCertCA.Name)
		require.Equal(t, "https://digicert.com", digiCertCA.URL)
		require.Equal(t, "profile_id", digiCertCA.ProfileID)
	})

	t.Run("DigiCert CA Update payload fields are trimmed and normalized", func(t *testing.T) {
		digiCertCAUpdate := &DigiCertCAUpdatePayload{
			Name:      ptr.String("  DigiCert CA  "),
			URL:       ptr.String("  https://digicert.com  "),
			ProfileID: ptr.String("  profile_id  "),
		}

		digiCertCAUpdate.Preprocess()

		require.Equal(t, "DigiCert CA", *digiCertCAUpdate.Name)
		require.Equal(t, "https://digicert.com", *digiCertCAUpdate.URL)
		require.Equal(t, "profile_id", *digiCertCAUpdate.ProfileID)
	})

	t.Run("Custom SCEP Proxy CA fields are trimmed and normalized", func(t *testing.T) {
		customSCEPProxyCA := &CustomSCEPProxyCA{
			Name: "  Custom SCEP Proxy CA  ",
			URL:  "  https://scep-proxy.com  ",
		}

		customSCEPProxyCA.Preprocess()

		require.Equal(t, "Custom SCEP Proxy CA", customSCEPProxyCA.Name)
		require.Equal(t, "https://scep-proxy.com", customSCEPProxyCA.URL)
	})

	t.Run("Custom SCEP Proxy CA Update payload fields are trimmed and normalized", func(t *testing.T) {
		customSCEPProxyCAUpdate := &CustomSCEPProxyCAUpdatePayload{
			Name: ptr.String("  Custom SCEP Proxy CA  "),
			URL:  ptr.String("  https://scep-proxy.com  "),
		}

		customSCEPProxyCAUpdate.Preprocess()

		require.Equal(t, "Custom SCEP Proxy CA", *customSCEPProxyCAUpdate.Name)
		require.Equal(t, "https://scep-proxy.com", *customSCEPProxyCAUpdate.URL)
	})

	t.Run("NDES CA fields are trimmed and normalized", func(t *testing.T) {
		ndesCA := &NDESSCEPProxyCA{
			URL:      "  https://ndes.com  ",
			AdminURL: "  https://ndes.com/admin  ",
			Username: "  admin  ",
			Password: "  password  ",
		}

		ndesCA.Preprocess()

		require.Equal(t, "https://ndes.com", ndesCA.URL)
		require.Equal(t, "https://ndes.com/admin", ndesCA.AdminURL)
		require.Equal(t, "admin", ndesCA.Username)
		require.Equal(t, "  password  ", ndesCA.Password)
	})

	t.Run("NDES CA Update payload fields are trimmed and normalized", func(t *testing.T) {
		ndesCAUpdate := &NDESSCEPProxyCAUpdatePayload{
			URL:      ptr.String("  https://ndes.com  "),
			AdminURL: ptr.String("  https://ndes.com/admin  "),
			Username: ptr.String("  admin  "),
			Password: ptr.String("  password  "),
		}

		ndesCAUpdate.Preprocess()

		require.Equal(t, "https://ndes.com", *ndesCAUpdate.URL)
		require.Equal(t, "https://ndes.com/admin", *ndesCAUpdate.AdminURL)
		require.Equal(t, "admin", *ndesCAUpdate.Username)
		require.Equal(t, "  password  ", *ndesCAUpdate.Password)
	})

	t.Run("Smallstep SCEP Proxy CA fields are trimmed and normalized", func(t *testing.T) {
		smallstepSCEPProxyCA := &SmallstepSCEPProxyCA{
			Name:         "  Smallstep SCEP Proxy CA  ",
			URL:          "  https://scep-proxy.com  ",
			ChallengeURL: "  https://scep-proxy.com/challenge  ",
			Username:     "  username  ",
			Password:     "  password  ",
		}

		smallstepSCEPProxyCA.Preprocess()

		require.Equal(t, "Smallstep SCEP Proxy CA", smallstepSCEPProxyCA.Name)
		require.Equal(t, "https://scep-proxy.com", smallstepSCEPProxyCA.URL)
		require.Equal(t, "https://scep-proxy.com/challenge", smallstepSCEPProxyCA.ChallengeURL)
		require.Equal(t, "username", smallstepSCEPProxyCA.Username)
		require.Equal(t, "  password  ", smallstepSCEPProxyCA.Password)
	})

	t.Run("Smallstep SCEP Proxy CA Update payload fields are trimmed and normalized", func(t *testing.T) {
		smallstepSCEPProxyCAUpdate := &SmallstepSCEPProxyCAUpdatePayload{
			Name:         ptr.String("  Smallstep SCEP Proxy CA  "),
			URL:          ptr.String("  https://scep-proxy.com  "),
			ChallengeURL: ptr.String("  https://scep-proxy.com/challenge  "),
			Username:     ptr.String("  username  "),
			Password:     ptr.String("  password  "),
		}

		smallstepSCEPProxyCAUpdate.Preprocess()

		require.Equal(t, "Smallstep SCEP Proxy CA", *smallstepSCEPProxyCAUpdate.Name)
		require.Equal(t, "https://scep-proxy.com", *smallstepSCEPProxyCAUpdate.URL)
		require.Equal(t, "https://scep-proxy.com/challenge", *smallstepSCEPProxyCAUpdate.ChallengeURL)
		require.Equal(t, "username", *smallstepSCEPProxyCAUpdate.Username)
		require.Equal(t, "  password  ", *smallstepSCEPProxyCAUpdate.Password)
	})
}

func TestRequestCertificatePayloadIdPCredentialsProvided(t *testing.T) {
	ptr := func(s string) *string { return &s }
	u, tok, c := ptr("https://company.okta.com/oauth2/v1/introspect"), ptr("a-token"), ptr("client")

	provided, err := RequestCertificatePayload{}.IdPCredentialsProvided()
	require.NoError(t, err)
	require.False(t, provided)

	provided, err = RequestCertificatePayload{IDPOauthURL: u, IDPToken: tok, IDPClientID: c}.IdPCredentialsProvided()
	require.NoError(t, err)
	require.True(t, provided)

	// A partial set is refused: were it allowed through, the caller would clear the allowlist on
	// the fields it did carry and then skip introspection entirely.
	for _, p := range []RequestCertificatePayload{
		{IDPOauthURL: u}, {IDPToken: tok}, {IDPClientID: c},
		{IDPOauthURL: u, IDPToken: tok}, {IDPOauthURL: u, IDPClientID: c}, {IDPToken: tok, IDPClientID: c},
	} {
		_, err := p.IdPCredentialsProvided()
		require.ErrorContains(t, err, "all must be provided")
		var bre *BadRequestError
		require.ErrorAs(t, err, &bre)
	}
}

func TestCertificateAuthorityESTProxyCA(t *testing.T) {
	for name, tc := range map[string]struct {
		ca      CertificateAuthority
		want    ESTProxyCA
		wantErr string
	}{
		"hydrant uses the client ID and secret": {
			ca: CertificateAuthority{
				ID: 1, Type: string(CATypeHydrant), Name: new("Hydrant"), URL: new("https://hydrant.example.com"),
				ClientID: new("client-id"), ClientSecret: new("client-secret"),
			},
			want: ESTProxyCA{ID: 1, Name: "Hydrant", URL: "https://hydrant.example.com", Username: "client-id", Password: "client-secret"},
		},
		"custom EST uses the username and password": {
			ca: CertificateAuthority{
				ID: 2, Type: string(CATypeCustomESTProxy), Name: new("EST"), URL: new("https://est.example.com"),
				Username: new("user"), Password: new("pass"),
			},
			want: ESTProxyCA{ID: 2, Name: "EST", URL: "https://est.example.com", Username: "user", Password: "pass"},
		},
		"hydrant without a client ID": {
			ca:      CertificateAuthority{Type: string(CATypeHydrant), Name: new("Hydrant"), URL: new("u"), ClientSecret: new("s")},
			wantErr: "Certificate authority does not have a client ID configured.",
		},
		"hydrant without a client secret": {
			ca:      CertificateAuthority{Type: string(CATypeHydrant), Name: new("Hydrant"), URL: new("u"), ClientID: new("c")},
			wantErr: "Certificate authority does not have a client secret configured.",
		},
		"custom EST without a username": {
			ca:      CertificateAuthority{Type: string(CATypeCustomESTProxy), Name: new("EST"), URL: new("u"), Password: new("p")},
			wantErr: "Certificate authority does not have a username configured.",
		},
		"custom EST without a password": {
			ca:      CertificateAuthority{Type: string(CATypeCustomESTProxy), Name: new("EST"), URL: new("u"), Username: new("u")},
			wantErr: "Certificate authority does not have a password configured.",
		},
		"without a URL": {
			ca:      CertificateAuthority{Type: string(CATypeCustomESTProxy), Name: new("EST"), Username: new("u"), Password: new("p")},
			wantErr: "Certificate authority does not have a URL configured.",
		},
		"without a name": {
			ca:   CertificateAuthority{Type: string(CATypeCustomESTProxy), URL: new("https://est.example.com"), Username: new("u"), Password: new("p")},
			want: ESTProxyCA{URL: "https://est.example.com", Username: "u", Password: "p"},
		},
		"another type is not an EST CA": {
			ca:      CertificateAuthority{Type: string(CATypeNDESSCEPProxy), Name: new("NDES"), URL: new("u")},
			wantErr: "Certificate authority of type ndes_scep_proxy is not an EST certificate authority.",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := tc.ca.ESTProxyCA()
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGroupGoogleCASCertificateAuthorities(t *testing.T) {
	ca := &CertificateAuthority{
		ID:                  7,
		Type:                string(CATypeGoogleCAS),
		Name:                new("GOOGLE_WIFI"),
		URL:                 new(""),
		GoogleCASProject:    new("my-project"),
		GoogleCASLocation:   new("us-central1"),
		GoogleCASCAPool:     new("my-ca-pool"),
		GoogleCASAPIKeyJSON: new(`{"type":"service_account"}`),
	}
	grouped, err := GroupCertificateAuthoritiesByType([]*CertificateAuthority{ca})
	require.NoError(t, err)
	require.Equal(t, []GoogleCASCA{{
		ID: 7, Name: "GOOGLE_WIFI", Project: "my-project", Location: "us-central1", CAPool: "my-ca-pool", APIKeyJSON: `{"type":"service_account"}`,
	}}, grouped.GoogleCAS)
	require.False(t, grouped.IsEmpty())

	// The YAML spec parses into the same shape.
	spec, err := ValidateCertificateAuthoritiesSpec(map[string]any{
		"google_cas": []any{map[string]any{
			"name": "GOOGLE_WIFI", "project": "my-project", "location": "us-central1", "ca_pool": "my-ca-pool", "api_key_json": "$KEY",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, []GoogleCASCA{{Name: "GOOGLE_WIFI", Project: "my-project", Location: "us-central1", CAPool: "my-ca-pool", APIKeyJSON: "$KEY"}},
		spec.GoogleCAS)
	require.Equal(t, "projects/my-project/locations/us-central1/caPools/my-ca-pool", spec.GoogleCAS[0].CAPoolResourceName())

	// YAML parses a variable holding the key file as a map, which is stored as the file's compact JSON.
	spec, err = ValidateCertificateAuthoritiesSpec(map[string]any{
		"google_cas": []any{map[string]any{
			"name": "GOOGLE_WIFI", "project": "my-project", "location": "us-central1", "ca_pool": "my-ca-pool",
			"api_key_json": map[string]any{"type": "service_account", "private_key": "-----BEGIN PRIVATE KEY-----\nabc\n"},
		}},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"service_account","private_key":"-----BEGIN PRIVATE KEY-----\nabc\n"}`, string(spec.GoogleCAS[0].APIKeyJSON))

	_, err = ValidateCertificateAuthoritiesSpec(map[string]any{"google_cas": "not a list"})
	require.ErrorContains(t, err, "org_settings.certificate_authorities.google_cas cannot be parsed")

	empty, err := ValidateCertificateAuthoritiesSpec(map[string]any{})
	require.NoError(t, err)
	require.Empty(t, empty.GoogleCAS)
	require.True(t, empty.IsEmpty())
}

func TestGoogleCASKeyJSON(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		want    *GoogleCASKeyJSON
		wantErr string
	}{
		{"string", `{"api_key_json":"{\"type\":\"service_account\"}"}`, new(GoogleCASKeyJSON(`{"type":"service_account"}`)), ""},
		{"object", `{"api_key_json":{ "type": "service_account" }}`, new(GoogleCASKeyJSON(`{"type":"service_account"}`)), ""},
		{"masked", `{"api_key_json":"********"}`, new(GoogleCASKeyJSON(MaskedPassword)), ""},
		{"absent", `{"name":"GOOGLE_VPN"}`, nil, ""},
		{"null", `{"api_key_json":null}`, nil, ""},
		{"number", `{"api_key_json":42}`, nil, "api_key_json must be the service account key file's JSON"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The update payload holds the key as a pointer, so absent and null leave the stored key alone.
			var update GoogleCASCAUpdatePayload
			err := json.Unmarshal([]byte(tc.body), &update)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, update.APIKeyJSON)
		})
	}

	// The update payload is embedded in the request struct, so its decoding must not take over the whole request.
	var req CertificateAuthorityUpdatePayload
	require.NoError(t, json.Unmarshal([]byte(`{"google_cas":{"name":"GOOGLE_VPN","api_key_json":{"type":"service_account"}}}`), &req))
	require.Equal(t, "GOOGLE_VPN", *req.GoogleCASCAUpdatePayload.Name)
	require.JSONEq(t, `{"type":"service_account"}`, string(*req.GoogleCASCAUpdatePayload.APIKeyJSON))
	require.Nil(t, req.HydrantCAUpdatePayload)
}

func TestGoogleCASCAEquals(t *testing.T) {
	a := GoogleCASCA{ID: 1, Name: "WIFI", Project: "p", Location: "l", CAPool: "c", APIKeyJSON: "k"}
	b := a
	b.ID = 0
	require.True(t, a.Equals(&b), "IDs are ignored")

	// The same key as pasted in the UI and as GitOps re-encodes it.
	a.APIKeyJSON = "{\n  \"type\": \"service_account\",\n  \"private_key\": \"-----BEGIN PRIVATE KEY-----\\nabc\\n\"\n}"
	b.APIKeyJSON = `{"private_key":"-----BEGIN PRIVATE KEY-----\nabc\n","type":"service_account"}`
	require.True(t, a.Equals(&b), "key formatting and field order are ignored")

	for _, change := range []func(*GoogleCASCA){
		func(ca *GoogleCASCA) { ca.Name = "other" },
		func(ca *GoogleCASCA) { ca.Project = "other" },
		func(ca *GoogleCASCA) { ca.Location = "other" },
		func(ca *GoogleCASCA) { ca.CAPool = "other" },
		func(ca *GoogleCASCA) { ca.APIKeyJSON = "other" },
		func(ca *GoogleCASCA) {
			ca.APIKeyJSON = `{"private_key":"-----BEGIN PRIVATE KEY-----\nxyz\n","type":"service_account"}`
		},
	} {
		c := a
		change(&c)
		require.False(t, a.Equals(&c))
	}
}
