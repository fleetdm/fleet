package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/beevik/etree"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"howett.net/plist"
)

// GetAppleMDM retrieves the Apple MDM APNs information.
func (c *Client) GetAppleMDM() (*fleet.AppleMDM, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/apple"
	var responseBody getAppleMDMResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, "")
	return responseBody.AppleMDM, err
}

// GetAppleBM retrieves the Apple Business Manager information.
func (c *Client) GetAppleBM() (*fleet.AppleBM, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/apple_bm"
	var responseBody getAppleBMResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, "")
	return responseBody.AppleBM, err
}

func (c *Client) CountABMTokens() (int, error) {
	verb, path := "GET", "/api/latest/fleet/abm_tokens/count"
	var responseBody countABMTokensResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, "")
	return responseBody.Count, err
}

// RequestAppleCSR requests a signed CSR from the Fleet server and returns the
// CSR bytes
func (c *Client) RequestAppleCSR() ([]byte, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/apple/request_csr"
	var resp getMDMAppleCSRResponse
	err := c.authenticatedRequest(nil, verb, path, &resp)
	return resp.CSR, err
}

// RequestAppleABM requests a signed CSR from the Fleet server and returns the
// public key bytes
func (c *Client) RequestAppleABM() ([]byte, error) {
	verb, path := "GET", "/api/latest/fleet/mdm/apple/abm_public_key"
	var resp generateABMKeyPairResponse
	err := c.authenticatedRequest(nil, verb, path, &resp)
	return resp.PublicKey, err
}

func (c *Client) GetBootstrapPackageMetadata(teamID uint, forUpdate bool) (*fleet.MDMAppleBootstrapPackage, error) {
	verb, path := "GET", fmt.Sprintf("/api/latest/fleet/mdm/bootstrap/%d/metadata", teamID)
	request := bootstrapPackageMetadataRequest{}
	var responseBody bootstrapPackageMetadataResponse
	var err error
	if forUpdate {
		err = c.authenticatedRequestWithQuery(request, verb, path, &responseBody, "for_update=true")
	} else {
		err = c.authenticatedRequest(request, verb, path, &responseBody)
	}
	return responseBody.MDMAppleBootstrapPackage, err
}

func (c *Client) DeleteBootstrapPackage(teamID uint) error {
	verb, path := "DELETE", fmt.Sprintf("/api/latest/fleet/mdm/bootstrap/%d", teamID)
	request := deleteBootstrapPackageRequest{}
	var responseBody deleteBootstrapPackageResponse
	err := c.authenticatedRequest(request, verb, path, &responseBody)
	return err
}

func (c *Client) UploadBootstrapPackage(pkg *fleet.MDMAppleBootstrapPackage) error {
	verb, path := "POST", "/api/latest/fleet/mdm/bootstrap"

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// add the package field
	fw, err := w.CreateFormFile("package", pkg.Name)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, bytes.NewBuffer(pkg.Bytes)); err != nil {
		return err
	}

	// add the team_id field
	if err := w.WriteField("team_id", fmt.Sprint(pkg.TeamID)); err != nil {
		return err
	}

	w.Close()

	response, err := c.doContextWithBodyAndHeaders(context.Background(), verb, path, "",
		b.Bytes(),
		map[string]string{
			"Content-Type":  w.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", c.token),
		},
	)
	if err != nil {
		return fmt.Errorf("do multipart request: %w", err)
	}
	defer response.Body.Close()

	var bpResponse uploadBootstrapPackageResponse
	if err := c.parseResponse(verb, path, response, &bpResponse); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	return nil
}

