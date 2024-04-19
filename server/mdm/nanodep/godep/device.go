package godep

import (
	"context"
	"net/http"
	"time"
)

// Device corresponds to the Apple DEP API "Device" structure.
// See https://developer.apple.com/documentation/devicemanagement/device
type Device struct {
	SerialNumber       string    `json:"serial_number"`
	Model              string    `json:"model"`
	Description        string    `json:"description"`
	Color              string    `json:"color"`
	AssetTag           string    `json:"asset_tag,omitempty"`
	ProfileStatus      string    `json:"profile_status"`
	ProfileUUID        string    `json:"profile_uuid,omitempty"`
	ProfileAssignTime  time.Time `json:"profile_assign_time,omitempty"`
	ProfilePushTime    time.Time `json:"profile_push_time,omitempty"`
	DeviceAssignedDate time.Time `json:"device_assigned_date,omitempty"`
	DeviceAssignedBy   string    `json:"device_assigned_by,omitempty"`
	OS                 string    `json:"os,omitempty"`
	DeviceFamily       string    `json:"device_family,omitempty"`
	// fetch/sync-only fields
	OpType string    `json:"op_type,omitempty"`
	OpDate time.Time `json:"op_date,omitempty"`
}

// deviceRequest corresponds to the Apple DEP API "FetchDeviceRequest" and
// "SyncDeviceRequest" structures.
// See https://developer.apple.com/documentation/devicemanagement/fetchdevicerequest
// and https://developer.apple.com/documentation/devicemanagement/syncdevicerequest
type deviceRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// DeviceResponse corresponds to the Apple DEP "FetchDeviceResponse" structure.
// See https://developer.apple.com/documentation/devicemanagement/fetchdeviceresponse
type DeviceResponse struct {
	Cursor       string    `json:"cursor,omitempty"`
	FetchedUntil time.Time `json:"fetched_until,omitempty"`
	MoreToFollow bool      `json:"more_to_follow"`
	Devices      []Device  `json:"devices,omitempty"`
}

type DeviceRequestOption func(*deviceRequest)

// WithCursor includes a cursor in the fetch or sync request. The initial
// fetch request should omit this option.
func WithCursor(cursor string) DeviceRequestOption {
	return func(d *deviceRequest) {
		d.Cursor = cursor
	}
}

// WithCursor includes a device limit in the fetch or sync request.
// Per Apple the API has a default of 100 and a maximum of 1000.
func WithLimit(limit int) DeviceRequestOption {
	return func(d *deviceRequest) {
		d.Limit = limit
	}
}

// FetchDevices uses the Apple "Get a List of Devices" API endpoint to retrieve
// a list of all devices corresponding to this configured DEP server (DEP name).
// The name parameter specifies the configured DEP name to use.
// You should provide a cursor that is returned from previous FetchDevices
// call responses on any subsequent calls.
// See https://developer.apple.com/documentation/devicemanagement/get_a_list_of_devices
func (c *Client) FetchDevices(ctx context.Context, name string, opts ...DeviceRequestOption) (*DeviceResponse, error) {
	req := new(deviceRequest)
	for _, opt := range opts {
		opt(req)
	}
	resp := new(DeviceResponse)
	return resp, c.doWithAfterHook(ctx, name, http.MethodPost, "/server/devices", req, resp)
}

// SyncDevices uses the Apple "Sync the List of Devices" API endpoint to get
// updates about the list of devices corresponding to this configured DEP
// server (DEP name).
// The name parameter specifies the configured DEP name to use.
// You should provide a cursor that is returned from previous FetchDevices or
// SyncDevices call responses.
// See https://developer.apple.com/documentation/devicemanagement/sync_the_list_of_devices
func (c *Client) SyncDevices(ctx context.Context, name string, opts ...DeviceRequestOption) (*DeviceResponse, error) {
	req := new(deviceRequest)
	for _, opt := range opts {
		opt(req)
	}
	resp := new(DeviceResponse)
	return resp, c.doWithAfterHook(ctx, name, http.MethodPost, "/devices/sync", req, resp)
}

// GetDevicesDetails uses the Apple "Get Device Details" API endpoint to
// retrieve the details (such as its assigned enrollment profile UUID) for the
// specified device, identified by its serial number.
// See https://developer.apple.com/documentation/devicemanagement/get_device_details
func (c *Client) GetDeviceDetails(ctx context.Context, name, serialNumber string) (*Device, error) {
	type request struct {
		Devices []string `json:"devices"`
	}
	type response struct {
		Devices map[string]*Device `json:"devices"`
	}
	resp := new(response)
	if err := c.doWithAfterHook(ctx, name, http.MethodPost, "/devices", request{
		Devices: []string{serialNumber},
	}, resp); err != nil {
		return nil, err
	}
	return resp.Devices[serialNumber], nil
}

// IsCursorExhausted returns true if err is a DEP "exhausted cursor" error.
func IsCursorExhausted(err error) bool {
	return httpErrorContains(err, http.StatusBadRequest, "EXHAUSTED_CURSOR")
}

// IsCursorInvalid returns true if err is a DEP "invalid cursor" error.
func IsCursorInvalid(err error) bool {
	return httpErrorContains(err, http.StatusBadRequest, "INVALID_CURSOR")
}

// IsCursorExpired returns true if err is a DEP "expired cursor" error.
// Per Apple this indicates the cursor is older than 7 days.
func IsCursorExpired(err error) bool {
	return httpErrorContains(err, http.StatusBadRequest, "EXPIRED_CURSOR")
}
