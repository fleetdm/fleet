package service

import "github.com/fleetdm/fleet/v4/server/fleet"

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
	err := c.authenticatedRequest(request, verb, path)
	return responseBody.AppleCSR, err
}
