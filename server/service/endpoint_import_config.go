package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

type importRequest struct {
	DryRun bool `json:"dry_run"`
	// Config contains a JSON osquery config supplied by the end user
	Config string `json:"config"`
	// ExternalPackConfigs contains a map of external Pack configs keyed by
	// Pack name, this includes external packs referenced by the globbing
	// feature.  Not in the case of globbed packs, we expect the user to
	// generate unique pack names since we don't know what they are, these
	// names must be included in the GlobPackNames field so that we can
	// validate that they've been accounted for.
	ExternalPackConfigs map[string]string `json:"external_pack_configs"`
	// GlobPackNames list of user generated names for external packs
	// referenced by the glob feature, the JSON for the globbed packs
	// is stored in ExternalPackConfigs keyed by the GlobPackName
	GlobPackNames []string `json:"glob_pack_names"`
}

type importResponse struct {
	Response *kolide.ImportConfigResponse `json:"response,omitempty"`
	Err      error                        `json:"error,omitempty"`
}

func (ir importResponse) error() error { return ir.Err }

func makeImportConfigEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config := request.(kolide.ImportConfig)
		resp, err := svc.ImportConfig(ctx, &config)
		if err != nil {
			return importResponse{Err: err}, nil
		}
		return importResponse{Response: resp}, nil
	}
}
