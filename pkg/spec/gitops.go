package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/text/unicode/norm"
)

const LabelAPIGlobalTeamName = "global"

// LabelChangesSummary carries extra context of the labels operations for a config.
type LabelChangesSummary struct {
	LabelsToUpdate  []string
	LabelsToAdd     []string
	LabelsToRemove  []string
	LabelsMovements []LabelMovement
}

// HasChanges returns true if there are any label additions, removals, updates, or movements.
func (s LabelChangesSummary) HasChanges() bool {
	return len(s.LabelsToAdd) > 0 || len(s.LabelsToRemove) > 0 || len(s.LabelsToUpdate) > 0 || len(s.LabelsMovements) > 0
}

func NewLabelChangesSummary(changes []LabelChange, moves []LabelMovement) LabelChangesSummary {
	r := LabelChangesSummary{
		LabelsMovements: moves,
	}

	lookUp := make(map[string]any)
	for _, m := range moves {
		lookUp[m.Name] = nil
	}

	for _, change := range changes {
		if _, ok := lookUp[change.Name]; ok {
			continue
		}
		switch change.Op {
		case "+":

			r.LabelsToAdd = append(r.LabelsToAdd, change.Name)
		case "-":
			r.LabelsToRemove = append(r.LabelsToRemove, change.Name)
		case "=":
			r.LabelsToUpdate = append(r.LabelsToUpdate, change.Name)
		}
	}
	return r
}

// LabelMovement specifies a label movement, a label is moved if its removed from one team and added to another
type LabelMovement struct {
	FromTeamName string // Source team name
	ToTeamName   string // Dest. team name
	Name         string // The globally unique label name
}

// LabelChange used for keeping track of label operations
type LabelChange struct {
	Name     string // The globally unique label name
	Op       string // What operation to perform on the label. +:add, -:remove, =:no-op
	TeamName string // The team this label belongs to.
	FileName string // The filename that contains the label change
}

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

// ParseUnknownKeyError represents an unknown/misspelled key found in a GitOps YAML file.
type ParseUnknownKeyError struct {
	Filename   string
	Path       string // dot-separated path context, e.g. "controls.macos_settings"
	Field      string // the unknown field name
	Suggestion string // suggested correct key, if a close match exists
}

