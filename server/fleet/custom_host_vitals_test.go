package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainsCustomHostVitalIDs(t *testing.T) {
	t.Run("both token forms and dedupe", func(t *testing.T) {
		doc := `
#!/bin/sh
echo $FLEET_HOST_VITAL_1
echo words${FLEET_HOST_VITAL_2}words
echo $FLEET_HOST_VITAL_1 again
`
		ids := ContainsCustomHostVitalIDs(doc)
		require.ElementsMatch(t, []uint{1, 2}, ids)
	})

	t.Run("ignores non-numeric and zero suffixes", func(t *testing.T) {
		doc := `$FLEET_HOST_VITAL_ABC ${FLEET_HOST_VITAL_} $FLEET_HOST_VITAL_0 $FLEET_HOST_VITAL_12X $FLEET_HOST_VITAL_7`
		ids := ContainsCustomHostVitalIDs(doc)
		require.Equal(t, []uint{7}, ids)
	})

	t.Run("no tokens", func(t *testing.T) {
		require.Empty(t, ContainsCustomHostVitalIDs("no vitals here $FLEET_SECRET_FOO $FLEET_VAR_HOST_UUID"))
	})

	t.Run("does not match a longer variable that starts with the prefix name", func(t *testing.T) {
		// FLEET_VAR_ prefix should not be caught.
		require.Empty(t, ContainsCustomHostVitalIDs("$FLEET_VAR_HOST_VITAL_1"))
	})
}

func TestMissingCustomHostVitalsError(t *testing.T) {
	single := MissingCustomHostVitalsError{MissingIDs: []uint{5}}
	require.Contains(t, single.Error(), `"$FLEET_HOST_VITAL_5"`)
	require.Contains(t, single.Error(), "Custom host vital ")

	multi := MissingCustomHostVitalsError{MissingIDs: []uint{5, 9}}
	require.Contains(t, multi.Error(), "Custom host vitals")
	require.Contains(t, multi.Error(), `"$FLEET_HOST_VITAL_5"`)
	require.Contains(t, multi.Error(), `"$FLEET_HOST_VITAL_9"`)
}

func TestMissingCustomHostVitalValueError(t *testing.T) {
	single := MissingCustomHostVitalValueError{MissingIDs: []uint{5}, MissingNames: []string{"Asset tag"}}
	require.Equal(
		t,
		"Couldn't populate the custom host vital Asset tag ($FLEET_HOST_VITAL_5) because there's no value set for this host.",
		single.Error(),
	)
	// Distinct from the upload-time "is not defined" wording.
	require.NotContains(t, single.Error(), "is not defined")

	multi := MissingCustomHostVitalValueError{MissingIDs: []uint{5, 9}, MissingNames: []string{"Asset tag", "Department"}}
	require.Contains(t, multi.Error(), "custom host vitals")
	require.Contains(t, multi.Error(), "Asset tag ($FLEET_HOST_VITAL_5)")
	require.Contains(t, multi.Error(), "Department ($FLEET_HOST_VITAL_9)")
}
