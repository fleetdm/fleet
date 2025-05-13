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
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
		query.Add("team_id", strconv.FormatUint(uint64(*teamID), 10))
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
		query.Add("team_id", strconv.FormatUint(uint64(*teamID), 10))
	}
	var responseBody listMDMConfigProfilesResponse
	if err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode()); err != nil {
		return nil, err
	}
	return responseBody.Profiles, nil
}

// Get the contents of a saved profile.
func (c *Client) GetProfileContents(profileID string) ([]byte, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/profiles/"+profileID
	response, err := c.AuthenticatedDo(verb, path, "alt=media", nil)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()
	err = c.parseResponse(verb, path, response, nil)
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

func (c *Client) AddProfile(teamID uint, configurationProfile []byte) (uint, error) {
	if c.token == "" {
		return 0, errors.New("authentication token is empty")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	teamIDField, err := writer.CreateFormField("team_id")
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
		c.baseURL.String()+"/api/latest/fleet/mdm/apple/profiles",
		body,
	)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	response, err := c.http.Do(request)
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
		query.Add("team_id", strconv.FormatUint(uint64(*teamID), 10))
	}
	var responseBody getMDMProfilesSummaryResponse
	if err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode()); err != nil {
		return nil, err
	}
	return &responseBody.MDMProfilesSummary, nil
}
