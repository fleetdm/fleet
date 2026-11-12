package etag_invalidate

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEveryWrappedMethodHasAHookCase makes "this decorator wraps X" and "we
// proved X fires an invalidation" the same statement.
//
// Declaring a method here without a case in TestDeploymentHooksFireOnSuccess or
// TestPerHostHooksFireOnSuccess would compile, pass CI, and silently do nothing
// if the d.invalidate(...) call were missing or wrong — the only symptom being
// hosts told {"etag":"ok"} against a changed config until the backstop TTL
// expires. This test closes that gap.
//
// What it deliberately does NOT do is detect a method that *should* have been
// wrapped and wasn't. That was tried: reflection cannot help, because Go
// synthesises wrapper methods in this package for everything promoted from the
// embedded fleet.Datastore, so a promoted method is indistinguishable from a
// declared one at runtime. Snapshotting the whole interface instead was
// rejected as well — fleet.Datastore gained 376 methods across 186 commits in
// the six months before this was written, so a snapshot would fail roughly
// daily in unrelated PRs, and the one-keystroke fix defaults to "not
// config-affecting". A guard that cries wolf every day cannot be trusted on the
// day it is right. Keeping unwrapped methods honest is a review concern; see
// the note on the Datastore type.
func TestEveryWrappedMethodHasAHookCase(t *testing.T) {
	covered := map[string]struct{}{}
	for _, c := range deploymentHookCases {
		covered[c.name] = struct{}{}
	}
	for _, c := range perHostHookCases {
		covered[c.name] = struct{}{}
	}

	var missing []string
	for _, name := range declaredMethods(t) {
		if _, ok := covered[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	require.Empty(t, missing,
		"these methods are declared in etag_invalidate.go but no hook case proves they "+
			"invalidate; add a case to deploymentHookCases or perHostHookCases (or, if the "+
			"method intentionally does not invalidate, drop the wrapper): %v", missing)
}

// declRE matches the decorator's exported method declarations. Source parsing
// is the only way to tell a declared method from one promoted by embedding.
var declRE = regexp.MustCompile(`(?m)^func \(d \*Datastore\) ([A-Z]\w*)\(`)

func declaredMethods(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		require.NoError(t, err)
		for _, m := range declRE.FindAllStringSubmatch(string(src), -1) {
			out = append(out, m[1])
		}
	}
	require.NotEmpty(t, out, "found no declared decorator methods — did the receiver name change?")
	return out
}
