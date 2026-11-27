package patch_policy

import (
	"strings"
)

// ProgramsColumn is a column of osquery's programs table that a ProgramsRule tests.
type ProgramsColumn string

const (
	ProgramsName      ProgramsColumn = "name"
	ProgramsPublisher ProgramsColumn = "publisher"
	ProgramsVersion   ProgramsColumn = "version"
)

// ProgramsOp is the comparison a ProgramsCondition applies, with osquery (SQLite)
// semantics: LIKE has no escape character.
type ProgramsOp int

const (
	// OpEqual holds when the column equals any of Values: one for =, several for IN.
	OpEqual ProgramsOp = iota
	OpLike
	OpNotLike
)

// ProgramsCondition is one comparison against a programs column. Values holds one
// literal, or every listed literal for IN. Trim is set for TRIM(column).
type ProgramsCondition struct {
	Column ProgramsColumn
	Op     ProgramsOp
	Values []string
	Trim   bool
}

// ProgramsRule is the programs check of a Windows patch policy, read back as a
// conjunction of clauses. A clause holds when any of its conditions does; only a
// parenthesized OR of name conditions yields more than one.
type ProgramsRule struct {
	Clauses [][]ProgramsCondition
}

// ParseWindowsProgramsRule reads the programs check back out of a Windows patch policy
// query built by GenerateQueryForManifest, optionally wrapped by ScopeToWindowsARM.
//
// It fails closed: any term it doesn't model fails the whole parse, because dropping a
// term would widen the rule (a version clause is what separates QGIS LTR from QGIS).
func ParseWindowsProgramsRule(patchQuery string) (ProgramsRule, bool) {
	toks, ok := tokenizeSQL(patchQuery)
	if !ok {
		return ProgramsRule{}, false
	}
	p := &ruleParser{toks: toks}

	armScoped := p.peekWords("SELECT", "1", "WHERE", "EXISTS")
	if armScoped && !p.words("SELECT", "1", "WHERE", "EXISTS", "(") {
		return ProgramsRule{}, false
	}
	if !p.words("SELECT", "1", "WHERE", "NOT", "EXISTS", "(", "SELECT", "1", "FROM", "programs", "WHERE") {
		return ProgramsRule{}, false
	}
	clauses, ok := p.conjunction(true)
	if !ok || len(clauses) == 0 || !p.words(")") {
		return ProgramsRule{}, false
	}
	if armScoped {
		if !p.words(")", "OR", "NOT", "EXISTS", "(", "SELECT", "1", "FROM", "system_info", "WHERE", "cpu_type", "LIKE") {
			return ProgramsRule{}, false
		}
		if s, ok := p.str(); !ok || s != "ARM%" || !p.words(")") {
			return ProgramsRule{}, false
		}
	}
	p.words(";")
	if !p.done() {
		return ProgramsRule{}, false
	}
	return ProgramsRule{Clauses: clauses}, true
}

type tokenKind int

const (
	tokWord tokenKind = iota
	tokString
	tokPunct
)

type sqlToken struct {
	kind tokenKind
	text string
}

func tokenizeSQL(s string) ([]sqlToken, bool) {
	var toks []sqlToken
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '\'':
			var b strings.Builder
			i++
			for {
				if i >= len(s) {
					return nil, false
				}
				if s[i] == '\'' {
					if i+1 < len(s) && s[i+1] == '\'' {
						b.WriteByte('\'')
						i += 2
						continue
					}
					i++
					break
				}
				b.WriteByte(s[i])
				i++
			}
			toks = append(toks, sqlToken{kind: tokString, text: b.String()})
		case isWordByte(c):
			j := i
			for j < len(s) && isWordByte(s[j]) {
				j++
			}
			toks = append(toks, sqlToken{kind: tokWord, text: s[i:j]})
			i = j
		case strings.IndexByte("(),=<;", c) >= 0:
			toks = append(toks, sqlToken{kind: tokPunct, text: string(c)})
			i++
		default:
			return nil, false
		}
	}
	return toks, true
}

