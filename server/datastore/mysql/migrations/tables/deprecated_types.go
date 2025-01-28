package tables

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Include a number of types here that were previously used but not needed
// at global scope any more

type decorator struct {
	ID uint `json:"id"`
	// Name is an optional human friendly name for the decorator
	Name string        `json:"name"`
	Type decoratorType `json:"type"`
	// Interval note this is only pertinent for decoratorInterval type.
	Interval uint   `json:"interval"`
	Query    string `json:"query"`
	// BuiltIn decorators are loaded in migrations and may not be changed
	BuiltIn bool `json:"built_in" db:"built_in"`
}

type decorators struct {
	Load     []string            `json:"load,omitempty"`
	Always   []string            `json:"always,omitempty"`
	Interval map[string][]string `json:"interval,omitempty"`
}

// optionType defines the type of the value assigned to an option
type optionType int

const (
	optionTypeString optionType = iota
	optionTypeInt
	optionTypeBool
)

// MarshalJSON marshals option type to strings
func (ot optionType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, ot)), nil
}

// UnmarshalJSON converts json to optionType
func (ot *optionType) UnmarshalJSON(b []byte) error {
	switch typ := string(b); strings.Trim(typ, `"`) {
	case "string":
		*ot = optionTypeString
	case "bool":
		*ot = optionTypeBool
	case "int":
		*ot = optionTypeInt
	default:
		return fmt.Errorf("unsupported option type '%s'", typ)
	}
	return nil
}

// String is used to marshal optionType to human readable strings used in JSON payloads
func (ot optionType) String() string {
	switch ot {
	case optionTypeString:
		return "string"
	case optionTypeInt:
		return "int"
	case optionTypeBool:
		return "bool"
	default:
		panic("stringer not implemented for optionType")
	}
}

// optionValue supports Valuer and Scan interfaces for reading and writing
// to the database, and also JSON marshaling
type optionValue struct {
	Val interface{}
}

// Value is called by the DB driver.  Note that we store data as JSON
// types, so we can use the JSON marshaller to assign the appropriate
// type when we fetch it from the db
func (ov optionValue) Value() (dv driver.Value, err error) {
	return json.Marshal(ov.Val)
}

// Scan takes db string and turns it into an option type
func (ov *optionValue) Scan(src interface{}) error {
	if err := json.Unmarshal(src.([]byte), &ov.Val); err != nil {
		return err
	}
	if v, ok := ov.Val.(float64); ok {
		ov.Val = int(v)
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (ov optionValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(ov.Val)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (ov *optionValue) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &ov.Val)
}

// option represents a possible osquery confguration option
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type option struct {
	// ID unique identifier for option assigned by the dbms
	ID uint `json:"id"`
	// Name of the option which must be unique
	Name string `json:"name"`
	// Type of value for the option
	Type optionType `json:"type"`
	// Value of the option which may be nil, bool, int, or string.
	Value optionValue `json:"value"`
	// ReadOnly if true, this option is required for Fleet to function
	// properly and cannot be modified by the end user
	ReadOnly bool `json:"read_only" db:"read_only"`
}

// GetValue returns the value associated with the option
func (opt option) GetValue() interface{} {
	return opt.Value.Val
}

// decoratorType refers to the allowable types of decorator queries.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type decoratorType int

const (
	decoratorLoad decoratorType = iota
	decoratorAlways
	decoratorInterval
	decoratorUndefined

	decoratorLoadName     = "load"
	decoratorAlwaysName   = "always"
	decoratorIntervalName = "interval"
)

func (dt decoratorType) String() string {
	switch dt {
	case decoratorLoad:
		return decoratorLoadName
	case decoratorAlways:
		return decoratorAlwaysName
	case decoratorInterval:
		return decoratorIntervalName
	default:
		return ""
	}
}

func (dt *decoratorType) MarshalJSON() ([]byte, error) {
	name := dt.String()
	if name == "" {
		return nil, errors.New("Invalid decorator type")
	}
	return []byte(`"` + name + `"`), nil
}

func (dt *decoratorType) UnmarshalJSON(data []byte) error {
	name := strings.Trim(string(data), `"`)
	switch name {
	case decoratorLoadName:
		*dt = decoratorLoad
	case decoratorAlwaysName:
		*dt = decoratorAlways
	case decoratorIntervalName:
		*dt = decoratorInterval
	default:
		*dt = decoratorUndefined
	}
	return nil
}
