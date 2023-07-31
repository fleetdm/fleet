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
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
)

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

func (c *Client) CheckMDMEnabled() error {
	return c.runAppConfigChecks(func(ac *fleet.EnrichedAppConfig) error {
		if !ac.MDM.EnabledAndConfigured {
			return errors.New("MDM features aren't turned on. Use `fleetctl generate mdm-apple` and then `fleet serve` with `mdm` configuration to turn on MDM features.")
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
			return errors.New("MDM features aren't turned on. Use `fleetctl generate mdm-apple` and then `fleet serve` with `mdm` configuration to turn on MDM features.")
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

// ApplyGroup applies the given spec group to Fleet.
func (c *Client) ApplyGroup(
	ctx context.Context,
	specs *spec.Group,
	baseDir string,
	logf func(format string, args ...interface{}),
	opts fleet.ApplySpecOptions,
) error {
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
				return fmt.Errorf("applying queries: %w", err)
			}
			logfn("[+] applied %d queries\n", len(specs.Queries))
		}
	}

	if len(specs.Labels) > 0 {
		if opts.DryRun {
			logfn("[!] ignoring labels, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyLabels(specs.Labels); err != nil {
				return fmt.Errorf("applying labels: %w", err)
			}
			logfn("[+] applied %d labels\n", len(specs.Labels))
		}
	}

	if len(specs.Policies) > 0 {
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
				return fmt.Errorf("applying policies: %w", err)
			}
			logfn("[+] applied %d policies\n", len(specs.Policies))
		}
	}

	if len(specs.Packs) > 0 {
		if opts.DryRun {
			logfn("[!] ignoring packs, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyPacks(specs.Packs); err != nil {
				return fmt.Errorf("applying packs: %w", err)
			}
			logfn("[+] applied %d packs\n", len(specs.Packs))
		}
	}

	if specs.AppConfig != nil {
		if macosCustomSettings := extractAppCfgMacOSCustomSettings(specs.AppConfig); macosCustomSettings != nil {
			files := resolveApplyRelativePaths(baseDir, macosCustomSettings)

			fileContents := make([][]byte, len(files))
			for i, f := range files {
				b, err := os.ReadFile(f)
				if err != nil {
					return fmt.Errorf("applying fleet config: %w", err)
				}
				fileContents[i] = b
			}
			if err := c.ApplyNoTeamProfiles(fileContents, opts); err != nil {
				return fmt.Errorf("applying custom settings: %w", err)
			}
		}
		if macosSetup := extractAppCfgMacOSSetup(specs.AppConfig); macosSetup != nil {
			if macosSetup.BootstrapPackage.Value != "" {
				pkg, err := c.ValidateBootstrapPackageFromURL(macosSetup.BootstrapPackage.Value)
				if err != nil {
					return fmt.Errorf("applying fleet config: %w", err)
				}

				if !opts.DryRun {
					if err := c.EnsureBootstrapPackage(pkg, uint(0)); err != nil {
						return fmt.Errorf("applying fleet config: %w", err)
					}
				}
			}
			if macosSetup.MacOSSetupAssistant.Value != "" {
				content, err := c.validateMacOSSetupAssistant(resolveApplyRelativePath(baseDir, macosSetup.MacOSSetupAssistant.Value))
				if err != nil {
					return fmt.Errorf("applying fleet config: %w", err)
				}
				if !opts.DryRun {
					if err := c.uploadMacOSSetupAssistant(content, nil, macosSetup.MacOSSetupAssistant.Value); err != nil {
						return fmt.Errorf("applying fleet config: %w", err)
					}
				}
			}
		}
		if err := c.ApplyAppConfig(specs.AppConfig, opts); err != nil {
			return fmt.Errorf("applying fleet config: %w", err)
		}
		if opts.DryRun {
			logfn("[+] would've applied fleet config\n")
		} else {
			logfn("[+] applied fleet config\n")
		}
	}

	if specs.EnrollSecret != nil {
		if opts.DryRun {
			logfn("[!] ignoring enroll secrets, dry run mode only supported for 'config' and 'team' specs\n")
		} else {
			if err := c.ApplyEnrollSecretSpec(specs.EnrollSecret); err != nil {
				return fmt.Errorf("applying enroll secrets: %w", err)
			}
			logfn("[+] applied enroll secrets\n")
		}
	}

	if len(specs.Teams) > 0 {
		// extract the teams' custom settings and resolve the files immediately, so
		// that any non-existing file error is found before applying the specs.
		tmMacSettings := extractTmSpecsMacOSCustomSettings(specs.Teams)

		tmFileContents := make(map[string][][]byte, len(tmMacSettings))
		for k, paths := range tmMacSettings {
			files := resolveApplyRelativePaths(baseDir, paths)
			fileContents := make([][]byte, len(files))
			for i, f := range files {
				b, err := os.ReadFile(f)
				if err != nil {
					return fmt.Errorf("applying teams: %w", err)
				}
				fileContents[i] = b
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
					return fmt.Errorf("applying teams: %w", err)
				}
				tmBootstrapPackages[k] = bp
			}
			if setup.MacOSSetupAssistant.Value != "" {
				b, err := c.validateMacOSSetupAssistant(resolveApplyRelativePath(baseDir, setup.MacOSSetupAssistant.Value))
				if err != nil {
					return fmt.Errorf("applying teams: %w", err)
				}
				tmMacSetupAssistants[k] = b
			}
		}

		// Next, apply the teams specs before saving the profiles, so that any
		// non-existing team gets created.
		teamIDsByName, err := c.ApplyTeams(specs.Teams, opts)
		if err != nil {
			return fmt.Errorf("applying teams: %w", err)
		}

		if len(tmFileContents) > 0 {
			for tmName, profs := range tmFileContents {
				if err := c.ApplyTeamProfiles(tmName, profs, opts); err != nil {
					return fmt.Errorf("applying custom settings for team %q: %w", tmName, err)
				}
			}
		}
		if len(tmBootstrapPackages)+len(tmMacSetupAssistants) > 0 && !opts.DryRun {
			for tmName, tmID := range teamIDsByName {
				if bp, ok := tmBootstrapPackages[tmName]; ok {
					if err := c.EnsureBootstrapPackage(bp, tmID); err != nil {
						return fmt.Errorf("uploading bootstrap package for team %q: %w", tmName, err)
					}
				}
				if b, ok := tmMacSetupAssistants[tmName]; ok {
					if err := c.uploadMacOSSetupAssistant(b, &tmID, tmMacSetup[tmName].MacOSSetupAssistant.Value); err != nil {
						return fmt.Errorf("uploading macOS setup assistant for team %q: %w", tmName, err)
					}
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
				return fmt.Errorf("applying user roles: %w", err)
			}
			logfn("[+] applied user roles\n")
		}
	}
	return nil
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

