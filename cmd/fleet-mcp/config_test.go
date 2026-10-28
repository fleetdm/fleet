package main

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLiveQueryWaits(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	cases := []struct {
		raw                          string
		wantCampaign, wantSingleHost time.Duration
	}{
		{"", 25 * time.Second, 25 * time.Second},
		{"45s", 45 * time.Second, 45 * time.Second},
		// The campaign wait budget can be minutes; the single-host block must not inherit that.
		{"900s", 900 * time.Second, maxSingleHostBlock},
		{"-45s", 25 * time.Second, 25 * time.Second},
		{"bogus", 25 * time.Second, 25 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			campaign, singleHost := liveQueryWaits(t.Context(), logger, tc.raw)
			assert.Equal(t, tc.wantCampaign, campaign)
			assert.Equal(t, tc.wantSingleHost, singleHost)
		})
	}
}
