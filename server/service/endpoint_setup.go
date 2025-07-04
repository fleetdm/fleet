package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	starterLibraryURL = "https://raw.githubusercontent.com/fleetdm/fleet/main/docs/01-Using-Fleet/starter-library/starter-library.yml"
	scriptsBaseURL    = "https://raw.githubusercontent.com/fleetdm/fleet/main/"
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

type applyGroupFunc func(context.Context, *spec.Group) error

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

			// Apply starter library using the admin token we just created
			if req.ServerURL != nil {
				if err := ApplyStarterLibrary(
					ctx,
					*req.ServerURL,
					session.Key,
					logger,
					fleethttp.NewClient,
					NewClient,
					nil,   // No mock ApplyGroup for production code
					false, // Don't skip teams on free license for normal setup
				); err != nil {
					level.Debug(logger).Log("endpoint", "setup", "op", "applyStarterLibrary", "err", err)
					// Continue even if there's an error applying the starter library
				}
			} else {
				level.Debug(logger).Log("endpoint", "setup", "msg", "Skipping starter library application due to missing server URL")
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

// ApplyStarterLibrary downloads the starter library from GitHub
// and applies it to the Fleet server using an authenticated client.
// TODO: Move the apply starter library logic to use the serve command as an entry point to simplify and leverage the entire fleet.Service.
// Entry point: https://github.com/fleetdm/fleet/blob/2dfadc0971c6ba45c19dad2f5f1f4cd0f1b89b20/cmd/fleet/serve.go#L1099-L1100
func ApplyStarterLibrary(
	ctx context.Context,
	serverURL string,
	token string,
	logger kitlog.Logger,
	httpClientFactory func(opts ...fleethttp.ClientOpt) *http.Client,
	clientFactory func(serverURL string, insecureSkipVerify bool, rootCA, urlPrefix string, options ...ClientOption) (*Client, error),
	// For testing only - if provided, this function will be used instead of client.ApplyGroup
	mockApplyGroup func(ctx context.Context, specs *spec.Group) error,
	// If true, teams will be skipped if the license is free
	skipTeamsOnFreeLicense bool,
) error {
	level.Debug(logger).Log("msg", "Applying starter library")

	// Create a request with context for downloading the starter library
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, starterLibraryURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for starter library: %w", err)
	}

	// Download the starter library from GitHub using the provided HTTP client factory
	httpClient := httpClientFactory(fleethttp.WithTimeout(5 * time.Second))
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download starter library: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download starter library, status: %d", resp.StatusCode)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read starter library response body: %w", err)
	}

	// Create a temporary directory to store downloaded scripts
	tempDir, err := os.MkdirTemp("", "fleet-scripts-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the temporary directory when done

	level.Debug(logger).Log("msg", "Created temporary directory for scripts", "path", tempDir)

	// Parse the YAML content into specs
	specs, err := spec.GroupFromBytes(buf)
	if err != nil {
		return fmt.Errorf("failed to parse starter library: %w", err)
	}

	// Find all script references in the YAML and download them
	scriptNames := ExtractScriptNames(specs)
	level.Debug(logger).Log("msg", "Found script references in starter library", "count", len(scriptNames))

	// Download scripts and update references in specs
	if len(scriptNames) > 0 {
		err = DownloadAndUpdateScripts(ctx, specs, scriptNames, tempDir, logger)
		if err != nil {
			return fmt.Errorf("failed to download and update scripts: %w", err)
		}
	}

	// Create an authenticated client and apply specs using the provided client factory
	client, err := clientFactory(serverURL, true, "", "")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	client.SetToken(token)

	// Check if we should skip teams on free license
	if skipTeamsOnFreeLicense {
		// Check if license is free
		appConfig, err := client.GetAppConfig()
		if err != nil {
			level.Debug(logger).Log("msg", "Error getting app config", "err", err)
			// Continue even if there's an error getting the app config
		} else if appConfig.License == nil || !appConfig.License.IsPremium() {
			// Remove teams from specs to avoid applying them
			level.Debug(logger).Log("msg", "Free license detected, skipping teams and team-related content in starter library")
			specs.Teams = nil

			// Filter out policies that reference teams
			if specs.Policies != nil {
				var filteredPolicies []*fleet.PolicySpec
				for _, policy := range specs.Policies {
					// Keep only policies that don't reference a team
					if policy.Team == "" {
						filteredPolicies = append(filteredPolicies, policy)
					}
				}
				specs.Policies = filteredPolicies
			}

			// Note: QuerySpec doesn't have a Team field, so we can't filter queries by team

			// Remove scripts from AppConfig if present
			if specs.AppConfig != nil {
				appConfigMap, ok := specs.AppConfig.(map[string]interface{})
				if ok {
					// Remove scripts from AppConfig
					delete(appConfigMap, "scripts")
				}
			}
		}
	}

	// Log function for ApplyGroup (minimal logging)
	logf := func(format string, a ...interface{}) {}

	// Assign the real implementation
	var applyGroupFn applyGroupFunc = func(ctx context.Context, specs *spec.Group) error {
		teamsSoftwareInstallers := make(map[string][]fleet.SoftwarePackageResponse)
		teamsScripts := make(map[string][]fleet.ScriptResponse)
		teamsVPPApps := make(map[string][]fleet.VPPAppResponse)

		_, _, _, _, err := client.ApplyGroup(
			ctx,
			false,
			specs,
			tempDir,
			logf,
			nil,
			fleet.ApplyClientSpecOptions{},
			teamsSoftwareInstallers,
			teamsVPPApps,
			teamsScripts,
		)
		return err
	}

	// Apply mock if mockApplyGroup is supplied
	if mockApplyGroup != nil {
		applyGroupFn = mockApplyGroup
	}

	if err := applyGroupFn(ctx, specs); err != nil {
		return fmt.Errorf("failed to apply starter library: %w", err)
	}

	level.Debug(logger).Log("msg", "Starter library applied successfully")
	return nil
}

