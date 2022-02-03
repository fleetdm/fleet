package service

func (c *Client) CreateGlobalPolicy(name, query, description, resolution, platform string) error {
	req := globalPolicyRequest{
		Name:        name,
		Query:       query,
		Description: description,
		Resolution:  resolution,
		Platform:    platform,
	}
	verb, path := "POST", "/api/v1/fleet/global/policies"
	var responseBody globalPolicyResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
