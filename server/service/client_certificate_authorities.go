package service

import "github.com/fleetdm/fleet/v4/server/fleet"

// GetCertificateAuthoritiesSpec fetches the certificate authorities stored on the server
func (c *Client) GetCertificateAuthoritiesSpec(includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
	req := getCertificateAuthoritiesSpecRequest{IncludeSecrets: includeSecrets}
	verb, path := "GET", "/api/latest/fleet/spec/certificate_authorities"
	var responseBody getCertificateAuthoritiesSpecResponse
	err := c.authenticatedRequest(req, verb, path, &responseBody)
	return responseBody.CertificateAuthorities, err
}

// ApplyCertificateAuthoritiesSpec applies the certificate authorities.
func (c *Client) ApplyCertificateAuthoritiesSpec(groupedCAs fleet.GroupedCertificateAuthorities, opts fleet.ApplySpecOptions) error {
	req := applyCertificateAuthoritiesSpecRequest{CertificateAuthorities: groupedCAs, DryRun: opts.DryRun}
	verb, path := "POST", "/api/latest/fleet/spec/certificate_authorities"
	var responseBody applyCertificateAuthoritiesSpecResponse
	return c.authenticatedRequestWithQuery(req, verb, path, &responseBody, opts.RawQuery())
}
