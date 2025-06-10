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