func (e *ParseUnknownKeyError) Error() string {
	key := e.Field
	if e.Path != "" {
		key = e.Path + "." + e.Field
	}
	var suffix string
	if e.Suggestion != "" {
		suffix = fmt.Sprintf("; did you mean %q?", e.Suggestion)
	}
	return fmt.Sprintf("unknown key %q in %q%s", key, e.Filename, suffix)
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

// If you add a new key to this struct, ensure the Set() method below also checks for it
type GitOpsControls struct {
	fleet.BaseItem
	MacOSUpdates   any               `json:"macos_updates"`
	IOSUpdates     any               `json:"ios_updates"`
	IPadOSUpdates  any               `json:"ipados_updates"`
	MacOSSettings  any               `json:"macos_settings" renameto:"apple_settings"`
	MacOSSetup     *fleet.MacOSSetup `json:"macos_setup" renameto:"setup_experience"`
	MacOSMigration any               `json:"macos_migration"`

	WindowsUpdates                 any `json:"windows_updates"`
	WindowsSettings                any `json:"windows_settings"`
	WindowsEnabledAndConfigured    any `json:"windows_enabled_and_configured"`
	WindowsMigrationEnabled        any `json:"windows_migration_enabled"`
	EnableTurnOnWindowsMDMManually any `json:"enable_turn_on_windows_mdm_manually"`
	WindowsEntraTenantIDs          any `json:"windows_entra_tenant_ids"`

	AndroidEnabledAndConfigured any `json:"android_enabled_and_configured"`
	AndroidSettings             any `json:"android_settings"`

	AppleRequireHardwareAttestation any `json:"apple_require_hardware_attestation"`

	EnableDiskEncryption       any              `json:"enable_disk_encryption"`
	EnableRecoveryLockPassword any              `json:"enable_recovery_lock_password"`
	RequireBitLockerPIN        any              `json:"windows_require_bitlocker_pin,omitempty"`
	Scripts                    []fleet.BaseItem `json:"scripts"`

	Defined bool
}

func (c GitOpsControls) Set() bool {
	return c.MacOSUpdates != nil || c.IOSUpdates != nil ||
		c.IPadOSUpdates != nil || c.MacOSSettings != nil ||
		c.MacOSSetup != nil || c.MacOSMigration != nil ||
		c.WindowsUpdates != nil || c.WindowsSettings != nil || c.WindowsEnabledAndConfigured != nil ||
		c.WindowsMigrationEnabled != nil || c.EnableDiskEncryption != nil || c.EnableRecoveryLockPassword != nil ||
		len(c.Scripts) > 0 || c.AndroidEnabledAndConfigured != nil || c.AndroidSettings != nil ||
		c.AppleRequireHardwareAttestation != nil || c.EnableTurnOnWindowsMDMManually != nil ||
		c.WindowsEntraTenantIDs != nil || c.RequireBitLockerPIN != nil
}

type Policy struct {
	fleet.BaseItem
	GitOpsPolicySpec
}

type GitOpsPolicySpec struct {
	fleet.PolicySpec
	RunScript       *PolicyRunScript                       `json:"run_script"`
	InstallSoftware optjson.BoolOr[*PolicyInstallSoftware] `json:"install_software"`
	// InstallSoftwareURL is populated after parsing the software installer yaml
	// referenced by InstallSoftware.PackagePath.
	InstallSoftwareURL string `json:"-"`
	// RunScriptName is populated after confirming the script exists on both the file system
	// and in the controls scripts list for the same team
	RunScriptName *string `json:"-"`
	// WebhooksAndTicketsEnabled indicates whether failing policy webhooks/tickets
	// should be enabled for this policy. This is a gitops-only convenience that
	// translates to adding the policy's ID to the failing_policies_webhook.policy_ids list.
	WebhooksAndTicketsEnabled bool `json:"webhooks_and_tickets_enabled"`
}

type PolicyRunScript struct {
	Path string `json:"path"`
}

type PolicyInstallSoftware struct {
	PackagePath            string `json:"package_path"`
	AppStoreID             string `json:"app_store_id"`
	HashSHA256             string `json:"hash_sha256"`
	FleetMaintainedAppSlug string `json:"fleet_maintained_app_slug"`
}

type Query struct {
	fleet.BaseItem
	fleet.QuerySpec
}

type Label struct {
	fleet.BaseItem
	fleet.LabelSpec
}

// UnmarshalJSON distinguishes between "hosts" key omitted (nil, preserve
// existing membership) and "hosts" key present with null value (clear all
// hosts). Both cases produce a nil HostsSlice after default unmarshaling,
// so we check the raw JSON for the key's presence.
func (l *Label) UnmarshalJSON(data []byte) error {
	// Use an alias to prevent infinite recursion.
	type LabelAlias Label
	var alias LabelAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*l = Label(alias)

	if l.Hosts == nil {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err == nil {
			if _, ok := raw["hosts"]; ok {
				// "hosts" key was explicitly set to null — clear all hosts.
				l.Hosts = []string{}
			}
		}
	}
	return nil
}

type SoftwarePackage struct {
	fleet.BaseItem
	fleet.SoftwarePackageSpec
}

func (spec SoftwarePackage) HydrateToPackageLevel(packageLevel fleet.SoftwarePackageSpec, ext string) (fleet.SoftwarePackageSpec, error) {
	if spec.InstallScript.Path != "" || spec.UninstallScript.Path != "" ||
		spec.PostInstallScript.Path != "" || spec.URL != "" || spec.SHA256 != "" || spec.PreInstallQuery.Path != "" {
		return packageLevel, fmt.Errorf("the software package defined in %s must not have icons, scripts, queries, URL, or hash specified at the team level", *spec.Path)
	}

	// Icon should be allowed at the team level yaml for script packages which must be specified as a path
	if spec.Icon.Path != "" {
		if ext != ".sh" && ext != ".ps1" {
			return packageLevel, fmt.Errorf("the software package defined in %s must not have icons, scripts, queries, URL, or hash specified at the team level", *spec.Path)
		}
	}

	packageLevel.Categories = spec.Categories
	packageLevel.LabelsIncludeAny = spec.LabelsIncludeAny
	packageLevel.LabelsExcludeAny = spec.LabelsExcludeAny
	packageLevel.LabelsIncludeAll = spec.LabelsIncludeAll
	packageLevel.InstallDuringSetup = spec.InstallDuringSetup
	packageLevel.SelfService = spec.SelfService

	// This will only override display name set at path: path/to/software.yml level
	// if display_name is specified at the team level yml
	if spec.DisplayName != "" {
		packageLevel.DisplayName = spec.DisplayName
	}

	return packageLevel, nil
}

type Software struct {
	Packages            []SoftwarePackage           `json:"packages"`
	AppStoreApps        []fleet.TeamSpecAppStoreApp `json:"app_store_apps"`
	FleetMaintainedApps []fleet.MaintainedAppSpec   `json:"fleet_maintained_apps"`
}

// GitOpsMDM extends fleet.MDM with gitops-only fields that are not part of the server type.
type GitOpsMDM struct {
	fleet.MDM
	EndUserLicenseAgreement any `json:"end_user_license_agreement,omitempty"`
}

// GitOpsOrgSettings defines the valid keys for the top-level `org_settings:` section.
// It embeds fleet.AppConfig for all standard settings and adds gitops-only keys
// that are extracted before the config is sent to the server API.
type GitOpsOrgSettings struct {
	fleet.AppConfig
	Secrets                any `json:"secrets"`
	CertificateAuthorities any `json:"certificate_authorities"`
}

// GitOpsOrgInfo extends fleet.OrgInfo with gitops-only path keys for uploading
// a custom org logo from a local file. The path keys are extracted from the
// OrgInfo before it's sent to the AppConfig PATCH endpoint, and the actual
// PUT /api/v1/fleet/logo upload runs after the PATCH succeeds (see
// Client.DoGitOps in server/service/client.go) so a PATCH failure leaves
// logo storage untouched.
type GitOpsOrgInfo struct {
	fleet.OrgInfo
	OrgLogoPathDarkMode  string `json:"org_logo_path_dark_mode,omitempty"`
	OrgLogoPathLightMode string `json:"org_logo_path_light_mode,omitempty"`
}

// GitOpsFleetSettings defines the valid keys for the top-level `settings:` section (fleet-level).
// It embeds fleet.TeamConfig for all standard settings and adds gitops-only keys
// that are extracted before the config is sent to the server API.
type GitOpsFleetSettings struct {
	fleet.TeamConfig
	Secrets any `json:"secrets"`
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

	Labels              []*fleet.LabelSpec
	LabelChangesSummary LabelChangesSummary

	// Software is only allowed on teams, not on global config.
	Software GitOpsSoftware
	// FleetSecrets is a map of secret names to their values, extracted from FLEET_SECRET_ environment variables used in profiles and scripts.
	FleetSecrets map[string]string

	// LabelsPresent indicates that the `labels:` key was explicitly present in the YAML file.
	LabelsPresent bool
	// SoftwarePresent indicates that the `software:` key was explicitly present in the YAML file.
	SoftwarePresent bool
	// SecretsPresent indicates that the `secrets:` key was explicitly present in the YAML file.
	SecretsPresent bool
}

type GitOpsSoftware struct {
	Packages            []*fleet.SoftwarePackageSpec
	AppStoreApps        []*fleet.TeamSpecAppStoreApp
	FleetMaintainedApps []*fleet.MaintainedAppSpec
}

type Logf func(format string, a ...interface{})

// GitOpsOptions configures optional behavior for GitOps file parsing.
type GitOpsOptions struct {
	// AllowUnknownKeys causes unknown key errors to be logged as warnings
	// instead of returned as errors.
	AllowUnknownKeys bool
	// SyntheticSoftwareByTeam maps team names to JSON-encoded software specs
	// from the server. When the software: key is excepted from GitOps and
	// omitted from the YAML, this data is injected so that parseSoftware can
	// populate result.Software for policy validation (install_software and
	// patch policy references). SoftwarePresent remains false so that
	// downstream exception enforcement still works correctly.
	SyntheticSoftwareByTeam map[string]json.RawMessage
}

// GitOpsFromFile parses a GitOps yaml file.
func GitOpsFromFile(filePath, baseDir string, appConfig *fleet.EnrichedAppConfig, logFn Logf, opts ...GitOpsOptions) (*GitOps, error) {
	var options GitOpsOptions
	if len(opts) > 1 {
		panic("too many options provided to GitOpsFromFile")
	} else if len(opts) == 1 {
		options = opts[0]
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s: %w", filePath, err)
	}

	// Replace $var and ${var} with env values.
	b, err = ExpandEnvBytes(b)
	if err != nil {
		return nil, fmt.Errorf("failed to expand environment in file %s: %w", filePath, err)
	}

	// First unmarshal to map[string]any for deprecation handling
	var rawData map[string]any
	if err := yaml.Unmarshal(b, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %w: \n", err)
	}

	// Apply deprecated key mappings (e.g., team_settings -> settings, queries -> reports)
	if err := ApplyDeprecatedKeyMappings(rawData, logFn); err != nil {
		return nil, fmt.Errorf("failed to process deprecated keys in file %s: %w", filePath, err)
	}

	// Re-marshal and unmarshal to map[string]json.RawMessage for existing parsing logic
	updatedBytes, err := json.Marshal(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to re-marshal file %s: %w", filePath, err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(updatedBytes, &top); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s: %w", filePath, err)
	}
	// This should never happen since we don't support empty yaml files,
	// but adding for defensive purposes.
	if top == nil {
		top = make(map[string]json.RawMessage)
	}

	var multiError *multierror.Error
	result := &GitOps{}
	result.FleetSecrets = make(map[string]string)

	topKeys := []string{"name", "settings", "org_settings", "agent_options", "controls", "policies", "reports", "software", "labels"}
	for k := range top {
		if !slices.Contains(topKeys, k) {
			multiError = multierror.Append(multiError, fmt.Errorf("unknown top-level field: %s", k))
		}
	}

	// Figure out if this is an org or fleet settings file
	teamRaw, teamOk := top["name"]
	settingsRaw, settingsOk := top["settings"]
	orgSettingsRaw, orgOk := top["org_settings"]
	switch {
	case orgOk:
		if teamOk || settingsOk {
			multiError = multierror.Append(multiError, errors.New("'org_settings' cannot be used with 'name', 'settings'"))
		} else {
			multiError = parseOrgSettings(orgSettingsRaw, result, baseDir, filePath, multiError)
		}
	case teamOk:
		multiError = parseName(teamRaw, result, filePath, multiError)
		// If the file is no-team.yml, the name must be "No team".
		switch {
		case filepath.Base(filePath) == "no-team.yml" && !result.IsNoTeam():
			multiError = multierror.Append(multiError, errors.New("`name` must be `No Team` for `no-team.yml`"))
			return result, multiError.ErrorOrNil()
		case filepath.Base(filePath) == "unassigned.yml" && !result.IsUnassignedTeam():
			multiError = multierror.Append(multiError, errors.New("`name` must be `Unassigned` for `unassigned.yml`"))
			return result, multiError.ErrorOrNil()
		case result.IsNoTeam() && filepath.Base(filePath) != "no-team.yml":
			multiError = multierror.Append(multiError, fmt.Errorf("file `%s` for No Team must be named `no-team.yml`", filePath))
			multiError = multierror.Append(multiError, errors.New("no-team.yml is deprecated; please rename the file to 'unassigned.yml' and update the team name to 'Unassigned'."))
			return result, multiError.ErrorOrNil()
		case result.IsUnassignedTeam() && filepath.Base(filePath) != "unassigned.yml":
			multiError = multierror.Append(multiError, fmt.Errorf("file `%s` for unassigned hosts must be named `unassigned.yml`", filePath))
			return result, multiError.ErrorOrNil()
		case result.IsNoTeam() || result.IsUnassignedTeam():
			// Coerce to "No Team" for easier processing.
			// TODO - Remove No Team in Fleet 5
			result.TeamName = ptr.String(noTeam)
			// For No Team, we allow settings but only process webhook_settings from it
			if settingsOk {
				multiError = parseNoTeamSettings(settingsRaw, result, filePath, multiError)
			}
		default:
			// Allow omitting settings key for teams, clearing all team settings as a result.
			if !settingsOk {
				settingsRaw = json.RawMessage("null")
			}
			multiError = parseTeamSettings(settingsRaw, result, baseDir, filePath, multiError)
		}
	default:
		switch filepath.Base(filePath) {
		case "no-team.yml":
			multiError = multierror.Append(multiError, errors.New("`name` must be `No Team` for `no-team.yml`"))
		case "unassigned.yml":
			multiError = multierror.Append(multiError, errors.New("`name` must be `Unassigned` for `unassigned.yml`"))
		default:
			multiError = multierror.Append(multiError, fmt.Errorf("No `name` was provided in %s. If this file is intended to define org-level settings, add `org_settings:` as a top-level key. Otherwise, use `name` to specify the fleet name.", filePath))
		}
	}

	for _, topKey := range topKeys {
		// "name" is handled later with special logic based on the filename.
		// "labels" and "software" are special cases where omitting may be a no-op (based on exception settings),
		// rather than a directive to clear settings. settings keys were handled above.
		if topKey == "name" || topKey == "labels" || topKey == "software" || topKey == "settings" || topKey == "org_settings" {
			continue
		}
		// "controls" can be set on _either_ global or "no team" file, and we can't say which it is if both
		// files aren't supplied, so play it safe and require it to be set on one or the other.
		if (result.IsNoTeam() || result.IsGlobal()) && topKey == "controls" {
			continue
		}
		// "agent_options" and "reports" are not supported in no-team/unassigned files.
		if result.IsNoTeam() && (topKey == "agent_options" || topKey == "reports") {
			continue
		}
		// Default top keys to null if not present.
		// This will clear the settings as if the key was provided with an empty value.
		if _, ok := top[topKey]; !ok {
			top[topKey] = json.RawMessage("null")
		}
	}

	// Get the labels. LabelsPresent tracks whether the key was in the YAML.
	if _, ok := top["labels"]; ok {
		result.LabelsPresent = true
		if result.IsNoTeam() {
			logFn("[!] 'labels' is not supported in %s. This key will be ignored.\n", filepath.Base(filePath))
		} else {
			multiError = parseLabels(top, result, baseDir, logFn, filePath, multiError)
		}
	}
	// Get other top-level entities.
	multiError = parseControls(top, result, logFn, filePath, multiError)
	multiError = parseAgentOptions(top, result, baseDir, logFn, filePath, multiError)
	multiError = parseReports(top, result, baseDir, logFn, filePath, multiError)

	if appConfig != nil && appConfig.License.IsPremium() {
		multiError = parseSoftware(top, result, baseDir, filePath, options, multiError)
	}

	// Policies can reference software installers and scripts, thus we parse them after parseSoftware and parseControls.
	multiError = parsePolicies(top, result, baseDir, logFn, filePath, multiError)

	// If AllowUnknownKeys is set, filter out ParseUnknownKeyError and log them as warnings.
	if options.AllowUnknownKeys {
		return result, filterWarnings(multiError, logFn, reflect.TypeFor[*ParseUnknownKeyError]())
	}

	return result, multiError.ErrorOrNil()
}

