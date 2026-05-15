package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/endpoint"
)

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

func makeSetupEndpoint(svc fleet.Service, logger *slog.Logger, applyStarterLibrary func(ctx context.Context, serverURL, token string) error) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
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
			logger.DebugContext(ctx, "setup login", "endpoint", "setup", "op", "login", "err", err)
		} else {
			token = &session.Key

			// Apply starter library using the admin token we just created
			if req.ServerURL != nil {
				if err := applyStarterLibrary(ctx, *req.ServerURL, session.Key); err != nil {
					logger.DebugContext(ctx, "setup apply starter library", "endpoint", "setup", "op", "applyStarterLibrary", "err", err)
					// Continue even if there's an error applying the starter library
				}
			} else {
				logger.DebugContext(ctx, "Skipping starter library application due to missing server URL", "endpoint", "setup")
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

// ApplyStarterLibrary scaffolds the starter GitOps templates via `fleetctl new`
// and applies them via `fleetctl gitops`, producing the same result as a user
// running those commands manually.
//
// The runFleetctl callback should run the fleetctl CLI with the given arguments.
// This keeps the CLI dependency out of the service package.
func ApplyStarterLibrary(
	ctx context.Context,
	serverURL string,
	token string,
	logger *slog.Logger,
	runFleetctl func(args []string) error,
) error {
	logger.DebugContext(ctx, "Applying starter library")

	// Create an authenticated client to fetch app config.
	client, err := NewClient(serverURL, true, "", "")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	client.SetToken(token)

	appConfig, err := client.GetAppConfig()
	if err != nil {
		return fmt.Errorf("failed to get app config: %w", err)
	}

	orgName := appConfig.OrgInfo.OrgName
	if orgName == "" {
		orgName = "Fleet"
	}

	// Create a temp directory for the rendered templates.
	tempDir, err := os.MkdirTemp("", "fleet-starter-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	outDir := filepath.Join(tempDir, "gitops")

	// Render templates using `fleetctl new`.
	if err := runFleetctl([]string{"new", "--org-name", orgName, "--dir", outDir}); err != nil {
		return fmt.Errorf("fleetctl new: %w", err)
	}

	// Set env overrides so GitOpsFromFile can expand $FLEET_URL without
	// polluting the process environment.
	spec.SetEnvOverrides(map[string]string{
		"FLEET_URL": serverURL,
	})
	defer spec.SetEnvOverrides(nil)

	// Write a temporary fleetctl config file with auth credentials.
	configFile, err := os.CreateTemp(tempDir, "fleetctl-config-*.yml")
	if err != nil {
		return fmt.Errorf("failed to create fleetctl config: %w", err)
	}
	fmt.Fprintf(configFile, "contexts:\n  default:\n    address: %s\n    tls-skip-verify: true\n    token: %s\n",
		serverURL, token)
	configFile.Close()

	// Build the gitops args: global config first, then team configs (premium only).
	args := []string{"gitops", "--config", configFile.Name(), "-f", filepath.Join(outDir, "default.yml")}

	if appConfig.License != nil && appConfig.License.IsPremium() {
		fleetDir := filepath.Join(outDir, "fleets")
		entries, err := os.ReadDir(fleetDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read fleets directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
				continue
			}
			args = append(args, "-f", filepath.Join(fleetDir, entry.Name()))
		}
	}

	if err := runFleetctl(args); err != nil {
		return fmt.Errorf("fleetctl gitops: %w", err)
	}

	logger.DebugContext(ctx, "Starter library applied successfully")
	return nil
}
