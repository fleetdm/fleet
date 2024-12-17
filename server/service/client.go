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
	"gopkg.in/yaml.v2"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
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

func (c *Client) AuthenticatedDoCustomHeaders(verb, path, rawQuery string, params interface{}, customHeaders map[string]string) (*http.Response, error) {
	if c.token == "" {
		return nil, errors.New("authentication token is empty")
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.token),
	}

	for key, value := range customHeaders {
		headers[key] = value
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
func getProfilesContents(baseDir string, macProfiles []fleet.MDMProfileSpec, windowsProfiles []fleet.MDMProfileSpec, expandEnv bool) ([]fleet.MDMProfileBatchPayload, error) {
	// map to check for duplicate names across all profiles
	extByName := make(map[string]string, len(macProfiles))
	result := make([]fleet.MDMProfileBatchPayload, 0, len(macProfiles))

	// iterate over the profiles for each platform
	for platform, profiles := range map[string][]fleet.MDMProfileSpec{
		"macos":   macProfiles,
		"windows": windowsProfiles,
	} {
		for _, profile := range profiles {
			filePath := resolveApplyRelativePath(baseDir, profile.Path)
			fileContents, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("applying custom settings: %w", err)
			}

			if expandEnv {
				// Secrets are handled earlier in the flow when config files are initially read
				fileContents, err = spec.ExpandEnvBytesIgnoreSecrets(fileContents)
				if err != nil {
					return nil, fmt.Errorf("expanding environment on file %q: %w", profile.Path, err)
				}
			}

			ext := filepath.Ext(filePath)
			// by default, use the file name (for macOS mobileconfig profiles, we'll switch to
			// their PayloadDisplayName when we parse the profile below)
			name := strings.TrimSuffix(filepath.Base(filePath), ext)
			// for validation errors, we want to include the platform and file name in the error message
			prefixErrMsg := fmt.Sprintf("Couldn't edit %s_settings.custom_settings (%s%s)", platform, name, ext)

			// validate macOS profiles
			if platform == "macos" {
				switch ext {
				case ".mobileconfig", ".xml": // allowing .xml for backwards compatibility
					mc, err := fleet.NewMDMAppleConfigProfile(fileContents, nil)
					if err != nil {
						errForMsg := errors.Unwrap(err)
						if errForMsg == nil {
							errForMsg = err
						}
						return nil, fmt.Errorf("%s: %w", prefixErrMsg, errForMsg)
					}
					name = strings.TrimSpace(mc.Name)
				case ".json":
					if mdm.GetRawProfilePlatform(fileContents) != "darwin" {
						return nil, fmt.Errorf("%s: %s", prefixErrMsg, "Declaration profiles should include valid JSON.")
					}
				default:
					return nil, fmt.Errorf("%s: %s", prefixErrMsg, "macOS configuration profiles must be .mobileconfig or .json files.")
				}
			}

			// validate windows profiles
			if platform == "windows" {
				switch ext {
				case ".xml":
					if mdm.GetRawProfilePlatform(fileContents) != "windows" {
						return nil, fmt.Errorf("%s: %s", prefixErrMsg, "Windows configuration profiles can only have <Replace> or <Add> top level elements")
					}
				default:
					return nil, fmt.Errorf("%s: %s", prefixErrMsg, "Windows configuration profiles must be .xml files.")
				}
			}

			// check for duplicate names across all profiles
			if e, isDuplicate := extByName[name]; isDuplicate {
				return nil, errors.New(fmtDuplicateNameErrMsg(name, e, ext))
			}
			extByName[name] = ext

			result = append(result, fleet.MDMProfileBatchPayload{
				Name:             name,
				Contents:         fileContents,
				Labels:           profile.Labels,
				LabelsIncludeAll: profile.LabelsIncludeAll,
				LabelsIncludeAny: profile.LabelsIncludeAny,
				LabelsExcludeAny: profile.LabelsExcludeAny,
			})

		}
	}
	return result, nil
}

// fileContent is used to store the name of a file and its content.
type fileContent struct {
	Filename string
	Content  []byte
}

// TODO: as confirmed by Noah and Marko on Slack:
//
//	> from Noah: "We want to support existing features w/ fleetctl apply for
//	> backwards compatibility GitOps but we don’t need to add new features."
//
// We should deprecate ApplyGroup and use it only for `fleetctl apply` (and
// its current minimal use in `preview`), and have a distinct implementation
// that is `gitops`-only, because both uses have subtle differences in
// behaviour that make it hard to reuse a single implementation (e.g. a missing
// key in gitops means "remove what is absent" while in apply it means "leave
// as-is").
//
// For now I'm just passing a "gitops" bool for a quick fix, but we should
// properly plan that separation and refactor so that gitops can be
// significantly cleaned up and simplified going forward.

