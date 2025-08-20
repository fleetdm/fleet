package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/text/unicode/norm"
)

type ParseTypeError struct {
	Filename string   // The name of the file being parsed
	Keys     []string // The complete path to the field
	Field    string   // The field we tried to assign to
	Type     string   // The type that we want to have
	Value    string   // The type of the value that we received
	err      error    // The original error
}

func (e *ParseTypeError) Error() string {
	var keyPath []string
	keyPath = append(keyPath, e.Keys...)
	if e.Field != "" {
		var clearFields []string
		fields := strings.Split(e.Field, ".")
	fieldcheck:
		for _, field := range fields {
			for _, r := range field {
				// Any field name that contains an upper case letter is probably an embedded struct,
				// remove it from the field path to reduce end user confusion
				if unicode.IsUpper(r) {
					continue fieldcheck
				}
			}
			clearFields = append(clearFields, field)
		}
		keyPath = append(keyPath, strings.Join(clearFields, "."))
	}
	return fmt.Sprintf("Couldn't edit \"%s\" at \"%s\", expected type %s but got %s", e.Filename, strings.Join(keyPath, "."), e.Type, e.Value)
}

func (e *ParseTypeError) Unwrap() error {
	return e.err
}

func MaybeParseTypeError(filename string, keysPath []string, err error) error {
	unmarshallErr := &json.UnmarshalTypeError{}
	if errors.As(err, &unmarshallErr) {
		return &ParseTypeError{
			Filename: filename,
			Keys:     keysPath,
			Field:    unmarshallErr.Field,
			Type:     unmarshallErr.Type.String(),
			Value:    unmarshallErr.Value,
			err:      err,
		}
	}

	return fmt.Errorf("failed to unmarshal file \"%s\" key \"%s\": %w", filename, strings.Join(keysPath, "."), err)
}

// YamlUnmarshal unmarshals YAML bytes into JSON and then into the output struct. We have to do this
// because the yaml package stringifys the JSON parsing error before returning it, so we can't
// extract field information to produce a helpful error for users.
func YamlUnmarshal(yamlBytes []byte, out any) error {
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML to JSON: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, out); err != nil {
		return fmt.Errorf("failed to unmarshal JSON bytes: %w", err)
	}

	return nil
}

type BaseItem struct {
	Path *string `json:"path"`
}

type GitOpsControls struct {
	BaseItem
	MacOSUpdates   interface{}       `json:"macos_updates"`
	IOSUpdates     interface{}       `json:"ios_updates"`
	IPadOSUpdates  interface{}       `json:"ipados_updates"`
	MacOSSettings  interface{}       `json:"macos_settings"`
	MacOSSetup     *fleet.MacOSSetup `json:"macos_setup"`
	MacOSMigration interface{}       `json:"macos_migration"`

	WindowsUpdates              interface{} `json:"windows_updates"`
	WindowsSettings             interface{} `json:"windows_settings"`
	WindowsEnabledAndConfigured interface{} `json:"windows_enabled_and_configured"`
	WindowsMigrationEnabled     interface{} `json:"windows_migration_enabled"`

	EnableDiskEncryption interface{} `json:"enable_disk_encryption"`
	RequireBitLockerPIN  interface{} `json:"windows_require_bitlocker_pin,omitempty"`

	Scripts []BaseItem `json:"scripts"`

	Defined bool
}

func (c GitOpsControls) Set() bool {
	return c.MacOSUpdates != nil || c.IOSUpdates != nil ||
		c.IPadOSUpdates != nil || c.MacOSSettings != nil ||
		c.MacOSSetup != nil || c.MacOSMigration != nil ||
		c.WindowsUpdates != nil || c.WindowsSettings != nil || c.WindowsEnabledAndConfigured != nil ||
		c.WindowsMigrationEnabled != nil || c.EnableDiskEncryption != nil || len(c.Scripts) > 0
}

type Policy struct {
	BaseItem
	GitOpsPolicySpec
}

type GitOpsPolicySpec struct {
	fleet.PolicySpec
	RunScript       *PolicyRunScript       `json:"run_script"`
	InstallSoftware *PolicyInstallSoftware `json:"install_software"`
	// InstallSoftwareURL is populated after parsing the software installer yaml
	// referenced by InstallSoftware.PackagePath.
	InstallSoftwareURL string `json:"-"`
	// RunScriptName is populated after confirming the script exists on both the file system
	// and in the controls scripts list for the same team
	RunScriptName *string `json:"-"`
}

type PolicyRunScript struct {
	Path string `json:"path"`
}

type PolicyInstallSoftware struct {
	PackagePath string `json:"package_path"`
	AppStoreID  string `json:"app_store_id"`
	HashSHA256  string `json:"hash_sha256"`
}

type Query struct {
	BaseItem
	fleet.QuerySpec
}

type Label struct {
	BaseItem
	fleet.LabelSpec
}

type SoftwarePackage struct {
	BaseItem
	fleet.SoftwarePackageSpec
}

type Software struct {
	Packages            []SoftwarePackage           `json:"packages"`
	AppStoreApps        []fleet.TeamSpecAppStoreApp `json:"app_store_apps"`
	FleetMaintainedApps []fleet.MaintainedAppSpec   `json:"fleet_maintained_apps"`
}

type GitOps struct {
	TeamID       *uint
	TeamName     *string
	TeamSettings map[string]interface{}
	OrgSettings  map[string]interface{}
	AgentOptions *json.RawMessage
	Controls     GitOpsControls
	Policies     []*GitOpsPolicySpec
	Queries      []*fleet.QuerySpec
	Labels       []*fleet.LabelSpec
	// Software is only allowed on teams, not on global config.
	Software GitOpsSoftware
	// FleetSecrets is a map of secret names to their values, extracted from FLEET_SECRET_ environment variables used in profiles and scripts.
	FleetSecrets map[string]string
}

