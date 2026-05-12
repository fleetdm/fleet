package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

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
func (c *Client) ApplyNoTeamProfiles(profiles []fleet.MDMProfileBatchPayload, opts fleet.ApplySpecOptions, assumeEnabled bool) error {
	verb, path := "POST", "/api/latest/fleet/mdm/profiles/batch"
	query := opts.RawQuery()
	if assumeEnabled {
		if query != "" {
			query += "&"
		}
		query += "assume_enabled=true"
	}
	return c.authenticatedRequestWithQuery(map[string]interface{}{"profiles": profiles}, verb, path, nil, query)
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
	// replaceWithURL is set on a delete action when the gitops YAML is
	// switching from a Fleet-hosted logo (path key) to an external URL.
	// DeleteOrgLogo unconditionally clears the URL fields it tracks, so
	// the new URL has to be written after the delete — see
	// doGitOpsOrgLogos for the follow-up PATCH.
	replaceWithURL string
}

// planAndStripOrgLogos plans the PUT/DELETE /logo calls that run after the
// AppConfig PATCH, and removes any org_info keys that shouldn't ride along
// on that PATCH:
//
//   - path keys are always removed: they're gitops-only, with no matching
//     field on fleet.OrgInfo.
//   - the URL key for a given mode is also removed when the YAML supplied
//     a path key for that same mode. Example: if the YAML sets
//     org_logo_path_dark_mode, we strip org_logo_url_dark_mode too so the
//     follow-up PUT is the sole writer of OrgLogoURLDarkMode — otherwise
//     PATCH would blank that field briefly before PUT corrects it.
//   - when the YAML supplies a URL key for a mode that currently holds a
//     Fleet-hosted blob, we strip the URL keys too. The follow-up DELETE
//     clears the URL fields it tracks, which would clobber any URL set
//     by the PATCH; doGitOpsOrgLogos re-PATCHes the intended URL after
//     the blob is gone.
//
// Modes the YAML doesn't mention are left alone (current server state preserved).
func (c *Client) planAndStripOrgLogos(
	orgSettings map[string]any,
	currentOrgInfo *fleet.OrgInfo,
	baseDir string,
	dryRun bool,
	logFn func(format string, args ...any),
) ([]orgLogoAction, error) {
	orgInfo, _ := orgSettings["org_info"].(map[string]any)
	if orgInfo == nil {
		return nil, nil
	}

	type modeSpec struct {
		mode             fleet.OrgLogoMode
		pathKey          string
		urlKey           string
		deprecatedURLKey string
		currentURL       string
	}
	specs := []modeSpec{
		{fleet.OrgLogoModeLight, "org_logo_path_light_mode", "org_logo_url_light_mode", "org_logo_url_light_background", currentOrgInfo.OrgLogoURLLightMode},
		{fleet.OrgLogoModeDark, "org_logo_path_dark_mode", "org_logo_url_dark_mode", "org_logo_url", currentOrgInfo.OrgLogoURLDarkMode},
	}

	var actions []orgLogoAction
	for _, s := range specs {
		_, pathPresent := orgInfo[s.pathKey]
		_, urlPresent := orgInfo[s.urlKey]
		if !pathPresent && !urlPresent {
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
			// Strip every URL key for this mode: PUT will set the served URL
			// (and its deprecated alias) after the PATCH, so we must keep
			// PATCH from writing anything to either URL field.
			delete(orgInfo, s.pathKey)
			delete(orgInfo, s.urlKey)
			delete(orgInfo, s.deprecatedURLKey)
			if dryRun {
				logFn("[+] would upload org logo (%s) from %s\n", s.mode, yamlPath)
			}
		case yamlURL != "":
			delete(orgInfo, s.pathKey)
			if fleet.IsFleetHostedLogoURL(s.currentURL) {
				// Switching from a Fleet-hosted logo to an external URL:
				// the orphan blob still needs a DELETE /logo, and that
				// call clears the URL fields it tracks. If the PATCH set
				// the new URL up front, the DELETE would clobber it.
				// Strip the URL keys from this PATCH and carry the new
				// URL on the delete action; doGitOpsOrgLogos re-PATCHes
				// it once the blob is gone.
				delete(orgInfo, s.urlKey)
				delete(orgInfo, s.deprecatedURLKey)
				actions = append(actions, orgLogoAction{mode: s.mode, replaceWithURL: yamlURL})
			} else {
				// No Fleet-hosted blob to clean up — the PATCH alone can
				// flip the URL. Mirror the new key into the deprecated
				// alias so server-side NormalizeLogoFields can't undo our
				// write (a PATCH that only sets the new key leaves the
				// deprecated field unchanged, and the post-merge
				// normalization copies the old value back into the new
				// field). See server/service/appconfig.go ModifyAppConfig.
				orgInfo[s.deprecatedURLKey] = yamlURL
			}
		default:
			if fleet.IsFleetHostedLogoURL(s.currentURL) {
				actions = append(actions, orgLogoAction{mode: s.mode})
			}
			delete(orgInfo, s.pathKey)
			// Same reason as above: send both keys as "" so the server
			// can't restore the previous value via NormalizeLogoFields.
			orgInfo[s.deprecatedURLKey] = ""
		}
	}
	return actions, nil
}