func (c *Client) EnsureBootstrapPackage(bp *fleet.MDMAppleBootstrapPackage, teamID uint) error {
	isFirstTime := false
	oldMeta, err := c.GetBootstrapPackageMetadata(teamID, true)
	if err != nil {
		// not found is OK, it means this is our first time uploading a package
		if !errors.As(err, &notFoundErr{}) {
			return fmt.Errorf("getting bootstrap package metadata: %w", err)
		}
		isFirstTime = true
	}

	if !isFirstTime {
		// compare checksums, if they're equal then we can skip the package upload.
		if bytes.Equal(oldMeta.Sha256, bp.Sha256) {
			return nil
		}

		// similar to the expected UI experience, delete the bootstrap package first
		err = c.DeleteBootstrapPackage(teamID)
		if err != nil {
			return fmt.Errorf("deleting old bootstrap package: %w", err)
		}
	}

	bp.TeamID = teamID
	if err := c.UploadBootstrapPackage(bp); err != nil {
		return err
	}

	return nil
}

func (c *Client) ValidateBootstrapPackageFromURL(url string) (*fleet.MDMAppleBootstrapPackage, error) {
	if err := c.CheckPremiumMDMEnabled(); err != nil {
		return nil, err
	}

	return downloadRemoteMacosBootstrapPackage(url)
}

func downloadRemoteMacosBootstrapPackage(pkgURL string) (*fleet.MDMAppleBootstrapPackage, error) {
	resp, err := http.Get(pkgURL) // nolint:gosec // we want this URL to be provided by the user. It will run on their machine.
	if err != nil {
		return nil, fmt.Errorf("downloading bootstrap package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("the URL to the bootstrap_package doesn't exist. Please make this URL publicly accessible to the internet.")
	}

	// try to extract the name from a header
	var filename string
	cdh, ok := resp.Header["Content-Disposition"]
	if ok && len(cdh) > 0 {
		_, params, err := mime.ParseMediaType(cdh[0])
		if err == nil {
			filename = params["filename"]
		}
	}

	// if it fails, try to extract it from the URL
	if filename == "" {
		filename = file.ExtractFilenameFromURLPath(pkgURL, "pkg")
	}

	// if all else fails, use a default name
	if filename == "" {
		filename = "bootstrap-package.pkg"
	}

	// get checksums
	var pkgBuf bytes.Buffer
	hash := sha256.New()
	if _, err := io.Copy(hash, io.TeeReader(resp.Body, &pkgBuf)); err != nil {
		return nil, fmt.Errorf("calculating sha256 of package: %w", err)
	}

	pkgReader := bytes.NewReader(pkgBuf.Bytes())
	if err := file.CheckPKGSignature(pkgReader); err != nil {
		switch {
		case errors.Is(err, file.ErrInvalidType):
			return nil, errors.New("Couldn’t edit bootstrap_package. The file must be a package (.pkg).")
		case errors.Is(err, file.ErrNotSigned):
			return nil, errors.New("Couldn’t edit bootstrap_package. The bootstrap_package must be signed. Learn how to sign the package in the Fleet documentation: https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#step-2-sign-the-package")
		default:
			return nil, fmt.Errorf("checking package signature: %w", err)
		}
	}

	return &fleet.MDMAppleBootstrapPackage{
		Name:   filename,
		Bytes:  pkgBuf.Bytes(),
		Sha256: hash.Sum(nil),
	}, nil
}

func (c *Client) validateMacOSSetupAssistant(fileName string) ([]byte, error) {
	if err := c.CheckAppleMDMEnabled(); err != nil {
		return nil, err
	}

	if strings.ToLower(filepath.Ext(fileName)) != ".json" {
		return nil, errors.New("Couldn’t edit macos_setup_assistant. The file should be a .json file.")
	}

	b, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("Couldn’t edit macos_setup_assistant. The file should include valid JSON: %w", err)
	}

	return b, nil
}

func (c *Client) uploadMacOSSetupAssistant(data []byte, teamID *uint, name string) error {
	verb, path := "POST", "/api/latest/fleet/mdm/apple/enrollment_profile"
	request := createMDMAppleSetupAssistantRequest{
		TeamID:            teamID,
		Name:              name,
		EnrollmentProfile: json.RawMessage(data),
	}
	return c.authenticatedRequest(request, verb, path, nil)
}