type GitOpsSoftware struct {
	Packages            []*fleet.SoftwarePackageSpec
	AppStoreApps        []*fleet.TeamSpecAppStoreApp
	FleetMaintainedApps []*fleet.MaintainedAppSpec
}

type Logf func(format string, a ...interface{})

// GitOpsFromFile parses a GitOps yaml file.
func GitOpsFromFile(filePath, baseDir string, appConfig *fleet.EnrichedAppConfig, logFn Logf) (*GitOps, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s: %w", filePath, err)
	}

	// Replace $var and ${var} with env values.
	b, err = ExpandEnvBytes(b)
	if err != nil {
		return nil, fmt.Errorf("failed to expand environment in file %s: %w", filePath, err)
	}

	var top map[string]json.RawMessage
	if err := yaml.Unmarshal(b, &top); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %w: \n", err)
	}

	var multiError *multierror.Error
	result := &GitOps{}
	result.FleetSecrets = make(map[string]string)

	topKeys := []string{"name", "team_settings", "org_settings", "agent_options", "controls", "policies", "queries", "software", "labels"}
	for k := range top {
		if !slices.Contains(topKeys, k) {
			multiError = multierror.Append(multiError, fmt.Errorf("unknown top-level field: %s", k))
		}
	}

	// Figure out if this is an org or team settings file
	teamRaw, teamOk := top["name"]
	teamSettingsRaw, teamSettingsOk := top["team_settings"]
	orgSettingsRaw, orgOk := top["org_settings"]
	switch {
	case orgOk:
		if teamOk || teamSettingsOk {
			multiError = multierror.Append(multiError, errors.New("'org_settings' cannot be used with 'name', 'team_settings'"))
		} else {
			multiError = parseOrgSettings(orgSettingsRaw, result, baseDir, filePath, multiError)
		}
	case teamOk:
		multiError = parseName(teamRaw, result, filePath, multiError)
		if result.IsNoTeam() {
			if teamSettingsOk {
				multiError = multierror.Append(multiError, fmt.Errorf("cannot set 'team_settings' on 'No team' file: %q", filePath))
			}
			if filepath.Base(filePath) != "no-team.yml" {
				multiError = multierror.Append(multiError, fmt.Errorf("file %q for 'No team' must be named 'no-team.yml'", filePath))
			}
		} else {
			if !teamSettingsOk {
				multiError = multierror.Append(multiError, errors.New("'team_settings' is required when 'name' is provided"))
			} else {
				multiError = parseTeamSettings(teamSettingsRaw, result, baseDir, filePath, multiError)
			}
		}
	default:
		multiError = multierror.Append(multiError, errors.New("either 'org_settings' or 'name' and 'team_settings' is required"))
	}

	// Get the labels. If `labels:` is specified but no labels are listed, this will
	// set Labels as nil.  If `labels:` isn't present at all, it will be set as an
	// empty array.
	_, ok := top["labels"]
	if !ok || !result.IsGlobal() {
		if ok && !result.IsGlobal() {
			logFn("[!] 'labels' is only supported in global settings.  This key will be ignored.\n")
		}
		result.Labels = make([]*fleet.LabelSpec, 0)
	} else {
		multiError = parseLabels(top, result, baseDir, filePath, multiError)
	}
	// Get other top-level entities.
	multiError = parseControls(top, result, multiError, filePath)
	multiError = parseAgentOptions(top, result, baseDir, logFn, filePath, multiError)
	multiError = parseQueries(top, result, baseDir, logFn, filePath, multiError)

	if appConfig != nil && appConfig.License.IsPremium() {
		multiError = parseSoftware(top, result, baseDir, filePath, multiError)
	}

	// Policies can reference software installers and scripts, thus we parse them after parseSoftware and parseControls.
	multiError = parsePolicies(top, result, baseDir, filePath, multiError)

	return result, multiError.ErrorOrNil()
}

func parseName(raw json.RawMessage, result *GitOps, filePath string, multiError *multierror.Error) *multierror.Error {
	if err := json.Unmarshal(raw, &result.TeamName); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"name"}, err))
	}
	if result.TeamName == nil || *result.TeamName == "" {
		return multierror.Append(multiError, errors.New("team 'name' is required"))
	}
	// Normalize team name for full Unicode support, so that we can assume team names are unique going forward
	normalized := norm.NFC.String(*result.TeamName)
	result.TeamName = &normalized
	return multiError
}

func (g *GitOps) global() bool {
	return g.TeamName == nil || *g.TeamName == ""
}

func (g *GitOps) IsGlobal() bool {
	return g.global()
}

func (g *GitOps) IsNoTeam() bool {
	return g.TeamName != nil && isNoTeam(*g.TeamName)
}

func isNoTeam(teamName string) bool {
	return strings.EqualFold(teamName, noTeam)
}

const noTeam = "No team"

