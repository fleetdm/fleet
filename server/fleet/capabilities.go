package fleet

import "strings"

// Capability represents a concrete feature of Fleet.
type Capability string

// CapabilityMap is an utility type to represent a set of capabilities.
type CapabilityMap map[Capability]struct{}

// PopulateFromString populates the CapabilityMap from a comma separated string.
// Example: "foo,bar,baz" => {"foo": struct{}, "bar": struct{}, "baz": struct{}}
func (c *CapabilityMap) PopulateFromString(s string) {
	if s == "" {
		return
	}
	raw := strings.Split(s, ",")
	*c = make(CapabilityMap, len(raw))
	for _, capability := range raw {
		(*c)[Capability(capability)] = struct{}{}
	}
}

// String returns a comma separated string with the capabilities in the map.
// Example: {"foo": struct{}, "bar": struct{}, "baz": struct{}} => "foo,bar,baz"
func (c *CapabilityMap) String() string {
	idx := 0
	capabilities := make([]string, len(*c))
	for capability := range *c {
		capabilities[idx] = string(capability)
		idx++
	}
	return strings.Join(capabilities, ",")
}

// The following are the capabilities that Fleet supports. These can be used by
// the Fleet server, Orbit or Fleet Desktop to communicate that a given feature
// is supported.
const (
	// CapabilityTokenRotation is the ability to rotate and expire host device
	// tokens over a given period of time.
	CapabilityTokenRotation Capability = "token_rotation"
)

// ServerOrbitCapabilities is a set of capabilities that server-side,
// Orbit-related endpoint supports.
// **it shouldn't be modified at runtime**
var ServerOrbitCapabilities = CapabilityMap{CapabilityTokenRotation: {}}

// ServerDeviceCapabilities is a set of capabilities that server-side,
// Device-related endpoint supports.
// **it shouldn't be modified at runtime**
var ServerDeviceCapabilities = CapabilityMap{}

// CapabilitiesHeader is the header name used to communicate the capabilities.
const CapabilitiesHeader = "X-Fleet-Capabilities"
