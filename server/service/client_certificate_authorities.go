package service

import "github.com/fleetdm/fleet/v4/server/fleet"

// GetCertificateAuthoritiesSpec fetches the certificate authorities stored on the server
func (c *Client) GetCertificateAuthoritiesSpec() (*fleet.CertificateAuthoritiesSpec, error) {
	// verb, path := "GET", "/api/latest/fleet/spec/certificate_authorities"
	// var responseBody getCertificateAuthoritiesSpecResponse
	// err := c.authenticatedRequest(nil, verb, path, &responseBody)
	// return responseBody.Spec, err

	// TODO(hca): use the new endpoints to get certs?
	return nil, nil
}

// ApplyCertificateAuthoritiesSpec applies the certificate authorities.
func (c *Client) ApplyCertificateAuthoritiesSpec(spec *fleet.CertificateAuthoritiesSpec, opts fleet.ApplySpecOptions) error {
	req := applyCertificateAuthoritiesSpecRequest{Spec: *spec}
	verb, path := "POST", "/api/latest/fleet/spec/certificate_authorities"
	var responseBody applyCertificateAuthoritiesSpecResponse
	return c.authenticatedRequestWithQuery(req, verb, path, &responseBody, opts.RawQuery())
}
