package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	kithttp "github.com/go-kit/kit/transport/http"
)

const batchSize = 100

// Client is used to consume Fleet APIs from Go code
type Client struct {
	*baseClient
	addr          string
	token         string
	customHeaders map[string]string

	outputWriter io.Writer
	errWriter    io.Writer
}

type ClientOption func(*Client) error

func NewClient(addr string, insecureSkipVerify bool, rootCA, urlPrefix string, options ...ClientOption) (*Client, error) {
	// TODO #265 refactor all optional parameters to functional options
	// API breaking change, needs a major version release
	baseClient, err := newBaseClient(addr, insecureSkipVerify, rootCA, urlPrefix, nil, fleet.CapabilityMap{})
	if err != nil {
		return nil, err
	}

	client := &Client{
		baseClient: baseClient,
		addr:       addr,
	}

	for _, option := range options {
		err := option(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func EnableClientDebug() ClientOption {
	return func(c *Client) error {
		httpClient, ok := c.http.(*http.Client)
		if !ok {
			return errors.New("client is not *http.Client")
		}
		httpClient.Transport = &logRoundTripper{roundtripper: httpClient.Transport}

		return nil
	}
}

func SetClientOutputWriter(w io.Writer) ClientOption {
	return func(c *Client) error {
		c.outputWriter = w
		return nil
	}
}

func SetClientErrorWriter(w io.Writer) ClientOption {
	return func(c *Client) error {
		c.errWriter = w
		return nil
	}
}

// WithCustomHeaders sets custom headers to be sent with every request made
// with the client.
func WithCustomHeaders(headers map[string]string) ClientOption {
	return func(c *Client) error {
		// clone the map to prevent any changes in the original affecting the client
		m := make(map[string]string, len(headers))
		for k, v := range headers {
			m[k] = v
		}
		c.customHeaders = m
		return nil
	}
}

func (c *Client) doContextWithBodyAndHeaders(ctx context.Context, verb, path, rawQuery string, bodyBytes []byte, headers map[string]string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		verb,
		c.url(path, rawQuery).String(),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating request object")
	}

	// set the custom headers first, they should not override the actual headers
	// we set explicitly.
	for k, v := range c.customHeaders {
		request.Header.Set(k, v)
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	resp, err := c.http.Do(request)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "do request")
	}

	if resp.Header.Get(fleet.HeaderLicenseKey) == fleet.HeaderLicenseValueExpired {
		fleet.WriteExpiredLicenseBanner(c.errWriter)
	}

	return resp, nil
}

func (c *Client) doContextWithHeaders(ctx context.Context, verb, path, rawQuery string, params interface{}, headers map[string]string) (*http.Response, error) {
	var bodyBytes []byte
	var err error
	if params != nil {
		bodyBytes, err = json.Marshal(params)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "marshaling json")
		}
	}
	return c.doContextWithBodyAndHeaders(ctx, verb, path, rawQuery, bodyBytes, headers)
}

func (c *Client) Do(verb, path, rawQuery string, params interface{}) (*http.Response, error) {
	return c.DoContext(context.Background(), verb, path, rawQuery, params)
}

func (c *Client) DoContext(ctx context.Context, verb, path, rawQuery string, params interface{}) (*http.Response, error) {
	headers := map[string]string{
		"Content-type": "application/json",
		"Accept":       "application/json",
	}

	return c.doContextWithHeaders(ctx, verb, path, rawQuery, params, headers)
}

func (c *Client) AuthenticatedDo(verb, path, rawQuery string, params interface{}) (*http.Response, error) {
	if c.token == "" {
		return nil, errors.New("authentication token is empty")
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.token),
	}

	return c.doContextWithHeaders(context.Background(), verb, path, rawQuery, params, headers)
}

func (c *Client) SetToken(t string) {
	c.token = t
}

// http.RoundTripper that will log debug information about the request and
// response, including paths, timing, and body.
//
// Inspired by https://stackoverflow.com/a/39528716/491710 and
// github.com/motemen/go-loghttp
type logRoundTripper struct {
	roundtripper http.RoundTripper
}

// RoundTrip implements http.RoundTripper
func (l *logRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request
	fmt.Fprintf(os.Stderr, "%s %s\n", req.Method, req.URL)
	reqBody, err := req.GetBody()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetBody error: %v\n", err)
	} else {
		defer reqBody.Close()
		if _, err := io.Copy(os.Stderr, reqBody); err != nil {
			fmt.Fprintf(os.Stderr, "Copy body error: %v\n", err)
		}
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Perform request using underlying roundtripper
	start := time.Now()
	res, err := l.roundtripper.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RoundTrip error: %v", err)
		return nil, err
	}

	// Log response
	took := time.Since(start).Truncate(time.Millisecond)
	fmt.Fprintf(os.Stderr, "%s %s %s (%s)\n", res.Request.Method, res.Request.URL, res.Status, took)

	resBody := &bytes.Buffer{}
	resBodyReader := io.TeeReader(res.Body, resBody)
	if _, err := io.Copy(os.Stderr, resBodyReader); err != nil {
		fmt.Fprintf(os.Stderr, "Read body error: %v", err)
		return nil, err
	}
	res.Body = io.NopCloser(resBody)

	return res, nil
}