func parseOrgSettings(raw json.RawMessage, result *GitOps, baseDir string, filePath string, multiError *multierror.Error) *multierror.Error {
	var orgSettingsTop BaseItem
	if err := json.Unmarshal(raw, &orgSettingsTop); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"org_settings"}, err))
	}
	noError := true
	if orgSettingsTop.Path != nil {
		fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *orgSettingsTop.Path))
		if err != nil {
			noError = false
			multiError = multierror.Append(multiError, fmt.Errorf("failed to read org settings file %s: %v", *orgSettingsTop.Path, err))
		} else {
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				noError = false
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *orgSettingsTop.Path, err),
				)
			} else {
				var pathOrgSettings BaseItem
				if err := YamlUnmarshal(fileBytes, &pathOrgSettings); err != nil {
					noError = false
					multiError = multierror.Append(
						multiError, MaybeParseTypeError(*orgSettingsTop.Path, []string{"org_settings"}, err),
					)
				} else {
					if pathOrgSettings.Path != nil {
						noError = false
						multiError = multierror.Append(
							multiError,
							fmt.Errorf("nested paths are not supported: %s in %s", *pathOrgSettings.Path, *orgSettingsTop.Path),
						)
					} else {
						raw = fileBytes
					}
				}
			}
		}
	}
	if noError {
		if err := YamlUnmarshal(raw, &result.OrgSettings); err != nil {
			// This error is currently unreachable because we know the file is valid YAML when we checked for nested path
			multiError = multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"org_settings"}, err))
		} else {
			multiError = parseSecrets(result, multiError)
		}
		// TODO: Validate that integrations.(jira|zendesk)[].api_token is not empty or fleet.MaskedPassword
	}
	return multiError
}

func parseTeamSettings(raw json.RawMessage, result *GitOps, baseDir string, filePath string, multiError *multierror.Error) *multierror.Error {
	var teamSettingsTop BaseItem
	if err := json.Unmarshal(raw, &teamSettingsTop); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"team_settings"}, err))
	}
	noError := true
	if teamSettingsTop.Path != nil {
		fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *teamSettingsTop.Path))
		if err != nil {
			noError = false
			multiError = multierror.Append(multiError, fmt.Errorf("failed to read team settings file %s: %v", *teamSettingsTop.Path, err))
		} else {
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				noError = false
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *teamSettingsTop.Path, err),
				)
			} else {
				var pathTeamSettings BaseItem
				if err := YamlUnmarshal(fileBytes, &pathTeamSettings); err != nil {
					noError = false
					multiError = multierror.Append(
						multiError, MaybeParseTypeError(*teamSettingsTop.Path, []string{"team_settings"}, err),
					)
				} else {
					if pathTeamSettings.Path != nil {
						noError = false
						multiError = multierror.Append(
							multiError,
							fmt.Errorf("nested paths are not supported: %s in %s", *pathTeamSettings.Path, *teamSettingsTop.Path),
						)
					} else {
						raw = fileBytes
					}
				}
			}
		}
	}
	if noError {
		if err := YamlUnmarshal(raw, &result.TeamSettings); err != nil {
			// This error is currently unreachable because we know the file is valid YAML when we checked for nested path
			multiError = multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"team_settings"}, err))
		} else {
			multiError = parseSecrets(result, multiError)
		}
	}
	return multiError
}

func parseSecrets(result *GitOps, multiError *multierror.Error) *multierror.Error {
	var rawSecrets interface{}
	var ok bool
	if result.TeamName == nil {
		rawSecrets, ok = result.OrgSettings["secrets"]
		if !ok {
			return multierror.Append(multiError, errors.New("'org_settings.secrets' is required"))
		}
	} else {
		rawSecrets, ok = result.TeamSettings["secrets"]
		if !ok {
			return multierror.Append(multiError, errors.New("'team_settings.secrets' is required"))
		}
	}
	// When secrets slice is empty, all secrets are removed.
	enrollSecrets := make([]*fleet.EnrollSecret, 0)
	if rawSecrets != nil {
		secrets, ok := rawSecrets.([]interface{})
		if !ok {
			return multierror.Append(multiError, errors.New("'secrets' must be a list of secret items"))
		}
		for _, enrollSecret := range secrets {
			var secret string
			var secretInterface interface{}
			secretMap, ok := enrollSecret.(map[string]interface{})
			if ok {
				secretInterface, ok = secretMap["secret"]
			}
			if ok {
				secret, ok = secretInterface.(string)
			}
			if !ok || secret == "" {
				multiError = multierror.Append(
					multiError, errors.New("each item in 'secrets' must have a 'secret' key containing an ASCII string value"),
				)
				break
			}
			enrollSecrets = append(
				enrollSecrets, &fleet.EnrollSecret{Secret: secret},
			)
		}
	}
	if result.TeamName == nil {
		result.OrgSettings["secrets"] = enrollSecrets
	} else {
		result.TeamSettings["secrets"] = enrollSecrets
	}
	return multiError
}

func parseAgentOptions(top map[string]json.RawMessage, result *GitOps, baseDir string, logFn Logf, filePath string, multiError *multierror.Error) *multierror.Error {
	agentOptionsRaw, ok := top["agent_options"]
	if result.IsNoTeam() {
		if ok {
			logFn("[!] 'agent_options' is not supported for \"No team\". This key will be ignored.\n")
		}
		return multiError
	} else if !ok {
		return multierror.Append(multiError, errors.New("'agent_options' is required"))
	}
	var agentOptionsTop BaseItem
	if err := json.Unmarshal(agentOptionsRaw, &agentOptionsTop); err != nil {
		multiError = multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"agent_options"}, err))
	} else {
		if agentOptionsTop.Path == nil {
			result.AgentOptions = &agentOptionsRaw
		} else {
			fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *agentOptionsTop.Path))
			if err != nil {
				return multierror.Append(multiError, fmt.Errorf("failed to read agent options file %s: %v", *agentOptionsTop.Path, err))
			}
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *agentOptionsTop.Path, err),
				)
			} else {
				var pathAgentOptions BaseItem
				if err := YamlUnmarshal(fileBytes, &pathAgentOptions); err != nil {
					return multierror.Append(
						multiError, MaybeParseTypeError(*agentOptionsTop.Path, []string{"agent_options"}, err),
					)
				}
				if pathAgentOptions.Path != nil {
					return multierror.Append(
						multiError,
						fmt.Errorf("nested paths are not supported: %s in %s", *pathAgentOptions.Path, *agentOptionsTop.Path),
					)
				}
				var raw json.RawMessage
				if err := YamlUnmarshal(fileBytes, &raw); err != nil {
					// This error is currently unreachable because we know the file is valid YAML when we checked for nested path
					return multierror.Append(
						multiError, MaybeParseTypeError(*agentOptionsTop.Path, []string{"agent_options"}, err),
					)
				}
				result.AgentOptions = &raw
			}
		}
	}
	return multiError
}

