package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitYaml(t *testing.T) {
	in := `
---
- Document
#---
--- Document2
---
Document3
`

	docs := SplitYaml(in)
	require.Equal(t, 3, len(docs))
	assert.Equal(t, "- Document\n#---", docs[0])
	assert.Equal(t, "Document2", docs[1])
	assert.Equal(t, "Document3", docs[2])
}

func gitRootPath(t *testing.T) string {
	path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(path))
}

func loadSpec(t *testing.T, relativePaths ...string) []byte {
	b, err := os.ReadFile(filepath.Join(
		append([]string{gitRootPath(t)}, relativePaths...)...,
	))
	require.NoError(t, err)
	return b
}

func TestGroupFromBytesWithStdLib(t *testing.T) {
	stdQueryLib := loadSpec(t,
		"docs", "01-Using-Fleet", "standard-query-library", "standard-query-library.yml",
	)
	g, err := GroupFromBytes(stdQueryLib)
	require.NoError(t, err)
	require.NotEmpty(t, g.Queries)
	require.NotEmpty(t, g.Policies)
}

func TestGroupFromBytesWithMacOS13CISQueries(t *testing.T) {
	cisQueries := loadSpec(t,
		"ee", "cis", "macos-13", "cis-policy-queries.yml",
	)
	g, err := GroupFromBytes(cisQueries)
	require.NoError(t, err)
	require.NotEmpty(t, g.Policies)
}

func TestGroupFromBytesWithWin10CISQueries(t *testing.T) {
	cisQueries := loadSpec(t,
		"ee", "cis", "win-10", "cis-policy-queries.yml",
	)
	g, err := GroupFromBytes(cisQueries)
	require.NoError(t, err)
	require.NotEmpty(t, g.Policies)
}