func (c *Client) authenticatedRequestWithQuery(params interface{}, verb string, path string, responseDest interface{}, query string) error {
	response, err := c.AuthenticatedDo(verb, path, query, params)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	return c.parseResponse(verb, path, response, responseDest)
}

func (c *Client) authenticatedRequest(params interface{}, verb string, path string, responseDest interface{}) error {
	return c.authenticatedRequestWithQuery(params, verb, path, responseDest, "")
}

func (c *Client) CheckAnyMDMEnabled() error {
	return c.runAppConfigChecks(func(ac *fleet.EnrichedAppConfig) error {
		if !ac.MDM.EnabledAndConfigured && !ac.MDM.WindowsEnabledAndConfigured {
			return errors.New(fleet.MDMNotConfiguredMessage)
		}
		return nil
	})
}

func (c *Client) CheckAppleMDMEnabled() error {
	return c.runAppConfigChecks(func(ac *fleet.EnrichedAppConfig) error {
		if !ac.MDM.EnabledAndConfigured {
			return errors.New(fleet.AppleMDMNotConfiguredMessage)
		}
		return nil
	})
}

func (c *Client) CheckPremiumMDMEnabled() error {
	return c.runAppConfigChecks(func(ac *fleet.EnrichedAppConfig) error {
		if ac.License == nil || !ac.License.IsPremium() {
			return errors.New("missing or invalid license")
		}
		if !ac.MDM.EnabledAndConfigured {
			return errors.New(fleet.AppleMDMNotConfiguredMessage)
		}
		return nil
	})
}

func (c *Client) runAppConfigChecks(fn func(ac *fleet.EnrichedAppConfig) error) error {
	appCfg, err := c.GetAppConfig()
	if err != nil {
		var sce kithttp.StatusCoder
		if errors.As(err, &sce) && sce.StatusCode() == http.StatusForbidden {
			// do not return an error, user may not have permission to read app
			// config (e.g. gitops) and those appconfig checks are just convenience
			// to avoid the round-trip with potentially large payload to the server.
			// Those will still be validated with the actual API call.
			return nil
		}
		return err
	}
	return fn(appCfg)
}

// getProfilesContents takes file paths and creates a slice of profile payloads
// ready to batch-apply.
func getProfilesContents(baseDir string, profiles []fleet.MDMProfileSpec) ([]fleet.MDMProfileBatchPayload, error) {
	// map to check for duplicate names
	extByName := make(map[string]string, len(profiles))
	result := make([]fleet.MDMProfileBatchPayload, 0, len(profiles))

	for _, profile := range profiles {
		filePath := resolveApplyRelativePath(baseDir, profile.Path)
		fileContents, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("applying fleet config: %w", err)
		}

		// by default, use the file name. macOS profiles use their PayloadDisplayName
		ext := filepath.Ext(filePath)
		name := strings.TrimSuffix(filepath.Base(filePath), ext)
		if mdm.GetRawProfilePlatform(fileContents) == "darwin" && ext == ".mobileconfig" {
			mc, err := fleet.NewMDMAppleConfigProfile(fileContents, nil)
			if err != nil {
				return nil, fmt.Errorf("applying fleet config: %w", err)
			}
			name = strings.TrimSpace(mc.Name)
		}
		if e, isDuplicate := extByName[name]; isDuplicate {
			return nil, errors.New(fmtDuplicateNameErrMsg(name, e, ext))
		}
		extByName[name] = ext
		result = append(result, fleet.MDMProfileBatchPayload{
			Name:     name,
			Contents: fileContents,
			Labels:   profile.Labels,
		})

	}
	return result, nil
}

