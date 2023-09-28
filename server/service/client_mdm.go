package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

// RequestAppleCSR requests a signed CSR from the Fleet server and returns the
// SCEP certificate and key along with the APNs key used for the CSR.
func (c *Client) RequestAppleCSR(email, org string) (*fleet.AppleCSR, error) {
	verb, path := "POST", "/api/latest/fleet/mdm/apple/request_csr"
	request := requestMDMAppleCSRRequest{
		EmailAddress: email,
		Organization: org,
	}
	var responseBody requestMDMAppleCSRResponse
	err := c.authenticatedRequest(request, verb, path, &responseBody)
	return responseBody.AppleCSR, err
}

func (c *Client) GetBootstrapPackageMetadata(teamID uint, forUpdate bool) (*fleet.MDMAppleBootstrapPackage, error) {
	verb, path := "GET", fmt.Sprintf("/api/latest/fleet/mdm/apple/bootstrap/%d/metadata", teamID)
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
	verb, path := "DELETE", fmt.Sprintf("/api/latest/fleet/mdm/apple/bootstrap/%d", teamID)
	request := deleteBootstrapPackageRequest{}
	var responseBody deleteBootstrapPackageResponse
	err := c.authenticatedRequest(request, verb, path, &responseBody)
	return err
}

func (c *Client) UploadBootstrapPackage(pkg *fleet.MDMAppleBootstrapPackage) error {
	verb, path := "POST", "/api/latest/fleet/mdm/apple/bootstrap"

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

func extractFilenameFromPath(p string) string {
	u, err := url.Parse(p)
	if err != nil {
		return ""
	}

	invalid := map[string]struct{}{
		"":  {},
		".": {},
		"/": {},
	}

	b := path.Base(u.Path)
	if _, ok := invalid[b]; ok {
		return ""
	}

	if _, ok := invalid[path.Ext(b)]; ok {
		return b + ".pkg"
	}

	return b
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
		filename = extractFilenameFromPath(pkgURL)
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
			return nil, errors.New("Couldn’t edit bootstrap_package. The bootstrap_package must be signed. Learn how to sign the package in the Fleet documentation: https://fleetdm.com/docs/using-fleet/mdm-macos-setup#step-2-sign-the-package")
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
	if err := c.CheckMDMEnabled(); err != nil {
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
