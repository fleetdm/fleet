package service

import (
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// GetCertificateTemplates retrieves the list of Certificate Templates for a team.
func (c *Client) GetCertificateTemplates(teamID string) ([]*fleet.CertificateTemplateResponseSummary, error) {
	verb, path := "GET", "/api/latest/fleet/certificates"
	var responseBody listCertificateTemplatesResponse
	query := "fleet_id=" + teamID
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
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data, err = endpointer.RewriteOldToNewKeys(data, endpointer.ExtractAliasRules(req))
	if err != nil {
		return err
	}
	return c.authenticatedRequest(data, verb, path, &responseBody)
}

// DeleteCertificateTemplates sends a list of certificate template IDs to be deleted.
func (c *Client) DeleteCertificateTemplates(certificateTemplateIDs []uint, teamID uint) error {
	verb, path := "DELETE", "/api/latest/fleet/spec/certificates"
	req := deleteCertificateTemplateSpecsRequest{IDs: certificateTemplateIDs, TeamID: teamID}
	var responseBody deleteCertificateTemplateSpecsResponse
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data, err = endpointer.RewriteOldToNewKeys(data, endpointer.ExtractAliasRules(req))
	if err != nil {
		return err
	}
	return c.authenticatedRequest(data, verb, path, &responseBody)
}
