package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
)

// TODO(mna): those methods are unused except for an internal tool, remove or
// migrate to new endpoints (those apple-specific endpoints are deprecated)?

func (c *Client) DeleteProfile(profileID uint) error {
	verb, path := "DELETE", "/api/latest/fleet/mdm/apple/profiles/"+strconv.FormatUint(uint64(profileID), 10)
	var responseBody deleteMDMAppleConfigProfileResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}

func (c *Client) ListProfiles(teamID *uint) ([]*fleet.MDMAppleConfigProfile, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/apple/profiles"
	query := make(url.Values)
	if teamID != nil {
		query.Add("fleet_id", strconv.FormatUint(uint64(*teamID), 10))
	}
	var responseBody listMDMAppleConfigProfilesResponse
	if err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode()); err != nil {
		return nil, err
	}
	return responseBody.ConfigProfiles, nil
}

func (c *Client) ListConfigurationProfiles(teamID *uint) ([]*fleet.MDMConfigProfilePayload, error) {
	verb, path := "GET", "/api/latest/fleet/configuration_profiles"
	query := make(url.Values)
	if teamID != nil {
		query.Add("fleet_id", strconv.FormatUint(uint64(*teamID), 10))
	}
	var responseBody listMDMConfigProfilesResponse
	if err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode()); err != nil {
		return nil, err
	}
	return responseBody.Profiles, nil
}

// windowsProfileDeletionWarning checks, before a profile batch runs, whether
// the batch deletes any of the fleet's Windows profiles. It returns a function
// that prints the warning, to call once the batch has succeeded.
//
// It only checks when Windows MDM is on before and after the run, the only
// case where a deletion sends <Delete> commands to hosts. Unlike the batch
// endpoint, the list endpoint requires MDM, and the warning is only
// informational, so a failed lookup is reported rather than failing the run.
func (c *Client) windowsProfileDeletionWarning(
	appConfig *fleet.EnrichedAppConfig,
	assumptions *fleet.TeamSpecsDryRunAssumptions,
	teamID *uint,
	fleetName string,
	incoming []fleet.MDMProfileBatchPayload,
	logFn func(format string, args ...any),
) func() {
	noWarning := func() {}
	if appConfig == nil || !appConfig.MDM.WindowsEnabledAndConfigured {
		return noWarning
	}
	if assumptions != nil && assumptions.WindowsEnabledAndConfigured.Valid && !assumptions.WindowsEnabledAndConfigured.Value {
		return noWarning
	}

	current, err := c.ListConfigurationProfiles(teamID)
	if err != nil {
		logFn("[!] couldn't check for deleted Windows profiles: %v\n", err)
		return noWarning
	}
	// The server keeps a Windows profile only if an incoming Windows profile
	// has the same trimmed name. Fleet-reserved profiles aren't listed.
	keep := make(map[string]struct{}, len(incoming))
	for _, p := range incoming {
		if mdm.GetRawProfilePlatform(p.Contents) == "windows" {
			keep[strings.TrimSpace(p.Name)] = struct{}{}
		}
	}
	deletes := slices.ContainsFunc(current, func(p *fleet.MDMConfigProfilePayload) bool {
		_, kept := keep[strings.TrimSpace(p.Name)]
		return p.Platform == "windows" && !kept
	})

	switch {
	case !deletes:
		return noWarning
	case appConfig.License == nil || !appConfig.License.IsPremium():
		// Fleet Free has no fleets to name.
		return func() {
			logFn("[!] Some Windows settings from deleted profiles may not be cleanly removed from hosts. Learn more: %s\n",
				fleet.WindowsProfileRemovalBehaviorURL)
		}
	default:
		return func() {
			logFn("[!] Fleet %q: some Windows settings from deleted profiles may not be cleanly removed from hosts. Learn more: %s\n",
				fleetName, fleet.WindowsProfileRemovalBehaviorURL)
		}
	}
}

// Get the contents of a saved profile.
func (c *Client) GetProfileContents(profileID string) ([]byte, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/profiles/"+profileID
	response, err := c.AuthenticatedDo(verb, path, "alt=media", nil)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()
	err = c.ParseResponse(verb, path, response, nil)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", verb, path, err)
	}
	if response.StatusCode != http.StatusNoContent {
		b, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}
		return b, nil
	}
	return nil, nil
}

// GetProfileActivation returns the custom activation attached to a declaration,
// or nil if it has none. GetProfileContents can't serve this because alt=media
// returns the declaration file itself, not the payload the activation rides on.
func (c *Client) GetProfileActivation(profileID string) ([]byte, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/profiles/"+profileID
	var responseBody getMDMConfigProfileResponse
	if err := c.authenticatedRequest(nil, verb, path, &responseBody); err != nil {
		return nil, err
	}
	if responseBody.MDMConfigProfilePayload == nil {
		return nil, nil
	}
	return responseBody.MDMConfigProfilePayload.Activation, nil
}

