// Package themes contains the curated character-name datasets that dibble
// uses to make seeded data fun to look at in the Fleet UI.
//
// Each theme is a Theme literal. New themes go in their own file alongside
// the existing ones and register themselves with init() calling Register.
//
// Pick is deterministic given a (theme, kind, index) tuple — so the same
// seed produces the same data.
package themes

import (
	"fmt"
	"sort"
	"strings"
)

// Person is one character from a theme.
type Person struct {
	First, Last, Handle string
}

// Named is a generic name+description pair used for policies, software, scripts, etc.
type Named struct {
	Name, Desc string
}

// Theme is a curated set of character references for one piece of media.
type Theme struct {
	Name     string
	Display  string // human-friendly title
	Domain   string // email domain suffix
	Users    []Person
	Teams    []string
	Policies []Named
	Software []Named
	Labels   []string
	Scripts  []Named

	// Suffix, when non-empty, is appended to every generated name so that
	// re-running dibble against an already-seeded Fleet produces fresh
	// entries instead of "already exists" skips.
	Suffix string
}

var registry = map[string]Theme{}

// Register adds a theme to the registry. Called from each theme's init().
func Register(t Theme) {
	if t.Name == "" {
		panic("themes: empty Name")
	}
	registry[t.Name] = t
}

// All returns every registered theme, sorted by Name.
func All() []Theme {
	out := make([]Theme, 0, len(registry))
	for _, t := range registry {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Names returns every registered theme name plus "mix", sorted.
func Names() []string {
	out := []string{"mix"}
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Get returns the named theme, or the mix theme for "mix"/"".
func Get(name string) (Theme, error) {
	if name == "" || name == "mix" {
		return Mix(), nil
	}
	t, ok := registry[name]
	if !ok {
		return Theme{}, fmt.Errorf("unknown theme %q (known: %s)", name, strings.Join(Names(), ", "))
	}
	return t, nil
}

// Mix interleaves every registered theme into one combined Theme. Useful
// when you want maximum chaos in a single seed run.
func Mix() Theme {
	mix := Theme{Name: "mix", Display: "Mix", Domain: "dibble.dev"}
	for _, t := range All() {
		mix.Users = append(mix.Users, t.Users...)
		mix.Teams = append(mix.Teams, t.Teams...)
		mix.Policies = append(mix.Policies, t.Policies...)
		mix.Software = append(mix.Software, t.Software...)
		mix.Labels = append(mix.Labels, t.Labels...)
		mix.Scripts = append(mix.Scripts, t.Scripts...)
	}
	return mix
}

// Email returns a themed email address for the i-th user, wrapping around.
func Email(t Theme, i int) string {
	if len(t.Users) == 0 {
		return fmt.Sprintf("user%d@%s", i, t.domain())
	}
	p := t.Users[i%len(t.Users)]
	handle := p.Handle
	if handle == "" {
		handle = strings.ToLower(p.First)
	}
	// Append the index past the first wrap so we never collide.
	if i >= len(t.Users) {
		handle = fmt.Sprintf("%s%d", handle, i/len(t.Users))
	}
	if t.Suffix != "" {
		// Drop the suffix into the local part so emails stay valid.
		handle = handle + "+" + emailSafe(t.Suffix)
	}
	return fmt.Sprintf("%s@%s", handle, t.domain())
}

// FullName returns the i-th user's display name.
func FullName(t Theme, i int) string {
	if len(t.Users) == 0 {
		return fmt.Sprintf("User %d", i)
	}
	p := t.Users[i%len(t.Users)]
	name := strings.TrimSpace(p.First + " " + p.Last)
	if i >= len(t.Users) {
		name = fmt.Sprintf("%s %d", name, i/len(t.Users))
	}
	return appendSuffix(name, t.Suffix)
}

// TeamName returns the i-th team name, wrapping with a numeric suffix to
// avoid duplicate-name conflicts on the Fleet side.
func TeamName(t Theme, i int) string {
	if len(t.Teams) == 0 {
		return fmt.Sprintf("Team %d", i+1)
	}
	name := t.Teams[i%len(t.Teams)]
	if i >= len(t.Teams) {
		name = fmt.Sprintf("%s %d", name, i/len(t.Teams)+1)
	}
	return appendSuffix(name, t.Suffix)
}

// Pick returns the i-th item of the named slice. kind is one of:
// "policy", "software", "label", "script". Wraps around with a numeric suffix.
func Pick(t Theme, kind string, i int) Named {
	var pool []Named
	switch kind {
	case "policy":
		pool = t.Policies
	case "software":
		pool = t.Software
	case "script":
		pool = t.Scripts
	case "label":
		labels := make([]Named, len(t.Labels))
		for k, l := range t.Labels {
			labels[k] = Named{Name: l, Desc: ""}
		}
		pool = labels
	default:
		return Named{Name: fmt.Sprintf("item-%d", i)}
	}
	if len(pool) == 0 {
		return Named{Name: fmt.Sprintf("%s-%d", kind, i+1)}
	}
	n := pool[i%len(pool)]
	if i >= len(pool) {
		n.Name = fmt.Sprintf("%s %d", n.Name, i/len(pool)+1)
	}
	n.Name = appendSuffix(n.Name, t.Suffix)
	return n
}

// appendSuffix tacks " (suffix)" onto a name when a suffix is set. The
// parens-and-space form reads well in the Fleet UI: "Heart of Gold (b3f1)".
func appendSuffix(name, suffix string) string {
	if suffix == "" {
		return name
	}
	return name + " (" + suffix + ")"
}

// emailSafe lower-cases and strips characters that would otherwise produce
// an invalid email local-part.
func emailSafe(s string) string {
	r := strings.NewReplacer(" ", "-", "(", "", ")", "", "/", "-", "\\", "-")
	return strings.ToLower(r.Replace(s))
}

func (t Theme) domain() string {
	if t.Domain != "" {
		return t.Domain
	}
	return "dibble.dev"
}