func parseName(raw json.RawMessage, result *GitOps, filePath string, multiError *multierror.Error) *multierror.Error {
	if err := json.Unmarshal(raw, &result.TeamName); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"name"}, err))
	}
	if result.TeamName != nil {
		*result.TeamName = strings.TrimSpace(*result.TeamName)
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
	return g.TeamName != nil && strings.EqualFold(*g.TeamName, noTeam)
}

func (g *GitOps) IsUnassignedTeam() bool {
	return g.TeamName != nil && strings.EqualFold(*g.TeamName, unassignedTeamName)
}

func (g *GitOps) CoercedTeamName() string {
	if g.global() {
		return LabelAPIGlobalTeamName
	}
	return *g.TeamName
}

const (
	noTeam             = "No team"
	unassignedTeamName = "Unassigned"
)

func parseOrgSettings(raw json.RawMessage, result *GitOps, baseDir string, filePath string, multiError *multierror.Error) *multierror.Error {
	var orgSettingsTop fleet.BaseItem
	if err := json.Unmarshal(raw, &orgSettingsTop); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"org_settings"}, err))
	}
	noError := true
	settingsFilePath := filePath
	if orgSettingsTop.Path != nil {
		settingsFilePath = *orgSettingsTop.Path
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
				var pathOrgSettings fleet.BaseItem
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
			multiError = validateOrgInfoLogo(result.OrgSettings, multiError)
		}
		// Validate unknown keys in org_settings section.
		multiError = multierror.Append(multiError, validateYAMLKeys(raw, reflect.TypeFor[GitOpsOrgSettings](), settingsFilePath, []string{"org_settings"})...)
		// TODO: Validate that integrations.(jira|zendesk)[].api_token is not empty or fleet.MaskedPassword
	}
	return multiError
}

// validateOrgInfoLogo rejects org_info configurations that specify both a path
// and a URL for the same mode. Deprecated URL keys are already migrated to the
// new mode-aware names by ApplyDeprecatedKeyMappings before this runs.
func validateOrgInfoLogo(orgSettings map[string]any, multiError *multierror.Error) *multierror.Error {
	orgInfo, _ := orgSettings["org_info"].(map[string]any)
	if orgInfo == nil {
		return multiError
	}
	check := func(mode, pathKey, urlKey string) {
		path, _ := orgInfo[pathKey].(string)
		urlVal, _ := orgInfo[urlKey].(string)
		if path != "" && urlVal != "" {
			multiError = multierror.Append(multiError, fmt.Errorf(
				"org_settings.org_info: cannot specify both '%s' and '%s' for %s mode; choose one",
				pathKey, urlKey, mode,
			))
		}
	}
	check("dark", "org_logo_path_dark_mode", "org_logo_url_dark_mode")
	check("light", "org_logo_path_light_mode", "org_logo_url_light_mode")
	return multiError
}

func parseTeamSettings(raw json.RawMessage, result *GitOps, baseDir string, filePath string, multiError *multierror.Error) *multierror.Error {
	var teamSettingsTop fleet.BaseItem
	if err := json.Unmarshal(raw, &teamSettingsTop); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"settings"}, err))
	}
	noError := true
	settingsFilePath := filePath
	if teamSettingsTop.Path != nil {
		settingsFilePath = *teamSettingsTop.Path
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
				var pathTeamSettings fleet.BaseItem
				if err := YamlUnmarshal(fileBytes, &pathTeamSettings); err != nil {
					noError = false
					multiError = multierror.Append(
						multiError, MaybeParseTypeError(*teamSettingsTop.Path, []string{"settings"}, err),
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
			multiError = multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"settings"}, err))
		} else {
			multiError = parseSecrets(result, multiError)
			// Validate webhook settings for regular teams
			multiError = validateTeamWebhookSettings(result.TeamSettings, multiError)
		}
		// Validate unknown keys in team settings section.
		multiError = multierror.Append(multiError, validateYAMLKeys(raw, reflect.TypeFor[GitOpsFleetSettings](), settingsFilePath, []string{"settings"})...)
	}
	return multiError
}

// validateTeamWebhookSettings validates webhook settings for regular teams
func validateTeamWebhookSettings(teamSettings map[string]any, multiError *multierror.Error) *multierror.Error {
	if webhookSettings, hasWebhook := teamSettings["webhook_settings"]; hasWebhook && webhookSettings != nil {
		webhookMap, ok := webhookSettings.(map[string]any)
		if !ok {
			return multierror.Append(multiError, errors.New("'settings.webhook_settings' must be an object or null"))
		}

		// Validate failing_policies_webhook if present
		if fpw, hasFPW := webhookMap["failing_policies_webhook"]; hasFPW && fpw != nil {
			fpwMap, ok := fpw.(map[string]any)
			if !ok {
				multiError = multierror.Append(multiError, errors.New("'settings.webhook_settings.failing_policies_webhook' must be an object or null"))
			} else {
				// Validate failing_policies_webhook structure
				if err := validateFailingPoliciesWebhook(fpwMap, "settings.webhook_settings.failing_policies_webhook"); err != nil {
					multiError = multierror.Append(multiError, err)
				}
			}
		}

		// Could add validation for other webhook types here in the future
		// e.g., host_status_webhook, vulnerabilities_webhook, etc.
	}
	return multiError
}

// validateFailingPoliciesWebhook validates the failing_policies_webhook configuration.
// It ensures policy_ids is an array if present.
func validateFailingPoliciesWebhook(fpwMap map[string]any, keyPath string) error {
	// Validate policy_ids is an array if present
	if policyIDs, hasPolicyIDs := fpwMap["policy_ids"]; hasPolicyIDs && policyIDs != nil {
		// Check if it's an array
		_, isArray := policyIDs.([]any)
		if !isArray {
			return fmt.Errorf("'%s.policy_ids' must be an array, got %T", keyPath, policyIDs)
		}
	}
	return nil
}

