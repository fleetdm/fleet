package ctxerr

import (
	"context"
	"encoding/json"
)

type errorAgg struct {
	Count int      `json:"count"`
	Loc   []string `json:"loc"`
}

// Aggregate retrieves all errors in the store and returns an aggregated,
// json-formatted summary containing:
// - The number of occurrences of each error
// - A reduced stack trace used for debugging the error
func Aggregate(ctx context.Context) (json.RawMessage, error) {
	const maxTraceLen = 3
	empty := json.RawMessage("[]")

	storedErrs, err := Retrieve(ctx)
	if err != nil {
		return empty, Wrap(ctx, err, "retrieve on aggregation")
	}

	aggs := make([]errorAgg, len(storedErrs))
	for i, stored := range storedErrs {
		var ferr []fleetErrorJSON
		if err = json.Unmarshal(stored.Chain, &ferr); err != nil {
			return empty, Wrap(ctx, err, "unmarshal on aggregation")
		}

		stack := aggregateStack(ferr, maxTraceLen)
		aggs[i] = errorAgg{stored.Count, stack}
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
