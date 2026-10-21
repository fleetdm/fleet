package profilevalidation

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

// This file contains the embedded raw data, the parsed JSON struct and the type for the entire schema,
// with subtypes and struct functions living in their respective files.

//go:embed outputs/profile_schema.json
var profileValidationSchemaRaw []byte

// current holds the schema in use. It starts as the embedded copy (see ParseProfileValidationSchema)
// and can be swapped for a newer one with Replace without touching callers.
var current atomic.Pointer[ProfileValidationSchema]

// Current returns the schema in use, or nil before ParseProfileValidationSchema has run. A schema is
// never modified after Parse returns it, so callers may hold on to it across a Replace.
func Current() *ProfileValidationSchema {
	return current.Load()
}

// Replace swaps in a schema obtained from Parse. New Current calls see it; schemas already handed out
// are unchanged.
func Replace(s *ProfileValidationSchema) error {
	if s == nil {
		return errors.New("replace profile validation schema: nil schema")
	}
	if s.FormatVersion != FormatVersion {
		return fmt.Errorf("replace profile validation schema: unexpected format version %d", s.FormatVersion)
	}
	if err := s.check(); err != nil {
		return fmt.Errorf("replace profile validation schema: %w", err)
	}
	current.Store(s)
	return nil
}

type ProfileValidationSchema struct {
	FormatVersion     int                          `json:"format_version"`
	Version           Version                      `json:"version"`
	AppleEnvelope     AppleEnvelope                `json:"apple_envelope"`
	AppleProfiles     map[string]*ApplePayloadType `json:"apple_profiles"`
	AppleDeclarations map[string]*ApplePayloadType `json:"apple_declarations"`
	// AppleDeclarationEnvelope holds the keys every DDM declaration shares (Type, Identifier, ...).
	AppleDeclarationEnvelope *ApplePayloadType `json:"apple_declaration_envelope"`
	// Windows maps a leaf LocURI, as declared, to its node.
	Windows map[string]*WindowsNode `json:"windows"`
	// WindowsXSDs maps a sha256 to an XSD document that chr nodes reference by it.
	WindowsXSDs map[string]string `json:"windows_xsds"`
}

// Range is an inclusive numeric range: an Apple key's range or repetition, or a Windows node's range.
// Pointers keep a 0 bound distinct from no bound.
type Range struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

type Version struct {
	Apple            AppleVersion            `json:"apple"`
	Windows          WindowsVersion          `json:"windows"`
	WindowsTemplates WindowsTemplatesVersion `json:"windows_templates"`
	GeneratedAt      time.Time               `json:"generated_at"`
}

// Floors and must-haves catch a reorganized upstream that would otherwise silently shrink the
// schema. Old servers apply them to every later download, so keep them well below today's counts.
// FormatVersion is the schema document format this package reads and the generator writes. A
// change in what an existing field means requires a new version; new optional fields don't.
const FormatVersion = 1

const (
	minAppleProfileTypes             = 100
	minAppleConfigurationDeclaration = 40
	appleConfigurationPrefix         = "com.apple.configuration."
	minWindowsNodes                  = 4000
)

// mustHaveAppleProfileKeys maps payload types to keys that must be present.
var mustHaveAppleProfileKeys = map[string][]string{
	"com.apple.applicationaccess": {"allowCamera"},
}

var mustHaveAppleDeclarations = []string{
	"com.apple.configuration.passcode.settings",
}

var mustHaveWindowsNodes = []string{
	"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock",
}

func ParseProfileValidationSchema() error {
	if len(profileValidationSchemaRaw) == 0 {
		return errors.New("no embedded profile validation schema found")
	}
	schema, err := Parse(profileValidationSchemaRaw)
	if err != nil {
		return fmt.Errorf("parse embedded profile validation schema: %w", err)
	}
	// Error on a second call to surface wiring mistakes; later updates go through Replace.
	if !current.CompareAndSwap(nil, schema) {
		return errors.New("profile validation schema already parsed")
	}
	return nil
}

