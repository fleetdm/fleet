package ddmpredicate

import (
	"errors"
	"fmt"
	"strings"
)

// Parse parses input as a DDM activation predicate and returns its AST.
// The returned error is always a *ParseError.
func Parse(input string) (Predicate, error) {
	p := &parser{lex: &lexer{input: input}}
	if err := p.advance(); err != nil {
		return nil, err
	}
	if p.tok.kind == tokEOF {
		return nil, p.errorf(0, "empty predicate")
	}
	pred, err := p.parsePredicate()
	if err != nil {
		return nil, err
	}
	if p.tok.kind != tokEOF {
		return nil, p.errorf(p.tok.pos, "unexpected %s after end of predicate", p.tok.describe())
	}
	return pred, nil
}

// Validate reports whether input is a valid DDM activation predicate.
func Validate(input string) error {
	_, err := Parse(input)
	return err
}

var reservedWords = map[string]struct{}{
	"AND": {}, "OR": {}, "NOT": {},
	"IN": {}, "BETWEEN": {}, "CONTAINS": {}, "LIKE": {}, "MATCHES": {},
	"BEGINSWITH": {}, "ENDSWITH": {},
	"ANY": {}, "ALL": {}, "NONE": {}, "SOME": {},
	"TRUE": {}, "FALSE": {}, "NULL": {}, "NIL": {}, "SELF": {},
	"TRUEPREDICATE": {}, "FALSEPREDICATE": {}, "SUBQUERY": {},
	"FIRST": {}, "LAST": {}, "SIZE": {},
}

// bnfFunctions is the function_name production of Apple's BNF.
var bnfFunctions = map[string]struct{}{
	"sum": {}, "count": {}, "min": {}, "max": {},
	"average": {}, "median": {}, "mode": {}, "stddev": {},
	"sqrt": {}, "log": {}, "ln": {}, "exp": {},
	"floor": {}, "ceiling": {}, "abs": {}, "trunc": {},
	"random": {}, "randomn": {}, "now": {},
}

// keyPathFuncs are the DDM key path functions, which require a
// parenthesized key path argument.
var keyPathFuncs = map[string]string{
	"status":   "@status(device.model.family)",
	"key":      "@key(identifier)",
	"property": "@property(age)",
}

// collectionOps are the standard NSPredicate/KVC collection operators,
// which appear bare (no parentheses), usually after a dot as in
// SUBQUERY(...).@count.
var collectionOps = map[string]struct{}{
	"count": {}, "avg": {}, "sum": {}, "min": {}, "max": {},
	"firstObject": {}, "lastObject": {},
	"unionOfObjects": {}, "distinctUnionOfObjects": {},
	"unionOfArrays": {}, "distinctUnionOfArrays": {},
	"distinctUnionOfSets": {},
}

type parser struct {
	lex *lexer
	tok token
}

type parserState struct {
	lexPos int
	tok    token
}