func parseControls(top map[string]json.RawMessage, result *GitOps, multiError *multierror.Error, yamlFilename string) *multierror.Error {
	controlsRaw, ok := top["controls"]
	if !ok {
		// Nothing to do, return.
		return multiError
	}

	var controlsTop GitOpsControls
	if err := json.Unmarshal(controlsRaw, &controlsTop); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(yamlFilename, []string{"controls"}, err))
	}
	controlsTop.Defined = true
	controlsFilePath := yamlFilename
	err := processControlsPathIfNeeded(controlsTop, result, &controlsFilePath)
	if err != nil {
		return multierror.Append(multiError, err)
	}

	controlsDir := filepath.Dir(controlsFilePath)
	result.Controls.Scripts, err = resolveScriptPaths(result.Controls.Scripts, controlsDir)
	if err != nil {
		return multierror.Append(multiError, fmt.Errorf("failed to parse scripts list in %s: %v", controlsFilePath, err))
	}

	// Find Fleet secrets in scripts.
	for _, script := range result.Controls.Scripts {
		if script.Path == nil {
			// This should never happen because we checked for missing paths above (with code added in https://github.com/fleetdm/fleet/pull/24639).
			return multierror.Append(multiError, errors.New("controls.scripts.path is missing"))
		}
		fileBytes, err := os.ReadFile(*script.Path)
		if err != nil {
			return multierror.Append(multiError, fmt.Errorf("failed to read scripts file %s: %v", *script.Path, err))
		}
		err = LookupEnvSecrets(string(fileBytes), result.FleetSecrets)
		if err != nil {
			return multierror.Append(multiError, err)
		}
	}

	// Find the Fleet Secrets in the macos setup script file
	if result.Controls.MacOSSetup != nil {
		if result.Controls.MacOSSetup.Script.Set {
			startupScriptPath := resolveApplyRelativePath(controlsDir, result.Controls.MacOSSetup.Script.Value)
			fileBytes, err := os.ReadFile(startupScriptPath)
			if err != nil {
				return multierror.Append(multiError, fmt.Errorf("failed to read macos_setup script file %s: %v", startupScriptPath, err))
			}
			err = LookupEnvSecrets(string(fileBytes), result.FleetSecrets)
			if err != nil {
				return multierror.Append(multiError, err)
			}
		}
	}

	// Find Fleet secrets in profiles
	if result.Controls.MacOSSettings != nil {
		// We are marshalling/unmarshalling to get the data into the fleet.MacOSSettings struct.
		// This is inefficient, but it is more robust and less error-prone.
		var macOSSettings fleet.MacOSSettings
		data, err := json.Marshal(result.Controls.MacOSSettings)
		if err != nil {
			return multierror.Append(multiError, fmt.Errorf("failed to process controls.macos_settings: %v", err))
		}
		err = json.Unmarshal(data, &macOSSettings)
		if err != nil {
			return multierror.Append(multiError, MaybeParseTypeError(controlsFilePath, []string{"controls", "macos_settings"}, err))
		}

		for i := range macOSSettings.CustomSettings {
			err := resolveAndUpdateProfilePathToAbsolute(controlsDir, &macOSSettings.CustomSettings[i], result)
			if err != nil {
				return multierror.Append(multiError, err)
			}
		}
		// Since we already unmarshalled and updated the path, we need to update the result struct.
		result.Controls.MacOSSettings = macOSSettings
	}
	if result.Controls.WindowsSettings != nil {
		// We are marshalling/unmarshalling to get the data into the fleet.WindowsSettings struct.
		// This is inefficient, but it is more robust and less error-prone.
		var windowsSettings fleet.WindowsSettings
		data, err := json.Marshal(result.Controls.WindowsSettings)
		if err != nil {
			return multierror.Append(multiError, fmt.Errorf("failed to process controls.windows_settings: %v", err))
		}
		err = json.Unmarshal(data, &windowsSettings)
		if err != nil {
			return multierror.Append(multiError, MaybeParseTypeError(controlsFilePath, []string{"controls", "windows_settings"}, err))
		}
		if windowsSettings.CustomSettings.Valid {
			for i := range windowsSettings.CustomSettings.Value {
				err := resolveAndUpdateProfilePathToAbsolute(controlsDir, &windowsSettings.CustomSettings.Value[i], result)
				if err != nil {
					return multierror.Append(multiError, err)
				}
			}
		}
		// Since we already unmarshalled and updated the path, we need to update the result struct.
		result.Controls.WindowsSettings = windowsSettings
	}

	return multiError
}

