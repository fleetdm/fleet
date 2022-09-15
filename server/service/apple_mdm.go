package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/micromdm/nanodep/client"
)

type createMDMAppleEnrollmentRequest struct {
	Name      string           `json:"name"`
	DEPConfig *json.RawMessage `json:"dep_config"`
}

type createMDMAppleEnrollmentResponse struct {
	ID  uint   `json:"enrollment_id"`
	URL string `json:"url"`
	Err error  `json:"error,omitempty"`
}

func (r createMDMAppleEnrollmentResponse) error() error { return r.Err }

func createMDMAppleEnrollmentEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createMDMAppleEnrollmentRequest)
	enrollment, url, err := svc.NewMDMAppleEnrollment(ctx, fleet.MDMAppleEnrollmentPayload{
		Name:      req.Name,
		DEPConfig: req.DEPConfig,
	})
	if err != nil {
		return createMDMAppleEnrollmentResponse{
			Err: err,
		}, nil
	}
	return createMDMAppleEnrollmentResponse{
		ID:  enrollment.ID,
		URL: url,
	}, nil
}

func (svc *Service) NewMDMAppleEnrollment(ctx context.Context, enrollmentPayload fleet.MDMAppleEnrollmentPayload) (*fleet.MDMAppleEnrollment, string, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEnrollment{}, fleet.ActionWrite); err != nil {
		return nil, "", ctxerr.Wrap(ctx, err)
	}

	enrollment, err := svc.ds.NewMDMAppleEnrollment(ctx, enrollmentPayload)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err)
	}
	if enrollment.DEPConfig != nil {
		if err := svc.setDEPProfile(ctx, enrollment); err != nil {
			return nil, "", ctxerr.Wrap(ctx, err)
		}
	}
	return enrollment, svc.mdmAppleEnrollURL(enrollment.ID), nil
}

func (svc *Service) mdmAppleEnrollURL(enrollmentID uint) string {
	// TODO(lucas): Define /mdm/apple/api/enroll path somewhere else.
	return fmt.Sprintf("https://%s/mdm/apple/api/enroll?id=%d", svc.config.MDMApple.ServerAddress, enrollmentID)
}

// setDEPProfile define a "DEP profile" on https://mdmenrollment.apple.com and
// sets the returned Profile UUID as the current DEP profile to apply to newly sync DEP devices.
func (svc *Service) setDEPProfile(ctx context.Context, enrollment *fleet.MDMAppleEnrollment) error {
	httpClient := fleethttp.NewClient()
	depTransport := client.NewTransport(httpClient.Transport, httpClient, svc.depStorage, nil)
	depClient := client.NewClient(fleethttp.NewClient(), depTransport)

	// TODO(lucas): Currently overriding the `url` and `configuration_web_url`.
	// We need to actually expose configuration.
	var depProfileRequest map[string]interface{}
	if err := json.Unmarshal(*enrollment.DEPConfig, &depProfileRequest); err != nil {
		return fmt.Errorf("invalid DEP profile: %w", err)
	}
	enrollURL := svc.mdmAppleEnrollURL(enrollment.ID)
	depProfileRequest["url"] = enrollURL
	depProfileRequest["configuration_web_url"] = enrollURL
	depProfile, err := json.Marshal(depProfileRequest)
	if err != nil {
		return fmt.Errorf("reserializing DEP profile: %w", err)
	}

	defineProfileRequest, err := client.NewRequestWithContext(
		ctx, apple.DEPName, svc.depStorage, "POST", "/profile", bytes.NewReader(depProfile),
	)
	if err != nil {
		return fmt.Errorf("create profile request: %w", err)
	}
	defineProfileHTTPResponse, err := depClient.Do(defineProfileRequest)
	if err != nil {
		return fmt.Errorf("exec profile request: %w", err)
	}
	defer defineProfileHTTPResponse.Body.Close()
	if defineProfileHTTPResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("profile request: %s", defineProfileHTTPResponse.Status)
	}
	defineProfileResponseBody, err := io.ReadAll(defineProfileHTTPResponse.Body)
	if err != nil {
		return fmt.Errorf("read profile response: %w", err)
	}
	type depProfileResponseFields struct {
		ProfileUUID string `json:"profile_uuid"`
	}
	defineProfileResponse := depProfileResponseFields{}
	if err := json.Unmarshal(defineProfileResponseBody, &defineProfileResponse); err != nil {
		return fmt.Errorf("parse profile response: %w", err)
	}
	if err := svc.depStorage.StoreAssignerProfile(
		ctx, apple.DEPName, defineProfileResponse.ProfileUUID,
	); err != nil {
		return fmt.Errorf("set profile UUID: %w", err)
	}
	return nil
}

type getMDMAppleCommandResultsRequest struct {
	CommandUUID string `query:"command_uuid,optional"`
}

type getMDMAppleCommandResultsResponse struct {
	Results map[string]*fleet.MDMAppleCommandResult `json:"results,omitempty"`
	Err     error                                   `json:"error,omitempty"`
}

func (r getMDMAppleCommandResultsResponse) error() error { return r.Err }

func getMDMAppleCommandResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getMDMAppleCommandResultsRequest)
	results, err := svc.GetMDMAppleCommandResults(ctx, req.CommandUUID)
	if err != nil {
		return getMDMAppleCommandResultsResponse{
			Err: err,
		}, nil
	}

	return getMDMAppleCommandResultsResponse{
		Results: results,
	}, nil
}

func (svc *Service) GetMDMAppleCommandResults(ctx context.Context, commandUUID string) (map[string]*fleet.MDMAppleCommandResult, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleCommandResult{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	results, err := svc.ds.GetMDMAppleCommandResults(ctx, commandUUID)
	if err != nil {
		return nil, err
	}

	return results, nil
}
