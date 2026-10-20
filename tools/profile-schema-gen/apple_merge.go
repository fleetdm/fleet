package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

const appleNotSupported = "n/a"

// convertAppleGroup converts every file of a type and merges them into one payload type. Where
// files define the same key differently, the loosest definition wins: newer data must never make a
// validator stricter, and a stricter variant file (e.g. extensiblesso(kerberos)) mustn't reject
// what the general file allows.
func convertAppleGroup(ctx context.Context, logger *slog.Logger, g *appleGroup) (*profilevalidation.ApplePayloadType, error) {
	m := &appleMerger{ctx: ctx, logger: logger.With("type", g.Name)}
	var out *profilevalidation.ApplePayloadType
	for _, f := range g.Files {
		platforms := f.Payload.SupportedOS.resolve(nil)
		keys, open, values, err := newAppleConverter(ctx, logger, f.Path).convertKeys(f.PayloadKeys, "", platforms)
		if err != nil {
			return nil, err
		}
		pt := &profilevalidation.ApplePayloadType{Platforms: platforms, Source: "apple", Keys: keys, Open: open, Values: values}
		if out == nil {
			out = pt
			continue
		}
		m.logger.DebugContext(ctx, "merging file into type", "file", f.Path)
		out.Platforms = m.mergePlatforms("(payload)", out.Platforms, pt.Platforms)
		out.Keys = m.mergeKeyMaps("", out.Keys, pt.Keys)
		out.Open, out.Values = m.mergeOpen("(payload)", out.Open, out.Values, pt.Open, pt.Values)
	}
	trimPlatforms(out.Keys, out.Platforms)
	trimKeyPlatforms(out.Values, out.Platforms)
	return out, nil
}

type appleMerger struct {
	ctx    context.Context
	logger *slog.Logger
}

// changed logs a field whose merged value differs from either input.
func (m *appleMerger) changed(path, field string, a, b, merged any) {
	if reflect.DeepEqual(a, b) {
		return
	}
	m.logger.InfoContext(m.ctx, "merged key to loosest definition", "path", path, "field", field,
		"a", logJSON(a), "b", logJSON(b), "merged", logJSON(merged))
}

func logJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func (m *appleMerger) mergeKeyMaps(parentPath string, a, b map[string]*profilevalidation.AppleKey) map[string]*profilevalidation.AppleKey {
	if a == nil && len(b) > 0 {
		a = map[string]*profilevalidation.AppleKey{}
	}
	for name, bk := range b {
		if ak, ok := a[name]; ok {
			a[name] = m.mergeKey(joinKeyPath(parentPath, name), ak, bk)
		} else {
			a[name] = bk
		}
	}
	return a
}

// mergeKey merges b into a, field by field, toward accepting more.
func (m *appleMerger) mergeKey(path string, a, b *profilevalidation.AppleKey) *profilevalidation.AppleKey {
	out := &profilevalidation.AppleKey{
		Type:       mergeTypes(a.Type, b.Type),
		Required:   a.Required && b.Required,
		RangeList:  mergeRangeList(a.RangeList, b.RangeList),
		Range:      widenRange(a.Range, b.Range),
		Format:     sameOrEmpty(a.Format, b.Format),
		Repetition: widenRange(a.Repetition, b.Repetition),
		ValueType:  sameOrEmpty(a.ValueType, b.ValueType),
		Ref:        a.Ref,
		Keys:       m.mergeKeyMaps(path, a.Keys, b.Keys),
		Platforms:  m.mergePlatforms(path, a.Platforms, b.Platforms),
	}
	out.Open, out.Values = m.mergeOpen(path, a.Open, a.Values, b.Open, b.Values)
	// An array without an item definition accepts any items.
	if a.Items != nil && b.Items != nil {
		out.Items = m.mergeKey(path+"[]", a.Items, b.Items)
	}

	m.changed(path, "type", a.Type, b.Type, out.Type)
	m.changed(path, "required", a.Required, b.Required, out.Required)
	m.changed(path, "rangelist", a.RangeList, b.RangeList, out.RangeList)
	m.changed(path, "range", fmtRange(a.Range), fmtRange(b.Range), fmtRange(out.Range))
	m.changed(path, "format", a.Format, b.Format, out.Format)
	m.changed(path, "repetition", fmtRange(a.Repetition), fmtRange(b.Repetition), fmtRange(out.Repetition))
	m.changed(path, "valuetype", a.ValueType, b.ValueType, out.ValueType)
	m.changed(path, "open", a.Open, b.Open, out.Open)
	if (a.Items == nil) != (b.Items == nil) {
		m.changed(path, "items", a.Items != nil, b.Items != nil, out.Items != nil)
	}
	if a.Ref != b.Ref {
		m.logger.WarnContext(m.ctx, "files disagree on ref; keeping the first", "path", path, "a", a.Ref, "b", b.Ref)
	}
	return out
}

