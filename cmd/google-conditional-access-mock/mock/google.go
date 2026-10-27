package mock

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/option"
)

const lookupScope = "https://www.googleapis.com/auth/cloud-identity.devices.lookup"

// handleServiceAccountKey returns a freshly generated service account key whose token_uri
// is the mock, so a Fleet server connected with it mints tokens here.
func (s *Server) handleServiceAccountKey(w http.ResponseWriter, r *http.Request) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"type":           "service_account",
		"project_id":     "fleet-mock",
		"private_key_id": randomSecret(40),
		"private_key":    string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"client_email":   "fleet-mock@fleet-mock.iam.gserviceaccount.com",
		"client_id":      "100000000000000000000",
		"auth_uri":       "https://accounts.google.com/o/oauth2/auth",
		"token_uri":      s.baseURL(r) + "/token",
	})
}

// handleToken answers the JWT bearer grant the service account flow sends. The assertion's
// signature isn't checked; its subject is recorded as the admin the token acts as.
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, "invalid_request", err.Error())
		return
	}
	if gt := r.PostForm.Get("grant_type"); gt != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
		writeOAuthError(w, "unsupported_grant_type", gt)
		return
	}
	parts := strings.Split(r.PostForm.Get("assertion"), ".")
	if len(parts) != 3 {
		writeOAuthError(w, "invalid_grant", "assertion is not a JWT")
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		writeOAuthError(w, "invalid_grant", "assertion payload: "+err.Error())
		return
	}
	var claims struct {
		Iss string `json:"iss"`
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		writeOAuthError(w, "invalid_grant", "assertion claims: "+err.Error())
		return
	}
	subject := claims.Sub
	if subject == "" {
		subject = claims.Iss
	}

	token := "mock-access-token-" + rand.Text()
	s.mu.Lock()
	s.tokens[token] = subject
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

// authed rejects Cloud Identity requests without a token the mock issued, and passes the
// token's subject on.
func (s *Server) authed(next func(w http.ResponseWriter, r *http.Request, subject string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		s.mu.Lock()
		subject, known := s.tokens[token]
		s.mu.Unlock()
		if !ok || !known {
			writeGoogleError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Request had invalid authentication credentials.")
			return
		}
		next(w, r, subject)
	}
}

// customerParam returns the customer ID from the customer query parameter, without the
// "customers/" prefix.
func customerParam(r *http.Request) string {
	return strings.TrimPrefix(r.URL.Query().Get("customer"), "customers/")
}

type googleDevice struct {
	Name       string `json:"name"`
	DeviceID   string `json:"deviceId"`
	DeviceType string `json:"deviceType"`
}

func toGoogleDevice(d *Device) googleDevice {
	deviceType := "IOS"
	if d.AndroidID != "" {
		deviceType = "ANDROID"
	}
	return googleDevice{Name: d.Name, DeviceID: d.DeviceID, DeviceType: deviceType}
}

// handleDevicesList serves the Fleet server's delegation check. It answers 403 for an
// unknown customer so both outcomes of the check can be exercised.
func (s *Server) handleDevicesList(w http.ResponseWriter, r *http.Request, _ string) {
	customerID := customerParam(r)
	s.mu.Lock()
	known := s.knownCustomer(customerID)
	s.mu.Unlock()
	if !known {
		writeGoogleError(w, http.StatusForbidden, "PERMISSION_DENIED", "The caller does not have permission")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"devices": []googleDevice{{Name: "devices/mock-delegation-check", DeviceID: "mock-delegation-check", DeviceType: "IOS"}},
	})
}

// lookupLiveDevice returns the device for a Cloud Identity request, or writes the error
// Google would answer with. Callers hold s.mu.
func (s *Server) lookupLiveDevice(w http.ResponseWriter, r *http.Request) (*Device, bool) {
	customerID := customerParam(r)
	if !s.knownCustomer(customerID) {
		writeGoogleError(w, http.StatusForbidden, "PERMISSION_DENIED", "The caller does not have permission")
		return nil, false
	}
	d, ok := s.devices[r.PathValue("device")]
	if !ok || d.Stale || (customerID != "" && customerID != d.CustomerID) {
		writeGoogleError(w, http.StatusNotFound, "NOT_FOUND", "Requested entity was not found.")
		return nil, false
	}
	return d, true
}

func (s *Server) handleDevicesGet(w http.ResponseWriter, r *http.Request, _ string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.lookupLiveDevice(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toGoogleDevice(d))
}

type googleClientState struct {
	Name     string `json:"name,omitempty"`
	Managed  string `json:"managed,omitempty"`
	CustomID string `json:"customId,omitempty"`
}

