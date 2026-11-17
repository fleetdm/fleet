package govulndb

import (
	"context"
	_ "embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ToolchainInstallsVersion is the rule format this package reads. Published rules in any other
// format are ignored.
const ToolchainInstallsVersion = "1"

// ToolchainInstalls are the rules that identify Go toolchain installs and read their Go version.
// The mirror publishes them in the artifact, so a rule can be fixed without a Fleet release. The
// mirror also checks that they are safe to apply; this package only compiles them.
type ToolchainInstalls struct {
	Version string                 `json:"version"`
	Rules   []ToolchainInstallRule `json:"rules"`
}

// ToolchainInstallRule claims the software of one source whose name is one of Names or starts
// with one of NamePrefixes. Names are matched exactly, not as patterns, because the iterator
// filters on them in MySQL while matching happens in Go.
type ToolchainInstallRule struct {
	Source       string   `json:"source"`
	Names        []string `json:"names,omitempty"`
	NamePrefixes []string `json:"name_prefixes,omitempty"`
	// Versions are tried in order: the first whose pattern matches and captures a Go version wins.
	Versions []VersionSource `json:"versions"`
}

// VersionSource reads a Go version out of one software field.
type VersionSource struct {
	// Field is "version" or "name".
	Field string `json:"field"`
	// Pattern is an RE2 regular expression whose one capture group is the Go version, with or
	// without a "go" prefix.
	Pattern string `json:"pattern"`
	// Note explains the pattern to whoever edits the rules. It is not read.
	Note string `json:"note,omitempty"`
}

// The rules live here, next to the types, so the mirror and this package's tests read the same
// file; servers only ever apply the copy published in the artifact.
//
//go:embed toolchain_installs.json
var publishedToolchainInstallsJSON []byte

// PublishedToolchainInstalls returns the rules the mirror publishes.
func PublishedToolchainInstalls() *ToolchainInstalls {
	var installs ToolchainInstalls
	if err := json.Unmarshal(publishedToolchainInstallsJSON, &installs); err != nil {
		panic(fmt.Sprintf("decoding Go toolchain install rules: %v", err))
	}
	return &installs
}

// ToolchainRules are compiled ToolchainInstalls. A nil *ToolchainRules claims nothing, so every
// Go toolchain install is left to NVD as before the rules existed.
type ToolchainRules struct {
	bySource map[string][]toolchainRule
}

type toolchainRule struct {
	names    []string
	prefixes []string
	versions []versionSource
}

type versionSource struct {
	fromName bool
	pattern  *regexp.Regexp
}

// NewToolchainRules compiles installs, failing on rules this package cannot apply.
func NewToolchainRules(installs *ToolchainInstalls) (*ToolchainRules, error) {
	if installs == nil {
		return nil, errors.New("no Go toolchain install rules")
	}
	if installs.Version != ToolchainInstallsVersion {
		return nil, fmt.Errorf("Go toolchain install rules have version %q, want %q", installs.Version, ToolchainInstallsVersion)
	}

	rules := &ToolchainRules{bySource: make(map[string][]toolchainRule)}
	for i, r := range installs.Rules {
		compiled, err := compileToolchainRule(r)
		if err != nil {
			return nil, fmt.Errorf("Go toolchain install rule %d (%s): %w", i, r.Source, err)
		}
		rules.bySource[r.Source] = append(rules.bySource[r.Source], compiled)
	}
	return rules, nil
}

func compileToolchainRule(r ToolchainInstallRule) (toolchainRule, error) {
	compiled := toolchainRule{names: r.Names, prefixes: r.NamePrefixes}
	for _, v := range r.Versions {
		if v.Field != "version" && v.Field != "name" {
			return toolchainRule{}, fmt.Errorf("version field %q is not \"version\" or \"name\"", v.Field)
		}
		pattern, err := regexp.Compile(v.Pattern)
		if err != nil {
			return toolchainRule{}, fmt.Errorf("version pattern: %w", err)
		}
		if pattern.NumSubexp() != 1 {
			return toolchainRule{}, fmt.Errorf("version pattern %q needs exactly one capture group", v.Pattern)
		}
		compiled.versions = append(compiled.versions, versionSource{fromName: v.Field == "name", pattern: pattern})
	}
	return compiled, nil
}

// LoadToolchainRules returns the rules published in the newest artifact in vulnPath, or nil when
// there is no artifact yet, it predates the rules, or this server cannot apply them.
func LoadToolchainRules(ctx context.Context, vulnPath string, logger *slog.Logger) *ToolchainRules {
	artifact, err := loadLatestArtifact(ctx, vulnPath)
	if err != nil {
		logger.WarnContext(ctx, "no Go toolchain install rules: reading artifact failed", "err", err)
		return nil
	}
	if artifact == nil || artifact.ToolchainInstalls == nil {
		return nil
	}
	rules, err := NewToolchainRules(artifact.ToolchainInstalls)
	if err != nil {
		logger.WarnContext(ctx, "no Go toolchain install rules: published rules cannot be applied", "err", err)
		return nil
	}
	return rules
}

// IsToolchainInstall reports whether software is an install of the Go toolchain itself, which
// is matched against the Go vulnerability database rather than NVD.
func (r *ToolchainRules) IsToolchainInstall(software *fleet.Software) bool {
	_, ok := r.rule(software)
	return ok
}

func (r *ToolchainRules) rule(software *fleet.Software) (toolchainRule, bool) {
	if r == nil {
		return toolchainRule{}, false
	}
	for _, rule := range r.bySource[software.Source] {
		if slices.Contains(rule.names, software.Name) ||
			slices.ContainsFunc(rule.prefixes, func(p string) bool { return strings.HasPrefix(software.Name, p) }) {
			return rule, true
		}
	}
	return toolchainRule{}, false
}

// iterOptions narrows the toolchain iterator to the rules' sources and names. MySQL's REGEXP
// ignores case and knows nothing about sources, so IsToolchainInstall still makes the exact call.
// It reports false when the rules claim no source: an iterator with no included sources would
// cover all software.
func (r *ToolchainRules) iterOptions() (fleet.SoftwareIterQueryOptions, bool) {
	if r == nil || len(r.bySource) == 0 {
		return fleet.SoftwareIterQueryOptions{}, false
	}
	var sources, alternatives []string
	for source, rules := range r.bySource {
		sources = append(sources, source)
		for _, rule := range rules {
			for _, name := range rule.names {
				alternatives = append(alternatives, regexp.QuoteMeta(name))
			}
			for _, prefix := range rule.prefixes {
				alternatives = append(alternatives, regexp.QuoteMeta(prefix)+".*")
			}
		}
	}
	slices.Sort(sources)
	slices.Sort(alternatives)
	return fleet.SoftwareIterQueryOptions{
		IncludedSources: sources,
		NameMatch:       "^(" + strings.Join(slices.Compact(alternatives), "|") + ")$",
	}, true
}

// version canonicalizes a toolchain install's Go version into semver.
func (rule toolchainRule) version(software *fleet.Software) (string, bool) {
	for _, v := range rule.versions {
		field := software.Version
		if v.fromName {
			field = software.Name
		}
		m := v.pattern.FindStringSubmatch(field)
		if m == nil {
			continue
		}
		if version, ok := toolchainVersion(m[1]); ok {
			return version, true
		}
	}
	return "", false
}

// match returns the stdlib and toolchain advisories affecting a Go toolchain install. Unlike a
// binary, upgrading the toolchain is the fix, so the fixed version is kept.
func (r *ToolchainRules) match(software *fleet.Software, artifact *Artifact) []fleet.SoftwareVulnerability {
	rule, ok := r.rule(software)
	if !ok {
		return nil
	}
	version, ok := rule.version(software)
	if !ok {
		return nil
	}

	vulns := newVulnSet(software.ID)
	vulns.addAffected(artifact.Modules[StdlibModule], version, upgradeTo(""))
	vulns.addAffected(artifact.Modules[ToolchainModule], version, upgradeTo(""))
	return vulns.vulns
}
