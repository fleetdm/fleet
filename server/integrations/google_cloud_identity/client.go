package google_cloud_identity

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	cloudidentity "google.golang.org/api/cloudidentity/v1beta1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Client wraps the official cloudidentity/v1beta1 SDK. The wrapping is
// intentionally thin — methods exist mainly to (a) constrain the surface
// area the rest of Fleet's code can use, and (b) make tests easier by
// funneling all API I/O through this type.
type Client struct {
	ci *cloudidentity.Service
}

// NewClient constructs a Client. opts are passed through to the SDK
// (option.WithTokenSource, option.WithEndpoint for tests, etc.).
func NewClient(ctx context.Context, opts ...option.ClientOption) (*Client, error) {
	ci, err := cloudidentity.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("init cloudidentity service: %w", err)
	}
	return &Client{ci: ci}, nil
}

// FindDeviceBySerial returns the Cloud Identity Device matching the given
// hardware serial number, or nil if no device matches. Used to resolve a
// Fleet host to its Cloud Identity device record via host.hardware_serial.
//
// Returns nil + nil error when no match is found (a Fleet host that has
// never had any Google-managed surface signed into it has no Cloud Identity
// record).
func (c *Client) FindDeviceBySerial(ctx context.Context, serial string) (*cloudidentity.Device, error) {
	// Filter syntax per
	// https://developers.google.com/workspace/admin/directory/v1/search-operators
	// — field is `serial`, not `serial_number`. The Cloud Identity devices.list
	// endpoint shares this search-field set with the Directory API.
	resp, err := c.ci.Devices.List().
		Customer("customers/my_customer").
		Filter(fmt.Sprintf("serial:%s", serial)).
		PageSize(5).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}
	if len(resp.Devices) == 0 {
		return nil, nil
	}
	// In practice each serial is unique; if multiples appear, prefer the
	// most recently synced one (last_sync_time descending).
	best := resp.Devices[0]
	for _, d := range resp.Devices[1:] {
		if d.LastSyncTime > best.LastSyncTime {
			best = d
		}
	}
	return best, nil
}

// ListDeviceUsers returns all deviceUser resources under a parent device.
// `parent` must be the canonical resource name returned by
// FindDeviceBySerial (i.e. `devices/{deviceId}`, with the deviceId in the
// exact form Google returned).
func (c *Client) ListDeviceUsers(ctx context.Context, parent string) ([]*cloudidentity.DeviceUser, error) {
	var out []*cloudidentity.DeviceUser
	err := c.ci.Devices.DeviceUsers.List(parent).
		Customer("customers/my_customer").
		Pages(ctx, func(resp *cloudidentity.ListDeviceUsersResponse) error {
			out = append(out, resp.DeviceUsers...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PatchClientState writes Fleet's desired ClientState onto a deviceUser
// resource. The `deviceUserName` argument is the canonical
// `devices/{deviceId}/deviceUsers/{deviceUserId}` resource name (from
// ListDeviceUsers); `partner` is the `{suffix}-{customerID-without-C}` form
// per the Access Context Manager spec; `state` is the ClientState body
// (with Etag set when this is a follow-up PATCH).
//
// updateMask is the field mask Google uses to decide which fields of
// `state` to apply.
func (c *Client) PatchClientState(
	ctx context.Context,
	deviceUserName string,
	partner string,
	state *cloudidentity.ClientState,
	updateMask string,
) (*cloudidentity.Operation, error) {
	if deviceUserName == "" {
		return nil, errors.New("PatchClientState: deviceUserName is required")
	}
	if partner == "" {
		return nil, errors.New("PatchClientState: partner is required")
	}
	if state == nil {
		return nil, errors.New("PatchClientState: state is required")
	}
	resourceName := fmt.Sprintf("%s/clientStates/%s", deviceUserName, partner)
	return c.ci.Devices.DeviceUsers.ClientStates.Patch(resourceName, state).
		Customer("customers/my_customer").
		UpdateMask(updateMask).
		Context(ctx).
		Do()
}

// IsPermissionDenied reports whether the error came back as HTTP 403, which
// on Cloud Identity ClientState PATCH most commonly means the customer's
// Workspace edition does not include Cloud Identity Premium security
// management (see proposal Customer-side prerequisites section).
func IsPermissionDenied(err error) bool {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == http.StatusForbidden
}
