package profilevalidation

// Mode picks how much the validators report.
type Mode int

const (
	// ModeLenient reports only wrong-type values on documented keys.
	ModeLenient Mode = iota
	// ModeStrict also reports unknown types, unknown keys, envelope problems, and for Windows a
	// <Meta><Format> or scope that disagrees with the DDF.
	ModeStrict
)

// Code is the kind of problem a Finding reports.
type Code string

const (
	CodeTypeMismatch          Code = "TYPE_MISMATCH"
	CodeUnknownType           Code = "UNKNOWN_TYPE"
	CodeUnknownKey            Code = "UNKNOWN_KEY"
	CodeMissingPayloadVersion Code = "MISSING_PAYLOAD_VERSION"
	CodeMalformedUUID         Code = "MALFORMED_UUID"
	CodeMalformedIdentifier   Code = "MALFORMED_IDENTIFIER"
	CodeFormatMismatch        Code = "FORMAT_MISMATCH"
	CodeWrongScope            Code = "WRONG_SCOPE"
)

// Finding is one problem in a profile. Callers build messages and log entries from it.
// If we add a "severity" field, we can easily add it later, but it would be harder to remove a useless field.
type Finding struct {
	Code Code `json:"code"`
	// PayloadType is the payload's PayloadType, a declaration's Type, or TopLevel for the profile root.
	// For Windows it's the Policy area, e.g. DeviceLock, or else the CSP, e.g. Firewall.
	PayloadType string `json:"payload_type"`
	// KeyPath is dotted from the payload dictionary or a declaration's Payload, with [n] for array
	// positions, e.g. Services.Camera[0].Allowed. Key names containing dots are written as-is.
	// For Windows it's the LocURI as written.
	KeyPath string `json:"key_path,omitempty"`
	// Expected lists every accepted type.
	Expected []string `json:"expected,omitempty"`
	Actual   string   `json:"actual,omitempty"`
}
