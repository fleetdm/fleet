package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/mail"
	"regexp"
	"strings"
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
