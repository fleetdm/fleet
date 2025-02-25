package godep

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// Profile corresponds to the Apple DEP API "Profile" structure.
// See https://developer.apple.com/documentation/devicemanagement/profile
type Profile struct {
	ProfileName  string `json:"profile_name"`
	URL          string `json:"url"`
	AllowPairing bool   `json:"allow_pairing,omitempty"`
	IsSupervised bool   `json:"is_supervised,omitempty"`
	IsMultiUser  bool   `json:"is_multi_user,omitempty"`
	IsMandatory  bool   `json:"is_mandatory,omitempty"`
	// AwaitDeviceConfigured should never be set in the profiles we store in the
	// database - it is now always forced to true when registering with Apple.
	AwaitDeviceConfigured bool     `json:"await_device_configured,omitempty"`
	IsMDMRemovable        bool     `json:"is_mdm_removable"` // default true
	SupportPhoneNumber    string   `json:"support_phone_number,omitempty"`
	AutoAdvanceSetup      bool     `json:"auto_advance_setup,omitempty"`
	SupportEmailAddress   string   `json:"support_email_address,omitempty"`
	OrgMagic              string   `json:"org_magic"`
	AnchorCerts           []string `json:"anchor_certs,omitempty"`
	SupervisingHostCerts  []string `json:"supervising_host_certs,omitempty"`
	Department            string   `json:"department,omitempty"`
	Devices               []string `json:"devices,omitempty"`
	Language              string   `json:"language,omitempty"`
	Region                string   `json:"region,omitempty"`
	ConfigurationWebURL   string   `json:"configuration_web_url,omitempty"`

	// See https://developer.apple.com/documentation/devicemanagement/skipkeys
	SkipSetupItems []string `json:"skip_setup_items,omitempty"`

	// additional undocumented key only returned when requesting a profile from Apple.
	ProfileUUID string `json:"profile_uuid,omitempty"`
}

// ProfileResponse corresponds to the Apple DEP API "AssignProfileResponse" structure.
// See https://developer.apple.com/documentation/devicemanagement/assignprofileresponse
type ProfileResponse struct {
	ProfileUUID string            `json:"profile_uuid"`
	Devices     map[string]string `json:"devices"`
}

// AssignProfiles uses the Apple "Assign a profile to a list of devices" API
// endpoint to assign a DEP profile UUID to a list of serial numbers.
// The name parameter specifies the configured DEP name to use.
// Note we use HTTP PUT for compatibility despite modern documentation
// listing HTTP POST for this endpoint.
// See https://developer.apple.com/documentation/devicemanagement/assign_a_profile
func (c *Client) AssignProfile(ctx context.Context, name, uuid string, serials ...string) (*ProfileResponse, error) {
	req := &struct {
		ProfileUUID string   `json:"profile_uuid"`
		Devices     []string `json:"devices"`
	}{
		ProfileUUID: uuid,
		Devices:     serials,
	}
	resp := new(ProfileResponse)
	// historically this has been an HTTP PUT and the DEP simulator depsim
	// requires this. however modern Apple documentation says this is a POST
	// now. we still use PUT here for compatibility.
	return resp, c.doWithAfterHook(ctx, name, http.MethodPut, "/profile/devices", req, resp)
}

// DefineProfile uses the Apple "Define a Profile" command to attempt to create a profile.
// This service defines a profile with Apple's servers that can then be assigned to specific devices.
// This command provides information about the MDM server that is assigned to manage one or more devices,
// information about the host that the managed devices can pair with, and various attributes that control
// the MDM association behavior of the device.
// See https://developer.apple.com/documentation/devicemanagement/define_a_profile
func (c *Client) DefineProfile(ctx context.Context, name string, profile *Profile) (*ProfileResponse, error) {
	resp := new(ProfileResponse)
	return resp, c.doWithAfterHook(ctx, name, http.MethodPost, "/profile", profile, resp)
}

// GetProfile uses the Apple "Get a Profile" API endpoint to get the details
// for the specified profile UUID.
// See https://developer.apple.com/documentation/devicemanagement/get_a_profile
func (c *Client) GetProfile(ctx context.Context, name, profileUUID string) (*json.RawMessage, error) {
	resp := &json.RawMessage{}
	qs := url.Values{"profile_uuid": {profileUUID}}
	return resp, c.doWithAfterHook(ctx, name, http.MethodGet, "/profile?"+qs.Encode(), nil, resp)
}
