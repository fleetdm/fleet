package fleet

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const customHostVitalNameMaxNameLen = 255

type CustomHostVital struct {
	ID        uint   `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

func (h CustomHostVital) AuthzType() string {
	return "custom_host_vital"
}

// HostCustomHostVital is a single host's value for a custom host vital.
type HostCustomHostVital struct {
	CustomHostVitalID uint   `json:"custom_host_vital_id" db:"custom_host_vital_id"`
	Name              string `json:"name" db:"name"`
	Value             string `json:"value" db:"value"`
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