// parseNoTeamSettings parses settings for "No Team" files, but only processes webhook_settings
func parseNoTeamSettings(raw json.RawMessage, result *GitOps, filePath string, multiError *multierror.Error) *multierror.Error {
	// Parse the raw JSON into a map to extract only webhook_settings
	var teamSettingsMap map[string]interface{}
	if err := json.Unmarshal(raw, &teamSettingsMap); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"settings"}, err))
	}

	// For No Team, only webhook_settings is allowed in settings
	// Jira/Zendesk integrations are not supported in gitops: https://github.com/fleetdm/fleet/issues/20287
	// Check for any other keys and error if found
	for key := range teamSettingsMap {
		if key != "webhook_settings" {
			multiError = multierror.Append(multiError,
				fmt.Errorf("unsupported settings option '%s' in %s - only 'webhook_settings' is allowed", key, filepath.Base(filePath)))
		}
	}

	// Initialize TeamSettings if nil
	if result.TeamSettings == nil {
		result.TeamSettings = make(map[string]interface{})
	}

	// For No Team, we only care about webhook_settings
	if webhookRaw, ok := teamSettingsMap["webhook_settings"]; ok {
		// Handle null webhook_settings (which means clear webhook settings)
		if webhookRaw == nil {
			// Store as nil to indicate webhook settings should be cleared
			result.TeamSettings["webhook_settings"] = nil
		} else {
			webhookMap, ok := webhookRaw.(map[string]any)
			if !ok {
				return multierror.Append(multiError, errors.New("'settings.webhook_settings' must be an object or null"))
			}
			for key := range webhookMap {
				if key != "failing_policies_webhook" {
					multiError = multierror.Append(multiError,
						fmt.Errorf("unsupported webhook_settings option '%s' in %s - only 'failing_policies_webhook' is allowed", key, filepath.Base(filePath)))
				}
			}
			// If present, ensure failing_policies_webhook is an object or null
			if fpw, ok := webhookMap["failing_policies_webhook"]; ok && fpw != nil {
				fpwMap, ok := fpw.(map[string]any)
				if !ok {
					multiError = multierror.Append(multiError, errors.New("'settings.webhook_settings.failing_policies_webhook' must be an object or null"))
				} else {
					// Validate failing_policies_webhook structure
					if err := validateFailingPoliciesWebhook(fpwMap, "settings.webhook_settings.failing_policies_webhook"); err != nil {
						multiError = multierror.Append(multiError, err)
					}
				}
			}
			// Store the webhook settings for later processing
			result.TeamSettings["webhook_settings"] = webhookMap
		}
	}

	return multiError
}

func parseSecrets(result *GitOps, multiError *multierror.Error) *multierror.Error {
	var rawSecrets interface{}
	var ok bool
	if result.TeamName == nil {
		rawSecrets, ok = result.OrgSettings["secrets"]
	} else {
		rawSecrets, ok = result.TeamSettings["secrets"]
	}
	if !ok {
		// Allow omitting secrets key, resulting in a no-op for secrets.
		// Any secrets present on the server will be retained.
		return multiError
	}
	result.SecretsPresent = true
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
			logFn("[!] 'agent_options' is not supported in %s. This key will be ignored.\n", filepath.Base(filePath))
		}
		return multiError
	} else if !ok {
		return multierror.Append(multiError, errors.New("'agent_options' is required"))
	}
	var agentOptionsTop fleet.BaseItem
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
				var pathAgentOptions fleet.BaseItem
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

func parseControls(top map[string]json.RawMessage, result *GitOps, logFn Logf, yamlFilename string, multiError *multierror.Error) *multierror.Error {
	controlsRaw, ok := top["controls"]
	if !ok {
		// Nothing to do, return.
		return multiError
	}

	controlsRaw, _, err := rewriteNewToOldKeys(controlsRaw, &GitOpsControls{})
	if err != nil {
		return multierror.Append(multiError, fmt.Errorf("failed to rewrite controls keys: %v", err))
	}

	var controlsTop GitOpsControls
	if err := json.Unmarshal(controlsRaw, &controlsTop); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(yamlFilename, []string{"controls"}, err))
	}
	// Validate unknown keys in controls section.
	multiError = multierror.Append(multiError, validateRawKeys(controlsRaw, reflect.TypeFor[GitOpsControls](), yamlFilename, []string{"controls"})...)
	controlsTop.Defined = true
	controlsFilePath := yamlFilename
	multiError = multierror.Append(multiError, processControlsPathIfNeeded(controlsTop, result, &controlsFilePath)...)

	controlsDir := filepath.Dir(controlsFilePath)
	var scriptErrs []error
	result.Controls.Scripts, scriptErrs = resolveScriptPaths(result.Controls.Scripts, controlsDir, logFn)
	for _, err := range scriptErrs {
		multiError = multierror.Append(multiError, fmt.Errorf("failed to parse scripts list in %s: %v", controlsFilePath, err))
	}

	// Find Fleet secrets in scripts.
	for _, script := range result.Controls.Scripts {
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
		data, _, err = rewriteNewToOldKeys(data, &macOSSettings)
		if err != nil {
			return multierror.Append(multiError, fmt.Errorf("failed to rewrite macos_settings keys: %v", err))
		}
		err = json.Unmarshal(data, &macOSSettings)
		if err != nil {
			return multierror.Append(multiError, MaybeParseTypeError(controlsFilePath, []string{"controls", "macos_settings"}, err))
		}

		// Expand globs in profile paths.
		var errs []error
		macOSSettings.CustomSettings, errs = expandBaseItems(macOSSettings.CustomSettings, controlsDir, "profile", GlobExpandOptions{
			AllowedExtensions: map[string]bool{".mobileconfig": true, ".json": true},
			LogFn:             logFn,
		})
		multiError = multierror.Append(multiError, errs...)
		// Then resolve the paths to absolute and find Fleet secrets in the profile files.
		for i := range macOSSettings.CustomSettings {

			err := resolveAndUpdateProfilePath(&macOSSettings.CustomSettings[i], result)
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
		data, _, err = rewriteNewToOldKeys(data, &windowsSettings)
		if err != nil {
			return multierror.Append(multiError, fmt.Errorf("failed to rewrite windows_settings keys: %v", err))
		}
		err = json.Unmarshal(data, &windowsSettings)
		if err != nil {
			return multierror.Append(multiError, MaybeParseTypeError(controlsFilePath, []string{"controls", "windows_settings"}, err))
		}
		if windowsSettings.CustomSettings.Valid {
			var errs []error
			windowsSettings.CustomSettings.Value, errs = expandBaseItems(windowsSettings.CustomSettings.Value, controlsDir, "profile", GlobExpandOptions{
				AllowedExtensions: map[string]bool{".xml": true},

				LogFn: logFn,
			})
			multiError = multierror.Append(multiError, errs...)

			for i := range windowsSettings.CustomSettings.Value {
				err := resolveAndUpdateProfilePath(&windowsSettings.CustomSettings.Value[i], result)
				if err != nil {
					return multierror.Append(multiError, err)
				}
			}
		}
		// Since we already unmarshalled and updated the path, we need to update the result struct.
		result.Controls.WindowsSettings = windowsSettings
	}

	if result.Controls.AndroidSettings != nil {
		// We are marshalling/unmarshalling to get the data into the fleet.AndroidSettings struct.
		// This is inefficient, but it is more robust and less error-prone.
		var androidSettings fleet.AndroidSettings
		data, err := json.Marshal(result.Controls.AndroidSettings)
		if err != nil {
			return multierror.Append(multiError, fmt.Errorf("failed to process controls.android_settings: %v", err))
		}
		data, _, err = rewriteNewToOldKeys(data, &androidSettings)
		if err != nil {
			return multierror.Append(multiError, fmt.Errorf("failed to rewrite android_settings keys: %v", err))
		}
		err = json.Unmarshal(data, &androidSettings)
		if err != nil {
			return multierror.Append(multiError, MaybeParseTypeError(controlsFilePath, []string{"controls", "android_settings"}, err))
		}

		if androidSettings.CustomSettings.Valid {
			var errs []error
			androidSettings.CustomSettings.Value, errs = expandBaseItems(androidSettings.CustomSettings.Value, controlsDir, "profile", GlobExpandOptions{
				AllowedExtensions: map[string]bool{".json": true},
				LogFn:             logFn,
			})
			multiError = multierror.Append(multiError, errs...)
			for i := range androidSettings.CustomSettings.Value {
				err := resolveAndUpdateProfilePath(&androidSettings.CustomSettings.Value[i], result)
				if err != nil {
					return multierror.Append(multiError, err)
				}
			}
		}

		if androidSettings.Certificates.Valid {
			for i, cert := range androidSettings.Certificates.Value {
				if cert.Name == "" {
					multiError = multierror.Append(multiError, fmt.Errorf("android_settings.certificates[%d]: name is required", i))
				}
				if cert.CertificateAuthorityName == "" {
					multiError = multierror.Append(multiError, fmt.Errorf("android_settings.certificates[%d]: certificate_authority_name is required", i))
				}
				if cert.SubjectName == "" {
					multiError = multierror.Append(multiError, fmt.Errorf("android_settings.certificates[%d]: subject_name is required", i))
				}
			}
		}

		// Since we already unmarshalled and updated the path, we need to update the result struct.
		result.Controls.AndroidSettings = androidSettings
	}

	return multiError
}

