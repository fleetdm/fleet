package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GetCertificateTemplates retrieves the list of Certificate Templates for a team.
func (c *Client) GetCertificateTemplates(teamID string) ([]*fleet.CertificateTemplateResponseSummary, error) {
	verb, path := "GET", "/api/latest/fleet/certificates"
	var responseBody listCertificateTemplatesResponse
	query := "team_id=" + teamID
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Certificates, nil
}

// ApplyCertificateSpecs sends a list of certificate specs to the fleet instance to be added/updated.
func (c *Client) ApplyCertificateSpecs(specs []*fleet.CertificateRequestSpec) error {
	req := applyCertificateTemplateSpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/certificates"
	var responseBody applyCertificateTemplateSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// DeleteCertificateTemplates sends a list of certificate template IDs to be deleted.
func (c *Client) DeleteCertificateTemplates(certificateTemplateIDs []uint, teamID uint) error {
	verb, path := "DELETE", "/api/latest/fleet/spec/certificates"
	req := deleteCertificateTemplateSpecsRequest{IDs: certificateTemplateIDs, TeamID: teamID}
	var responseBody deleteCertificateTemplateSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