func processControlsPathIfNeeded(controlsTop GitOpsControls, result *GitOps, controlsFilePath *string) error {
	if controlsTop.Path == nil {
		result.Controls = controlsTop
		return nil
	}

	// There is a path attribute which points to the real controls section in a separate file, so we need to process that.
	controlsFilePath = ptr.String(resolveApplyRelativePath(filepath.Dir(*controlsFilePath), *controlsTop.Path))
	fileBytes, err := os.ReadFile(*controlsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read controls file %s: %v", *controlsTop.Path, err)
	}

	// Replace $var and ${var} with env values.
	fileBytes, err = ExpandEnvBytes(fileBytes)
	if err != nil {
		return fmt.Errorf("failed to expand environment in file %s: %v", *controlsTop.Path, err)
	}

	var pathControls GitOpsControls
	if err := YamlUnmarshal(fileBytes, &pathControls); err != nil {
		return MaybeParseTypeError(*controlsTop.Path, []string{"controls"}, err)
	}
	if pathControls.Path != nil {
		return fmt.Errorf("nested paths are not supported: %s in %s", *pathControls.Path, *controlsTop.Path)
	}
	pathControls.Defined = true
	result.Controls = pathControls
	return nil
}

func resolveAndUpdateProfilePathToAbsolute(controlsDir string, profile *fleet.MDMProfileSpec, result *GitOps) error {
	resolvedPath := resolveApplyRelativePath(controlsDir, profile.Path)
	// We switch to absolute path so that we don't have to keep track of the base directory.
	// This is useful because controls section can come from either the global config file or the no-team file.
	var err error
	profile.Path, err = filepath.Abs(resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to resolve profile path %s: %v", resolvedPath, err)
	}
	fileBytes, err := os.ReadFile(resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to read profile file %s: %v", resolvedPath, err)
	}
	err = LookupEnvSecrets(string(fileBytes), result.FleetSecrets)
	if err != nil {
		return err
	}
	return nil
}

func resolveScriptPaths(input []BaseItem, baseDir string) ([]BaseItem, error) {
	var resolved []BaseItem
	for _, item := range input {
		if item.Path == nil {
			return nil, errors.New(`script entry was specified without a path; check for a stray "-" in your scripts list`)
		}

		resolvedPath := resolveApplyRelativePath(baseDir, *item.Path)
		item.Path = &resolvedPath
		resolved = append(resolved, item)
	}

	return resolved, nil
}

func parseLabels(top map[string]json.RawMessage, result *GitOps, baseDir string, filePath string, multiError *multierror.Error) *multierror.Error {
	labelsRaw, ok := top["labels"]

	// This shouldn't happen as we check for the property earlier,
	// but better safe than sorry.
	if !ok {
		return multiError
	}

	var labels []Label
	if err := json.Unmarshal(labelsRaw, &labels); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"labels"}, err))
	}
	for _, item := range labels {
		item := item
		if item.Path == nil {
			result.Labels = append(result.Labels, &item.LabelSpec)
		} else {
			fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *item.Path))
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to read labels file %s: %v", *item.Path, err))
				continue
			}
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *item.Path, err),
				)
			} else {
				var pathLabels []*Label
				if err := YamlUnmarshal(fileBytes, &pathLabels); err != nil {
					multiError = multierror.Append(multiError, MaybeParseTypeError(*item.Path, []string{"labels"}, err))
					continue
				}
				for _, pq := range pathLabels {
					pq := pq
					if pq != nil {
						if pq.Path != nil {
							multiError = multierror.Append(
								multiError, fmt.Errorf("nested paths are not supported: %s in %s", *pq.Path, *item.Path),
							)
						} else {
							result.Labels = append(result.Labels, &pq.LabelSpec)
						}
					}
				}
			}
		}
	}
	// Make sure team name is correct and do additional validation
	for _, l := range result.Labels {
		if l.Name == "" {
			multiError = multierror.Append(multiError, errors.New("name is required for each label"))
		}

		if l.LabelMembershipType != fleet.LabelMembershipTypeManual && l.Query == "" && l.HostVitalsCriteria == nil {
			multiError = multierror.Append(multiError, errors.New("a SQL query or host vitals criteria is required for each non-manual label"))
		}

		// Manual labels can have empty hosts lists, just make sure we initialize the empty list
		if l.LabelMembershipType == fleet.LabelMembershipTypeManual && l.Hosts == nil {
			l.Hosts = []string{}
		}
		// Don't use non-ASCII
		if !isASCII(l.Name) {
			multiError = multierror.Append(multiError, fmt.Errorf("label name must be in ASCII: %s", l.Name))
		}
		// Check that host vitals criteria is valid
		if l.HostVitalsCriteria != nil {
			criteriaJson, err := json.Marshal(l.HostVitalsCriteria)
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to marshal host vitals criteria for label %s: %v", l.Name, err))
				continue
			}
			label := fleet.Label{
				Name:               l.Name,
				HostVitalsCriteria: ptr.RawMessage(criteriaJson),
			}
			if _, _, err := label.CalculateHostVitalsQuery(); err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("invalid host vitals criteria for label %s: %v", l.Name, err))
			}
		}
	}
	duplicates := getDuplicateNames(
		result.Labels, func(l *fleet.LabelSpec) string {
			return l.Name
		},
	)
	if len(duplicates) > 0 {
		multiError = multierror.Append(multiError, fmt.Errorf("duplicate label names: %v", duplicates))
	}
	return multiError
}

