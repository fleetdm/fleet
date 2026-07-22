package fleet

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const CustomHostVitalPrefix = "FLEET_HOST_VITAL_"

const customHostVitalNameMaxNameLen = 255

type CustomHostVital struct {
	ID        uint   `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

func (h CustomHostVital) AuthzType() string {
	return "custom_vital"
}

// HostCustomHostVital is a single host's value for a custom host vital.
type HostCustomHostVital struct {
	CustomHostVitalID uint   `json:"custom_host_vital_id" db:"custom_host_vital_id"`
	Name              string `json:"name" db:"name"`
	Value             string `json:"value" db:"value"`
}

// HostCustomHostVitalValue is the authz subject for setting a host's custom host vital value.
type HostCustomHostVitalValue struct {
	TeamID *uint `json:"team_id" renameto:"fleet_id"`
}

func (HostCustomHostVitalValue) AuthzType() string {
	return "host_custom_vital"
}

type MissingCustomHostVitalsError struct {
	MissingIDs []uint
}

func (e MissingCustomHostVitalsError) Error() string {
	tokens := make([]string, 0, len(e.MissingIDs))
	for _, id := range e.MissingIDs {
		tokens = append(tokens, fmt.Sprintf("\"$%s%d\"", CustomHostVitalPrefix, id))
	}
	plural := ""
	if len(tokens) > 1 {
		plural = "s"
	}
	return fmt.Sprintf("Custom host vital%s %s is not defined", plural, strings.Join(tokens, ", "))
}

// InvalidCustomHostVitalRefError is returned on upload when a document contains a
// $FLEET_HOST_VITAL_<x> token whose <x> is not a valid custom host vital ID
// (a positive integer) — e.g. a typo like $FLEET_HOST_VITAL_asset_tag.
type InvalidCustomHostVitalRefError struct {
	// Refs are the offending tokens without the leading '$', e.g. "FLEET_HOST_VITAL_asset_tag".
	Refs []string
}

func (e InvalidCustomHostVitalRefError) Error() string {
	tokens := make([]string, 0, len(e.Refs))
	for _, r := range e.Refs {
		tokens = append(tokens, fmt.Sprintf("\"$%s\"", r))
	}
	plural := ""
	if len(tokens) > 1 {
		plural = "s"
	}
	return fmt.Sprintf(
		"Invalid custom host vital reference%s %s; the value after $%s must be a custom host vital ID",
		plural, strings.Join(tokens, ", "), CustomHostVitalPrefix,
	)
}

// MissingCustomHostVitalValueError is returned when expanding $FLEET_HOST_VITAL_<id>
// at delivery time for a host that has no value set for that (existing) vital.
// Distinct from MissingCustomHostVitalsError (upload-time: the id doesn't exist)
// so the delivery failure detail shown to admins names the real cause.
type MissingCustomHostVitalValueError struct {
	MissingIDs []uint
}

func (e MissingCustomHostVitalValueError) Error() string {
	tokens := make([]string, 0, len(e.MissingIDs))
	for _, id := range e.MissingIDs {
		tokens = append(tokens, fmt.Sprintf("\"$%s%d\"", CustomHostVitalPrefix, id))
	}
	plural := ""
	if len(tokens) > 1 {
		plural = "s"
	}
	return fmt.Sprintf("Couldn't populate custom host vital%s %s: no value set for this host", plural, strings.Join(tokens, ", "))
}

// IsInvalidReferencedCustomHostVitalsError reports whether err is a user-input validation failure:
// - an unknown vital ID (MissingCustomHostVitalsError)
// - or a malformed $FLEET_HOST_VITAL_<x> reference (InvalidCustomHostVitalRefError)
func IsInvalidReferencedCustomHostVitalsError(err error) bool {
	var missing *MissingCustomHostVitalsError
	var invalid *InvalidCustomHostVitalRefError
	return errors.As(err, &missing) || errors.As(err, &invalid)
}

// CustomHostVitalEntity identifies the kind of entity that can reference a custom host vital.
type CustomHostVitalEntity string

const (
	CustomHostVitalEntityScript                CustomHostVitalEntity = "script"
	CustomHostVitalEntityAppleProfile          CustomHostVitalEntity = "apple_profile"
	CustomHostVitalEntityAppleDeclaration      CustomHostVitalEntity = "apple_declaration"
	CustomHostVitalEntityWindowsProfile        CustomHostVitalEntity = "windows_profile"
	CustomHostVitalEntitySoftwareInstaller     CustomHostVitalEntity = "software_installer"
	CustomHostVitalEntitySetupExperienceScript CustomHostVitalEntity = "setup_experience_script"
	CustomHostVitalEntityLabel                 CustomHostVitalEntity = "label"
	CustomHostVitalEntityHostNameTemplate      CustomHostVitalEntity = "host_name_template"
)

// Describes an entity that references a custom host vital.
type EntityUsingCustomHostVital struct {
	Type CustomHostVitalEntity
	// Name is the name of the entity.
	Name string
	// FleetName is the name of the fleet (team) the entity belongs to.
	FleetName string
}

// CustomHostVitalUsedInfo describes a script/profile/declaration that references a custom host vital.
type CustomHostVitalUsedInfo struct {
	CustomHostVitalID   uint
	CustomHostVitalName string
	Entity              EntityUsingCustomHostVital
}

// Message returns the human-readable "X is used by Y" explanation.
func (i CustomHostVitalUsedInfo) Message() string {
	if i.Entity.Type == CustomHostVitalEntityHostNameTemplate {
		// there's no separate entity name to report, just the fleet whose template references the vital.
		return fmt.Sprintf(
			"Custom host vital %q (used as $%s%d) is used by the host name template in the %q fleet. Please edit or clear the host name template and try again.",
			i.CustomHostVitalName, CustomHostVitalPrefix, i.CustomHostVitalID, i.Entity.FleetName,
		)
	}

	noun, action := "configuration profile", "Please delete the configuration profile and try again."
	switch i.Entity.Type {
	case CustomHostVitalEntityScript:
		noun, action = "script", "Please edit or delete the script and try again."
	case CustomHostVitalEntitySoftwareInstaller:
		noun, action = "software", "Please edit or delete the software and try again."
	case CustomHostVitalEntitySetupExperienceScript:
		noun, action = "setup experience script", "Please edit or delete the setup experience script and try again."
	case CustomHostVitalEntityLabel:
		noun, action = "label", "Please edit or delete the label and try again."
	}
	return fmt.Sprintf(
		"Custom host vital %q (used as $%s%d) is used by the %q %s in the %q fleet. %s",
		i.CustomHostVitalName, CustomHostVitalPrefix, i.CustomHostVitalID, i.Entity.Name, noun, i.Entity.FleetName, action,
	)
}

// CustomHostVitalUsedError wraps CustomHostVitalUsedInfo as an error, returned when
// a custom host vital can't be deleted because it is still referenced.
type CustomHostVitalUsedError struct {
	CustomHostVitalUsedInfo
}

func (e *CustomHostVitalUsedError) Error() string {
	return e.Message()
}

func ValidateCustomHostVitalName(name string) error {
	if len(name) == 0 {
		return NewInvalidArgumentError("name", "custom host vital name cannot be empty")
	}
	if strings.TrimSpace(name) != name {
		return NewInvalidArgumentError("name", "custom host vital name cannot have leading or trailing whitespace")
	}
	if utf8.RuneCountInString(name) > customHostVitalNameMaxNameLen {
		return NewInvalidArgumentError("name", fmt.Sprintf("custom host vital name is too long: %s", name))
	}
	return nil
}

func ContainsCustomHostVitalIDs(text string) []uint {
	suffixes := ContainsPrefixVars(text, CustomHostVitalPrefix)
	if len(suffixes) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(suffixes))
	ids := make([]uint, 0, len(suffixes))
	for _, s := range suffixes {
		id, err := strconv.ParseUint(s, 10, strconv.IntSize)
		if err != nil || id == 0 {
			continue
		}
		if _, ok := seen[uint(id)]; ok {
			continue
		}
		seen[uint(id)] = struct{}{}
		ids = append(ids, uint(id))
	}
	return ids
}

// ContainsMalformedCustomHostVitalRefs returns the $FLEET_HOST_VITAL_<x> tokens in
// text whose <x> is not a valid custom host vital ID (a positive integer), e.g. a
// typo like $FLEET_HOST_VITAL_asset_tag.
// Returned tokens omit the leading '$' (e.g. "FLEET_HOST_VITAL_asset_tag").
func ContainsMalformedCustomHostVitalRefs(text string) []string {
	var malformed []string
	for _, s := range ContainsPrefixVars(text, CustomHostVitalPrefix) {
		if id, err := strconv.ParseUint(s, 10, strconv.IntSize); err != nil || id == 0 {
			malformed = append(malformed, CustomHostVitalPrefix+s)
		}
	}
	return malformed
}
