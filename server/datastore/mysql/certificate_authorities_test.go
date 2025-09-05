package mysql

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestCertificateAuthority(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetCertificateAuthorityByID", testGetCertificateAuthorityByID},
		{"GetAllCertificateAuthorities", testGetAllCertificateAuthorities},
		{"ListCertificateAuthorities", testListCertificateAuthorities},
		{"CreateCertificateAuthority", testCreateCertificateAuthority},
		{"Delete", testDeleteCertificateAuthority},
		{"UpdateCertificateAuthorityByID", testUpdateCertificateAuthorityByID},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func compareCA(t *testing.T, expected, actual *fleet.CertificateAuthority, expectSecrets bool) {
	if expected.ID != 0 {
		require.Equal(t, expected.ID, actual.ID)
	}
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.URL, actual.URL)
	require.Equal(t, expected.Type, actual.Type)

	// Digicert specific non-secret fields
	if expected.ProfileID != nil {
		require.NotNil(t, actual.ProfileID)
		require.Equal(t, *expected.ProfileID, *actual.ProfileID)
	} else {
		require.Nil(t, actual.ProfileID)
	}
	if expected.CertificateCommonName != nil {
		require.NotNil(t, actual.CertificateCommonName)
		require.Equal(t, *expected.CertificateCommonName, *actual.CertificateCommonName)
	} else {
		require.Nil(t, actual.CertificateCommonName)
	}
	if expected.CertificateUserPrincipalNames != nil {
		require.ElementsMatch(t, *expected.CertificateUserPrincipalNames, *actual.CertificateUserPrincipalNames)
	} else {
		require.Nil(t, actual.CertificateUserPrincipalNames)
	}
	if expected.CertificateSeatID != nil {
		require.NotNil(t, actual.CertificateSeatID)
		require.Equal(t, *expected.CertificateSeatID, *actual.CertificateSeatID)
	} else {
		require.Nil(t, actual.CertificateSeatID)
	}

	// NDES specific non-secret fields
	if expected.AdminURL != nil {
		require.NotNil(t, actual.AdminURL)
		require.Equal(t, *expected.AdminURL, *actual.AdminURL)
	} else {
		require.Nil(t, actual.AdminURL)
	}
	if expected.Username != nil {
		require.NotNil(t, actual.Username)
		require.Equal(t, *expected.Username, *actual.Username)
	} else {
		require.Nil(t, actual.Username)
	}

	// Hydrant specific non-secret fields
	if expected.ClientID != nil {
		require.NotNil(t, actual.ClientID)
		require.Equal(t, *expected.ClientID, *actual.ClientID)
	} else {
		require.Nil(t, actual.ClientID)
	}

	if expectSecrets {
		if expected.APIToken != nil {
			require.NotNil(t, actual.APIToken)
			require.Equal(t, *expected.APIToken, *actual.APIToken)
		} else {
			require.Nil(t, actual.APIToken)
		}
		if expected.Password != nil {
			require.NotNil(t, actual.Password)
			require.Equal(t, *expected.Password, *actual.Password)
		} else {
			require.Nil(t, actual.Password)
		}
		if expected.Challenge != nil {
			require.NotNil(t, actual.Challenge)
			require.Equal(t, *expected.Challenge, *actual.Challenge)
		} else {
			require.Nil(t, actual.Challenge)
		}
		if expected.ClientSecret != nil {
			require.NotNil(t, actual.ClientSecret)
			require.Equal(t, *expected.ClientSecret, *actual.ClientSecret)
		} else {
			require.Nil(t, actual.ClientSecret)
		}
	} else {
		if expected.APIToken != nil {
			require.NotNil(t, actual.APIToken)
			require.Equal(t, fleet.MaskedPassword, *actual.APIToken)
		} else {
			require.Nil(t, actual.APIToken)
		}
		if expected.Password != nil {
			require.NotNil(t, actual.Password)
			require.Equal(t, fleet.MaskedPassword, *actual.Password)
		} else {
			require.Nil(t, actual.Password)
		}
		if expected.Challenge != nil {
			require.NotNil(t, actual.Challenge)
			require.Equal(t, fleet.MaskedPassword, *actual.Challenge)
		} else {
			require.Nil(t, actual.Challenge)
		}
		if expected.ClientSecret != nil {
			require.NotNil(t, actual.ClientSecret)
			require.Equal(t, fleet.MaskedPassword, *actual.ClientSecret)
		} else {
			require.Nil(t, actual.ClientSecret)
		}
	}
}