func parsePolicies(top map[string]json.RawMessage, result *GitOps, baseDir string, filePath string, multiError *multierror.Error) *multierror.Error {
	policiesRaw, ok := top["policies"]
	if !ok {
		return multierror.Append(multiError, errors.New("'policies' key is required"))
	}
	var policies []Policy
	if err := json.Unmarshal(policiesRaw, &policies); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"policies"}, err))
	}
	for _, item := range policies {
		item := item
		if item.Path == nil {
			if err := parsePolicyInstallSoftware(baseDir, result.TeamName, &item, result.Software.Packages, result.Software.AppStoreApps); err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to parse policy install_software %q: %v", item.Name, err))
				continue
			}
			if err := parsePolicyRunScript(baseDir, result.TeamName, &item, result.Controls.Scripts); err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to parse policy run_script %q: %v", item.Name, err))
				continue
			}
			result.Policies = append(result.Policies, &item.GitOpsPolicySpec)
		} else {
			filePath := resolveApplyRelativePath(baseDir, *item.Path)
			fileBytes, err := os.ReadFile(filePath)
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to read policies file %s: %v", *item.Path, err))
				continue
			}
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *item.Path, err),
				)
			} else {
				var pathPolicies []*Policy
				if err := YamlUnmarshal(fileBytes, &pathPolicies); err != nil {
					multiError = multierror.Append(multiError, MaybeParseTypeError(*item.Path, []string{"policies"}, err))
					continue
				}
				for _, pp := range pathPolicies {
					pp := pp
					if pp != nil {
						if pp.Path != nil {
							multiError = multierror.Append(
								multiError, fmt.Errorf("nested paths are not supported: %s in %s", *pp.Path, *item.Path),
							)
						} else {
							if err := parsePolicyInstallSoftware(filepath.Dir(filePath), result.TeamName, pp, result.Software.Packages, result.Software.AppStoreApps); err != nil {
								multiError = multierror.Append(multiError, fmt.Errorf("failed to parse policy install_software %q: %v", pp.Name, err))
								continue
							}
							if err := parsePolicyRunScript(filepath.Dir(filePath), result.TeamName, pp, result.Controls.Scripts); err != nil {
								multiError = multierror.Append(multiError, fmt.Errorf("failed to parse policy run_script %q: %v", pp.Name, err))
								continue
							}
							result.Policies = append(result.Policies, &pp.GitOpsPolicySpec)
						}
					}
				}
			}
		}
	}
	// Make sure team name is correct, and do additional validation
	for _, item := range result.Policies {
		if item.Name == "" {
			multiError = multierror.Append(multiError, errors.New("policy name is required for each policy"))
		} else {
			item.Name = norm.NFC.String(item.Name)
		}
		if item.Query == "" {
			multiError = multierror.Append(multiError, errors.New("policy query is required for each policy"))
		}
		if result.TeamName != nil {
			item.Team = *result.TeamName
		} else {
			item.Team = ""
		}
		if item.CalendarEventsEnabled && result.IsNoTeam() {
			multiError = multierror.Append(multiError, fmt.Errorf("calendar events are not supported on \"No team\" policies: %q", item.Name))
		}
	}
	duplicates := getDuplicateNames(
		result.Policies, func(p *GitOpsPolicySpec) string {
			return p.Name
		},
	)
	if len(duplicates) > 0 {
		multiError = multierror.Append(multiError, fmt.Errorf("duplicate policy names: %v", duplicates))
	}
	return multiError
}

func parsePolicyRunScript(baseDir string, teamName *string, policy *Policy, scripts []BaseItem) error {
	if policy.RunScript == nil {
		policy.ScriptID = ptr.Uint(0) // unset the script
		return nil
	}
	if policy.RunScript != nil && policy.RunScript.Path != "" && teamName == nil {
		return errors.New("run_script can only be set on team policies")
	}

	if policy.RunScript.Path == "" {
		return errors.New("empty run_script path")
	}

	scriptPath := resolveApplyRelativePath(baseDir, policy.RunScript.Path)
	_, err := os.Stat(scriptPath)
	if err != nil {
		return fmt.Errorf("script file does not exist %q: %v", policy.RunScript.Path, err)
	}

	scriptOnTeamFound := false
	for _, script := range scripts {
		if scriptPath == *script.Path {
			scriptOnTeamFound = true
			break
		}
	}
	if !scriptOnTeamFound {
		if *teamName == noTeam {
			return fmt.Errorf("policy script %s was not defined in controls in no-team.yml", scriptPath)
		}
		return fmt.Errorf("policy script %s was not defined in controls for %s", scriptPath, *teamName)
	}

	scriptName := filepath.Base(policy.RunScript.Path)
	policy.RunScriptName = &scriptName

	return nil
}

func parsePolicyInstallSoftware(baseDir string, teamName *string, policy *Policy, packages []*fleet.SoftwarePackageSpec, appStoreApps []*fleet.TeamSpecAppStoreApp) error {
	if policy.InstallSoftware == nil {
		policy.SoftwareTitleID = ptr.Uint(0) // unset the installer
		return nil
	}
	if policy.InstallSoftware != nil && (policy.InstallSoftware.PackagePath != "" || policy.InstallSoftware.AppStoreID != "") && teamName == nil {
		return errors.New("install_software can only be set on team policies")
	}
	if policy.InstallSoftware.PackagePath == "" && policy.InstallSoftware.AppStoreID == "" && policy.InstallSoftware.HashSHA256 == "" {
		return errors.New("install_software must include either a package_path, an app_store_id or a hash_sha256")
	}
	if policy.InstallSoftware.PackagePath != "" && policy.InstallSoftware.AppStoreID != "" {
		return errors.New("install_software must have only one of package_path or app_store_id")
	}

	if policy.InstallSoftware.PackagePath != "" {
		fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, policy.InstallSoftware.PackagePath))
		if err != nil {
			return fmt.Errorf("failed to read install_software.package_path file %q: %v", policy.InstallSoftware.PackagePath, err)
		}
		var policyInstallSoftwareSpec fleet.SoftwarePackageSpec
		if err := YamlUnmarshal(fileBytes, &policyInstallSoftwareSpec); err != nil {
			return MaybeParseTypeError(policy.InstallSoftware.PackagePath, []string{"policy", "install_software", "package_path"}, err)
		}
		installerOnTeamFound := false
		for _, pkg := range packages {
			if (pkg.URL != "" && pkg.URL == policyInstallSoftwareSpec.URL) || (pkg.SHA256 != "" && pkg.SHA256 == policyInstallSoftwareSpec.SHA256) {
				installerOnTeamFound = true
				break
			}
		}
		if !installerOnTeamFound {
			if policyInstallSoftwareSpec.URL != "" {
				return fmt.Errorf("install_software.package_path URL %s not found on team: %s", policyInstallSoftwareSpec.URL, policy.InstallSoftware.PackagePath)
			}
			return fmt.Errorf("install_software.package_path SHA256 %s not found on team: %s", policyInstallSoftwareSpec.SHA256, policy.InstallSoftware.PackagePath)
		}

		policy.InstallSoftwareURL = policyInstallSoftwareSpec.URL
		policy.InstallSoftware.HashSHA256 = policyInstallSoftwareSpec.SHA256
	}

	if policy.InstallSoftware.AppStoreID != "" {
		appOnTeamFound := false
		for _, app := range appStoreApps {
			if app.AppStoreID == policy.InstallSoftware.AppStoreID {
				appOnTeamFound = true
				break
			}
		}
		if !appOnTeamFound {
			return fmt.Errorf("install_software.app_store_id %s not found on team %s", policy.InstallSoftware.AppStoreID, *teamName)
		}
	}

	return nil
}

