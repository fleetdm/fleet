package fleet

import (
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/stretchr/testify/require"
)

func TestFindCustomHostVitalIDs(t *testing.T) {
	t.Run("both token forms and dedupe", func(t *testing.T) {
		doc := `
#!/bin/sh
echo $FLEET_HOST_VITAL_1
echo words${FLEET_HOST_VITAL_2}words
echo $FLEET_HOST_VITAL_1 again
`
		ids := FindCustomHostVitalIDs(doc)
		require.ElementsMatch(t, []uint{1, 2}, ids)
	})

	t.Run("ignores non-numeric and zero suffixes", func(t *testing.T) {
		doc := `$FLEET_HOST_VITAL_ABC ${FLEET_HOST_VITAL_} $FLEET_HOST_VITAL_0 $FLEET_HOST_VITAL_12X $FLEET_HOST_VITAL_7`
		ids := FindCustomHostVitalIDs(doc)
		require.Equal(t, []uint{7}, ids)
	})

	t.Run("no tokens", func(t *testing.T) {
		require.Empty(t, FindCustomHostVitalIDs("no vitals here $FLEET_SECRET_FOO $FLEET_VAR_HOST_UUID"))
	})

	t.Run("does not match a longer variable that starts with the prefix name", func(t *testing.T) {
		// FLEET_VAR_ prefix should not be caught.
		require.Empty(t, FindCustomHostVitalIDs("$FLEET_VAR_HOST_VITAL_1"))
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

func TestCustomHostVitalUsedInfoMessageHostNameTemplate(t *testing.T) {
	info := CustomHostVitalUsedInfo{
		CustomHostVitalID:   5,
		CustomHostVitalName: "FUNCTION",
		Entity: EntityUsingCustomHostVital{
			Type:      CustomHostVitalEntityHostNameTemplate,
			FleetName: "Workstations",
		},
	}
	want := `Custom host vital "FUNCTION" (used as $FLEET_HOST_VITAL_5) is used by the host name template in the "Workstations" fleet. Please edit or clear the host name template and try again.`
	require.Equal(t, want, info.Message())

	err := (&CustomHostVitalUsedError{CustomHostVitalUsedInfo: info}).Error()
	require.Equal(t, want, err)
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

// TestIsInvalidReferencedCustomHostVitalsError covers the classification
// callers of ValidateReferencedCustomHostVitals rely on to decide whether to
// report a 422 (unknown ID / malformed reference) or propagate the error as-is
// (any other error, e.g. a wrapped DB failure, which must surface as a 500).
func TestIsInvalidReferencedCustomHostVitalsError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"unknown vital id", &MissingCustomHostVitalsError{MissingIDs: []uint{5}}, true},
		{"malformed reference", &InvalidCustomHostVitalRefError{Refs: []string{"FLEET_HOST_VITAL_asset_tag"}}, true},
		{"plain infra error", errors.New("connection refused"), false},
		{"wrapped infra error", ctxerr.Wrap(t.Context(), errors.New("connection refused"), "validating custom host vitals"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, IsInvalidReferencedCustomHostVitalsError(c.err))
		})
	}
}
