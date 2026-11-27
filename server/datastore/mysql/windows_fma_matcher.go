package mysql

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/pkg/patch_policy"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// windowsFMAMatchesCacheTTL bounds how long ingestion may keep matching against a stale
// set of Windows Fleet-maintained apps. Short, because it delays a newly added app from
// collapsing titles; a var so tests can shorten it further.
var windowsFMAMatchesCacheTTL = 30 * time.Second

// getWindowsFMAMatcherCached returns the matcher built from the Windows FMAs that
// reported program names are matched against, from a short-lived in-process cache. See
// the field comments on Datastore for why this is TTL-only.
func (ds *Datastore) getWindowsFMAMatcherCached(ctx context.Context) (*windowsFMAMatcher, error) {
	ds.windowsFMAMatchesMu.RLock()
	matcher, expiry := ds.windowsFMAMatcher, ds.windowsFMAMatchesExpiry
	ds.windowsFMAMatchesMu.RUnlock()
	if time.Now().Before(expiry) {
		return matcher, nil
	}

	result, err, _ := ds.windowsFMAMatchesSF.Do("windowsFMAMatcher", func() (any, error) {
		fmas, err := ds.GetWindowsFMAMatches(ctx)
		if err != nil {
			return nil, err
		}
		fresh := newWindowsFMAMatcher(fmas)
		ds.windowsFMAMatchesMu.Lock()
		ds.windowsFMAMatcher = fresh
		ds.windowsFMAMatchesExpiry = time.Now().Add(windowsFMAMatchesCacheTTL)
		ds.windowsFMAMatchesMu.Unlock()
		return fresh, nil
	})
	if err != nil {
		return nil, err
	}
	fresh, _ := result.(*windowsFMAMatcher)
	return fresh, nil
}

// clearWindowsFMAMatchesCache drops the cached Windows FMA set, forcing the next
// ingestion to read through. Used by tests, which share a Datastore across cases.
func (ds *Datastore) clearWindowsFMAMatchesCache() {
	ds.windowsFMAMatchesMu.Lock()
	defer ds.windowsFMAMatchesMu.Unlock()
	ds.windowsFMAMatcher = nil
	ds.windowsFMAMatchesExpiry = time.Time{}
}

// expireWindowsFMAMatchesCache backdates the cache expiry, leaving the cached set in
// place. Lets tests exercise the TTL path without sleeping: unlike
// clearWindowsFMAMatchesCache, the next read misses because the entry is stale rather
// than absent, which is what expiry actually looks like.
func (ds *Datastore) expireWindowsFMAMatchesCache() {
	ds.windowsFMAMatchesMu.Lock()
	defer ds.windowsFMAMatchesMu.Unlock()
	ds.windowsFMAMatchesExpiry = time.Now().Add(-time.Second)
}

// windowsFMATitle is the software title a matched Windows program belongs to.
type windowsFMATitle struct {
	titleName string
	titleID   uint
}

// windowsFMAPrefix is one candidate program-name prefix, normalized for comparison,
// paired with the software title it resolves to.
//
// The prefix and the destination are deliberately separate. Matching considers every
// name an app may report under, but the destination is always the title the app's
// installer owns: a Windows FMA's title is never renamed when the catalog name
// changes, so resolving the destination from the catalog name could land software on a
// title no installer owns, leaving the uninstall action hidden and the reconcile pass
// unable to repair it.
type windowsFMAPrefix struct {
	prefix string
	windowsFMATitle
}

// windowsFMAMatcher holds the match candidates of every added Windows FMA. Built once
// per cache refresh so per-row matching never reparses a patch query.
type windowsFMAMatcher struct {
	prefixes []windowsFMAPrefix
	rules    []windowsFMARule
}

