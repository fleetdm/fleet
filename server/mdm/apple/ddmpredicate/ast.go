package ddmpredicate

import (
	"fmt"
	"strings"
)

// Predicate is a parsed predicate node: a comparison, a compound predicate,
// or one of the constant predicates.
type Predicate interface {
	fmt.Stringer
	isPredicate()
}

// Expr is a parsed expression node usable on either side of a comparison.
type Expr interface {
	fmt.Stringer
	isExpr()
}

// TruePredicate is the constant TRUEPREDICATE.
type TruePredicate struct{}

// FalsePredicate is the constant FALSEPREDICATE.
type FalsePredicate struct{}

// AndPredicate is `left AND right`.
type AndPredicate struct {
	Left, Right Predicate
}

// OrPredicate is `left OR right`.
type OrPredicate struct {
	Left, Right Predicate
}

// NotPredicate is `NOT operand`.
type NotPredicate struct {
	Operand Predicate
}

// ComparisonPredicate is `left op right`, optionally preceded by aggregate
// qualifiers (ANY, ALL, NONE, SOME) and carrying string options such as [cd].
type ComparisonPredicate struct {
	Qualifiers []string // uppercase: ANY, ALL, NONE, SOME
	Left       Expr
	Op         string // canonical: ==, !=, <, >, <=, >=, BETWEEN, CONTAINS, IN, BEGINSWITH, ENDSWITH, LIKE, MATCHES
	Options    string // "", "c", "d", or "cd"
	Right      Expr
}

// StringLiteral is a quoted string with escapes resolved.
type StringLiteral struct {
	Value string
}

// NumberLiteral is a numeric constant, kept as written.
type NumberLiteral struct {
	Text string
}

// BoolLiteral is TRUE or FALSE.
type BoolLiteral struct {
	Value bool
}

// NullLiteral is NULL (or NIL).
type NullLiteral struct{}

// SelfExpr is SELF.
type SelfExpr struct{}

// Variable is `$name`.
type Variable struct {
	Name string
}

// Ident is a bare key path identifier. Escaped marks the `#name` form used
// to treat a reserved word as an identifier.
type Ident struct {
	Name    string
	Escaped bool
}

// KeyPathFunc is one of the DDM key path functions: @status(kp), @key(kp),
// or @property(kp). Func is "status", "key", or "property".
type KeyPathFunc struct {
	Func    string
	KeyPath string
}

// CollectionOp is a bare @-collection operator such as @count. Name omits
// the leading @.
type CollectionOp struct {
	Name string
}

// DotExpr is `base.component`, where the component is an Ident,
// KeyPathFunc, or CollectionOp.
type DotExpr struct {
	Base      Expr
	Component Expr
}

// BinaryExpr is `left op right` for the arithmetic operators + - * / **.
type BinaryExpr struct {
	Op          string
	Left, Right Expr
}

// UnaryMinus is `-operand`.
type UnaryMinus struct {
	Operand Expr
}

// Aggregate is the literal aggregate `{a, b, c}`.
type Aggregate struct {
	Elements []Expr
}

// IndexExpr is `base[index]` or `base[FIRST|LAST|SIZE]`. Exactly one of
// Index and Special is set; Special is "FIRST", "LAST", or "SIZE".
type IndexExpr struct {
	Base    Expr
	Index   Expr
	Special string
}

// FunctionCall is a BNF function expression such as sum(x) or now().
type FunctionCall struct {
	Name string
	Args []Expr
}

// Subquery is SUBQUERY(collection, $variable, predicate).
type Subquery struct {
	Collection Expr
	Variable   string
	Predicate  Predicate
}

// Assignment is `$variable := value`.
type Assignment struct {
	Variable string
	Value    Expr
}

func (*TruePredicate) isPredicate()       {}
func (*FalsePredicate) isPredicate()      {}
func (*AndPredicate) isPredicate()        {}
func (*OrPredicate) isPredicate()         {}
func (*NotPredicate) isPredicate()        {}
func (*ComparisonPredicate) isPredicate() {}

