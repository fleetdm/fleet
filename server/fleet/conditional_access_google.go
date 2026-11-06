package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
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

// ConditionalAccessGoogleIntegration is the connection state of the Google
// conditional access integration (one global row).
type ConditionalAccessGoogleIntegration struct {
	// CustomerID is the Google customer ID with its leading "C", as the Admin console shows it.
	CustomerID string `db:"customer_id"`
	// AdminEmail is the delegation subject the connection was verified with.
	AdminEmail string `db:"admin_email"`
	// ProxyServerSecret authenticates this server with fleetdm.com and fleetdm.com with this server.
	ProxyServerSecret string `db:"proxy_server_secret" json:"-"`
	// OAuthClientID is Fleet's Web OAuth client ID as returned by fleetdm.com.
	// TODO: the Fleet Android app might not need this client ID; drop it if it doesn't.
	OAuthClientID string                       `db:"oauth_client_id"`
	State         ConditionalAccessGoogleState `db:"state"`
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
	// ConditionalAccessGoogleStateDeleted keeps the registration and the device rows after
	// a DELETE, so re-creating the integration for the same customer reuses them.
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
	HostUUID     string                                 `db:"host_uuid" json:"-"`
	Name         string                                 `db:"device_user_name" json:"name"`
	State        GoogleConditionalAccessDeviceUserState `db:"state" json:"state"`
	ErrorMessage *string                                `db:"error_message" json:"error_message"`
	UpdatedAt    time.Time                              `db:"updated_at" json:"updated_at"`
}

// GoogleConditionalAccessDevice is a host's registration row with its DeviceUsers.
type GoogleConditionalAccessDevice struct {
	HostUUID                   string     `db:"host_uuid"`
	CustomerID                 string     `db:"customer_id"`
	RegistrationAttempts       uint       `db:"registration_attempts"`
	FirstRegistrationAttemptAt *time.Time `db:"first_registration_attempt_at"`
	DeviceUsers                []GoogleConditionalAccessDeviceUser
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

// GoogleConditionalAccessBookmarkHost is a host with a Google device ID from the
// bookmark flow and no stored DeviceUsers.
type GoogleConditionalAccessBookmarkHost struct {
	HostUUID       string `db:"host_uuid"`
	GoogleDeviceID string `db:"google_device_id"`
}

// GoogleConditionalAccessDeviceUserConflictError is returned when a DeviceUser is
// already stored on another host.
type GoogleConditionalAccessDeviceUserConflictError struct {
	Name string
}

func (e *GoogleConditionalAccessDeviceUserConflictError) Error() string {
	return "This Google device user is already registered to another host."
}

func (e *GoogleConditionalAccessDeviceUserConflictError) Status() int { return http.StatusConflict }

func (e *GoogleConditionalAccessDeviceUserConflictError) IsClientError() bool { return true }