func newWindowsFMAMatcher(fmas []fleet.MaintainedApp) *windowsFMAMatcher {
	m := &windowsFMAMatcher{prefixes: make([]windowsFMAPrefix, 0, len(fmas)*3)}
	for _, f := range fmas {
		prefixes := f.WinMatchPrefixes()
		if f.TitleID == nil || len(prefixes) == 0 {
			continue
		}
		title := windowsFMATitle{titleName: f.TitleName, titleID: *f.TitleID}
		ownPrefixes := make([]string, 0, len(prefixes))
		for _, p := range prefixes {
			normalized := normalizeForMatch(p)
			ownPrefixes = append(ownPrefixes, normalized)
			m.prefixes = append(m.prefixes, windowsFMAPrefix{prefix: normalized, windowsFMATitle: title})
		}
		if rule, ok := newWindowsFMARule(f.PatchQuery, title, ownPrefixes); ok {
			m.rules = append(m.rules, rule)
		}
	}
	return m
}

// windowsFMARule is an app's curated programs rule, read back from its installer's
// patch query, with every literal normalized for comparison. All clauses must hold;
// a clause holds when any of its conditions does. The parser only groups name
// conditions, so every condition in a clause tests the same column.
type windowsFMARule struct {
	clauses [][]patch_policy.ProgramsCondition
	windowsFMATitle
}

// newWindowsFMARule returns false for a patch query the parser doesn't understand,
// for a rule that doesn't pin both a name and a publisher (those were written to
// detect an install, and without both they are too loose to decide identity), and for
// a rule that only relaxes the app's own name prefix (see onlyRelaxesPrefix).
func newWindowsFMARule(patchQuery string, title windowsFMATitle, ownPrefixes []string) (windowsFMARule, bool) {
	parsed, ok := patch_policy.ParseWindowsProgramsRule(patchQuery)
	if !ok {
		return windowsFMARule{}, false
	}

	rule := windowsFMARule{windowsFMATitle: title}
	var hasName, hasPublisher bool
	for _, clause := range parsed.Clauses {
		conds := make([]patch_policy.ProgramsCondition, 0, len(clause))
		pins := true
		for _, c := range clause {
			values := make([]string, 0, len(c.Values))
			for _, v := range c.Values {
				v = normalizeForMatch(v)
				// A value of only wildcards, or an empty one, pins nothing.
				pins = pins && strings.ContainsFunc(v, func(r rune) bool { return r != '%' && r != '_' })
				values = append(values, v)
			}
			c.Values = values
			conds = append(conds, c)
			pins = pins && c.Op != patch_policy.OpNotLike
		}
		if pins {
			switch clause[0].Column {
			case patch_policy.ProgramsName:
				hasName = true
			case patch_policy.ProgramsPublisher:
				hasPublisher = true
			}
		}
		rule.clauses = append(rule.clauses, conds)
	}
	return rule, hasName && hasPublisher && !onlyRelaxesPrefix(rule.clauses, ownPrefixes)
}

// onlyRelaxesPrefix reports whether a rule's only name condition is one of the app's
// own (normalized) prefix candidates followed by %, with no version condition, such as
// Git's "Git%". Over the prefix rule it adds nothing but names glued to the candidate,
// which can be another product ("GitKraken") as easily as a glued version ("Git2.45"),
// so it is left out and the prefix rule decides as it does without curated rules.
func onlyRelaxesPrefix(clauses [][]patch_policy.ProgramsCondition, ownPrefixes []string) bool {
	var nameConds []patch_policy.ProgramsCondition
	for _, clause := range clauses {
		switch clause[0].Column {
		case patch_policy.ProgramsVersion:
			return false
		case patch_policy.ProgramsName:
			nameConds = append(nameConds, clause...)
		}
	}
	if len(nameConds) != 1 || nameConds[0].Op != patch_policy.OpLike {
		return false
	}
	literal, ok := strings.CutSuffix(nameConds[0].Values[0], "%")
	if !ok || strings.ContainsAny(literal, "%_") {
		return false
	}
	return slices.Contains(ownPrefixes, literal)
}

// acceptsName reports whether the rule's name clauses accept a normalized name. It
// also returns the length of the longest literal text a matching name condition
// starts with, which is how specific the match is.
func (r *windowsFMARule) acceptsName(name string) (int, bool) {
	var literal int
	for _, clause := range r.clauses {
		if clause[0].Column != patch_policy.ProgramsName {
			continue
		}
		n, ok := windowsFMAClauseHolds(clause, name)
		if !ok {
			return 0, false
		}
		literal = max(literal, n)
	}
	return literal, true
}

