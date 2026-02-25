package client

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GetCertificateTemplates retrieves the list of Certificate Templates for a team.
func (c *Client) GetCertificateTemplates(teamID string) ([]*fleet.CertificateTemplateResponseSummary, error) {
	verb, path := "GET", "/api/latest/fleet/certificates"
	var responseBody fleet.ListCertificateTemplatesResponse
	query := "team_id=" + teamID
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Certificates, nil
}

// ApplyCertificateSpecs sends a list of certificate specs to the fleet instance to be added/updated.
func (c *Client) ApplyCertificateSpecs(specs []*fleet.CertificateRequestSpec) error {
	req := fleet.ApplyCertificateTemplateSpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/certificates"
	var responseBody fleet.ApplyCertificateTemplateSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// DeleteCertificateTemplates sends a list of certificate template IDs to be deleted.
func (c *Client) DeleteCertificateTemplates(certificateTemplateIDs []uint, teamID uint) error {
	verb, path := "DELETE", "/api/latest/fleet/spec/certificates"
	req := fleet.DeleteCertificateTemplateSpecsRequest{IDs: certificateTemplateIDs, TeamID: teamID}
	var responseBody fleet.DeleteCertificateTemplateSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