func processControlsPathIfNeeded(controlsTop GitOpsControls, result *GitOps, controlsFilePath *string) []error {
	if controlsTop.Path == nil {
		result.Controls = controlsTop
		return nil
	}

	// There is a path attribute which points to the real controls section in a separate file, so we need to process that.
	controlsFilePath = ptr.String(resolveApplyRelativePath(filepath.Dir(*controlsFilePath), *controlsTop.Path))
	fileBytes, err := os.ReadFile(*controlsFilePath)
	if err != nil {
		return []error{fmt.Errorf("failed to read controls file %s: %v", *controlsTop.Path, err)}
	}

	// Replace $var and ${var} with env values.
	fileBytes, err = ExpandEnvBytes(fileBytes)
	if err != nil {
		return []error{fmt.Errorf("failed to expand environment in file %s: %v", *controlsTop.Path, err)}
	}

	var errs []error
	var pathControls GitOpsControls
	jsonBytes, err := yaml.YAMLToJSON(fileBytes)
	if err != nil {
		return []error{MaybeParseTypeError(*controlsTop.Path, []string{"controls"}, fmt.Errorf("failed to unmarshal YAML to JSON: %w", err))}
	}
	jsonBytes, _, err = rewriteNewToOldKeys(jsonBytes, &GitOpsControls{})
	if err != nil {
		return []error{fmt.Errorf("failed to rewrite controls keys in %s: %v", *controlsTop.Path, err)}
	}
	if err := json.Unmarshal(jsonBytes, &pathControls); err != nil {
		return []error{MaybeParseTypeError(*controlsTop.Path, []string{"controls"}, err)}
	}
	// Validate unknown keys in path-referenced controls file.
	errs = append(errs, validateYAMLKeys(fileBytes, reflect.TypeFor[GitOpsControls](), *controlsTop.Path, []string{"controls"})...)
	if pathControls.Path != nil {
		return append(errs, fmt.Errorf("nested paths are not supported: %s in %s", *pathControls.Path, *controlsTop.Path))
	}
	pathControls.Defined = true
	result.Controls = pathControls
	return errs
}

func resolveAndUpdateProfilePath(profile *fleet.MDMProfileSpec, result *GitOps) error {
	// Path has already been resolved by expandBaseItems; just ensure it's absolute.
	var err error
	profile.Path, err = filepath.Abs(profile.Path)
	if err != nil {
		return fmt.Errorf("failed to resolve profile path %s: %v", profile.Path, err)
	}
	fileBytes, err := os.ReadFile(profile.Path)
	if err != nil {
		return fmt.Errorf("failed to read profile file %s: %v", profile.Path, err)
	}
	err = LookupEnvSecrets(string(fileBytes), result.FleetSecrets)
	if err != nil {
		return err
	}
	return nil
}

// defaultAllowedExtensions is the default set of file extensions allowed for
// glob expansion (YAML files). Entity types that need different extensions
// (e.g. scripts) should override this in their GlobExpandOptions.
var defaultAllowedExtensions = map[string]bool{
	".yml":  true,
	".yaml": true,
}

// allowedScriptExtensions is the set of file extensions allowed for scripts.
var allowedScriptExtensions = map[string]bool{
	".sh":  true,
	".ps1": true,
	".py":  true,
}

// GlobExpandOptions configures how flattenBaseItems expands glob patterns.
type GlobExpandOptions struct {
	// AllowedExtensions filters glob results to only these extensions.
	// Files with other extensions are skipped with a warning.
	// Defaults to {".yml", ".yaml"} if nil.
	AllowedExtensions map[string]bool
	// RequireUniqueBasenames, if true, returns an error when two items resolve to the
	// same filename (filepath.Base).
	RequireUniqueBasenames bool
	// RequireFileReference, if true, returns an error when an item has neither
	// "path" nor "paths" set. When false, such items are passed through unchanged.
	RequireFileReference bool
	// Optional function to log warnings (e.g. about files skipped due to extension mismatch).
	LogFn Logf
}

func (o *GlobExpandOptions) setDefaults() {
	if o.AllowedExtensions == nil {
		o.AllowedExtensions = defaultAllowedExtensions
	}
	if o.LogFn == nil {
		o.LogFn = func(_ string, _ ...any) {}
	}
}

// containsGlobMeta returns true if the string contains glob metacharacters.
func containsGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[{")
}

// expandGlobPattern expands a glob pattern relative to baseDir and returns
// all of the matching files with allowed extensions.
func expandGlobPattern(pattern string, baseDir string, entityType string, opts GlobExpandOptions) ([]string, error) {
	absPattern := resolveApplyRelativePath(baseDir, pattern)
	matches, err := doublestar.FilepathGlob(absPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	var result []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", match, err)
		}
		if info.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(match))
		if !opts.AllowedExtensions[ext] {
			opts.LogFn("[!] glob pattern %q matched non-%s file %q, skipping\n", pattern, entityType, match)
			continue
		}
		result = append(result, match)
	}

	slices.Sort(result)
	return result, nil
}

// expandBaseItems validates path/paths fields on each entity (e.g. Label), expands glob
// patterns in "paths" entries, and returns a flat list where every entity with
// a file reference has only Path set (resolved to an absolute path). Entities
// without path/paths are passed through unchanged unless
// opts.RequireFileReference is set, in which case an error is returned.
// Errors are collected rather than returned early, so callers get all
// problems in one pass.
func expandBaseItems[T any, PT interface {
	*T
	fleet.SupportsFileInclude
}](inputEntities []T, baseDir string, entityType string, opts GlobExpandOptions) ([]T, []error) {
	opts.setDefaults()
	var result []T
	var errs []error
	seenBasenames := make(map[string]string) // basename -> source (path or pattern)

	for _, entity := range inputEntities {
		baseItem := PT(&entity).GetBaseItem()
		hasPath := baseItem.Path != nil
		hasPaths := baseItem.Paths != nil

		switch {
		case hasPath && hasPaths:
			errs = append(errs, fmt.Errorf(`%s entry cannot have both "path" and "paths" fields`, entityType))
			continue
		// Inline entity (no file reference).
		case !hasPath && !hasPaths:
			if opts.RequireFileReference {
				errs = append(errs, fmt.Errorf(`%s entry has no "path" or "paths" field; check for a stray "-" in the list`, entityType))
				continue
			}
			result = append(result, entity)
		// Single path -- resolve to absolute path and add to result.
		case hasPath:
			if containsGlobMeta(*baseItem.Path) {
				errs = append(errs, fmt.Errorf(`%s "path" %q contains glob characters; use "paths" for glob patterns`, entityType, *baseItem.Path))
				continue
			}
			resolved := resolveApplyRelativePath(baseDir, *baseItem.Path)
			// Check for duplicate filenames if requested.
			if opts.RequireUniqueBasenames {
				base := filepath.Base(resolved)
				if existing, ok := seenBasenames[base]; ok {
					errs = append(errs, fmt.Errorf("duplicate %s basename %q (from %q and %q)", entityType, base, existing, *baseItem.Path))
					continue
				}
				seenBasenames[base] = *baseItem.Path
			}
			PT(&entity).SetBaseItem(fleet.BaseItem{Path: &resolved})
			result = append(result, entity)
		// Glob -- expand and add files to result.
		case hasPaths:
			if !containsGlobMeta(*baseItem.Paths) {
				errs = append(errs, fmt.Errorf(`%s "paths" %q does not contain glob characters; use "path" for a specific file`, entityType, *baseItem.Paths))
				continue
			}
			expanded, err := expandGlobPattern(*baseItem.Paths, baseDir, entityType, opts)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if len(expanded) == 0 {
				opts.LogFn("[!] glob pattern %q matched no %s files\n", *baseItem.Paths, entityType)
				continue
			}
			for _, p := range expanded {
				// Check for duplicate filenames if requested.
				if opts.RequireUniqueBasenames {
					base := filepath.Base(p)
					if existing, ok := seenBasenames[base]; ok {
						errs = append(errs, fmt.Errorf("duplicate %s basename %q (from %q and %q)", entityType, base, existing, *baseItem.Paths))
						continue
					}
					seenBasenames[base] = *baseItem.Paths
				}
				newItem := entity // clone to preserve non-BaseItem fields (e.g. labels)
				PT(&newItem).SetBaseItem(fleet.BaseItem{Path: &p})
				result = append(result, newItem)
			}
		}
	}

	return result, errs
}

