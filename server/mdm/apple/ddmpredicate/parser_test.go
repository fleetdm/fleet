package ddmpredicate

import (
	"strings"
	"testing"
)

// appleDocumentedPredicates are predicates taken verbatim from Apple's DDM
// documentation and sessions.
var appleDocumentedPredicates = []struct {
	name   string
	source string
	input  string
}{
	{
		name:   "activation simple apple tv",
		source: "https://developer.apple.com/documentation/devicemanagement/activationsimple",
		input:  `@status(device.model.family) == 'AppleTV'`,
	},
	{
		name:   "status item comparison",
		source: "WWDC22 session 10046, Adopt declarative device management",
		input:  `@status(device.model.family) == 'iPhone'`,
	},
	{
		name:   "subquery over mdm app status items",
		source: "WWDC22 session 10046, Adopt declarative device management",
		input: `SUBQUERY(@status(mdm.app), $app, ($app.@key(identifier) == "com.example.app") AND ` +
			`($app.@key(state) == "managed")).@count == 1`,
	},
	{
		name:   "management properties",
		source: "WWDC22 session 10046, Adopt declarative device management",
		input:  `(@property(age) >= 18) AND ("Grade12" IN @property(roles))`,
	},
	{
		name:   "subquery over active configurations",
		source: "https://developer.apple.com/forums/ (activation predicate discussion)",
		input: `SUBQUERY(@status(management.declarations.configurations), $declaration, ` +
			`($declaration.@key(identifier) == "com.abc.declarationname" AND ` +
			`$declaration.@key(active) == true)).@count == 1`,
	},
}

func TestAppleDocumentedPredicates(t *testing.T) {
	for _, tc := range appleDocumentedPredicates {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.input); err != nil {
				t.Errorf("Validate(%q) = %v, want nil\nsource: %s", tc.input, err, tc.source)
			}
		})
	}
}

func TestValidPredicates(t *testing.T) {
	inputs := []string{
		// Constant predicates and compounds.
		`TRUEPREDICATE`,
		`FALSEPREDICATE`,
		`NOT FALSEPREDICATE`,
		`TRUEPREDICATE AND FALSEPREDICATE OR TRUEPREDICATE`,
		`truepredicate and not falsepredicate`, // keywords are case-insensitive
		`NOT (@status(device.model.family) == 'Mac')`,
		`@status(a.b) == 1 && @property(c) == 2`,
		`@status(a.b) == 1 || !(@property(c) == 2)`,

		// DDM key path functions, including dashed item names.
		`@status(device.operating-system.version) >= '17.0'`,
		`@status(passcode.is-compliant) == TRUE`,
		`@status(device.identifier.serial-number) != NULL`,
		`@property(nickname) != NULL`,
		`@status( device.model.family ) == 'iPad'`, // whitespace inside the parens
		`@key(identifier) == 'com.example.app'`,

		// Comparison operators and their BNF spellings.
		`@property(age) = 18`,
		`@property(age) <> 18`,
		`@property(age) => 18`,
		`@property(age) =< 18`,
		`@status(x.y) BETWEEN {1, 10}`,
		`@status(device.model.family) IN {'iPhone', 'iPad'}`,
		`@property(x) IN {}`,

		// String operators and options.
		`@status(device.operating-system.build-version) BEGINSWITH[c] '21'`,
		`@status(device.model.marketing-name) MATCHES '.*Pro.*'`,
		`@property(name) CONTAINS[cd] 'smith'`,
		`@property(name) ENDSWITH 'son'`,
		`@property(name) LIKE 'J*'`,
		`@property(name) ==[c] 'jordan'`,

		// Aggregate qualifiers.
		`ANY @property(roles) == 'admin'`,
		`ALL @property(scores) >= 50`,
		`NONE @property(roles) == 'banned'`,
		`SOME @property(roles) == 'teacher'`,

		// Collection operators, subqueries, indexes, functions, arithmetic.
		`@status(test.array).@count > 0`,
		`SUBQUERY(@status(mdm.app), $app, $app.@key(state) == 'managed').@count >= 1`,
		`@property(list)[FIRST] == 'a'`,
		`@property(list)[LAST] == 'z'`,
		`@property(list)[SIZE] == 3`,
		`@property(list)[0] == 'a'`,
		`sum(@property(scores)) > 100`,
		`now() > $lastSeen`,
		`1 + 2 * 3 == 7`,
		`(1 + 2) * 3 == 9`,
		`2 ** 3 == 8`,
		`-5 < @property(x)`,
		`@property(total) / 2 >= 10`,

		// Literals, variables, escapes, and identifier escaping.
		`@property(name) == 'O\'Brien'`,
		`@property(name) == "double \"quoted\""`,
		`@property(code) == 0x1F`,
		`@property(ratio) == 1.5e-3`,
		`$context == 'kiosk'`,
		`SELF == 'value'`,
		`#size == 5`,       // reserved word escaped as an identifier
		`device.#size > 2`, // escaped identifier as a key path component
		`($x := 5) == 5`,
		`item.name == 'x'`, // plain key path per the BNF
	}
	for _, input := range inputs {
		if err := Validate(input); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", input, err)
		}
	}
}