// doGitOpsOrgLogos executes the actions planned by planAndStripOrgLogos. Runs
// after the AppConfig PATCH so a PATCH failure leaves storage untouched.
func (c *Client) doGitOpsOrgLogos(
	actions []orgLogoAction, dryRun bool, logFn func(format string, args ...any),
) error {
	for _, a := range actions {
		if a.uploadPath != "" {
			if dryRun {
				continue // already logged at planning time
			}
			if err := c.UploadOrgLogo(a.mode, a.uploadPath); err != nil {
				return fmt.Errorf("uploading org logo (%s): %w", a.mode, err)
			}
			logFn("[+] applied org logo (%s) from %s\n", a.mode, a.uploadPath)
			continue
		}
		if dryRun {
			if a.replaceWithURL != "" {
				logFn("[+] would replace org logo (%s) with %s\n", a.mode, a.replaceWithURL)
			} else {
				logFn("[+] would delete org logo (%s)\n", a.mode)
			}
			continue
		}
		if err := c.DeleteOrgLogo(a.mode); err != nil {
			return fmt.Errorf("deleting org logo (%s): %w", a.mode, err)
		}
		if a.replaceWithURL != "" {
			// DeleteOrgLogo cleared the URL fields it tracks, so write
			// the intended external URL now. Both the new and deprecated
			// keys are mirrored to keep server-side NormalizeLogoFields a
			// no-op.
			if err := c.setOrgLogoURL(a.mode, a.replaceWithURL); err != nil {
				return fmt.Errorf("setting org logo URL (%s): %w", a.mode, err)
			}
			logFn("[+] replaced org logo (%s) with %s\n", a.mode, a.replaceWithURL)
			continue
		}
		logFn("[+] deleted org logo (%s)\n", a.mode)
	}
	return nil
}

func (c *Client) setOrgLogoURL(mode fleet.OrgLogoMode, url string) error {
	var urlKey, deprecatedURLKey string
	switch mode {
	case fleet.OrgLogoModeLight:
		urlKey, deprecatedURLKey = "org_logo_url_light_mode", "org_logo_url_light_background"
	case fleet.OrgLogoModeDark:
		urlKey, deprecatedURLKey = "org_logo_url_dark_mode", "org_logo_url"
	default:
		return fmt.Errorf("unsupported mode %q for org logo URL set", mode)
	}
	payload := map[string]any{
		"org_info": map[string]any{
			urlKey:           url,
			deprecatedURLKey: url,
		},
	}
	return c.ApplyAppConfig(payload, fleet.ApplySpecOptions{})
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
