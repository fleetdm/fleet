package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GetCertificateTemplates retrieves the list of Certificate Templates for a team.
func (c *Client) GetCertificateTemplates(teamID string) ([]*fleet.CertificateTemplateResponseSummary, error) {
	verb, path := "GET", "/api/latest/fleet/certificates"
	query := "fleet_id=" + teamID
	// "id" is the only order key the endpoint allows.
	return listAllPages(query, "id", func(query string) ([]*fleet.CertificateTemplateResponseSummary, error) {
		var responseBody listCertificateTemplatesResponse
		if err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query); err != nil {
			return nil, err
		}
		return responseBody.Certificates, nil
	})
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
