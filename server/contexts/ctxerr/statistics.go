package ctxerr

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type ErrorAgg struct {
	Count    int             `json:"count"`
	Loc      []string        `json:"loc"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// Aggregate retrieves all errors in the store and returns an aggregated,
// json-formatted summary containing:
// - The number of occurrences of each error
// - A reduced stack trace used for debugging the error
// - Additional metadata present for vital errors
func Aggregate(ctx context.Context) (json.RawMessage, error) {
	const maxTraceLen = 3
	empty := json.RawMessage("[]")

	storedErrs, err := Retrieve(ctx)
	if err != nil {
		return empty, Wrap(ctx, err, "retrieve on aggregation")
	}

	aggs := make([]ErrorAgg, len(storedErrs))
	for i, stored := range storedErrs {
		var ferr []fleetErrorJSON
		if err = json.Unmarshal(stored.Chain, &ferr); err != nil {
			return empty, Wrap(ctx, err, "unmarshal on aggregation")
		}

		stack := aggregateStack(ferr, maxTraceLen)
		meta := getVitalMetadata(ferr)
		aggs[i] = ErrorAgg{stored.Count, stack, meta}
	}

	return json.Marshal(aggs)
}

// aggregateStack creates a single stack trace by joining all the stack traces in
// an error chain
func aggregateStack(chain []fleetErrorJSON, max int) []string {
	stack := make([]string, max)
	stackIdx := 0

out:
	for _, e := range chain {
		for _, m := range e.Stack {
			if stackIdx >= max {
				break out
			}

			stack[stackIdx] = m
			stackIdx++
		}
	}

	return stack[:stackIdx]
}

func getVitalMetadata(chain []fleetErrorJSON) json.RawMessage {
	for _, e := range chain {
		if len(e.Data) > 0 {
			// Currently, only vital fleetd errors contain metadata.
			// Note: vital errors should not contain any sensitive info
			var fleetdErr fleet.FleetdError
			var err error
			if err = json.Unmarshal(e.Data, &fleetdErr); err != nil || !fleetdErr.Vital {
				continue
			}
			var export = map[string]interface{}{
				"error_source":          fleetdErr.ErrorSource,
				"error_source_version":  fleetdErr.ErrorSourceVersion,
				"error_message":         fleetdErr.ErrorMessage,
				"error_additional_info": fleetdErr.ErrorAdditionalInfo,
			}
			var meta json.RawMessage
			if meta, err = json.Marshal(export); err != nil {
				return nil
			}
			return meta
		}
	}
	return nil
}
