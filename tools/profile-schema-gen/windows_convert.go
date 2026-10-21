package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

// ddfIgnoredProps are DFProperties elements in Microsoft's DDF v2 schema that the generator
// deliberately drops. Anything else not decoded is reported as unrecognized.
var ddfIgnoredProps = []string{
	"AccessType", "DefaultValue", "Description", "Occurrence", "Scope", "DFTitle", "DFType", "CaseSense",
	"DynamicNodeNaming", "ReplaceBehavior", "RebootBehavior", "GpMapping", "CommonErrorResults",
	"DependencyBehavior", "ConflictResolution", "AtomicRequired",
}

// ddfRangeRe matches "[0-999]", "[(-1)-999]" and "[0,1]".
var ddfRangeRe = regexp.MustCompile(`^\[\(?(-?\d+)\)?[-,]\(?(-?\d+)\)?\]$`)

// convertWindows fills the schema's Windows sections from the loaded DDF nodes.
func convertWindows(ctx context.Context, logger *slog.Logger, entries []ddfEntry, schema *profilevalidation.ProfileValidationSchema) {
	schema.Windows = make(map[string]*profilevalidation.WindowsNode, len(entries))
	schema.WindowsXSDs = map[string]string{}
	// Reported once per name, with the first LocURI seen, so the log stays readable.
	unknownProps := map[string]string{}
	unknownValueTypes := map[string]string{}
	valueTypes := map[string]int{}

	for _, e := range entries {
		p := e.Props
		n := &profilevalidation.WindowsNode{
			Format: profilevalidation.WindowsFormats{profilevalidation.WindowsFormat(p.Format.value())},
			Scope:  profilevalidation.WindowsScopeDevice,
		}
		if strings.HasPrefix(e.LocURI, "./User/") {
			n.Scope = profilevalidation.WindowsScopeUser
		}
		if p.Applicability != nil {
			for b := range strings.SplitSeq(p.Applicability.OsBuildVersion, ",") {
				if b = strings.TrimSpace(b); b != "" {
					n.MinBuild = append(n.MinBuild, b)
				}
			}
		}
		if p.Deprecated != nil {
			n.Deprecated = &profilevalidation.WindowsDeprecated{Build: strings.TrimSpace(p.Deprecated.Build)}
		}
		for _, o := range p.Other {
			if name := o.XMLName.Local; !slices.Contains(ddfIgnoredProps, name) {
				if _, ok := unknownProps[name]; !ok {
					unknownProps[name] = e.LocURI
				}
			}
		}

		if av := p.AllowedValues; av != nil {
			valueTypes[av.ValueType]++
			if av.List != nil {
				n.Delimiter = av.List.Delimiter
			}
			switch av.ValueType {
			case "ENUM":
				n.Enum = enumValues(av)
			case "Flag":
				n.Flag = enumValues(av)
			case "Range":
				m := ddfRangeRe.FindStringSubmatch(strings.TrimSpace(av.Value))
				if m == nil {
					// Leave it unconstrained rather than guess; the schema must only ever be looser.
					logger.WarnContext(ctx, "unrecognized DDF range; leaving the node unconstrained", "loc_uri", e.LocURI, "range", av.Value)
					break
				}
				lo, errLo := strconv.ParseFloat(m[1], 64)
				hi, errHi := strconv.ParseFloat(m[2], 64)
				if errLo != nil || errHi != nil {
					logger.WarnContext(ctx, "unrecognized DDF range; leaving the node unconstrained", "loc_uri", e.LocURI, "range", av.Value)
					break
				}
				n.Range = &profilevalidation.Range{Min: &lo, Max: &hi}
			case "RegEx":
				n.RegEx = av.Value
			case "XSD":
				sum := sha256.Sum256([]byte(av.Value))
				key := hex.EncodeToString(sum[:])
				schema.WindowsXSDs[key] = av.Value
				n.XSD = key
			case "ADMX":
				n.ADMXBacked = true
			case "None":
			case "JSON", "SDDL":
				logger.DebugContext(ctx, "DDF value type not modeled; leaving the node unconstrained", "loc_uri", e.LocURI, "value_type", av.ValueType)
			default:
				if _, ok := unknownValueTypes[av.ValueType]; !ok {
					unknownValueTypes[av.ValueType] = e.LocURI
				}
			}
		}
		schema.Windows[e.LocURI] = n
	}

	for _, name := range slices.Sorted(maps.Keys(unknownProps)) {
		logger.WarnContext(ctx, "unrecognized DDF property; ignored", "property", name, "first_loc_uri", unknownProps[name])
	}
	for _, vt := range slices.Sorted(maps.Keys(unknownValueTypes)) {
		logger.WarnContext(ctx, "unrecognized DDF allowed-values type; nodes left unconstrained", "value_type", vt, "first_loc_uri", unknownValueTypes[vt])
	}
	args := []any{"nodes", len(schema.Windows), "xsds", len(schema.WindowsXSDs)}
	for _, vt := range slices.Sorted(maps.Keys(valueTypes)) {
		args = append(args, "allowed_"+vt, valueTypes[vt])
	}
	logger.InfoContext(ctx, "converted DDF nodes", args...)
}

func enumValues(av *ddfAllowedValues) []string {
	values := make([]string, 0, len(av.Enums))
	for _, e := range av.Enums {
		values = append(values, e.Value)
	}
	return values
}
