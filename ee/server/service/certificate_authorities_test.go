package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/ee/server/service/hydrant"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	scep_mock "github.com/fleetdm/fleet/v4/server/mock/scep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCertificateAuthorityByID(t *testing.T) {
	t.Parallel()

	ds := new(mock.Store)
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	svc := newTestService(t, ds)

	t.Run("GET a CA - Happy path", func(t *testing.T) {
		ds.GetCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
			require.False(t, includeSecrets, "includeSecrets should be false - the API should never return the secrets")
			return &fleet.CertificateAuthority{
				ID:                            1,
				Name:                          ptr.String("Test CA"),
				URL:                           ptr.String("https://example.com"),
				Type:                          string(fleet.CATypeDigiCert),
				APIToken:                      ptr.String(fleet.MaskedPassword),
				ProfileID:                     ptr.String("digicert-profile-id"),
				CertificateCommonName:         ptr.String("digicert-common-name"),
				CertificateUserPrincipalNames: &[]string{"digicert-upn1"},
				CertificateSeatID:             ptr.String("digicert-seat-id"),
			}, nil
		}

		returnedCA, err := svc.GetCertificateAuthority(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, returnedCA)
	})

	t.Run("GET a CA - CA not found", func(t *testing.T) {
		ds.GetCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
			require.False(t, includeSecrets, "includeSecrets should be false - the API should never return the secrets")
			return nil, common_mysql.NotFound("CertificateAuthority").WithID(id)
		}

		returnedCA, err := svc.GetCertificateAuthority(ctx, 1)
		require.ErrorContains(t, err, "not found")
		require.Nil(t, returnedCA)
	})
}

func TestListCertificateAuthorities(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	t.Run("GET all CAs - Happy path", func(t *testing.T) {
		ds.ListCertificateAuthoritiesFunc = func(ctx context.Context) ([]*fleet.CertificateAuthoritySummary, error) {
			return []*fleet.CertificateAuthoritySummary{
				{
					ID:   1,
					Name: "TestDigicertCA",
					Type: string(fleet.CATypeDigiCert),
				},
				{
					ID:   2,
					Name: "TestSCEPCA",
					Type: string(fleet.CATypeCustomSCEPProxy),
				},
			}, nil
		}

		returnedCAs, err := svc.ListCertificateAuthorities(ctx)
		require.NoError(t, err)
		require.Len(t, returnedCAs, 2)
	})

	t.Run("GET all CAs - No CAs found", func(t *testing.T) {
		ds.ListCertificateAuthoritiesFunc = func(ctx context.Context) ([]*fleet.CertificateAuthoritySummary, error) {
			return []*fleet.CertificateAuthoritySummary{}, nil
		}

		returnedCAs, err := svc.ListCertificateAuthorities(ctx)
		require.NoError(t, err)
		require.Len(t, returnedCAs, 0)
	})
}

func setupMockCAServers(t *testing.T) (digicertServer, hydrantServer *httptest.Server) {
	// Setup mock digicert server
	pathRegex := regexp.MustCompile(`^/mpki/api/v2/profile/([a-zA-Z0-9_-]+)$`)
	mockDigiCertServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		matches := pathRegex.FindStringSubmatch(r.URL.Path)
		if len(matches) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		profileID := matches[1]

		resp := map[string]string{
			"id":     profileID,
			"name":   "Test CA",
			"status": "Active",
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))

	mockHydrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/cacerts" {
			w.Header().Set("Content-Type", "application/pkcs7-mime")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Imagine if there was actually CA cert data here..."))
			require.NoError(t, err)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))

	return mockDigiCertServer, mockHydrantServer
}

