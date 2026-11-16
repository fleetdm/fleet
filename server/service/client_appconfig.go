package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/fleetdm/fleet/v4/server/version"
)

// ApplyAppConfig sends the application config to be applied to the Fleet instance.
func (c *Client) ApplyAppConfig(payload interface{}, opts fleet.ApplySpecOptions) error {
	verb, path := "PATCH", "/api/latest/fleet/config"
	var responseBody appConfigResponse
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data, err = endpointer.RewriteOldToNewKeys(data, endpointer.ExtractAliasRules(fleet.AppConfig{}))
	if err != nil {
		return err
	}
	return c.authenticatedRequestWithQuery(data, verb, path, &responseBody, opts.RawQuery())
}

// ApplyNoTeamProfiles sends the list of profiles to be applied for the hosts
// in no team.
func (c *Client) ApplyNoTeamProfiles(profiles []fleet.MDMProfileBatchPayload, opts fleet.ApplySpecOptions, assumptions *fleet.TeamSpecsDryRunAssumptions) error {
	verb, path := "POST", "/api/latest/fleet/mdm/profiles/batch"
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return err
	}
	if assumptions != nil {
		if assumptions.WindowsEnabledAndConfigured.Valid && assumptions.WindowsEnabledAndConfigured.Value {
			query.Add("assume_enabled", "true")
		}
		if assumptions.EnableProfileValidation.Valid {
			query.Add("assume_profile_validation", strconv.FormatBool(assumptions.EnableProfileValidation.Value))
		}
	}
	err = c.authenticatedRequestWithQuery(map[string]any{"profiles": profiles}, verb, path, nil, query.Encode())
	return rewrapProfileBatchNameErr(err, profiles)
}

// GetAppConfig fetches the application config from the server API
func (c *Client) GetAppConfig() (*fleet.EnrichedAppConfig, error) {
	verb, path := "GET", "/api/latest/fleet/config"
	var responseBody fleet.EnrichedAppConfig
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return &responseBody, err
}

// GetEnrollSecretSpec fetches the enroll secrets stored on the server
func (c *Client) GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/enroll_secret"
	var responseBody getEnrollSecretSpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// ApplyEnrollSecretSpec applies the enroll secrets.
func (c *Client) ApplyEnrollSecretSpec(spec *fleet.EnrollSecretSpec, opts fleet.ApplySpecOptions) error {
	req := applyEnrollSecretSpecRequest{Spec: spec}
	verb, path := "POST", "/api/latest/fleet/spec/enroll_secret"
	var responseBody applyEnrollSecretSpecResponse
	return c.authenticatedRequestWithQuery(req, verb, path, &responseBody, opts.RawQuery())
}

func (c *Client) Version() (*version.Info, error) {
	verb, path := "GET", "/api/latest/fleet/version"
	var responseBody versionResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Info, err
}

type orgLogoAction struct {
	mode       fleet.OrgLogoMode
	uploadPath string
}