// accepts is acceptsName plus the rule's publisher and version clauses.
func (r *windowsFMARule) accepts(name, vendor, version string) (int, bool) {
	literal, ok := r.acceptsName(name)
	if !ok {
		return 0, false
	}
	for _, clause := range r.clauses {
		var value string
		switch clause[0].Column {
		case patch_policy.ProgramsPublisher:
			value = vendor
		case patch_policy.ProgramsVersion:
			value = version
		default:
			continue
		}
		if _, ok := windowsFMAClauseHolds(clause, value); !ok {
			return 0, false
		}
	}
	return literal, true
}

// windowsFMAClauseHolds reports whether any condition of the clause holds, and the
// longest literal text the holding positive conditions matched on.
func windowsFMAClauseHolds(clause []patch_policy.ProgramsCondition, s string) (int, bool) {
	var literal int
	held := false
	for i := range clause {
		if n, ok := windowsFMAConditionHolds(&clause[i], s); ok {
			held = true
			literal = max(literal, n)
		}
	}
	return literal, held
}

// windowsFMAConditionHolds evaluates the condition, returning the length of the
// literal text it matched on: the whole value for an equality, the text before the
// first wildcard for LIKE, and nothing for NOT LIKE.
func windowsFMAConditionHolds(c *patch_policy.ProgramsCondition, s string) (int, bool) {
	if c.Trim {
		s = strings.Trim(s, " ")
	}
	switch c.Op {
	case patch_policy.OpEqual:
		for _, v := range c.Values {
			if s == v {
				return len(v), true
			}
		}
	case patch_policy.OpLike:
		if windowsFMALike(s, c.Values[0]) {
			return len(likeLiteralPrefix(c.Values[0])), true
		}
	case patch_policy.OpNotLike:
		return 0, !windowsFMALike(s, c.Values[0])
	}
	return 0, false
}

// likeLiteralPrefix returns the text a LIKE pattern starts with, before any wildcard.
func likeLiteralPrefix(pattern string) string {
	if i := strings.IndexAny(pattern, "%_"); i >= 0 {
		return pattern[:i]
	}
	return pattern
}

// windowsFMALike reports whether s matches a SQL LIKE pattern, with osquery (SQLite)
// semantics: % matches any run of characters, _ exactly one, and there is no escape
// character. Callers normalize both sides, which makes it case-insensitive.
func windowsFMALike(s, pattern string) bool {
	si, pi := 0, 0
	// Where the most recent % sits in the pattern, and where in s it stopped absorbing.
	// On a mismatch, that % absorbs one more character and matching resumes after it.
	starP, starS := -1, 0
	for si < len(s) {
		if pi < len(pattern) {
			switch pattern[pi] {
			case '%':
				starP, starS = pi, si
				pi++
				continue
			case '_':
				_, n := utf8.DecodeRuneInString(s[si:])
				si += n
				pi++
				continue
			default:
				pr, pn := utf8.DecodeRuneInString(pattern[pi:])
				sr, sn := utf8.DecodeRuneInString(s[si:])
				if pr == sr {
					si += sn
					pi += pn
					continue
				}
			}
		}
		if starP < 0 {
			return false
		}
		_, n := utf8.DecodeRuneInString(s[starS:])
		starS += n
		si, pi = starS, starP+1
	}
	for pi < len(pattern) && pattern[pi] == '%' {
		pi++
	}
	return pi == len(pattern)
}

// match returns the software title a reported Windows program belongs to, or false
// when it matches no Fleet-maintained app.
//
// The name-prefix rule decides, except where exactly one app's curated rule accepts
// the program and that rule is at least as specific as the prefix that matched.
// When several apps' rules accept, the prefix result stands.
func (m *windowsFMAMatcher) match(name, vendor, version string) (windowsFMATitle, bool) {
	normName := normalizeForMatch(name)
	prefixMatch, prefixLen, prefixOK := m.matchPrefix(normName)

	ruleMatch, literalLen, ruleOK := m.matchRules(
		normName,
		normalizeForMatch(vendor),
		normalizeForMatch(version),
	)

	// A rule whose matching name patterns all start with a wildcard pins no name text,
	// and the reconcile pass can't narrow by it, so it never takes a name.
	if ruleOK && literalLen > 0 && literalLen >= prefixLen {
		return ruleMatch, true
	}
	return prefixMatch, prefixOK
}

