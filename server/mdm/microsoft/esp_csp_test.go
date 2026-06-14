package microsoft_mdm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestESPSoftwareFailureContinuableErrorText(t *testing.T) {
	const suffix = "Reset your device to try again, or proceed and install missing software via self-service. " +
		"If unavailable, contact your IT admin."

	tests := []struct {
		name        string
		failedNames []string
		want        string
	}{
		{
			name:        "no names falls back to generic text",
			failedNames: nil,
			want:        "Some software failed to install. " + suffix,
		},
		{
			name:        "empty names are skipped",
			failedNames: []string{"", ""},
			want:        "Some software failed to install. " + suffix,
		},
		{
			name:        "one name",
			failedNames: []string{"Slack"},
			want:        "Slack failed to install. " + suffix,
		},
		{
			name:        "two names",
			failedNames: []string{"Slack", "Zoom"},
			want:        "Slack and Zoom failed to install. " + suffix,
		},
		{
			name:        "three names use Oxford comma",
			failedNames: []string{"Slack", "Zoom", "Docker"},
			want:        "Slack, Zoom, and Docker failed to install. " + suffix,
		},
		{
			name:        "four names",
			failedNames: []string{"Slack", "Zoom", "Docker", "1Password"},
			want:        "Slack, Zoom, Docker, and 1Password failed to install. " + suffix,
		},
		{
			name:        "empty name mixed in is skipped",
			failedNames: []string{"Slack", "", "Zoom"},
			want:        "Slack and Zoom failed to install. " + suffix,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ESPSoftwareFailureContinuableErrorText(tc.failedNames))
		})
	}

	t.Run("long list is truncated to N more", func(t *testing.T) {
		manyNames := make([]string, 50)
		for i := range manyNames {
			manyNames[i] = fmt.Sprintf("Application With A Fairly Long Name %02d", i)
		}
		got := ESPSoftwareFailureContinuableErrorText(manyNames)
		assert.Contains(t, got, manyNames[0], "first name must always be included")
		assert.Contains(t, got, "more failed to install. ", "truncated list must summarize the remainder as N more")
		assert.NotContains(t, got, manyNames[len(manyNames)-1], "names past the cap must not be listed")
		assert.True(t, strings.HasSuffix(got, suffix))
		// The cap bounds the name list; the full message adds ", and N more failed to install. " plus the suffix.
		assert.Less(t, len(got), espMaxFailedNamesLen+len(suffix)+50, "message must stay near the cap")
	})

	t.Run("single name longer than the cap is still included", func(t *testing.T) {
		hugeName := strings.Repeat("x", espMaxFailedNamesLen+50)
		got := ESPSoftwareFailureContinuableErrorText([]string{hugeName})
		assert.Equal(t, hugeName+" failed to install. "+suffix, got)
	})
}
