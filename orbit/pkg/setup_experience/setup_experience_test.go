package setupexperience

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSetupExperienceStatusFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Test first execution of reading the setup experience file.
	s, err := ReadSetupExperienceStatusFile(tmpDir)
	require.NoError(t, err)
	require.Nil(t, s)

	err = WriteSetupExperienceStatusFile(tmpDir, &SetupExperienceInfo{
		TimeInitiated: time.Now(),
		Enabled:       true,
	})
	require.NoError(t, err)

	s, err = ReadSetupExperienceStatusFile(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, s)
	require.NotZero(t, s.TimeInitiated)
	timeInitiated := s.TimeInitiated
	require.True(t, s.Enabled)
	require.Nil(t, s.TimeFinished)

	s.TimeFinished = ptr.Time(time.Now())
	err = WriteSetupExperienceStatusFile(tmpDir, s)
	require.NoError(t, err)

	s, err = ReadSetupExperienceStatusFile(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, s)
	require.Equal(t, timeInitiated, s.TimeInitiated)
	require.True(t, s.Enabled)
	require.NotNil(t, s.TimeFinished)
	require.NotZero(t, *s.TimeFinished)
}
