package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

const (
	appleNotSupported   = "n/a"
	appleServerTokenKey = "ServerToken"
)

// convertApple fills the Apple sections of schema from the loaded sources.
func convertApple(ctx context.Context, logger *slog.Logger, src *appleSources, schema *profilevalidation.ProfileValidationSchema) error {
	var err error
	if schema.AppleEnvelope.TopLevel, err = convertAppleGroup(ctx, logger, src.TopLevel); err != nil {
		return err
	}
	if schema.AppleEnvelope.CommonPayloadKeys, err = convertAppleGroup(ctx, logger, src.CommonPayloadKeys); err != nil {
		return err
	}
	if schema.AppleDeclarationEnvelope, err = convertAppleGroup(ctx, logger, src.DeclarationBase); err != nil {
		return err
	}
	// Fleet's server sets ServerToken when it serves a declaration, so uploads must never be
	// validated for it.
	if _, ok := schema.AppleDeclarationEnvelope.Keys[appleServerTokenKey]; ok {
		delete(schema.AppleDeclarationEnvelope.Keys, appleServerTokenKey)
		logger.InfoContext(ctx, "dropped server-managed key from the declaration envelope", "key", appleServerTokenKey)
	}
	if schema.AppleProfiles, err = convertAppleGroups(ctx, logger, src.Profiles); err != nil {
		return err
	}
	if schema.AppleDeclarations, err = convertAppleGroups(ctx, logger, src.Declarations); err != nil {
		return err
	}
	logger.InfoContext(ctx, "converted Apple sources", "payload_types", len(schema.AppleProfiles), "declaration_types", len(schema.AppleDeclarations))
	return nil
}

func convertAppleGroups(ctx context.Context, logger *slog.Logger, groups map[string]*appleGroup) (map[string]*profilevalidation.ApplePayloadType, error) {
	out := make(map[string]*profilevalidation.ApplePayloadType, len(groups))
	for name, g := range groups {
		pt, err := convertAppleGroup(ctx, logger, g)
		if err != nil {
			return nil, err
		}
		out[name] = pt
	}
	return out, nil
}

// convertAppleGroup converts every file of a type and merges them into one payload type. Where
// files define the same key differently, the loosest definition wins: newer data must never make a
// validator stricter, and a stricter variant file (e.g. extensiblesso(kerberos)) mustn't reject
// what the general file allows.
func convertAppleGroup(ctx context.Context, logger *slog.Logger, g *appleGroup) (*profilevalidation.ApplePayloadType, error) {
	if g == nil {
		return nil, errors.New("convert Apple group: no group")
	}
	m := &appleMerger{ctx: ctx, logger: logger.With("type", g.Name)}
	var merged *profilevalidation.AppleKey
	for _, f := range g.Files {
		root, err := newAppleConverter(ctx, logger, f.Path).convertPayload(&f.PayloadKeys, f.Payload.SupportedOS)
		if err != nil {
			return nil, err
		}
		if merged == nil {
			merged = root
			continue
		}
		m.logger.DebugContext(ctx, "merging file into type", "file", f.Path)
		merged = m.mergeKey("", merged, root)
	}
	if merged == nil {
		return nil, fmt.Errorf("convert %s: no files", g.Name)
	}
	trimPlatforms(merged, nil)
	return &profilevalidation.ApplePayloadType{
		Platforms: merged.Platforms,
		Source:    profilevalidation.AppleSourceApple,
		Keys:      merged.Keys,
		Open:      merged.Open,
		Values:    merged.Values,
	}, nil
}

// trimPlatforms replaces k's resolved platforms, and its descendants', with only those that differ
// from the parent's. It runs after merging, so differences are relative to the merged parent.
func trimPlatforms(k *profilevalidation.AppleKey, parent map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) {
	if k == nil {
		return
	}
	resolved := k.Platforms
	for _, child := range k.Keys {
		trimPlatforms(child, resolved)
	}
	trimPlatforms(k.Items, resolved)
	trimPlatforms(k.Values, resolved)
	k.Platforms = platformsDiff(resolved, parent)
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
	if path == "" {
		path = "(payload)"
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
	out.Open, out.Values = m.mergeOpen(path, a, b)
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
func (m *appleMerger) mergeOpen(path string, a, b *profilevalidation.AppleKey) (bool, *profilevalidation.AppleKey) {
	switch {
	case a.Open && a.Values == nil, b.Open && b.Values == nil:
		return a.Open || b.Open, nil
	case a.Open && b.Open:
		return true, m.mergeKey(joinKeyPath(path, appleAnyKey), a.Values, b.Values)
	case a.Open:
		return true, a.Values
	case b.Open:
		return true, b.Values
	}
	return false, nil
}

func (m *appleMerger) mergePlatforms(path string, a, b map[profilevalidation.AppleOS]profilevalidation.ApplePlatform) map[profilevalidation.AppleOS]profilevalidation.ApplePlatform {
	if a == nil {
		return b
	}
	out := maps.Clone(a)
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
// supervision only when both sides have them. "n/a" means unsupported, so the other side wins. An
// empty introduced also loses: it happens when a key's supportedOS sets only other fields for a
// platform its parent doesn't list.
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
func widenRange(a, b *profilevalidation.Range) *profilevalidation.Range {
	if a == nil || b == nil {
		return nil
	}
	out := &profilevalidation.Range{}
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

func fmtRange(r *profilevalidation.Range) string {
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

// compareVersions compares dotted versions numerically ("10.10" > "10.9"), falling back to string
// order for anything that isn't a version.
func compareVersions(a, b string) int {
	va, errA := semver.NewVersion(a)
	vb, errB := semver.NewVersion(b)
	if errA != nil || errB != nil {
		return strings.Compare(a, b)
	}
	return va.Compare(vb)
}