// ruleMayDecide reports whether, for some publisher and version, a curated rule could
// override the name-prefix result for a normalized name, given the length of the
// prefix that matched it. When it can't, match gives the prefix result whatever the
// publisher and version are.
func (m *windowsFMAMatcher) ruleMayDecide(normalized string, prefixLen int) bool {
	for i := range m.rules {
		if n, ok := m.rules[i].acceptsName(normalized); ok && n > 0 && n >= prefixLen {
			return true
		}
	}
	return false
}

// ruleNamePatterns returns, deduplicated, what the positive name conditions of the
// curated rules can match: exact names, and the literal text LIKE patterns start with.
// Lowercased, which MySQL's case-insensitive collation doesn't mind.
func (m *windowsFMAMatcher) ruleNamePatterns() (exact, prefixes []string) {
	exactSet, prefixSet := make(map[string]struct{}), make(map[string]struct{})
	for _, r := range m.rules {
		for _, clause := range r.clauses {
			for _, c := range clause {
				if c.Column != patch_policy.ProgramsName || c.Op == patch_policy.OpNotLike {
					continue
				}
				for _, v := range c.Values {
					switch {
					// TRIM(name) = 'x' also accepts a name with trailing spaces.
					case c.Op == patch_policy.OpEqual && !c.Trim:
						exactSet[v] = struct{}{}
					case c.Op == patch_policy.OpEqual:
						prefixSet[v] = struct{}{}
					case likeLiteralPrefix(v) != "":
						prefixSet[likeLiteralPrefix(v)] = struct{}{}
					}
				}
			}
		}
	}
	return slices.Sorted(maps.Keys(exactSet)), slices.Sorted(maps.Keys(prefixSet))
}

// matchRules returns the one title whose curated rules accept the program, and the
// longest literal those rules matched on. Rules of different titles accepting is no
// match.
func (m *windowsFMAMatcher) matchRules(name, vendor, version string) (windowsFMATitle, int, bool) {
	var found windowsFMATitle
	var literalLen int
	matched := false
	for i := range m.rules {
		r := &m.rules[i]
		n, ok := r.accepts(name, vendor, version)
		if !ok {
			continue
		}
		if matched && r.titleID != found.titleID {
			return windowsFMATitle{}, 0, false
		}
		found, matched = r.windowsFMATitle, true
		literalLen = max(literalLen, n)
	}
	return found, literalLen, matched
}

// matchPrefix applies the name-prefix rule to a normalized name. It also returns the
// length of the longest matching prefix, even when the match is ambiguous.
//
// A candidate matches when the reported name equals it or begins with
// "<candidate> " — the trailing space is what keeps "Zoombie 5.0" away from the
// "Zoom" FMA. The longest matching candidate wins so a more specific app is
// preferred, and a tie between candidates resolving to different titles is treated as
// no match: there is no principled winner, and merging into the wrong app is worse
// than leaving the title alone. This mirrors the rule the darwin reconcile passes
// apply to a bundle identifier shared by more than one FMA.
func (m *windowsFMAMatcher) matchPrefix(normalized string) (windowsFMATitle, int, bool) {
	var best windowsFMATitle
	var bestLen int
	ambiguous := false
	for _, p := range m.prefixes {
		if len(p.prefix) < bestLen {
			continue
		}
		if normalized != p.prefix && !strings.HasPrefix(normalized, p.prefix+" ") {
			continue
		}
		switch {
		case len(p.prefix) > bestLen:
			best, bestLen, ambiguous = p.windowsFMATitle, len(p.prefix), false
		case p.titleID != best.titleID:
			ambiguous = true
		}
	}

	if ambiguous || bestLen == 0 {
		return windowsFMATitle{}, bestLen, false
	}
	return best, bestLen, true
}
