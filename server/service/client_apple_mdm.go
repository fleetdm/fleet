package service

import (
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (c *Client) CreateEnrollment(name string, depConfig *json.RawMessage) (*fleet.MDMAppleEnrollment, string, error) {
	request := createMDMAppleEnrollmentRequest{
		Name:      name,
		DEPConfig: depConfig,
	}
	var response createMDMAppleEnrollmentResponse
	if err := c.authenticatedRequest(request, "POST", "/api/latest/fleet/mdm/apple/enrollments", &response); err != nil {
		return nil, "", fmt.Errorf("request: %w", err)
	}
	return &fleet.MDMAppleEnrollment{
		ID:        response.ID,
		Name:      name,
		DEPConfig: depConfig,
	}, response.URL, nil
}