// ApplyGroup applies the given spec group to Fleet.
func (c *Client) ApplyGroup(
	ctx context.Context,
	specs *spec.Group,
	baseDir string,
	logf func(format string, args ...interface{}),
	opts fleet.ApplySpecOptions,
) (map[string]uint, error) {
	logfn := func(format string, args ...interface{}) {
		if logf != nil {
			logf(format, args...)
		}
	}

	// specs.Queries must be applied before specs.Packs because packs reference queries.
	if len(specs.Queries) > 0 {
		if opts.DryRun {
			logfn("[!] ignoring queries, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyQueries(specs.Queries); err != nil {
				return nil, fmt.Errorf("applying queries: %w", err)
			}
			logfn("[+] applied %d queries\n", len(specs.Queries))
		}
	}

	if len(specs.Labels) > 0 {
		if opts.DryRun {
			logfn("[!] ignoring labels, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyLabels(specs.Labels); err != nil {
				return nil, fmt.Errorf("applying labels: %w", err)
			}
			logfn("[+] applied %d labels\n", len(specs.Labels))
		}
	}

	if len(specs.Policies) > 0 {
		if opts.DryRun {
			logfn("[!] ignoring policies, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			// Policy names must be unique, return error if duplicate policy names are found
			if policyName := fleet.FirstDuplicatePolicySpecName(specs.Policies); policyName != "" {
				return nil, fmt.Errorf(
					"applying policies: policy names must be unique. Please correct policy %q and try again.", policyName,
				)
			}

			// If set, override the team in all the policies.
			if opts.TeamForPolicies != "" {
				for _, policySpec := range specs.Policies {
					policySpec.Team = opts.TeamForPolicies
				}
			}
			if err := c.ApplyPolicies(specs.Policies); err != nil {
				return nil, fmt.Errorf("applying policies: %w", err)
			}
			logfn("[+] applied %d policies\n", len(specs.Policies))
		}
	}

	if len(specs.Packs) > 0 {
		if opts.DryRun {
			logfn("[!] ignoring packs, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyPacks(specs.Packs); err != nil {
				return nil, fmt.Errorf("applying packs: %w", err)
			}
			logfn("[+] applied %d packs\n", len(specs.Packs))
		}
	}

	if specs.AppConfig != nil {
		windowsCustomSettings := extractAppCfgWindowsCustomSettings(specs.AppConfig)
		macosCustomSettings := extractAppCfgMacOSCustomSettings(specs.AppConfig)
		allCustomSettings := append(macosCustomSettings, windowsCustomSettings...)

		// if there is no custom setting but the windows and mac settings are
		// non-nil, this means that we want to clear the existing custom settings,
		// so we still go on with calling the batch-apply endpoint.
		//
		// TODO(mna): shouldn't that be an || instead of && ? I.e. if there are no
		// custom settings but windows is present and empty (but mac is absent),
		// shouldn't that clear the windows ones?
		if (windowsCustomSettings != nil && macosCustomSettings != nil) || len(allCustomSettings) > 0 {
			fileContents, err := getProfilesContents(baseDir, allCustomSettings)
			if err != nil {
				return nil, err
			}
			// Figure out if MDM should be enabled.
			assumeEnabled := false
			// This cast is safe because we've already checked AppConfig when extracting custom settings
			mdmConfigMap, ok := specs.AppConfig.(map[string]interface{})["mdm"].(map[string]interface{})
			if ok {
				mdmEnabled, ok := mdmConfigMap["windows_enabled_and_configured"]
				if ok {
					assumeEnabled, ok = mdmEnabled.(bool)
					assumeEnabled = ok && assumeEnabled
				}
			}
			if err := c.ApplyNoTeamProfiles(fileContents, opts, assumeEnabled); err != nil {
				return nil, fmt.Errorf("applying custom settings: %w", err)
			}
		}
		if macosSetup := extractAppCfgMacOSSetup(specs.AppConfig); macosSetup != nil {
			if macosSetup.BootstrapPackage.Value != "" {
				pkg, err := c.ValidateBootstrapPackageFromURL(macosSetup.BootstrapPackage.Value)
				if err != nil {
					return nil, fmt.Errorf("applying fleet config: %w", err)
				}

				if !opts.DryRun {
					if err := c.EnsureBootstrapPackage(pkg, uint(0)); err != nil {
						return nil, fmt.Errorf("applying fleet config: %w", err)
					}
				}
			}
			if macosSetup.MacOSSetupAssistant.Value != "" {
				content, err := c.validateMacOSSetupAssistant(resolveApplyRelativePath(baseDir, macosSetup.MacOSSetupAssistant.Value))
				if err != nil {
					return nil, fmt.Errorf("applying fleet config: %w", err)
				}
				if !opts.DryRun {
					if err := c.uploadMacOSSetupAssistant(content, nil, macosSetup.MacOSSetupAssistant.Value); err != nil {
						return nil, fmt.Errorf("applying fleet config: %w", err)
					}
				}
			}
		}
		if scripts := extractAppCfgScripts(specs.AppConfig); scripts != nil {
			files := resolveApplyRelativePaths(baseDir, scripts)
			scriptPayloads := make([]fleet.ScriptPayload, len(files))
			for i, f := range files {
				b, err := os.ReadFile(f)
				if err != nil {
					return nil, fmt.Errorf("applying fleet config: %w", err)
				}
				scriptPayloads[i] = fleet.ScriptPayload{
					ScriptContents: b,
					Name:           filepath.Base(f),
				}
			}
			if err := c.ApplyNoTeamScripts(scriptPayloads, opts); err != nil {
				return nil, fmt.Errorf("applying custom settings: %w", err)
			}
		}
		if err := c.ApplyAppConfig(specs.AppConfig, opts); err != nil {
			return nil, fmt.Errorf("applying fleet config: %w", err)
		}
		if opts.DryRun {
			logfn("[+] would've applied fleet config\n")
		} else {
			logfn("[+] applied fleet config\n")
		}
	}

	if specs.EnrollSecret != nil {
		if opts.DryRun {
			logfn("[+] would've applied enroll secrets\n")
		} else {
			if err := c.ApplyEnrollSecretSpec(specs.EnrollSecret); err != nil {
				return nil, fmt.Errorf("applying enroll secrets: %w", err)
			}
			logfn("[+] applied enroll secrets\n")
		}
	}

	var teamIDsByName map[string]uint
	if len(specs.Teams) > 0 {
		// extract the teams' custom settings and resolve the files immediately, so
		// that any non-existing file error is found before applying the specs.
		tmMDMSettings := extractTmSpecsMDMCustomSettings(specs.Teams)

		tmFileContents := make(map[string][]fleet.MDMProfileBatchPayload, len(tmMDMSettings))
		for k, paths := range tmMDMSettings {
			fileContents, err := getProfilesContents(baseDir, paths)
			if err != nil {
				return nil, err
			}
			tmFileContents[k] = fileContents
		}

		tmMacSetup := extractTmSpecsMacOSSetup(specs.Teams)
		tmBootstrapPackages := make(map[string]*fleet.MDMAppleBootstrapPackage, len(tmMacSetup))
		tmMacSetupAssistants := make(map[string][]byte, len(tmMacSetup))
		for k, setup := range tmMacSetup {
			if setup.BootstrapPackage.Value != "" {
				bp, err := c.ValidateBootstrapPackageFromURL(setup.BootstrapPackage.Value)
				if err != nil {
					return nil, fmt.Errorf("applying teams: %w", err)
				}
				tmBootstrapPackages[k] = bp
			}
			if setup.MacOSSetupAssistant.Value != "" {
				b, err := c.validateMacOSSetupAssistant(resolveApplyRelativePath(baseDir, setup.MacOSSetupAssistant.Value))
				if err != nil {
					return nil, fmt.Errorf("applying teams: %w", err)
				}
				tmMacSetupAssistants[k] = b
			}
		}

		tmScripts := extractTmSpecsScripts(specs.Teams)
		tmScriptsPayloads := make(map[string][]fleet.ScriptPayload, len(tmScripts))
		for k, paths := range tmScripts {
			files := resolveApplyRelativePaths(baseDir, paths)
			scriptPayloads := make([]fleet.ScriptPayload, len(files))
			for i, f := range files {
				b, err := os.ReadFile(f)
				if err != nil {
					return nil, fmt.Errorf("applying fleet config: %w", err)
				}
				scriptPayloads[i] = fleet.ScriptPayload{
					ScriptContents: b,
					Name:           filepath.Base(f),
				}
			}
			tmScriptsPayloads[k] = scriptPayloads
		}

		// Next, apply the teams specs before saving the profiles, so that any
		// non-existing team gets created.
		var err error
		teamIDsByName, err = c.ApplyTeams(specs.Teams, opts)
		if err != nil {
			return nil, fmt.Errorf("applying teams: %w", err)
		}

		if len(tmFileContents) > 0 {
			for tmName, profs := range tmFileContents {
				teamID, ok := teamIDsByName[tmName]
				if opts.DryRun && (teamID == 0 || !ok) {
					logfn("[+] would've applied MDM profiles for new team %s\n", tmName)
				} else {
					logfn("[+] applying MDM profiles for team %s\n", tmName)
					if err := c.ApplyTeamProfiles(tmName, profs, opts); err != nil {
						return nil, fmt.Errorf("applying custom settings for team %q: %w", tmName, err)
					}
				}
			}
		}
		if len(tmBootstrapPackages)+len(tmMacSetupAssistants) > 0 && !opts.DryRun {
			for tmName, tmID := range teamIDsByName {
				if bp, ok := tmBootstrapPackages[tmName]; ok {
					if err := c.EnsureBootstrapPackage(bp, tmID); err != nil {
						return nil, fmt.Errorf("uploading bootstrap package for team %q: %w", tmName, err)
					}
				}
				if b, ok := tmMacSetupAssistants[tmName]; ok {
					if err := c.uploadMacOSSetupAssistant(b, &tmID, tmMacSetup[tmName].MacOSSetupAssistant.Value); err != nil {
						return nil, fmt.Errorf("uploading macOS setup assistant for team %q: %w", tmName, err)
					}
				}
			}
		}
		if len(tmScriptsPayloads) > 0 {
			for tmName, scripts := range tmScriptsPayloads {
				if err := c.ApplyTeamScripts(tmName, scripts, opts); err != nil {
					return nil, fmt.Errorf("applying scripts for team %q: %w", tmName, err)
				}
			}
		}
		if opts.DryRun {
			logfn("[+] would've applied %d teams\n", len(specs.Teams))
		} else {
			logfn("[+] applied %d teams\n", len(specs.Teams))
		}
	}

	if specs.UsersRoles != nil {
		if opts.DryRun {
			logfn("[!] ignoring user roles, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyUsersRoleSecretSpec(specs.UsersRoles); err != nil {
				return nil, fmt.Errorf("applying user roles: %w", err)
			}
			logfn("[+] applied user roles\n")
		}
	}
	return teamIDsByName, nil
}

func extractAppCfgMacOSSetup(appCfg any) *fleet.MacOSSetup {
	asMap, ok := appCfg.(map[string]interface{})
	if !ok {
		return nil
	}
	mmdm, ok := asMap["mdm"].(map[string]interface{})
	if !ok {
		return nil
	}
	mos, ok := mmdm["macos_setup"].(map[string]interface{})
	if !ok || mos == nil {
		return nil
	}
	bp, _ := mos["bootstrap_package"].(string) // if not a string, bp == ""
	msa, _ := mos["macos_setup_assistant"].(string)
	return &fleet.MacOSSetup{
		BootstrapPackage:    optjson.SetString(bp),
		MacOSSetupAssistant: optjson.SetString(msa),
	}
}

func resolveApplyRelativePath(baseDir, path string) string {
	return resolveApplyRelativePaths(baseDir, []string{path})[0]
}

// resolves the paths to an absolute path relative to the baseDir, which should
// be the path of the YAML file where the relative paths were specified. If the
// path is already absolute, it is left untouched.
func resolveApplyRelativePaths(baseDir string, paths []string) []string {
	if baseDir == "" {
		return paths
	}

	resolved := make([]string, 0, len(paths))
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		resolved = append(resolved, p)
	}
	return resolved
}