func parseQueries(top map[string]json.RawMessage, result *GitOps, baseDir string, logFn Logf, filePath string, multiError *multierror.Error) *multierror.Error {
	queriesRaw, ok := top["queries"]
	if result.IsNoTeam() {
		if ok {
			logFn("[!] 'queries' is not supported for \"No team\". This key will be ignored.\n")
		}
		return multiError
	} else if !ok {
		return multierror.Append(multiError, errors.New("'queries' key is required"))
	}
	var queries []Query
	if err := json.Unmarshal(queriesRaw, &queries); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"queries"}, err))
	}
	for _, item := range queries {
		item := item
		if item.Path == nil {
			result.Queries = append(result.Queries, &item.QuerySpec)
		} else {
			fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *item.Path))
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to read queries file %s: %v", *item.Path, err))
				continue
			}
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *item.Path, err),
				)
			} else {
				var pathQueries []*Query
				if err := YamlUnmarshal(fileBytes, &pathQueries); err != nil {
					multiError = multierror.Append(multiError, MaybeParseTypeError(*item.Path, []string{"queries"}, err))
					continue
				}
				for _, pq := range pathQueries {
					pq := pq
					if pq != nil {
						if pq.Path != nil {
							multiError = multierror.Append(
								multiError, fmt.Errorf("nested paths are not supported: %s in %s", *pq.Path, *item.Path),
							)
						} else {
							result.Queries = append(result.Queries, &pq.QuerySpec)
						}
					}
				}
			}
		}
	}
	// Make sure team name is correct and do additional validation
	for _, q := range result.Queries {
		if q.Name == "" {
			multiError = multierror.Append(multiError, errors.New("query name is required for each query"))
		}
		if q.Query == "" {
			multiError = multierror.Append(multiError, errors.New("query SQL query is required for each query"))
		}
		// Don't use non-ASCII
		if !isASCII(q.Name) {
			multiError = multierror.Append(multiError, fmt.Errorf("query name must be in ASCII: %s", q.Name))
		}
		if result.TeamName != nil {
			q.TeamName = *result.TeamName
		} else {
			q.TeamName = ""
		}
	}
	duplicates := getDuplicateNames(
		result.Queries, func(q *fleet.QuerySpec) string {
			return q.Name
		},
	)
	if len(duplicates) > 0 {
		multiError = multierror.Append(multiError, fmt.Errorf("duplicate query names: %v", duplicates))
	}
	return multiError
}

var validSHA256Value = regexp.MustCompile(`\b[a-f0-9]{64}\b`)