// handleClientStatePatch records the fields in updateMask (all of them when it's empty) and
// answers with a completed operation, like Google does.
func (s *Server) handleClientStatePatch(w http.ResponseWriter, r *http.Request, subject string) {
	var body googleClientState
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeGoogleError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid json: "+err.Error())
		return
	}
	setManaged, setCustomID := true, true
	if m := r.URL.Query().Get("updateMask"); m != "" {
		setManaged, setCustomID = false, false
		for f := range strings.SplitSeq(m, ",") {
			switch strings.TrimSpace(f) {
			case "managed":
				setManaged = true
			case "customId", "custom_id":
				setCustomID = true
			default:
				writeGoogleError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "unsupported update mask field "+f)
				return
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.lookupLiveDevice(w, r)
	if !ok {
		return
	}
	if d.DeviceUserName != "devices/"+r.PathValue("device")+"/deviceUsers/"+r.PathValue("user") {
		writeGoogleError(w, http.StatusNotFound, "NOT_FOUND", "Requested entity was not found.")
		return
	}
	name := d.DeviceUserName + "/clientStates/" + r.PathValue("state")
	cs, ok := s.clientStates[name]
	if !ok {
		cs = &ClientState{Name: name}
		s.clientStates[name] = cs
	}
	if setManaged {
		cs.Managed = body.Managed
	}
	if setCustomID {
		cs.CustomID = body.CustomID
	}
	cs.UpdatedBy = subject
	cs.UpdatedAt = time.Now().UTC()

	writeJSON(w, http.StatusOK, map[string]any{
		"name": "operations/mock-" + rand.Text(),
		"done": true,
		"response": map[string]string{
			"@type":    "type.googleapis.com/google.apps.cloudidentity.devices.v1.ClientState",
			"name":     cs.Name,
			"managed":  cs.Managed,
			"customId": cs.CustomID,
		},
	})
}

// ---- Forwarding mode ----

func (s *Server) oauthConfig(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GoogleClientID,
		ClientSecret: s.cfg.GoogleClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{lookupScope},
	}
}

type lookupKind int

const (
	lookupIOS lookupKind = iota
	lookupAndroid
)

var errNoRecord = errors.New("Google has no device record for this end user and device")

type foundDevice struct {
	deviceID       string
	deviceUserName string
	email          string
}

// lookupDevice exchanges the end user's code with Fleet's real client and finds their
// device in Google, as fleetdm.com does.
func (s *Server) lookupDevice(ctx context.Context, code, redirectURL string, kind lookupKind, id string) (*foundDevice, error) {
	cfg := s.oauthConfig(redirectURL)
	var opts []oauth2.AuthCodeOption
	if redirectURL == "" {
		opts = append(opts, oauth2.SetAuthURLParam("redirect_uri", ""))
	}
	tok, err := cfg.Exchange(context.WithValue(ctx, oauth2.HTTPClient, s.httpClient), code, opts...)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	svc, err := cloudidentity.NewService(ctx, option.WithTokenSource(cfg.TokenSource(ctx, tok)))
	if err != nil {
		return nil, fmt.Errorf("cloud identity client: %w", err)
	}

	call := svc.Devices.DeviceUsers.Lookup("devices/-/deviceUsers").UserId("me")
	switch kind {
	case lookupIOS:
		call = call.IosDeviceId(id).Partner(s.cfg.PartnerID)
	case lookupAndroid:
		call = call.AndroidId(id)
	}
	res, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("deviceUsers.lookup: %w", err)
	}
	if len(res.Names) == 0 {
		return nil, errNoRecord
	}
	deviceUserName := res.Names[0]
	deviceName, _, _ := strings.Cut(deviceUserName, "/deviceUsers/")

	getDevice := svc.Devices.Get(deviceName)
	if res.Customer != "" {
		getDevice = getDevice.Customer(res.Customer)
	}
	dev, err := getDevice.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("devices.get: %w", err)
	}

	// The email is informational; the lookup scope may not cover reading it.
	var email string
	getUser := svc.Devices.DeviceUsers.Get(deviceUserName)
	if res.Customer != "" {
		getUser = getUser.Customer(res.Customer)
	}
	if du, err := getUser.Context(ctx).Do(); err == nil {
		email = du.UserEmail
	} else {
		s.logger.WarnContext(ctx, "reading device user email failed", "device_user", deviceUserName, "err", err)
	}

	return &foundDevice{deviceID: dev.DeviceId, deviceUserName: deviceUserName, email: email}, nil
}