func isWordByte(c byte) bool {
	return c == '_' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')
}

type ruleParser struct {
	toks []sqlToken
	pos  int
}

func (p *ruleParser) done() bool { return p.pos >= len(p.toks) }

func (p *ruleParser) peekWords(words ...string) bool {
	if p.pos+len(words) > len(p.toks) {
		return false
	}
	for i, w := range words {
		t := p.toks[p.pos+i]
		if t.kind == tokString || !strings.EqualFold(t.text, w) {
			return false
		}
	}
	return true
}

// words consumes the given keyword and punctuation sequence, case-insensitively.
func (p *ruleParser) words(words ...string) bool {
	if !p.peekWords(words...) {
		return false
	}
	p.pos += len(words)
	return true
}

func (p *ruleParser) str() (string, bool) {
	if p.done() || p.toks[p.pos].kind != tokString {
		return "", false
	}
	p.pos++
	return p.toks[p.pos-1].text, true
}

// conjunction parses `item AND item ...`. At the top level the last item must be the
// generator's version_compare, which is consumed and not returned.
func (p *ruleParser) conjunction(top bool) ([][]ProgramsCondition, bool) {
	var clauses [][]ProgramsCondition
	for {
		if top && p.words("version_compare", "(", "version", ",") {
			if _, ok := p.str(); !ok {
				return nil, false
			}
			if !p.words(")", "<", "0") {
				return nil, false
			}
			return clauses, true
		}

		item, ok := p.item()
		if !ok {
			return nil, false
		}
		clauses = append(clauses, item...)
		if !p.words("AND") {
			return clauses, !top
		}
	}
}

// item parses one condition or a parenthesized group. A group is either a
// conjunction, which is flattened into the enclosing one (the generator wraps WHERE
// bodies containing OR in parens), or an OR of single name conditions.
func (p *ruleParser) item() ([][]ProgramsCondition, bool) {
	if !p.words("(") {
		c, ok := p.condition()
		if !ok {
			return nil, false
		}
		return [][]ProgramsCondition{{c}}, true
	}

	var branches [][][]ProgramsCondition
	for {
		branch, ok := p.conjunction(false)
		if !ok {
			return nil, false
		}
		branches = append(branches, branch)
		if !p.words("OR") {
			break
		}
	}
	if !p.words(")") {
		return nil, false
	}
	if len(branches) == 1 {
		return branches[0], true
	}

	var anyOf []ProgramsCondition
	for _, branch := range branches {
		if len(branch) != 1 {
			return nil, false
		}
		for _, c := range branch[0] {
			if c.Column != ProgramsName {
				return nil, false
			}
		}
		anyOf = append(anyOf, branch[0]...)
	}
	return [][]ProgramsCondition{anyOf}, true
}

func (p *ruleParser) condition() (ProgramsCondition, bool) {
	var c ProgramsCondition
	if p.words("TRIM", "(") {
		c.Trim = true
	}
	switch {
	case p.words(string(ProgramsName)):
		c.Column = ProgramsName
	case p.words(string(ProgramsPublisher)):
		c.Column = ProgramsPublisher
	case p.words(string(ProgramsVersion)):
		c.Column = ProgramsVersion
	default:
		return c, false
	}
	if c.Trim && !p.words(")") {
		return c, false
	}

	switch {
	case p.words("="):
		c.Op = OpEqual
	case p.words("LIKE"):
		c.Op = OpLike
	case p.words("NOT", "LIKE"):
		c.Op = OpNotLike
	case p.words("IN", "("):
		c.Op = OpEqual
		for {
			s, ok := p.str()
			if !ok {
				return c, false
			}
			c.Values = append(c.Values, s)
			if !p.words(",") {
				break
			}
		}
		return c, p.words(")")
	default:
		return c, false
	}

	s, ok := p.str()
	if !ok {
		return c, false
	}
	c.Values = []string{s}
	return c, true
}