func testCreateCertificateAuthority(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	digicertCA1 := &fleet.CertificateAuthority{
		Name:                          ptr.String("Test Digicert CA"),
		URL:                           ptr.String("https://digicert1.example.com"),
		Type:                          string(fleet.CATypeDigiCert),
		APIToken:                      ptr.String("test-api-token"),
		ProfileID:                     ptr.String("test-profile-id"),
		CertificateCommonName:         ptr.String("test-common-name $FLEET_VAR_HOST_HARDWARE_SERIAL"),
		CertificateUserPrincipalNames: &[]string{"test-upn $FLEET_VAR_HOST_HARDWARE_SERIAL"},
		CertificateSeatID:             ptr.String("test-seat-id"),
	}

	digicertCA2 := &fleet.CertificateAuthority{
		Name:                          ptr.String("Test Digicert CA 2"),
		URL:                           ptr.String("https://digicert2.example.com"),
		Type:                          string(fleet.CATypeDigiCert),
		APIToken:                      ptr.String("test-api-token2"),
		ProfileID:                     ptr.String("test-profile-id2"),
		CertificateCommonName:         ptr.String("test-common-name2 $FLEET_VAR_HOST_HARDWARE_SERIAL"),
		CertificateUserPrincipalNames: &[]string{"test-upn2 $FLEET_VAR_HOST_HARDWARE_SERIAL"},
		CertificateSeatID:             ptr.String("test-seat-id2"),
	}

	hydrantCA1 := &fleet.CertificateAuthority{
		Name:         ptr.String("Hydrant CA"),
		URL:          ptr.String("https://hydrant1.example.com"),
		Type:         string(fleet.CATypeHydrant),
		ClientID:     ptr.String("hydrant-client-id"),
		ClientSecret: ptr.String("hydrant-client-secret"),
	}

	hydrantCA2 := &fleet.CertificateAuthority{
		Name:         ptr.String("Hydrant CA 2"),
		URL:          ptr.String("https://hydrant2.example.com"),
		Type:         string(fleet.CATypeHydrant),
		ClientID:     ptr.String("hydrant-client-id2"),
		ClientSecret: ptr.String("hydrant-client-secret2"),
	}

	// Custom SCEP CAs
	customSCEPCA1 := &fleet.CertificateAuthority{
		Name:      ptr.String("Custom SCEP CA"),
		URL:       ptr.String("https://custom-scep.example.com"),
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Challenge: ptr.String("custom-scep-challenge"),
	}
	customSCEPCA2 := &fleet.CertificateAuthority{
		Name:      ptr.String("Custom SCEP CA 2"),
		URL:       ptr.String("https://custom-scep2.example.com"),
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Challenge: ptr.String("custom-scep-challenge2"),
	}

	// NDES CA
	ndesCA := &fleet.CertificateAuthority{
		Name:     ptr.String("NDES"),
		URL:      ptr.String("https://ndes.example.com"),
		AdminURL: ptr.String("https://ndes-admin.example.com"),
		Type:     string(fleet.CATypeNDESSCEPProxy),
		Username: ptr.String("ndes-username"),
		Password: ptr.String("ndes-password"),
	}

	casToCreate := []*fleet.CertificateAuthority{
		digicertCA1,
		digicertCA2,
		hydrantCA1,
		hydrantCA2,
		customSCEPCA1,
		customSCEPCA2,
		ndesCA,
	}

	for _, ca := range casToCreate {
		createdCA, err := ds.NewCertificateAuthority(ctx, ca)
		require.NoError(t, err)
		require.NotNil(t, createdCA)
		ca.ID = createdCA.ID

		// Try to create the same CA again, it should return an error
		_, err = ds.NewCertificateAuthority(ctx, ca)
		require.ErrorAs(t, err, &fleet.ConflictError{})

		// Get the CA and ensure it matches with and without secrets
		retrievedCA, err := ds.GetCertificateAuthorityByID(ctx, ca.ID, true)
		require.NoError(t, err)
		compareCA(t, ca, retrievedCA, true)

		retrievedCANoSecrets, err := ds.GetCertificateAuthorityByID(ctx, ca.ID, false)
		require.NoError(t, err)
		compareCA(t, ca, retrievedCANoSecrets, false)
	}

	// List all CAs and ensure they match
	allCASummaries, err := ds.ListCertificateAuthorities(ctx)
	require.NoError(t, err)
	require.Len(t, allCASummaries, len(casToCreate))
	slices.SortFunc(allCASummaries, func(a, b *fleet.CertificateAuthoritySummary) int {
		return cmp.Compare(a.ID, b.ID)
	})

	allCAsWithSecrets, err := ds.GetAllCertificateAuthorities(ctx, true)
	require.NoError(t, err)
	require.Len(t, allCAsWithSecrets, len(casToCreate))
	slices.SortFunc(allCAsWithSecrets, func(a, b *fleet.CertificateAuthority) int {
		return cmp.Compare(a.ID, b.ID)
	})

	allCAsWithoutSecrets, err := ds.GetAllCertificateAuthorities(ctx, false)
	require.NoError(t, err)
	require.Len(t, allCAsWithoutSecrets, len(casToCreate))
	slices.SortFunc(allCAsWithoutSecrets, func(a, b *fleet.CertificateAuthority) int {
		return cmp.Compare(a.ID, b.ID)
	})

	for i, ca := range casToCreate {
		// Ensure the CA summary matches
		require.Equal(t, ca.ID, allCASummaries[i].ID)
		require.Equal(t, *ca.Name, allCASummaries[i].Name)
		require.Equal(t, ca.Type, allCASummaries[i].Type)
	}
}