func extractAppCfgCustomSettings(appCfg interface{}, platformKey string) []fleet.MDMProfileSpec {
	asMap, ok := appCfg.(map[string]interface{})
	if !ok {
		return nil
	}
	mmdm, ok := asMap["mdm"].(map[string]interface{})
	if !ok {
		return nil
	}
	mos, ok := mmdm[platformKey].(map[string]interface{})
	if !ok || mos == nil {
		return nil
	}

	cs, ok := mos["custom_settings"]
	if !ok {
		// custom settings is not present
		return nil
	}

	csAny, ok := cs.([]interface{})
	if !ok || csAny == nil {
		// return a non-nil, empty slice instead, so the caller knows that the
		// custom_settings key was actually provided.
		return []fleet.MDMProfileSpec{}
	}

	csSpecs := make([]fleet.MDMProfileSpec, 0, len(csAny))
	for _, v := range csAny {
		if m, ok := v.(map[string]interface{}); ok {
			var profSpec fleet.MDMProfileSpec

			// extract the Path field
			if path, ok := m["path"].(string); ok {
				profSpec.Path = path
			}

			// extract the Labels field, labels are cleared if not provided
			if labels, ok := m["labels"].([]interface{}); ok {
				for _, label := range labels {
					if strLabel, ok := label.(string); ok {
						profSpec.Labels = append(profSpec.Labels, strLabel)
					}
				}
			}

			if profSpec.Path != "" {
				csSpecs = append(csSpecs, profSpec)
			}
		} else if m, ok := v.(string); ok { // for backwards compatibility with the old way to define profiles
			if m != "" {
				csSpecs = append(csSpecs, fleet.MDMProfileSpec{Path: m})
			}
		}
	}
	return csSpecs
}

