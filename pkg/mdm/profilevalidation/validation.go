package profilevalidation

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// This file contains the embedded raw data, the parsed JSON struct and the type for the entire schema,
// with subtypes and struct functions living in their respective files.

//go:embed outputs/profile_schema.json
var profileValidationSchemaRaw []byte

// ProfileValidation holds the parsed profile validation schema. It is populated by calling ParseProfileValidationSchema().
var ProfileValidation *ProfileValidationSchema

type ProfileValidationSchema struct {
	FormatVersion     int                          `json:"format_version"`
	Version           Version                      `json:"version"`
	AppleEnvelope     AppleEnvelope                `json:"apple_envelope"`
	AppleProfiles     map[string]*ApplePayloadType `json:"apple_profiles"`
	AppleDeclarations map[string]*ApplePayloadType `json:"apple_declarations"`
	// Windows sections are filled by a later change; kept raw until their shape is defined.
	Windows     json.RawMessage `json:"windows"`
	WindowsXSDs json.RawMessage `json:"windows_xsds"`
}

type Version struct {
	Apple       AppleVersion `json:"apple"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// Floors and must-haves catch a reorganized upstream that would otherwise silently shrink the
// schema. Old servers apply them to every later download, so keep them well below today's counts.
const (
	supportedFormatVersion           = 1
	minAppleProfileTypes             = 100
	minAppleConfigurationDeclaration = 40
	appleConfigurationPrefix         = "com.apple.configuration."
)

// mustHaveAppleProfileKeys maps payload types to keys that must be present.
var mustHaveAppleProfileKeys = map[string][]string{
	"com.apple.applicationaccess": {"allowCamera"},
}

var mustHaveAppleDeclarations = []string{
	"com.apple.configuration.passcode.settings",
}

func ParseProfileValidationSchema() error {
	if len(profileValidationSchemaRaw) == 0 {
		return errors.New("no embedded profile validation schema found")
	}
	if ProfileValidation != nil {
		// error on multiple calls to see issues, this should only be called, changes should be swapped in place for the parsed schema struct.
		return errors.New("profile validation schema already parsed")
	}

	schema, err := Parse(profileValidationSchemaRaw)
	if err != nil {
		return fmt.Errorf("parse embedded profile validation schema: %w", err)
	}
	ProfileValidation = schema
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
	if formatVersion.FormatVersion != supportedFormatVersion {
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
	if s.AppleEnvelope.TopLevel == nil || s.AppleEnvelope.CommonPayloadKeys == nil {
		return errors.New("apple_envelope is missing TopLevel or CommonPayloadKeys")
	}
	if n := len(s.AppleProfiles); n < minAppleProfileTypes {
		return fmt.Errorf("apple_profiles has %d payload types, want at least %d", n, minAppleProfileTypes)
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
