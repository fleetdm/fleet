// Package preservedsources carries, through the context of a distributed query results request,
// the software sources whose current rows the host's software ingestion must keep because the
// extra query reporting them did not run.
package preservedsources

import "context"

type key int

const sourcesKey key = 0

// NewContext returns a new context carrying the software sources to keep.
func NewContext(ctx context.Context, sources []string) context.Context {
	if len(sources) == 0 {
		return ctx
	}
	return context.WithValue(ctx, sourcesKey, sources)
}

// FromContext returns the software sources to keep, if any.
func FromContext(ctx context.Context) []string {
	sources, _ := ctx.Value(sourcesKey).([]string)
	return sources
}