func extractAppCfgMacOSCustomSettings(appCfg interface{}) []fleet.MDMProfileSpec {
	return extractAppCfgCustomSettings(appCfg, "macos_settings")
}

func extractAppCfgWindowsCustomSettings(appCfg interface{}) []fleet.MDMProfileSpec {
	return extractAppCfgCustomSettings(appCfg, "windows_settings")
}

func extractAppCfgScripts(appCfg interface{}) []string {
	asMap, ok := appCfg.(map[string]interface{})
	if !ok {
		return nil
	}

	scripts, ok := asMap["scripts"]
	if !ok {
		// scripts is not present
		return nil
	}

	scriptsAny, ok := scripts.([]interface{})
	if !ok || scriptsAny == nil {
		// return a non-nil, empty slice instead, so the caller knows that the
		// scripts key was actually provided.
		return []string{}
	}

	scriptsStrings := make([]string, 0, len(scriptsAny))
	for _, v := range scriptsAny {
		s, _ := v.(string)
		if s != "" {
			scriptsStrings = append(scriptsStrings, s)
		}
	}
	return scriptsStrings
}

// returns the custom macOS and Windows settings keyed by team name.
func extractTmSpecsMDMCustomSettings(tmSpecs []json.RawMessage) map[string][]fleet.MDMProfileSpec {
	var m map[string][]fleet.MDMProfileSpec
	for _, tm := range tmSpecs {
		var spec struct {
			Name string `json:"name"`
			MDM  struct {
				MacOSSettings struct {
					CustomSettings json.RawMessage `json:"custom_settings"`
				} `json:"macos_settings"`
				WindowsSettings struct {
					CustomSettings json.RawMessage `json:"custom_settings"`
				} `json:"windows_settings"`
			} `json:"mdm"`
		}
		if err := json.Unmarshal(tm, &spec); err != nil {
			// ignore, this will fail in the call to apply team specs
			continue
		}
		spec.Name = norm.NFC.String(spec.Name)
		if spec.Name != "" {
			var macOSSettings []fleet.MDMProfileSpec
			var windowsSettings []fleet.MDMProfileSpec

			// to keep existing bahavior, if any of the custom
			// settings is provided, make the map a non-nil map
			if len(spec.MDM.MacOSSettings.CustomSettings) > 0 ||
				len(spec.MDM.WindowsSettings.CustomSettings) > 0 {
				if m == nil {
					m = make(map[string][]fleet.MDMProfileSpec)
				}
			}

			if len(spec.MDM.MacOSSettings.CustomSettings) > 0 {
				if err := json.Unmarshal(spec.MDM.MacOSSettings.CustomSettings, &macOSSettings); err != nil {
					// ignore, will fail in apply team specs call
					continue
				}
				if macOSSettings == nil {
					// to be consistent with the AppConfig custom settings, set it to an
					// empty slice if the provided custom settings are present but empty.
					macOSSettings = []fleet.MDMProfileSpec{}
				}
			}
			if len(spec.MDM.WindowsSettings.CustomSettings) > 0 {
				if err := json.Unmarshal(spec.MDM.WindowsSettings.CustomSettings, &windowsSettings); err != nil {
					// ignore, will fail in apply team specs call
					continue
				}
				if windowsSettings == nil {
					// to be consistent with the AppConfig custom settings, set it to an
					// empty slice if the provided custom settings are present but empty.
					windowsSettings = []fleet.MDMProfileSpec{}
				}
			}

			// TODO: validate equal names here and API?
			if macOSSettings != nil || windowsSettings != nil {
				m[spec.Name] = append(macOSSettings, windowsSettings...)
			}
		}
	}
	return m
}

