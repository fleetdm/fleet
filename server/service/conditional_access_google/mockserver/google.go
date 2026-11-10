package mockserver

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Device states the admin route can set.
const (
	DeviceActive = "active"
	// DeviceDeleted is a record deleted in Google: calls on it are a 404.
	DeviceDeleted = "deleted"
	// DeviceLeft is a device that left the customer's Workspace: calls on it are a 403.
	DeviceLeft = "left"
)

type device struct {
	id           string // resource ID, the "devices/{id}" part of the names
	customerID   string // bare
	deviceID     string // Google's deviceId field: the Android Management API device ID on Android
	deviceType   string // IOS, ANDROID
	model        string
	osVersion    string
	state        string
	lastSyncTime time.Time
	users        []*deviceUser
}

type deviceUser struct {
	id           string
	email        string
	clientStates map[string]*ClientState // by client state ID, "{bare customer ID}-fleet"
}

// ClientState is what a client state patch recorded.
type ClientState struct {
	Managed    string    `json:"managed,omitempty"`
	CustomID   string    `json:"customId,omitempty"`
	UpdateTime time.Time `json:"updateTime"`
}

// Device is a device record in the state dump.
type Device struct {
	Name         string       `json:"name"`
	CustomerID   string       `json:"customer_id"`
	DeviceID     string       `json:"device_id"`
	DeviceType   string       `json:"device_type"`
	Model        string       `json:"model"`
	OSVersion    string       `json:"os_version"`
	State        string       `json:"state"`
	LastSyncTime time.Time    `json:"last_sync_time"`
	Users        []DeviceUser `json:"users"`
}

// DeviceUser is a DeviceUser in the state dump.
type DeviceUser struct {
	Name         string                  `json:"name"`
	Email        string                  `json:"email"`
	ClientStates map[string]*ClientState `json:"client_states"`
}

func (d *device) name() string { return "devices/" + d.id }

func (d *device) userName(u *deviceUser) string { return d.name() + "/deviceUsers/" + u.id }

func (d *device) export() Device {
	out := Device{
		Name: d.name(), CustomerID: d.customerID, DeviceID: d.deviceID, DeviceType: d.deviceType,
		Model: d.model, OSVersion: d.osVersion, State: d.state, LastSyncTime: d.lastSyncTime,
	}
	for _, u := range d.users {
		cs := map[string]*ClientState{}
		for k, v := range u.clientStates {
			c := *v
			cs[k] = &c
		}
		out.Users = append(out.Users, DeviceUser{Name: d.userName(u), Email: u.email, ClientStates: cs})
	}
	return out
}

func (d *device) user(id string) *deviceUser {
	for _, u := range d.users {
		if u.id == id {
			return u
		}
	}
	return nil
}

func sortState(st *State) {
	slices.SortFunc(st.Registrations, func(a, b Registration) int { return strings.Compare(a.CustomerID, b.CustomerID) })
	slices.Sort(st.Customers)
	slices.SortFunc(st.Devices, func(a, b Device) int { return strings.Compare(a.Name, b.Name) })
}

func (s *Server) googleRoutes() {
	s.mux.HandleFunc("POST /token", s.simulate(s.handleToken, failToken))
	s.mux.HandleFunc("GET /v1/devices", s.google(s.handleListDevices))
	s.mux.HandleFunc("GET /v1/devices/{device}", s.google(s.handleGetDevice))
	s.mux.HandleFunc("GET /v1/devices/{device}/deviceUsers", s.google(s.handleListDeviceUsers))
	s.mux.HandleFunc("PATCH /v1/devices/{device}/deviceUsers/{user}/clientStates/{state}", s.google(s.handlePatchClientState))

	s.mux.HandleFunc("POST /_mock/google/customers", s.admin(s.handleAddCustomer))
	s.mux.HandleFunc("POST /_mock/google/devices", s.admin(s.handleSeedDevice))
	s.mux.HandleFunc("POST /_mock/google/devices/{device}/state", s.admin(s.handleSetDeviceState))
	s.mux.HandleFunc("POST /_mock/google/token-error", s.admin(s.handleSetTokenError))
}

// handleToken mints a delegated token for a service account's signed JWT assertion. The
// signature isn't checked: any key JSON works.
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	assertion := r.PostForm.Get("assertion")
	if r.PostForm.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" || assertion == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "expected a JWT bearer assertion"})
		return
	}
	s.mu.Lock()
	tokenErr := s.tokenError
	s.mu.Unlock()
	if tokenErr != "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": tokenErr, "error_description": "simulated by the mock"})
		return
	}
	var claims struct {
		Iss   string `json:"iss"`
		Sub   string `json:"sub"`
		Scope string `json:"scope"`
	}
	jwtClaims(assertion, &claims)
	s.logger.InfoContext(r.Context(), "minted delegated token", "service_account", claims.Iss, "subject", claims.Sub, "scope", claims.Scope)
	writeJSON(w, http.StatusOK, map[string]any{"access_token": "mock-" + randomHex(16), "token_type": "Bearer", "expires_in": 3600})
}

