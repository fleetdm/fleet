package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log/ctxlog"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/sync"
)

// RetrieveAssignerProfileHandler returns the assigner profile UUID for the
// given DEP name.
//
// Note the whole URL path is used as the DEP name. This necessitates
// stripping the URL prefix before using this handler. Also note we expose Go
// errors to the output as this is meant for "API" users.
func RetrieveAssignerProfileHandler(store sync.AssignerProfileRetriever, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if r.URL.Path == "" {
			logger.Info("msg", "DEP name check", "err", "missing DEP name")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		logger = logger.With("name", r.URL.Path)
		profileUUID, _, err := store.RetrieveAssignerProfile(r.Context(), r.URL.Path)
		if err != nil {
			logger.Info("msg", "retrieving assigner profile", "err", err)
			jsonError(w, err)
			return
		}
		logger = logger.With("profile_uuid", profileUUID)
		w.Header().Set("Content-type", "application/json")
		profile := &struct {
			ProfileUUID string `json:"profile_uuid,omitempty"`
		}{ProfileUUID: profileUUID}
		err = json.NewEncoder(w).Encode(profile)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}

type AssignerProfileStorer interface {
	StoreAssignerProfile(ctx context.Context, name string, profileUUID string) error
}

// StoreAssignerProfileHandler saves the assigner profile UUID for the
// given DEP name.
//
// Note the whole URL path is used as the DEP name. This necessitates
// stripping the URL prefix before using this handler. Also note we expose Go
// errors to the output as this is meant for "API" users.
func StoreAssignerProfileHandler(store AssignerProfileStorer, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if r.URL.Path == "" {
			logger.Info("msg", "DEP name check", "err", "missing DEP name")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		logger = logger.With("name", r.URL.Path)
		profileUUID := r.URL.Query().Get("profile_uuid")
		if profileUUID == "" {
			logger.Info("msg", "reading profile UUID", "err", "empty profile UUID")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		logger = logger.With("profile_uuid", profileUUID)
		err := store.StoreAssignerProfile(r.Context(), r.URL.Path, profileUUID)
		if err != nil {
			logger.Info("msg", "storing assigner profile", "err", err)
			jsonError(w, err)
			return
		}
		logger.Debug("msg", "stored assigner profile")
		w.Header().Set("Content-type", "application/json")
		profile := &struct {
			ProfileUUID string `json:"profile_uuid,omitempty"`
		}{ProfileUUID: profileUUID}
		err = json.NewEncoder(w).Encode(profile)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}