// planAndStripOrgLogos plans the PUT /logo uploads that run after the
// AppConfig PATCH and strips the gitops-only `path` keys from the PATCH
// payload (they have no matching field on fleet.OrgInfo). When a `path`
// is given, the URL keys for that mode are stripped too so the follow-up
// PUT is the sole writer.
//
// URL changes (set or clear) ride on the PATCH. The server deletes any
// orphan blob whose URL was just overwritten with an external or empty
// value, so the client doesn't need to orchestrate a separate DELETE.
func (c *Client) planAndStripOrgLogos(
	orgSettings map[string]any,
	baseDir string,
	dryRun bool,
	logFn func(format string, args ...any),
) ([]orgLogoAction, error) {
	orgInfo, _ := orgSettings["org_info"].(map[string]any)
	if orgInfo == nil {
		return nil, nil
	}

	type modeSpec struct {
		mode    fleet.OrgLogoMode
		pathKey string
		urlKey  string
	}
	specs := []modeSpec{
		{fleet.OrgLogoModeLight, "org_logo_path_light_mode", "org_logo_url_light_mode"},
		{fleet.OrgLogoModeDark, "org_logo_path_dark_mode", "org_logo_url_dark_mode"},
	}

	var actions []orgLogoAction
	for _, s := range specs {
		_, pathPresent := orgInfo[s.pathKey]
		_, urlPresent := orgInfo[s.urlKey]
		if !pathPresent && !urlPresent {
			orgInfo[s.urlKey] = ""
			continue
		}

		yamlPath, _ := orgInfo[s.pathKey].(string)
		yamlURL, _ := orgInfo[s.urlKey].(string)
		if yamlPath != "" && yamlURL != "" {
			return nil, fmt.Errorf(
				"org_settings.org_info: cannot specify both '%s' and '%s' for %s mode",
				s.pathKey, s.urlKey, s.mode,
			)
		}

		switch {
		case yamlPath != "":
			absPath := resolveApplyRelativePath(baseDir, yamlPath)
			if err := validateOrgLogoFile(absPath); err != nil {
				return nil, fmt.Errorf("org logo (%s): %w", s.mode, err)
			}
			actions = append(actions, orgLogoAction{mode: s.mode, uploadPath: absPath})
			// Strip the URL key for this mode: PUT will set the served URL
			// after the PATCH, so we must keep PATCH from writing it.
			delete(orgInfo, s.pathKey)
			delete(orgInfo, s.urlKey)
			if dryRun {
				logFn("[+] would upload org logo (%s) from %s\n", s.mode, yamlPath)
			}
		case yamlURL != "":
			delete(orgInfo, s.pathKey)
		default:
			delete(orgInfo, s.pathKey)
			orgInfo[s.urlKey] = ""
		}
	}
	return actions, nil
}

// doGitOpsOrgLogos executes the upload actions planned by
// planAndStripOrgLogos. Runs after the AppConfig PATCH so a PATCH failure
// leaves the logo store untouched.
func (c *Client) doGitOpsOrgLogos(
	actions []orgLogoAction, dryRun bool, logFn func(format string, args ...any),
) error {
	for _, a := range actions {
		if dryRun {
			continue // already logged at planning time
		}
		if err := c.UploadOrgLogo(a.mode, a.uploadPath); err != nil {
			return fmt.Errorf("uploading org logo (%s): %w", a.mode, err)
		}
		logFn("[+] applied org logo (%s) from %s\n", a.mode, a.uploadPath)
	}
	return nil
}

// validateOrgLogoFile reads the file at path and runs the canonical
// fleet.ValidateOrgLogoBytes check on its contents, so a YAML referencing
// an invalid image fails fast at gitops apply time rather than mid-PATCH.
// The LimitReader caps the read at the file-size cap so a mis-pointed
// huge file doesn't get slurped into memory before being rejected.
func validateOrgLogoFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening logo file %q: %w", path, err)
	}
	defer f.Close()
	body, err := io.ReadAll(io.LimitReader(f, fleet.OrgLogoMaxFileSize+1))
	if err != nil {
		return fmt.Errorf("reading logo file %q: %w", path, err)
	}
	if err := fleet.ValidateOrgLogoBytes(body); err != nil {
		return fmt.Errorf("logo file at %q: %w", path, err)
	}
	return nil
}

// UploadOrgLogo uploads the file at logoPath as the org logo for the given
// mode (light or dark) via PUT /api/latest/fleet/logo. The endpoint is
// multipart/form-data with a single "logo" field. Server-side validation
// rejects files larger than 100KB or that aren't PNG/JPEG/WebP.
func (c *Client) UploadOrgLogo(mode fleet.OrgLogoMode, logoPath string) error {
	verb, path := "PUT", "/api/latest/fleet/logo"

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("logo", filepath.Base(logoPath))
	if err != nil {
		return err
	}
	file, err := os.Open(logoPath)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(fw, file); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing writer: %w", err)
	}

	resp, err := c.doContextWithBodyAndHeaders(context.Background(), verb, path,
		fmt.Sprintf("mode=%s", mode),
		b.Bytes(),
		map[string]string{
			"Content-Type":  w.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", c.token),
		})
	if err != nil {
		return fmt.Errorf("do multipart request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("uploading org logo (%s): %s: %s", mode, resp.Status, string(body))
	}
	return nil
}

