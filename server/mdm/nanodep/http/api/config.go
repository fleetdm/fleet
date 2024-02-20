package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log/ctxlog"
)

// RetrieveConfigHandler returns the DEP server config for the DEP
// name in the path.
//
// Note the whole URL path is used as the DEP name. This necessitates
// stripping the URL prefix before using this handler. Also note we expose Go
// errors to the output as this is meant for "API" users.
func RetrieveConfigHandler(store client.ConfigRetriever, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if r.URL.Path == "" {
			logger.Info("msg", "DEP name check", "err", "missing DEP name")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		logger = logger.With("name", r.URL.Path)
		config, err := store.RetrieveConfig(r.Context(), r.URL.Path)
		if err != nil {
			logger.Info("msg", "retrieving config", "err", err)
			jsonError(w, err)
			return
		}
		w.Header().Set("Content-type", "application/json")
		err = json.NewEncoder(w).Encode(config)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}

type ConfigStorer interface {
	StoreConfig(ctx context.Context, name string, config *client.Config) error
}

// StoreConfigHandler stores the DEP server config for the DEP
// name in the path.
//
// Note the whole URL path is used as the DEP name. This necessitates
// stripping the URL prefix before using this handler. Also note we expose Go
// errors to the output as this is meant for "API" users.
func StoreConfigHandler(store ConfigStorer, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if r.URL.Path == "" {
			logger.Info("msg", "DEP name check", "err", "missing DEP name")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		logger = logger.With("name", r.URL.Path)
		config := new(client.Config)
		err := json.NewDecoder(r.Body).Decode(config)
		if err != nil {
			logger.Info("msg", "decoding request body", "err", err)
			jsonError(w, err)
			return
		}
		defer r.Body.Close()
		err = store.StoreConfig(r.Context(), r.URL.Path, config)
		if err != nil {
			logger.Info("msg", "storing config", "err", err)
			jsonError(w, err)
			return
		}
		logger.Debug("msg", "stored config")
		w.Header().Set("Content-type", "application/json")
		err = json.NewEncoder(w).Encode(config)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}
