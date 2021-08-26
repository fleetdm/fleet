package fleet

import (
	"context"
	"encoding/json"
)

// JSONLogger defines an interface for loggers that can write JSON to various
// output sources.
type JSONLogger interface {
	// Write writes the JSON log entries to the appropriate destination,
	// returning any errors that occurred.
	Write(ctx context.Context, logs []json.RawMessage) error
}