// GetAllCertificateAuthorities tests that make less sense to have in testCreateCertificateAuthority
func testGetAllCertificateAuthorities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// list is empty
	cas, err := ds.GetAllCertificateAuthorities(ctx, true)
	require.NoError(t, err)
	require.Empty(t, cas)

	// testCreateCertificateAuthority will create several CAs and test this method more thoroughly
}

// GetCertificateAuthorityByID tests that make less sense to have in testCreateCertificateAuthority
func testGetCertificateAuthorityByID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// get unknown CA
	id := uint(9999)
	_, err := ds.GetCertificateAuthorityByID(ctx, id, true)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// same without secrets
	_, err = ds.GetCertificateAuthorityByID(ctx, id, false)
	require.ErrorAs(t, err, &nfe)

	// testCreateCertificateAuthority will create several CAs and test this get method more thoroughly
}

// ListCertificateAuthorities tests that make less sense to have in testCreateCertificateAuthority
func testListCertificateAuthorities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// list is empty
	cas, err := ds.ListCertificateAuthorities(ctx)
	require.NoError(t, err)
	// Should return an empty list, not nil
	require.NotNil(t, cas)
	require.Empty(t, cas)

	// testCreateCertificateAuthority will create several CAs and test this list method more thoroughly
}

func testDeleteCertificateAuthority(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	ca, err := ds.GetCertificateAuthorityByID(ctx, 1, true)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, ca)

	ca, err = ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type: string(fleet.CATypeHydrant),
		Name: ptr.String("Hydrant CA"),
		URL:  ptr.String("https://localhost"),
	})
	require.NoError(t, err)
	require.NotNil(t, ca)

	ca, err = ds.GetCertificateAuthorityByID(ctx, ca.ID, true)
	require.NoError(t, err)
	require.NotNil(t, ca)

	_, err = ds.DeleteCertificateAuthority(ctx, ca.ID)
	require.NoError(t, err)

	deletedCA, err := ds.GetCertificateAuthorityByID(ctx, ca.ID, true)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, deletedCA)

	_, err = ds.DeleteCertificateAuthority(ctx, ca.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func testUpdateCertificateAuthorityByID(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	digicertCA1 := &fleet.CertificateAuthority{
		Name:                          ptr.String("Test Digicert CA"),
		URL:                           ptr.String("https://digicert1.example.com"),
		Type:                          string(fleet.CATypeDigiCert),
		APIToken:                      ptr.String("test-api-token"),
		ProfileID:                     ptr.String("test-profile-id"),
		CertificateCommonName:         ptr.String("test-common-name $FLEET_VAR_HOST_HARDWARE_SERIAL"),
		CertificateUserPrincipalNames: &[]string{"test-upn $FLEET_VAR_HOST_HARDWARE_SERIAL"},
		CertificateSeatID:             ptr.String("test-seat-id"),
	}

	hydrantCA1 := &fleet.CertificateAuthority{
		Name:         ptr.String("Hydrant CA"),
		URL:          ptr.String("https://hydrant1.example.com"),
		Type:         string(fleet.CATypeHydrant),
		ClientID:     ptr.String("hydrant-client-id"),
		ClientSecret: ptr.String("hydrant-client-secret"),
	}

	// Custom SCEP CAs
	customSCEPCA1 := &fleet.CertificateAuthority{
		Name:      ptr.String("Custom SCEP CA"),
		URL:       ptr.String("https://custom-scep.example.com"),
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Challenge: ptr.String("custom-scep-challenge"),
	}

	// NDES CA
	ndesCA1 := &fleet.CertificateAuthority{
		Name:     ptr.String("NDES"),
		URL:      ptr.String("https://ndes.example.com"),
		AdminURL: ptr.String("https://ndes-admin.example.com"),
		Type:     string(fleet.CATypeNDESSCEPProxy),
		Username: ptr.String("ndes-username"),
		Password: ptr.String("ndes-password"),
	}

	casToCreate := []*fleet.CertificateAuthority{
		digicertCA1,
		hydrantCA1,
		customSCEPCA1,
		ndesCA1,
	}

	caMap := map[fleet.CAType]*fleet.CertificateAuthority{}

	for _, ca := range casToCreate {
		// Create one of each type of CA
		createdCA, err := ds.NewCertificateAuthority(ctx, ca)
		require.NoError(t, err)
		caMap[fleet.CAType(createdCA.Type)] = createdCA
	}

	t.Run("fails if certificate authority is not found", func(t *testing.T) {
		err := ds.UpdateCertificateAuthorityByID(ctx, uint(999), &fleet.CertificateAuthority{})
		var nfe fleet.NotFoundError
		require.ErrorAs(t, err, &nfe)
	})

	t.Run("successfully updates digicert CA", func(t *testing.T) {
		digicertCA := caMap[fleet.CATypeDigiCert]

		digicertCA.Name = ptr.String("New Digicert")
		digicertCA.URL = ptr.String("https://localhost")
		digicertCA.APIToken = ptr.String("my-new-api-token")
		digicertCA.ProfileID = ptr.String("updated-profile-id")
		digicertCA.CertificateCommonName = ptr.String("updated certificate common name")
		upns := []string{"updated-upns"}
		digicertCA.CertificateUserPrincipalNames = &upns
		digicertCA.CertificateSeatID = ptr.String("updated-seat-id")

		err := ds.UpdateCertificateAuthorityByID(ctx, digicertCA.ID, digicertCA)
		require.NoError(t, err)

		updatedCA, err := ds.GetCertificateAuthorityByID(ctx, digicertCA.ID, true)
		require.NoError(t, err)

		require.Equal(t, "New Digicert", *updatedCA.Name)
		require.Equal(t, "https://localhost", *updatedCA.URL)
		require.Equal(t, "my-new-api-token", *updatedCA.APIToken)
		require.Equal(t, "updated-profile-id", *updatedCA.ProfileID)
		require.Equal(t, "updated certificate common name", *updatedCA.CertificateCommonName)
		require.ElementsMatch(t, []string{"updated-upns"}, *updatedCA.CertificateUserPrincipalNames)
		require.Equal(t, "updated-seat-id", *updatedCA.CertificateSeatID)
	})

	t.Run("successfully updates Hydrant CA", func(t *testing.T) {
		hydrantCA := caMap[fleet.CATypeHydrant]

		hydrantCA.Name = ptr.String("updated Hydrant")
		hydrantCA.URL = ptr.String("https://localhost")
		hydrantCA.ClientID = ptr.String("updated-client-id")
		hydrantCA.ClientSecret = ptr.String("updated-client-secret")

		err := ds.UpdateCertificateAuthorityByID(ctx, hydrantCA.ID, hydrantCA)
		require.NoError(t, err)

		updatedCA, err := ds.GetCertificateAuthorityByID(ctx, hydrantCA.ID, true)
		require.NoError(t, err)

		require.Equal(t, "updated Hydrant", *updatedCA.Name)
		require.Equal(t, "https://localhost", *updatedCA.URL)
		require.Equal(t, "updated-client-id", *updatedCA.ClientID)
		require.Equal(t, "updated-client-secret", *updatedCA.ClientSecret)
	})

	t.Run("successfully updates ndes scep proxy CA", func(t *testing.T) {
		ndesCA := caMap[fleet.CATypeNDESSCEPProxy]

		ndesCA.Name = ptr.String("Updated NDES")

		ndesCA.URL = ptr.String("https://localhost")
		ndesCA.AdminURL = ptr.String("https://admin.localhost")
		ndesCA.Username = ptr.String("admin")
		ndesCA.Password = ptr.String("password")

		err := ds.UpdateCertificateAuthorityByID(ctx, ndesCA.ID, ndesCA)
		require.NoError(t, err)

		updatedCA, err := ds.GetCertificateAuthorityByID(ctx, ndesCA.ID, true)
		require.NoError(t, err)

		require.Equal(t, "NDES", *updatedCA.Name)
		require.Equal(t, "https://localhost", *updatedCA.URL)
		require.Equal(t, "https://admin.localhost", *updatedCA.AdminURL)
		require.Equal(t, "admin", *updatedCA.Username)
		require.Equal(t, "password", *updatedCA.Password)
	})

	t.Run("successfully updates custom SCEP proxy CA", func(t *testing.T) {
		scepCA := caMap[fleet.CATypeCustomSCEPProxy]

		scepCA.Name = ptr.String("Updated SCEP")
		scepCA.URL = ptr.String("https://localhost")
		scepCA.Challenge = ptr.String("updated-challenge")

		err := ds.UpdateCertificateAuthorityByID(ctx, scepCA.ID, scepCA)
		require.NoError(t, err)

		updatedCA, err := ds.GetCertificateAuthorityByID(ctx, scepCA.ID, true)
		require.NoError(t, err)

		require.Equal(t, "Updated SCEP", *updatedCA.Name)
		require.Equal(t, "https://localhost", *updatedCA.URL)
		require.Equal(t, "updated-challenge", *updatedCA.Challenge)
	})
}

// // TODO(hca): refactor old app config test to cover new implementations
// func testNDESSCEPProxyPassword(t *testing.T, ds *Datastore) {
// 	ctx := context.Background()
// 	ctx = ctxdb.BypassCachedMysql(ctx, true)
// 	defer TruncateTables(t, ds)

// 	ac, err := ds.AppConfig(ctx)
// 	require.NoError(t, err)

// 	adminURL := "https://localhost:8080/mscep_admin/"
// 	username := "admin"
// 	url := "https://localhost:8080/mscep/mscep.dll"
// 	password := "password"

// 	ac.Integrations.NDESSCEPProxy = optjson.Any[fleet.NDESSCEPProxyCA]{
// 		Valid: true,
// 		Set:   true,
// 		Value: fleet.NDESSCEPProxyCA{
// 			AdminURL: adminURL,
// 			Username: username,
// 			Password: password,
// 			URL:      url,
// 		},
// 	}

// 	err = ds.SaveAppConfig(ctx, ac)
// 	require.NoError(t, err)

// 	checkProxyConfig := func() {
// 		result, err := ds.AppConfig(ctx)
// 		require.NoError(t, err)
// 		require.NotNil(t, result.Integrations.NDESSCEPProxy)
// 		assert.Equal(t, url, result.Integrations.NDESSCEPProxy.Value.URL)
// 		assert.Equal(t, adminURL, result.Integrations.NDESSCEPProxy.Value.AdminURL)
// 		assert.Equal(t, username, result.Integrations.NDESSCEPProxy.Value.Username)
// 		assert.Equal(t, fleet.MaskedPassword, result.Integrations.NDESSCEPProxy.Value.Password)
// 	}

// 	checkProxyConfig()

// 	checkPassword := func() {
// 		assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetNDESPassword}, nil)
// 		require.NoError(t, err)
// 		require.Len(t, assets, 1)
// 		assert.Equal(t, password, string(assets[fleet.MDMAssetNDESPassword].Value))
// 	}
// 	checkPassword()

// 	// Set password to masked password -- should not update
// 	ac.Integrations.NDESSCEPProxy.Value.Password = fleet.MaskedPassword
// 	err = ds.SaveAppConfig(ctx, ac)
// 	require.NoError(t, err)
// 	checkProxyConfig()
// 	checkPassword()

// 	// Set password to empty -- password should not update
// 	url = "https://newurl.com"
// 	ac.Integrations.NDESSCEPProxy.Value.Password = ""
// 	ac.Integrations.NDESSCEPProxy.Value.URL = url
// 	err = ds.SaveAppConfig(ctx, ac)
// 	require.NoError(t, err)
// 	checkProxyConfig()
// 	checkPassword()

// 	// Set password to a new value
// 	password = "newpassword"
// 	ac.Integrations.NDESSCEPProxy.Value.Password = password
// 	err = ds.SaveAppConfig(ctx, ac)
// 	require.NoError(t, err)
// 	checkProxyConfig()
// 	checkPassword()
// }
