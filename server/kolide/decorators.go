package kolide

// DecoratorStore methods to manipulate decorator queries.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type DecoratorStore interface {
	// NewDecorator creates a decorator query.
	NewDecorator(decorator *Decorator) (*Decorator, error)
	// DeleteDecorator removes a decorator query.
	DeleteDecorator(id uint) error
	// Decorator retrieves a decorator query with supplied ID.
	Decorator(id uint) (*Decorator, error)
	// ListDecorators returns all decorator queries.
	ListDecorators() ([]*Decorator, error)
}

// DecoratorType refers to the allowable types of decorator queries.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type DecoratorType int

const (
	DecoratorLoad DecoratorType = iota
	DecoratorAlways
	DecoratorInterval
)

// Decorator contains information about a decorator query.
type Decorator struct {
	UpdateCreateTimestamps
	ID   uint
	Type DecoratorType
	// Interval note this is only pertainent for DecoratorInterval type.
	Interval uint
	Query    string
}
