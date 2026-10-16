package service

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// orbitConfigFallbackChanges counts the orbit config fields that agents found
// changed on a fallback poll, i.e. that changed without a nudge. Near zero for
// every field means the fallback poll interval can be raised; a field that
// keeps showing up needs a nudge where it is written.
var orbitConfigFallbackChanges = mustNewOrbitConfigFallbackChangesCounter()

func mustNewOrbitConfigFallbackChangesCounter() metric.Int64Counter {
	c, err := otel.Meter("fleet").Int64Counter(
		"fleet.agentws.orbit_config.fallback_changes",
		metric.WithDescription("Count of orbit config fields that agents found changed on a fallback poll, by field"),
		metric.WithUnit("{field}"),
	)
	if err != nil {
		panic(err)
	}
	return c
}

// orbitConfigFields is the set of field names an agent may report in
// fleet.OrbitConfigFallbackChangesHeader; anything else is ignored, bounding
// the metric's cardinality.
var orbitConfigFields = func() map[string]struct{} {
	fields := make(map[string]struct{})
	add := func(typ reflect.Type, prefix string) {
		for f := range typ.Fields() {
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			fields[prefix+name] = struct{}{}
		}
	}
	add(reflect.TypeFor[fleet.OrbitConfig](), "")
	add(reflect.TypeFor[fleet.OrbitConfigNotifications](), "notifications.")
	return fields
}()

type orbitConfigFallbackChangesKey struct{}

// orbitConfigFallbackChangesContext stores the request's
// fleet.OrbitConfigFallbackChangesHeader in the context. It is only read
// once the host is authenticated (see recordOrbitConfigFallbackChanges).
func orbitConfigFallbackChangesContext(ctx context.Context, r *http.Request) context.Context {
	if v := r.Header.Get(fleet.OrbitConfigFallbackChangesHeader); v != "" {
		return context.WithValue(ctx, orbitConfigFallbackChangesKey{}, v)
	}
	return ctx
}

func recordOrbitConfigFallbackChanges(ctx context.Context, logger *slog.Logger, hostID uint) {
	v, _ := ctx.Value(orbitConfigFallbackChangesKey{}).(string)
	if v == "" {
		return
	}
	const maxFields = 64
	var recorded []string
	for field := range strings.SplitSeq(v, ",") {
		field = strings.TrimSpace(field)
		if _, ok := orbitConfigFields[field]; !ok || len(recorded) == maxFields {
			continue
		}
		recorded = append(recorded, field)
		orbitConfigFallbackChanges.Add(ctx, 1, metric.WithAttributes(attribute.String("field", field)))
	}
	if len(recorded) > 0 {
		logger.DebugContext(ctx, "orbit config change found by fallback poll", "host_id", hostID, "fields", recorded)
	}
}