// ListDDMAssets returns the Apple DDM assets for the given team.
func (c *Client) ListDDMAssets(teamID *uint) ([]*fleet.DDMAsset, error) {
	verb, path := "GET", "/api/latest/fleet/assets"
	query := make(url.Values)
	if teamID != nil {
		query.Add("fleet_id", strconv.FormatUint(uint64(*teamID), 10))
	}
	var responseBody listAppleDDMAssetsResponse
	if err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode()); err != nil {
		return nil, err
	}
	return responseBody.Assets, nil
}

// DownloadDDMAsset returns the raw JSON contents of the DDM asset with the given UUID.
func (c *Client) DownloadDDMAsset(assetUUID string) ([]byte, error) {
	verb, path := "GET", "/api/latest/fleet/assets/"+assetUUID
	response, err := c.AuthenticatedDo(verb, path, "alt=media", nil)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()
	if err := c.ParseResponse(verb, path, response, nil); err != nil {
		return nil, fmt.Errorf("%s %s: %w", verb, path, err)
	}
	return io.ReadAll(response.Body)
}

func (c *Client) AddProfile(teamID uint, configurationProfile []byte) (uint, error) {
	if c.token == "" {
		return 0, errors.New("authentication token is empty")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	teamIDField, err := writer.CreateFormField("fleet_id")
	if err != nil {
		return 0, err
	}
	if _, err := teamIDField.Write([]byte(strconv.FormatUint(uint64(teamID), 10))); err != nil {
		return 0, err
	}
	profileField, err := writer.CreateFormFile("profile", "mobileconfig")
	if err != nil {
		return 0, err
	}
	if _, err := profileField.Write(configurationProfile); err != nil {
		return 0, err
	}
	if err := writer.Close(); err != nil {
		return 0, err
	}

	request, err := http.NewRequest(
		"POST",
		c.BaseURL.String()+"/api/latest/fleet/mdm/apple/profiles",
		body,
	)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	response, err := c.HTTP.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.Header.Get(fleet.HeaderLicenseKey) == fleet.HeaderLicenseValueExpired {
		fleet.WriteExpiredLicenseBanner(c.errWriter)
	}

	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("request failed: %s", response.Status)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	var addProfileResponse *newMDMAppleConfigProfileResponse
	if err := json.Unmarshal(responseBody, &addProfileResponse); err != nil {
		return 0, err
	}

	return addProfileResponse.ProfileID, nil
}

func (c *Client) GetConfigProfilesSummary(teamID *uint) (*fleet.MDMProfilesSummary, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/profiles/summary"
	query := make(url.Values)
	if teamID != nil {
		query.Add("fleet_id", strconv.FormatUint(uint64(*teamID), 10))
	}
	var responseBody getMDMProfilesSummaryResponse
	if err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode()); err != nil {
		return nil, err
	}
	return &responseBody.MDMProfilesSummary, nil
}

// Get the Apple setup assistant profile for the given team, if any.
func (c *Client) GetAppleMDMEnrollmentProfile(teamID uint) (*fleet.MDMAppleSetupAssistant, error) {
	verb, path := "GET", "/api/latest/fleet/enrollment_profiles/automatic"
	var query string
	if teamID != 0 {
		query = fmt.Sprintf("fleet_id=%d", teamID)
	}
	var responseBody createMDMAppleSetupAssistantResponse
	if err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query); err != nil {
		if isNotFoundErr(err) {
			// If the profile is not found, return nil instead of an error.
			return nil, nil
		}
		return nil, err
	}
	return &responseBody.MDMAppleSetupAssistant, nil
}

// rewrapProfileBatchNameErr surfaces the offending profile's name when the
// batch-set profiles endpoint rejects one of them. The server puts the name in
// the error's name field, bare or as "profiles[<name>]", but not in the reason,
// so fleetctl output could not say which profile failed. Names that don't match
// a profile in the batch (e.g. "mdm", "labels") are left alone.
func rewrapProfileBatchNameErr(err error, profiles []fleet.MDMProfileBatchPayload) error {
	var scErr *StatusCodeErr
	if !errors.As(err, &scErr) || scErr.Name == "" {
		return err
	}
	name := scErr.Name
	if inner, ok := strings.CutPrefix(name, "profiles["); ok {
		inner, ok = strings.CutSuffix(inner, "]")
		if !ok {
			return err
		}
		name = inner
	}
	for _, p := range profiles {
		if p.Name == name {
			return fmt.Errorf("profile %q: %w", name, err)
		}
	}
	return err
}
