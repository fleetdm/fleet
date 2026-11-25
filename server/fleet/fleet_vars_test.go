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
	for _, tc := range []struct {
		text   string
		prefix string
		want   []string
	}{
		{script, ServerSecretPrefix, []string{"FOO", "BAR", "BAZ", "QUX"}},
		{"<!-- Delivered via $FLEET_SECRET_ (kept out of git). -->\n<Data>$FLEET_SECRET_FOO</Data>", ServerSecretPrefix, []string{"FOO"}},
		{"# $FLEET_SECRET_. and $FLEET_SECRET_{X} and $FLEET_SECRET_", ServerSecretPrefix, []string{}},
		// Not a bare prefix: braced, or something follows it.
		{"$FLEET_SECRET_$X $FLEET_SECRET_${X} $FLEET_SECRET_<b> $FLEET_SECRET_&amp; $FLEET_SECRET_\\n [$FLEET_SECRET_] \"$FLEET_SECRET_\" '$FLEET_SECRET_' ${FLEET_SECRET_}", ServerSecretPrefix, []string{"", "", "", "", "", "", "", "", ""}},
		{"$FLEET_HOST_SECRET_ and $FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD", HostSecretPrefix, []string{"RECOVERY_LOCK_PASSWORD"}},
		{"$FLEET_HOST_VITAL_ and ${FLEET_HOST_VITAL_12}", CustomHostVitalPrefix, []string{"12"}},
	} {
		require.Equal(t, tc.want, ContainsPrefixVars(tc.text, tc.prefix), tc.text)
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
