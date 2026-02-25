package client

import "github.com/fleetdm/fleet/v4/server/fleet"

// GetCertificateAuthoritiesSpec fetches the certificate authorities stored on the server
func (c *Client) GetCertificateAuthoritiesSpec(includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
	verb, path := "GET", "/api/latest/fleet/spec/certificate_authorities"
	var responseBody fleet.GetCertificateAuthoritiesSpecResponse
	query := ""
	if includeSecrets {
		query = "include_secrets=true"
	}
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	return responseBody.CertificateAuthorities, err
}

// ApplyCertificateAuthoritiesSpec applies the certificate authorities.
func (c *Client) ApplyCertificateAuthoritiesSpec(groupedCAs fleet.GroupedCertificateAuthorities, opts fleet.ApplySpecOptions) error {
	req := fleet.BatchApplyCertificateAuthoritiesRequest{CertificateAuthorities: groupedCAs, DryRun: opts.DryRun}
	verb, path := "POST", "/api/latest/fleet/spec/certificate_authorities"
	var responseBody fleet.BatchApplyCertificateAuthoritiesResponse
	return c.authenticatedRequestWithQuery(req, verb, path, &responseBody, opts.RawQuery())
}

// GetCertificateAuthorities fetches the list of certificate authorities
func (c *Client) GetCertificateAuthorities() ([]*fleet.CertificateAuthoritySummary, error) {
	verb, path := "GET", "/api/latest/fleet/certificate_authorities"
	var responseBody fleet.ListCertificateAuthoritiesResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.CertificateAuthorities, err
}