func resolveScriptPaths(input []fleet.BaseItem, baseDir string, logFn Logf) ([]fleet.BaseItem, []error) {
	return expandBaseItems(input, baseDir, "script", GlobExpandOptions{
		AllowedExtensions:      allowedScriptExtensions,
		RequireUniqueBasenames: true,
		RequireFileReference:   true,
		LogFn:                  logFn,
	})
}

func parseLabels(top map[string]json.RawMessage, result *GitOps, baseDir string, logFn Logf, filePath string, multiError *multierror.Error) *multierror.Error {
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
	var errs []error
	if labels, errs = expandBaseItems(labels, baseDir, "label", GlobExpandOptions{
		LogFn: logFn,
	}); len(errs) > 0 {
		multiError = multierror.Append(multiError, errs...)
	}
	// Validate unknown keys in labels section.
	multiError = multierror.Append(multiError, validateRawKeys(labelsRaw, reflect.TypeFor[[]Label](), filePath, []string{"labels"})...)
	for _, item := range labels {
		if item.Path == nil {
			result.Labels = append(result.Labels, &item.LabelSpec)
		} else {
			fileBytes, err := os.ReadFile(*item.Path)
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
				// Validate unknown keys in path-referenced labels file.
				multiError = multierror.Append(multiError, validateYAMLKeys(fileBytes, reflect.TypeFor[[]Label](), *item.Path, []string{"labels"})...)
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

		// Validate mutually exclusive field combinations per label membership type
		if err := fleet.ValidateLabelMembershipFields(l); err != nil {
			for _, inv := range err.Invalid() {
				multiError = multierror.Append(multiError, fmt.Errorf("%s", inv["reason"]))
			}
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

func parsePolicies(top map[string]json.RawMessage, result *GitOps, baseDir string, logFn Logf, filePath string, multiError *multierror.Error) *multierror.Error {
	parentFilePath := filePath
	policiesRaw, ok := top["policies"]
	if !ok {
		return multierror.Append(multiError, errors.New("'policies' key is required"))
	}
	var policies []Policy
	if err := json.Unmarshal(policiesRaw, &policies); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"policies"}, err))
	}

	// make an index of all FMAs by slug
	fmasBySlug := make(map[string]struct{}, len(result.Software.FleetMaintainedApps))
	for _, s := range result.Software.FleetMaintainedApps {
		fmasBySlug[s.Slug] = struct{}{}
	}
	var errs []error
	if policies, errs = expandBaseItems(policies, baseDir, "policy", GlobExpandOptions{
		LogFn: logFn,
	}); len(errs) > 0 {
		multiError = multierror.Append(multiError, errs...)
	}

	// Validate unknown keys in policies section.
	multiError = multierror.Append(multiError, validateRawKeys(policiesRaw, reflect.TypeFor[[]Policy](), filePath, []string{"policies"})...)
	for _, item := range policies {
		if item.Path == nil {
			if errs := parsePolicyInstallSoftware(baseDir, result.TeamName, &item, result.Software.Packages, result.Software.AppStoreApps, fmasBySlug); errs != nil {
				multiError = multierror.Append(multiError, errs...)
				continue
			}
			if err := parsePolicyRunScript(baseDir, parentFilePath, result.TeamName, &item, result.Controls.Scripts); err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to parse policy run_script %q: %v", item.Name, err))
				continue
			}
			result.Policies = append(result.Policies, &item.GitOpsPolicySpec)
		} else {
			fileBytes, err := os.ReadFile(*item.Path)
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
				// Validate unknown keys in path-referenced policies file.
				multiError = multierror.Append(multiError, validateYAMLKeys(fileBytes, reflect.TypeFor[[]Policy](), *item.Path, []string{"policies"})...)
				for _, pp := range pathPolicies {
					if pp != nil {
						if pp.Path != nil {
							multiError = multierror.Append(
								multiError, fmt.Errorf("nested paths are not supported: %s in %s", *pp.Path, *item.Path),
							)
						} else {
							if errs := parsePolicyInstallSoftware(filepath.Dir(*item.Path), result.TeamName, pp, result.Software.Packages, result.Software.AppStoreApps, fmasBySlug); errs != nil {
								multiError = multierror.Append(multiError, errs...)
								continue
							}
							if err := parsePolicyRunScript(filepath.Dir(*item.Path), parentFilePath, result.TeamName, pp, result.Controls.Scripts); err != nil {
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
		if item.Type == "" {
			item.Type = fleet.PolicyTypeDynamic
		}
		if item.Query == "" && item.Type != fleet.PolicyTypePatch {
			multiError = multierror.Append(multiError, errors.New("policy query is required for each policy"))
		}
		if item.Type == fleet.PolicyTypePatch {
			if _, ok := fmasBySlug[item.FleetMaintainedAppSlug]; !ok {
				multiError = multierror.Append(
					multiError,
					fmt.Errorf(
						`Couldn't apply "%s": "%s" is specified in the patch policy, but it isn't specified under "software.fleet_maintained_apps."`,
						filepath.Base(parentFilePath),
						item.FleetMaintainedAppSlug,
					),
				)
			}
		}
		if result.TeamName != nil {
			item.Team = *result.TeamName
		} else {
			item.Team = ""
		}
		if item.CalendarEventsEnabled && result.IsNoTeam() {
			multiError = multierror.Append(multiError, fmt.Errorf("calendar events are not supported on policies included in `%s`: %q", filepath.Base(parentFilePath), item.Name))
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

func parsePolicyRunScript(baseDir string, parentFilePath string, teamName *string, policy *Policy, scripts []fleet.BaseItem) error {
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
			return fmt.Errorf("policy script %s was not defined in controls in %s", scriptPath, filepath.Base(parentFilePath))
		}
		return fmt.Errorf("policy script %s was not defined in controls for %s", scriptPath, *teamName)
	}

	scriptName := filepath.Base(policy.RunScript.Path)
	policy.RunScriptName = &scriptName

	return nil
}

func parsePolicyInstallSoftware(baseDir string, teamName *string, policy *Policy, packages []*fleet.SoftwarePackageSpec, appStoreApps []*fleet.TeamSpecAppStoreApp, fmasBySlug map[string]struct{}) []error {
	installSoftwareObj := policy.InstallSoftware.Other
	if installSoftwareObj == nil {
		policy.SoftwareTitleID = ptr.Uint(0) // unset the installer
		return nil
	}
	errPrefix := fmt.Sprintf("failed to parse policy install_software %q: ", policy.Name)
	wrapErr := func(err error) error {
		return fmt.Errorf("%s%w", errPrefix, err)
	}
	wrapErrs := func(err error) []error {
		return []error{wrapErr(err)}
	}
	if (installSoftwareObj.PackagePath != "" || installSoftwareObj.AppStoreID != "" || installSoftwareObj.HashSHA256 != "" || installSoftwareObj.FleetMaintainedAppSlug != "") && teamName == nil {
		return wrapErrs(errors.New("install_software can only be set on team policies"))
	}
	if installSoftwareObj.PackagePath == "" && installSoftwareObj.AppStoreID == "" && installSoftwareObj.HashSHA256 == "" && installSoftwareObj.FleetMaintainedAppSlug == "" {
		return wrapErrs(errors.New("install_software must include either a package_path, an app_store_id, a hash_sha256 or a fleet_maintained_app_slug"))
	}
	setCount := 0
	for _, s := range []string{installSoftwareObj.PackagePath, installSoftwareObj.AppStoreID, installSoftwareObj.HashSHA256, installSoftwareObj.FleetMaintainedAppSlug} {
		if s != "" {
			setCount++
		}
	}
	if setCount > 1 {
		return wrapErrs(errors.New("install_software must have only one of package_path, app_store_id, hash_sha256 or fleet_maintained_app_slug"))
	}

	var errs []error
	if installSoftwareObj.PackagePath != "" {
		fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, installSoftwareObj.PackagePath))
		if err != nil {
			return wrapErrs(fmt.Errorf("failed to read install_software.package_path file %q: %v", installSoftwareObj.PackagePath, err))
		}
		// Replace $var and ${var} with env values.
		fileBytes, err = ExpandEnvBytes(fileBytes)
		if err != nil {
			return wrapErrs(fmt.Errorf("failed to expand environment in file %q: %v", installSoftwareObj.PackagePath, err))
		}
		var policyInstallSoftwareSpec fleet.SoftwarePackageSpec
		if err := YamlUnmarshal(fileBytes, &policyInstallSoftwareSpec); err != nil {
			// see if the issue is that a package path was passed in that references multiple packages
			var multiplePackages []fleet.SoftwarePackageSpec
			if err := YamlUnmarshal(fileBytes, &multiplePackages); err != nil || len(multiplePackages) == 0 {
				return wrapErrs(fmt.Errorf("file %q does not contain a valid software package definition", installSoftwareObj.PackagePath))
			}

			if len(multiplePackages) > 1 {
				return wrapErrs(fmt.Errorf("file %q contains multiple packages, so cannot be used as a target for policy automation", installSoftwareObj.PackagePath))
			}

			errs = append(errs, validateYAMLKeys(fileBytes, reflect.TypeFor[[]fleet.SoftwarePackageSpec](), installSoftwareObj.PackagePath, []string{"software", "packages"})...)
			policyInstallSoftwareSpec = multiplePackages[0]
		} else {
			errs = append(errs, validateYAMLKeys(fileBytes, reflect.TypeFor[fleet.SoftwarePackageSpec](), installSoftwareObj.PackagePath, []string{"software", "packages"})...)
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
				errs = append(errs, wrapErr(fmt.Errorf("install_software.package_path URL %s not found on team: %s", policyInstallSoftwareSpec.URL, installSoftwareObj.PackagePath)))
			} else {
				errs = append(errs, wrapErr(fmt.Errorf("install_software.package_path SHA256 %s not found on team: %s", policyInstallSoftwareSpec.SHA256, installSoftwareObj.PackagePath)))
			}
			return errs
		}

		policy.InstallSoftwareURL = policyInstallSoftwareSpec.URL
		policy.InstallSoftware.Other.HashSHA256 = policyInstallSoftwareSpec.SHA256
	}

	if policy.InstallSoftware.Other.AppStoreID != "" {
		appOnTeamFound := false
		for _, app := range appStoreApps {
			if app.AppStoreID == policy.InstallSoftware.Other.AppStoreID {
				appOnTeamFound = true
				break
			}
		}
		if !appOnTeamFound {
			errs = append(errs, wrapErr(fmt.Errorf("install_software.app_store_id %s not found on team %s", policy.InstallSoftware.Other.AppStoreID, *teamName)))
		}
	}

	if installSoftwareObj.FleetMaintainedAppSlug != "" {
		if _, ok := fmasBySlug[installSoftwareObj.FleetMaintainedAppSlug]; !ok {
			errs = append(errs, wrapErr(fmt.Errorf("install_software.fleet_maintained_app_slug %q not found in software.fleet_maintained_apps for team %s", installSoftwareObj.FleetMaintainedAppSlug, *teamName)))
		}
		policy.FleetMaintainedAppSlug = installSoftwareObj.FleetMaintainedAppSlug
	}

	return errs
}

func validateReport(r *fleet.QuerySpec, filePath string, itemPath string) error {
	if r.Name == "" {
		return fmt.Errorf("`name` is required for each report in %s at %s", filepath.Base(filePath), itemPath)
	}
	if r.Query == "" {
		return fmt.Errorf("`query` is required for each report in %s at %s", filepath.Base(filePath), itemPath)
	}
	if !isASCII(r.Name) {
		return fmt.Errorf("`name` must be in ASCII: %s in %s at %s", r.Name, filepath.Base(filePath), itemPath)
	}
	return nil
}

func parseReports(top map[string]json.RawMessage, result *GitOps, baseDir string, logFn Logf, filePath string, multiError *multierror.Error) *multierror.Error {
	reportsRaw, ok := top["reports"]
	if result.IsNoTeam() {
		if ok {
			logFn("[!] 'reports' is not supported in %s. This key will be ignored.\n", filepath.Base(filePath))
		}
		return multiError
	} else if !ok {
		return multierror.Append(multiError, errors.New("'reports' key is required"))
	}
	var queries []Query
	if err := json.Unmarshal(reportsRaw, &queries); err != nil {
		return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"reports"}, err))
	}
	var errs []error
	if queries, errs = expandBaseItems(queries, baseDir, "report", GlobExpandOptions{
		LogFn: logFn,
	}); len(errs) > 0 {
		multiError = multierror.Append(multiError, errs...)
	}

	// Validate unknown keys in reports section.
	multiError = multierror.Append(multiError, validateRawKeys(reportsRaw, reflect.TypeFor[[]Query](), filePath, []string{"reports"})...)
	for i, item := range queries {
		if item.Path == nil {
			if err := validateReport(&item.QuerySpec, filePath, fmt.Sprintf("reports[%d]", i)); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
			item.QuerySpec.TeamName = ptr.ValOrZero(result.TeamName)
			result.Queries = append(result.Queries, &item.QuerySpec)
		} else {
			fileBytes, err := os.ReadFile(*item.Path)
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to read reports file %s: %v", *item.Path, err))
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
					multiError = multierror.Append(multiError, MaybeParseTypeError(*item.Path, []string{"reports"}, err))
					continue
				}
				// Validate unknown keys in path-referenced reports file.
				multiError = multierror.Append(multiError, validateYAMLKeys(fileBytes, reflect.TypeFor[[]Query](), *item.Path, []string{"reports"})...)
				for i, pq := range pathQueries {
					if pq != nil {
						if pq.Path != nil {
							multiError = multierror.Append(
								multiError, fmt.Errorf("nested paths are not supported: %s in %s", *pq.Path, *item.Path),
							)
						} else {
							if err := validateReport(&pq.QuerySpec, *item.Path, fmt.Sprintf("reports[%d]", i)); err != nil {
								multiError = multierror.Append(multiError, err)
								continue
							}
							pq.QuerySpec.TeamName = ptr.ValOrZero(result.TeamName)
							result.Queries = append(result.Queries, &pq.QuerySpec)
						}
					}
				}
			}
		}
	}
	duplicates := getDuplicateNames(
		result.Queries, func(q *fleet.QuerySpec) string {
			return q.Name
		},
	)
	if len(duplicates) > 0 {
		multiError = multierror.Append(multiError, fmt.Errorf("duplicate report names: %v", duplicates))
	}
	return multiError
}

