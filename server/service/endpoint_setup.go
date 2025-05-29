package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const standardQueryLibraryURL = "https://raw.githubusercontent.com/fleetdm/fleet/main/docs/01-Using-Fleet/standard-query-library/standard-query-library.yml"

type setupRequest struct {
	Admin        *fleet.UserPayload `json:"admin"`
	OrgInfo      *fleet.OrgInfo     `json:"org_info"`
	ServerURL    *string            `json:"server_url,omitempty"`
	EnrollSecret *string            `json:"osquery_enroll_secret,omitempty"`
}

type setupResponse struct {
	Admin        *fleet.User    `json:"admin,omitempty"`
	OrgInfo      *fleet.OrgInfo `json:"org_info,omitempty"`
	ServerURL    *string        `json:"server_url"`
	EnrollSecret *string        `json:"osquery_enroll_secret"`
	Token        *string        `json:"token,omitempty"`
	Err          error          `json:"error,omitempty"`
}

func (r setupResponse) Error() error { return r.Err }

func makeSetupEndpoint(svc fleet.Service, logger kitlog.Logger) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(setupRequest)
		config := &fleet.AppConfig{}
		if req.OrgInfo != nil {
			config.OrgInfo = *req.OrgInfo
		}
		if req.ServerURL != nil {
			config.ServerSettings.ServerURL = *req.ServerURL
		}
		config, err := svc.NewAppConfig(ctx, *config)
		if err != nil {
			return setupResponse{Err: err}, nil
		}

		if req.Admin == nil {
			return setupResponse{Err: ctxerr.New(ctx, "setup request must provide admin")}, nil
		}

		// creating the user should be the last action. If there's a user
		// present and other errors occur, the setup endpoint closes.
		adminPayload := *req.Admin
		if adminPayload.Email == nil || *adminPayload.Email == "" {
			err := ctxerr.New(ctx, "admin email cannot be empty")
			return setupResponse{Err: err}, nil
		}
		if adminPayload.Password == nil || *adminPayload.Password == "" {
			err := ctxerr.New(ctx, "admin password cannot be empty")
			return setupResponse{Err: err}, nil
		}
		// Make the user an admin
		adminPayload.GlobalRole = ptr.String(fleet.RoleAdmin)
		admin, err := svc.CreateInitialUser(ctx, adminPayload)
		if err != nil {
			return setupResponse{Err: err}, nil
		}

		// If everything works to this point, log the user in and return token.
		// If the login fails for some reason, ignore the error and don't return
		// a token, forcing the user to log in manually.
		var token *string
		_, session, err := svc.Login(ctx, *req.Admin.Email, *req.Admin.Password, false)
		if err != nil {
			level.Debug(logger).Log("endpoint", "setup", "op", "login", "err", err)
		} else {
			token = &session.Key

			// Apply standard query library using the admin token we just created
			if req.ServerURL != nil {
				if err := applyStandardQueryLibrary(ctx, *req.ServerURL, session.Key, logger); err != nil {
					level.Debug(logger).Log("endpoint", "setup", "op", "applyStandardQueryLibrary", "err", err)
					// Continue even if there's an error applying the standard query library
				}
			} else {
				level.Debug(logger).Log("endpoint", "setup", "msg", "Skipping standard query library application due to missing server URL")
			}
		}

		return setupResponse{
			Admin:     admin,
			OrgInfo:   &config.OrgInfo,
			ServerURL: req.ServerURL,
			Token:     token,
		}, nil
	}
}

// applyStandardQueryLibrary downloads the standard query library from GitHub
// and applies it to the Fleet server using an authenticated client.
func applyStandardQueryLibrary(ctx context.Context, serverURL string, token string, logger kitlog.Logger) error {
	level.Debug(logger).Log("msg", "Applying standard query library")

	// Download the standard query library from GitHub
	resp, err := http.Get(standardQueryLibraryURL)
	if err != nil {
		return fmt.Errorf("failed to download standard query library: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download standard query library, status: %d", resp.StatusCode)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read standard query library response body: %w", err)
	}

	// Parse the YAML content into specs
	specs, err := spec.GroupFromBytes(buf)
	if err != nil {
		return fmt.Errorf("failed to parse standard query library: %w", err)
	}

	// Create an authenticated client and apply specs
	client, err := NewClient(serverURL, true, "", "")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	client.SetToken(token)

	// Log function for ApplyGroup (minimal logging)
	logf := func(format string, a ...interface{}) {}

	// Apply the specs using the client's ApplyGroup method
	teamsSoftwareInstallers := make(map[string][]fleet.SoftwarePackageResponse)
	teamsScripts := make(map[string][]fleet.ScriptResponse)
	teamsVPPApps := make(map[string][]fleet.VPPAppResponse)

	_, _, _, _, err = client.ApplyGroup(
		ctx,
		false,
		specs,
		".",
		logf,
		nil,
		fleet.ApplyClientSpecOptions{},
		teamsSoftwareInstallers,
		teamsVPPApps,
		teamsScripts,
	)
	if err != nil {
		return fmt.Errorf("failed to apply standard query library: %w", err)
	}

	level.Debug(logger).Log("msg", "Standard query library applied successfully")
	return nil
}