func extractTmSpecsScripts(tmSpecs []json.RawMessage) map[string][]string {
	var m map[string][]string
	for _, tm := range tmSpecs {
		var spec struct {
			Name    string          `json:"name"`
			Scripts json.RawMessage `json:"scripts"`
		}
		if err := json.Unmarshal(tm, &spec); err != nil {
			// ignore, this will fail in the call to apply team specs
			continue
		}
		spec.Name = norm.NFC.String(spec.Name)
		if spec.Name != "" && len(spec.Scripts) > 0 {
			if m == nil {
				m = make(map[string][]string)
			}
			var scripts []string
			if err := json.Unmarshal(spec.Scripts, &scripts); err != nil {
				// ignore, will fail in apply team specs call
				continue
			}
			if scripts == nil {
				// to be consistent with the AppConfig custom settings, set it to an
				// empty slice if the provided custom settings are present but empty.
				scripts = []string{}
			}
			m[spec.Name] = scripts
		}
	}
	return m
}

// returns the macos_setup keyed by team name.
func extractTmSpecsMacOSSetup(tmSpecs []json.RawMessage) map[string]*fleet.MacOSSetup {
	var m map[string]*fleet.MacOSSetup
	for _, tm := range tmSpecs {
		var spec struct {
			Name string `json:"name"`
			MDM  struct {
				MacOSSetup fleet.MacOSSetup `json:"macos_setup"`
			} `json:"mdm"`
		}
		if err := json.Unmarshal(tm, &spec); err != nil {
			// ignore, this will fail in the call to apply team specs
			continue
		}
		spec.Name = norm.NFC.String(spec.Name)
		if spec.Name != "" {
			if m == nil {
				m = make(map[string]*fleet.MacOSSetup)
			}
			m[spec.Name] = &spec.MDM.MacOSSetup
		}
	}
	return m
}

