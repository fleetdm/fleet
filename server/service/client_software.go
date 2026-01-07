package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ListSoftwareVersions retrieves the software versions installed on hosts.
func (c *Client) ListSoftwareVersions(query string) ([]fleet.Software, error) {
	verb, path := "GET", "/api/latest/fleet/software/versions"
	var responseBody listSoftwareVersionsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Software, nil
}

// ListSoftwareTitles retrieves the software titles installed on hosts.
func (c *Client) ListSoftwareTitles(query string) ([]fleet.SoftwareTitleListResult, error) {
	verb, path := "GET", "/api/latest/fleet/software/titles"
	var responseBody listSoftwareTitlesResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.SoftwareTitles, nil
}

// Get the software titles available for the setup experience.
func (c *Client) GetSetupExperienceSoftware(platform string, teamID uint) ([]fleet.SoftwareTitleListResult, error) {
	verb, path := "GET", "/api/latest/fleet/setup_experience/software"
	var responseBody getSetupExperienceSoftwareResponse
	query := fmt.Sprintf("platform=%s&team_id=%d", platform, teamID)
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.SoftwareTitles, nil
}

// GetSoftwareTitleByID retrieves a software title by ID.
//
//nolint:gocritic // ignore captLocal
func (c *Client) GetSoftwareTitleByID(ID uint, teamID *uint) (*fleet.SoftwareTitle, error) {
	var query string
	if teamID != nil {
		query = fmt.Sprintf("team_id=%d", *teamID)
	}
	verb, path := "GET", "/api/latest/fleet/software/titles/"+fmt.Sprint(ID)
	var responseBody getSoftwareTitleResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.SoftwareTitle, nil
}

func (c *Client) GetSoftwareTitleIcon(titleID uint, teamID uint) ([]byte, error) {
	verb, path := "GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon", titleID)
	response, err := c.AuthenticatedDo(verb, path, fmt.Sprintf("team_id=%d", teamID), nil)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()
	err = c.parseResponse(verb, path, response, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing icon response: %w", err)
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

func (c *Client) ApplyNoTeamSoftwareInstallers(softwareInstallers []fleet.SoftwareInstallerPayload, opts fleet.ApplySpecOptions) ([]fleet.SoftwarePackageResponse, error) {
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	return c.applySoftwareInstallers(softwareInstallers, query, opts.DryRun)
}

func (c *Client) applySoftwareInstallers(softwareInstallers []fleet.SoftwareInstallerPayload, query url.Values, dryRun bool) ([]fleet.SoftwarePackageResponse, error) {
	path := "/api/latest/fleet/software/batch"
	var resp batchSetSoftwareInstallersResponse
	if err := c.authenticatedRequestWithQuery(map[string]any{"software": softwareInstallers}, "POST", path, &resp, query.Encode()); err != nil {
		return nil, err
	}
	if dryRun && resp.RequestUUID == "" {
		return nil, nil
	}

	requestUUID := resp.RequestUUID
	for {
		var resp batchSetSoftwareInstallersResultResponse
		if err := c.authenticatedRequestWithQuery(nil, "GET", path+"/"+requestUUID, &resp, query.Encode()); err != nil {
			return nil, err
		}
		switch {
		case resp.Status == fleet.BatchSetSoftwareInstallersStatusProcessing:
			time.Sleep(1 * time.Second)
		case resp.Status == fleet.BatchSetSoftwareInstallersStatusFailed:
			return nil, errors.New(resp.Message)
		case resp.Status == fleet.BatchSetSoftwareInstallersStatusCompleted:
			return matchPackageIcons(softwareInstallers, resp.Packages), nil
		default:
			return nil, fmt.Errorf("unknown status: %q", resp.Status)
		}
	}
}

// matchPackageIcons hydrates software responses with references to icons in the request payload, so we can track
// which API calls to make to add/update/delete icons
func matchPackageIcons(request []fleet.SoftwareInstallerPayload, response []fleet.SoftwarePackageResponse) []fleet.SoftwarePackageResponse {
	// On the client side, software installer entries can have a URL or a hash or both ...
	byURL := make(map[string]*fleet.SoftwareInstallerPayload)
	byHash := make(map[string]*fleet.SoftwareInstallerPayload)

	for i := range request {
		clientSide := &request[i]

		if clientSide.URL != "" {
			byURL[clientSide.URL] = clientSide
		}
		if clientSide.SHA256 != "" {
			byHash[clientSide.SHA256] = clientSide
		}
	}

	for i := range response {
		serverSide := &response[i]

		// All server side entries have a hash, so first try to match by that
		if clientSide, ok := byHash[serverSide.HashSHA256]; ok {
			serverSide.LocalIconHash = clientSide.IconHash
			serverSide.LocalIconPath = clientSide.IconPath
			continue
		}

		// ... Then by URL
		if clientSide, ok := byURL[serverSide.URL]; ok {
			serverSide.LocalIconHash = clientSide.IconHash
			serverSide.LocalIconPath = clientSide.IconPath
		}
	}

	return response
}

func (c *Client) UploadIcon(teamID uint, titleID uint, filename string, iconReader io.Reader) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	fileWriter, err := writer.CreateFormFile("icon", filename)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fileWriter, iconReader); err != nil {
		return err
	}
	// Close the writer before using the buffer
	if err := writer.Close(); err != nil {
		return err
	}

	return c.putIcon(teamID, titleID, writer, buf)
}

func (c *Client) UpdateIcon(teamID uint, titleID uint, filename string, hash string) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("hash_sha256", hash); err != nil {
		return err
	}
	if err := writer.WriteField("filename", filename); err != nil {
		return err
	}
	// Close the writer before using the buffer
	if err := writer.Close(); err != nil {
		return err
	}

	return c.putIcon(teamID, titleID, writer, buf)
}

func (c *Client) putIcon(teamID uint, titleID uint, writer *multipart.Writer, buf bytes.Buffer) error {
	response, err := c.doContextWithBodyAndHeaders(
		context.Background(),
		"PUT",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon", titleID),
		fmt.Sprintf("team_id=%d", teamID),
		buf.Bytes(),
		map[string]string{
			"Content-Type":  writer.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", c.token),
		},
	)
	if err != nil {
		return fmt.Errorf("do multipart request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("update icon: unexpected status code: %d", response.StatusCode)
	}

	return nil
}

func (c *Client) DeleteIcon(teamID uint, titleID uint) error {
	response, err := c.AuthenticatedDo(
		"DELETE",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon", titleID),
		fmt.Sprintf("team_id=%d", teamID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("delete icon: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("delete icon: unexpected status code: %d", response.StatusCode)
	}

	return nil
}

// InstallSoftware triggers a software installation (VPP or software package)
// on the specified host.
func (c *Client) InstallSoftware(hostID uint, softwareTitleID uint) error {
	verb, path := "POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostID, softwareTitleID)
	var responseBody installSoftwareResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
