package kolide

import "encoding/json"

// JSONLogger defines an interface for loggers that can write JSON to various
// output sources.
type JSONLogger interface {
	// Write writes the JSON log entries to the appropriate destination,
	// returning any errors that occurred.
	Write(logs []json.RawMessage) error
}
