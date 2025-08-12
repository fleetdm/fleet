package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	scep_mock "github.com/fleetdm/fleet/v4/server/mock/scep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	createdCAs := []*fleet.CertificateAuthority{}
	baseSetupForCATests := func() (*Service, context.Context) {
		ds := new(mock.Store)
		// Reset createdCAs before each test
		createdCAs = []*fleet.CertificateAuthority{}
		// Setup DS mocks
		ds.NewCertificateAuthorityFunc = func(ctx context.Context, ca *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
			ca.ID = 1
			return ca, nil
		}
		authorizer, err := authz.NewAuthorizer()
		require.NoError(t, err)

		svc := &Service{
			logger:          log.NewLogfmtLogger(os.Stdout),
			ds:              ds,
			authz:           authorizer,
			digiCertService: digicert.NewService(),
			scepConfigService: &scep_mock.SCEPConfigService{
				ValidateSCEPURLFunc:          func(_ context.Context, _ string) error { return nil },
				ValidateNDESSCEPAdminURLFunc: func(_ context.Context, _ fleet.NDESSCEPProxyCA) error { return nil },
			},
		}
		svc.config.Server.PrivateKey = "supersecret"

		// TODO EJM figure out why this is failign the authz check
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		return svc, ctx
	}

	t.Run("Create DigiCert CA", func(t *testing.T) {
		svc, ctx := baseSetupForCATests()

		createDigicertRequest := fleet.CertificateAuthorityPayload{
			DigiCert: &fleet.DigiCertCA{
				Name:                          "WIFI",
				URL:                           mockDigiCertServer.URL,
				APIToken:                      "api_token",
				ProfileID:                     "profile_id",
				CertificateCommonName:         "common_name",
				CertificateUserPrincipalNames: []string{"user_principal_name"},
				CertificateSeatID:             "seat_id",
			},
		}

		ca, err := svc.NewCertificateAuthority(ctx, createDigicertRequest)
		require.NoError(t, err)
		require.NotNil(t, ca)
		require.Len(t, createdCAs, 1)
		createdCA := createdCAs[0]

		// Should have returned the new CA's ID
		require.Equal(t, 1, createdCA.ID)
		assert.Equal(t, createDigicertRequest.DigiCert.Name, createdCA.Name)
		assert.Equal(t, createDigicertRequest.DigiCert.URL, createdCA.URL)
		assert.Equal(t, fleet.CATypeDigiCert, createdCA.Type)
		require.NotNil(t, createdCA.APIToken)
		assert.Equal(t, createDigicertRequest.DigiCert.APIToken, *createdCA.APIToken)
		require.NotNil(t, createdCA.ProfileID)
		assert.Equal(t, createDigicertRequest.DigiCert.ProfileID, *createdCA.ProfileID)
		require.NotNil(t, createdCA.CertificateCommonName)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateCommonName, *createdCA.CertificateCommonName)
		assert.ElementsMatch(t, createDigicertRequest.DigiCert.CertificateUserPrincipalNames, createdCA.CertificateUserPrincipalNames)
		require.NotNil(t, createdCA.CertificateSeatID)
		assert.Equal(t, createDigicertRequest.DigiCert.CertificateSeatID, *createdCA.CertificateSeatID)
	})
}