// ApplyGroup applies the given spec group to Fleet.
func (c *Client) ApplyGroup(
	ctx context.Context,
	viaGitOps bool,
	specs *spec.Group,
	baseDir string,
	logf func(format string, args ...interface{}),
	appconfig *fleet.EnrichedAppConfig,
	opts fleet.ApplyClientSpecOptions,
	teamsSoftwareInstallers map[string][]fleet.SoftwarePackageResponse,
	teamsScripts map[string][]fleet.ScriptResponse,
) (map[string]uint, map[string][]fleet.SoftwarePackageResponse, map[string][]fleet.ScriptResponse, error) {
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
				return nil, nil, nil, fmt.Errorf("applying queries: %w", err)
			}
			logfn("[+] applied %d queries\n", len(specs.Queries))
		}
	}

	if len(specs.Labels) > 0 {
		if opts.DryRun {
			logfn("[!] ignoring labels, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyLabels(specs.Labels); err != nil {
				return nil, nil, nil, fmt.Errorf("applying labels: %w", err)
			}
			logfn("[+] applied %d labels\n", len(specs.Labels))
		}
	}

	if len(specs.Packs) > 0 {
		if opts.DryRun {
			logfn("[!] ignoring packs, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyPacks(specs.Packs); err != nil {
				return nil, nil, nil, fmt.Errorf("applying packs: %w", err)
			}
			logfn("[+] applied %d packs\n", len(specs.Packs))
		}
	}

	if specs.AppConfig != nil {
		windowsCustomSettings := extractAppCfgWindowsCustomSettings(specs.AppConfig)
		macosCustomSettings := extractAppCfgMacOSCustomSettings(specs.AppConfig)

		// if there is no custom setting but the windows and mac settings are
		// non-nil, this means that we want to clear the existing custom settings,
		// so we still go on with calling the batch-apply endpoint.
		//
		// TODO(mna): shouldn't that be an || instead of && ? I.e. if there are no
		// custom settings but windows is present and empty (but mac is absent),
		// shouldn't that clear the windows ones?
		if (windowsCustomSettings != nil && macosCustomSettings != nil) || len(windowsCustomSettings)+len(macosCustomSettings) > 0 {
			fileContents, err := getProfilesContents(baseDir, macosCustomSettings, windowsCustomSettings, opts.ExpandEnvConfigProfiles)
			if err != nil {
				return nil, nil, nil, err
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
			if err := c.ApplyNoTeamProfiles(fileContents, opts.ApplySpecOptions, assumeEnabled); err != nil {
				return nil, nil, nil, fmt.Errorf("applying custom settings: %w", err)
			}
		}
		if macosSetup := extractAppCfgMacOSSetup(specs.AppConfig); macosSetup != nil {
			if macosSetup.BootstrapPackage.Value != "" {
				pkg, err := c.ValidateBootstrapPackageFromURL(macosSetup.BootstrapPackage.Value)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying fleet config: %w", err)
				}

				if !opts.DryRun {
					if err := c.EnsureBootstrapPackage(pkg, uint(0)); err != nil {
						return nil, nil, nil, fmt.Errorf("applying fleet config: %w", err)
					}
				}
			}
			if macosSetup.MacOSSetupAssistant.Value != "" {
				content, err := c.validateMacOSSetupAssistant(resolveApplyRelativePath(baseDir, macosSetup.MacOSSetupAssistant.Value))
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying fleet config: %w", err)
				}
				if !opts.DryRun {
					if err := c.uploadMacOSSetupAssistant(content, nil, macosSetup.MacOSSetupAssistant.Value); err != nil {
						return nil, nil, nil, fmt.Errorf("applying fleet config: %w", err)
					}
				}
			}
		}
		if scripts := extractAppCfgScripts(specs.AppConfig); scripts != nil {
			scriptPayloads := make([]fleet.ScriptPayload, len(scripts))
			for i, f := range scripts {
				b, err := os.ReadFile(f)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying no-team scripts: %w", err)
				}
				scriptPayloads[i] = fleet.ScriptPayload{
					ScriptContents: b,
					Name:           filepath.Base(f),
				}
			}
			noTeamScripts, err := c.ApplyNoTeamScripts(scriptPayloads, opts.ApplySpecOptions)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("applying no-team scripts: %w", err)
			}
			teamsScripts["No team"] = noTeamScripts
		}

		rules, err := extractAppCfgYaraRules(specs.AppConfig)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("applying yara rules: %w", err)
		}
		if rules != nil {
			rulePayloads := make([]fleet.YaraRule, len(rules))
			for i, f := range rules {
				path := resolveApplyRelativePath(baseDir, f.Path)
				b, err := os.ReadFile(path)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying yara rules: %w", err)
				}
				rulePayloads[i] = fleet.YaraRule{
					Contents: string(b),
					Name:     filepath.Base(f.Path),
				}
			}
			specs.AppConfig.(map[string]interface{})["yara_rules"] = rulePayloads
		}

		if err := c.ApplyAppConfig(specs.AppConfig, opts.ApplySpecOptions); err != nil {
			return nil, nil, nil, fmt.Errorf("applying fleet config: %w", err)
		}
		if opts.DryRun {
			logfn("[+] would've applied fleet config\n")
		} else {
			logfn("[+] applied fleet config\n")
		}
	}

	if specs.EnrollSecret != nil {
		if err := c.ApplyEnrollSecretSpec(specs.EnrollSecret, opts.ApplySpecOptions); err != nil {
			return nil, nil, nil, fmt.Errorf("applying enroll secrets: %w", err)
		}
		if opts.DryRun {
			logfn("[+] would've applied enroll secrets\n")
		} else {
			logfn("[+] applied enroll secrets\n")
		}
	}

	var teamIDsByName map[string]uint
	if len(specs.Teams) > 0 {
		// extract the teams' custom settings and resolve the files immediately, so
		// that any non-existing file error is found before applying the specs.
		tmMDMSettings := extractTmSpecsMDMCustomSettings(specs.Teams)

		tmFileContents := make(map[string][]fleet.MDMProfileBatchPayload, len(tmMDMSettings))
		for k, profileSpecs := range tmMDMSettings {
			fileContents, err := getProfilesContents(baseDir, profileSpecs.macos, profileSpecs.windows, opts.ExpandEnvConfigProfiles)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("Team %s: %w", k, err) // TODO: consider adding team name to improve error messages generally for other parts of the config because multiple team configs can be processed at once
			}
			tmFileContents[k] = fileContents
		}

		tmMacSetup := extractTmSpecsMacOSSetup(specs.Teams)
		tmBootstrapPackages := make(map[string]*fleet.MDMAppleBootstrapPackage, len(tmMacSetup))
		tmMacSetupAssistants := make(map[string][]byte, len(tmMacSetup))

		// those are gitops-only features
		tmMacSetupScript := make(map[string]fileContent, len(tmMacSetup))
		tmMacSetupSoftware := make(map[string][]*fleet.MacOSSetupSoftware, len(tmMacSetup))
		// this is a set of software packages or VPP apps that are configured as
		// install_during_setup, by team. This is a gitops-only setting, so it will
		// only be filled when called via this command.
		tmSoftwareMacOSSetup := make(map[string]map[fleet.MacOSSetupSoftware]struct{}, len(tmMacSetup))

		for k, setup := range tmMacSetup {
			if setup.BootstrapPackage.Value != "" {
				bp, err := c.ValidateBootstrapPackageFromURL(setup.BootstrapPackage.Value)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying teams: %w", err)
				}
				tmBootstrapPackages[k] = bp
			}
			if setup.MacOSSetupAssistant.Value != "" {
				b, err := c.validateMacOSSetupAssistant(resolveApplyRelativePath(baseDir, setup.MacOSSetupAssistant.Value))
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying teams: %w", err)
				}
				tmMacSetupAssistants[k] = b
			}
			if setup.Script.Value != "" {
				b, err := c.validateMacOSSetupScript(resolveApplyRelativePath(baseDir, setup.Script.Value))
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying teams: %w", err)
				}
				tmMacSetupScript[k] = fileContent{Filename: filepath.Base(setup.Script.Value), Content: b}
			}
			if viaGitOps {
				m, err := extractTeamOrNoTeamMacOSSetupSoftware(baseDir, setup.Software.Value)
				if err != nil {
					return nil, nil, nil, err
				}
				tmSoftwareMacOSSetup[k] = m
				tmMacSetupSoftware[k] = setup.Software.Value
			}
		}

		tmScripts := extractTmSpecsScripts(specs.Teams)
		tmScriptsPayloads := make(map[string][]fleet.ScriptPayload, len(tmScripts))
		for k, paths := range tmScripts {
			scriptPayloads := make([]fleet.ScriptPayload, len(paths))
			for i, f := range paths {
				b, err := os.ReadFile(f)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying fleet config: %w", err)
				}
				scriptPayloads[i] = fleet.ScriptPayload{
					ScriptContents: b,
					Name:           filepath.Base(f),
				}
			}
			tmScriptsPayloads[k] = scriptPayloads
		}

		tmSoftwarePackages := extractTmSpecsSoftwarePackages(specs.Teams)
		tmSoftwarePackagesPayloads := make(map[string][]fleet.SoftwareInstallerPayload, len(tmSoftwarePackages))
		tmSoftwarePackageByPath := make(map[string]map[string]fleet.SoftwarePackageSpec, len(tmSoftwarePackages))
		for tmName, software := range tmSoftwarePackages {
			installDuringSetupKeys := tmSoftwareMacOSSetup[tmName]
			softwarePayloads, err := buildSoftwarePackagesPayload(software, installDuringSetupKeys)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("applying software installers for team %q: %w", tmName, err)
			}
			tmSoftwarePackagesPayloads[tmName] = softwarePayloads
			for _, swSpec := range software {
				if swSpec.ReferencedYamlPath != "" {
					// can be referenced by macos_setup.software.package_path
					if tmSoftwarePackageByPath[tmName] == nil {
						tmSoftwarePackageByPath[tmName] = make(map[string]fleet.SoftwarePackageSpec, len(software))
					}
					tmSoftwarePackageByPath[tmName][swSpec.ReferencedYamlPath] = swSpec
				}
			}
		}

		tmSoftwareApps := extractTmSpecsSoftwareApps(specs.Teams)
		tmSoftwareAppsPayloads := make(map[string][]fleet.VPPBatchPayload)
		tmSoftwareAppsByAppID := make(map[string]map[string]fleet.TeamSpecAppStoreApp, len(tmSoftwareApps))
		for tmName, apps := range tmSoftwareApps {
			installDuringSetupKeys := tmSoftwareMacOSSetup[tmName]
			appPayloads := make([]fleet.VPPBatchPayload, 0, len(apps))
			for _, app := range apps {
				var installDuringSetup *bool
				if installDuringSetupKeys != nil {
					_, ok := installDuringSetupKeys[fleet.MacOSSetupSoftware{AppStoreID: app.AppStoreID}]
					installDuringSetup = &ok
				}
				appPayloads = append(appPayloads, fleet.VPPBatchPayload{
					AppStoreID:         app.AppStoreID,
					SelfService:        app.SelfService,
					InstallDuringSetup: installDuringSetup,
				})
				// can be referenced by macos_setup.software.app_store_id
				if tmSoftwareAppsByAppID[tmName] == nil {
					tmSoftwareAppsByAppID[tmName] = make(map[string]fleet.TeamSpecAppStoreApp, len(apps))
				}
				tmSoftwareAppsByAppID[tmName][app.AppStoreID] = app
			}
			tmSoftwareAppsPayloads[tmName] = appPayloads
		}

		// if macos_setup.software has some values, they must exist in the software
		// packages or vpp apps.
		for tmName, setupSw := range tmMacSetupSoftware {
			if err := validateTeamOrNoTeamMacOSSetupSoftware(tmName, setupSw, tmSoftwarePackageByPath[tmName], tmSoftwareAppsByAppID[tmName]); err != nil {
				return nil, nil, nil, err
			}
		}

		// Next, apply the teams specs before saving the profiles, so that any
		// non-existing team gets created.
		var err error
		teamOpts := fleet.ApplyTeamSpecOptions{
			ApplySpecOptions:  opts.ApplySpecOptions,
			DryRunAssumptions: specs.TeamsDryRunAssumptions,
		}
		// In dry-run, the team names returned are the old team names (when team name is modified via gitops)
		teamIDsByName, err = c.ApplyTeams(specs.Teams, teamOpts)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("applying teams: %w", err)
		}

		// When using GitOps, the team name could change, so we need to check for that
		getTeamName := func(teamName string) string {
			return teamName
		}
		if len(specs.Teams) == 1 && len(teamIDsByName) == 1 {
			for key := range teamIDsByName {
				if key != extractTeamName(specs.Teams[0]) {
					getTeamName = func(teamName string) string {
						return key
					}
				}
			}
		}

		if len(tmFileContents) > 0 {
			for tmName, profs := range tmFileContents {
				// For non-dry run, currentTeamName and tmName are the same
				currentTeamName := getTeamName(tmName)
				teamID, ok := teamIDsByName[currentTeamName]
				if opts.DryRun && (teamID == 0 || !ok) {
					logfn("[+] would've applied MDM profiles for new team %s\n", tmName)
				} else {
					logfn("[+] applying MDM profiles for team %s\n", tmName)
					if err := c.ApplyTeamProfiles(currentTeamName, profs, teamOpts); err != nil {
						return nil, nil, nil, fmt.Errorf("applying custom settings for team %q: %w", tmName, err)
					}
				}
			}
		}
		if len(tmBootstrapPackages)+len(tmMacSetupAssistants) > 0 && !opts.DryRun {
			for tmName, tmID := range teamIDsByName {
				if bp, ok := tmBootstrapPackages[tmName]; ok {
					if err := c.EnsureBootstrapPackage(bp, tmID); err != nil {
						return nil, nil, nil, fmt.Errorf("uploading bootstrap package for team %q: %w", tmName, err)
					}
				}
				if b, ok := tmMacSetupAssistants[tmName]; ok {
					if err := c.uploadMacOSSetupAssistant(b, &tmID, tmMacSetup[tmName].MacOSSetupAssistant.Value); err != nil {
						if strings.Contains(err.Error(), "Couldn't upload") {
							// Then the error should look something like this:
							// "Couldn't upload. CONFIG_NAME_INVALID"
							// We want the part after the period (this is the error name from Apple)
							// to render a more helpful error message.
							parts := strings.Split(err.Error(), ".")
							if len(parts) < 2 {
								return nil, nil, nil, fmt.Errorf("unexpected error while uploading macOS setup assistant for team %q: %w", tmName, err)
							}
							return nil, nil, nil, fmt.Errorf("Couldn't edit macos_setup_assistant. Response from Apple: %s. Learn more at %s", strings.Trim(parts[1], " "), "https://fleetdm.com/learn-more-about/dep-profile")
						}
						return nil, nil, nil, fmt.Errorf("uploading macOS setup assistant for team %q: %w", tmName, err)
					}
				}
			}
		}
		if viaGitOps && !opts.DryRun {
			for tmName, tmID := range teamIDsByName {
				if fc, ok := tmMacSetupScript[tmName]; ok {
					if err := c.uploadMacOSSetupScript(fc.Filename, fc.Content, &tmID); err != nil {
						return nil, nil, nil, fmt.Errorf("uploading setup experience script for team %q: %w", tmName, err)
					}
				} else {
					if err := c.deleteMacOSSetupScript(&tmID); err != nil {
						return nil, nil, nil, fmt.Errorf("deleting setup experience script for team %q: %w", tmName, err)
					}
				}
			}
		}
		if len(tmScriptsPayloads) > 0 {
			for tmName, scripts := range tmScriptsPayloads {
				// For non-dry run, currentTeamName and tmName are the same
				currentTeamName := getTeamName(tmName)
				scriptResponses, err := c.ApplyTeamScripts(currentTeamName, scripts, opts.ApplySpecOptions)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying scripts for team %q: %w", tmName, err)
				}
				teamsScripts[tmName] = scriptResponses
			}
		}
		if len(tmSoftwarePackagesPayloads) > 0 {
			for tmName, software := range tmSoftwarePackagesPayloads {
				// For non-dry run, currentTeamName and tmName are the same
				currentTeamName := getTeamName(tmName)
				logfn("[+] applying %d software packages for team %s\n", len(software), tmName)
				installers, err := c.ApplyTeamSoftwareInstallers(currentTeamName, software, opts.ApplySpecOptions)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("applying software installers for team %q: %w", tmName, err)
				}
				teamsSoftwareInstallers[tmName] = installers
			}
		}
		if len(tmSoftwareAppsPayloads) > 0 {
			for tmName, apps := range tmSoftwareAppsPayloads {
				// For non-dry run, currentTeamName and tmName are the same
				currentTeamName := getTeamName(tmName)
				logfn("[+] applying %d app store apps for team %s\n", len(apps), tmName)
				if err := c.ApplyTeamAppStoreAppsAssociation(currentTeamName, apps, opts.ApplySpecOptions); err != nil {
					return nil, nil, nil, fmt.Errorf("applying app store apps for team: %q: %w", tmName, err)
				}
			}
		}
		if opts.DryRun {
			logfn("[+] would've applied %d teams\n", len(specs.Teams))
		} else {
			logfn("[+] applied %d teams\n", len(specs.Teams))
		}
	}

	// Policies can reference software installers thus they are applied at this point.
	if len(specs.Policies) > 0 {
		// Policy names must be unique, return error if duplicate policy names are found
		if policyName := fleet.FirstDuplicatePolicySpecName(specs.Policies); policyName != "" {
			return nil, nil, nil, fmt.Errorf(
				"applying policies: policy names must be unique. Please correct policy %q and try again.", policyName,
			)
		}
		if opts.DryRun {
			logfn("[!] ignoring policies, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			// If set, override the team in all the policies.
			if opts.TeamForPolicies != "" {
				for _, policySpec := range specs.Policies {
					policySpec.Team = opts.TeamForPolicies
				}
			}
			if err := c.ApplyPolicies(specs.Policies); err != nil {
				return nil, nil, nil, fmt.Errorf("applying policies: %w", err)
			}
			logfn("[+] applied %d policies\n", len(specs.Policies))
		}
	}

	if specs.UsersRoles != nil {
		if opts.DryRun {
			logfn("[!] ignoring user roles, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyUsersRoleSecretSpec(specs.UsersRoles); err != nil {
				return nil, nil, nil, fmt.Errorf("applying user roles: %w", err)
			}
			logfn("[+] applied user roles\n")
		}
	}

	return teamIDsByName, teamsSoftwareInstallers, teamsScripts, nil
}

