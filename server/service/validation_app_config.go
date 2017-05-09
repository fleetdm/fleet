package service

import (
	"context"

	"github.com/kolide/kolide/server/kolide"
)

func (mw validationMiddleware) ModifyAppConfig(ctx context.Context, p kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	invalid := &invalidArgumentError{}
	validateSSOSettings(p, invalid)
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

func validateSSOSettings(p kolide.AppConfigPayload, invalid *invalidArgumentError) {
	if p.SSOSettings != nil && p.SSOSettings.EnableSSO != nil {
		if *p.SSOSettings.EnableSSO {
			if !isSet(p.SSOSettings.Metadata) && !isSet(p.SSOSettings.MetadataURL) {
				invalid.Append("metadata", "either metadata or metadata_url must be defined")
			}
			if isSet(p.SSOSettings.Metadata) && isSet(p.SSOSettings.MetadataURL) {
				invalid.Append("metadata", "both metadata and metadata_url are defined, only one is allowed")
			}
			if !isSet(p.SSOSettings.EntityID) {
				invalid.Append("entity_id", "required")
			} else {
				if len(*p.SSOSettings.EntityID) < 5 {
					invalid.Append("entity_id", "must be 5 or more characters")
				}
			}
			if !isSet(p.SSOSettings.IDPName) {
				invalid.Append("idp_name", "required")
			} else {
				if len(*p.SSOSettings.IDPName) < 5 {
					invalid.Append("idp_name", "must be 5 or more characters")
				}
			}
		}
	}
}
