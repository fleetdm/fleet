package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
)

// httpClient is used for all verify requests so slow or hung servers fail
// fast instead of blocking indefinitely.
var httpClient = &http.Client{Timeout: 30 * time.Second}

type checkStatus int

const (
	statusVerified checkStatus = iota
	statusPartial
	statusFailed
)

type checkResult struct {
	Label  string
	Status checkStatus
	Detail string
}

func newSpecValidator(specBytes []byte) (validator.Validator, error) {
	doc, err := libopenapi.NewDocument(specBytes)
	if err != nil {
		return nil, err
	}
	v, errs := validator.NewValidator(doc)
	if len(errs) > 0 {
		return nil, fmt.Errorf("building validator: %v", errs)
	}
	return v, nil
}

// checkEndpoint performs one request and validates the response body against
// the spec. body == nil means GET-style no payload.
func checkEndpoint(v validator.Validator, server, token, method, path string, body any) checkResult {
	label := method + " " + path
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return checkResult{label, statusFailed, err.Error()}
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, server+path, reader)
	if err != nil {
		return checkResult{label, statusFailed, err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return checkResult{label, statusFailed, err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return checkResult{label, statusFailed, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, payload)}
	}
	ok, verrs := v.ValidateHttpResponse(req, resp)
	if !ok {
		var msgs []string
		for _, e := range verrs {
			msgs = append(msgs, e.Message)
		}
		return checkResult{label, statusFailed, strings.Join(msgs, "; ")}
	}
	return checkResult{label, statusVerified, ""}
}

func runVerify(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	server := fs.String("server", "", "Fleet server URL, for example http://localhost:1337")
	token := fs.String("token", "", "API token (or use --email and --password)")
	email := fs.String("email", "", "admin email for login")
	password := fs.String("password", "", "admin password for login")
	specPath := fs.String("spec", "openapi.yml", "path to the generated spec")
	strict := fs.Bool("strict", false, "fail on partially verified endpoints")
	mdmHostUUID := fs.String("mdm-host-uuid", "", "UUID of an MDM-enrolled host to fully verify POST /commands/run")
	fs.Parse(args)

	if *server == "" {
		fmt.Fprintln(os.Stderr, "error: --server is required")
		return 1
	}
	specBytes, err := os.ReadFile(*specPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	v, err := newSpecValidator(specBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if *token == "" {
		*token, err = login(*server, *email, *password)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: login failed:", err)
			return 1
		}
	}

	results := runChecks(v, *server, *token, *mdmHostUUID)

	failed := false
	for _, r := range results {
		switch r.Status {
		case statusVerified:
			fmt.Printf("  ok       %s\n", r.Label)
		case statusPartial:
			fmt.Printf("  partial  %s (%s)\n", r.Label, r.Detail)
			if *strict {
				failed = true
			}
		case statusFailed:
			fmt.Printf("  FAIL     %s: %s\n", r.Label, r.Detail)
			failed = true
		}
	}
	if failed {
		return 1
	}
	return 0
}

func runChecks(v validator.Validator, server, token, mdmHostUUID string) []checkResult {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var results []checkResult
	add := func(r checkResult) { results = append(results, r) }

	// Writes first: they seed data so the list endpoints below return rows.
	add(checkEndpoint(v, server, token, "POST", "/api/v1/fleet/global/policies", map[string]any{
		"name":        "openapi-verify-policy-" + suffix,
		"query":       "SELECT 1;",
		"description": "Created by tools/openapi verify.",
		"resolution":  "Delete this policy.",
	}))
	add(checkEndpoint(v, server, token, "POST", "/api/v1/fleet/reports", map[string]any{
		"name":        "openapi-verify-report-" + suffix,
		"query":       "SELECT 1;",
		"description": "Created by tools/openapi verify.",
	}))
	// Seed a fleet so GET /fleets returns a row. POST /fleets is not in the
	// pilot spec, so this call is raw (not validated).
	if err := rawPost(server, token, "/api/v1/fleet/fleets", map[string]any{
		"name": "openapi-verify-" + suffix,
	}); err != nil {
		add(checkResult{"seed POST /api/v1/fleet/fleets", statusFailed, err.Error()})
	}

	add(checkEndpoint(v, server, token, "GET", "/api/v1/fleet/hosts", nil))
	add(hostByIDCheck(v, server, token))
	add(checkEndpoint(v, server, token, "GET", "/api/v1/fleet/software/titles", nil))
	add(checkEndpoint(v, server, token, "GET", "/api/v1/fleet/global/policies", nil))
	add(checkEndpoint(v, server, token, "GET", "/api/v1/fleet/reports", nil))
	add(checkEndpoint(v, server, token, "GET", "/api/v1/fleet/fleets", nil))
	add(checkEndpoint(v, server, token, "GET", "/api/v1/fleet/commands", nil))
	add(runCommandCheck(v, server, token, mdmHostUUID))
	return results
}

// hostByIDCheck resolves a real host ID from the list endpoint. With no
// enrolled hosts (plain CI) the endpoint can't be exercised: partial.
func hostByIDCheck(v validator.Validator, server, token string) checkResult {
	label := "GET /api/v1/fleet/hosts/{id}"
	var listing struct {
		Hosts []struct {
			ID int `json:"id"`
		} `json:"hosts"`
	}
	if err := rawGet(server, token, "/api/v1/fleet/hosts?per_page=1", &listing); err != nil {
		return checkResult{label, statusFailed, err.Error()}
	}
	if len(listing.Hosts) == 0 {
		return checkResult{label, statusPartial, "no enrolled hosts on this server"}
	}
	return checkEndpoint(v, server, token, "GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", listing.Hosts[0].ID), nil)
}

// runCommandCheck needs an MDM-enrolled host for the happy path. Without one
// it asserts Fleet's standard error envelope and reports partial.
func runCommandCheck(v validator.Validator, server, token, hostUUID string) checkResult {
	label := "POST /api/v1/fleet/commands/run"
	if hostUUID != "" {
		plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Command</key><dict><key>RequestType</key><string>DeviceInformation</string></dict>
  <key>CommandUUID</key><string>openapi-verify</string>
</dict></plist>`
		return checkEndpoint(v, server, token, "POST", "/api/v1/fleet/commands/run", map[string]any{
			"command":    base64.StdEncoding.EncodeToString([]byte(plist)),
			"host_uuids": []string{hostUUID},
		})
	}
	body := map[string]any{
		"command":    base64.StdEncoding.EncodeToString([]byte("<plist/>")),
		"host_uuids": []string{"00000000-0000-0000-0000-000000000000"},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", server+"/api/v1/fleet/commands/run", bytes.NewReader(b))
	if err != nil {
		return checkResult{label, statusFailed, err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return checkResult{label, statusFailed, err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		return checkResult{label, statusFailed, fmt.Sprintf("expected an error without an MDM host, got HTTP %d", resp.StatusCode)}
	}
	var envelope struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil || envelope.Message == "" {
		return checkResult{label, statusFailed, "error response is not Fleet's standard error envelope"}
	}
	return checkResult{label, statusPartial, "no MDM-enrolled host; verified error envelope only"}
}

func login(server, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", fmt.Errorf("provide --token, or both --email and --password")
	}
	b, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := httpClient.Post(server+"/api/v1/fleet/login", "application/json", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, payload)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Token, nil
}

func rawGet(server, token, path string, into any) error {
	req, err := http.NewRequest("GET", server+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GET %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func rawPost(server, token, path string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", server+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("POST %s: HTTP %d: %s", path, resp.StatusCode, payload)
	}
	return nil
}