func (c *Client) MDMListCommands(opts fleet.MDMCommandListOptions) ([]*fleet.MDMCommand, error) {
	const defaultCommandsPerPage = 20

	verb, path := http.MethodGet, "/api/latest/fleet/mdm/commands"

	query := url.Values{}
	query.Set("per_page", fmt.Sprint(defaultCommandsPerPage))
	query.Set("order_key", "updated_at")
	query.Set("order_direction", "desc")
	query.Set("host_identifier", opts.Filters.HostIdentifier)
	query.Set("request_type", opts.Filters.RequestType)

	var responseBody listMDMCommandsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, err
	}

	return responseBody.Results, nil
}

func (c *Client) MDMGetCommandResults(commandUUID string) ([]*fleet.MDMCommandResult, error) {
	verb, path := http.MethodGet, "/api/latest/fleet/mdm/commandresults"

	query := url.Values{}
	query.Set("command_uuid", commandUUID)

	var responseBody getMDMCommandResultsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return responseBody.Results, nil
}

func (c *Client) RunMDMCommand(hostUUIDs []string, rawCmd []byte, forPlatform string) (*fleet.CommandEnqueueResult, error) {
	var prepareFn func([]byte) ([]byte, error)
	switch forPlatform {
	case "darwin":
		prepareFn = c.prepareAppleMDMCommand
	case "windows":
		prepareFn = c.prepareWindowsMDMCommand
	default:
		return nil, fmt.Errorf("Invalid platform %q. You can only run MDM commands on Windows or Apple hosts.", forPlatform)
	}

	rawCmd, err := prepareFn(rawCmd)
	if err != nil {
		return nil, err
	}

	request := runMDMCommandRequest{
		Command:   base64.RawStdEncoding.EncodeToString(rawCmd),
		HostUUIDs: hostUUIDs,
	}
	var response runMDMCommandResponse
	if err := c.authenticatedRequest(request, "POST", "/api/latest/fleet/mdm/commands/run", &response); err != nil {
		return nil, fmt.Errorf("run command request: %w", err)
	}
	return response.CommandEnqueueResult, nil
}

func (c *Client) prepareWindowsMDMCommand(rawCmd []byte) ([]byte, error) {
	if _, err := fleet.ParseWindowsMDMCommand(rawCmd); err != nil {
		return nil, err
	}

	// ensure there's a CmdID with a random UUID value, we're manipulating
	// the document this way to make sure we don't introduce any unintended
	// changes to the command XML.
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(rawCmd); err != nil {
		return nil, err
	}
	element := doc.FindElement("//CmdID")
	// if we can't find a CmdID, just add one.
	if element == nil {
		root := doc.Root()
		element = root.CreateElement("CmdID")
	}
	element.SetText(uuid.NewString())

	return doc.WriteToBytes()
}

func (c *Client) prepareAppleMDMCommand(rawCmd []byte) ([]byte, error) {
	var commandPayload map[string]interface{}
	if _, err := plist.Unmarshal(rawCmd, &commandPayload); err != nil {
		return nil, fmt.Errorf("The payload isn't valid XML. Please provide a file with valid XML: %w", err)
	}
	if commandPayload == nil {
		return nil, errors.New("The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file.")
	}

	// generate a random command UUID
	commandPayload["CommandUUID"] = uuid.New().String()

	b, err := plist.Marshal(commandPayload, plist.XMLFormat)
	if err != nil {
		return nil, fmt.Errorf("marshal command plist: %w", err)
	}
	return b, nil
}

func (c *Client) MDMLockHost(hostID uint) error {
	var response lockHostResponse
	if err := c.authenticatedRequest(nil, "POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", hostID), &response); err != nil {
		return fmt.Errorf("lock host request: %w", err)
	}
	return nil
}

func (c *Client) MDMUnlockHost(hostID uint) (string, error) {
	var response unlockHostResponse
	if err := c.authenticatedRequest(nil, "POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", hostID), &response); err != nil {
		return "", fmt.Errorf("lock host request: %w", err)
	}
	return response.UnlockPIN, nil
}

func (c *Client) MDMWipeHost(hostID uint) error {
	var response wipeHostResponse
	if err := c.authenticatedRequest(nil, "POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", hostID), &response); err != nil {
		return fmt.Errorf("wipe host request: %w", err)
	}
	return nil
}