func TestCreatingCertificateAuthorities(t *testing.T) {
	digicertServer, hydrantServer := setupMockCAServers(t)
	digicertURL := digicertServer.URL
	hydrantURL := hydrantServer.URL
	t.Cleanup(func() {
		digicertServer.Close()
		hydrantServer.Close()
	})

	verifyNilFieldsForType := func(t *testing.T, ca *fleet.CertificateAuthority) {
		if ca.Type != string(fleet.CATypeDigiCert) {
			assert.Nil(t, ca.APIToken)
			assert.Nil(t, ca.ProfileID)
			assert.Nil(t, ca.CertificateCommonName)
			assert.Nil(t, ca.CertificateUserPrincipalNames)
			assert.Nil(t, ca.CertificateSeatID)
		}
		if ca.Type != string(fleet.CATypeHydrant) {
			assert.Nil(t, ca.ClientID)
			assert.Nil(t, ca.ClientSecret)
		}
		if ca.Type != string(fleet.CATypeCustomSCEPProxy) {
			assert.Nil(t, ca.Challenge)
		}
		if ca.Type != string(fleet.CATypeNDESSCEPProxy) {
			assert.Nil(t, ca.AdminURL)
		}
		if ca.Type != string(fleet.CATypeSmallstep) {
			assert.Nil(t, ca.ChallengeURL)
		}

		// Since username and password is now shared for NDES and Smallstep
		if ca.Type != string(fleet.CATypeNDESSCEPProxy) && ca.Type != string(fleet.CATypeSmallstep) {
			assert.Nil(t, ca.Username)
			assert.Nil(t, ca.Password)
		}
	}

	createdCAs := []*fleet.CertificateAuthority{}
	baseSetupForCATests := func() (*Service, context.Context) {
		ds := new(mock.Store)
		// Reset createdCAs before each test
		createdCAs = []*fleet.CertificateAuthority{}
		// Setup DS mocks
		ds.NewCertificateAuthorityFunc = func(ctx context.Context, ca *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
			ca.ID = uint(len(createdCAs) + 1) // nolint:gosec // Disable G115 since this is just a test
			createdCAs = append(createdCAs, ca)
			// NewCertificateAuthority would normally call NewActivity() after calling this
			// however it is not easy to mock that method here and it will just panic, so
			// instead we return a specific error from this datastore mock that tests should
			// check for
			return nil, errors.New("mock error to avoid NewActivity panic")
		}
		authorizer, err := authz.NewAuthorizer()
		require.NoError(t, err)

		svc := &Service{
			logger:          log.NewLogfmtLogger(os.Stdout),
			ds:              ds,
			authz:           authorizer,
			digiCertService: digicert.NewService(),
			hydrantService:  hydrant.NewService(),
			scepConfigService: &scep_mock.SCEPConfigService{
				ValidateSCEPURLFunc:               func(_ context.Context, _ string) error { return nil },
				ValidateNDESSCEPAdminURLFunc:      func(_ context.Context, _ fleet.NDESSCEPProxyCA) error { return nil },
				ValidateSmallstepChallengeURLFunc: func(_ context.Context, _ fleet.SmallstepSCEPProxyCA) error { return nil },
			},
		}
		svc.config.Server.PrivateKey = "supersecret"

		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		return svc, ctx
	}

	t.Run("Errors when no CA type is specified", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createRequest := fleet.CertificateAuthorityPayload{}
		createdCA, err := svc.NewCertificateAuthority(ctx, createRequest)
		require.EqualError(t, err, "Couldn't add certificate authority. A certificate authority must be specified")
		require.Nil(t, createdCA)
	})

	t.Run("Errors when multiple CA types are specified", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()
		createRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{},
			Hydrant:  &fleet.HydrantCA{},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createRequest)
		require.EqualError(t, err, "Couldn't add certificate authority. Only one certificate authority can be created at a time")
		require.Nil(t, createdCA)
	})

	t.Run("Create DigiCert CA - Happy path", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: []string{"digicert_user_principal_name"},
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		_, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		assert.Equal(t, createDigicertRequest.DigiCert.Name, *createdCA.Name)
		assert.Equal(t, createDigicertRequest.DigiCert.URL, *createdCA.URL)
		assert.Equal(t, string(fleet.CATypeDigiCert), createdCA.Type)
		require.NotNil(t, createdCA.APIToken)
		assert.Equal(t, createDigicertRequest.DigiCert.APIToken, *createdCA.APIToken)
		require.NotNil(t, createdCA.ProfileID)
		assert.Equal(t, createDigicertRequest.DigiCert.ProfileID, *createdCA.ProfileID)
		require.NotNil(t, createdCA.CertificateCommonName)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateCommonName, *createdCA.CertificateCommonName)
		assert.ElementsMatch(t, createDigicertRequest.DigiCert.CertificateUserPrincipalNames, *createdCA.CertificateUserPrincipalNames)
		require.NotNil(t, createdCA.CertificateSeatID)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateSeatID, *createdCA.CertificateSeatID)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create DigiCert CA - Happy path with variables", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "$FLEET_VAR_HOST_HARDWARE_SERIAL",
				CertificateUserPrincipalNames: []string{"$FLEET_VAR_HOST_HARDWARE_SERIAL"},
				CertificateSeatID:             "$FLEET_VAR_HOST_HARDWARE_SERIAL",
			},
		}

		_, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		assert.Equal(t, createDigicertRequest.DigiCert.Name, *createdCA.Name)
		assert.Equal(t, createDigicertRequest.DigiCert.URL, *createdCA.URL)
		assert.Equal(t, string(fleet.CATypeDigiCert), createdCA.Type)
		require.NotNil(t, createdCA.APIToken)
		assert.Equal(t, createDigicertRequest.DigiCert.APIToken, *createdCA.APIToken)
		require.NotNil(t, createdCA.ProfileID)
		assert.Equal(t, createDigicertRequest.DigiCert.ProfileID, *createdCA.ProfileID)
		require.NotNil(t, createdCA.CertificateCommonName)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateCommonName, *createdCA.CertificateCommonName)
		assert.ElementsMatch(t, createDigicertRequest.DigiCert.CertificateUserPrincipalNames, *createdCA.CertificateUserPrincipalNames)
		require.NotNil(t, createdCA.CertificateSeatID)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateSeatID, *createdCA.CertificateSeatID)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create DigiCert CA - Happy path with no UPNs", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: nil,
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		_, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		assert.Equal(t, createDigicertRequest.DigiCert.Name, *createdCA.Name)
		assert.Equal(t, createDigicertRequest.DigiCert.URL, *createdCA.URL)
		assert.Equal(t, string(fleet.CATypeDigiCert), createdCA.Type)
		require.NotNil(t, createdCA.APIToken)
		assert.Equal(t, createDigicertRequest.DigiCert.APIToken, *createdCA.APIToken)
		require.NotNil(t, createdCA.ProfileID)
		assert.Equal(t, createDigicertRequest.DigiCert.ProfileID, *createdCA.ProfileID)
		require.NotNil(t, createdCA.CertificateCommonName)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateCommonName, *createdCA.CertificateCommonName)
		assert.Nil(t, *createdCA.CertificateUserPrincipalNames)
		require.NotNil(t, createdCA.CertificateSeatID)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateSeatID, *createdCA.CertificateSeatID)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create Hydrant CA - Happy path", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()
		fmt.Println(hydrantURL)
		createHydrantRequest := fleet.CertificateAuthorityPayload{
			Hydrant: &fleet.HydrantCA{
				Name:         "HydrantWIFI",
				URL:          hydrantURL,
				ClientID:     "hydrant_client_id",
				ClientSecret: "hydrant_client_secret",
			},
		}

		_, err := svc.NewCertificateAuthority(ctx, createHydrantRequest)
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		assert.Equal(t, createHydrantRequest.Hydrant.Name, *createdCA.Name)
		assert.Equal(t, createHydrantRequest.Hydrant.URL, *createdCA.URL)
		assert.Equal(t, string(fleet.CATypeHydrant), createdCA.Type)
		require.NotNil(t, createdCA.ClientID)
		assert.Equal(t, createHydrantRequest.Hydrant.ClientID, *createdCA.ClientID)
		require.NotNil(t, createdCA.ClientSecret)
		assert.Equal(t, createHydrantRequest.Hydrant.ClientSecret, *createdCA.ClientSecret)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create Custom SCEP CA - Happy path", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createCustomSCEPRequest := fleet.CertificateAuthorityPayload{
			CustomSCEPProxy: &fleet.CustomSCEPProxyCA{
				Name:      "CustomSCEPWIFI",
				URL:       "https://customscep.example.com",
				Challenge: "challenge",
			},
		}

		_, err := svc.NewCertificateAuthority(ctx, createCustomSCEPRequest)
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		assert.Equal(t, createCustomSCEPRequest.CustomSCEPProxy.Name, *createdCA.Name)
		assert.Equal(t, createCustomSCEPRequest.CustomSCEPProxy.URL, *createdCA.URL)
		assert.Equal(t, string(fleet.CATypeCustomSCEPProxy), createdCA.Type)
		require.NotNil(t, createdCA.Challenge)
		assert.Equal(t, createCustomSCEPRequest.CustomSCEPProxy.Challenge, *createdCA.Challenge)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create NDES SCEP CA - Happy path", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createNDESSCEPRequest := fleet.CertificateAuthorityPayload{
			NDESSCEPProxy: &fleet.NDESSCEPProxyCA{
				URL:      "https://ndes.example.com",
				AdminURL: "https://ndes-admin.example.com",
				Username: "ndes_user",
				Password: "ndes_password",
			},
		}

		_, err := svc.NewCertificateAuthority(ctx, createNDESSCEPRequest)
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		assert.Equal(t, "NDES", *createdCA.Name)
		assert.Equal(t, createNDESSCEPRequest.NDESSCEPProxy.URL, *createdCA.URL)
		assert.Equal(t, string(fleet.CATypeNDESSCEPProxy), createdCA.Type)
		require.NotNil(t, createdCA.AdminURL)
		assert.Equal(t, createNDESSCEPRequest.NDESSCEPProxy.AdminURL, *createdCA.AdminURL)
		require.NotNil(t, createdCA.Username)
		assert.Equal(t, createNDESSCEPRequest.NDESSCEPProxy.Username, *createdCA.Username)
		require.NotNil(t, createdCA.Password)
		assert.Equal(t, createNDESSCEPRequest.NDESSCEPProxy.Password, *createdCA.Password)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create Smallstep SCEP CA - Happy path", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createSmallstepRequest := fleet.CertificateAuthorityPayload{
			Smallstep: &fleet.SmallstepSCEPProxyCA{
				Name:         "SmallstepWIFI",
				URL:          "https://smallstep.example.com",
				ChallengeURL: "https://smallstep.example.com/challenge",
				Username:     "smallstep_user",
				Password:     "smallstep_password",
			},
		}

		_, err := svc.NewCertificateAuthority(ctx, createSmallstepRequest)
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		assert.Equal(t, createSmallstepRequest.Smallstep.Name, *createdCA.Name)
		assert.Equal(t, createSmallstepRequest.Smallstep.URL, *createdCA.URL)
		assert.Equal(t, string(fleet.CATypeSmallstep), createdCA.Type)
		require.NotNil(t, createdCA.ChallengeURL)
		assert.Equal(t, createSmallstepRequest.Smallstep.ChallengeURL, *createdCA.ChallengeURL)
		require.NotNil(t, createdCA.Username)
		assert.Equal(t, createSmallstepRequest.Smallstep.Username, *createdCA.Username)
		require.NotNil(t, createdCA.Password)
		assert.Equal(t, createSmallstepRequest.Smallstep.Password, *createdCA.Password)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create DigiCert CA - Bad Name", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "Digicert WIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: []string{"digicert_user_principal_name"},
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "Invalid characters in the \"name\" field")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create DigiCert CA - empty CN", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "Digicert WIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "",
				CertificateUserPrincipalNames: []string{"digicert_user_principal_name"},
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "Invalid characters in the \"name\" field.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create DigiCert CA - unsupported variable in CN", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "$FLEET_VAR_BOZO",
				CertificateUserPrincipalNames: []string{"digicert_user_principal_name"},
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "FLEET_VAR_BOZO is not allowed in CA Common Name")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	// Digicert errors
	t.Run("Create DigiCert CA - Bad URL format", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           "bozo",
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: []string{"digicert_user_principal_name"},
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "Invalid DigiCert URL")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create DigiCert CA - Bad URL path", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL + "/invalid",
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: []string{"digicert_user_principal_name"},
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "Could not verify DigiCert profile ID")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create DigiCert CA - empty seat ID", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: []string{"digicert_user_principal_name"},
				CertificateSeatID:             "",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "CA Seat ID cannot be empty")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create DigiCert CA - unsupported variable in seat id", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: []string{"digicert_user_principal_name"},
				CertificateSeatID:             "$FLEET_VAR_BOZO",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "FLEET_VAR_BOZO is not allowed in DigiCert Seat ID")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create DigiCert CA - empty UPN", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: []string{""},
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "DigiCert certificate_user_principal_name cannot be empty if specified")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create DigiCert CA - unsupported variable in UPNs", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           digicertURL,
				APIToken:                      "digicert_api_token",
				ProfileID:                     "digicert_profile_id",
				CertificateCommonName:         "digicert_common_name",
				CertificateUserPrincipalNames: []string{"$FLEET_VAR_BOZO"},
				CertificateSeatID:             "digicert_seat_id",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.ErrorContains(t, err, "FLEET_VAR_BOZO is not allowed in CA User Principal Name")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Hydrant CA - Bad name", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createHydrantRequest := fleet.CertificateAuthorityPayload{
			Hydrant: &fleet.HydrantCA{
				Name:         "Hydrant WIFI",
				URL:          "https://hydrant.example.com",
				ClientID:     "hydrant_client_id",
				ClientSecret: "hydrant_client_secret",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createHydrantRequest)
		require.ErrorContains(t, err, "Invalid characters in the \"name\" field.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Hydrant CA - Bad URL", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createHydrantRequest := fleet.CertificateAuthorityPayload{
			Hydrant: &fleet.HydrantCA{
				Name:         "HydrantWIFI",
				URL:          "bozo",
				ClientID:     "hydrant_client_id",
				ClientSecret: "hydrant_client_secret",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createHydrantRequest)
		require.ErrorContains(t, err, "Invalid Hydrant URL.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Hydrant CA - Bad URL but looks like a hydrant URL", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createHydrantRequest := fleet.CertificateAuthorityPayload{
			Hydrant: &fleet.HydrantCA{
				Name:         "HydrantWIFI",
				URL:          "https://hydrant.example.com/invalid",
				ClientID:     "hydrant_client_id",
				ClientSecret: "hydrant_client_secret",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createHydrantRequest)
		require.ErrorContains(t, err, "Invalid Hydrant URL.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Hydrant CA - missing client_id", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createHydrantRequest := fleet.CertificateAuthorityPayload{
			Hydrant: &fleet.HydrantCA{
				Name:         "HydrantWIFI",
				URL:          "https://hydrant.example.com/invalid",
				ClientID:     "",
				ClientSecret: "hydrant_client_secret",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createHydrantRequest)
		require.ErrorContains(t, err, "Invalid Hydrant Client ID.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Hydrant CA - missing client_secret", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createHydrantRequest := fleet.CertificateAuthorityPayload{
			Hydrant: &fleet.HydrantCA{
				Name:         "HydrantWIFI",
				URL:          "https://hydrant.example.com/invalid",
				ClientID:     "hydrant_client_id",
				ClientSecret: "",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createHydrantRequest)
		require.ErrorContains(t, err, "Invalid Hydrant Client Secret.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Custom SCEP CA - bad name", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createCustomSCEPRequest := fleet.CertificateAuthorityPayload{
			CustomSCEPProxy: &fleet.CustomSCEPProxyCA{
				Name:      "Custom SCEP WIFI",
				URL:       "https://customscep.example.com",
				Challenge: "challenge",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createCustomSCEPRequest)
		require.ErrorContains(t, err, "Invalid characters in the \"name\" field.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Custom SCEP CA - invalid URL format", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createCustomSCEPRequest := fleet.CertificateAuthorityPayload{
			CustomSCEPProxy: &fleet.CustomSCEPProxyCA{
				Name:      "CustomSCEPWIFI",
				URL:       "bozo",
				Challenge: "challenge",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createCustomSCEPRequest)
		require.ErrorContains(t, err, "Invalid SCEP URL.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Custom SCEP CA - missing challenge", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createCustomSCEPRequest := fleet.CertificateAuthorityPayload{
			CustomSCEPProxy: &fleet.CustomSCEPProxyCA{
				Name:      "CustomSCEPWIFI",
				URL:       "https://customscep.example.com",
				Challenge: "",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createCustomSCEPRequest)
		require.ErrorContains(t, err, "Custom SCEP Proxy challenge cannot be empty")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Custom SCEP CA - bad SCEP URL", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		svc.scepConfigService = &scep_mock.SCEPConfigService{
			ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return errors.New("some error") },
		}

		createCustomSCEPRequest := fleet.CertificateAuthorityPayload{
			CustomSCEPProxy: &fleet.CustomSCEPProxyCA{
				Name:      "CustomSCEPWIFI",
				URL:       "https://customscep.example.com",
				Challenge: "challenge",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createCustomSCEPRequest)
		require.ErrorContains(t, err, "Invalid SCEP URL. Please correct and try again.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create NDES SCEP CA - bad URL format", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createNDESSCEPRequest := fleet.CertificateAuthorityPayload{
			NDESSCEPProxy: &fleet.NDESSCEPProxyCA{
				URL:      "bozo",
				AdminURL: "https://ndes-admin.example.com",
				Username: "ndes_user",
				Password: "ndes_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createNDESSCEPRequest)
		require.ErrorContains(t, err, "Invalid NDES SCEP URL.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create NDES SCEP CA - bad SCEP URL", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		svc.scepConfigService = &scep_mock.SCEPConfigService{
			ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return errors.New("some error") },
		}

		createNDESSCEPRequest := fleet.CertificateAuthorityPayload{
			NDESSCEPProxy: &fleet.NDESSCEPProxyCA{
				URL:      "https://ndes.example.com",
				AdminURL: "https://ndes-admin.example.com",
				Username: "ndes_user",
				Password: "ndes_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createNDESSCEPRequest)
		require.ErrorContains(t, err, "Invalid SCEP URL.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create NDES SCEP CA - bad admin URL generic error", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		svc.scepConfigService = &scep_mock.SCEPConfigService{
			ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return nil },
			ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error {
				return errors.New("some error")
			},
		}

		createNDESSCEPRequest := fleet.CertificateAuthorityPayload{
			NDESSCEPProxy: &fleet.NDESSCEPProxyCA{
				URL:      "https://ndes.example.com",
				AdminURL: "https://ndes-admin.example.com",
				Username: "ndes_user",
				Password: "ndes_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createNDESSCEPRequest)
		require.ErrorContains(t, err, "Invalid NDES SCEP admin URL or credentials.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create NDES SCEP CA - bad admin URL NDES Invalid error", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		svc.scepConfigService = &scep_mock.SCEPConfigService{
			ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return nil },
			ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error {
				return NewNDESInvalidError("some error")
			},
		}

		createNDESSCEPRequest := fleet.CertificateAuthorityPayload{
			NDESSCEPProxy: &fleet.NDESSCEPProxyCA{
				URL:      "https://ndes.example.com",
				AdminURL: "https://ndes-admin.example.com",
				Username: "ndes_user",
				Password: "ndes_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createNDESSCEPRequest)
		require.ErrorContains(t, err, "Invalid NDES SCEP admin URL or credentials.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create NDES SCEP CA - bad admin URL with cache full error", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		svc.scepConfigService = &scep_mock.SCEPConfigService{
			ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return nil },
			ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error {
				return NewNDESPasswordCacheFullError("mock error")
			},
		}

		createNDESSCEPRequest := fleet.CertificateAuthorityPayload{
			NDESSCEPProxy: &fleet.NDESSCEPProxyCA{
				URL:      "https://ndes.example.com",
				AdminURL: "https://ndes-admin.example.com",
				Username: "ndes_user",
				Password: "ndes_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createNDESSCEPRequest)
		require.ErrorContains(t, err, "The NDES password cache is full. Please increase the number of cached passwords in NDES and try again.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create NDES SCEP CA - bad admin URL with insufficient permissions error", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		svc.scepConfigService = &scep_mock.SCEPConfigService{
			ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return nil },
			ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error {
				return NewNDESInsufficientPermissionsError("mock error")
			},
		}

		createNDESSCEPRequest := fleet.CertificateAuthorityPayload{
			NDESSCEPProxy: &fleet.NDESSCEPProxyCA{
				URL:      "https://ndes.example.com",
				AdminURL: "https://ndes-admin.example.com",
				Username: "ndes_user",
				Password: "ndes_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createNDESSCEPRequest)
		require.ErrorContains(t, err, "Insufficient permissions for NDES SCEP admin URL. Please correct and try again.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Smallstep SCEP CA - bad name", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createSmallstepRequest := fleet.CertificateAuthorityPayload{
			Smallstep: &fleet.SmallstepSCEPProxyCA{
				Name:         "Smallstep SCEP WIFI",
				URL:          "https://smallstep.example.com",
				ChallengeURL: "https://smallstep.example.com/challenge",
				Username:     "smallstep_user",
				Password:     "smallstep_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createSmallstepRequest)
		require.ErrorContains(t, err, "Invalid characters in the \"name\" field.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Smallstep SCEP CA - invalid URL format", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createSmallstepRequest := fleet.CertificateAuthorityPayload{
			Smallstep: &fleet.SmallstepSCEPProxyCA{
				Name:         "SmallstepSCEPWIFI",
				URL:          "bozo",
				ChallengeURL: "https://smallstep.example.com/challenge",
				Username:     "smallstep_user",
				Password:     "smallstep_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createSmallstepRequest)
		require.ErrorContains(t, err, "Invalid Smallstep SCEP URL.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Smallstep SCEP CA - empty username", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createSmallstepRequest := fleet.CertificateAuthorityPayload{
			Smallstep: &fleet.SmallstepSCEPProxyCA{
				Name:         "SmallstepSCEPWIFI",
				URL:          "https://smallstep.example.com",
				ChallengeURL: "https://smallstep.example.com/challenge",
				Username:     "",
				Password:     "smallstep_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createSmallstepRequest)
		require.ErrorContains(t, err, "Smallstep username cannot be empty")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Smallstep SCEP CA - empty password", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createSmallstepRequest := fleet.CertificateAuthorityPayload{
			Smallstep: &fleet.SmallstepSCEPProxyCA{
				Name:         "SmallstepSCEPWIFI",
				URL:          "https://smallstep.example.com",
				ChallengeURL: "https://smallstep.example.com/challenge",
				Username:     "smallstep_user",
				Password:     "",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createSmallstepRequest)
		require.ErrorContains(t, err, "Smallstep password cannot be empty")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Smallstep SCEP CA - masked password", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createSmallstepRequest := fleet.CertificateAuthorityPayload{
			Smallstep: &fleet.SmallstepSCEPProxyCA{
				Name:         "SmallstepSCEPWIFI",
				URL:          "https://smallstep.example.com",
				ChallengeURL: "https://smallstep.example.com/challenge",
				Username:     "smallstep_user",
				Password:     fleet.MaskedPassword,
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createSmallstepRequest)
		require.ErrorContains(t, err, "Smallstep password cannot be empty")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Smallstep SCEP CA - invalid SCEP URL", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		svc.scepConfigService = &scep_mock.SCEPConfigService{
			ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return errors.New("some error") },
		}

		createSmallstepRequest := fleet.CertificateAuthorityPayload{
			Smallstep: &fleet.SmallstepSCEPProxyCA{
				Name:         "SmallstepSCEPWIFI",
				URL:          "https://smallstep.example.com",
				ChallengeURL: "https://smallstep.example.com/challenge",
				Username:     "smallstep_user",
				Password:     "smallstep_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createSmallstepRequest)
		require.ErrorContains(t, err, "Invalid SCEP URL. Please correct and try again.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})

	t.Run("Create Smallstep SCEP CA - invalid challenge validation", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		svc.scepConfigService = &scep_mock.SCEPConfigService{
			ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return nil },
			ValidateSmallstepChallengeURLFunc: func(_ context.Context, _ fleet.SmallstepSCEPProxyCA) error {
				return errors.New("some error")
			},
		}

		createSmallstepRequest := fleet.CertificateAuthorityPayload{
			Smallstep: &fleet.SmallstepSCEPProxyCA{
				Name:         "SmallstepSCEPWIFI",
				URL:          "https://smallstep.example.com",
				ChallengeURL: "bozo",
				Username:     "smallstep_user",
				Password:     "smallstep_password",
			},
		}

		createdCA, err := svc.NewCertificateAuthority(ctx, createSmallstepRequest)
		require.ErrorContains(t, err, "Invalid challenge URL or credentials.")
		require.Len(t, createdCAs, 0)
		require.Nil(t, createdCA)
	})
}

func TestUpdatingCertificateAuthorities(t *testing.T) {
	t.Parallel()

	digicertServer, hydrantServer := setupMockCAServers(t)
	digicertURL := digicertServer.URL
	hydrantURL := hydrantServer.URL
	t.Cleanup(func() {
		digicertServer.Close()
		hydrantServer.Close()
	})

	digicertID := uint(1)
	hydrantID := uint(2)
	scepID := uint(3)
	ndesID := uint(4)
	smallstepID := uint(5)
	createdCAs := []*fleet.CertificateAuthority{}
	baseSetupForCATests := func() (*Service, context.Context) {
		ds := new(mock.Store)
		// Reset createdCAs before each test
		createdCAs = []*fleet.CertificateAuthority{}
		// Setup CA's mocks
		digicertCA := &fleet.CertificateAuthority{
			ID:                            digicertID,
			Name:                          ptr.String("Test Digicert CA"),
			URL:                           ptr.String("https://digicert1.example.com"),
			Type:                          string(fleet.CATypeDigiCert),
			APIToken:                      ptr.String("test-api-token"),
			ProfileID:                     ptr.String("test-profile-id"),
			CertificateCommonName:         ptr.String("test-common-name $FLEET_VAR_HOST_HARDWARE_SERIAL"),
			CertificateUserPrincipalNames: &[]string{"test-upn $FLEET_VAR_HOST_HARDWARE_SERIAL"},
			CertificateSeatID:             ptr.String("test-seat-id"),
		}

		hydrantCA := &fleet.CertificateAuthority{
			ID:           hydrantID,
			Name:         ptr.String("Hydrant CA"),
			URL:          ptr.String("https://hydrant1.example.com"),
			Type:         string(fleet.CATypeHydrant),
			ClientID:     ptr.String("hydrant-client-id"),
			ClientSecret: ptr.String("hydrant-client-secret"),
		}

		// Custom SCEP CAs
		customSCEPCA := &fleet.CertificateAuthority{
			ID:        scepID,
			Name:      ptr.String("Custom SCEP CA"),
			URL:       ptr.String("https://custom-scep.example.com"),
			Type:      string(fleet.CATypeCustomSCEPProxy),
			Challenge: ptr.String("custom-scep-challenge"),
		}

		// NDES CA
		ndesCA := &fleet.CertificateAuthority{
			ID:       ndesID,
			Name:     ptr.String("NDES"),
			URL:      ptr.String("https://ndes.example.com"),
			AdminURL: ptr.String("https://ndes-admin.example.com"),
			Type:     string(fleet.CATypeNDESSCEPProxy),
			Username: ptr.String("ndes-username"),
			Password: ptr.String("ndes-password"),
		}

		smallstepCA := &fleet.CertificateAuthority{
			ID:           smallstepID,
			Name:         ptr.String("Smallstep CA"),
			URL:          ptr.String("https://smallstep.example.com"),
			Type:         string(fleet.CATypeSmallstep),
			ChallengeURL: ptr.String("https://smallstep.example.com/challenge"),
			Username:     ptr.String("smallstep-username"),
			Password:     ptr.String("smallstep-password"),
		}

		createdCAs = append(createdCAs, digicertCA, hydrantCA, customSCEPCA, ndesCA, smallstepCA)
		ds.GetCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
			for _, ca := range createdCAs {
				if ca.ID == id {
					return ca, nil
				}
			}

			return nil, common_mysql.NotFound("get ca for update")
		}
		ds.UpdateCertificateAuthorityByIDFunc = func(ctx context.Context, certificateAuthorityID uint, updatedCA *fleet.CertificateAuthority) error {
			for index, ca := range createdCAs {
				if ca.ID != certificateAuthorityID {
					continue
				}

				createdCAs[index] = updatedCA

				// UpdateCertificateAuthority would normally call NewActivity() after calling this
				// however it is not easy to mock that method here and it will just panic, so
				// instead we return a specific error from this datastore mock that tests should
				// check for
				return errors.New("mock error to avoid NewActivity panic")
			}

			return common_mysql.NotFound("update") // Did not find CA in list of created CAs
		}
		authorizer, err := authz.NewAuthorizer()
		require.NoError(t, err)

		svc := &Service{
			logger:          log.NewLogfmtLogger(os.Stdout),
			ds:              ds,
			authz:           authorizer,
			digiCertService: digicert.NewService(),
			hydrantService:  hydrant.NewService(),
			scepConfigService: &scep_mock.SCEPConfigService{
				ValidateSCEPURLFunc:               func(_ context.Context, _ string) error { return nil },
				ValidateNDESSCEPAdminURLFunc:      func(_ context.Context, _ fleet.NDESSCEPProxyCA) error { return nil },
				ValidateSmallstepChallengeURLFunc: func(_ context.Context, _ fleet.SmallstepSCEPProxyCA) error { return nil },
			},
		}
		svc.config.Server.PrivateKey = "supersecret"

		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		return svc, ctx
	}

	t.Run("Errors on empty payload", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		err := svc.UpdateCertificateAuthority(ctx, digicertID, fleet.CertificateAuthorityUpdatePayload{})
		require.EqualError(t, err, "Couldn't edit certificate authority. A certificate authority must be specified")
	})

	t.Run("Errors on multiple payloads", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		payload := fleet.CertificateAuthorityUpdatePayload{
			DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
				APIToken: ptr.String("updated-api-token"),
			},
			HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{
				ClientSecret: ptr.String("updated-secret"),
			},
		}

		err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
		require.EqualError(t, err, "Couldn't edit certificate authority. Only one certificate authority can be edited at a time")
	})

	t.Run("Errors if no certificate authority is found", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		payload := fleet.CertificateAuthorityUpdatePayload{
			DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{},
		}

		err := svc.UpdateCertificateAuthority(ctx, 999, payload)
		require.EqualError(t, err, "Couldn't edit certificate authority. Certificate authority with ID 999 does not exist.")
	})

	t.Run("Errors on empty inner update payload", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()
		payloadMap := map[uint]fleet.CertificateAuthorityUpdatePayload{
			digicertID: {
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{},
			},
			hydrantID: {
				HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{},
			},
			scepID: {
				CustomSCEPProxyCAUpdatePayload: &fleet.CustomSCEPProxyCAUpdatePayload{},
			},
			ndesID: {
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{},
			},
			smallstepID: {
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{},
			},
		}

		for id, payload := range payloadMap {

			err := svc.UpdateCertificateAuthority(ctx, id, payload)
			require.Contains(t, err.Error(), "update payload is empty")
		}
	})

	t.Run("Errors on wrong existing CA type", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		payload := fleet.CertificateAuthorityUpdatePayload{
			HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{
				ClientSecret: ptr.String("updated-secret"),
			},
		}

		err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
		require.EqualError(t, err, "Couldn't edit certificate authority. The certificate authority types must be the same.")
	})

	t.Run("Digicert", func(t *testing.T) {
		t.Run("Full update succeeds", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			upns := []string{"updated-upns"}
			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					Name:                          ptr.String("Updated_Digicert"),
					URL:                           &digicertURL,
					APIToken:                      ptr.String("updated-api-token"),
					ProfileID:                     ptr.String("profile-id"),
					CertificateCommonName:         ptr.String("updated-cn"),
					CertificateSeatID:             ptr.String("updated-seat-id"),
					CertificateUserPrincipalNames: &upns,
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "mock error to avoid NewActivity panic")
		})

		t.Run("Allows variable for certificate related fields", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					CertificateCommonName:         ptr.String("$FLEET_VAR_HOST_HARDWARE_SERIAL"),
					CertificateUserPrincipalNames: &[]string{"$FLEET_VAR_HOST_HARDWARE_SERIAL"},
					CertificateSeatID:             ptr.String("$FLEET_VAR_HOST_HARDWARE_SERIAL"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.Error(t, err, "mock error to avoid NewActivity panic")
		})

		t.Run("Fails updating URL if no api token", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					URL: &digicertURL,
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. \"api_token\" must be set when modifying \"url\", or \"profile_id\" of an existing certificate authority: Test Digicert CA")
		})

		t.Run("Fails updating profile ID if no api token", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					ProfileID: ptr.String("fake-profile-id"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. \"api_token\" must be set when modifying \"url\", or \"profile_id\" of an existing certificate authority: Test Digicert CA")
		})

		t.Run("Invalid URL Format", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					URL:      ptr.String("bozo"),
					APIToken: ptr.String("secret-token"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "validation failed: url Couldn't edit certificate authority. Invalid DigiCert URL. Please correct and try again.")
		})

		t.Run("Bad URL Path", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					URL:      ptr.String(digicertURL + "/invalid"),
					APIToken: ptr.String("secret-token"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.ErrorContains(t, err, "Couldn't edit certificate authority. Could not verify DigiCert URL: unexpected DigiCert status code: 400. Please correct and try again.")
		})

		t.Run("Empty CN", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					CertificateCommonName: ptr.String(""),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "validation failed: certificate_common_name Couldn't edit certificate authority. CA Common Name (CN) cannot be empty")
		})

		t.Run("Unsupported variable in CN", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					CertificateCommonName: ptr.String("$FLEET_VAR_BOZO"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "validation failed: certificate_common_name Couldn't edit certificate authority. FLEET_VAR_BOZO is not allowed in CA Common Name (CN)")
		})

		t.Run("Empty Seat ID", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					CertificateSeatID: ptr.String(""),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "validation failed: certificate_seat_id Couldn't edit certificate authority. CA Seat ID cannot be empty")
		})

		t.Run("Unsupported variable in Seat ID", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					CertificateSeatID: ptr.String("$FLEET_VAR_BOZO"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "validation failed: certificate_seat_id Couldn't edit certificate authority. FLEET_VAR_BOZO is not allowed in DigiCert Seat ID")
		})

		t.Run("Empty UPN", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					CertificateUserPrincipalNames: &[]string{""},
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "validation failed: certificate_user_principal_names Couldn't edit certificate authority. DigiCert certificate_user_principal_name cannot be empty if specified")
		})

		t.Run("Unsupported variable in UPN", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				DigiCertCAUpdatePayload: &fleet.DigiCertCAUpdatePayload{
					CertificateUserPrincipalNames: &[]string{"$FLEET_VAR_BOZO"},
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, digicertID, payload)
			require.EqualError(t, err, "validation failed: certificate_user_principal_names Couldn't edit certificate authority. FLEET_VAR_BOZO is not allowed in CA User Principal Name")
		})
	})

	t.Run("Hydrant", func(t *testing.T) {
		t.Run("Full update succeeds", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{
					Name:         ptr.String("Updated_Hydrant"),
					URL:          &hydrantURL,
					ClientID:     ptr.String("updated-id"),
					ClientSecret: ptr.String("updated-secret"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, hydrantID, payload)
			require.EqualError(t, err, "mock error to avoid NewActivity panic")
		})

		t.Run("Bad name", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{
					Name:         ptr.String("Updated Hydrant"),
					URL:          &hydrantURL,
					ClientID:     ptr.String("updated-id"),
					ClientSecret: ptr.String("updated-secret"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, hydrantID, payload)
			require.EqualError(t, err, "validation failed: name Couldn't edit certificate authority. Invalid characters in the \"name\" field. Only letters, numbers and underscores allowed.")
		})

		t.Run("Invalid URL format", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{
					Name:         ptr.String("UpdatedHydrant"),
					URL:          ptr.String("bozo"),
					ClientID:     ptr.String("updated-id"),
					ClientSecret: ptr.String("updated-secret"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, hydrantID, payload)
			require.EqualError(t, err, "validation failed: url Couldn't edit certificate authority. Invalid Hydrant URL. Please correct and try again.")
		})

		t.Run("Bad URL", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{
					Name:         ptr.String("UpdatedHydrant"),
					URL:          ptr.String("https://hydrant.example.com/invalid"),
					ClientID:     ptr.String("updated-id"),
					ClientSecret: ptr.String("updated-secret"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, hydrantID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Invalid Hydrant URL. Please correct and try again.")
		})

		t.Run("Empty ClientID", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{
					ClientID: ptr.String(""),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, hydrantID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Invalid Hydrant Client ID. Please correct and try again.")
		})

		t.Run("Empty ClientSecret", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				HydrantCAUpdatePayload: &fleet.HydrantCAUpdatePayload{
					ClientSecret: ptr.String(""),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, hydrantID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Invalid Hydrant Client Secret. Please correct and try again.")
		})
	})

	t.Run("Custom SCEP", func(t *testing.T) {
		t.Run("Full update succeeds", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				CustomSCEPProxyCAUpdatePayload: &fleet.CustomSCEPProxyCAUpdatePayload{
					Name:      ptr.String("Updated_Scep"),
					URL:       ptr.String("https://customscep.example.com"),
					Challenge: ptr.String("updated-challenge"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, scepID, payload)
			require.EqualError(t, err, "mock error to avoid NewActivity panic")
		})

		t.Run("Bad name", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				CustomSCEPProxyCAUpdatePayload: &fleet.CustomSCEPProxyCAUpdatePayload{
					Name: ptr.String("Updated SCEP"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, scepID, payload)
			require.EqualError(t, err, "validation failed: name Couldn't edit certificate authority. Invalid characters in the \"name\" field. Only letters, numbers and underscores allowed.")
		})

		t.Run("Invalid URL format", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				CustomSCEPProxyCAUpdatePayload: &fleet.CustomSCEPProxyCAUpdatePayload{
					URL:       ptr.String("bozo"),
					Challenge: ptr.String("challenge"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, scepID, payload)
			require.EqualError(t, err, "validation failed: url Couldn't edit certificate authority. Invalid SCEP URL. Please correct and try again.")
		})

		t.Run("Requires challenge when updating URL", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				CustomSCEPProxyCAUpdatePayload: &fleet.CustomSCEPProxyCAUpdatePayload{
					URL: ptr.String("https://customscep.localhost.com"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, scepID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. \"challenge\" must be set when modifying \"url\" of an existing certificate authority: Custom SCEP CA")
		})

		t.Run("Bad SCEP URL", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			svc.scepConfigService = &scep_mock.SCEPConfigService{
				ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return errors.New("some error") },
			}

			payload := fleet.CertificateAuthorityUpdatePayload{
				CustomSCEPProxyCAUpdatePayload: &fleet.CustomSCEPProxyCAUpdatePayload{
					URL:       ptr.String("https://customscep.localhost.com"),
					Challenge: ptr.String("updated-challenge"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, scepID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Invalid SCEP URL. Please correct and try again.")
		})
	})

	t.Run("NDES SCEP", func(t *testing.T) {
		t.Run("Full update succeeds", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{
					URL:      ptr.String("https://ndes.example.com"),
					AdminURL: ptr.String("https://ndes-admin.example.com"),
					Username: ptr.String("ndes_user"),
					Password: ptr.String("ndes_password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, ndesID, payload)
			require.EqualError(t, err, "mock error to avoid NewActivity panic")
		})

		t.Run("Invalid URL format", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{
					URL:      ptr.String("bozo"),
					Password: ptr.String("updated-pasword"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, ndesID, payload)
			require.EqualError(t, err, "validation failed: url Couldn't edit certificate authority. Invalid NDES SCEP URL. Please correct and try again.")
		})

		t.Run("Bad SCEP URL", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			svc.scepConfigService = &scep_mock.SCEPConfigService{
				ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return errors.New("some error") },
			}

			payload := fleet.CertificateAuthorityUpdatePayload{
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{
					URL:      ptr.String("https://ndes.example.com"),
					Password: ptr.String("updated-pasword"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, ndesID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Invalid SCEP URL. Please correct and try again.")
		})

		t.Run("Missing password", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{
					URL: ptr.String("https://ndes.example.com"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, ndesID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. \"password\" must be set when modifying an existing certificate authority: NDES")
		})

		t.Run("Bad admin URL generic error", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			svc.scepConfigService = &scep_mock.SCEPConfigService{
				ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error {
					return errors.New("some error")
				},
			}

			payload := fleet.CertificateAuthorityUpdatePayload{
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{
					AdminURL: ptr.String("https://ndes.example.com"),
					Password: ptr.String("updated-password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, ndesID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Invalid NDES SCEP admin URL or credentials. Please correct and try again.")
		})

		t.Run("Bad admin URL NDES Invalid error", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			svc.scepConfigService = &scep_mock.SCEPConfigService{
				ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error {
					return NewNDESInvalidError("some error")
				},
			}

			payload := fleet.CertificateAuthorityUpdatePayload{
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{
					AdminURL: ptr.String("https://ndes.example.com"),
					Password: ptr.String("updated-password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, ndesID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Invalid NDES SCEP admin URL or credentials. Please correct and try again.")
		})

		t.Run("Bad admin URL NDES Password cache full error", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			svc.scepConfigService = &scep_mock.SCEPConfigService{
				ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error {
					return NewNDESPasswordCacheFullError("some error")
				},
			}

			payload := fleet.CertificateAuthorityUpdatePayload{
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{
					AdminURL: ptr.String("https://ndes.example.com"),
					Password: ptr.String("updated-password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, ndesID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. The NDES password cache is full. Please increase the number of cached passwords in NDES and try again.")
		})

		t.Run("Bad admin URL NDES Insufficient permissions", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			svc.scepConfigService = &scep_mock.SCEPConfigService{
				ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error {
					return NewNDESInsufficientPermissionsError("some error")
				},
			}

			payload := fleet.CertificateAuthorityUpdatePayload{
				NDESSCEPProxyCAUpdatePayload: &fleet.NDESSCEPProxyCAUpdatePayload{
					AdminURL: ptr.String("https://ndes.example.com"),
					Password: ptr.String("updated-password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, ndesID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Insufficient permissions for NDES SCEP admin URL. Please correct and try again.")
		})
	})

	t.Run("Smallstep SCEP", func(t *testing.T) {
		t.Run("Full update succeeds", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					Name:         ptr.String("Updated_Smallstep"),
					URL:          ptr.String("https://smallstep.example.com"),
					ChallengeURL: ptr.String("https://smallstep.example.com/challenge"),
					Username:     ptr.String("smallstep_user"),
					Password:     ptr.String("smallstep_password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "mock error to avoid NewActivity panic")
		})

		t.Run("Bad name", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					Name: ptr.String("Updated Smallstep"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "validation failed: name Couldn't edit certificate authority. Invalid characters in the \"name\" field. Only letters, numbers and underscores allowed.")
		})

		t.Run("Invalid URL format", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					URL:          ptr.String("bozo"),
					ChallengeURL: ptr.String("https://smallstep.example.com/challenge"),
					Username:     ptr.String("updated-username"),
					Password:     ptr.String("updated-password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "validation failed: url Couldn't edit certificate authority. Invalid SCEP URL. Please correct and try again.")
		})

		t.Run("Invalid Challenge URL format", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					URL:          ptr.String("https://smallstep.example.com"),
					ChallengeURL: ptr.String("bozo"),
					Username:     ptr.String("updated-username"),
					Password:     ptr.String("updated-password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "validation failed: url Couldn't edit certificate authority. Invalid Challenge URL. Please correct and try again.")
		})

		t.Run("Bad Smallstep SCEP URL", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()

			svc.scepConfigService = &scep_mock.SCEPConfigService{
				ValidateSCEPURLFunc: func(_ context.Context, _ string) error { return nil },
				ValidateSmallstepChallengeURLFunc: func(_ context.Context, _ fleet.SmallstepSCEPProxyCA) error {
					return errors.New("some error")
				},
			}

			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					URL:          ptr.String("https://smallstep.example.com"),
					ChallengeURL: ptr.String("https://smallstep.example.com/challenge"),
					Username:     ptr.String("updated-username"),
					Password:     ptr.String("updated-password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Invalid challenge URL or credentials. Please correct and try again.")
		})

		t.Run("Requires all fields when updating URL", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()
			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					URL: ptr.String("https://smallstep.example.com"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. \"challenge_url\", \"username\" and \"password\" must be set when modifying \"url\" of an existing certificate authority: Smallstep CA.")
		})

		t.Run("Requires password and username when updating challenge URL", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()
			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					ChallengeURL: ptr.String("https://smallstep.example.com/challenge"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. \"username\" and \"password\" must be set when modifying \"challenge_url\" of an existing certificate authority: Smallstep CA.")
		})

		t.Run("Requires password when updating username", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()
			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					Username: ptr.String("updated-username"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. \"password\" must be set when modifying \"username\" of an existing certificate authority: Smallstep CA.")
		})

		t.Run("Errors on empty username", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()
			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					Username: ptr.String(""),
					Password: ptr.String("updated-password"),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Smallstep SCEP Proxy username cannot be empty")
		})

		t.Run("Errors on empty password", func(t *testing.T) {
			svc, ctx := baseSetupForCATests()
			payload := fleet.CertificateAuthorityUpdatePayload{
				SmallstepSCEPProxyCAUpdatePayload: &fleet.SmallstepSCEPProxyCAUpdatePayload{
					Username: ptr.String("updated-username"),
					Password: ptr.String(""),
				},
			}

			err := svc.UpdateCertificateAuthority(ctx, smallstepID, payload)
			require.EqualError(t, err, "Couldn't edit certificate authority. Smallstep SCEP Proxy password cannot be empty")
		})
	})
}

func TestDeleteCertificateAuthority(t *testing.T) {
	t.Parallel()

	ds := new(mock.Store)
	ctx := context.Background()
	svc := newTestService(t, ds)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

	t.Run("successfully deletes certificate", func(t *testing.T) {
		ds.DeleteCertificateAuthorityFunc = func(ctx context.Context, certificateAuthorityID uint) (*fleet.CertificateAuthoritySummary, error) {
			return nil, errors.New("forced error to short-circuit activity creation")
		}
		err := svc.DeleteCertificateAuthority(ctx, 1)
		require.Error(t, err)
		require.Equal(t, "forced error to short-circuit activity creation", err.Error())
	})

	t.Run("returns not found error if certificate authority does not exist", func(t *testing.T) {
		ds.DeleteCertificateAuthorityFunc = func(ctx context.Context, certificateAuthorityID uint) (*fleet.CertificateAuthoritySummary, error) {
			return nil, common_mysql.NotFound("certificate authority")
		}
		err := svc.DeleteCertificateAuthority(ctx, 999)
		require.Error(t, err)
		require.Contains(t, err.Error(), "certificate authority was not found")
	})
}
