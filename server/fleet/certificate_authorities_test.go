package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestValidateCertificateAuthoritiesSpec(t *testing.T) {
	// TODO(hca): placeholder for additional tests to the extent not otherwise covered by client_test
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

func TestCertificateAuthorityNDESSCEPProxyCA(t *testing.T) {
	ndes := func(url, adminURL, username, password *string) CertificateAuthority {
		return CertificateAuthority{
			ID: 1, Type: string(CATypeNDESSCEPProxy), Name: new("NDES"),
			URL: url, AdminURL: adminURL, Username: username, Password: password,
		}
	}
	for name, tc := range map[string]struct {
		ca      CertificateAuthority
		want    NDESSCEPProxyCA
		wantErr string
	}{
		"all fields": {
			ca:   ndes(new("https://ndes.example.com/scep"), new("https://ndes.example.com/admin"), new("user"), new("pass")),
			want: NDESSCEPProxyCA{ID: 1, URL: "https://ndes.example.com/scep", AdminURL: "https://ndes.example.com/admin", Username: "user", Password: "pass"},
		},
		"without a SCEP URL": {
			ca:      ndes(nil, new("a"), new("u"), new("p")),
			wantErr: "Certificate authority does not have a SCEP URL configured.",
		},
		"without an admin URL": {
			ca:      ndes(new("s"), nil, new("u"), new("p")),
			wantErr: "Certificate authority does not have an admin URL configured.",
		},
		"without a username": {
			ca:      ndes(new("s"), new("a"), nil, new("p")),
			wantErr: "Certificate authority does not have a username configured.",
		},
		"without a password": {
			ca:      ndes(new("s"), new("a"), new("u"), nil),
			wantErr: "Certificate authority does not have a password configured.",
		},
		"another type is not an NDES CA": {
			ca:      CertificateAuthority{Type: string(CATypeHydrant), Name: new("Hydrant"), URL: new("u")},
			wantErr: "Certificate authority of type hydrant is not an NDES certificate authority.",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := tc.ca.NDESSCEPProxyCA()
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
