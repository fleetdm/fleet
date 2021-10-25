package service

func (c *Client) CreatePolicy(queryID uint, resolution string) error {
	req := globalPolicyRequest{
		QueryID:    queryID,
		Resolution: resolution,
	}
	verb, path := "POST", "/api/v1/fleet/global/policies"
	var responseBody globalPolicyResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
