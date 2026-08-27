// Package evidence gathers multi-signal facts about AI tools and scores
// candidate detections. Collectors remain responsible for row assembly;
// this package is read-only and never executes discovered binaries.
package evidence

import (
	"sort"
	"strings"
)

// Weights for independent signal families (design §3).
const (
	WeightCatalog      = 100
	WeightFramework    = 45
	WeightShapeStrong  = 50
	WeightShapeWeak    = 15
	WeightToolHome     = 25
	WeightBinary       = 20
	WeightPkg          = 25
	WeightRunning      = 20
	WeightEgress       = 25
	WeightOSUnit       = 20
	WeightMCPConfig    = 15
	WeightInstructions = 10
	WeightNameMatch    = 5

	// Emit floors.
	EmitScoreMin       = 40
	EmitMultiSignalMin = 30
)

// Signals is a set of evidence tokens (e.g. "tool_home", "framework:crewai").
type Signals map[string]bool

// Add records a token.
func (s Signals) Add(tok string) {
	if tok == "" {
		return
	}
	if s == nil {
		return
	}
	s[tok] = true
}

// List returns sorted tokens for stable evidence CSV.
func (s Signals) List() []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0, len(s))
	for t := range s {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// CSV joins tokens as a comma-separated evidence string.
func (s Signals) CSV() string {
	return strings.Join(s.List(), ",")
}

// Score computes 0–100 confidence from signal tokens.
func Score(s Signals) int {
	if s == nil || len(s) == 0 {
		return 0
	}
	if s["catalog"] {
		return 100
	}
	total := 0
	// Family: framework (any framework:* counts once)
	if hasPrefix(s, "framework:") {
		total += WeightFramework
	}
	if s["workspace_shape"] {
		// Strong shape is tagged workspace_shape; weak uses workspace_shape_weak.
		total += WeightShapeStrong
	} else if s["workspace_shape_weak"] {
		total += WeightShapeWeak
	}
	if s["tool_home"] {
		total += WeightToolHome
	}
	if s["binary"] {
		total += WeightBinary
	}
	if s["pkg"] {
		total += WeightPkg
	}
	if s["running"] {
		total += WeightRunning
	}
	if s["ai_egress"] || s["mcp_egress"] {
		total += WeightEgress
	}
	if s["launchd"] || s["systemd"] || s["registry"] {
		total += WeightOSUnit
	}
	// mcp_config / instructions only add when not already counted via strong shape
	// (they still contribute if shape is weak or absent).
	if !s["workspace_shape"] {
		if s["mcp_config"] {
			total += WeightMCPConfig
		}
		if s["instructions"] {
			total += WeightInstructions
		}
	}
	if s["name_match"] {
		total += WeightNameMatch
	}
	if total > 100 {
		return 100
	}
	return total
}

// ShouldEmit implements two-tier Tier B rules (design §3).
// Catalog is always emit (handled by callers forcing catalog).
func ShouldEmit(s Signals) bool {
	if s == nil || len(s) == 0 {
		return false
	}
	if s["catalog"] {
		return true
	}
	// Name alone never emits.
	if onlyName(s) {
		return false
	}
	if hasPrefix(s, "framework:") {
		return true
	}
	if s["workspace_shape"] { // strong shape — aggressive offline emit
		return true
	}
	sc := Score(s)
	if sc >= EmitScoreMin {
		return true
	}
	if families(s) >= 2 && sc >= EmitMultiSignalMin {
		return true
	}
	return false
}

func onlyName(s Signals) bool {
	for t := range s {
		if t != "name_match" {
			return false
		}
	}
	return s["name_match"]
}

func hasPrefix(s Signals, prefix string) bool {
	for t := range s {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

// families counts independent signal families (excluding name_match).
func families(s Signals) int {
	n := 0
	if hasPrefix(s, "framework:") {
		n++
	}
	if s["workspace_shape"] || s["workspace_shape_weak"] {
		n++
	}
	if s["tool_home"] {
		n++
	}
	if s["binary"] {
		n++
	}
	if s["pkg"] {
		n++
	}
	if s["running"] {
		n++
	}
	if s["ai_egress"] || s["mcp_egress"] {
		n++
	}
	if s["launchd"] || s["systemd"] || s["registry"] {
		n++
	}
	if s["mcp_config"] {
		n++
	}
	if s["instructions"] {
		n++
	}
	return n
}
