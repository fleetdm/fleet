package microsoft_mdm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestESPSoftwareFailureContinuableErrorText(t *testing.T) {
	tests := []struct {
		name        string
		failedNames []string
		want        string
	}{
		{
			name:        "no names falls back to generic text",
			failedNames: nil,
			want:        "Some software failed to install. " + espContinuableErrorSuffix,
		},
		{
			name:        "empty names are skipped",
			failedNames: []string{"", ""},
			want:        "Some software failed to install. " + espContinuableErrorSuffix,
		},
		{
			name:        "one name",
			failedNames: []string{"Slack"},
			want:        "Slack failed to install. " + espContinuableErrorSuffix,
		},
		{
			name:        "two names",
			failedNames: []string{"Slack", "Zoom"},
			want:        "Slack and Zoom failed to install. " + espContinuableErrorSuffix,
		},
		{
			// The cap (espMaxFailedNamesShown) is 3, so three names list in full.
			name:        "three names use Oxford comma",
			failedNames: []string{"Slack", "Zoom", "Docker"},
			want:        "Slack, Zoom, and Docker failed to install. " + espContinuableErrorSuffix,
		},
		{
			// Above the cap, the rest is summarized as "N more": four names -> first three plus "and 1 more".
			name:        "four names list first three and one more",
			failedNames: []string{"Slack", "Zoom", "Docker", "1Password"},
			want:        "Slack, Zoom, Docker, and 1 more failed to install. " + espContinuableErrorSuffix,
		},
		{
			name:        "six names list first three and three more",
			failedNames: []string{"Slack", "Zoom", "Docker", "1Password", "Notion", "Chrome"},
			want:        "Slack, Zoom, Docker, and 3 more failed to install. " + espContinuableErrorSuffix,
		},
		{
			name:        "empty name mixed in is skipped",
			failedNames: []string{"Slack", "", "Zoom"},
			want:        "Slack and Zoom failed to install. " + espContinuableErrorSuffix,
		},
		{
			// Empties are dropped before the cap is applied, so this counts as four names, not seven.
			name:        "empties are dropped before the cap is counted",
			failedNames: []string{"Slack", "", "Zoom", "", "Docker", "", "1Password"},
			want:        "Slack, Zoom, Docker, and 1 more failed to install. " + espContinuableErrorSuffix,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ESPSoftwareFailureContinuableErrorText(tc.failedNames))
		})
	}
}
