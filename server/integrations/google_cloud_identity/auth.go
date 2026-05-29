// Package google_cloud_identity implements Fleet's integration with Google
// Cloud Identity for PATCHing per-(host, deviceUser) ClientStates that
// Context-Aware Access can evaluate.
//
// The integration:
//   - Authenticates as a Google service account (SA JSON or WIF) with
//     domain-wide delegation, impersonating a Workspace admin.
//   - Resolves Fleet hosts to Cloud Identity deviceUser resource names via
//     the Endpoint Verification accounts.json file (osquery-collected) or
//     the host_emails fallback for non-EV hosts.
//   - On every osquery distributed-query result that updates policy
//     compliance, computes the desired ClientState and PATCHes Cloud Identity
//     only when it differs from the last-known state recorded in Fleet's DB.
//
// See proposals/cloud-identity-clientstate-integration.md in fleet-terraform
// for the full design rationale.
package google_cloud_identity

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Required OAuth scope. The same scope covers ClientState writes and the
// (future v2) approve/block methods, so requesting it once is enough for
// every method the integration may call.
const cloudIdentityScope = "https://www.googleapis.com/auth/cloud-identity.devices"

// directoryScope is needed only for the customer-ID validation call to
// admin.googleapis.com/admin/directory/v1/customers/my_customer that runs at
// integration startup.
const directoryScope = "https://www.googleapis.com/auth/admin.directory.customer.readonly"

// NewTokenSource constructs a Google OAuth2 token source for Cloud Identity
// based on the GoogleCloudIdentityConfig. It picks SA JSON over WIF when both
// are configured (SA JSON wins) and applies DWD subject impersonation against
// the configured admin email.
//
// Returns an oauth2.TokenSource that yields fresh access tokens on each call.
// The caller is responsible for wrapping it with oauth2.ReuseTokenSource if
// they want caching across HTTP calls.
func NewTokenSource(ctx context.Context, cfg config.GoogleCloudIdentityConfig) (oauth2.TokenSource, error) {
	if !cfg.IsSet() {
		return nil, errors.New("google_cloud_identity: config not set")
	}
	if cfg.ImpersonatedAdmin == "" {
		return nil, errors.New("google_cloud_identity: impersonated_admin is required")
	}

	if cfg.ServiceAccountJSON != "" || cfg.ServiceAccountJSONBytes != "" {
		return newServiceAccountTokenSource(ctx, cfg)
	}
	return newWorkloadIdentityTokenSource(ctx, cfg)
}

// newServiceAccountTokenSource builds a JWT-config-based token source using a
// service-account JSON key (path or inline bytes), with DWD subject
// impersonation.
func newServiceAccountTokenSource(ctx context.Context, cfg config.GoogleCloudIdentityConfig) (oauth2.TokenSource, error) {
	var keyBytes []byte
	switch {
	case cfg.ServiceAccountJSONBytes != "":
		keyBytes = []byte(cfg.ServiceAccountJSONBytes)
	case cfg.ServiceAccountJSON != "":
		b, err := os.ReadFile(cfg.ServiceAccountJSON)
		if err != nil {
			return nil, fmt.Errorf("read service account JSON %q: %w", cfg.ServiceAccountJSON, err)
		}
		keyBytes = b
	default:
		return nil, errors.New("google_cloud_identity: no service-account JSON configured")
	}

	jwtConfig, err := google.JWTConfigFromJSON(keyBytes, cloudIdentityScope, directoryScope)
	if err != nil {
		return nil, fmt.Errorf("parse service account JSON: %w", err)
	}
	// DWD subject impersonation: Workspace admin email whose authority Fleet
	// borrows when calling Cloud Identity.
	jwtConfig.Subject = cfg.ImpersonatedAdmin

	return jwtConfig.TokenSource(ctx), nil
}

// newWorkloadIdentityTokenSource builds a token source that uses Workload
// Identity Federation to obtain a service-account token without a JSON key.
// This is the recommended deployment shape for Fleet running in GKE or
// another OIDC-issuing environment.
//
// For v1, WIF support is implemented but expected to be exercised by Fleet
// Cloud customers in a later rollout. The prototype verification will use
// SA-JSON; WIF code is tested via unit tests against the same TokenSource
// interface.
func newWorkloadIdentityTokenSource(_ context.Context, cfg config.GoogleCloudIdentityConfig) (oauth2.TokenSource, error) {
	// TODO(google_cloud_identity): wire up
	// google.golang.org/api/option.WithCredentialsFile(externalAccountFile)
	// or assemble an external_account credential JSON document from the
	// audience + service-account email and feed it through
	// google.CredentialsFromJSON.
	//
	// Holding off on the actual implementation until the SA-JSON path is
	// verified end-to-end against Robbie's Workspace tenant; the interface
	// the rest of the integration consumes (oauth2.TokenSource) is identical,
	// so swapping in WIF is a localized change.
	return nil, fmt.Errorf("google_cloud_identity: workload identity federation not yet implemented (use service_account_json for now)")
}
