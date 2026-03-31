package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/pkg/startertemplates"
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

func makeSetupEndpoint(svc fleet.Service, logger *slog.Logger) endpoint.Endpoint {
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
				if err := ApplyStarterLibrary(
					ctx,
					*req.ServerURL,
					session.Key,
					logger,
					NewClient,
				); err != nil {
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

// ApplyStarterLibrary renders the starter templates and applies them to the
// Fleet server via the GitOps pipeline, producing the same result as running
// `fleetctl new` followed by `fleetctl gitops`.
func ApplyStarterLibrary(
	ctx context.Context,
	serverURL string,
	token string,
	logger *slog.Logger,
	clientFactory func(serverURL string, insecureSkipVerify bool, rootCA, urlPrefix string, options ...ClientOption) (*Client, error),
) error {
	logger.DebugContext(ctx, "Applying starter library")

	// Create an authenticated client.
	client, err := clientFactory(serverURL, true, "", "")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	client.SetToken(token)

	// Fetch app config to get the org name and license info.
	appConfig, err := client.GetAppConfig()
	if err != nil {
		return fmt.Errorf("failed to get app config: %w", err)
	}

	orgName := appConfig.OrgInfo.OrgName
	if orgName == "" {
		orgName = "Fleet"
	}

	// Render templates to a temp directory.
	tempDir, err := startertemplates.RenderToTempDir(orgName)
	if err != nil {
		return fmt.Errorf("failed to render starter templates: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Set env overrides so GitOpsFromFile can expand $FLEET_URL without
	// polluting the process environment.
	spec.SetEnvOverrides(map[string]string{
		"FLEET_URL": serverURL,
	})
	defer spec.SetEnvOverrides(nil)

	logf := func(format string, a ...interface{}) {
		logger.DebugContext(ctx, fmt.Sprintf(format, a...))
	}

	// Determine which files to process. Global config is always applied;
	// team configs are only applied for premium licenses.
	type configFile struct {
		path     string
		isGlobal bool
	}
	files := []configFile{
		{path: filepath.Join(tempDir, "default.yml"), isGlobal: true},
	}
	if appConfig.License != nil && appConfig.License.IsPremium() {
		fleetDir := filepath.Join(tempDir, "fleets")
		entries, err := os.ReadDir(fleetDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read fleets directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
				continue
			}
			files = append(files, configFile{
				path:     filepath.Join(fleetDir, entry.Name()),
				isGlobal: false,
			})
		}
	}

	// Parse and apply each config file, global first.
	teamsSoftwareInstallers := make(map[string][]fleet.SoftwarePackageResponse)
	teamsVPPApps := make(map[string][]fleet.VPPAppResponse)
	teamsScripts := make(map[string][]fleet.ScriptResponse)

	for _, f := range files {
		baseDir := filepath.Dir(f.path)
		config, err := spec.GitOpsFromFile(f.path, baseDir, appConfig, logf)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", filepath.Base(f.path), err)
		}

		if f.isGlobal && !config.Controls.Set() {
			// Controls are required for global config; the default.yml template
			// includes an empty controls section so this shouldn't happen, but
			// handle it gracefully.
			logger.DebugContext(ctx, "global config missing controls section, skipping")
		}

		// Compute label changes. On a fresh instance there are no existing
		// labels so every label in the template is an addition.
		if len(config.Labels) > 0 {
			var changes []spec.LabelChange
			for _, l := range config.Labels {
				changes = append(changes, spec.LabelChange{
					Name:     l.Name,
					Op:       "+",
					TeamName: config.CoercedTeamName(),
					FileName: filepath.Base(f.path),
				})
			}
			config.LabelChangesSummary = spec.NewLabelChangesSummary(changes, nil)
		}

		_, err = client.DoGitOps(
			ctx,
			config,
			f.path,
			logf,
			false, // not a dry run
			nil,   // no dry run assumptions
			appConfig,
			teamsSoftwareInstallers,
			teamsVPPApps,
			teamsScripts,
			nil, // no icon settings
		)
		if err != nil {
			return fmt.Errorf("failed to apply %s: %w", filepath.Base(f.path), err)
		}
	}

	logger.DebugContext(ctx, "Starter library applied successfully")
	return nil
}
