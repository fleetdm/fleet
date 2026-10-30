package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestBatchApplyCertificateAuthoritiesFree(t *testing.T) {
	svc, ctx := newTestService(t, new(mock.Store), nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierFree}})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	opts := fleet.BatchApplyCertificateAuthoritiesOpts{ViaGitOps: true}

	// No CAs is what most Free GitOps repos send, and must keep working.
	require.NoError(t, svc.BatchApplyCertificateAuthorities(ctx, fleet.GroupedCertificateAuthorities{}, opts))

	// Any CA type needs Premium, including the ones the Free check used to miss.
	for name, cas := range map[string]fleet.GroupedCertificateAuthorities{
		"custom est": {EST: []fleet.ESTProxyCA{{Name: "E"}}},
		"smallstep":  {Smallstep: []fleet.SmallstepSCEPProxyCA{{Name: "S"}}},
		"google cas": {GoogleCAS: []fleet.GoogleCASCA{{Name: "G"}}},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, svc.BatchApplyCertificateAuthorities(ctx, cas, opts), fleet.ErrMissingLicense)
		})
	}
}

func newMockDigicertCA(url string, name string) fleet.DigiCertCA {
	digiCertCA := fleet.DigiCertCA{
		Name:                          name,
		URL:                           url,
		APIToken:                      "api_token",
		ProfileID:                     "profile_id",
		CertificateCommonName:         "common_name",
		CertificateUserPrincipalNames: []string{"user_principal_name"},
		CertificateSeatID:             "seat_id",
	}
	return digiCertCA
}

func newMockCustomSCEPProxyCA(url string, name string) fleet.CustomSCEPProxyCA {
	challenge, _ := server.GenerateRandomText(6)
	return fleet.CustomSCEPProxyCA{
		Name:      name,
		URL:       url,
		Challenge: challenge,
	}
}

func newMockSmallstepSCEPProxyCA(url, challengeURL, name string) fleet.SmallstepSCEPProxyCA {
	return fleet.SmallstepSCEPProxyCA{
		Name:         name,
		URL:          url,
		ChallengeURL: challengeURL,
		Username:     "username",
		Password:     "password",
	}
}