// DoGitOps applies the GitOps config to Fleet.
func (c *Client) DoGitOps(
	ctx context.Context,
	config *spec.GitOps,
	baseDir string,
	logf func(format string, args ...interface{}),
	dryRun bool,
	appConfig *fleet.EnrichedAppConfig,
) error {
	var err error
	logFn := func(format string, args ...interface{}) {
		if logf != nil {
			logf(format, args...)
		}
	}
	group := spec.Group{}
	scripts := make([]interface{}, len(config.Controls.Scripts))
	for i, script := range config.Controls.Scripts {
		scripts[i] = *script.Path
	}
	var mdmAppConfig map[string]interface{}
	var team map[string]interface{}
	if config.TeamName == nil {
		group.AppConfig = config.OrgSettings
		group.EnrollSecret = &fleet.EnrollSecretSpec{Secrets: config.OrgSettings["secrets"].([]*fleet.EnrollSecret)}
		group.AppConfig.(map[string]interface{})["agent_options"] = config.AgentOptions
		delete(config.OrgSettings, "secrets") // secrets are applied separately in Client.ApplyGroup

		// Integrations
		var integrations interface{}
		var ok bool
		if integrations, ok = group.AppConfig.(map[string]interface{})["integrations"]; !ok || integrations == nil {
			integrations = map[string]interface{}{}
			group.AppConfig.(map[string]interface{})["integrations"] = integrations
		}
		if jira, ok := integrations.(map[string]interface{})["jira"]; !ok || jira == nil {
			integrations.(map[string]interface{})["jira"] = []interface{}{}
		}
		if zendesk, ok := integrations.(map[string]interface{})["zendesk"]; !ok || zendesk == nil {
			integrations.(map[string]interface{})["zendesk"] = []interface{}{}
		}
		if googleCal, ok := integrations.(map[string]interface{})["google_calendar"]; !ok || googleCal == nil {
			integrations.(map[string]interface{})["google_calendar"] = []interface{}{}
		}

		// Ensure mdm config exists
		mdmConfig, ok := group.AppConfig.(map[string]interface{})["mdm"]
		if !ok || mdmConfig == nil {
			mdmConfig = map[string]interface{}{}
			group.AppConfig.(map[string]interface{})["mdm"] = mdmConfig
		}
		mdmAppConfig, ok = mdmConfig.(map[string]interface{})
		if !ok {
			return errors.New("org_settings.mdm config is not a map")
		}

		// Put in default values for macos_migration
		if config.Controls.MacOSMigration != nil {
			mdmAppConfig["macos_migration"] = config.Controls.MacOSMigration
		} else {
			mdmAppConfig["macos_migration"] = map[string]interface{}{}
		}
		macOSMigration := mdmAppConfig["macos_migration"].(map[string]interface{})
		if enable, ok := macOSMigration["enable"]; !ok || enable == nil {
			macOSMigration["enable"] = false
		}
		// Put in default values for windows_enabled_and_configured
		mdmAppConfig["windows_enabled_and_configured"] = config.Controls.WindowsEnabledAndConfigured
		if config.Controls.WindowsEnabledAndConfigured != nil {
			mdmAppConfig["windows_enabled_and_configured"] = config.Controls.WindowsEnabledAndConfigured
		} else {
			mdmAppConfig["windows_enabled_and_configured"] = false
		}
		group.AppConfig.(map[string]interface{})["scripts"] = scripts
	} else {
		team = make(map[string]interface{})
		team["name"] = *config.TeamName
		team["agent_options"] = config.AgentOptions
		if hostExpirySettings, ok := config.TeamSettings["host_expiry_settings"]; ok {
			team["host_expiry_settings"] = hostExpirySettings
		}
		if features, ok := config.TeamSettings["features"]; ok {
			team["features"] = features
		}
		team["scripts"] = scripts
		team["secrets"] = config.TeamSettings["secrets"]
		team["webhook_settings"] = map[string]interface{}{}
		clearHostStatusWebhook := true
		if webhookSettings, ok := config.TeamSettings["webhook_settings"]; ok {
			if hostStatusWebhook, ok := webhookSettings.(map[string]interface{})["host_status_webhook"]; ok {
				clearHostStatusWebhook = false
				team["webhook_settings"].(map[string]interface{})["host_status_webhook"] = hostStatusWebhook
			}
		}
		if clearHostStatusWebhook {
			// Clear out any existing host_status_webhook settings
			team["webhook_settings"].(map[string]interface{})["host_status_webhook"] = map[string]interface{}{}
		}
		// Integrations
		var integrations interface{}
		var ok bool
		if integrations, ok = config.TeamSettings["integrations"]; !ok || integrations == nil {
			integrations = map[string]interface{}{}
		}
		team["integrations"] = integrations
		_, ok = integrations.(map[string]interface{})
		if !ok {
			return errors.New("team_settings.integrations config is not a map")
		}
		if googleCal, ok := integrations.(map[string]interface{})["google_calendar"]; !ok || googleCal == nil {
			integrations.(map[string]interface{})["google_calendar"] = map[string]interface{}{}
		} else {
			_, ok = googleCal.(map[string]interface{})
			if !ok {
				return errors.New("team_settings.integrations.google_calendar config is not a map")
			}
		}

		team["mdm"] = map[string]interface{}{}
		mdmAppConfig = team["mdm"].(map[string]interface{})
	}
	// Common controls settings between org and team settings
	// Put in default values for macos_settings
	if config.Controls.MacOSSettings != nil {
		mdmAppConfig["macos_settings"] = config.Controls.MacOSSettings
	} else {
		mdmAppConfig["macos_settings"] = map[string]interface{}{}
	}
	macOSSettings := mdmAppConfig["macos_settings"].(map[string]interface{})
	if customSettings, ok := macOSSettings["custom_settings"]; !ok || customSettings == nil {
		macOSSettings["custom_settings"] = []interface{}{}
	}
	// Put in default values for macos_updates
	if config.Controls.MacOSUpdates != nil {
		mdmAppConfig["macos_updates"] = config.Controls.MacOSUpdates
	} else {
		mdmAppConfig["macos_updates"] = map[string]interface{}{}
	}
	macOSUpdates := mdmAppConfig["macos_updates"].(map[string]interface{})
	if minimumVersion, ok := macOSUpdates["minimum_version"]; !ok || minimumVersion == nil {
		macOSUpdates["minimum_version"] = ""
	}
	if deadline, ok := macOSUpdates["deadline"]; !ok || deadline == nil {
		macOSUpdates["deadline"] = ""
	}
	// Put in default values for macos_setup
	if config.Controls.MacOSSetup != nil {
		mdmAppConfig["macos_setup"] = config.Controls.MacOSSetup
	} else {
		mdmAppConfig["macos_setup"] = map[string]interface{}{}
	}
	macOSSetup := mdmAppConfig["macos_setup"].(map[string]interface{})
	if bootstrapPackage, ok := macOSSetup["bootstrap_package"]; !ok || bootstrapPackage == nil {
		macOSSetup["bootstrap_package"] = ""
	}
	if enableEndUserAuthentication, ok := macOSSetup["enable_end_user_authentication"]; !ok || enableEndUserAuthentication == nil {
		macOSSetup["enable_end_user_authentication"] = false
	}
	if macOSSetupAssistant, ok := macOSSetup["macos_setup_assistant"]; !ok || macOSSetupAssistant == nil {
		macOSSetup["macos_setup_assistant"] = ""
	}
	// Put in default values for windows_settings
	if config.Controls.WindowsSettings != nil {
		mdmAppConfig["windows_settings"] = config.Controls.WindowsSettings
	} else {
		mdmAppConfig["windows_settings"] = map[string]interface{}{}
	}
	windowsSettings := mdmAppConfig["windows_settings"].(map[string]interface{})
	if customSettings, ok := windowsSettings["custom_settings"]; !ok || customSettings == nil {
		windowsSettings["custom_settings"] = []interface{}{}
	}
	// Put in default values for windows_updates
	if config.Controls.WindowsUpdates != nil {
		mdmAppConfig["windows_updates"] = config.Controls.WindowsUpdates
	} else {
		mdmAppConfig["windows_updates"] = map[string]interface{}{}
	}
	if appConfig.License.IsPremium() {
		windowsUpdates := mdmAppConfig["windows_updates"].(map[string]interface{})
		if deadlineDays, ok := windowsUpdates["deadline_days"]; !ok || deadlineDays == nil {
			windowsUpdates["deadline_days"] = 0
		}
		if gracePeriodDays, ok := windowsUpdates["grace_period_days"]; !ok || gracePeriodDays == nil {
			windowsUpdates["grace_period_days"] = 0
		}
	}
	// Put in default value for enable_disk_encryption
	if config.Controls.EnableDiskEncryption != nil {
		mdmAppConfig["enable_disk_encryption"] = config.Controls.EnableDiskEncryption
	} else {
		mdmAppConfig["enable_disk_encryption"] = false
	}
	if config.TeamName != nil {
		rawTeam, err := json.Marshal(team)
		if err != nil {
			return fmt.Errorf("error marshalling team spec: %w", err)
		}
		group.Teams = []json.RawMessage{rawTeam}
	}

	// Apply org settings, scripts, enroll secrets, and controls
	teamIDsByName, err := c.ApplyGroup(ctx, &group, baseDir, logf, fleet.ApplySpecOptions{DryRun: dryRun})
	if err != nil {
		return err
	}
	if config.TeamName != nil {
		teamID, ok := teamIDsByName[*config.TeamName]
		if !ok || teamID == 0 {
			if dryRun {
				logFn("[+] would've added any policies/queries to new team %s\n", *config.TeamName)
				return nil
			}
			return fmt.Errorf("team %s not created", *config.TeamName)
		}
		config.TeamID = &teamID
	}

	err = c.doGitOpsPolicies(config, logFn, dryRun)
	if err != nil {
		return err
	}

	err = c.doGitOpsQueries(config, logFn, dryRun)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) doGitOpsPolicies(config *spec.GitOps, logFn func(format string, args ...interface{}), dryRun bool) error {
	// Get the ids and names of current policies to figure out which ones to delete
	policies, err := c.GetPolicies(config.TeamID)
	if err != nil {
		return fmt.Errorf("error getting current policies: %w", err)
	}
	if len(config.Policies) > 0 {
		numPolicies := len(config.Policies)
		logFn("[+] syncing %d policies\n", numPolicies)
		if !dryRun {
			totalApplied := 0
			for i := 0; i < len(config.Policies); i += batchSize {
				end := i + batchSize
				if end > len(config.Policies) {
					end = len(config.Policies)
				}
				totalApplied += end - i
				// Note: We are reusing the spec flow here for adding/updating policies, instead of creating a new flow for GitOps.
				if err := c.ApplyPolicies(config.Policies[i:end]); err != nil {
					return fmt.Errorf("error applying policies: %w", err)
				}
				logFn("[+] synced %d policies\n", totalApplied)
			}
		}
	}
	var policiesToDelete []uint
	for _, oldItem := range policies {
		found := false
		for _, newItem := range config.Policies {
			if oldItem.Name == newItem.Name {
				found = true
				break
			}
		}
		if !found {
			policiesToDelete = append(policiesToDelete, oldItem.ID)
			fmt.Printf("[-] deleting policy %s\n", oldItem.Name)
		}
	}
	if len(policiesToDelete) > 0 {
		logFn("[-] deleting %d policies\n", len(policiesToDelete))
		if !dryRun {
			totalDeleted := 0
			for i := 0; i < len(policiesToDelete); i += batchSize {
				end := i + batchSize
				if end > len(policiesToDelete) {
					end = len(policiesToDelete)
				}
				totalDeleted += end - i
				if err := c.DeletePolicies(config.TeamID, policiesToDelete[i:end]); err != nil {
					return fmt.Errorf("error deleting policies: %w", err)
				}
				logFn("[-] deleted %d policies\n", totalDeleted)
			}
		}
	}
	return nil
}