var validSHA256Value = regexp.MustCompile(`\b[a-f0-9]{64}\b`)

func parseSoftware(top map[string]json.RawMessage, result *GitOps, baseDir string, filePath string, options GitOpsOptions, multiError *multierror.Error) *multierror.Error {
	softwareRaw, ok := top["software"]
	if ok {
		result.SoftwarePresent = true
	}
	if result.global() {
		if ok && string(softwareRaw) != "null" {
			return multierror.Append(multiError, errors.New("'software' cannot be set on global file"))
		}
	} else if !ok {
		// Software key is absent. If we have synthetic server-side data for this
		// team (because software is excepted from GitOps), inject it so that
		// policy install_software and patch policy references can be validated.
		// SoftwarePresent remains false so downstream exception enforcement works.
		if result.TeamName != nil && options.SyntheticSoftwareByTeam != nil {
			if synthetic, hasSynthetic := options.SyntheticSoftwareByTeam[*result.TeamName]; hasSynthetic {
				softwareRaw = synthetic
				ok = true // allow processing below
			}
		}
		if !ok {
			return multiError
		}
	}
	var software Software
	if len(softwareRaw) > 0 {
		if err := json.Unmarshal(softwareRaw, &software); err != nil {
			return multierror.Append(multiError, MaybeParseTypeError(filePath, []string{"software"}, err))
		}
		// Validate unknown keys in software section.
		multiError = multierror.Append(multiError, validateRawKeys(softwareRaw, reflect.TypeFor[Software](), filePath, []string{"software"})...)
	}
	for _, item := range software.AppStoreApps {
		if item.AppStoreID == "" {
			multiError = multierror.Append(multiError, errors.New("software app store id required"))
			continue
		}

		var count int
		for _, set := range [][]string{item.LabelsExcludeAny, item.LabelsIncludeAny, item.LabelsIncludeAll} {
			if len(set) > 0 {
				count++
			}
		}
		if count > 1 {
			multiError = multierror.Append(multiError, fmt.Errorf(`only one of "labels_include_all", "labels_exclude_any" or "labels_include_any" can be specified for app store app %q`, item.AppStoreID))
			continue
		}

		// Validate display_name length (matches database VARCHAR(255))
		if len(item.DisplayName) > 255 {
			multiError = multierror.Append(multiError, fmt.Errorf("app_store_id %q display_name is too long (max 255 characters)", item.AppStoreID))
			continue
		}

		item = item.ResolvePaths(baseDir)

		result.Software.AppStoreApps = append(result.Software.AppStoreApps, &item)
	}
	for _, maintainedAppSpec := range software.FleetMaintainedApps {
		if maintainedAppSpec.Slug == "" {
			multiError = multierror.Append(multiError, errors.New("fleet maintained app slug is required"))
			continue
		}

		var count int
		for _, set := range [][]string{maintainedAppSpec.LabelsExcludeAny, maintainedAppSpec.LabelsIncludeAny, maintainedAppSpec.LabelsIncludeAll} {
			if len(set) > 0 {
				count++
			}
		}
		if count > 1 {
			multiError = multierror.Append(multiError, fmt.Errorf(`only one of "labels_include_all", "labels_exclude_any" or "labels_include_any" can be specified for fleet maintained app %q`, maintainedAppSpec.Slug))
			continue
		}

		maintainedAppSpec = maintainedAppSpec.ResolveSoftwarePackagePaths(baseDir)

		// handle secrets
		if maintainedAppSpec.InstallScript.Path != "" {
			if err := gatherFileSecrets(result, maintainedAppSpec.InstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}
		if maintainedAppSpec.PostInstallScript.Path != "" {
			if err := gatherFileSecrets(result, maintainedAppSpec.PostInstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}
		if maintainedAppSpec.UninstallScript.Path != "" {
			if err := gatherFileSecrets(result, maintainedAppSpec.UninstallScript.Path); err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}
		}

		result.Software.FleetMaintainedApps = append(result.Software.FleetMaintainedApps, &maintainedAppSpec)
	}
	for _, teamLevelPackage := range software.Packages {
		// A single item in Packages can result in multiple SoftwarePackageSpecs being generated
		var softwarePackageSpecs []*fleet.SoftwarePackageSpec
		if teamLevelPackage.Path != nil {
			resolvedPath := resolveApplyRelativePath(baseDir, *teamLevelPackage.Path)
			fileBytes, err := os.ReadFile(resolvedPath)
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to read software package file %s: %w", *teamLevelPackage.Path, err))
				continue
			}

			ext := strings.ToLower(filepath.Ext(resolvedPath))
			switch ext {
			case ".sh", ".ps1":
				// Script files: only gather FLEET_SECRET_ variables, don't expand
				// regular env vars (they are shell variables meant for the endpoint).
				if err := gatherFileSecrets(result, resolvedPath); err != nil {
					multiError = multierror.Append(multiError, err)
					continue
				}
				// Script file becomes the install script for a script-only package
				scriptSpec := fleet.SoftwarePackageSpec{
					ReferencedYamlPath: resolvedPath,
					Icon:               teamLevelPackage.Icon,
				}
				// Icon path needs to be resolved, but since this function will set
				// the install script it needs to be set to the correct path again.
				scriptSpec = scriptSpec.ResolveSoftwarePackagePaths(baseDir)
				scriptSpec.InstallScript.Path = resolvedPath

				scriptSpec, err = teamLevelPackage.HydrateToPackageLevel(scriptSpec, ext)
				if err != nil {
					multiError = multierror.Append(multiError, err)
					continue
				}
				softwarePackageSpecs = append(softwarePackageSpecs, &scriptSpec)

			case ".yml", ".yaml":
				// Replace $var and ${var} with env values in YAML files only.
				fileBytes, err = ExpandEnvBytes(fileBytes)
				if err != nil {
					multiError = multierror.Append(multiError, fmt.Errorf("failed to expand environment in file %s: %w", *teamLevelPackage.Path, err))
					continue
				}
				var singlePackageSpec SoftwarePackage
				singlePackageSpec.ReferencedYamlPath = resolvedPath
				if err := YamlUnmarshal(fileBytes, &singlePackageSpec); err == nil {
					multiError = multierror.Append(multiError, validateYAMLKeys(fileBytes, reflect.TypeFor[SoftwarePackage](), *teamLevelPackage.Path, []string{"software", "packages"})...)
					if singlePackageSpec.IncludesFieldsDisallowedInPackageFile() {
						multiError = multierror.Append(multiError, fmt.Errorf("labels, categories, setup_experience, and self_service values must be specified at the team level; package-level specified in %s", *teamLevelPackage.Path))
						continue
					}
					softwarePackageSpecs = append(softwarePackageSpecs, &singlePackageSpec.SoftwarePackageSpec)
				} else if err = YamlUnmarshal(fileBytes, &softwarePackageSpecs); err == nil {
					// Failing that, try to unmarshal as a list of SoftwarePackageSpecs
					multiError = multierror.Append(multiError, validateYAMLKeys(fileBytes, reflect.TypeFor[[]fleet.SoftwarePackageSpec](), *teamLevelPackage.Path, []string{"software", "packages"})...)
					for i, spec := range softwarePackageSpecs {
						if spec.IncludesFieldsDisallowedInPackageFile() {
							multiError = multierror.Append(multiError, fmt.Errorf("labels, categories, setup_experience, and self_service values must be specified at the team level; package-level specified in %s", *teamLevelPackage.Path))
							continue
						}

						softwarePackageSpecs[i].ReferencedYamlPath = resolvedPath
					}
				} else {
					// If we reached here, we couldn't unmarshal as either format.
					multiError = multierror.Append(multiError, MaybeParseTypeError(*teamLevelPackage.Path, []string{"software", "packages"}, err))
					continue
				}

				for i, spec := range softwarePackageSpecs {
					softwarePackageSpec := spec.ResolveSoftwarePackagePaths(filepath.Dir(spec.ReferencedYamlPath))
					softwarePackageSpec, err = teamLevelPackage.HydrateToPackageLevel(softwarePackageSpec, ext)
					if err != nil {
						multiError = multierror.Append(multiError, err)
						continue
					}
					softwarePackageSpecs[i] = &softwarePackageSpec
				}

			default:
				multiError = multierror.Append(multiError, fmt.Errorf("software package path %s has unsupported extension %q; only .yml, .yaml, .sh, or .ps1 files are supported", *teamLevelPackage.Path, ext))
				continue
			}
		} else {
			softwarePackageSpec := teamLevelPackage.SoftwarePackageSpec.ResolveSoftwarePackagePaths(baseDir)
			softwarePackageSpecs = append(softwarePackageSpecs, &softwarePackageSpec)
		}

		for i, softwarePackageSpec := range softwarePackageSpecs {
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
			if softwarePackageSpec.UninstallScript.Path != "" {
				if err := gatherFileSecrets(result, softwarePackageSpec.UninstallScript.Path); err != nil {
					multiError = multierror.Append(multiError, err)
					continue
				}
			}

			var count int
			for _, set := range [][]string{softwarePackageSpec.LabelsExcludeAny, softwarePackageSpec.LabelsIncludeAny, softwarePackageSpec.LabelsIncludeAll} {
				if len(set) > 0 {
					count++
				}
			}
			if count > 1 {
				multiError = multierror.Append(multiError, fmt.Errorf(`only one of "labels_include_all", "labels_exclude_any" or "labels_include_any" can be specified for software URL %q`, softwarePackageSpec.URL))
				continue
			}

			if softwarePackageSpec.SHA256 != "" && !validSHA256Value.MatchString(softwarePackageSpec.SHA256) {
				multiError = multierror.Append(multiError, fmt.Errorf("hash_sha256 value %q must be a valid lower-case hex-encoded (64-character) SHA-256 hash value", softwarePackageSpec.SHA256))
				continue
			}
			// Script packages from path don't require URL or hash_sha256
			isScriptPackageFromPath := fleet.IsScriptPackage(filepath.Ext(softwarePackageSpec.ReferencedYamlPath))
			if !isScriptPackageFromPath && softwarePackageSpec.SHA256 == "" && softwarePackageSpec.URL == "" {
				errorMessage := "at least one of hash_sha256 or url is required for each software package"
				if softwarePackageSpec.ReferencedYamlPath != "" {
					errorMessage += fmt.Sprintf("; missing in %s", softwarePackageSpec.ReferencedYamlPath)
				}
				if len(softwarePackageSpecs) > 1 {
					errorMessage += fmt.Sprintf(", list item #%d", i+1)
				}

				multiError = multierror.Append(multiError, errors.New(errorMessage))
				continue
			}

			// Skip URL-related validations for script packages from path
			if !isScriptPackageFromPath {
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
					} else if strings.HasSuffix(parsedUrl.Path, ".tar.gz") || strings.HasSuffix(parsedUrl.Path, ".tgz") {
						multiError = multierror.Append(multiError, fmt.Errorf("software URL %s refers to a .tar.gz archive, which requires both install_script and uninstall_script", softwarePackageSpec.URL))
						continue
					}
				}
			}

			// Validate display_name length (matches database VARCHAR(255))
			if len(softwarePackageSpec.DisplayName) > 255 {
				multiError = multierror.Append(multiError, fmt.Errorf("software package %q display_name is too long (max 255 characters)", softwarePackageSpec.URL))
				continue
			}

			result.Software.Packages = append(result.Software.Packages, softwarePackageSpec)
		}
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
