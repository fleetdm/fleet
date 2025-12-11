package service

import (
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

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
