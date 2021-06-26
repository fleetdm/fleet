package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (mw validationMiddleware) ModifyAppConfig(ctx context.Context, p fleet.AppConfigPayload) (*fleet.AppConfig, error) {
	existing, err := mw.ds.AppConfig()
	if err != nil {
		return nil, errors.Wrap(err, "fetching existing app config in validation")
	}
	invalid := &fleet.InvalidArgumentError{}
	validateSSOSettings(p, existing, invalid)
	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.ModifyAppConfig(ctx, p)
}

func isSet(val *string) bool {
	if val != nil {
		return len(*val) > 0
	}
	return false
}

func validateSSOSettings(p fleet.AppConfigPayload, existing *fleet.AppConfig, invalid *fleet.InvalidArgumentError) {
	if p.SSOSettings != nil && p.SSOSettings.EnableSSO != nil {
		if *p.SSOSettings.EnableSSO {
			if !isSet(p.SSOSettings.Metadata) && !isSet(p.SSOSettings.MetadataURL) {
				if existing.Metadata == "" && existing.MetadataURL == "" {
					invalid.Append("metadata", "either metadata or metadata_url must be defined")
				}
			}
			if isSet(p.SSOSettings.Metadata) && isSet(p.SSOSettings.MetadataURL) {
				invalid.Append("metadata", "both metadata and metadata_url are defined, only one is allowed")
			}
			if !isSet(p.SSOSettings.EntityID) {
				if existing.EntityID == "" {
					invalid.Append("entity_id", "required")
				}
			} else {
				if len(*p.SSOSettings.EntityID) < 5 {
					invalid.Append("entity_id", "must be 5 or more characters")
				}
			}
			if !isSet(p.SSOSettings.IDPName) {
				if existing.IDPName == "" {
					invalid.Append("idp_name", "required")
				}
			} else {
				if len(*p.SSOSettings.IDPName) < 4 {
					invalid.Append("idp_name", "must be 4 or more characters")
				}
			}
		}
	}
}
