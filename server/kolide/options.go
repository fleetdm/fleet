package kolide

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// OptionStore interface describes methods to access datastore
type OptionStore interface {
	// SaveOptions transactional write of options to storage.  If one or more
	// values fails validation none of the writes will succeed. Note only option
	// values are written.  Other option fields are created in migration and do
	// not change. Attempting to write ReadOnly options will cause an error.
	SaveOptions(opts []Option, args ...OptionalArg) error
	// Options returns all options
	ListOptions() ([]Option, error)
	// Option return an option by ID
	Option(id uint) (*Option, error)
	// OptionByName returns an option uniquely identified by name
	OptionByName(name string, args ...OptionalArg) (*Option, error)
	// GetOsqueryConfigOptions returns options in a format that will be the options
	// section of osquery configuration
	GetOsqueryConfigOptions() (map[string]interface{}, error)
	// ResetOptions will reset options to their initial values. This should be used
	// with caution as it will remove any options or changes to defaults made by
	// the user. Returns a list of default options.
	ResetOptions() ([]Option, error)
}

// OptionService interface describes methods that operate on osquery options
type OptionService interface {
	// GetOptions retrieves all options
	GetOptions(ctx context.Context) ([]Option, error)
	// ModifyOptions will change values of the options in OptionRequest.  Note
	// passing ReadOnly options will cause an error.
	ModifyOptions(ctx context.Context, req OptionRequest) ([]Option, error)
	// ResetOptions resets all options to their default values
	ResetOptions(ctx context.Context) ([]Option, error)
}

const (
	ReadOnly    = true
	NotReadOnly = !ReadOnly
)

// OptionType defines the type of the value assigned to an option
type OptionType int

const (
	OptionTypeString OptionType = iota
	OptionTypeInt
	OptionTypeBool
)

// String values that map from JSON to OptionType
const (
	optionTypeString = "string"
	optionTypeInt    = "int"
	optionTypeBool   = "bool"
)

// MarshalJSON marshals option type to strings
func (ot OptionType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, ot)), nil
}

// UnmarshalJSON converts json to OptionType
func (ot *OptionType) UnmarshalJSON(b []byte) error {
	switch typ := string(b); strings.Trim(typ, `"`) {
	case optionTypeString:
		*ot = OptionTypeString
	case optionTypeBool:
		*ot = OptionTypeBool
	case optionTypeInt:
		*ot = OptionTypeInt
	default:
		return fmt.Errorf("unsupported option type '%s'", typ)
	}
	return nil
}

// String is used to marshal OptionType to human readable strings used in JSON payloads
func (ot OptionType) String() string {
	switch ot {
	case OptionTypeString:
		return optionTypeString
	case OptionTypeInt:
		return optionTypeInt
	case OptionTypeBool:
		return optionTypeBool
	default:
		panic("stringer not implemented for OptionType")
	}
}

// OptionValue supports Valuer and Scan interfaces for reading and writing
// to the database, and also JSON marshaling
type OptionValue struct {
	Val interface{}
}

// Value is called by the DB driver.  Note that we store data as JSON
// types, so we can use the JSON marshaller to assign the appropriate
// type when we fetch it from the db
func (ov OptionValue) Value() (dv driver.Value, err error) {
	return json.Marshal(ov.Val)
}

// Scan takes db string and turns it into an option type
func (ov *OptionValue) Scan(src interface{}) error {
	if err := json.Unmarshal(src.([]byte), &ov.Val); err != nil {
		return err
	}
	switch v := ov.Val.(type) {
	case float64:
		ov.Val = int(v)
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (ov OptionValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(ov.Val)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (ov *OptionValue) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &ov.Val)
}

// Option represents a possible osquery confguration option
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type Option struct {
	// ID unique identifier for option assigned by the dbms
	ID uint `json:"id"`
	// Name of the option which must be unique
	Name string `json:"name"`
	// Type of value for the option
	Type OptionType `json:"type"`
	// Value of the option which may be nil, bool, int, or string.
	Value OptionValue `json:"value"`
	// ReadOnly if true, this option is required for Fleet to function
	// properly and cannot be modified by the end user
	ReadOnly bool `json:"read_only" db:"read_only"`
}

// SetValue sets the value associated with the option
func (opt *Option) SetValue(v interface{}) {
	opt.Value.Val = v
}

func (opt *Option) SameType(compare interface{}) bool {
	switch compare.(type) {
	case float64:
		return opt.Type == OptionTypeInt
	case string:
		return opt.Type == OptionTypeString
	case bool:
		return opt.Type == OptionTypeBool
	default:
		return false
	}
}

// OptionSet returns true if the option has a value assigned to it
func (opt *Option) OptionSet() bool {
	return opt.Value.Val != nil
}

// GetValue returns the value associated with the option
func (opt Option) GetValue() interface{} {
	return opt.Value.Val
}

// OptionRequest contains options that are passed to modify options requests.
type OptionRequest struct {
	Options []Option `json:"options"`
}
