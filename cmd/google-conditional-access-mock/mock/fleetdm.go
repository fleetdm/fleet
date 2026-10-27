package mock

import (
	"bytes"
	"cmp"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type createRequest struct {
	CustomerID string `json:"googleCustomerId"`
}

type createResponse struct {
	FleetServerSecret string `json:"fleet_server_secret"`
	OAuthClientID     string `json:"oauth_client_id"`
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		writeError(w, http.StatusBadRequest, "missing Origin header")
		return
	}
	if req.CustomerID == "" {
		writeError(w, http.StatusBadRequest, "missing googleCustomerId")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.registrations[req.CustomerID]; ok {
		writeError(w, http.StatusConflict, "a registration already exists for this customer")
		return
	}
	reg := &Registration{
		CustomerID:        req.CustomerID,
		Origin:            origin,
		FleetServerSecret: randomSecret(30),
		OAuthClientID:     cmp.Or(s.cfg.GoogleClientID, defaultOAuthClientID),
	}
	s.registrations[req.CustomerID] = reg
	s.logger.InfoContext(r.Context(), "registered Fleet server", "customer_id", reg.CustomerID, "origin", origin)
	writeJSON(w, http.StatusOK, createResponse{FleetServerSecret: reg.FleetServerSecret, OAuthClientID: reg.OAuthClientID})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("googleCustomerId")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.registrationLocked(customerID, r.URL.Query().Get("fleetServerSecret")) == nil {
		writeError(w, http.StatusNotFound, "no registration for this customer and secret")
		return
	}
	delete(s.registrations, customerID)
	s.logger.InfoContext(r.Context(), "deleted registration", "customer_id", customerID)
	writeJSON(w, http.StatusOK, struct{}{})
}

// registrationLocked returns the registration matching the customer and secret, or nil.
// Callers hold s.mu.
func (s *Server) registrationLocked(customerID, secret string) *Registration {
	reg, ok := s.registrations[customerID]
	if !ok || subtle.ConstantTimeCompare([]byte(reg.FleetServerSecret), []byte(secret)) != 1 {
		return nil
	}
	return reg
}

func (s *Server) registration(customerID string) (Registration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reg, ok := s.registrations[customerID]
	if !ok {
		return Registration{}, false
	}
	return *reg, true
}

func (s *Server) handleSignIn(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	customerID, token, hostUUID, platform := q.Get("googleCustomerId"), q.Get("token"), q.Get("hostUuid"), q.Get("platform")
	if _, ok := s.registration(customerID); !ok {
		http.Error(w, "unknown customer", http.StatusNotFound)
		return
	}
	if token == "" || hostUUID == "" {
		http.Error(w, "missing token or hostUuid", http.StatusBadRequest)
		return
	}
	if platform != "ios" && platform != "ipados" {
		http.Error(w, "platform must be ios or ipados", http.StatusBadRequest)
		return
	}

	// The state is a server-side nonce, so the token never reaches Google.
	state := rand.Text()
	redirectURL := s.baseURL(r) + "/api/v1/google-conditional-access/callback"
	s.mu.Lock()
	now := time.Now()
	for k, v := range s.signIns {
		if now.After(v.expiresAt) {
			delete(s.signIns, k)
		}
	}
	s.signIns[state] = &signIn{
		customerID:  customerID,
		token:       token,
		hostUUID:    hostUUID,
		redirectURL: redirectURL,
		expiresAt:   now.Add(signInTTL),
	}
	s.mu.Unlock()

	if s.forwarding() {
		http.Redirect(w, r, s.oauthConfig(redirectURL).AuthCodeURL(state), http.StatusFound)
		return
	}
	renderHTML(w, http.StatusOK, fakeSignInPage, map[string]string{"State": state, "Email": fakeEmail})
}

// takeSignIn returns and forgets the sign-in for state, so a replayed callback fails.
func (s *Server) takeSignIn(state string) (*signIn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	si, ok := s.signIns[state]
	delete(s.signIns, state)
	if !ok || time.Now().After(si.expiresAt) {
		return nil, false
	}
	return si, true
}

const (
	pageDone       = "done"
	pageNotAllowed = "not-allowed"
	pageError      = "error"
)

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	page := s.callback(r)
	http.Redirect(w, r, "/google-conditional-access/"+page, http.StatusSeeOther)
}

func (s *Server) callback(r *http.Request) string {
	ctx := r.Context()
	q := r.URL.Query()
	si, ok := s.takeSignIn(q.Get("state"))
	if !ok {
		s.logger.WarnContext(ctx, "callback with unknown or expired state")
		return pageError
	}
	if e := q.Get("error"); e != "" {
		s.logger.WarnContext(ctx, "Google sign-in failed", "error", e)
		return pageError
	}
	reg, ok := s.registration(si.customerID)
	if !ok {
		s.logger.WarnContext(ctx, "registration deleted during sign-in", "customer_id", si.customerID)
		return pageError
	}

	dev := Device{CustomerID: si.customerID, HostUUID: si.hostUUID}
	if s.forwarding() {
		found, err := s.lookupDevice(ctx, q.Get("code"), si.redirectURL, lookupIOS, si.hostUUID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Google device lookup failed", "host_uuid", si.hostUUID, "err", err)
			return pageError
		}
		dev.DeviceID, dev.DeviceUserName, dev.Email = found.deviceID, found.deviceUserName, found.email
	} else {
		dev.DeviceID, dev.DeviceUserName = fakeDevice(si.hostUUID)
		dev.Email = cmp.Or(q.Get("email"), fakeEmail)
	}
	dev.Name = "devices/" + dev.DeviceID
	s.upsertDevice(dev)

	status, err := s.registerWithFleet(r, reg, si.token, dev)
	if err != nil {
		s.logger.ErrorContext(ctx, "calling Fleet server failed", "err", err)
		return pageError
	}
	s.logger.InfoContext(ctx, "Fleet server answered register", "status", status, "host_uuid", si.hostUUID, "google_device_id", dev.DeviceID)
	switch status {
	case http.StatusOK:
		return pageDone
	case http.StatusForbidden, http.StatusConflict:
		return pageNotAllowed
	default:
		return pageError
	}
}