// ExtractScriptNames extracts all script names from the specs
func ExtractScriptNames(specs *spec.Group) []string {
	var scriptNames []string
	scriptMap := make(map[string]bool) // Use a map to deduplicate script names

	// Process team specs
	for _, teamRaw := range specs.Teams {
		var teamData map[string]interface{}
		if err := json.Unmarshal(teamRaw, &teamData); err != nil {
			continue // Skip if we can't unmarshal
		}

		if scripts, ok := teamData["scripts"].([]interface{}); ok {
			for _, script := range scripts {
				if scriptName, ok := script.(string); ok && !scriptMap[scriptName] {
					scriptMap[scriptName] = true
					scriptNames = append(scriptNames, scriptName)
				}
			}
		}
	}

	return scriptNames
}

// DownloadAndUpdateScripts downloads scripts from URLs and updates the specs to reference local files
func DownloadAndUpdateScripts(ctx context.Context, specs *spec.Group, scriptNames []string, tempDir string, logger kitlog.Logger) error {
	// Create a single HTTP client to be reused for all requests
	httpClient := fleethttp.NewClient(fleethttp.WithTimeout(5 * time.Second))

	// Map to store local paths for each script
	scriptPaths := make(map[string]string, len(scriptNames))

	// Download each script sequentially
	for _, scriptName := range scriptNames {
		// Sanitize the script name to prevent path traversal
		sanitizedName := filepath.Clean(scriptName)
		if strings.HasPrefix(sanitizedName, "..") || filepath.IsAbs(sanitizedName) {
			return fmt.Errorf("invalid script name %s: must be a relative path", scriptName)
		}

		localPath := filepath.Join(tempDir, sanitizedName)
		scriptPaths[scriptName] = localPath

		// Create parent directories if they don't exist
		parentDir := filepath.Dir(localPath)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return fmt.Errorf("failed to create parent directories for script %s: %w", scriptName, err)
		}

		scriptURL := fmt.Sprintf("%s/%s", scriptsBaseURL, scriptName)
		level.Debug(logger).Log("msg", "Downloading script", "name", scriptName, "url", scriptURL, "local_path", localPath)

		// Create the request with context
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, scriptURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request for script %s: %w", scriptName, err)
		}

		// Download the script using the shared HTTP client
		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download script %s: %w", scriptName, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download script %s, status: %d", scriptName, resp.StatusCode)
		}

		// Create the local file
		file, err := os.Create(localPath)
		if err != nil {
			return fmt.Errorf("failed to create local file for script %s: %w", scriptName, err)
		}
		defer file.Close()

		// Copy the content to the local file
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write script %s to local file: %w", scriptName, err)
		}
	}

	// Read script contents and store them in memory
	scriptContents := make(map[string][]byte, len(scriptNames))
	for _, scriptName := range scriptNames {
		localPath := scriptPaths[scriptName]
		content, err := os.ReadFile(localPath)
		if err != nil {
			return fmt.Errorf("failed to read script %s from local file: %w", scriptName, err)
		}
		scriptContents[scriptName] = content
	}

	// Extract scripts from AppConfig if present
	appConfigScripts := extractAppCfgScripts(specs.AppConfig)
	if appConfigScripts != nil {
		// Replace script paths with actual script contents
		appScripts := make([]string, 0, len(appConfigScripts))
		for _, scriptPath := range appConfigScripts {
			if content, exists := scriptContents[scriptPath]; exists {
				// Create a temporary file with the script content
				tempFile, err := os.CreateTemp(tempDir, "script-*")
				if err != nil {
					return fmt.Errorf("failed to create temporary script file: %w", err)
				}
				if _, err := tempFile.Write(content); err != nil {
					tempFile.Close()
					return fmt.Errorf("failed to write script content to temporary file: %w", err)
				}
				tempFile.Close()

				// Add the temporary file path to the list
				appScripts = append(appScripts, tempFile.Name())
			} else {
				// Keep the original path if it's not one of our downloaded scripts
				appScripts = append(appScripts, scriptPath)
			}
		}

		// Update the AppConfig with the new script paths
		if specs.AppConfig != nil {
			specs.AppConfig.(map[string]interface{})["scripts"] = appScripts
		}
	}

	// Update script references in the specs to point to local files
	for i, teamRaw := range specs.Teams {
		var teamData map[string]interface{}
		if err := json.Unmarshal(teamRaw, &teamData); err != nil {
			continue // Skip if we can't unmarshal
		}

		if scripts, ok := teamData["scripts"].([]interface{}); ok {
			for j, script := range scripts {
				if scriptName, ok := script.(string); ok {
					// Update the script reference to the local path from our map
					if localPath, exists := scriptPaths[scriptName]; exists {
						scripts[j] = localPath
					}
				}
			}

			// Update the team data with modified scripts
			teamData["scripts"] = scripts

			// Marshal back to JSON
			updatedTeamRaw, err := json.Marshal(teamData)
			if err != nil {
				level.Debug(logger).Log("msg", "Failed to marshal updated team data", "err", err)
				continue
			}

			// Update the team in the specs
			specs.Teams[i] = updatedTeamRaw
		}
	}

	return nil
}
