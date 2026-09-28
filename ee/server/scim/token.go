package scim

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func newSCIMTokenHandler(ds fleet.Datastore, svc fleet.Service, authorizer *authz.Authorizer, logger *slog.Logger) http.Handler {
	type rotator interface {
		EnsureIDPConnection(ctx context.Context, name string) error
		RotateIDPConnectionSCIMToken(ctx context.Context, name string) (string, error)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/scim_token") {
			http.NotFound(w, r)
			return
		}
		if err := authorizer.Authorize(r.Context(), &fleet.ScimUser{}, fleet.ActionWrite); err != nil {
			errorHandler(w, logger, err.Error(), http.StatusForbidden)
			return
		}
		lic, err := svc.License(r.Context())
		if err != nil || lic == nil || !lic.IsPremium() {
			errorHandler(w, logger, fleet.ErrMissingLicense.Error(), http.StatusPaymentRequired)
			return
		}
		name := identityProviderNameFromPath(r.URL.Path)
		if name == "" {
			errorHandler(w, logger, "identity provider name is required", http.StatusBadRequest)
			return
		}
		store, ok := ds.(rotator)
		if !ok {
			errorHandler(w, logger, "identity provider connections are unavailable", http.StatusInternalServerError)
			return
		}
		if err := store.EnsureIDPConnection(r.Context(), name); err != nil {
			errorHandler(w, logger, err.Error(), http.StatusInternalServerError)
			return
		}
		secret, err := store.RotateIDPConnectionSCIMToken(r.Context(), name)
		if err != nil {
			status := http.StatusInternalServerError
			if fleet.IsNotFound(err) {
				status = http.StatusNotFound
			}
			errorHandler(w, logger, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token":    secret,
			"scim_url": "/api/latest/fleet/scim",
		})
	})
}

func identityProviderNameFromPath(path string) string {
	const marker = "/identity_providers/"
	_, rest, ok := strings.Cut(path, marker)
	if !ok {
		return ""
	}
	rest = strings.Trim(rest, "/")
	name, _, _ := strings.Cut(rest, "/")
	return name
}
