package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/option"
)

// googleForwarder wraps an authenticated Google Android Management API client
// for forwarding requests targeting real devices.
type googleForwarder struct {
	svc *androidmanagement.Service
}

func newGoogleForwarder(credentialsJSON string) (*googleForwarder, error) {
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(credentialsJSON), androidmanagement.AndroidmanagementScope) //nolint:staticcheck // SA1019 -- load testing tool, credentials are from a trusted source
	if err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	svc, err := androidmanagement.NewService(ctx,
		option.WithCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("create android management service: %w", err)
	}

	return &googleForwarder{svc: svc}, nil
}

// ForwardDevicesGet forwards a GET .../devices/{id} request to Google.
func (g *googleForwarder) ForwardDevicesGet(w http.ResponseWriter, r *http.Request) {
	name := deviceName(r)
	device, err := g.svc.Enterprises.Devices.Get(name).Context(r.Context()).Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	writeJSON(w, device)
}

// ForwardDevicesPatch forwards a PATCH .../devices/{id} request to Google.
func (g *googleForwarder) ForwardDevicesPatch(w http.ResponseWriter, r *http.Request) {
	name := deviceName(r)
	var device androidmanagement.Device
	if err := readBody(r, &device); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := g.svc.Enterprises.Devices.Patch(name, &device).Context(r.Context()).Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	writeJSON(w, result)
}

// ForwardDevicesDelete forwards a DELETE .../devices/{id} request to Google.
func (g *googleForwarder) ForwardDevicesDelete(w http.ResponseWriter, r *http.Request) {
	name := deviceName(r)
	_, err := g.svc.Enterprises.Devices.Delete(name).Context(r.Context()).Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "{}")
}

// ForwardIssueCommand forwards a POST .../devices/{id}:issueCommand request to Google.
func (g *googleForwarder) ForwardIssueCommand(w http.ResponseWriter, r *http.Request) {
	name := deviceName(r)
	var cmd androidmanagement.Command
	if err := readBody(r, &cmd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	op, err := g.svc.Enterprises.Devices.IssueCommand(name, &cmd).Context(r.Context()).Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	writeJSON(w, op)
}

// ForwardDevicesList forwards a GET .../devices request to Google and returns all device names.
func (g *googleForwarder) ForwardDevicesList(enterpriseName string, ctx context.Context) ([]map[string]string, error) {
	var allDevices []map[string]string
	pageToken := ""

	for {
		call := g.svc.Enterprises.Devices.List(enterpriseName).Context(ctx).PageSize(100).Fields("nextPageToken", "devices/name")
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list devices from Google: %w", err)
		}
		for _, d := range resp.Devices {
			allDevices = append(allDevices, map[string]string{"name": d.Name})
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return allDevices, nil
}

// ForwardPoliciesPatch forwards a PATCH .../policies/{id} request to Google.
func (g *googleForwarder) ForwardPoliciesPatch(w http.ResponseWriter, r *http.Request) {
	name := policyName(r)
	var policy androidmanagement.Policy
	if err := readBody(r, &policy); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	call := g.svc.Enterprises.Policies.Patch(name, &policy).Context(r.Context())
	if mask := r.URL.Query().Get("updateMask"); mask != "" {
		call = call.UpdateMask(mask)
	}
	result, err := call.Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	writeJSON(w, result)
}

// ForwardEnrollmentTokenCreate forwards a POST .../enrollmentTokens request to Google.
func (g *googleForwarder) ForwardEnrollmentTokenCreate(w http.ResponseWriter, r *http.Request) {
	enterpriseName := "enterprises/" + r.PathValue("eid")
	var token androidmanagement.EnrollmentToken
	if err := readBody(r, &token); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := g.svc.Enterprises.EnrollmentTokens.Create(enterpriseName, &token).Context(r.Context()).Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	writeJSON(w, result)
}

// ForwardApplicationsGet forwards a GET .../applications/{package} request to Google.
func (g *googleForwarder) ForwardApplicationsGet(w http.ResponseWriter, r *http.Request) {
	name := "enterprises/" + r.PathValue("eid") + "/applications/" + r.PathValue("pkg")
	result, err := g.svc.Enterprises.Applications.Get(name).Context(r.Context()).Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	writeJSON(w, result)
}

// ForwardWebAppsCreate forwards a POST .../webApps request to Google.
func (g *googleForwarder) ForwardWebAppsCreate(w http.ResponseWriter, r *http.Request) {
	enterpriseName := "enterprises/" + r.PathValue("eid")
	var webApp androidmanagement.WebApp
	if err := readBody(r, &webApp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := g.svc.Enterprises.WebApps.Create(enterpriseName, &webApp).Context(r.Context()).Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	writeJSON(w, result)
}

// ForwardEnterprisesList forwards a GET /v1/enterprises request to Google.
func (g *googleForwarder) ForwardEnterprisesList(w http.ResponseWriter, r *http.Request) {
	resp, err := g.svc.Enterprises.List().Context(r.Context()).Do()
	if err != nil {
		writeGoogleError(w, err)
		return
	}
	writeJSON(w, resp)
}

// ---- helpers ----

func readBody(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeGoogleError(w http.ResponseWriter, err error) {
	log.Printf("googleForwarder: Google API error: %v", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    502,
			"message": err.Error(),
			"status":  "BAD_GATEWAY",
		},
	}) //nolint:errcheck
}