// DeleteOrgLogo clears the stored org logo for the given mode (light, dark, or
// all) via DELETE /api/latest/fleet/logo. The endpoint is idempotent — deleting
// an absent logo is a no-op server-side.
func (c *Client) DeleteOrgLogo(mode fleet.OrgLogoMode) error {
	verb, path := "DELETE", "/api/latest/fleet/logo"
	var responseBody deleteOrgLogoResponse
	return c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, fmt.Sprintf("mode=%s", mode))
}

// GetOrgLogoContent fetches the stored org logo bytes for the given mode
// (light or dark) via GET /api/latest/fleet/logo. Returns the bytes and the
// detected Content-Type. The endpoint returns 404 when no logo is stored for
// the requested mode; callers should treat that as "no logo present".
func (c *Client) GetOrgLogoContent(mode fleet.OrgLogoMode) (body []byte, contentType string, err error) {
	verb, path := "GET", "/api/latest/fleet/logo"
	resp, err := c.AuthenticatedDo(verb, path, fmt.Sprintf("mode=%s", mode), nil)
	if err != nil {
		return nil, "", fmt.Errorf("fetching org logo (%s): %w", mode, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, "", &notFoundErr{Msg: fmt.Sprintf("no org logo stored for %s mode", mode)}
	}
	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("fetching org logo (%s): %s: %s", mode, resp.Status, string(errBody))
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading org logo (%s) body: %w", mode, err)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

// gitOpsHighRiskOrgSettings are the org_settings sections whose reset is called
// out to the user, because it loses a secret, unlocks the UI, or deletes data.
var gitOpsHighRiskOrgSettings = []string{"smtp_settings", "gitops", "server_settings"}

// gitOpsOrgSettingsDefaults returns the org_settings a GitOps run falls back to
// for anything the YAML leaves out. Only sections that the PATCH would
// otherwise merge onto the stored config are listed here.
func gitOpsOrgSettingsDefaults() (map[string]any, error) {
	var cfg fleet.AppConfig
	cfg.ApplyDefaultsForNewInstalls()

	raw, err := json.Marshal(map[string]any{
		"host_expiry_settings":     cfg.HostExpirySettings,
		"activity_expiry_settings": cfg.ActivityExpirySettings,
		"smtp_settings":            cfg.SMTPSettings,
		"server_settings":          cfg.ServerSettings,
		"fleet_desktop":            cfg.FleetDesktop,
		"webhook_settings":         cfg.WebhookSettings,
		"org_info":                 map[string]any{"contact_url": fleet.DefaultOrgInfoContactURL},
		"gitops":                   map[string]any{"gitops_mode_enabled": false, "repository_url": ""},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal org_settings defaults: %w", err)
	}
	var defaults map[string]any
	if err := json.Unmarshal(raw, &defaults); err != nil {
		return nil, fmt.Errorf("unmarshal org_settings defaults: %w", err)
	}

	serverSettings := defaults["server_settings"].(map[string]any)
	// The server URL has no default to fall back to.
	delete(serverSettings, "server_url")
	// Tagged omitempty, so it has to be sent explicitly to be cleared.
	serverSettings["debug_host_ids"] = []any{}
	// Computed by the server from the test email.
	delete(defaults["smtp_settings"].(map[string]any), "configured")
	// Policy IDs come from the policies themselves, not from this section.
	delete(defaults["webhook_settings"].(map[string]any)["failing_policies_webhook"].(map[string]any), "policy_ids")

	return defaults, nil
}

// fillGitOpsDefaults sets every key of defaults that dst omits or leaves blank,
// descending into nested sections. It returns the dotted paths it filled, using
// the old (json tag) key names. A key supplied under its renamed alias counts
// as present, since sending both names is rejected as a conflict.
func fillGitOpsDefaults(dst, defaults map[string]any, aliases map[string]string, prefix string) []string {
	var filled []string
	for key, def := range defaults {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		dstKey := key
		if _, ok := dst[key]; !ok {
			if alias, hasAlias := aliases[key]; hasAlias {
				if _, ok := dst[alias]; ok {
					dstKey = alias
				}
			}
		}

		defMap, defIsMap := def.(map[string]any)
		cur := dst[dstKey]
		if cur == nil {
			dst[dstKey] = def
			if defIsMap {
				filled = append(filled, leafPaths(defMap, path)...)
			} else {
				filled = append(filled, path)
			}
			continue
		}
		if curMap, ok := cur.(map[string]any); ok && defIsMap {
			filled = append(filled, fillGitOpsDefaults(curMap, defMap, aliases, path)...)
		}
	}
	return filled
}

func leafPaths(m map[string]any, prefix string) []string {
	var paths []string
	for key, val := range m {
		path := prefix + "." + key
		if sub, ok := val.(map[string]any); ok {
			paths = append(paths, leafPaths(sub, path)...)
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func lookupPath(m map[string]any, path string) (any, bool) {
	var cur any = m
	for part := range strings.SplitSeq(path, ".") {
		asMap, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = asMap[part]; !ok {
			return nil, false
		}
	}
	return cur, true
}

// applyGitOpsOrgSettingsDefaults resets the org_settings the YAML omits and
// returns, per high-risk section, the omitted keys whose value on the server
// is not already the default.
func applyGitOpsOrgSettingsDefaults(orgSettings map[string]any, current *fleet.EnrichedAppConfig) (map[string][]string, error) {
	defaults, err := gitOpsOrgSettingsDefaults()
	if err != nil {
		return nil, err
	}
	aliases := make(map[string]string)
	for _, rule := range endpointer.ExtractAliasRules(fleet.AppConfig{}) {
		aliases[rule.OldKey] = rule.NewKey
	}
	// The API never returns the password, so a repo generated from the server
	// holds a blank one. Sending no password keeps the stored one; omitting the
	// whole section is what clears it.
	smtp, _ := orgSettings["smtp_settings"].(map[string]any)
	keepSMTPPassword := false
	if smtp != nil {
		password, _ := smtp["password"].(string)
		keepSMTPPassword = password == ""
	}

	filled := fillGitOpsDefaults(orgSettings, defaults, aliases, "")
	if keepSMTPPassword {
		delete(smtp, "password")
		filled = slices.DeleteFunc(filled, func(path string) bool { return path == "smtp_settings.password" })
	}

	if current == nil {
		return nil, nil
	}
	raw, err := json.Marshal(current.AppConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal current config: %w", err)
	}
	var currentMap map[string]any
	if err := json.Unmarshal(raw, &currentMap); err != nil {
		return nil, fmt.Errorf("unmarshal current config: %w", err)
	}

	resets := make(map[string][]string)
	for _, path := range filled {
		section, key, _ := strings.Cut(path, ".")
		if !slices.Contains(gitOpsHighRiskOrgSettings, section) {
			continue
		}
		// A section the token can't read (smtp_settings for non-admins) is
		// absent here, so nothing can be said about it.
		currentVal, ok := lookupPath(currentMap, path)
		if !ok {
			continue
		}
		defaultVal, _ := lookupPath(defaults, path)
		if reflect.DeepEqual(currentVal, defaultVal) || isEmptyJSONValue(currentVal) && isEmptyJSONValue(defaultVal) {
			continue
		}
		if alias, ok := aliases[key]; ok {
			key = alias
		}
		resets[section] = append(resets[section], key)
	}
	for section := range resets {
		slices.Sort(resets[section])
	}
	return resets, nil
}

func isEmptyJSONValue(v any) bool {
	switch tv := v.(type) {
	case nil:
		return true
	case []any:
		return len(tv) == 0
	case string:
		return tv == ""
	}
	return false
}
