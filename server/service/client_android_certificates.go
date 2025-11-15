package service

import "github.com/fleetdm/fleet/v4/server/fleet"

// ApplyCertificateSpecs sends a list of  certificate specs to the fleet instance.
func (c *Client) ApplyCertificateSpecs(specs []*fleet.CertificateRequestSpec) error {
	req := applyCertificateSpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/certificates"
	var responseBody applyCertificateSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
