package fleet

import (
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
	return fmt.Sprintf("Couldn't add. Custom host vital%s %s missing from database", plural, strings.Join(tokens, ", "))
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

// Describes an entity that references a custom host vital.
type EntityUsingCustomHostVital struct {
	// "script", "apple_profile", "apple_declaration", "windows_profile",
	// "software_installer", or "setup_experience_script".
	Type string
	// Name is the name of the entity.
	Name string
	// FleetName is the name of the fleet (team) the entity belongs to.
	FleetName string
}

type CustomHostVitalUsedError struct {
	CustomHostVitalID   uint
	CustomHostVitalName string
	Entity              EntityUsingCustomHostVital
}

// Error implements the error interface.
func (e *CustomHostVitalUsedError) Error() string {
	noun, action := "configuration profile", "Please delete the configuration profile and try again."
	switch e.Entity.Type {
	case "script":
		noun, action = "script", "Please edit or delete the script and try again."
	case "software_installer":
		noun, action = "software", "Please edit or delete the software and try again."
	case "setup_experience_script":
		noun, action = "setup experience script", "Please edit or delete the setup experience script and try again."
	}
	return fmt.Sprintf(
		"Custom host vital %q (used as $%s%d) is used by the %q %s in the %q team. %s",
		e.CustomHostVitalName, CustomHostVitalPrefix, e.CustomHostVitalID, e.Entity.Name, noun, e.Entity.FleetName, action,
	)
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
		id, err := strconv.ParseUint(s, 10, 64)
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
