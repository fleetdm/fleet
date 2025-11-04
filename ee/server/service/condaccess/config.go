package condaccess

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
)

func initAssets(ctx context.Context, ds fleet.Datastore) error {
	// Check if we have existing assets for both SCEP CA and IdP
	expectedAssets := []fleet.MDMAssetName{
		fleet.MDMAssetConditionalAccessCACert,
		fleet.MDMAssetConditionalAccessCAKey,
		fleet.MDMAssetConditionalAccessIDPCert,
		fleet.MDMAssetConditionalAccessIDPKey,
	}
	savedAssets, err := ds.GetAllMDMConfigAssetsByName(ctx, expectedAssets, nil)
	if err != nil {
		// allow not found errors as it means we're generating the assets for the first time.
		if !fleet.IsNotFound(err) {
			return fmt.Errorf("loading existing conditional access assets from the database: %w", err)
		}
	}

	// Check if CA assets need to be created
	_, hasCACert := savedAssets[fleet.MDMAssetConditionalAccessCACert]
	_, hasCAKey := savedAssets[fleet.MDMAssetConditionalAccessCAKey]
	if !hasCACert || !hasCAKey {
		// Create CA cert and key for SCEP
		caCert := depot.NewCACert(
			depot.WithYears(10),
			depot.WithCommonName("Fleet conditional access CA"),
			// Signal that the CA is local to the deployment and not necessarily managed by Fleet or another external vendor
			depot.WithOrganization("Local certificate authority"),
		)
		scepCert, scepKey, err := depot.NewCACertKey(caCert)
		if err != nil {
			return fmt.Errorf("generating conditional access SCEP cert and key: %w", err)
		}

		// Store CA assets encrypted
		var assets []fleet.MDMConfigAsset
		for k, v := range map[fleet.MDMAssetName][]byte{
			fleet.MDMAssetConditionalAccessCACert: certificate.EncodeCertPEM(scepCert),
			fleet.MDMAssetConditionalAccessCAKey:  certificate.EncodePrivateKeyPEM(scepKey),
		} {
			assets = append(assets, fleet.MDMConfigAsset{
				Name:  k,
				Value: v,
			})
		}

		if err := ds.InsertMDMConfigAssets(ctx, assets, nil); err != nil {
			return fmt.Errorf("inserting conditional access SCEP assets: %w", err)
		}
	}

	// Check if IdP assets need to be created
	_, hasIDPCert := savedAssets[fleet.MDMAssetConditionalAccessIDPCert]
	_, hasIDPKey := savedAssets[fleet.MDMAssetConditionalAccessIDPKey]
	if !hasIDPCert || !hasIDPKey {
		// Create IdP cert and key for SAML signing
		idpCert := depot.NewCACert(
			depot.WithYears(10),
			depot.WithCommonName("Fleet conditional access IdP"),
			depot.WithOrganization("Local certificate authority"),
		)
		idpCertX509, idpKey, err := depot.NewCACertKey(idpCert)
		if err != nil {
			return fmt.Errorf("generating conditional access IdP cert and key: %w", err)
		}

		// Store IdP assets encrypted
		var assets []fleet.MDMConfigAsset
		for k, v := range map[fleet.MDMAssetName][]byte{
			fleet.MDMAssetConditionalAccessIDPCert: certificate.EncodeCertPEM(idpCertX509),
			fleet.MDMAssetConditionalAccessIDPKey:  certificate.EncodePrivateKeyPEM(idpKey),
		} {
			assets = append(assets, fleet.MDMConfigAsset{
				Name:  k,
				Value: v,
			})
		}

		if err := ds.InsertMDMConfigAssets(ctx, assets, nil); err != nil {
			return fmt.Errorf("inserting conditional access IdP assets: %w", err)
		}
	}

	return nil
}
