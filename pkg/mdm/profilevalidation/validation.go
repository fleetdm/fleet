package profilevalidation

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
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

func ParseProfileValidationSchema() error {
	if profileValidationSchemaRaw == nil {
		// no-embedded schema
		return errors.New("no embedded profile validation schema found")
	}

	// parse the raw embedded into the JSON struct
	if ProfileValidation != nil {
		// error on multiple calls to see issues, this should only be called, changes should be swapped in place for the parsed schema struct.
		return errors.New("profile validation schema already parsed")
	}

	// first we parse just the format version, so we can reject incompatible versions early instead of failing on JSON parsing.
	type formatVersionOnly struct {
		FormatVersion int `json:"format_version"`
	}

	var formatVersion formatVersionOnly
	if err := json.Unmarshal(profileValidationSchemaRaw, &formatVersion); err != nil {
		return err
	}
	if formatVersion.FormatVersion != 1 {
		return fmt.Errorf("unsupported profile validation schema format version: %d", formatVersion.FormatVersion)
	}

	if err := json.Unmarshal(profileValidationSchemaRaw, &ProfileValidation); err != nil {
		return err
	}

	return nil
}
