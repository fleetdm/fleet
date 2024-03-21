package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log/ctxlog"
)

type AuthTokensStorer interface {
	StoreAuthTokens(ctx context.Context, name string, tokens *client.OAuth1Tokens) error
}

// RetrieveAuthTokensHandler returns the DEP server OAuth1 tokens for the DEP
// name in the path.
//
// Note the whole URL path is used as the DEP name. This necessitates
// stripping the URL prefix before using this handler. Also note we expose Go
// errors to the output as this is meant for "API" users.
func RetrieveAuthTokensHandler(store client.AuthTokensRetriever, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if r.URL.Path == "" {
			logger.Info("msg", "DEP name check", "err", "missing DEP name")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		logger = logger.With("name", r.URL.Path)
		tokens, err := store.RetrieveAuthTokens(r.Context(), r.URL.Path)
		if err != nil {
			logger.Info("msg", "retrieving auth tokens", "err", err)
			jsonError(w, err)
			return
		}
		w.Header().Set("Content-type", "application/json")
		err = json.NewEncoder(w).Encode(tokens)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}

// StoreAuthTokensHandler reads DEP server OAuth1 tokens as a JSON body and
// saves them using store.
//
// Note the whole URL path is used as the DEP name. This necessitates
// stripping the URL prefix before using this handler. Also note we expose Go
// errors to the output as this is meant for "API" users.
func StoreAuthTokensHandler(store AuthTokensStorer, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if r.URL.Path == "" {
			logger.Info("msg", "DEP name check", "err", "missing DEP name")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		logger = logger.With("name", r.URL.Path)
		tokens := new(client.OAuth1Tokens)
		err := json.NewDecoder(r.Body).Decode(tokens)
		if err != nil {
			logger.Info("msg", "decoding request body", "err", err)
			jsonError(w, err)
			return
		}
		defer r.Body.Close()
		if !tokens.Valid() {
			logger.Info("msg", "checking auth token validity", "err", "invalid tokens")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		err = store.StoreAuthTokens(r.Context(), r.URL.Path, tokens)
		if err != nil {
			logger.Info("msg", "storing auth tokens", "err", err)
			jsonError(w, err)
			return
		}
		logger.Debug("msg", "stored auth tokens")
		w.Header().Set("Content-type", "application/json")
		err = json.NewEncoder(w).Encode(tokens)
		if err != nil {
			logger.Info("msg", "encoding response body", "err", err)
			return
		}
	}
}