func (p *parser) save() parserState     { return parserState{lexPos: p.lex.pos, tok: p.tok} }
func (p *parser) restore(s parserState) { p.lex.pos = s.lexPos; p.tok = s.tok }
func (p *parser) errorf(pos int, format string, args ...any) error {
	return &ParseError{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

func (p *parser) advance() error {
	t, err := p.lex.next()
	if err != nil {
		return err
	}
	p.tok = t
	return nil
}

func (p *parser) expectPunct(s string) error {
	if !p.tok.isPunct(s) {
		return p.errorf(p.tok.pos, "expected '%s', got %s", s, p.tok.describe())
	}
	return p.advance()
}

// parsePredicate parses at OR precedence: NOT binds tightest, then AND,
// then OR.
func (p *parser) parsePredicate() (Predicate, error) {
	left, err := p.parseAndPredicate()
	if err != nil {
		return nil, err
	}
	for p.tok.keyword() == "OR" || p.tok.isPunct("||") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseAndPredicate()
		if err != nil {
			return nil, err
		}
		left = &OrPredicate{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAndPredicate() (Predicate, error) {
	left, err := p.parseUnaryPredicate()
	if err != nil {
		return nil, err
	}
	for p.tok.keyword() == "AND" || p.tok.isPunct("&&") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseUnaryPredicate()
		if err != nil {
			return nil, err
		}
		left = &AndPredicate{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnaryPredicate() (Predicate, error) {
	switch {
	case p.tok.keyword() == "NOT" || p.tok.isPunct("!"):
		if err := p.advance(); err != nil {
			return nil, err
		}
		operand, err := p.parseUnaryPredicate()
		if err != nil {
			return nil, err
		}
		return &NotPredicate{Operand: operand}, nil
	case p.tok.keyword() == "TRUEPREDICATE":
		return &TruePredicate{}, p.advance()
	case p.tok.keyword() == "FALSEPREDICATE":
		return &FalsePredicate{}, p.advance()
	case p.tok.isPunct("("):
		// A "(" opens either a parenthesized predicate or a parenthesized
		// expression on the left of a comparison. Try the predicate reading
		// first and fall back to a comparison, keeping whichever error got
		// further into the input.
		save := p.save()
		pred, perr := p.parseParenPredicate()
		if perr == nil {
			return pred, nil
		}
		p.restore(save)
		cmp, cerr := p.parseComparison()
		if cerr == nil {
			return cmp, nil
		}
		return nil, furthestError(cerr, perr)
	default:
		return p.parseComparison()
	}
}

func (p *parser) parseParenPredicate() (Predicate, error) {
	if err := p.advance(); err != nil { // consume "("
		return nil, err
	}
	pred, err := p.parsePredicate()
	if err != nil {
		return nil, err
	}
	if err := p.expectPunct(")"); err != nil {
		return nil, err
	}
	if p.tok.kind == tokPunct {
		switch p.tok.text {
		case ".", "[", "+", "-", "*", "/", "**":
			// The parentheses enclosed an expression, not a predicate.
			return nil, p.errorf(p.tok.pos, "unexpected %s after parenthesized predicate", p.tok.describe())
		}
	}
	return pred, nil
}

func furthestError(a, b error) error {
	var pa, pb *ParseError
	if errors.As(a, &pa) && errors.As(b, &pb) && pb.Pos > pa.Pos {
		return b
	}
	return a
}

func (p *parser) parseComparison() (Predicate, error) {
	var quals []string
	for {
		kw := p.tok.keyword()
		if kw != "ANY" && kw != "ALL" && kw != "NONE" && kw != "SOME" {
			break
		}
		quals = append(quals, kw)
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	left, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	op, opts, err := p.parseComparisonOp()
	if err != nil {
		return nil, err
	}
	right, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &ComparisonPredicate{Qualifiers: quals, Left: left, Op: op, Options: opts, Right: right}, nil
}

func (p *parser) parseComparisonOp() (op, options string, err error) {
	switch p.tok.kind {
	case tokPunct:
		switch p.tok.text {
		case "=", "==":
			op = "=="
		case "!=", "<>":
			op = "!="
		case ">=", "=>":
			op = ">="
		case "<=", "=<":
			op = "<="
		case "<", ">":
			op = p.tok.text
		}
	default:
		switch p.tok.keyword() {
		case "BETWEEN", "CONTAINS", "IN", "BEGINSWITH", "ENDSWITH", "LIKE", "MATCHES":
			op = p.tok.keyword()
		}
	}
	if op == "" {
		return "", "", p.errorf(p.tok.pos,
			"expected a comparison operator (==, !=, <, >, <=, >=, BETWEEN, CONTAINS, IN, BEGINSWITH, ENDSWITH, LIKE, MATCHES), got %s", p.tok.describe())
	}
	if err := p.advance(); err != nil {
		return "", "", err
	}
	if !p.tok.isPunct("[") {
		return op, "", nil
	}
	if op == "BETWEEN" {
		return "", "", p.errorf(p.tok.pos, "BETWEEN does not accept string options")
	}
	if err := p.advance(); err != nil {
		return "", "", err
	}
	if p.tok.kind != tokIdent {
		return "", "", p.errorf(p.tok.pos, "invalid string options: use [c], [d], or [cd]")
	}
	options = strings.ToLower(p.tok.text)
	if options != "c" && options != "d" && options != "cd" {
		return "", "", p.errorf(p.tok.pos, "invalid string options %q: use [c], [d], or [cd]", p.tok.text)
	}
	if err := p.advance(); err != nil {
		return "", "", err
	}
	if err := p.expectPunct("]"); err != nil {
		return "", "", err
	}
	return op, options, nil
}

// parseExpr parses a full expression, including the rarely used
// assignment form `$variable := expression`.
func (p *parser) parseExpr() (Expr, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	if !p.tok.isPunct(":=") {
		return left, nil
	}
	v, ok := left.(*Variable)
	if !ok {
		return nil, p.errorf(p.tok.pos, "left side of ':=' must be a $variable")
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	value, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &Assignment{Variable: v.Name, Value: value}, nil
}

func (p *parser) parseAdditive() (Expr, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.tok.isPunct("+") || p.tok.isPunct("-") {
		op := p.tok.text
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseMultiplicative() (Expr, error) {
	left, err := p.parsePower()
	if err != nil {
		return nil, err
	}
	for p.tok.isPunct("*") || p.tok.isPunct("/") {
		op := p.tok.text
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parsePower()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parsePower() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.tok.isPunct("**") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: "**", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (Expr, error) {
	if p.tok.isPunct("-") {
		if err := p.advance(); err != nil {
			return nil, err
		}
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryMinus{Operand: operand}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch {
		case p.tok.isPunct("."):
			if err := p.advance(); err != nil {
				return nil, err
			}
			comp, err := p.parseKeyPathComponent()
			if err != nil {
				return nil, err
			}
			expr = &DotExpr{Base: expr, Component: comp}
		case p.tok.isPunct("["):
			if err := p.advance(); err != nil {
				return nil, err
			}
			idx, err := p.parseIndex()
			if err != nil {
				return nil, err
			}
			idx.Base = expr
			expr = idx
		default:
			return expr, nil
		}
	}
}

func (p *parser) parseKeyPathComponent() (Expr, error) {
	switch p.tok.kind {
	case tokAt:
		return p.parseAtExpr()
	case tokIdent:
		if _, ok := reservedWords[p.tok.keyword()]; ok {
			return nil, p.errorf(p.tok.pos, "reserved word %q cannot be used as a key path component; escape it as '#%s'", p.tok.text, p.tok.text)
		}
		ident := &Ident{Name: p.tok.text, Escaped: p.tok.escaped}
		return ident, p.advance()
	default:
		return nil, p.errorf(p.tok.pos, "expected a key path component after '.', got %s", p.tok.describe())
	}
}

func (p *parser) parseIndex() (*IndexExpr, error) {
	switch p.tok.keyword() {
	case "FIRST", "LAST", "SIZE":
		idx := &IndexExpr{Special: p.tok.keyword()}
		if err := p.advance(); err != nil {
			return nil, err
		}
		if err := p.expectPunct("]"); err != nil {
			return nil, err
		}
		return idx, nil
	}
	inner, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expectPunct("]"); err != nil {
		return nil, err
	}
	return &IndexExpr{Index: inner}, nil
}

func (p *parser) parsePrimary() (Expr, error) {
	switch p.tok.kind {
	case tokNumber:
		lit := &NumberLiteral{Text: p.tok.text}
		return lit, p.advance()
	case tokString:
		lit := &StringLiteral{Value: p.tok.text}
		return lit, p.advance()
	case tokVariable:
		v := &Variable{Name: p.tok.text}
		return v, p.advance()
	case tokAt:
		return p.parseAtExpr()
	case tokFormat:
		return nil, p.errorf(p.tok.pos,
			"format argument %q is not allowed: DDM predicates are evaluated on device without substitution arguments", p.tok.text)
	case tokIdent:
		return p.parseIdentExpr()
	case tokPunct:
		switch p.tok.text {
		case "(":
			if err := p.advance(); err != nil {
				return nil, err
			}
			inner, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			return inner, p.expectPunct(")")
		case "{":
			return p.parseAggregate()
		}
	}
	return nil, p.errorf(p.tok.pos, "expected an expression, got %s", p.tok.describe())
}

func (p *parser) parseIdentExpr() (Expr, error) {
	switch p.tok.keyword() {
	case "TRUE":
		return &BoolLiteral{Value: true}, p.advance()
	case "FALSE":
		return &BoolLiteral{Value: false}, p.advance()
	case "NULL", "NIL":
		return &NullLiteral{}, p.advance()
	case "SELF":
		return &SelfExpr{}, p.advance()
	case "SUBQUERY":
		return p.parseSubquery()
	}
	if _, ok := reservedWords[p.tok.keyword()]; ok {
		return nil, p.errorf(p.tok.pos, "reserved word %q cannot be used as a key path; escape it as '#%s'", p.tok.text, p.tok.text)
	}
	name, pos, escaped := p.tok.text, p.tok.pos, p.tok.escaped
	if err := p.advance(); err != nil {
		return nil, err
	}
	if !p.tok.isPunct("(") || escaped {
		return &Ident{Name: name, Escaped: escaped}, nil
	}
	if _, ok := bnfFunctions[name]; !ok {
		lower := strings.ToLower(name)
		if _, ok := bnfFunctions[lower]; ok {
			return nil, p.errorf(pos, "unknown function %q; function names are lowercase: did you mean %q?", name, lower)
		}
		return nil, p.errorf(pos, "unknown function %q", name)
	}
	if err := p.advance(); err != nil { // consume "("
		return nil, err
	}
	call := &FunctionCall{Name: name}
	if p.tok.isPunct(")") {
		return call, p.advance()
	}
	for {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		call.Args = append(call.Args, arg)
		if !p.tok.isPunct(",") {
			break
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	return call, p.expectPunct(")")
}

func (p *parser) parseSubquery() (Expr, error) {
	if err := p.advance(); err != nil { // consume SUBQUERY
		return nil, err
	}
	if err := p.expectPunct("("); err != nil {
		return nil, err
	}
	collection, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expectPunct(","); err != nil {
		return nil, err
	}
	if p.tok.kind != tokVariable {
		return nil, p.errorf(p.tok.pos, "expected a $variable as the second argument of SUBQUERY, got %s", p.tok.describe())
	}
	variable := p.tok.text
	if err := p.advance(); err != nil {
		return nil, err
	}
	if err := p.expectPunct(","); err != nil {
		return nil, err
	}
	pred, err := p.parsePredicate()
	if err != nil {
		return nil, err
	}
	if err := p.expectPunct(")"); err != nil {
		return nil, err
	}
	return &Subquery{Collection: collection, Variable: variable, Predicate: pred}, nil
}

func (p *parser) parseAggregate() (Expr, error) {
	if err := p.advance(); err != nil { // consume "{"
		return nil, err
	}
	agg := &Aggregate{}
	if p.tok.isPunct("}") {
		return agg, p.advance()
	}
	for {
		el, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		agg.Elements = append(agg.Elements, el)
		if !p.tok.isPunct(",") {
			break
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	return agg, p.expectPunct("}")
}

// parseAtExpr parses an @-prefixed key path expression. Only the DDM key
// path functions @status(...), @key(...), and @property(...) and the
// standard collection operators (@count, ...) are accepted; anything else,
// including a bare @status with no parentheses, is an error.
func (p *parser) parseAtExpr() (Expr, error) {
	name, pos := p.tok.text, p.tok.pos
	if example, ok := keyPathFuncs[name]; ok {
		if err := p.advance(); err != nil {
			return nil, err
		}
		if !p.tok.isPunct("(") {
			return nil, p.errorf(pos, "@%s must be written as @%s(<key path>), e.g. %s; a bare @%s is not allowed", name, name, example, name)
		}
		// The parser keeps one token of lookahead, so the lexer now sits
		// just past the "(" — scan the raw key path from there (it may
		// contain dashes the tokenizer would split).
		arg, argPos := p.lex.scanKeyPathArg()
		if arg == "" {
			return nil, p.errorf(argPos, "@%s(...) requires a key path argument, e.g. %s", name, example)
		}
		if problem := keyPathProblem(arg); problem != "" {
			return nil, p.errorf(argPos, "invalid key path %q in @%s(...): %s", arg, name, problem)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		if !p.tok.isPunct(")") {
			return nil, p.errorf(p.tok.pos, "expected ')' after the @%s key path, got %s", name, p.tok.describe())
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		return &KeyPathFunc{Func: name, KeyPath: arg}, nil
	}
	if _, ok := collectionOps[name]; ok {
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.tok.isPunct("(") {
			return nil, p.errorf(pos, "@%s is a collection operator and does not take arguments", name)
		}
		return &CollectionOp{Name: name}, nil
	}
	if lower := strings.ToLower(name); keyPathFuncs[lower] != "" {
		return nil, p.errorf(pos, "unknown key path expression '@%s'; did you mean @%s(...)?", name, lower)
	}
	return nil, p.errorf(pos, "unknown key path expression '@%s': only @status(...), @key(...), @property(...) and collection operators such as @count are allowed", name)
}

// keyPathProblem reports what is wrong with a key path argument, or ""
// if it is well-formed: dot-separated segments that start with a letter or
// underscore and continue with letters, digits, underscores, or dashes.
func keyPathProblem(keyPath string) string {
	for seg := range strings.SplitSeq(keyPath, ".") {
		switch {
		case seg == "":
			return "empty key path segment"
		case !isIdentStart(seg[0]):
			return fmt.Sprintf("segment %q must start with a letter or underscore", seg)
		case seg[len(seg)-1] == '-':
			return fmt.Sprintf("segment %q must not end with a dash", seg)
		}
	}
	return ""
}
