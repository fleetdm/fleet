package certificate

import (
	"errors"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type groupedCertificateAuthorities struct {
	Hydrant         []fleet.HydrantCA
	DigiCert        []fleet.DigiCertCA
	NDESSCEP        *fleet.NDESSCEPProxyCA
	CustomScepProxy []fleet.CustomSCEPProxyCA
}

func GroupCertificateAuthoritiesByType(cas []*fleet.CertificateAuthority) (*groupedCertificateAuthorities, error) {
	grouped := &groupedCertificateAuthorities{
		DigiCert:        []fleet.DigiCertCA{},
		Hydrant:         []fleet.HydrantCA{},
		CustomScepProxy: []fleet.CustomSCEPProxyCA{},
		NDESSCEP:        nil,
	}

	for _, ca := range cas {
		switch ca.Type {
		case string(fleet.CATypeDigiCert):
			grouped.DigiCert = append(grouped.DigiCert, fleet.DigiCertCA{
				Name:                          ca.Name,
				CertificateCommonName:         *ca.CertificateCommonName,
				CertificateSeatID:             *ca.CertificateSeatID,
				CertificateUserPrincipalNames: ca.CertificateUserPrincipalNames,
				APIToken:                      *ca.APIToken,
				URL:                           ca.URL,
				ProfileID:                     *ca.ProfileID,
			})
		case string(fleet.CATypeNDESSCEPProxy):
			if grouped.NDESSCEP != nil {
				return nil, errors.New("multiple NDESSCEP proxy CAs found when grouping")
			}

			grouped.NDESSCEP = &fleet.NDESSCEPProxyCA{
				URL:      ca.URL,
				AdminURL: *ca.AdminURL,
				Username: *ca.Username,
				Password: *ca.Password,
			}

		case string(fleet.CATypeHydrant):
			grouped.Hydrant = append(grouped.Hydrant, fleet.HydrantCA{
				Name:         ca.Name,
				URL:          ca.URL,
				ClientID:     *ca.ClientID,
				ClientSecret: *ca.ClientSecret,
			})
		case string(fleet.CATypeCustomSCEPProxy):
			grouped.CustomScepProxy = append(grouped.CustomScepProxy, fleet.CustomSCEPProxyCA{
				Name:      ca.Name,
				URL:       ca.URL,
				Challenge: *ca.Challenge,
			})
		}
	}

	return grouped, nil
}