type fleetRegisterRequest struct {
	Token                string `json:"token"`
	GoogleDeviceID       string `json:"google_device_id"`
	GoogleDeviceUserName string `json:"google_device_user_name"`
	GoogleEmail          string `json:"google_email"`
}

// registerWithFleet makes the inbound call to the Fleet server and returns its status.
func (s *Server) registerWithFleet(r *http.Request, reg Registration, token string, dev Device) (int, error) {
	body, err := json.Marshal(fleetRegisterRequest{
		Token:                token,
		GoogleDeviceID:       dev.DeviceID,
		GoogleDeviceUserName: dev.DeviceUserName,
		GoogleEmail:          dev.Email,
	})
	if err != nil {
		return 0, err
	}
	fleetURL := strings.TrimSuffix(cmp.Or(s.cfg.FleetServerURL, reg.Origin), "/")
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		fleetURL+"/api/v1/fleet/google_conditional_access/register", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+reg.FleetServerSecret)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

type androidRegisterRequest struct {
	CustomerID        string `json:"googleCustomerId"`
	FleetServerSecret string `json:"fleetServerSecret"`
	ServerAuthCode    string `json:"serverAuthCode"`
	AndroidID         string `json:"androidId"`
	HostUUID          string `json:"hostUuid"`
}

type androidRegisterResponse struct {
	GoogleDeviceID       string `json:"google_device_id"`
	GoogleDeviceUserName string `json:"google_device_user_name"`
	GoogleEmail          string `json:"google_email"`
}

func (s *Server) handleAndroidRegister(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req androidRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	s.mu.Lock()
	reg := s.registrationLocked(req.CustomerID, req.FleetServerSecret)
	_, noRecord := s.androidNoRecord[req.AndroidID]
	s.mu.Unlock()
	if reg == nil {
		writeError(w, http.StatusNotFound, "no registration for this customer and secret")
		return
	}
	if req.ServerAuthCode == "" || req.AndroidID == "" || req.HostUUID == "" {
		writeError(w, http.StatusBadRequest, "serverAuthCode, androidId, and hostUuid are required")
		return
	}

	dev := Device{CustomerID: req.CustomerID, HostUUID: req.HostUUID, AndroidID: req.AndroidID}
	switch {
	case noRecord:
		writeError(w, http.StatusUnprocessableEntity, errNoRecord.Error())
		return
	case s.forwarding():
		// Server auth codes from the Android app are exchanged with an empty redirect URI.
		found, err := s.lookupDevice(ctx, req.ServerAuthCode, "", lookupAndroid, req.AndroidID)
		if errors.Is(err, errNoRecord) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if err != nil {
			s.logger.ErrorContext(ctx, "Google device lookup failed", "android_id", req.AndroidID, "err", err)
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		dev.DeviceID, dev.DeviceUserName, dev.Email = found.deviceID, found.deviceUserName, found.email
	default:
		dev.DeviceID, dev.DeviceUserName = fakeDevice(req.HostUUID)
		dev.Email = fakeEmail
	}
	dev.Name = "devices/" + dev.DeviceID
	s.upsertDevice(dev)
	s.logger.InfoContext(ctx, "registered Android device", "host_uuid", req.HostUUID, "google_device_id", dev.DeviceID)
	writeJSON(w, http.StatusOK, androidRegisterResponse{
		GoogleDeviceID:       dev.DeviceID,
		GoogleDeviceUserName: dev.DeviceUserName,
		GoogleEmail:          dev.Email,
	})
}

// resultPages carries the end-user copy fleetdm.com shows after sign-in.
var resultPages = map[string]string{
	pageDone:       "You're done! 🎉 You can log in to apps with your company email.",
	pageNotAllowed: "This device isn't allowed to access company data. Only company-owned devices that are enrolled to Fleet are allowed.",
	pageError:      "Something went wrong. Try again.",
}

func (s *Server) handleResultPage(w http.ResponseWriter, r *http.Request) {
	text, ok := resultPages[r.PathValue("page")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	renderHTML(w, http.StatusOK, resultPage, map[string]string{"Text": text})
}

var fakeSignInPage = template.Must(template.New("sign-in").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Google sign-in (mock)</title></head>
<body style="font-family: sans-serif; padding: 24px">
<h1>Google sign-in (mock)</h1>
<form method="get" action="callback">
<input type="hidden" name="state" value="{{.State}}">
<input type="hidden" name="code" value="mock-code">
<label>Email <input type="email" name="email" value="{{.Email}}"></label>
<button type="submit">Continue</button>
</form>
<script>document.forms[0].submit();</script>
</body></html>
`))

var resultPage = template.Must(template.New("result").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Fleet</title></head>
<body style="font-family: sans-serif; padding: 24px"><p>{{.Text}}</p></body></html>
`))