func parseSoftware(top map[string]json.RawMessage, result *GitOps, baseDir string, filePath string, multiError *multierror.Error) *multierror.Error {
	softwareRaw, ok := top["software"]
	if result.global() {
		if ok && string(softwareRaw) != "null" {
			return multierror.Append(multiError, errors.New("'software' cannot be set on global file"))
		}
	} else if !ok {
		return multierror.Append(multiError, errors.New("'software' is required"))
	}
	var software Software
	if len(softwareRaw) > 0 {
		if err := json.Unmarshal(softwareRaw, &software); err != nil {
			return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"software"}, err))
		}
	}
	for _, item := range software.AppStoreApps {
		item := item
		if item.AppStoreID == "" {
			multiError = multierror.Append(multiError, errors.New("software app store id required"))
			continue
		}

		if len(item.LabelsExcludeAny) > 0 && len(item.LabelsIncludeAny) > 0 {
			multiError = multierror.Append(multiError, fmt.Errorf(`only one of "labels_exclude_any" or "labels_include_any" can be specified for app store app %q`, item.AppStoreID))
			continue
		}

		result.Software.AppStoreApps = append(result.Software.AppStoreApps, &item)
	}
	for _, item := range software.FleetMaintainedApps {
		item := item
		if item.Slug == "" {
			multiError = multierror.Append(multiError, errors.New("fleet maintained app slug is required"))
			continue
		}

		if len(item.LabelsExcludeAny) > 0 && len(item.LabelsIncludeAny) > 0 {
			multiError = multierror.Append(multiError, fmt.Errorf(`only one of "labels_exclude_any" or "labels_include_any" can be specified for fleet maintained app %q`, item.Slug))
			continue
		}

		item = item.ResolveSoftwarePackagePaths(baseDir)

		// handle secrets
		if item.InstallScript.Path != "" {
			if err := gatherFileSecrets(result, item.InstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}
		if item.PostInstallScript.Path != "" {
			if err := gatherFileSecrets(result, item.PostInstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}
		if item.UninstallScript.Path != "" {
			if err := gatherFileSecrets(result, item.UninstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}

		result.Software.FleetMaintainedApps = append(result.Software.FleetMaintainedApps, &item)
	}
	for _, item := range software.Packages {
		var softwarePackageSpec fleet.SoftwarePackageSpec
		if item.Path != nil {
			softwarePackageSpec.ReferencedYamlPath = resolveApplyRelativePath(baseDir, *item.Path)
			fileBytes, err := os.ReadFile(softwarePackageSpec.ReferencedYamlPath)
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to read software package file %s: %v", *item.Path, err))
				continue
			}
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to expand environmet in file %s: %v", *item.Path, err))
				continue
			}
			if err := YamlUnmarshal(fileBytes, &softwarePackageSpec); err != nil {
				multiError = multierror.Append(multiError, MaybeParseTypeError(*item.Path, []string{"software", "packages"}, err))
				continue
			}

			softwarePackageSpec = softwarePackageSpec.ResolveSoftwarePackagePaths(filepath.Dir(softwarePackageSpec.ReferencedYamlPath))
		} else {
			softwarePackageSpec = item.SoftwarePackageSpec.ResolveSoftwarePackagePaths(baseDir)
		}
		if softwarePackageSpec.InstallScript.Path != "" {
			if err := gatherFileSecrets(result, softwarePackageSpec.InstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}
		if softwarePackageSpec.PostInstallScript.Path != "" {
			if err := gatherFileSecrets(result, softwarePackageSpec.PostInstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}
		if softwarePackageSpec.SHA256 != "" && !validSHA256Value.MatchString(softwarePackageSpec.SHA256) {
			multiError = multierror.Append(multiError, fmt.Errorf("hash_256 value %q must be a valid lower-case hex-encoded (64-character) SHA-256 hash value", softwarePackageSpec.SHA256))
			continue
		}
		if softwarePackageSpec.SHA256 == "" && softwarePackageSpec.URL == "" {
			multiError = multierror.Append(multiError, errors.New("at least one of hash_sha256 or url is required for each software package"))
			continue
		}
		if softwarePackageSpec.UninstallScript.Path != "" {
			if err := gatherFileSecrets(result, softwarePackageSpec.UninstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}
		if len(softwarePackageSpec.LabelsExcludeAny) > 0 && len(softwarePackageSpec.LabelsIncludeAny) > 0 {
			multiError = multierror.Append(multiError, fmt.Errorf(`only one of "labels_exclude_any" or "labels_include_any" can be specified for software URL %q`, softwarePackageSpec.URL))
			continue
		}
		if len(softwarePackageSpec.URL) > fleet.SoftwareInstallerURLMaxLength {
			multiError = multierror.Append(multiError, fmt.Errorf("software URL %q is too long, must be %d characters or less", softwarePackageSpec.URL, fleet.SoftwareInstallerURLMaxLength))
			continue
		}
		parsedUrl, err := url.Parse(softwarePackageSpec.URL)
		if err != nil {
			multiError = multierror.Append(multiError, fmt.Errorf("software URL %s is not a valid URL", softwarePackageSpec.URL))
			continue
		}

		if softwarePackageSpec.InstallScript.Path == "" || softwarePackageSpec.UninstallScript.Path == "" {
			// URL checks won't catch everything, but might as well include a lightweight check here to fail fast if it's
			// certain that the package will fail later.
			if strings.HasSuffix(parsedUrl.Path, ".exe") {
				multiError = multierror.Append(multiError, fmt.Errorf("software URL %s refers to an .exe package, which requires both install_script and uninstall_script", softwarePackageSpec.URL))
				continue
			}
			if strings.HasSuffix(parsedUrl.Path, ".tar.gz") || strings.HasSuffix(parsedUrl.Path, ".tgz") {
				multiError = multierror.Append(multiError, fmt.Errorf("software URL %s refers to a .tar.gz archive, which requires both install_script and uninstall_script", softwarePackageSpec.URL))
				continue
			}
		}

		result.Software.Packages = append(result.Software.Packages, &softwarePackageSpec)
	}

	return multiError
}

func gatherFileSecrets(result *GitOps, filePath string) error {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	err = LookupEnvSecrets(string(fileBytes), result.FleetSecrets)
	if err != nil {
		return fmt.Errorf("failed to lookup environment secrets for %s: %w", filePath, err)
	}

	return nil
}

func getDuplicateNames[T any](slice []T, getComparableString func(T) string) []string {
	// We are using the allKeys map as a set here. True means the item is a duplicate.
	allKeys := make(map[string]bool)
	var duplicates []string
	for _, item := range slice {
		name := getComparableString(item)
		if isDuplicate, exists := allKeys[name]; exists {
			// If this name hasn't already been marked as a duplicate.
			if !isDuplicate {
				duplicates = append(duplicates, name)
			}
			allKeys[name] = true
		} else {
			allKeys[name] = false
		}
	}
	return duplicates
}

func isASCII(s string) bool {
	for _, c := range s {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// resolves the paths to an absolute path relative to the baseDir, which should
// be the path of the YAML file where the relative paths were specified. If the
// path is already absolute, it is left untouched.
func resolveApplyRelativePath(baseDir string, path string) string {
	if baseDir == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}
