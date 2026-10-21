package profilevalidation

import (
	"encoding/json"
)

// AppleVersion pins the apple/device-management commit the Apple sections were generated from.
type AppleVersion struct {
	Release string `json:"release"`
	Commit  string `json:"commit"`
}

// AppleEnvelope describes the profile itself rather than a payload type: TopLevel is the outer
// dictionary, CommonPayloadKeys the keys every payload carries.
type AppleEnvelope struct {
	TopLevel          *ApplePayloadType `json:"TopLevel"`
	CommonPayloadKeys *ApplePayloadType `json:"CommonPayloadKeys"`
}

// ApplePayloadType describes one .mobileconfig payload type, DDM declaration type, or envelope schema.
type ApplePayloadType struct {
	Platforms map[AppleOS]ApplePlatform `json:"platforms,omitempty"`
	// Source is "apple" for upstream entries and "override" for allow entries.
	Source string               `json:"source,omitempty"`
	Keys   map[string]*AppleKey `json:"keys,omitempty"`
	// Open and Values work as on AppleKey, for payload types whose top level accepts any key.
	Open   bool      `json:"open,omitempty"`
	Values *AppleKey `json:"values,omitempty"`
	// Override is the reason from the Fleet override that added or changed this entry, if any.
	Override string `json:"override,omitempty"`
}

// AppleKey describes a payload key. Constraints are recorded even when no validator enforces them yet.
type AppleKey struct {
	Type       AppleTypes     `json:"type"`
	Required   bool           `json:"required,omitempty"`
	RangeList  []any          `json:"rangelist,omitempty"`
	Range      *Range         `json:"range,omitempty"`
	Format     string         `json:"format,omitempty"`
	Repetition *Range         `json:"repetition,omitempty"`
	ValueType  AppleValueType `json:"valuetype,omitempty"`
	// Open means keys not listed in Keys are accepted. Values, when set, is what their values must
	// look like (Apple's "ANY" subkey).
	Open bool `json:"open,omitempty"`
	// Ref is the key path of an enclosing definition that this key repeats, with "[]" for array items.
	Ref string `json:"ref,omitempty"`
	// Items describes array elements; Keys describes dictionary subkeys.
	Items  *AppleKey            `json:"items,omitempty"`
	Keys   map[string]*AppleKey `json:"keys,omitempty"`
	Values *AppleKey            `json:"values,omitempty"`
	// Platforms is set only where it differs from the enclosing payload type's.
	Platforms map[AppleOS]ApplePlatform `json:"platforms,omitempty"`
	// Override is the reason from the Fleet override that added or changed this key, if any.
	Override string `json:"override,omitempty"`
}

// AppleOS is a platform key from the "supportedOS Object" in Apple's docs/schema.md. iPadOS is
// covered by iOS. Unknown values are carried as declared.
type AppleOS string

const (
	AppleOSiOS      AppleOS = "iOS"
	AppleOSmacOS    AppleOS = "macOS"
	AppleOStvOS     AppleOS = "tvOS"
	AppleOSvisionOS AppleOS = "visionOS"
	AppleOSwatchOS  AppleOS = "watchOS"
)

// ApplePlatform is a per-OS lifecycle. Apple writes "n/a" in Introduced for unsupported platforms.
type ApplePlatform struct {
	Introduced string `json:"introduced,omitempty"`
	Deprecated string `json:"deprecated,omitempty"`
	Removed    string `json:"removed,omitempty"`
	Supervised bool   `json:"supervised,omitempty"`
}

// AppleType is a key's value type, from the "Type Values" list in Apple's docs/schema.md (without
// the angle brackets). Values outside this list are carried as declared, so newer upstream data
// never fails parsing.
type AppleType string

const (
	AppleTypeString     AppleType = "string"
	AppleTypeInteger    AppleType = "integer"
	AppleTypeReal       AppleType = "real"
	AppleTypeBoolean    AppleType = "boolean"
	AppleTypeDate       AppleType = "date" // deprecated upstream in favor of string + valuetype
	AppleTypeData       AppleType = "data"
	AppleTypeArray      AppleType = "array"
	AppleTypeDictionary AppleType = "dictionary"
	AppleTypeAny        AppleType = "any"
)

// AppleValueType is the format of a string value, from the "Valuetype Values" list in Apple's
// docs/schema.md. Unknown values are carried as declared.
type AppleValueType string

const (
	AppleValueTypeDomain       AppleValueType = "domain"
	AppleValueTypeDomainPrefix AppleValueType = "domain-prefix"
	AppleValueTypeEmail        AppleValueType = "email"
	AppleValueTypeHostname     AppleValueType = "hostname"
	AppleValueTypeLocalTime    AppleValueType = "localtime"
	AppleValueTypeRegex        AppleValueType = "regex"
	AppleValueTypeTimestamp    AppleValueType = "timestamp"
	AppleValueTypeURL          AppleValueType = "url"
	AppleValueTypeUUID         AppleValueType = "uuid"
)

// AppleTypes is a key's type. It's a single string in JSON unless an override widened it to a list.
type AppleTypes []AppleType

func (t AppleTypes) MarshalJSON() ([]byte, error) {
	if len(t) == 1 {
		return json.Marshal(t[0])
	}
	return json.Marshal([]AppleType(t))
}

func (t *AppleTypes) UnmarshalJSON(data []byte) error {
	var s AppleType
	if err := json.Unmarshal(data, &s); err == nil {
		*t = AppleTypes{s}
		return nil
	}
	var list []AppleType
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	*t = list
	return nil
}
