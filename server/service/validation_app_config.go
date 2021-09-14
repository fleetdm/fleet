package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (mw validationMiddleware) ModifyAppConfig(ctx context.Context, p []byte) (*fleet.AppConfig, error) {
	existing, err := mw.ds.AppConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "fetching existing app config in validation")
	}
	invalid := &fleet.InvalidArgumentError{}
	var appConfig fleet.AppConfig
	err = json.Unmarshal(p, &appConfig)
	if err != nil {
		return nil, err
	}
	validateSSOSettings(appConfig, existing, invalid)
	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.ModifyAppConfig(ctx, p)
}

func validateSSOSettings(p fleet.AppConfig, existing *fleet.AppConfig, invalid *fleet.InvalidArgumentError) {
	if p.SSOSettings.EnableSSO {
		if p.SSOSettings.Metadata == "" && p.SSOSettings.MetadataURL == "" {
			if existing.SSOSettings.Metadata == "" && existing.SSOSettings.MetadataURL == "" {
				invalid.Append("metadata", "either metadata or metadata_url must be defined")
			}
		}
		if p.SSOSettings.Metadata != "" && p.SSOSettings.MetadataURL != "" {
			invalid.Append("metadata", "both metadata and metadata_url are defined, only one is allowed")
		}
		if p.SSOSettings.EntityID == "" {
			if existing.SSOSettings.EntityID == "" {
				invalid.Append("entity_id", "required")
			}
		} else {
			if len(p.SSOSettings.EntityID) < 5 {
				invalid.Append("entity_id", "must be 5 or more characters")
			}
		}
		if p.SSOSettings.IDPName == "" {
			if existing.SSOSettings.IDPName == "" {
				invalid.Append("idp_name", "required")
			}
		} else {
			if len(p.SSOSettings.IDPName) < 4 {
				invalid.Append("idp_name", "must be 4 or more characters")
			}
		}
	}
}
