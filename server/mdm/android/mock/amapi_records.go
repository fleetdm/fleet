package mock

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

// amapiRecords stands in for Google's record of devices and command operations. Fleet
// checks Pub/Sub notifications that reset, unenroll or finish commands against AMAPI, so
// test helpers that build those notifications register what they report here, and the
// lookups set up by InitDeviceLookupMocks answer from it.
var amapiRecords = struct {
	mu         sync.Mutex
	devices    map[string]*androidmanagement.Device
	operations map[string]*androidmanagement.Operation
}{
	devices:    map[string]*androidmanagement.Device{},
	operations: map[string]*androidmanagement.Operation{},
}

// RegisterDevice records the device a test reports to Fleet in a Pub/Sub notification as
// AMAPI's current state of that device. A DELETED device is reported as not found.
func RegisterDevice(device androidmanagement.Device) {
	if device.Name == "" {
		return
	}
	amapiRecords.mu.Lock()
	defer amapiRecords.mu.Unlock()
	amapiRecords.devices[device.Name] = &device
}

// RegisterOperation records a command operation a test reports to Fleet in a COMMAND
// notification as AMAPI's current state of that operation.
func RegisterOperation(op androidmanagement.Operation) {
	if op.Name == "" {
		return
	}
	amapiRecords.mu.Lock()
	defer amapiRecords.mu.Unlock()
	amapiRecords.operations[op.Name] = &op
}

func notFoundError() error {
	return &googleapi.Error{Code: http.StatusNotFound, Message: "Requested entity was not found."}
}

// InitDeviceLookupMocks sets EnterprisesDevicesGet and EnterprisesDevicesOperationsGet to
// answer from the devices and operations registered with RegisterDevice and
// RegisterOperation.
func (p *Client) InitDeviceLookupMocks() {
	p.EnterprisesDevicesGetFunc = func(_ context.Context, deviceName string) (*androidmanagement.Device, error) {
		amapiRecords.mu.Lock()
		defer amapiRecords.mu.Unlock()
		device, ok := amapiRecords.devices[deviceName]
		if !ok || strings.EqualFold(device.State, string(android.DeviceStateDeleted)) ||
			strings.EqualFold(device.AppliedState, string(android.DeviceStateDeleted)) {
			return nil, notFoundError()
		}
		ret := *device
		return &ret, nil
	}
	p.EnterprisesDevicesOperationsGetFunc = func(_ context.Context, operationName string) (*androidmanagement.Operation, error) {
		amapiRecords.mu.Lock()
		defer amapiRecords.mu.Unlock()
		op, ok := amapiRecords.operations[operationName]
		if !ok {
			return nil, notFoundError()
		}
		ret := *op
		return &ret, nil
	}
}