func (c *Client) doGitOpsQueries(config *spec.GitOps, logFn func(format string, args ...interface{}), dryRun bool) error {
	batchSize := 100
	// Get the ids and names of current queries to figure out which ones to delete
	queries, err := c.GetQueries(config.TeamID, nil)
	if err != nil {
		return fmt.Errorf("error getting current queries: %w", err)
	}
	if len(config.Queries) > 0 {
		numQueries := len(config.Queries)
		logFn("[+] syncing %d queries\n", numQueries)
		if !dryRun {
			appliedCount := 0
			for i := 0; i < len(config.Queries); i += batchSize {
				end := i + batchSize
				if end > len(config.Queries) {
					end = len(config.Queries)
				}
				appliedCount += end - i
				// Note: We are reusing the spec flow here for adding/updating queries, instead of creating a new flow for GitOps.
				if err := c.ApplyQueries(config.Queries[i:end]); err != nil {
					return fmt.Errorf("error applying queries: %w", err)
				}
				logFn("[+] synced %d queries\n", appliedCount)
			}
		}
	}
	var queriesToDelete []uint
	for _, oldQuery := range queries {
		found := false
		for _, newQuery := range config.Queries {
			if oldQuery.Name == newQuery.Name {
				found = true
				break
			}
		}
		if !found {
			queriesToDelete = append(queriesToDelete, oldQuery.ID)
			fmt.Printf("[-] deleting query %s\n", oldQuery.Name)
		}
	}
	if len(queriesToDelete) > 0 {
		logFn("[-] deleting %d queries\n", len(queriesToDelete))
		if !dryRun {
			deleteCount := 0
			for i := 0; i < len(queriesToDelete); i += batchSize {
				end := i + batchSize
				if end > len(queriesToDelete) {
					end = len(queriesToDelete)
				}
				deleteCount += end - i
				if err := c.DeleteQueries(queriesToDelete[i:end]); err != nil {
					return fmt.Errorf("error deleting queries: %w", err)
				}
				logFn("[-] deleted %d queries\n", deleteCount)
			}
		}
	}
	return nil
}