func TestInvalidPredicates(t *testing.T) {
	cases := []struct {
		input   string
		wantErr string
	}{
		// Unknown or bare @-key paths, the core DDM validation.
		{`@invalid == 'Apple TV'`, "unknown key path expression '@invalid'"},
		{`@status == 'AppleTV'`, "a bare @status is not allowed"},
		{`@property > 5`, "a bare @property is not allowed"},
		{`@key != 'x'`, "a bare @key is not allowed"},
		{`$app.@key == 'x'`, "a bare @key is not allowed"},
		{`@Status(device.model.family) == 'x'`, "did you mean @status(...)"},
		{`@count(x) == 1`, "@count is a collection operator and does not take arguments"},
		{`@ status(a) == 1`, "expected a key path function after '@'"},

		// Malformed key path arguments.
		{`@status() == 'x'`, "requires a key path argument"},
		{`@status('device.model.family') == 'x'`, "requires a key path argument"},
		{`@status(device..family) == 'x'`, "empty key path segment"},
		{`@status(.family) == 'x'`, "empty key path segment"},
		{`@status(9foo) == 1`, `segment "9foo" must start with a letter or underscore`},
		{`@status(foo-) == 1`, `segment "foo-" must not end with a dash`},
		{`@status(device.model.family == 'x'`, "expected ')' after the @status key path"},
		{`@status(a b) == 1`, "expected ')' after the @status key path"},

		// General syntax errors.
		{``, "empty predicate"},
		{`   `, "empty predicate"},
		{`== 5`, "expected an expression"},
		{`@status(a) ==`, "expected an expression, got end of predicate"},
		{`@status(a) === 1`, "expected an expression, got '='"},
		{`@status(a) == 'x' OR`, "expected an expression, got end of predicate"},
		{`@status(a)`, "expected a comparison operator"},
		{`@property(roles) @count == 1`, "expected a comparison operator"},
		{`@status(a) == 'x' extra`, "unexpected 'extra' after end of predicate"},
		{`TRUEPREDICATE TRUEPREDICATE`, "unexpected 'TRUEPREDICATE' after end of predicate"},
		{`(@status(a) == 1`, "expected ')'"},
		{`@status(a) == 'unterminated`, "unterminated string"},

		// Operators, options, functions, reserved words.
		{`@status(a) CONTAINS[x] 'b'`, "invalid string options"},
		{`@status(a) BETWEEN[c] {1, 2}`, "BETWEEN does not accept string options"},
		{`foo(1) == 2`, `unknown function "foo"`},
		{`SUM(1) == 2`, `did you mean "sum"`},
		{`@status(a) == %@`, "format argument"},
		{`name == %K`, "format argument"},
		{`AND == 1`, `reserved word "AND" cannot be used as a key path`},
		{`device.size > 2`, `reserved word "size" cannot be used as a key path component`},
		{`SUBQUERY(@status(a.b), app, TRUEPREDICATE) == 1`, "expected a $variable as the second argument of SUBQUERY"},
		{`5 := 6`, "left side of ':=' must be a $variable"},
	}
	for _, tc := range cases {
		err := Validate(tc.input)
		if err == nil {
			t.Errorf("Validate(%q) = nil, want error containing %q", tc.input, tc.wantErr)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("Validate(%q) = %q, want error containing %q", tc.input, err, tc.wantErr)
		}
	}
}

