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
// json-formated summary containing:
// - The number of ocurrences of each error
// - A reduced stack trace used for debugging the error
func Aggregate(ctx context.Context) (json.RawMessage, error) {
	empty := json.RawMessage("[]")
	maxTraceLen := 3

	storedErrs, err := Retrieve(ctx)
	if err != nil {
		return empty, Wrap(ctx, err, "retrieve on aggregation")
	}

	aggs := make([]errorAgg, len(storedErrs))
	for i, stored := range storedErrs {
		var em FleetErrorChainJSON
		if err = json.Unmarshal(stored.Error, &em); err != nil {
			return empty, Wrap(ctx, err, "unmarshal on aggregation")
		}

		// build a full stack trace
		stack := em.Cause.Stack
		for _, wrap := range em.Wraps {
			stack = append(stack, wrap.Stack...)
		}

		// store the topmost stack traces for each error
		max := len(stack)
		if max > maxTraceLen {
			max = maxTraceLen
		}

		aggs[i] = errorAgg{stored.Count, stack[:max]}
	}

	return json.Marshal(aggs)
}