// google checks the bearer token and the customer, records the call, and serves it
// under the lock.
func (s *Server) google(next func(w http.ResponseWriter, r *http.Request, customer string)) http.HandlerFunc {
	return s.simulate(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			writeGoogleError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Request is missing required authentication credential.")
			return
		}
		customer := bareCustomerID(r.URL.Query().Get("customer"))
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.calls) >= maxCalls {
			s.calls = s.calls[len(s.calls)-maxCalls+1:]
		}
		s.calls = append(s.calls, r.Method+" "+r.URL.Path)
		if _, ok := s.customers[customer]; !ok {
			writeGoogleError(w, http.StatusForbidden, "PERMISSION_DENIED", "The caller does not have permission")
			return
		}
		next(w, r, customer)
	}, failGoogleAPI)
}

type apiDevice struct {
	Name         string `json:"name"`
	DeviceID     string `json:"deviceId,omitempty"`
	DeviceType   string `json:"deviceType,omitempty"`
	Model        string `json:"model,omitempty"`
	OSVersion    string `json:"osVersion,omitempty"`
	OwnerType    string `json:"ownerType"`
	LastSyncTime string `json:"lastSyncTime,omitempty"`
}

func (d *device) api() apiDevice {
	return apiDevice{
		Name: d.name(), DeviceID: d.deviceID, DeviceType: d.deviceType, Model: d.model, OSVersion: d.osVersion,
		OwnerType: "COMPANY", LastSyncTime: d.lastSyncTime.UTC().Format(time.RFC3339),
	}
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request, customer string) {
	q := r.URL.Query()
	// Only the filter Fleet sends is supported: the records with a given deviceId.
	filter := q.Get("filter")
	deviceID, byID := strings.CutPrefix(filter, "id:")
	if filter != "" && !byID {
		writeGoogleError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "the mock only supports the id: filter")
		return
	}
	var matched []*device
	for _, d := range s.devices {
		if d.customerID == customer && d.state != DeviceDeleted && (!byID || d.deviceID == deviceID) {
			matched = append(matched, d)
		}
	}
	slices.SortFunc(matched, func(a, b *device) int { return strings.Compare(a.id, b.id) })

	offset, _ := strconv.Atoi(q.Get("pageToken"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	resp := struct {
		Devices       []apiDevice `json:"devices,omitempty"`
		NextPageToken string      `json:"nextPageToken,omitempty"`
	}{}
	for i := max(offset, 0); i < len(matched) && len(resp.Devices) < pageSize; i++ {
		resp.Devices = append(resp.Devices, matched[i].api())
		if i+1 < len(matched) && len(resp.Devices) == pageSize {
			resp.NextPageToken = strconv.Itoa(i + 1)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// deviceFor answers the 404 or 403 of a missing, deleted or departed device and returns
// nil then.
func (s *Server) deviceFor(w http.ResponseWriter, r *http.Request, customer string) *device {
	d := s.devices[r.PathValue("device")]
	switch {
	case d == nil || d.customerID != customer || d.state == DeviceDeleted:
		writeGoogleError(w, http.StatusNotFound, "NOT_FOUND", "Requested entity was not found.")
		return nil
	case d.state == DeviceLeft:
		writeGoogleError(w, http.StatusForbidden, "PERMISSION_DENIED", "The caller does not have permission")
		return nil
	}
	return d
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request, customer string) {
	if d := s.deviceFor(w, r, customer); d != nil {
		writeJSON(w, http.StatusOK, d.api())
	}
}

func (s *Server) handleListDeviceUsers(w http.ResponseWriter, r *http.Request, customer string) {
	d := s.deviceFor(w, r, customer)
	if d == nil {
		return
	}
	type apiDeviceUser struct {
		Name      string `json:"name"`
		UserEmail string `json:"userEmail,omitempty"`
	}
	resp := struct {
		DeviceUsers []apiDeviceUser `json:"deviceUsers,omitempty"`
	}{}
	for _, u := range d.users {
		resp.DeviceUsers = append(resp.DeviceUsers, apiDeviceUser{Name: d.userName(u), UserEmail: u.email})
	}
	writeJSON(w, http.StatusOK, resp)
}

// handlePatchClientState creates a device or DeviceUser it doesn't know, so names that
// real Google returned to a real relay can be marked here.
func (s *Server) handlePatchClientState(w http.ResponseWriter, r *http.Request, customer string) {
	var body struct {
		Managed  string `json:"managed"`
		CustomID string `json:"customId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeGoogleError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
		return
	}
	if r.URL.Query().Get("updateMask") == "" {
		writeGoogleError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "updateMask is required")
		return
	}
	// Validated in full first, so a refused patch changes nothing.
	var setManaged, setCustomID bool
	for field := range strings.SplitSeq(r.URL.Query().Get("updateMask"), ",") {
		switch strings.TrimSpace(field) {
		case "managed":
			if body.Managed != "MANAGED" && body.Managed != "UNMANAGED" {
				writeGoogleError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "managed must be MANAGED or UNMANAGED")
				return
			}
			setManaged = true
		case "customId":
			setCustomID = true
		default:
			writeGoogleError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "unsupported updateMask field "+field)
			return
		}
	}
	stateID := r.PathValue("state")
	if stateID != customer+"-fleet" {
		s.logger.WarnContext(r.Context(), "client state ID isn't {customer}-fleet", "client_state", stateID)
	}

	deviceID := r.PathValue("device")
	d := s.devices[deviceID]
	if d == nil {
		d = &device{id: deviceID, customerID: customer, state: DeviceActive, lastSyncTime: time.Now()}
		s.devices[deviceID] = d
	} else if s.deviceFor(w, r, customer) == nil {
		return
	}
	u := d.user(r.PathValue("user"))
	if u == nil {
		u = &deviceUser{id: r.PathValue("user")}
		d.users = append(d.users, u)
	}
	if u.clientStates == nil {
		u.clientStates = map[string]*ClientState{}
	}
	cs := u.clientStates[stateID]
	if cs == nil {
		cs = &ClientState{}
		u.clientStates[stateID] = cs
	}
	if setManaged {
		cs.Managed = body.Managed
	}
	if setCustomID {
		cs.CustomID = body.CustomID
	}
	cs.UpdateTime = time.Now().UTC()
	s.logger.InfoContext(r.Context(), "client state patched", "device_user", d.userName(u), "managed", cs.Managed, "custom_id", cs.CustomID)
	writeJSON(w, http.StatusOK, map[string]any{
		"name": "operations/" + randomHex(8),
		"done": true,
		"response": map[string]any{
			"@type":    "type.googleapis.com/google.apps.cloudidentity.devices.v1.ClientState",
			"name":     r.URL.Path[len("/v1/"):],
			"managed":  cs.Managed,
			"customId": cs.CustomID,
		},
	})
}

func (s *Server) handleAddCustomer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string `json:"customer_id"`
	}
	if err := decodeJSON(r, &req); err != nil || bareCustomerID(req.CustomerID) == "" {
		http.Error(w, "customer_id is required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.customers[bareCustomerID(req.CustomerID)] = struct{}{}
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// SeedDevice is a device record to create, with one DeviceUser per distinct email.
type SeedDevice struct {
	CustomerID string   `json:"customer_id"`
	DeviceID   string   `json:"device_id"`
	DeviceType string   `json:"device_type"`
	Model      string   `json:"model"`
	OSVersion  string   `json:"os_version"`
	Emails     []string `json:"emails"`
}

// Seed creates a device record and returns it. The customer becomes known.
func (s *Server) Seed(sd SeedDevice) Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seedLocked(sd, "").export()
}

// seedLocked creates the record, keyed by resourceID when given or by a random ID.
func (s *Server) seedLocked(sd SeedDevice, resourceID string) *device {
	if resourceID == "" {
		resourceID = "mock" + randomHex(8)
	}
	d := &device{
		id: resourceID, customerID: bareCustomerID(sd.CustomerID), deviceID: sd.DeviceID,
		deviceType: strings.ToUpper(sd.DeviceType), model: sd.Model, osVersion: sd.OSVersion,
		state: DeviceActive, lastSyncTime: time.Now().UTC(),
	}
	for _, email := range sd.Emails {
		d.users = append(d.users, &deviceUser{id: fakeUserID(resourceID, email), email: email})
	}
	s.devices[resourceID] = d
	s.customers[d.customerID] = struct{}{}
	return d
}

func (s *Server) handleSeedDevice(w http.ResponseWriter, r *http.Request) {
	var sd SeedDevice
	if err := decodeJSON(r, &sd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if bareCustomerID(sd.CustomerID) == "" {
		http.Error(w, "customer_id is required", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, s.Seed(sd))
}

// SetDeviceState sets a device to DeviceActive, DeviceDeleted or DeviceLeft, by its
// resource ID. It reports whether the device exists.
func (s *Server) SetDeviceState(resourceID, state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.devices[strings.TrimPrefix(resourceID, "devices/")]
	if d == nil {
		return false
	}
	d.state = state
	return true
}

func (s *Server) handleSetDeviceState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		State string `json:"state"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch req.State {
	case DeviceActive, DeviceDeleted, DeviceLeft:
	default:
		http.Error(w, "state must be active, deleted or left", http.StatusBadRequest)
		return
	}
	if !s.SetDeviceState(r.PathValue("device"), req.State) {
		http.Error(w, "no such device", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetTokenError makes the token endpoint refuse with an OAuth error code, such as
// unauthorized_client for a broken delegation; empty clears it.
func (s *Server) SetTokenError(code string) {
	s.mu.Lock()
	s.tokenError = code
	s.mu.Unlock()
}

func (s *Server) handleSetTokenError(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Error string `json:"error"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.SetTokenError(req.Error)
	w.WriteHeader(http.StatusNoContent)
}
