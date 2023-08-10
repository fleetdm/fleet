package fleet

import (
	"strings"
	"sync"
)

// Capability represents a concrete feature of Fleet.
type Capability string

// CapabilityMap is an utility type to represent a set of capabilities.
type CapabilityMap map[Capability]struct{}

// mu is used to allow for safe access to the capability map.
var mu sync.Mutex

// PopulateFromString populates the CapabilityMap from a comma separated string.
// Example: "foo,bar,baz" => {"foo": struct{}, "bar": struct{}, "baz": struct{}}
func (c *CapabilityMap) PopulateFromString(s string) {
	mu.Lock()
	defer mu.Unlock()
	*c = make(CapabilityMap)

	if s == "" {
		return
	}

	for _, capability := range strings.Split(s, ",") {
		(*c)[Capability(capability)] = struct{}{}
	}
}

// String returns a comma separated string with the capabilities in the map.
// Example: {"foo": struct{}, "bar": struct{}, "baz": struct{}} => "foo,bar,baz"
func (c *CapabilityMap) String() string {
	mu.Lock()
	defer mu.Unlock()
	idx := 0
	capabilities := make([]string, len(*c))
	for capability := range *c {
		capabilities[idx] = string(capability)
		idx++
	}
	return strings.Join(capabilities, ",")
}

// Has returns true if the CapabilityMap contains the given capability.
func (c CapabilityMap) Has(capability Capability) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := c[capability]
	return ok
}

// The following are the capabilities that Fleet supports. These can be used by
// the Fleet server, Orbit or Fleet Desktop to communicate that a given feature
// is supported.
const (
	// CapabilityOrbitEndpoints denotes the presence of server endpoints
	// dedicated to communicating with Orbit. These endpoints start with
	// `/api/fleet/orbit`, and allow enrolling a host through Orbit among other
	// functionality.
	CapabilityOrbitEndpoints Capability = "orbit_endpoints"
	// CapabilityTokenRotation denotes the ability of the server to support
	// periodic rotation of device tokens
	CapabilityTokenRotation Capability = "token_rotation"
)

// ServerOrbitCapabilities is a set of capabilities that server-side,
// Orbit-related endpoint supports.
// **it shouldn't be modified at runtime**
var ServerOrbitCapabilities = CapabilityMap{
	CapabilityOrbitEndpoints: {},
	CapabilityTokenRotation:  {},
}

// ServerDeviceCapabilities is a set of capabilities that server-side,
// Device-related endpoint supports.
// **it shouldn't be modified at runtime**
var ServerDeviceCapabilities = CapabilityMap{}

// CapabilitiesHeader is the header name used to communicate the capabilities.
const CapabilitiesHeader = "X-Fleet-Capabilities"
