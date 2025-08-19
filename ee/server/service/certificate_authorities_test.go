package service

import (
	"context"
	"encoding/json"
	"errors"
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
				Name:                          "Test CA",
				URL:                           "https://example.com",
				Type:                          string(fleet.CATypeDigiCert),
				APIToken:                      ptr.String(fleet.MaskedPassword),
				ProfileID:                     ptr.String("digicert-profile-id"),
				CertificateCommonName:         ptr.String("digicert-common-name"),
				CertificateUserPrincipalNames: []string{"digicert-upn1"},
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

func TestCreatingCertificateAuthorities(t *testing.T) {
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
	defer mockDigiCertServer.Close()

	mockHydrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/cacerts" {
			w.Header().Set("Content-Type", "application/pkcs7-mime")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Imagine if there was actually CA cert data here..."))
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer mockHydrantServer.Close()

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
				ValidateSCEPURLFunc:          func(_ context.Context, _ string) error { return nil },
				ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error { return nil },
			},
		}
		svc.config.Server.PrivateKey = "supersecret"

		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		return svc, ctx
	}

	t.Run("Create DigiCert CA - Happy path", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           mockDigiCertServer.URL,
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

		assert.Equal(t, createDigicertRequest.DigiCert.Name, createdCA.Name)
		assert.Equal(t, createDigicertRequest.DigiCert.URL, createdCA.URL)
		assert.Equal(t, string(fleet.CATypeDigiCert), createdCA.Type)
		require.NotNil(t, createdCA.APIToken)
		assert.Equal(t, createDigicertRequest.DigiCert.APIToken, *createdCA.APIToken)
		require.NotNil(t, createdCA.ProfileID)
		assert.Equal(t, createDigicertRequest.DigiCert.ProfileID, *createdCA.ProfileID)
		require.NotNil(t, createdCA.CertificateCommonName)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateCommonName, *createdCA.CertificateCommonName)
		assert.ElementsMatch(t, createDigicertRequest.DigiCert.CertificateUserPrincipalNames, createdCA.CertificateUserPrincipalNames)
		require.NotNil(t, createdCA.CertificateSeatID)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateSeatID, *createdCA.CertificateSeatID)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create DigiCert CA - Happy path with variables", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           mockDigiCertServer.URL,
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

		assert.Equal(t, createDigicertRequest.DigiCert.Name, createdCA.Name)
		assert.Equal(t, createDigicertRequest.DigiCert.URL, createdCA.URL)
		assert.Equal(t, string(fleet.CATypeDigiCert), createdCA.Type)
		require.NotNil(t, createdCA.APIToken)
		assert.Equal(t, createDigicertRequest.DigiCert.APIToken, *createdCA.APIToken)
		require.NotNil(t, createdCA.ProfileID)
		assert.Equal(t, createDigicertRequest.DigiCert.ProfileID, *createdCA.ProfileID)
		require.NotNil(t, createdCA.CertificateCommonName)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateCommonName, *createdCA.CertificateCommonName)
		assert.ElementsMatch(t, createDigicertRequest.DigiCert.CertificateUserPrincipalNames, createdCA.CertificateUserPrincipalNames)
		require.NotNil(t, createdCA.CertificateSeatID)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateSeatID, *createdCA.CertificateSeatID)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create DigiCert CA - Happy path with no UPNs", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "DigicertWIFI",
				URL:                           mockDigiCertServer.URL,
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

		assert.Equal(t, createDigicertRequest.DigiCert.Name, createdCA.Name)
		assert.Equal(t, createDigicertRequest.DigiCert.URL, createdCA.URL)
		assert.Equal(t, string(fleet.CATypeDigiCert), createdCA.Type)
		require.NotNil(t, createdCA.APIToken)
		assert.Equal(t, createDigicertRequest.DigiCert.APIToken, *createdCA.APIToken)
		require.NotNil(t, createdCA.ProfileID)
		assert.Equal(t, createDigicertRequest.DigiCert.ProfileID, *createdCA.ProfileID)
		require.NotNil(t, createdCA.CertificateCommonName)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateCommonName, *createdCA.CertificateCommonName)
		assert.Nil(t, createdCA.CertificateUserPrincipalNames)
		require.NotNil(t, createdCA.CertificateSeatID)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateSeatID, *createdCA.CertificateSeatID)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create Hydrant CA - Happy path", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createHydrantRequest := fleet.CertificateAuthorityPayload{
			Hydrant: &fleet.HydrantCA{
				Name:         "HydrantWIFI",
				URL:          mockHydrantServer.URL,
				ClientID:     "hydrant_client_id",
				ClientSecret: "hydrant_client_secret",
			},
		}

		_, err := svc.NewCertificateAuthority(ctx, createHydrantRequest)
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		assert.Equal(t, createHydrantRequest.Hydrant.Name, createdCA.Name)
		assert.Equal(t, createHydrantRequest.Hydrant.URL, createdCA.URL)
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

		assert.Equal(t, createCustomSCEPRequest.CustomSCEPProxy.Name, createdCA.Name)
		assert.Equal(t, createCustomSCEPRequest.CustomSCEPProxy.URL, createdCA.URL)
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

		assert.Equal(t, "NDES", createdCA.Name)
		assert.Equal(t, createNDESSCEPRequest.NDESSCEPProxy.URL, createdCA.URL)
		assert.Equal(t, string(fleet.CATypeNDESSCEPProxy), createdCA.Type)
		require.NotNil(t, createdCA.AdminURL)
		assert.Equal(t, createNDESSCEPRequest.NDESSCEPProxy.AdminURL, *createdCA.AdminURL)
		require.NotNil(t, createdCA.Username)
		assert.Equal(t, createNDESSCEPRequest.NDESSCEPProxy.Username, *createdCA.Username)
		require.NotNil(t, createdCA.Password)
		assert.Equal(t, createNDESSCEPRequest.NDESSCEPProxy.Password, *createdCA.Password)
		verifyNilFieldsForType(t, createdCA)
	})

	t.Run("Create DigiCert CA - Bad name", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "Digicert WIFI",
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

	t.Run("Create DigiCert CA - empty CN", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "Digicert WIFI",
				URL:                           mockDigiCertServer.URL,
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
				URL:                           mockDigiCertServer.URL,
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
				URL:                           mockDigiCertServer.URL + "/invalid",
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
				URL:                           mockDigiCertServer.URL,
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
				URL:                           mockDigiCertServer.URL,
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
				URL:                           mockDigiCertServer.URL,
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
				URL:                           mockDigiCertServer.URL,
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
}

// TODO: Revisit this test, as it seems rather useless (at least the success case) due to it's simplicity
// and not being possible atm. to mock/call free service methods.
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
