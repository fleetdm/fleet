package mysql

import (
	"cmp"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"EditResendsProfiles", testEditResendsProfiles},
		{"EJBCANamesAreUnique", testEJBCANamesAreUnique},
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

	// EJBCA specific non-secret fields
	for _, field := range []struct{ expected, actual *string }{
		{expected.MTLSCertPEM, actual.MTLSCertPEM},
		{expected.TrustCABundlePEM, actual.TrustCABundlePEM},
		{expected.EJBCACAName, actual.EJBCACAName},
		{expected.EJBCACertificateProfileName, actual.EJBCACertificateProfileName},
		{expected.EJBCAEndEntityProfileName, actual.EJBCAEndEntityProfileName},
		{expected.EJBCAUsernameTemplate, actual.EJBCAUsernameTemplate},
	} {
		require.Equal(t, field.expected, field.actual)
	}
	if expected.MTLSCertExpiresAt != nil {
		require.NotNil(t, actual.MTLSCertExpiresAt)
		require.True(t, expected.MTLSCertExpiresAt.Equal(*actual.MTLSCertExpiresAt))
	} else {
		require.Nil(t, actual.MTLSCertExpiresAt)
	}

	// Google CAS specific non-secret fields
	for _, field := range []struct{ expected, actual *string }{
		{expected.GoogleCASProject, actual.GoogleCASProject},
		{expected.GoogleCASLocation, actual.GoogleCASLocation},
		{expected.GoogleCASCAPool, actual.GoogleCASCAPool},
	} {
		require.Equal(t, field.expected, field.actual)
	}

	if expectSecrets {
		require.Equal(t, expected.GoogleCASAPIKeyJSON, actual.GoogleCASAPIKeyJSON)
		if expected.MTLSKeyPEM != nil {
			require.NotNil(t, actual.MTLSKeyPEM)
			require.Equal(t, *expected.MTLSKeyPEM, *actual.MTLSKeyPEM)
		} else {
			require.Nil(t, actual.MTLSKeyPEM)
		}
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
		if expected.GoogleCASAPIKeyJSON != nil {
			require.NotNil(t, actual.GoogleCASAPIKeyJSON)
			require.Equal(t, fleet.MaskedPassword, *actual.GoogleCASAPIKeyJSON)
		} else {
			require.Nil(t, actual.GoogleCASAPIKeyJSON)
		}
		if expected.MTLSKeyPEM != nil {
			require.NotNil(t, actual.MTLSKeyPEM)
			require.Equal(t, fleet.MaskedPassword, *actual.MTLSKeyPEM)
		} else {
			require.Nil(t, actual.MTLSKeyPEM)
		}
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

	estCA1 := &fleet.CertificateAuthority{
		Name:     ptr.String("Custom EST CA"),
		URL:      ptr.String("http://customcerts.example.com"),
		Type:     string(fleet.CATypeCustomESTProxy),
		Username: ptr.String("custom-est-username"),
		Password: ptr.String("custom-est-password"),
	}
	estCA2 := &fleet.CertificateAuthority{
		Name:     ptr.String("Custom EST CA 2"),
		URL:      ptr.String("http://customcerts2.example.com"),
		Type:     string(fleet.CATypeCustomESTProxy),
		Username: ptr.String("custom-est-username2"),
		Password: ptr.String("custom-est-password2"),
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

	// Smallstep CA
	smallstepCA1 := &fleet.CertificateAuthority{
		Name:         ptr.String("Smallstep SCEP CA"),
		URL:          ptr.String("https://smallstep-scep.example.com"),
		Type:         string(fleet.CATypeSmallstep),
		ChallengeURL: ptr.String("https://smallstep-scep.example.com/challenge"),
		Username:     ptr.String("smallstep-username"),
		Password:     ptr.String("smallstep-password"),
	}
	smallstepCA2 := &fleet.CertificateAuthority{
		Name:         ptr.String("Smallstep SCEP CA 2"),
		URL:          ptr.String("https://smallstep-scep2.example.com"),
		Type:         string(fleet.CATypeSmallstep),
		ChallengeURL: ptr.String("https://smallstep-scep2.example.com/challenge"),
		Username:     ptr.String("smallstep-username2"),
		Password:     ptr.String("smallstep-password2"),
	}

	// EJBCA CAs, one with all the optional fields
	ejbcaNotAfter := time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Second)
	ejbcaCert := newMTLSCertPEM(t, ejbcaNotAfter)
	ejbcaCA1 := &fleet.CertificateAuthority{
		Name:                          new("EJBCA CA"),
		URL:                           new("https://ejbca1.example.com:8443"),
		Type:                          string(fleet.CATypeEJBCA),
		MTLSCertPEM:                   &ejbcaCert,
		MTLSKeyPEM:                    new("ejbca-mtls-key"),
		TrustCABundlePEM:              new("ejbca-trust-bundle"),
		EJBCACAName:                   new("ManagementCA"),
		EJBCACertificateProfileName:   new("certificate-profile"),
		EJBCAEndEntityProfileName:     new("end-entity-profile"),
		EJBCAUsernameTemplate:         new("$FLEET_VAR_HOST_HARDWARE_SERIAL"),
		CertificateCommonName:         new("$FLEET_VAR_HOST_END_USER_EMAIL_IDP"),
		CertificateUserPrincipalNames: &[]string{"$FLEET_VAR_HOST_END_USER_EMAIL_IDP"},
		MTLSCertExpiresAt:             &ejbcaNotAfter,
	}
	ejbcaCA2 := &fleet.CertificateAuthority{
		Name:                        new("EJBCA CA 2"),
		URL:                         new("https://ejbca2.example.com:8443"),
		Type:                        string(fleet.CATypeEJBCA),
		MTLSCertPEM:                 &ejbcaCert,
		MTLSKeyPEM:                  new("ejbca-mtls-key2"),
		EJBCACAName:                 new("ManagementCA"),
		EJBCACertificateProfileName: new("certificate-profile2"),
		EJBCAEndEntityProfileName:   new("end-entity-profile2"),
		CertificateCommonName:       new("$FLEET_VAR_HOST_UUID"), // no username template, so it defaults to the common name
		MTLSCertExpiresAt:           &ejbcaNotAfter,
	}

	// Google CAS CAs. The url column is NOT NULL, so the type stores it empty.
	googleCASCA1 := &fleet.CertificateAuthority{
		Name:                new("Google CAS CA"),
		URL:                 new(""),
		Type:                string(fleet.CATypeGoogleCAS),
		GoogleCASProject:    new("my-project"),
		GoogleCASLocation:   new("us-central1"),
		GoogleCASCAPool:     new("my-ca-pool"),
		GoogleCASAPIKeyJSON: new(`{"type":"service_account","client_email":"sa@my-project.iam.gserviceaccount.com","private_key":"key"}`),
	}
	googleCASCA2 := &fleet.CertificateAuthority{
		Name:                new("Google CAS CA 2"),
		URL:                 new(""),
		Type:                string(fleet.CATypeGoogleCAS),
		GoogleCASProject:    new("my-project-2"),
		GoogleCASLocation:   new("europe-west1"),
		GoogleCASCAPool:     new("my-ca-pool-2"),
		GoogleCASAPIKeyJSON: new(`{"type":"service_account","client_email":"sa2@my-project.iam.gserviceaccount.com","private_key":"key2"}`),
	}

	casToCreate := []*fleet.CertificateAuthority{
		digicertCA1,
		digicertCA2,
		hydrantCA1,
		hydrantCA2,
		estCA1,
		estCA2,
		customSCEPCA1,
		customSCEPCA2,
		ndesCA,
		smallstepCA1,
		smallstepCA2,
		ejbcaCA1,
		ejbcaCA2,
		googleCASCA1,
		googleCASCA2,
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
		// Only EJBCA entries carry the client certificate expiry.
		if ca.Type == string(fleet.CATypeEJBCA) {
			require.NotNil(t, allCASummaries[i].MTLSCertExpiresAt)
			require.True(t, ca.MTLSCertExpiresAt.Equal(*allCASummaries[i].MTLSCertExpiresAt))
		} else {
			require.Nil(t, allCASummaries[i].MTLSCertExpiresAt)
		}
		require.Nil(t, allCASummaries[i].MTLSCertPEM)

		// Ensure the full CAs match with and without secrets
		compareCA(t, ca, allCAsWithSecrets[i], true)
		compareCA(t, ca, allCAsWithoutSecrets[i], false)
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

	// Test that deleting a CA referenced by a certificate template returns a conflict error
	ca, err = ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type: string(fleet.CATypeCustomSCEPProxy),
		Name: ptr.String("Referenced CA"),
		URL:  ptr.String("https://localhost"),
	})
	require.NoError(t, err)
	require.NotNil(t, ca)

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "cert-test-team"})
	require.NoError(t, err)

	// Insert a certificate template that references the CA
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO certificate_templates (team_id, certificate_authority_id, name, subject_name) VALUES (?, ?, ?, ?)`,
		team.ID, ca.ID, "test-cert", "CN=test",
	)
	require.NoError(t, err)

	_, err = ds.DeleteCertificateAuthority(ctx, ca.ID)
	require.Error(t, err)
	var conflictErr fleet.ConflictError
	require.ErrorAs(t, err, &conflictErr)
	require.Contains(t, conflictErr.Error(), fleet.DeleteCAReferencedByTemplatesErrMsg)
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

	customESTCA := &fleet.CertificateAuthority{
		Name:     ptr.String("Custom EST CA"),
		URL:      ptr.String("https://estca.example.com"),
		Type:     string(fleet.CATypeCustomESTProxy),
		Username: ptr.String("custom-est-username"),
		Password: ptr.String("custom-est-password"),
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

	// Smallstep CA
	smallstepCA1 := &fleet.CertificateAuthority{
		Name:         ptr.String("Smallstep SCEP CA"),
		URL:          ptr.String("https://smallstep-scep.example.com"),
		Type:         string(fleet.CATypeSmallstep),
		ChallengeURL: ptr.String("https://smallstep-scep.example.com/challenge"),
		Username:     ptr.String("smallstep-username"),
		Password:     ptr.String("smallstep-password"),
	}

	// EJBCA CA
	originalNotAfter := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second)
	ejbcaCA1 := &fleet.CertificateAuthority{
		Name:                        new("EJBCA CA"),
		URL:                         new("https://ejbca.example.com:8443"),
		Type:                        string(fleet.CATypeEJBCA),
		MTLSCertPEM:                 new(newMTLSCertPEM(t, originalNotAfter)),
		MTLSKeyPEM:                  new("ejbca-mtls-key"),
		EJBCACAName:                 new("ManagementCA"),
		EJBCACertificateProfileName: new("certificate-profile"),
		EJBCAEndEntityProfileName:   new("end-entity-profile"),
		EJBCAUsernameTemplate:       new("$FLEET_VAR_HOST_HARDWARE_SERIAL"),
		CertificateCommonName:       new("$FLEET_VAR_HOST_END_USER_EMAIL_IDP"),
	}

	googleCASCA1 := &fleet.CertificateAuthority{
		Name:                new("Google CAS CA"),
		URL:                 new(""),
		Type:                string(fleet.CATypeGoogleCAS),
		GoogleCASProject:    new("my-project"),
		GoogleCASLocation:   new("us-central1"),
		GoogleCASCAPool:     new("my-ca-pool"),
		GoogleCASAPIKeyJSON: new(`{"type":"service_account"}`),
	}

	casToCreate := []*fleet.CertificateAuthority{
		digicertCA1,
		hydrantCA1,
		customESTCA,
		customSCEPCA1,
		ndesCA1,
		smallstepCA1,
		ejbcaCA1,
		googleCASCA1,
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

	t.Run("successfully updates custom est proxy CA", func(t *testing.T) {
		customESTCA := caMap[fleet.CATypeCustomESTProxy]

		customESTCA.Name = ptr.String("updated EST")
		customESTCA.URL = ptr.String("https://coolguy.localhost")
		customESTCA.Username = ptr.String("updated-username")
		customESTCA.Password = ptr.String("updated-password")

		err := ds.UpdateCertificateAuthorityByID(ctx, customESTCA.ID, customESTCA)
		require.NoError(t, err)

		updatedCA, err := ds.GetCertificateAuthorityByID(ctx, customESTCA.ID, true)
		require.NoError(t, err)

		require.Equal(t, "updated EST", *updatedCA.Name)
		require.Equal(t, "https://coolguy.localhost", *updatedCA.URL)
		require.Equal(t, "updated-username", *updatedCA.Username)
		require.Equal(t, "updated-password", *updatedCA.Password)
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

	t.Run("successfully updates EJBCA CA", func(t *testing.T) {
		ejbcaCA := caMap[fleet.CATypeEJBCA]

		// Only the fields that are set change, and a replaced mTLS certificate and key are stored and read back.
		newNotAfter := time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Second)
		newCert := newMTLSCertPEM(t, newNotAfter)
		update := &fleet.CertificateAuthority{
			Type:                  string(fleet.CATypeEJBCA),
			Name:                  new("Updated EJBCA"),
			MTLSCertPEM:           &newCert,
			MTLSKeyPEM:            new("updated-mtls-key"),
			TrustCABundlePEM:      new("updated-trust-bundle"),
			EJBCAUsernameTemplate: new("updated-username"),
			CertificateCommonName: new("updated-common-name"),
		}
		require.NoError(t, ds.UpdateCertificateAuthorityByID(ctx, ejbcaCA.ID, update))

		want := *ejbcaCA
		want.Name = update.Name
		want.MTLSCertPEM = update.MTLSCertPEM
		want.MTLSKeyPEM = update.MTLSKeyPEM
		want.TrustCABundlePEM = update.TrustCABundlePEM
		want.EJBCAUsernameTemplate = update.EJBCAUsernameTemplate
		want.CertificateCommonName = update.CertificateCommonName
		want.MTLSCertExpiresAt = &newNotAfter
		for _, includeSecrets := range []bool{true, false} {
			updatedCA, err := ds.GetCertificateAuthorityByID(ctx, ejbcaCA.ID, includeSecrets)
			require.NoError(t, err)
			compareCA(t, &want, updatedCA, includeSecrets)
		}

		// An mTLS certificate that can't be parsed is reported without an expiry instead of failing the read.
		require.NoError(t, ds.UpdateCertificateAuthorityByID(ctx, ejbcaCA.ID,
			&fleet.CertificateAuthority{Type: string(fleet.CATypeEJBCA), MTLSCertPEM: new("not a certificate")}))
		invalidCertCA, err := ds.GetCertificateAuthorityByID(ctx, ejbcaCA.ID, false)
		require.NoError(t, err)
		require.Nil(t, invalidCertCA.MTLSCertExpiresAt)

		// The expiry is of the first certificate, even when the key comes first, like the TLS client reads it.
		keyFirst := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("key")})) + newCert
		require.NoError(t, ds.UpdateCertificateAuthorityByID(ctx, ejbcaCA.ID,
			&fleet.CertificateAuthority{Type: string(fleet.CATypeEJBCA), MTLSCertPEM: &keyFirst}))
		keyFirstCA, err := ds.GetCertificateAuthorityByID(ctx, ejbcaCA.ID, false)
		require.NoError(t, err)
		require.NotNil(t, keyFirstCA.MTLSCertExpiresAt)
		require.True(t, newNotAfter.Equal(*keyFirstCA.MTLSCertExpiresAt))

		// Emptying an optional field stores NULL, as it is when the field isn't set on create, so a read leaves it out.
		require.NoError(t, ds.UpdateCertificateAuthorityByID(ctx, ejbcaCA.ID, &fleet.CertificateAuthority{
			Type: string(fleet.CATypeEJBCA), TrustCABundlePEM: new(""), EJBCAUsernameTemplate: new(""), CertificateUserPrincipalNames: &[]string{},
		}))
		cleared, err := ds.GetCertificateAuthorityByID(ctx, ejbcaCA.ID, false)
		require.NoError(t, err)
		require.Nil(t, cleared.TrustCABundlePEM)
		require.Nil(t, cleared.EJBCAUsernameTemplate)
		require.Nil(t, cleared.CertificateUserPrincipalNames)
	})

	t.Run("successfully updates Google CAS CA", func(t *testing.T) {
		googleCASCA := caMap[fleet.CATypeGoogleCAS]

		// A partial update keeps the fields it doesn't set.
		update := &fleet.CertificateAuthority{
			Type:                string(fleet.CATypeGoogleCAS),
			Name:                new("Updated Google CAS"),
			GoogleCASCAPool:     new("updated-ca-pool"),
			GoogleCASAPIKeyJSON: new(`{"type":"service_account","client_email":"new"}`),
		}
		require.NoError(t, ds.UpdateCertificateAuthorityByID(ctx, googleCASCA.ID, update))

		updatedCA, err := ds.GetCertificateAuthorityByID(ctx, googleCASCA.ID, true)
		require.NoError(t, err)
		require.Equal(t, "Updated Google CAS", *updatedCA.Name)
		require.Equal(t, "my-project", *updatedCA.GoogleCASProject)
		require.Equal(t, "us-central1", *updatedCA.GoogleCASLocation)
		require.Equal(t, "updated-ca-pool", *updatedCA.GoogleCASCAPool)
		require.Equal(t, *update.GoogleCASAPIKeyJSON, *updatedCA.GoogleCASAPIKeyJSON)

		maskedCA, err := ds.GetCertificateAuthorityByID(ctx, googleCASCA.ID, false)
		require.NoError(t, err)
		require.Equal(t, fleet.MaskedPassword, *maskedCA.GoogleCASAPIKeyJSON)
	})

	t.Run("successfully updates smallstep SCEP proxy CA", func(t *testing.T) {
		smallstepCA := caMap[fleet.CATypeSmallstep]

		smallstepCA.Name = ptr.String("Updated Smallstep")
		smallstepCA.URL = ptr.String("https://localhost")
		smallstepCA.ChallengeURL = ptr.String("https://localhost/challenge")
		smallstepCA.Username = ptr.String("updated-username")
		smallstepCA.Password = ptr.String("updated-password")

		err := ds.UpdateCertificateAuthorityByID(ctx, smallstepCA.ID, smallstepCA)
		require.NoError(t, err)

		updatedCA, err := ds.GetCertificateAuthorityByID(ctx, smallstepCA.ID, true)
		require.NoError(t, err)

		require.Equal(t, "Updated Smallstep", *updatedCA.Name)
		require.Equal(t, "https://localhost", *updatedCA.URL)
		require.Equal(t, "https://localhost/challenge", *updatedCA.ChallengeURL)
		require.Equal(t, "updated-username", *updatedCA.Username)
		require.Equal(t, "updated-password", *updatedCA.Password)
	})
}

func testEditResendsProfiles(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team"})
	require.NoError(t, err)
	host := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	teamHost := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHost.ID})))

	ca := newDigiCertCA(t, ds, "ca", "cn", "seat", &[]string{"upn"})
	otherCA := newDigiCertCA(t, ds, "ca_other", "cn", "seat", nil)
	prof := newDigiCertProfile(t, ds, "prof", "ca", 0)
	teamProf := newDigiCertProfile(t, ds, "team_prof", "ca", team.ID)
	otherProf := newDigiCertProfile(t, ds, "other_prof", "ca_other", 0)
	ejbcaCA := newEJBCACA(t, ds, "ejbca_ca", "cn", new("user"), &[]string{"upn"})
	ejbcaProf := newCAProfile(t, ds, fleet.FleetVarEJBCADataPrefix, "ejbca_prof", "ejbca_ca", 0)
	ejbcaTeamProf := newCAProfile(t, ds, fleet.FleetVarEJBCADataPrefix, "ejbca_team_prof", "ejbca_ca", team.ID)
	// A profile names its CA with the exact spelling, so this one is for the CA once the case of its name changes.
	ejbcaUpperCaseProf := newCAProfile(t, ds, fleet.FleetVarEJBCADataPrefix, "ejbca_upper_case_prof", "EJBCA_CA", 0)

	apiEdit := func(update fleet.CertificateAuthority) func() error {
		return func() error {
			update.Type = ca.Type
			return ds.UpdateCertificateAuthorityByID(ctx, ca.ID, &update)
		}
	}
	apiEditEJBCA := func(update fleet.CertificateAuthority) func() error {
		return func() error {
			update.Type = ejbcaCA.Type
			return ds.UpdateCertificateAuthorityByID(ctx, ejbcaCA.ID, &update)
		}
	}
	// gitOpsUpdate sends the full CA, as GitOps does, with mod applied to its current values.
	gitOpsUpdate := func(id uint, mod func(*fleet.CertificateAuthority)) error {
		current, err := ds.GetCertificateAuthorityByID(ctx, id, true)
		require.NoError(t, err)
		mod(current)
		return ds.BatchApplyCertificateAuthorities(ctx, fleet.CertificateAuthoritiesBatchOperations{Update: []*fleet.CertificateAuthority{current}})
	}
	cases := []struct {
		name string
		edit func() error
		// wantResend is whether the DigiCert profiles of ca are resent, wantEJBCAResend the same for the profiles of ejbcaCA.
		wantResend, wantEJBCAResend bool
		// wantUpperCaseResend is whether the profile that names the EJBCA CA in upper case is resent.
		wantUpperCaseResend bool
	}{
		{name: "API edit of the URL", edit: apiEdit(fleet.CertificateAuthority{URL: new("https://new.example.com")})},
		{name: "API edit to the same seat ID", edit: apiEdit(fleet.CertificateAuthority{CertificateSeatID: new("seat")})},
		{name: "API edit to the same profile ID", edit: apiEdit(fleet.CertificateAuthority{ProfileID: new("profile")})},
		{name: "API edit of the profile ID", edit: apiEdit(fleet.CertificateAuthority{ProfileID: new("new-profile")}), wantResend: true},
		{name: "API edit of the common name", edit: apiEdit(fleet.CertificateAuthority{CertificateCommonName: new("new-cn")}), wantResend: true},
		{
			name: "API edit of the UPN", edit: apiEdit(fleet.CertificateAuthority{CertificateUserPrincipalNames: &[]string{"new-upn"}}),
			wantResend: true,
		},
		{name: "GitOps edit of the API token", edit: func() error {
			return gitOpsUpdate(ca.ID, func(c *fleet.CertificateAuthority) { c.APIToken = new("new-token") })
		}},
		// otherCA was stored without UPNs, and GitOps sends an empty list.
		{name: "GitOps edit of another CA with an empty UPN list", edit: func() error {
			return gitOpsUpdate(otherCA.ID, func(c *fleet.CertificateAuthority) {
				c.APIToken = new("new-token")
				c.CertificateUserPrincipalNames = &[]string{}
			})
		}},
		{name: "GitOps edit of the seat ID", edit: func() error {
			return gitOpsUpdate(ca.ID, func(c *fleet.CertificateAuthority) { c.CertificateSeatID = new("new-seat") })
		}, wantResend: true},
		{
			name: "EJBCA API edit of the connection settings",
			edit: apiEditEJBCA(fleet.CertificateAuthority{
				URL: new("https://new.example.com"), TrustCABundlePEM: new("bundle"), MTLSCertPEM: new("new-cert"), MTLSKeyPEM: new("new-key"),
			}),
		},
		{name: "EJBCA API edit to the same username template", edit: apiEditEJBCA(fleet.CertificateAuthority{EJBCAUsernameTemplate: new("user")})},
		{name: "EJBCA API edit to the same common name", edit: apiEditEJBCA(fleet.CertificateAuthority{CertificateCommonName: new("cn")})},
		{
			name: "EJBCA API edit of the common name", edit: apiEditEJBCA(fleet.CertificateAuthority{CertificateCommonName: new("new-cn")}),
			wantEJBCAResend: true,
		},
		{
			name: "EJBCA API edit of the username template", edit: apiEditEJBCA(fleet.CertificateAuthority{EJBCAUsernameTemplate: new("new-user")}),
			wantEJBCAResend: true,
		},
		{
			name: "EJBCA API edit of the EJBCA CA name", edit: apiEditEJBCA(fleet.CertificateAuthority{EJBCACAName: new("OtherCA")}),
			wantEJBCAResend: true,
		},
		{
			name:            "EJBCA API edit of the certificate profile",
			edit:            apiEditEJBCA(fleet.CertificateAuthority{EJBCACertificateProfileName: new("new-certificate-profile")}),
			wantEJBCAResend: true,
		},
		{
			name:            "EJBCA API edit of the end entity profile",
			edit:            apiEditEJBCA(fleet.CertificateAuthority{EJBCAEndEntityProfileName: new("new-end-entity-profile")}),
			wantEJBCAResend: true,
		},
		{
			name: "EJBCA API edit of the UPN", edit: apiEditEJBCA(fleet.CertificateAuthority{CertificateUserPrincipalNames: &[]string{"new-upn"}}),
			wantEJBCAResend: true,
		},
		{name: "EJBCA GitOps edit of the mTLS key", edit: func() error {
			return gitOpsUpdate(ejbcaCA.ID, func(c *fleet.CertificateAuthority) { c.MTLSKeyPEM = new("new-key") })
		}},
		{name: "EJBCA GitOps edit of the end entity profile", edit: func() error {
			return gitOpsUpdate(ejbcaCA.ID, func(c *fleet.CertificateAuthority) { c.EJBCAEndEntityProfileName = new("gitops-end-entity-profile") })
		}, wantEJBCAResend: true},
		{name: "EJBCA GitOps removal of the username template", edit: func() error {
			return gitOpsUpdate(ejbcaCA.ID, func(c *fleet.CertificateAuthority) { c.EJBCAUsernameTemplate = nil })
		}, wantEJBCAResend: true},
		{name: "EJBCA GitOps removal of the UPNs", edit: func() error {
			return gitOpsUpdate(ejbcaCA.ID, func(c *fleet.CertificateAuthority) { c.CertificateUserPrincipalNames = nil })
		}, wantEJBCAResend: true},
		{name: "GitOps edit of a DigiCert CA and an EJBCA CA in one batch", edit: func() error {
			digiCert, err := ds.GetCertificateAuthorityByID(ctx, ca.ID, true)
			require.NoError(t, err)
			ejbca, err := ds.GetCertificateAuthorityByID(ctx, ejbcaCA.ID, true)
			require.NoError(t, err)
			digiCert.CertificateSeatID = new("batch-seat")
			ejbca.EJBCAUsernameTemplate = new("batch-user")
			return ds.BatchApplyCertificateAuthorities(ctx,
				fleet.CertificateAuthoritiesBatchOperations{Update: []*fleet.CertificateAuthority{digiCert, ejbca}})
		}, wantResend: true, wantEJBCAResend: true},
		// The name column ignores case, so the CA is the same one, and the profiles that use the new spelling are resent for the changed field.
		{name: "EJBCA GitOps edit that changes the case of the name and the common name", edit: func() error {
			return gitOpsUpdate(ejbcaCA.ID, func(c *fleet.CertificateAuthority) {
				c.Name = new(strings.ToUpper(*c.Name))
				c.CertificateCommonName = new("case-rename-cn")
			})
		}, wantUpperCaseResend: true},
	}
	pendingIf := func(resend bool) fleet.MDMDeliveryStatus {
		if resend {
			return fleet.MDMDeliveryPending
		}
		return fleet.MDMDeliveryVerified
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, p := range []*fleet.MDMAppleConfigProfile{prof, otherProf, ejbcaProf, ejbcaUpperCaseProf} {
				forceSetAppleHostProfileStatus(t, ds, host.UUID, p, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
			}
			for _, p := range []*fleet.MDMAppleConfigProfile{teamProf, ejbcaTeamProf} {
				forceSetAppleHostProfileStatus(t, ds, teamHost.UUID, p, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
			}

			require.NoError(t, c.edit())

			assertHostProfileStatus(t, ds, host.UUID,
				hostProfileStatus{prof.ProfileUUID, pendingIf(c.wantResend)},
				hostProfileStatus{otherProf.ProfileUUID, fleet.MDMDeliveryVerified},
				hostProfileStatus{ejbcaProf.ProfileUUID, pendingIf(c.wantEJBCAResend)},
				hostProfileStatus{ejbcaUpperCaseProf.ProfileUUID, pendingIf(c.wantUpperCaseResend)})
			assertHostProfileStatus(t, ds, teamHost.UUID,
				hostProfileStatus{teamProf.ProfileUUID, pendingIf(c.wantResend)},
				hostProfileStatus{ejbcaTeamProf.ProfileUUID, pendingIf(c.wantEJBCAResend)})
		})
	}
}

// testEJBCANamesAreUnique checks that the name of an EJBCA CA can't be shared with a CA of any other type, however the CAs are written,
// and that CAs of the other types that already share a name are not affected.
func testEJBCANamesAreUnique(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	scepCA := func(name string) *fleet.CertificateAuthority {
		return &fleet.CertificateAuthority{
			Type: string(fleet.CATypeCustomSCEPProxy), Name: &name, URL: new("https://scep.example.com"), Challenge: new("challenge"),
		}
	}
	inUse := func(t *testing.T, err error, name string) {
		nameInUse, ok := errors.AsType[fleet.EJBCANameInUseError](err)
		require.True(t, ok, "got %v", err)
		require.Equal(t, name, nameInUse.Name)
	}
	names := func() []string {
		cas, err := ds.ListCertificateAuthorities(ctx)
		require.NoError(t, err)
		var res []string
		for _, ca := range cas {
			res = append(res, ca.Type+":"+ca.Name)
		}
		return res
	}

	// CAs of other types may keep sharing a name, which the unique index on type and name allows, and a batch that keeps the name is fine.
	digiCert := newDigiCertCA(t, ds, "shared", "cn", "seat", nil)
	_, err := ds.NewCertificateAuthority(ctx, scepCA("shared"))
	require.NoError(t, err)
	digiCert, err = ds.GetCertificateAuthorityByID(ctx, digiCert.ID, true)
	require.NoError(t, err)
	digiCert.CertificateSeatID = new("batch-seat")
	require.NoError(t, ds.BatchApplyCertificateAuthorities(ctx, fleet.CertificateAuthoritiesBatchOperations{Update: []*fleet.CertificateAuthority{digiCert}}))

	ejbca := newEJBCACA(t, ds, "ejbca", "cn", nil, nil)
	unchanged := []string{"digicert:shared", "custom_scep_proxy:shared", "ejbca:ejbca"}

	t.Run("create", func(t *testing.T) {
		// An EJBCA CA can't take a name another CA has, and no other CA can take the name of an EJBCA CA.
		_, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
			Type: string(fleet.CATypeEJBCA), Name: new("shared"), URL: new("https://ejbca.example.com"),
		})
		inUse(t, err, "shared")
		_, err = ds.NewCertificateAuthority(ctx, scepCA("ejbca"))
		inUse(t, err, "ejbca")
		// The names are compared without regard to case, like the column is.
		_, err = ds.NewCertificateAuthority(ctx, scepCA("EJBCA"))
		inUse(t, err, "ejbca")
		require.ElementsMatch(t, unchanged, names())
	})

	t.Run("rename", func(t *testing.T) {
		err := ds.UpdateCertificateAuthorityByID(ctx, ejbca.ID, &fleet.CertificateAuthority{Type: ejbca.Type, Name: new("shared")})
		inUse(t, err, "shared")
		err = ds.UpdateCertificateAuthorityByID(ctx, digiCert.ID, &fleet.CertificateAuthority{Type: digiCert.Type, Name: new("ejbca")})
		inUse(t, err, "ejbca")
		require.ElementsMatch(t, unchanged, names())

		// A name nobody has is fine.
		require.NoError(t, ds.UpdateCertificateAuthorityByID(ctx, ejbca.ID, &fleet.CertificateAuthority{Type: ejbca.Type, Name: new("renamed")}))
	})

	t.Run("batch apply", func(t *testing.T) {
		ejbca, err := ds.GetCertificateAuthorityByID(ctx, ejbca.ID, true)
		require.NoError(t, err)
		before := names()

		// A new CA with an EJBCA name is rejected, and the whole batch rolls back.
		err = ds.BatchApplyCertificateAuthorities(ctx, fleet.CertificateAuthoritiesBatchOperations{
			Add: []*fleet.CertificateAuthority{scepCA("fresh"), scepCA(*ejbca.Name)},
		})
		inUse(t, err, *ejbca.Name)
		require.ElementsMatch(t, before, names())

		// A CA of another type can replace the EJBCA CA of the same name in one batch, since the old one is deleted in it.
		require.NoError(t, ds.BatchApplyCertificateAuthorities(ctx, fleet.CertificateAuthoritiesBatchOperations{
			Add: []*fleet.CertificateAuthority{scepCA(*ejbca.Name)}, Delete: []*fleet.CertificateAuthority{ejbca},
		}))
		require.ElementsMatch(t, []string{"digicert:shared", "custom_scep_proxy:shared", "custom_scep_proxy:" + *ejbca.Name}, names())
	})
}

// newEJBCACA creates an EJBCA CA with the given common name template, optional username template and UPNs.
func newEJBCACA(t *testing.T, ds *Datastore, name, commonName string, usernameTemplate *string, upns *[]string,
) *fleet.CertificateAuthority {
	ca, err := ds.NewCertificateAuthority(t.Context(), &fleet.CertificateAuthority{
		Type: string(fleet.CATypeEJBCA), Name: &name, URL: new("https://ejbca.example.com:8443"),
		MTLSCertPEM: new(newMTLSCertPEM(t, time.Now().Add(time.Hour))), MTLSKeyPEM: new("key"), EJBCACAName: new("ManagementCA"),
		EJBCACertificateProfileName: new("certificate-profile"), EJBCAEndEntityProfileName: new("end-entity-profile"),
		CertificateCommonName: &commonName, EJBCAUsernameTemplate: usernameTemplate, CertificateUserPrincipalNames: upns,
	})
	require.NoError(t, err)
	return ca
}

// newMTLSCertPEM returns a self-signed certificate that expires at notAfter.
func newMTLSCertPEM(t *testing.T, notAfter time.Time) string {
	certPEM, _, err := GenerateTestCertBytes(&x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Fleet"},
		NotBefore:    notAfter.Add(-time.Hour),
		NotAfter:     notAfter,
	})
	require.NoError(t, err)
	return string(certPEM)
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
