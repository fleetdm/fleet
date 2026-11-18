package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

// GoogleConditionalAccessIntegration is the app config part of the integration.
type GoogleConditionalAccessIntegration struct {
	// CustomerID includes the leading "C".
	CustomerID string `json:"customer_id"`
	// AdminEmail is the Workspace admin the service account impersonates.
	AdminEmail string `json:"admin_email"`
	// APIKeyJSON is only ever an input: ModifyAppConfig stores the key encrypted in
	// mdm_config_assets and clears it before saving the app config.
	APIKeyJSON GoogleServiceAccountKey `json:"api_key_json,omitempty"`
}

// GoogleServiceAccountKey is a service account key JSON object, or MaskedPassword. It is a
// string so the settings stay comparable.
type GoogleServiceAccountKey string

// Provided reports whether it holds a key rather than nothing or the masked value.
func (k GoogleServiceAccountKey) Provided() bool {
	return k != "" && k != MaskedPassword
}

func (k GoogleServiceAccountKey) MarshalJSON() ([]byte, error) {
	switch k {
	case "":
		return []byte("null"), nil
	case MaskedPassword:
		return json.Marshal(MaskedPassword)
	}
	return []byte(k), nil
}

func (k *GoogleServiceAccountKey) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	switch {
	case string(data) == "null":
		*k = ""
	case len(data) > 0 && data[0] == '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil || s != MaskedPassword {
			return errors.New("api_key_json must be a JSON object or the masked value")
		}
		*k = MaskedPassword
	case len(data) > 0 && data[0] == '{':
		var compacted bytes.Buffer
		if err := json.Compact(&compacted, data); err != nil {
			return err
		}
		*k = GoogleServiceAccountKey(compacted.String())
	default:
		return errors.New("api_key_json must be a JSON object or the masked value")
	}
	return nil
}

// ConditionalAccessGoogleStatus is the integration's connection state.
type ConditionalAccessGoogleStatus struct {
	ServiceAccountKeyConfigured bool `json:"service_account_key_configured"`
	// State is empty when the integration is off.
	State        ConditionalAccessGoogleState `json:"state"`
	ErrorMessage *string                      `json:"error_message"`
}

// ConditionalAccessGoogleIntegration is the fleetdm.com registration and connection
// state of the Google conditional access integration (one global row). The customer ID
// and admin email live in the app config only: the registration is always for the
// customer the config names.
type ConditionalAccessGoogleIntegration struct {
	// ProxyServerSecret authenticates this server with fleetdm.com and fleetdm.com with this server.
	ProxyServerSecret string                       `db:"proxy_server_secret" json:"-"`
	State             ConditionalAccessGoogleState `db:"state"`
	// ErrorMessage is the reason the integration isn't connected, meant for the admin.
	ErrorMessage *string `db:"error_message"`
}

// ConditionalAccessGoogleState is the connection state of the Google conditional
// access integration.
type ConditionalAccessGoogleState string

const (
	// ConditionalAccessGoogleStatePending is saved before registering with fleetdm.com.
	ConditionalAccessGoogleStatePending            ConditionalAccessGoogleState = "pending"
	ConditionalAccessGoogleStateDelegationFailed   ConditionalAccessGoogleState = "delegation_failed"
	ConditionalAccessGoogleStateRegistrationFailed ConditionalAccessGoogleState = "registration_failed"
	ConditionalAccessGoogleStateConnected          ConditionalAccessGoogleState = "connected"
	// ConditionalAccessGoogleStateDeleted marks a removed integration. Its device rows are
	// kept, so adding it back for the same customer finds them; its updated_at is when it
	// was removed.
	ConditionalAccessGoogleStateDeleted ConditionalAccessGoogleState = "deleted"
)

// AuthzType implements authz.AuthzTyper.
func (c *ConditionalAccessGoogleIntegration) AuthzType() string {
	return "conditional_access_google"
}

// Registered reports whether this server is registered with fleetdm.com.
func (c *ConditionalAccessGoogleIntegration) Registered() bool {
	return c != nil && c.ProxyServerSecret != ""
}

var googleCustomerIDRegexp = regexp.MustCompile(`^C[0-9A-Za-z]{6,32}$`)

// Validate trims the fields in place and appends any error to invalid.
func (g *GoogleConditionalAccessIntegration) Validate(invalid *InvalidArgumentError) {
	g.CustomerID = strings.TrimSpace(g.CustomerID)
	g.AdminEmail = strings.TrimSpace(g.AdminEmail)

	if g.CustomerID == "" {
		invalid.Append("customer_id", "customer_id is required")
	} else if !googleCustomerIDRegexp.MatchString(g.CustomerID) {
		invalid.Append("customer_id", "must be a Google customer ID starting with C, for example C01234abc")
	}

	if g.AdminEmail == "" {
		invalid.Append("admin_email", "admin_email is required")
	} else if addr, err := mail.ParseAddress(g.AdminEmail); err != nil || addr.Address != g.AdminEmail {
		invalid.Append("admin_email", "must be a valid email address")
	}

	if g.APIKeyJSON.Provided() {
		var key struct {
			ClientEmail string `json:"client_email"`
			PrivateKey  string `json:"private_key"`
		}
		if err := json.Unmarshal([]byte(g.APIKeyJSON), &key); err != nil || key.ClientEmail == "" || key.PrivateKey == "" {
			invalid.Append("api_key_json", "api_key_json must include client_email and private_key")
		}
	}
}

