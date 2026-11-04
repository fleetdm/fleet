package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexLinearIntegrations(t *testing.T) {
	indexed, err := IndexLinearIntegrations([]*LinearIntegration{
		{APIKey: "k1", TeamID: "team-a"},
		{APIKey: "k2", TeamID: "team-b"},
	})
	require.NoError(t, err)
	require.Len(t, indexed, 2)
	require.Equal(t, "k2", indexed["team-b"].APIKey)

	_, err = IndexLinearIntegrations([]*LinearIntegration{
		{APIKey: "k1", TeamID: "team-a"},
		{APIKey: "k2", TeamID: "team-a"},
	})
	require.ErrorContains(t, err, "duplicate Linear integration for team id team-a")
}

func TestValidateLinearIntegrations(t *testing.T) {
	ctx := context.Background()
	ori := map[string]LinearIntegration{
		"team-a": {APIKey: "k1", TeamID: "team-a"},
		"team-b": {APIKey: "k2", TeamID: "team-b"},
	}

	t.Run("unchanged items are not tested and removed ones are returned", func(t *testing.T) {
		deleted, err := ValidateLinearIntegrations(ctx, ori, []*LinearIntegration{{APIKey: "k1", TeamID: "team-a"}})
		require.NoError(t, err)
		require.Len(t, deleted, 1)
		require.Equal(t, "team-b", deleted[0].TeamID)
	})

	t.Run("empty list deletes everything", func(t *testing.T) {
		deleted, err := ValidateLinearIntegrations(ctx, ori, []*LinearIntegration{})
		require.NoError(t, err)
		require.Len(t, deleted, 2)
	})

	t.Run("duplicate team id in the new list", func(t *testing.T) {
		_, err := ValidateLinearIntegrations(ctx, ori, []*LinearIntegration{
			{APIKey: "k1", TeamID: "team-a"},
			{APIKey: "k1", TeamID: "team-a"},
		})
		require.ErrorContains(t, err, "duplicate Linear integration for team id team-a")
	})

	t.Run("missing or masked key fails before any request", func(t *testing.T) {
		for _, key := range []string{"", MaskedPassword} {
			_, err := ValidateLinearIntegrations(ctx, ori, []*LinearIntegration{{APIKey: key, TeamID: "team-new"}})
			require.ErrorAs(t, err, &IntegrationTestError{})
			require.ErrorContains(t, err, "missing or invalid API key")
		}
	})

	t.Run("missing team id fails before any request", func(t *testing.T) {
		_, err := ValidateLinearIntegrations(ctx, ori, []*LinearIntegration{{APIKey: "k1", TeamID: ""}})
		var testErr IntegrationTestError
		require.True(t, errors.As(err, &testErr))
		require.ErrorContains(t, err, "missing team id")
	})
}

func TestLinearObfuscateAndClone(t *testing.T) {
	ac := &AppConfig{}
	ac.Integrations.Linear = []*LinearIntegration{{APIKey: "secret", TeamID: "team-a"}}

	cloned, err := ac.Clone()
	require.NoError(t, err)
	clonedAC := cloned.(*AppConfig)
	require.Len(t, clonedAC.Integrations.Linear, 1)
	clonedAC.Integrations.Linear[0].APIKey = "mutated"
	assert.Equal(t, "secret", ac.Integrations.Linear[0].APIKey)

	ac.Obfuscate()
	assert.Equal(t, MaskedPassword, ac.Integrations.Linear[0].APIKey)
	assert.Equal(t, "team-a", ac.Integrations.Linear[0].TeamID)
	assert.Equal(t, "mutated", clonedAC.Integrations.Linear[0].APIKey)
}

func TestLinearIntegrationJSON(t *testing.T) {
	// no linear key is emitted when none is configured, so existing config output is unchanged
	b, err := json.Marshal(Integrations{})
	require.NoError(t, err)
	assert.NotContains(t, string(b), "linear")

	var ac AppConfig
	err = json.Unmarshal([]byte(`{"integrations":{"linear":[{"api_key":"k","team_id":"team-a"}]}}`), &ac)
	require.NoError(t, err)
	require.Len(t, ac.Integrations.Linear, 1)
	assert.Equal(t, "k", ac.Integrations.Linear[0].APIKey)
	assert.Equal(t, "team-a", ac.Integrations.Linear[0].TeamID)

	// a request that omits linear leaves it nil, which the app config update treats as "keep"
	var omitted AppConfig
	err = json.Unmarshal([]byte(`{"integrations":{"jira":[],"zendesk":[]}}`), &omitted)
	require.NoError(t, err)
	assert.Nil(t, omitted.Integrations.Linear)
}