func TestGroupFromBytesMissingFields(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{
			"missing spec",
			[]byte(`
---
apiVersion: v1
kind: team
			`),
			`Missing required fields ("spec") on provided "team" configuration.`,
		},
		{
			"missing spec and kind",
			[]byte(`
---
apiVersion: v1
			`),
			`Missing required fields ("spec", "kind") on provided configuration`,
		},
		{
			"missing spec and empty string kind",
			[]byte(`
---
apiVersion: v1
kind: ""
			`),
			`Missing required fields ("spec", "kind") on provided configuration`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GroupFromBytes(tt.in)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestEscapeString(t *testing.T) {
	for _, tc := range []struct {
		s         string
		expResult string
	}{
		{`$foo`, `$foo`},                                      // nothing to escape
		{`bar$foo`, `bar$foo`},                                // nothing to escape
		{`bar${foo}`, `bar${foo}`},                            // nothing to escape
		{`\$foo`, `$PREVENT_ESCAPING_foo`},                    // escaping
		{`bar\$foo`, `bar$PREVENT_ESCAPING_foo`},              // escaping
		{`\\$foo`, `\\$foo`},                                  // no escaping
		{`bar\\$foo`, `bar\\$foo`},                            // no escaping
		{`\\\$foo`, `\$PREVENT_ESCAPING_foo`},                 // escaping
		{`bar\\\$foo`, `bar\$PREVENT_ESCAPING_foo`},           // escaping
		{`bar\\\${foo}bar`, `bar\$PREVENT_ESCAPING_{foo}bar`}, // escaping
		{`\\\\$foo`, `\\\\$foo`},                              // no escaping
		{`bar\\\\$foo`, `bar\\\\$foo`},                        // no escaping
		{`bar\\\\${foo}`, `bar\\\\${foo}`},                    // no escaping
	} {
		result := escapeString(tc.s, "PREVENT_ESCAPING_")
		require.Equal(t, tc.expResult, result)
	}
}

func checkMultiErrors(t *testing.T, errs ...string) func(err error) {
	return func(err error) {
		me, ok := err.(*multierror.Error)
		require.True(t, ok)
		require.Len(t, me.Errors, len(errs))
		for i, err := range me.Errors {
			require.Equal(t, errs[i], err.Error())
		}
	}
}

func TestExpandEnv(t *testing.T) {
	for _, tc := range []struct {
		environment map[string]string
		s           string
		expResult   string
		checkErr    func(error)
	}{
		{map[string]string{"foo": "1"}, `$foo`, `1`, nil},
		{map[string]string{"foo": "1"}, `$foo $FLEET_VAR_BAR ${FLEET_VAR_BAR}x ${foo}`, `1 $FLEET_VAR_BAR ${FLEET_VAR_BAR}x 1`, nil},
		{map[string]string{"foo": ""}, `$foo`, ``, nil},
		{map[string]string{"foo": "", "bar": "", "zoo": ""}, `$foo${bar}$zoo`, ``, nil},
		{map[string]string{}, `$foo`, ``, checkMultiErrors(t, "environment variable \"foo\" not set")},
		{map[string]string{"foo": "1"}, `$foo$bar`, ``, checkMultiErrors(t, "environment variable \"bar\" not set")},
		{map[string]string{"bar": "1"}, `$foo $bar $zoo`, ``,
			checkMultiErrors(t, "environment variable \"foo\" not set", "environment variable \"zoo\" not set")},
		{map[string]string{"foo": "4", "bar": "2"}, `$foo$bar`, `42`, nil},
		{map[string]string{"foo": "42", "bar": ""}, `$foo$bar`, `42`, nil},
		{map[string]string{}, `$$`, ``, checkMultiErrors(t, "environment variable \"$\" not set")},
		{map[string]string{"foo": "1"}, `$$foo`, ``, checkMultiErrors(t, "environment variable \"$\" not set")},
		{map[string]string{"foo": "1"}, `\$${foo}`, `$1`, nil},
		{map[string]string{}, `\$foo`, `$foo`, nil},                     // escaped
		{map[string]string{"foo": "1"}, `\\$foo`, `\\1`, nil},           // not escaped
		{map[string]string{}, `\\\$foo`, `\$foo`, nil},                  // escaped
		{map[string]string{}, `\\\$foo$`, `\$foo$`, nil},                // escaped
		{map[string]string{}, `bar\\\$foo$`, `bar\$foo$`, nil},          // escaped
		{map[string]string{"foo": "1"}, `$foo var`, `1 var`, nil},       // not escaped
		{map[string]string{"foo": "1"}, `${foo}var`, `1var`, nil},       // not escaped
		{map[string]string{"foo": "1"}, `\${foo}var`, `${foo}var`, nil}, // escaped
		{map[string]string{"foo": ""}, `${foo}var`, `var`, nil},
		{map[string]string{"foo": "", "$": "2"}, `${$}${foo}var`, `2var`, nil},
		{map[string]string{}, `${foo}var`, ``, checkMultiErrors(t, "environment variable \"foo\" not set")},
		{map[string]string{}, `foo PREVENT_ESCAPING_bar $ FLEET_VAR_`, `foo PREVENT_ESCAPING_bar $ FLEET_VAR_`, nil}, // nothing to replace
		{map[string]string{"foo": "BAR"}, `\$FLEET_VAR_$foo \${FLEET_VAR_$foo} \${FLEET_VAR_${foo}2}`,
			`$FLEET_VAR_BAR ${FLEET_VAR_BAR} ${FLEET_VAR_BAR2}`, nil}, // nested variables
	} {
		os.Clearenv()
		for k, v := range tc.environment {
			_ = os.Setenv(k, v)
		}
		result, err := ExpandEnv(tc.s)
		if tc.checkErr == nil {
			require.NoError(t, err)
		} else {
			tc.checkErr(err)
		}
		require.Equal(t, tc.expResult, result)
	}
}

func TestLookupEnvSecrets(t *testing.T) {
	for _, tc := range []struct {
		environment map[string]string
		s           string
		expResult   map[string]string
		checkErr    func(error)
	}{
		{map[string]string{"foo": "1"}, `$foo`, map[string]string{}, nil},
		{map[string]string{"FLEET_SECRET_foo": "1"}, `$FLEET_SECRET_foo`, map[string]string{"FLEET_SECRET_foo": "1"}, nil},
		{map[string]string{"foo": "1"}, `$FLEET_SECRET_foo`, map[string]string{},
			checkMultiErrors(t, "environment variable \"FLEET_SECRET_foo\" not set")},
	} {
		os.Clearenv()
		for k, v := range tc.environment {
			_ = os.Setenv(k, v)
		}
		secretsMap := make(map[string]string)
		err := LookupEnvSecrets(tc.s, secretsMap)
		if tc.checkErr == nil {
			require.NoError(t, err)
		} else {
			tc.checkErr(err)
		}
		require.Equal(t, tc.expResult, secretsMap)
	}
}