// Parse decodes and checks a schema document. It's the only way to get a ProfileValidationSchema.
// Unknown fields are ignored so newer documents of the same format version still parse.
func Parse(data []byte) (*ProfileValidationSchema, error) {
	// Check the format version first, so an incompatible document is rejected with a clear error
	// instead of a confusing decode failure.
	var formatVersion struct {
		FormatVersion int `json:"format_version"`
	}
	if err := json.Unmarshal(data, &formatVersion); err != nil {
		return nil, fmt.Errorf("decode format version: %w", err)
	}
	if formatVersion.FormatVersion != FormatVersion {
		return nil, fmt.Errorf("unsupported profile validation schema format version: %d", formatVersion.FormatVersion)
	}

	var schema ProfileValidationSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}
	if err := schema.check(); err != nil {
		return nil, err
	}
	return &schema, nil
}

func (s *ProfileValidationSchema) check() error {
	if err := s.checkApple(); err != nil {
		return err
	}
	return s.checkWindows()
}

func (s *ProfileValidationSchema) checkApple() error {
	if s.AppleEnvelope.TopLevel == nil || s.AppleEnvelope.CommonPayloadKeys == nil {
		return errors.New("apple_envelope is missing TopLevel or CommonPayloadKeys")
	}
	if s.AppleDeclarationEnvelope == nil {
		return errors.New("apple_declaration_envelope is missing")
	}
	n := 0
	for _, p := range s.AppleProfiles {
		if p != nil && p.Source != "override" {
			n++
		}
	}
	if n < minAppleProfileTypes {
		return fmt.Errorf("apple_profiles has %d payload types, want at least %d non-overridden", n, minAppleProfileTypes)
	}
	var configurations int
	for name := range s.AppleDeclarations {
		if strings.HasPrefix(name, appleConfigurationPrefix) {
			configurations++
		}
	}
	if configurations < minAppleConfigurationDeclaration {
		return fmt.Errorf("apple_declarations has %d configuration declarations, want at least %d", configurations, minAppleConfigurationDeclaration)
	}
	for payloadType, keys := range mustHaveAppleProfileKeys {
		p := s.AppleProfiles[payloadType]
		if p == nil {
			return fmt.Errorf("apple_profiles is missing %s", payloadType)
		}
		for _, k := range keys {
			if p.Keys[k] == nil {
				return fmt.Errorf("apple_profiles %s is missing key %s", payloadType, k)
			}
		}
	}
	for _, d := range mustHaveAppleDeclarations {
		if s.AppleDeclarations[d] == nil {
			return fmt.Errorf("apple_declarations is missing %s", d)
		}
	}
	return nil
}

func (s *ProfileValidationSchema) checkWindows() error {
	if len(s.Windows) < minWindowsNodes {
		return fmt.Errorf("windows has %d nodes, want at least %d", len(s.Windows), minWindowsNodes)
	}
	for _, n := range mustHaveWindowsNodes {
		if s.Windows[n] == nil {
			return fmt.Errorf("windows is missing %s", n)
		}
	}
	for _, locURI := range slices.Sorted(maps.Keys(s.Windows)) {
		n := s.Windows[locURI]
		if n == nil {
			return fmt.Errorf("windows %s is null", locURI)
		}
		if n.XSD != "" && s.WindowsXSDs[n.XSD] == "" {
			return fmt.Errorf("windows %s references missing xsd %s", locURI, n.XSD)
		}
	}
	return nil
}

// AppleProfile returns the .mobileconfig payload type, or nil if the schema doesn't know it.
func (s *ProfileValidationSchema) AppleProfile(payloadType string) *ApplePayloadType {
	return s.AppleProfiles[payloadType]
}

// AppleDeclaration returns the DDM declaration type, or nil if the schema doesn't know it.
func (s *ProfileValidationSchema) AppleDeclaration(declarationType string) *ApplePayloadType {
	return s.AppleDeclarations[declarationType]
}
