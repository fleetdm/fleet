package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kolide/kolide/server/kolide"
)

func decodeImportConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	// Unmarshal main config
	conf := kolide.ImportConfig{
		DryRun:        req.DryRun,
		Packs:         make(kolide.PackNameMap),
		ExternalPacks: make(kolide.PackNameToPackDetails),
	}
	if err := json.Unmarshal([]byte(req.Config), &conf); err != nil {
		return nil, err
	}
	// Unmarshal external packs
	for packName, packConfig := range req.ExternalPackConfigs {
		var pack kolide.PackDetails
		if err := json.Unmarshal([]byte(packConfig), &pack); err != nil {
			return nil, err
		}
		conf.ExternalPacks[packName] = pack
	}
	conf.GlobPackNames = req.GlobPackNames
	return conf, nil
}