func extractTeamOrNoTeamMacOSSetupSoftware(baseDir string, software []*fleet.MacOSSetupSoftware) (map[fleet.MacOSSetupSoftware]struct{}, error) {
	m := make(map[fleet.MacOSSetupSoftware]struct{}, len(software))
	for _, sw := range software {
		if sw.AppStoreID != "" && sw.PackagePath != "" {
			return nil, errors.New("applying teams: only one of app_store_id or package_path can be set")
		}
		if sw.PackagePath != "" {
			sw.PackagePath = resolveApplyRelativePath(baseDir, sw.PackagePath)
		}
		m[*sw] = struct{}{}
	}
	return m, nil
}

func validateTeamOrNoTeamMacOSSetupSoftware(teamName string, macOSSetupSoftware []*fleet.MacOSSetupSoftware, packagesByPath map[string]fleet.SoftwarePackageSpec, vppAppsByAppID map[string]fleet.TeamSpecAppStoreApp) error {
	// if macos_setup.software has some values, they must exist in the software
	// packages or vpp apps.
	for _, ssw := range macOSSetupSoftware {
		var valid bool
		if ssw.AppStoreID != "" {
			// check that it exists in the team's Apps
			_, valid = vppAppsByAppID[ssw.AppStoreID]
		} else if ssw.PackagePath != "" {
			// check that it exists in the team's Software installers (PackagePath is
			// already resolved to abs dir)
			_, valid = packagesByPath[ssw.PackagePath]
		}
		if !valid {
			label := ssw.AppStoreID
			if label == "" {
				label = ssw.PackagePath
			}
			return fmt.Errorf("applying macOS setup experience software for team %q: software %q does not exist for that team", teamName, label)
		}
	}
	return nil
}