// mergeOpen merges the open/values pair. An open side without values accepts anything, so it wins.
func (m *appleMerger) mergeOpen(path string, aOpen bool, aValues *profilevalidation.AppleKey, bOpen bool, bValues *profilevalidation.AppleKey) (bool, *profilevalidation.AppleKey) {
	switch {
	case aOpen && aValues == nil, bOpen && bValues == nil:
		return aOpen || bOpen, nil
	case aOpen && bOpen:
		return true, m.mergeKey(path+".ANY", aValues, bValues)
	case aOpen:
		return true, aValues
	case bOpen:
		return true, bValues
	}
	return false, nil
}

func (m *appleMerger) mergePlatforms(path string, a, b map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) map[profilevalidation.AppleOS]profilevalidation.ApplePlatform {
	if a == nil {
		return b
	}
	out := make(map[profilevalidation.AppleOS]profilevalidation.ApplePlatform, len(a))
	for os, pa := range a {
		out[os] = pa
	}
	for os, pb := range b {
		pa, ok := out[os]
		if !ok {
			out[os] = pb
			continue
		}
		merged := mergePlatform(pa, pb)
		m.changed(path, "platforms."+string(os), pa, pb, merged)
		out[os] = merged
	}
	return out
}

// mergePlatform keeps the widest support: the earliest introduction, and deprecation, removal and
// supervision only when both sides have them. "n/a" means unsupported, so the other side wins.
func mergePlatform(a, b profilevalidation.ApplePlatform) profilevalidation.ApplePlatform {
	if a.Introduced == appleNotSupported || a.Introduced == "" {
		return b
	}
	if b.Introduced == appleNotSupported || b.Introduced == "" {
		return a
	}
	return profilevalidation.ApplePlatform{
		Introduced: minVersion(a.Introduced, b.Introduced),
		Deprecated: laterVersionIfBoth(a.Deprecated, b.Deprecated),
		Removed:    laterVersionIfBoth(a.Removed, b.Removed),
		Supervised: a.Supervised && b.Supervised,
	}
}

func mergeTypes(a, b profilevalidation.AppleTypes) profilevalidation.AppleTypes {
	out := slices.Clone(a)
	for _, t := range b {
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

// mergeRangeList returns the union, or nil (any value) if either side doesn't restrict values.
func mergeRangeList(a, b []any) []any {
	if a == nil || b == nil {
		return nil
	}
	out := slices.Clone(a)
	for _, v := range b {
		if !slices.ContainsFunc(out, func(o any) bool { return reflect.DeepEqual(o, v) }) {
			out = append(out, v)
		}
	}
	return out
}

// widenRange returns bounds covering both, dropping a bound either side lacks.
func widenRange(a, b *profilevalidation.AppleRange) *profilevalidation.AppleRange {
	if a == nil || b == nil {
		return nil
	}
	out := &profilevalidation.AppleRange{}
	if a.Min != nil && b.Min != nil {
		out.Min = new(min(*a.Min, *b.Min))
	}
	if a.Max != nil && b.Max != nil {
		out.Max = new(max(*a.Max, *b.Max))
	}
	if out.Min == nil && out.Max == nil {
		return nil
	}
	return out
}

func fmtRange(r *profilevalidation.AppleRange) string {
	if r == nil {
		return "none"
	}
	bound := func(f *float64) string {
		if f == nil {
			return "-"
		}
		return strconv.FormatFloat(*f, 'g', -1, 64)
	}
	return bound(r.Min) + ".." + bound(r.Max)
}

func sameOrEmpty[T comparable](a, b T) T {
	if a == b {
		return a
	}
	var zero T
	return zero
}

func minVersion(a, b string) string {
	if compareVersions(a, b) <= 0 {
		return a
	}
	return b
}

func laterVersionIfBoth(a, b string) string {
	if a == "" || b == "" {
		return ""
	}
	if compareVersions(a, b) >= 0 {
		return a
	}
	return b
}

// compareVersions compares dotted versions numerically ("10.10" > "10.9"); missing parts are 0.
// Non-numeric versions fall back to string order.
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range max(len(as), len(bs)) {
		var ap, bp string
		if i < len(as) {
			ap = as[i]
		}
		if i < len(bs) {
			bp = bs[i]
		}
		an, aerr := strconv.Atoi(cmpZero(ap))
		bn, berr := strconv.Atoi(cmpZero(bp))
		if aerr != nil || berr != nil {
			return strings.Compare(a, b)
		}
		if an != bn {
			return an - bn
		}
	}
	return 0
}

func cmpZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}
