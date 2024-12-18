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
	secrets := ContainsPrefixVars(script, FLEET_SECRET_PREFIX)
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

	mapping := map[string]string{
		"BANANA":     "BREAD",
		"STRAWBERRY": "SHORTCAKE",
	}

	mapper := func(s string) (string, bool) {
		if strings.HasPrefix(s, FLEET_SECRET_PREFIX) {
			return mapping[strings.TrimPrefix(s, FLEET_SECRET_PREFIX)], true
		}
		return "", false
	}

	expanded := MaybeExpand(script, mapper)

	require.Equal(t, expected, expanded)
}