func buildSoftwarePackagesPayload(specs []fleet.SoftwarePackageSpec, installDuringSetupKeys map[fleet.MacOSSetupSoftware]struct{}) ([]fleet.SoftwareInstallerPayload, error) {
	softwarePayloads := make([]fleet.SoftwareInstallerPayload, len(specs))
	for i, si := range specs {
		var qc string
		var err error
		if si.PreInstallQuery.Path != "" {
			queryFile := si.PreInstallQuery.Path
			rawSpec, err := os.ReadFile(queryFile)
			if err != nil {
				return nil, fmt.Errorf("reading pre-install query: %w", err)
			}

			rawSpecExpanded, err := spec.ExpandEnvBytes(rawSpec)
			if err != nil {
				return nil, fmt.Errorf("Couldn't exit software (%s). Unable to expand environment variable in YAML file %s: %w", si.URL, queryFile, err)
			}

			var top any

			if err := yaml.Unmarshal(rawSpecExpanded, &top); err != nil {
				return nil, fmt.Errorf("Couldn't exit software (%s). Unable to expand environment variable in YAML file %s: %w", si.URL, queryFile, err)
			}

			if _, ok := top.(map[any]any); ok {
				// Old apply format
				group, err := spec.GroupFromBytes(rawSpecExpanded)
				if err != nil {
					return nil, fmt.Errorf("Couldn't edit software (%s). Unable to parse pre-install apply format query YAML file %s: %w", si.URL, queryFile, err)
				}

				if len(group.Queries) > 1 {
					return nil, fmt.Errorf("Couldn't edit software (%s). Pre-install query YAML file %s should have only one query.", si.URL, queryFile)
				}

				if len(group.Queries) == 0 {
					return nil, fmt.Errorf("Couldn't edit software (%s). Pre-install query YAML file %s doesn't have a query defined.", si.URL, queryFile)
				}

				qc = group.Queries[0].Query
			} else {
				// Gitops format
				var querySpecs []fleet.QuerySpec
				if err := yaml.Unmarshal(rawSpecExpanded, &querySpecs); err != nil {
					return nil, fmt.Errorf("Couldn't edit software (%s). Unable to parse pre-install query YAML file %s: %w", si.URL, queryFile, err)
				}

				if len(querySpecs) > 1 {
					return nil, fmt.Errorf("Couldn't edit software (%s). Pre-install query YAML file %s should have only one query.", si.URL, queryFile)
				}

				if len(querySpecs) == 0 {
					return nil, fmt.Errorf("Couldn't edit software (%s). Pre-install query YAML file %s doesn't have a query defined.", si.URL, queryFile)
				}

				qc = querySpecs[0].Query
			}
		}

		var ic []byte
		if si.InstallScript.Path != "" {
			installScriptFile := si.InstallScript.Path
			ic, err = os.ReadFile(installScriptFile)
			if err != nil {
				return nil, fmt.Errorf("Couldn't edit software (%s). Unable to read install script file %s: %w", si.URL, si.InstallScript.Path, err)
			}
		}

		var pc []byte
		if si.PostInstallScript.Path != "" {
			postInstallScriptFile := si.PostInstallScript.Path
			pc, err = os.ReadFile(postInstallScriptFile)
			if err != nil {
				return nil, fmt.Errorf("Couldn't edit software (%s). Unable to read post-install script file %s: %w", si.URL, si.PostInstallScript.Path, err)
			}
		}

		var us []byte
		if si.UninstallScript.Path != "" {
			uninstallScriptFile := si.UninstallScript.Path
			us, err = os.ReadFile(uninstallScriptFile)
			if err != nil {
				return nil, fmt.Errorf("Couldn't edit software (%s). Unable to read uninstall script file %s: %w", si.URL,
					si.UninstallScript.Path, err)
			}
		}

		var installDuringSetup *bool
		if installDuringSetupKeys != nil {
			_, ok := installDuringSetupKeys[fleet.MacOSSetupSoftware{PackagePath: si.ReferencedYamlPath}]
			installDuringSetup = &ok
		}
		softwarePayloads[i] = fleet.SoftwareInstallerPayload{
			URL:                si.URL,
			SelfService:        si.SelfService,
			PreInstallQuery:    qc,
			InstallScript:      string(ic),
			PostInstallScript:  string(pc),
			UninstallScript:    string(us),
			InstallDuringSetup: installDuringSetup,
			LabelsIncludeAny:   si.LabelsIncludeAny,
			LabelsExcludeAny:   si.LabelsExcludeAny,
		}
	}

	return softwarePayloads, nil
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

	extractLabelField := func(parentMap map[string]interface{}, fieldName string) []string {
		var ret []string
		if labels, ok := parentMap[fieldName].([]interface{}); ok {
			for _, label := range labels {
				if strLabel, ok := label.(string); ok {
					ret = append(ret, strLabel)
				}
			}
		}
		return ret
	}

	csSpecs := make([]fleet.MDMProfileSpec, 0, len(csAny))
	for _, v := range csAny {
		if m, ok := v.(map[string]interface{}); ok {
			var profSpec fleet.MDMProfileSpec

			// extract the Path field
			if path, ok := m["path"].(string); ok {
				profSpec.Path = path
			}

			// at this stage we extract and return all supported label fields, the
			// validations are done later on in the Fleet API endpoint.
			profSpec.Labels = extractLabelField(m, "labels")
			profSpec.LabelsIncludeAll = extractLabelField(m, "labels_include_all")
			profSpec.LabelsIncludeAny = extractLabelField(m, "labels_include_any")
			profSpec.LabelsExcludeAny = extractLabelField(m, "labels_exclude_any")

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

func extractAppCfgYaraRules(appCfg interface{}) ([]fleet.YaraRuleSpec, error) {
	asMap, ok := appCfg.(map[string]interface{})
	if !ok {
		return nil, errors.New("extract yara rules: app config is not a map")
	}

	rules, ok := asMap["yara_rules"]
	if !ok {
		// yara_rules is not present. Return an empty slice so that the value is cleared.
		return []fleet.YaraRuleSpec{}, nil
	}

	rulesAny, ok := rules.([]interface{})
	if !ok || rulesAny == nil {
		// If nil, return an empty slice so the value will be cleared.
		return []fleet.YaraRuleSpec{}, nil
	}

	ruleSpecs := make([]fleet.YaraRuleSpec, 0, len(rulesAny))
	for _, v := range rulesAny {
		smap, ok := v.(map[string]interface{})
		if !ok {
			return nil, errors.New("extract yara rules: rule entry is not a map")
		}

		pathEntry, ok := smap["path"]
		if !ok {
			return nil, errors.New("extract yara rules: rule entry missing path")
		}

		path, ok := pathEntry.(string)
		if !ok {
			return nil, errors.New("extract yara rules: rule entry path is not string")
		}

		ruleSpecs = append(ruleSpecs, fleet.YaraRuleSpec{Path: path})
	}
	return ruleSpecs, nil
}

type profileSpecsByPlatform struct {
	macos   []fleet.MDMProfileSpec
	windows []fleet.MDMProfileSpec
}

func extractTeamName(tmSpec json.RawMessage) string {
	var s struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(tmSpec, &s); err != nil {
		return ""
	}
	return norm.NFC.String(s.Name)
}

// returns the custom macOS and Windows settings keyed by team name.
func extractTmSpecsMDMCustomSettings(tmSpecs []json.RawMessage) map[string]profileSpecsByPlatform {
	var m map[string]profileSpecsByPlatform
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
					m = make(map[string]profileSpecsByPlatform)
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
			var result profileSpecsByPlatform
			if macOSSettings != nil {
				result.macos = macOSSettings
			}
			if windowsSettings != nil {
				result.windows = windowsSettings
			}

			if macOSSettings != nil || windowsSettings != nil {
				m[spec.Name] = result
			}
		}
	}
	return m
}

func extractTmSpecsSoftwarePackages(tmSpecs []json.RawMessage) map[string][]fleet.SoftwarePackageSpec {
	var m map[string][]fleet.SoftwarePackageSpec
	for _, tm := range tmSpecs {
		var spec struct {
			Name     string          `json:"name"`
			Software json.RawMessage `json:"software"`
		}
		if err := json.Unmarshal(tm, &spec); err != nil {
			// ignore, this will fail in the call to apply team specs
			continue
		}
		spec.Name = norm.NFC.String(spec.Name)
		if spec.Name != "" && len(spec.Software) > 0 {
			if m == nil {
				m = make(map[string][]fleet.SoftwarePackageSpec)
			}
			var software fleet.SoftwareSpec
			var packages []fleet.SoftwarePackageSpec
			if err := json.Unmarshal(spec.Software, &software); err != nil {
				// ignore, will fail in apply team specs call
				continue
			}
			if !software.Packages.Valid {
				// to be consistent with the AppConfig custom settings, set it to an
				// empty slice if the provided custom settings are present but empty.
				packages = []fleet.SoftwarePackageSpec{}
			} else {
				packages = software.Packages.Value
			}
			m[spec.Name] = packages
		}
	}
	return m
}

func extractTmSpecsSoftwareApps(tmSpecs []json.RawMessage) map[string][]fleet.TeamSpecAppStoreApp {
	var m map[string][]fleet.TeamSpecAppStoreApp
	for _, tm := range tmSpecs {
		var spec struct {
			Name     string          `json:"name"`
			Software json.RawMessage `json:"software"`
		}
		if err := json.Unmarshal(tm, &spec); err != nil {
			// ignore, this will fail in the call to apply team specs
			continue
		}
		spec.Name = norm.NFC.String(spec.Name)
		if spec.Name != "" && len(spec.Software) > 0 {
			if m == nil {
				m = make(map[string][]fleet.TeamSpecAppStoreApp)
			}
			var software fleet.SoftwareSpec
			var apps []fleet.TeamSpecAppStoreApp
			if err := json.Unmarshal(spec.Software, &software); err != nil {
				// ignore, will fail in apply team specs call
				continue
			}
			if !software.AppStoreApps.Valid {
				// to be consistent with the AppConfig custom settings, set it to an
				// empty slice if the provided custom settings are present but empty.
				apps = []fleet.TeamSpecAppStoreApp{}
			} else {
				apps = software.AppStoreApps.Value
			}
			m[spec.Name] = apps
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

func (c *Client) SaveEnvSecrets(alreadySaved map[string]string, toSave map[string]string, dryRun bool) error {
	if len(toSave) == 0 {
		return nil
	}
	// Figure out which secrets need to be saved
	var secretsToSave []fleet.SecretVariable
	for key, value := range toSave {
		if _, ok := alreadySaved[key]; !ok {
			secretsToSave = append(secretsToSave, fleet.SecretVariable{Name: key, Value: value})
			alreadySaved[key] = value
		}
	}
	if len(secretsToSave) == 0 {
		return nil
	}
	return c.SaveSecretVariables(secretsToSave, dryRun)
}

// DoGitOps applies the GitOps config to Fleet.
func (c *Client) DoGitOps(
	ctx context.Context,
	config *spec.GitOps,
	fullFilename string,
	logf func(format string, args ...interface{}),
	dryRun bool,
	teamDryRunAssumptions *fleet.TeamSpecsDryRunAssumptions,
	appConfig *fleet.EnrichedAppConfig,
	// pass-by-ref to build lists
	teamsSoftwareInstallers map[string][]fleet.SoftwarePackageResponse,
	teamsScripts map[string][]fleet.ScriptResponse,
) (*fleet.TeamSpecsDryRunAssumptions, error) {
	baseDir := filepath.Dir(fullFilename)
	filename := filepath.Base(fullFilename)
	var teamAssumptions *fleet.TeamSpecsDryRunAssumptions
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
		integrations, ok = integrations.(map[string]interface{})
		if !ok {
			return nil, errors.New("org_settings.integrations config is not a map")
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
		if ndesSCEPProxy, ok := integrations.(map[string]interface{})["ndes_scep_proxy"]; !ok || ndesSCEPProxy == nil {
			// Per backend patterns.md, best practice is to clear a JSON config field with `null`
			integrations.(map[string]interface{})["ndes_scep_proxy"] = nil
		} else {
			if _, ok = ndesSCEPProxy.(map[string]interface{}); !ok {
				return nil, errors.New("org_settings.integrations.ndes_scep_proxy config is not a map")
			}
		}

		// Ensure mdm config exists
		mdmConfig, ok := group.AppConfig.(map[string]interface{})["mdm"]
		if !ok || mdmConfig == nil {
			mdmConfig = map[string]interface{}{}
			group.AppConfig.(map[string]interface{})["mdm"] = mdmConfig
		}
		mdmAppConfig, ok = mdmConfig.(map[string]interface{})
		if !ok {
			return nil, errors.New("org_settings.mdm config is not a map")
		}

		// Put in default value for volume_purchasing_program to clear the configuration if it's not set.
		if v, ok := mdmAppConfig["volume_purchasing_program"]; !ok || v == nil {
			mdmAppConfig["volume_purchasing_program"] = []interface{}{}
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
		// Put in default values for windows_migration_enabled
		mdmAppConfig["windows_migration_enabled"] = config.Controls.WindowsMigrationEnabled
		if config.Controls.WindowsMigrationEnabled == nil {
			mdmAppConfig["windows_migration_enabled"] = false
		}
		if windowsEnabledAndConfiguredAssumption, ok := mdmAppConfig["windows_enabled_and_configured"].(bool); ok {
			teamAssumptions = &fleet.TeamSpecsDryRunAssumptions{
				WindowsEnabledAndConfigured: optjson.SetBool(windowsEnabledAndConfiguredAssumption),
			}
		}
		group.AppConfig.(map[string]interface{})["scripts"] = scripts
	} else if !config.IsNoTeam() {
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
		team["software"] = map[string]any{}
		team["software"].(map[string]any)["app_store_apps"] = config.Software.AppStoreApps
		team["software"].(map[string]any)["packages"] = config.Software.Packages
		team["secrets"] = config.TeamSettings["secrets"]
		team["webhook_settings"] = map[string]interface{}{}
		clearHostStatusWebhook := true
		if webhookSettings, ok := config.TeamSettings["webhook_settings"]; ok {
			if _, ok := webhookSettings.(map[string]interface{}); ok {
				if hostStatusWebhook, ok := webhookSettings.(map[string]interface{})["host_status_webhook"]; ok {
					clearHostStatusWebhook = false
					team["webhook_settings"].(map[string]interface{})["host_status_webhook"] = hostStatusWebhook
				}
			} else if webhookSettings != nil {
				return nil, fmt.Errorf("team_settings.webhook_settings config is not a map but a %T", webhookSettings)
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
			return nil, errors.New("team_settings.integrations config is not a map")
		}
		if googleCal, ok := integrations.(map[string]interface{})["google_calendar"]; !ok || googleCal == nil {
			integrations.(map[string]interface{})["google_calendar"] = map[string]interface{}{}
		} else {
			_, ok = googleCal.(map[string]interface{})
			if !ok {
				return nil, errors.New("team_settings.integrations.google_calendar config is not a map")
			}
		}

		team["mdm"] = map[string]interface{}{}
		mdmAppConfig = team["mdm"].(map[string]interface{})
	}

	if !config.IsNoTeam() {
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
		// Put in default values for ios_updates
		if config.Controls.IOSUpdates != nil {
			mdmAppConfig["ios_updates"] = config.Controls.IOSUpdates
		} else {
			mdmAppConfig["ios_updates"] = map[string]interface{}{}
		}
		iOSUpdates := mdmAppConfig["ios_updates"].(map[string]interface{})
		if minimumVersion, ok := iOSUpdates["minimum_version"]; !ok || minimumVersion == nil {
			iOSUpdates["minimum_version"] = ""
		}
		if deadline, ok := iOSUpdates["deadline"]; !ok || deadline == nil {
			iOSUpdates["deadline"] = ""
		}
		// Put in default values for ipados_updates
		if config.Controls.IPadOSUpdates != nil {
			mdmAppConfig["ipados_updates"] = config.Controls.IPadOSUpdates
		} else {
			mdmAppConfig["ipados_updates"] = map[string]interface{}{}
		}
		iPadOSUpdates := mdmAppConfig["ipados_updates"].(map[string]interface{})
		if minimumVersion, ok := iPadOSUpdates["minimum_version"]; !ok || minimumVersion == nil {
			iPadOSUpdates["minimum_version"] = ""
		}
		if deadline, ok := iPadOSUpdates["deadline"]; !ok || deadline == nil {
			iPadOSUpdates["deadline"] = ""
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
				windowsUpdates["deadline_days"] = nil
			}
			if gracePeriodDays, ok := windowsUpdates["grace_period_days"]; !ok || gracePeriodDays == nil {
				windowsUpdates["grace_period_days"] = nil
			}
		}
		// Put in default value for enable_disk_encryption
		if config.Controls.EnableDiskEncryption != nil {
			mdmAppConfig["enable_disk_encryption"] = config.Controls.EnableDiskEncryption
		} else {
			mdmAppConfig["enable_disk_encryption"] = false
		}

		if config.TeamName != nil {
			team["gitops_filename"] = filename
			rawTeam, err := json.Marshal(team)
			if err != nil {
				return nil, fmt.Errorf("error marshalling team spec: %w", err)
			}
			group.Teams = []json.RawMessage{rawTeam}
			group.TeamsDryRunAssumptions = teamDryRunAssumptions
		}
	}

	// Apply org settings, scripts, enroll secrets, team entities (software, scripts, etc.), and controls.
	teamIDsByName, teamsSoftwareInstallers, teamsScripts, err := c.ApplyGroup(ctx, true, &group, baseDir, logf, appConfig, fleet.ApplyClientSpecOptions{
		ApplySpecOptions: fleet.ApplySpecOptions{
			DryRun: dryRun,
		},
		ExpandEnvConfigProfiles: true,
	}, teamsSoftwareInstallers, teamsScripts)
	if err != nil {
		return nil, err
	}

	var teamSoftwareInstallers []fleet.SoftwarePackageResponse
	var teamScripts []fleet.ScriptResponse
	if config.TeamName != nil {
		if !config.IsNoTeam() {
			if len(teamIDsByName) != 1 {
				return nil, fmt.Errorf("expected 1 team spec to be applied, got %d", len(teamIDsByName))
			}
			teamID, ok := teamIDsByName[*config.TeamName]
			if ok && teamID == 0 {
				if dryRun {
					logFn("[+] would've added any policies/queries to new team %s\n", *config.TeamName)
					return nil, nil
				}
				return nil, fmt.Errorf("team %s not created", *config.TeamName)
			}
			for _, teamID = range teamIDsByName {
				config.TeamID = &teamID
			}
			teamSoftwareInstallers = teamsSoftwareInstallers[*config.TeamName]
			teamScripts = teamsScripts[*config.TeamName]
		} else {
			noTeamSoftwareInstallers, err := c.doGitOpsNoTeamSoftware(config, baseDir, appConfig, logFn, dryRun)
			if err != nil {
				return nil, err
			}
			teamSoftwareInstallers = noTeamSoftwareInstallers
			teamScripts = teamsScripts["No team"]
		}
	}

	err = c.doGitOpsPolicies(config, teamSoftwareInstallers, teamScripts, logFn, dryRun)
	if err != nil {
		return nil, err
	}

	// We currently don't support queries for "No team" thus
	// we just do GitOps for queries for global and team files.
	if !config.IsNoTeam() {
		err = c.doGitOpsQueries(config, logFn, dryRun)
		if err != nil {
			return nil, err
		}
	}

	return teamAssumptions, nil
}

func (c *Client) doGitOpsNoTeamSoftware(
	config *spec.GitOps,
	baseDir string,
	appconfig *fleet.EnrichedAppConfig,
	logFn func(format string, args ...interface{}),
	dryRun bool,
) ([]fleet.SoftwarePackageResponse, error) {
	if !config.IsNoTeam() || appconfig == nil || !appconfig.License.IsPremium() {
		return nil, nil
	}

	// marshaling dance to get the macos_setup data - config.Controls.MacOSSetup
	// is of type any and contains a generic map[string]any. By
	// marshal-unmarshaling it into a properly typed struct, we avoid having to
	// do a bunch of error-prone and unmaintainable type-assertions to walk down
	// the untyped map.
	b, err := json.Marshal(config.Controls.MacOSSetup)
	if err != nil {
		return nil, fmt.Errorf("applying software installers: json-encode controls.macos_setup: %w", err)
	}
	var macOSSetup fleet.MacOSSetup
	if err := json.Unmarshal(b, &macOSSetup); err != nil {
		return nil, fmt.Errorf("applying software installers: json-decode controls.macos_setup: %w", err)
	}

	// load the no-team macos_setup.script if any
	var macosSetupScript *fileContent
	if macOSSetup.Script.Value != "" {
		b, err := c.validateMacOSSetupScript(resolveApplyRelativePath(baseDir, macOSSetup.Script.Value))
		if err != nil {
			return nil, fmt.Errorf("applying no team macos_setup.script: %w", err)
		}
		macosSetupScript = &fileContent{Filename: filepath.Base(macOSSetup.Script.Value), Content: b}
	}

	noTeamSoftwareMacOSSetup, err := extractTeamOrNoTeamMacOSSetupSoftware(baseDir, macOSSetup.Software.Value)
	if err != nil {
		return nil, err
	}

	var softwareInstallers []fleet.SoftwarePackageResponse

	packages := make([]fleet.SoftwarePackageSpec, 0, len(config.Software.Packages))
	packagesByPath := make(map[string]fleet.SoftwarePackageSpec, len(config.Software.Packages))
	for _, software := range config.Software.Packages {
		if software != nil {
			packages = append(packages, *software)
			if software.ReferencedYamlPath != "" {
				// can be referenced by macos_setup.software
				packagesByPath[software.ReferencedYamlPath] = *software
			}
		}
	}

	var appsPayload []fleet.VPPBatchPayload
	appsByAppID := make(map[string]fleet.TeamSpecAppStoreApp, len(config.Software.AppStoreApps))
	for _, vppApp := range config.Software.AppStoreApps {
		if vppApp != nil {
			// can be referenced by macos_setup.software
			appsByAppID[vppApp.AppStoreID] = *vppApp

			_, installDuringSetup := noTeamSoftwareMacOSSetup[fleet.MacOSSetupSoftware{AppStoreID: vppApp.AppStoreID}]
			appsPayload = append(appsPayload, fleet.VPPBatchPayload{
				AppStoreID:         vppApp.AppStoreID,
				SelfService:        vppApp.SelfService,
				InstallDuringSetup: &installDuringSetup,
			})
		}
	}

	if err := validateTeamOrNoTeamMacOSSetupSoftware(*config.TeamName, macOSSetup.Software.Value, packagesByPath, appsByAppID); err != nil {
		return nil, err
	}
	swPkgPayload, err := buildSoftwarePackagesPayload(packages, noTeamSoftwareMacOSSetup)
	if err != nil {
		return nil, fmt.Errorf("applying software installers: %w", err)
	}

	if macosSetupScript != nil {
		logFn("[+] applying macos setup experience script for 'No team'\n")
		if err := c.uploadMacOSSetupScript(macosSetupScript.Filename, macosSetupScript.Content, nil); err != nil {
			return nil, fmt.Errorf("uploading setup experience script for No team: %w", err)
		}
	} else if err := c.deleteMacOSSetupScript(nil); err != nil {
		return nil, fmt.Errorf("deleting setup experience script for No team: %w", err)
	}

	logFn("[+] applying %d software packages for 'No team'\n", len(swPkgPayload))
	softwareInstallers, err = c.ApplyNoTeamSoftwareInstallers(swPkgPayload, fleet.ApplySpecOptions{DryRun: dryRun})
	if err != nil {
		return nil, fmt.Errorf("applying software installers: %w", err)
	}
	logFn("[+] applying %d app store apps for 'No team'\n", len(appsPayload))
	if err := c.ApplyNoTeamAppStoreAppsAssociation(appsPayload, fleet.ApplySpecOptions{DryRun: dryRun}); err != nil {
		return nil, fmt.Errorf("applying app store apps: %w", err)
	}

	if dryRun {
		logFn("[+] would've applied 'No Team' software packages\n")
	} else {
		logFn("[+] applied 'No Team' software packages\n")
	}
	return softwareInstallers, nil
}

func (c *Client) doGitOpsPolicies(config *spec.GitOps, teamSoftwareInstallers []fleet.SoftwarePackageResponse, teamScripts []fleet.ScriptResponse, logFn func(format string, args ...interface{}), dryRun bool) error {
	var teamID *uint // Global policies (nil)
	switch {
	case config.TeamID != nil: // Team policies
		teamID = config.TeamID
	case config.IsNoTeam(): // "No team" policies
		teamID = ptr.Uint(0)
	}
	if teamID != nil {
		// Get software titles of packages for the team.
		softwareTitleURLs := make(map[string]uint)
		for _, softwareInstaller := range teamSoftwareInstallers {
			if softwareInstaller.TitleID == nil {
				// Should not happen, but to not panic we just log a warning.
				logFn("[!] software installer without title id: team_id=%d, url=%s\n", *teamID, softwareInstaller.URL)
				continue
			}
			if softwareInstaller.URL == "" {
				// Should not happen because we previously applied packages via gitops, but to not panic we just log a warning.
				logFn("[!] software installer without url: team_id=%d, title_id=%d\n", *teamID, *softwareInstaller.TitleID)
				continue
			}
			softwareTitleURLs[softwareInstaller.URL] = *softwareInstaller.TitleID
		}
		for i := range config.Policies {
			config.Policies[i].SoftwareTitleID = ptr.Uint(0) // 0 unsets the installer

			if config.Policies[i].InstallSoftware == nil {
				continue
			}
			softwareTitleID, ok := softwareTitleURLs[config.Policies[i].InstallSoftwareURL]
			if !ok {
				// Should not happen because software packages are uploaded first.
				if !dryRun {
					logFn("[!] software URL without software title id: %s\n", config.Policies[i].InstallSoftwareURL)
				}
				continue
			}
			config.Policies[i].SoftwareTitleID = &softwareTitleID
		}

		// Get scripts for the team.
		scriptIDsByName := make(map[string]uint)
		for _, script := range teamScripts {
			scriptIDsByName[script.Name] = script.ID
		}
		for i := range config.Policies {
			config.Policies[i].ScriptID = ptr.Uint(0) // 0 unsets the script

			if config.Policies[i].RunScript == nil {
				continue
			}
			scriptID, ok := scriptIDsByName[*config.Policies[i].RunScriptName]
			if !ok {
				if !dryRun { // this shouldn't happen
					logFn("[!] reference to an unknown script: %s\n", *config.Policies[i].RunScriptName)
				}
				continue
			}
			config.Policies[i].ScriptID = &scriptID
		}
	}

	// Get the ids and names of current policies to figure out which ones to delete
	policies, err := c.GetPolicies(teamID)
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
				policiesToApply := config.Policies[i:end]
				policiesSpec := make([]*fleet.PolicySpec, len(policiesToApply))
				for i := range policiesToApply {
					policiesSpec[i] = &policiesToApply[i].PolicySpec
				}
				if err := c.ApplyPolicies(policiesSpec); err != nil {
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
			if !dryRun {
				logFn("[-] deleting policy %s\n", oldItem.Name)
			} else {
				logFn("[-] would've deleted policy %s\n", oldItem.Name)
			}
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
				var teamID *uint
				switch {
				case config.TeamID != nil: // Team policies
					teamID = config.TeamID
				case config.IsNoTeam(): // No team policies
					teamID = ptr.Uint(fleet.PolicyNoTeamID)
				default: // Global policies
					teamID = nil
				}
				if err := c.DeletePolicies(teamID, policiesToDelete[i:end]); err != nil {
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
			if !dryRun {
				fmt.Printf("[-] deleting query %s\n", oldQuery.Name)
			} else {
				fmt.Printf("[-] would've deleted query %s\n", oldQuery.Name)
			}
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

func (c *Client) GetGitOpsSecrets(
	config *spec.GitOps,
) []string {
	if config.TeamName == nil {
		orgSecrets, ok := config.OrgSettings["secrets"]
		if ok {
			secrets, ok := orgSecrets.([]*fleet.EnrollSecret)
			if ok {
				secretValues := make([]string, 0, len(secrets))
				for _, secret := range secrets {
					secretValues = append(secretValues, secret.Secret)
				}
				return secretValues
			}
		}
	} else {
		teamSecrets, ok := config.TeamSettings["secrets"]
		if ok {
			secrets, ok := teamSecrets.([]*fleet.EnrollSecret)
			if ok {
				secretValues := make([]string, 0, len(secrets))
				for _, secret := range secrets {
					secretValues = append(secretValues, secret.Secret)
				}
				return secretValues
			}
		}
	}
	return nil
}
