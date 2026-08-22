package fleet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainsPrefixVars(t *testing.T) {
	script := `
#!/bin/sh

echo $FLEET_SECRET_FOO is the secret
echo words${FLEET_SECRET_BAR}words
$FLEET_SECRET_BAZ
${FLEET_SECRET_QUX}
`
	secrets := ContainsPrefixVars(script, ServerSecretPrefix)
	require.Contains(t, secrets, "FOO")
	require.Contains(t, secrets, "BAR")
	require.Contains(t, secrets, "BAZ")
	require.Contains(t, secrets, "QUX")
}

// A token that is exactly the prefix names nothing: getShellName stops at the first
// non-word byte, so "$FLEET_SECRET_ " leaves an empty name after the prefix. Callers
// treat this result as the set of variables a document embeds and then look each one
// up, so an empty entry becomes a nameless missing secret.
func TestContainsPrefixVarsBarePrefix(t *testing.T) {
	for _, tc := range []struct {
		name    string
		text    string
		prefix  string
		expVars []string
	}{
		{
			name:    "bare secret prefix in an XML comment is not a reference",
			text:    "<!-- Delivered via $FLEET_SECRET_ (registered via gitops). -->\n<Data>$FLEET_SECRET_REAL</Data>",
			prefix:  ServerSecretPrefix,
			expVars: []string{"REAL"},
		},
		{
			name:    "braced bare secret prefix is not a reference",
			text:    "# ${FLEET_SECRET_} is the prefix\necho ${FLEET_SECRET_REAL}",
			prefix:  ServerSecretPrefix,
			expVars: []string{"REAL"},
		},
		{
			name:    "bare secret prefix on its own yields no variables",
			text:    "$FLEET_SECRET_ and $FLEET_SECRET_ again",
			prefix:  ServerSecretPrefix,
			expVars: []string{},
		},
		{
			name:    "bare server var prefix is not a reference",
			text:    "# $FLEET_VAR_ is the prefix\n$FLEET_VAR_HOST_END_USER_IDP_USERNAME",
			prefix:  ServerVarPrefix,
			expVars: []string{"HOST_END_USER_IDP_USERNAME"},
		},
		{
			name:    "bare host secret prefix is not a reference",
			text:    "# $FLEET_HOST_SECRET_ is the prefix\n$FLEET_HOST_SECRET_REAL",
			prefix:  HostSecretPrefix,
			expVars: []string{"REAL"},
		},
		{
			// This row passes on unmodified main too — it pins the namespace
			// separation the fix must not disturb, not the fix itself. Every other
			// row above fails without the guard.
			name:    "a named secret is not reported under the host prefix",
			text:    "$FLEET_SECRET_REAL",
			prefix:  HostSecretPrefix,
			expVars: []string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vars := ContainsPrefixVars(tc.text, tc.prefix)
			require.Equal(t, tc.expVars, vars)
			require.NotContains(t, vars, "", "an empty variable name reached the caller")
		})
	}
}

func TestMaybeExpand(t *testing.T) {
	script := `
This is $OTHER_VAR, $ $$ $* ${} in a sentence with${ALSO_OTHER_VAR}in the middle.
We want to remember $FLEET_SECRET_BANANA and also${FLEET_SECRET_STRAWBERRY}are important.
`
	expected := `
This is $OTHER_VAR, $ $$ $* ${} in a sentence with${ALSO_OTHER_VAR}in the middle.
We want to remember BREAD and alsoSHORTCAKEare important.
`

	envVars := map[string]string{
		"BANANA":     "BREAD",
		"STRAWBERRY": "SHORTCAKE",
	}

	expectedPositions := [][]int{
		{9, 19},
		{23, 25},
		{26, 28},
		{51, 68},
		{103, 123},
		{132, 158},
	}

	mapper := func(s string, startPos, endPos int) (string, bool) {
		require.Contains(t, expectedPositions, []int{startPos, endPos}, script[startPos:endPos])

		if strings.HasPrefix(s, ServerSecretPrefix) {
			return envVars[strings.TrimPrefix(s, ServerSecretPrefix)], true
		}

		return "", false
	}

	expanded := MaybeExpand(script, mapper)

	require.Equal(t, expected, expanded)
}

func TestContainsVar(t *testing.T) {
	script := `
#!/bin/sh

echo $FLEET_SECRET_FOO is the secret ${FLEET_SECRET_ZOO}
echo words${FLEET_SECRET_BAR}words
$FLEET_SECRET_BAZ
${FLEET_SECRET_END}
`
	require.True(t, ContainsVar(script, "FLEET_SECRET_FOO"))
	require.True(t, ContainsVar(script, "FLEET_SECRET_ZOO"))
	require.False(t, ContainsVar(script, "FLEET_SECRET_ZO"))
	require.False(t, ContainsVar(script, "FLEET_SECRET_FO"))
	require.True(t, ContainsVar(script, "FLEET_SECRET_BAR"))
	require.True(t, ContainsVar(script, "FLEET_SECRET_BAZ"))
	require.True(t, ContainsVar(script, "FLEET_SECRET_END"))
	require.False(t, ContainsVar(script, "OTHER"))
	require.False(t, ContainsVar(script, ""))
}
