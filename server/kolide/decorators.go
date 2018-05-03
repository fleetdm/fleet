package kolide

import (
	"errors"
	"strings"
)

// DEPRECATED
// Decorators are now stored as JSON in the config, so these types are only
// useful for migrating existing Fleet installations.

// DecoratorType refers to the allowable types of decorator queries.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type DecoratorType int

const (
	DecoratorLoad DecoratorType = iota
	DecoratorAlways
	DecoratorInterval
	DecoratorUndefined

	DecoratorLoadName     = "load"
	DecoratorAlwaysName   = "always"
	DecoratorIntervalName = "interval"
)

func (dt DecoratorType) String() string {
	switch dt {
	case DecoratorLoad:
		return DecoratorLoadName
	case DecoratorAlways:
		return DecoratorAlwaysName
	case DecoratorInterval:
		return DecoratorIntervalName
	default:
		return ""
	}
}

func (dt *DecoratorType) MarshalJSON() ([]byte, error) {
	name := dt.String()
	if name == "" {
		return nil, errors.New("Invalid decorator type")
	}
	return []byte(`"` + name + `"`), nil
}

var decNameToType = map[string]DecoratorType{
	DecoratorLoadName:     DecoratorLoad,
	DecoratorAlwaysName:   DecoratorAlways,
	DecoratorIntervalName: DecoratorInterval,
}

func (dt *DecoratorType) UnmarshalJSON(data []byte) error {
	name := strings.Trim(string(data), `"`)
	switch name {
	case DecoratorLoadName:
		*dt = DecoratorLoad
	case DecoratorAlwaysName:
		*dt = DecoratorAlways
	case DecoratorIntervalName:
		*dt = DecoratorInterval
	default:
		*dt = DecoratorUndefined
	}
	return nil
}

// Decorator contains information about a decorator query.
type Decorator struct {
	UpdateCreateTimestamps
	ID uint `json:"id"`
	// Name is an optional human friendly name for the decorator
	Name string        `json:"name"`
	Type DecoratorType `json:"type"`
	// Interval note this is only pertinent for DecoratorInterval type.
	Interval uint   `json:"interval"`
	Query    string `json:"query"`
	// BuiltIn decorators are loaded in migrations and may not be changed
	BuiltIn bool `json:"built_in" db:"built_in"`
}

type DecoratorPayload struct {
	ID            uint           `json:"id"`
	Name          *string        `json:"name"`
	DecoratorType *DecoratorType `json:"type"`
	Interval      *uint          `json:"interval"`
	Query         *string        `json:"query"`
}