func TestParseAST(t *testing.T) {
	pred, err := Parse(`@status(device.model.family) == 'AppleTV'`)
	if err != nil {
		t.Fatal(err)
	}
	cmp, ok := pred.(*ComparisonPredicate)
	if !ok {
		t.Fatalf("got %T, want *ComparisonPredicate", pred)
	}
	left, ok := cmp.Left.(*KeyPathFunc)
	if !ok {
		t.Fatalf("left side is %T, want *KeyPathFunc", cmp.Left)
	}
	if left.Func != "status" || left.KeyPath != "device.model.family" {
		t.Errorf("left side = @%s(%s), want @status(device.model.family)", left.Func, left.KeyPath)
	}
	if cmp.Op != "==" || cmp.Options != "" {
		t.Errorf("op = %q options %q, want == with no options", cmp.Op, cmp.Options)
	}
	right, ok := cmp.Right.(*StringLiteral)
	if !ok || right.Value != "AppleTV" {
		t.Errorf("right side = %#v, want string literal AppleTV", cmp.Right)
	}
}

func TestParseSubqueryAST(t *testing.T) {
	pred, err := Parse(appleDocumentedPredicates[2].input)
	if err != nil {
		t.Fatal(err)
	}
	cmp, ok := pred.(*ComparisonPredicate)
	if !ok {
		t.Fatalf("got %T, want *ComparisonPredicate", pred)
	}
	dot, ok := cmp.Left.(*DotExpr)
	if !ok {
		t.Fatalf("left side is %T, want *DotExpr", cmp.Left)
	}
	sub, ok := dot.Base.(*Subquery)
	if !ok {
		t.Fatalf("dot base is %T, want *Subquery", dot.Base)
	}
	if sub.Variable != "app" {
		t.Errorf("subquery variable = %q, want app", sub.Variable)
	}
	coll, ok := sub.Collection.(*KeyPathFunc)
	if !ok || coll.Func != "status" || coll.KeyPath != "mdm.app" {
		t.Errorf("subquery collection = %#v, want @status(mdm.app)", sub.Collection)
	}
	if _, ok := sub.Predicate.(*AndPredicate); !ok {
		t.Errorf("subquery predicate is %T, want *AndPredicate", sub.Predicate)
	}
	op, ok := dot.Component.(*CollectionOp)
	if !ok || op.Name != "count" {
		t.Errorf("dot component = %#v, want @count", dot.Component)
	}
}

// TestStringRoundTrip checks that String output reparses to the same
// canonical form.
func TestStringRoundTrip(t *testing.T) {
	var inputs []string
	for _, tc := range appleDocumentedPredicates {
		inputs = append(inputs, tc.input)
	}
	inputs = append(inputs,
		`@status(x.y) BETWEEN {1, 10}`,
		`@property(name) CONTAINS[cd] 'it\'s'`,
		`ANY @property(roles) == 'admin'`,
		`@property(list)[FIRST] == 'a'`,
		`NOT (1 + 2 * 3 == 7)`,
	)
	for _, input := range inputs {
		first, err := Parse(input)
		if err != nil {
			t.Errorf("Parse(%q) = %v", input, err)
			continue
		}
		second, err := Parse(first.String())
		if err != nil {
			t.Errorf("Parse(%q).String() = %q, which does not reparse: %v", input, first.String(), err)
			continue
		}
		if first.String() != second.String() {
			t.Errorf("round trip of %q changed: %q != %q", input, first.String(), second.String())
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, tc := range appleDocumentedPredicates {
		f.Add(tc.input)
	}
	f.Add(`@status(a) == 'x' AND (b.c[FIRST] BETWEEN {1, 2} OR sum(1, 2) != 3)`)
	f.Add(`@invalid == 'Apple TV'`)
	f.Fuzz(func(t *testing.T, input string) {
		pred, err := Parse(input)
		if err == nil {
			// Whatever parses must reparse from its own String form.
			if _, err := Parse(pred.String()); err != nil {
				t.Errorf("Parse(%q) succeeded but String %q does not reparse: %v", input, pred.String(), err)
			}
		}
	})
}
