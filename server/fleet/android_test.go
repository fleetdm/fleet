package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAndroidPolicyFieldValid(t *testing.T) {
	isValid := IsAndroidPolicyFieldValid("bogusKeyThatWillNeverExist")
	require.False(t, isValid)

	isValid = IsAndroidPolicyFieldValid("name") // "name" is a valid top-level policy field, that we assume will exist forever
	require.True(t, isValid)
}