// GoogleConditionalAccessDeviceUserState is the marking state of a stored DeviceUser.
type GoogleConditionalAccessDeviceUserState string

const (
	// GoogleConditionalAccessDeviceUserPending is stored but not yet marked MANAGED.
	GoogleConditionalAccessDeviceUserPending GoogleConditionalAccessDeviceUserState = "pending"
	GoogleConditionalAccessDeviceUserManaged GoogleConditionalAccessDeviceUserState = "managed"
	// GoogleConditionalAccessDeviceUserStale is a name Google refused (404 or 403). It
	// doesn't count as registered, so the host signs in again on the next open.
	GoogleConditionalAccessDeviceUserStale GoogleConditionalAccessDeviceUserState = "stale"
	// GoogleConditionalAccessDeviceUserUnmanaging is queued for the UNMANAGED write.
	GoogleConditionalAccessDeviceUserUnmanaging GoogleConditionalAccessDeviceUserState = "unmanaging"
)

const (
	// GoogleConditionalAccessMaxRegistrationAttempts is how many sign-ins a host can
	// start without a register call within GoogleConditionalAccessRegistrationWindow.
	GoogleConditionalAccessMaxRegistrationAttempts = 3
	GoogleConditionalAccessRegistrationWindow      = 24 * time.Hour
)

// GoogleConditionalAccessDeviceUser is a Google DeviceUser resource name stored on a host.
type GoogleConditionalAccessDeviceUser struct {
	HostUUID     string                                 `db:"host_uuid"`
	Name         string                                 `db:"device_user_name"`
	State        GoogleConditionalAccessDeviceUserState `db:"state"`
	ErrorMessage *string                                `db:"error_message"`
	UpdatedAt    time.Time                              `db:"updated_at"`
}

// GoogleConditionalAccessClientState is what Fleet tells Google about a DeviceUser. A nil
// field is left as Google has it; the client writes only the fields that are set.
type GoogleConditionalAccessClientState struct {
	Managed *bool
	// CustomID is Google's free-form identifier for the device; Fleet sets the host UUID.
	CustomID *string
}

// GoogleConditionalAccessDesiredClientState decides the client state a stored DeviceUser
// should have in Google. It is the single place to extend when Fleet reports more than
// enrollment, such as compliance from policies.
func GoogleConditionalAccessDesiredClientState(du GoogleConditionalAccessDeviceUser) GoogleConditionalAccessClientState {
	if du.State == GoogleConditionalAccessDeviceUserUnmanaging {
		return GoogleConditionalAccessClientState{Managed: new(false)}
	}
	return GoogleConditionalAccessClientState{Managed: new(true), CustomID: new(du.HostUUID)}
}

// GoogleConditionalAccessDevice is a host's registration row with its DeviceUsers.
type GoogleConditionalAccessDevice struct {
	HostUUID    string `db:"host_uuid"`
	DeviceUsers []GoogleConditionalAccessDeviceUser
}

// Registered reports whether the host has a name that is marked or waiting to be.
func (d *GoogleConditionalAccessDevice) Registered() bool {
	if d == nil {
		return false
	}
	for _, du := range d.DeviceUsers {
		if du.State == GoogleConditionalAccessDeviceUserPending || du.State == GoogleConditionalAccessDeviceUserManaged {
			return true
		}
	}
	return false
}

// Managed reports whether the host is registered and every name it is registered with
// is marked MANAGED.
func (d *GoogleConditionalAccessDevice) Managed() bool {
	if d == nil || !d.Registered() {
		return false
	}
	for _, du := range d.DeviceUsers {
		if du.State == GoogleConditionalAccessDeviceUserPending {
			return false
		}
	}
	return true
}

// GoogleConditionalAccessDeviceUserFilter selects DeviceUsers for the sync cron.
type GoogleConditionalAccessDeviceUserFilter struct {
	CustomerID string
	States     []GoogleConditionalAccessDeviceUserState
	// HostExists skips names whose host was deleted.
	HostExists bool
	// AfterName pages through the names in order.
	AfterName string
	Limit     int
}

// GoogleDeviceIDFromDeviceUserName returns the device ID of a DeviceUser resource name
// (devices/{device}/deviceUsers/{deviceUser}), or "" if the name isn't one.
func GoogleDeviceIDFromDeviceUserName(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != "devices" || parts[2] != "deviceUsers" {
		return ""
	}
	return parts[1]
}

// GoogleConditionalAccessBookmarkHost is a host with a Google device ID and no stored
// DeviceUsers: a row migrated from the bookmark flow that the sync cron still has to
// resolve.
type GoogleConditionalAccessBookmarkHost struct {
	HostID         uint   `db:"host_id"`
	HostUUID       string `db:"host_uuid"`
	GoogleDeviceID string `db:"google_device_id"`
}

// GoogleConditionalAccessAndroidHost is a company-owned Android host with no registered
// DeviceUser, which the sync cron resolves from its Android Management API device ID.
type GoogleConditionalAccessAndroidHost struct {
	HostID   uint   `db:"host_id"`
	HostUUID string `db:"host_uuid"`
	// AndroidDeviceID is also the deviceId of the host's record in Google.
	AndroidDeviceID string `db:"android_device_id"`
}

// GoogleConditionalAccessDeviceUserConflictError is returned when a DeviceUser is
// already stored on another host.
type GoogleConditionalAccessDeviceUserConflictError struct {
	Name string
}

func (e *GoogleConditionalAccessDeviceUserConflictError) Error() string {
	return "This Google device user is already registered to another host."
}