func (*StringLiteral) isExpr() {}
func (*NumberLiteral) isExpr() {}
func (*BoolLiteral) isExpr()   {}
func (*NullLiteral) isExpr()   {}
func (*SelfExpr) isExpr()      {}
func (*Variable) isExpr()      {}
func (*Ident) isExpr()         {}
func (*KeyPathFunc) isExpr()   {}
func (*CollectionOp) isExpr()  {}
func (*DotExpr) isExpr()       {}
func (*BinaryExpr) isExpr()    {}
func (*UnaryMinus) isExpr()    {}
func (*Aggregate) isExpr()     {}
func (*IndexExpr) isExpr()     {}
func (*FunctionCall) isExpr()  {}
func (*Subquery) isExpr()      {}
func (*Assignment) isExpr()    {}

func (*TruePredicate) String() string  { return "TRUEPREDICATE" }
func (*FalsePredicate) String() string { return "FALSEPREDICATE" }

func (p *AndPredicate) String() string {
	return "(" + p.Left.String() + " AND " + p.Right.String() + ")"
}

func (p *OrPredicate) String() string {
	return "(" + p.Left.String() + " OR " + p.Right.String() + ")"
}

func (p *NotPredicate) String() string { return "NOT " + p.Operand.String() }

func (p *ComparisonPredicate) String() string {
	var b strings.Builder
	for _, q := range p.Qualifiers {
		b.WriteString(q)
		b.WriteByte(' ')
	}
	b.WriteString(p.Left.String())
	b.WriteByte(' ')
	b.WriteString(p.Op)
	if p.Options != "" {
		b.WriteString("[" + p.Options + "]")
	}
	b.WriteByte(' ')
	b.WriteString(p.Right.String())
	return b.String()
}

func (e *StringLiteral) String() string {
	quoted := strings.ReplaceAll(e.Value, `\`, `\\`)
	quoted = strings.ReplaceAll(quoted, `'`, `\'`)
	return "'" + quoted + "'"
}

func (e *NumberLiteral) String() string { return e.Text }

func (e *BoolLiteral) String() string {
	if e.Value {
		return "TRUE"
	}
	return "FALSE"
}

func (*NullLiteral) String() string { return "NULL" }
func (*SelfExpr) String() string    { return "SELF" }
func (e *Variable) String() string  { return "$" + e.Name }

func (e *Ident) String() string {
	if e.Escaped {
		return "#" + e.Name
	}
	return e.Name
}

func (e *KeyPathFunc) String() string  { return "@" + e.Func + "(" + e.KeyPath + ")" }
func (e *CollectionOp) String() string { return "@" + e.Name }

func (e *DotExpr) String() string {
	return e.Base.String() + "." + e.Component.String()
}

func (e *BinaryExpr) String() string {
	return "(" + e.Left.String() + " " + e.Op + " " + e.Right.String() + ")"
}

func (e *UnaryMinus) String() string { return "-" + e.Operand.String() }

func (e *Aggregate) String() string {
	parts := make([]string, len(e.Elements))
	for i, el := range e.Elements {
		parts[i] = el.String()
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func (e *IndexExpr) String() string {
	if e.Special != "" {
		return e.Base.String() + "[" + e.Special + "]"
	}
	return e.Base.String() + "[" + e.Index.String() + "]"
}

func (e *FunctionCall) String() string {
	parts := make([]string, len(e.Args))
	for i, a := range e.Args {
		parts[i] = a.String()
	}
	return e.Name + "(" + strings.Join(parts, ", ") + ")"
}

func (e *Subquery) String() string {
	return "SUBQUERY(" + e.Collection.String() + ", $" + e.Variable + ", " + e.Predicate.String() + ")"
}

func (e *Assignment) String() string {
	return "($" + e.Variable + " := " + e.Value.String() + ")"
}