func extractAppCfgMacOSCustomSettings(appCfg interface{}) []string {
	asMap, ok := appCfg.(map[string]interface{})
	if !ok {
		return nil
	}
	mmdm, ok := asMap["mdm"].(map[string]interface{})
	if !ok {
		return nil
	}
	mos, ok := mmdm["macos_settings"].(map[string]interface{})
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
		return []string{}
	}

	csStrings := make([]string, 0, len(csAny))
	for _, v := range csAny {
		s, _ := v.(string)
		if s != "" {
			csStrings = append(csStrings, s)
		}
	}
	return csStrings
}

// returns the custom settings keyed by team name.
func extractTmSpecsMacOSCustomSettings(tmSpecs []json.RawMessage) map[string][]string {
	var m map[string][]string
	for _, tm := range tmSpecs {
		var spec struct {
			Name string `json:"name"`
			MDM  struct {
				MacOSSettings struct {
					CustomSettings json.RawMessage `json:"custom_settings"`
				} `json:"macos_settings"`
			} `json:"mdm"`
		}
		if err := json.Unmarshal(tm, &spec); err != nil {
			// ignore, this will fail in the call to apply team specs
			continue
		}
		if spec.Name != "" && len(spec.MDM.MacOSSettings.CustomSettings) > 0 {
			if m == nil {
				m = make(map[string][]string)
			}
			var cs []string
			if err := json.Unmarshal(spec.MDM.MacOSSettings.CustomSettings, &cs); err != nil {
				// ignore, will fail in apply team specs call
				continue
			}
			if cs == nil {
				// to be consistent with the AppConfig custom settings, set it to an
				// empty slice if the provided custom settings are present but empty.
				cs = []string{}
			}
			m[spec.Name] = cs
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
		if spec.Name != "" {
			if m == nil {
				m = make(map[string]*fleet.MacOSSetup)
			}
			m[spec.Name] = &spec.MDM.MacOSSetup
		}
	}
	return m
}
