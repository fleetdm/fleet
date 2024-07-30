package webhooks

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerHostStatusWebhook(t *testing.T) {
	ds := new(mock.Store)

	requestBody := ""
	count := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(requestBodyBytes)
		count++
	}))
	defer ts.Close()

	ac := &fleet.AppConfig{
		WebhookSettings: fleet.WebhookSettings{
			HostStatusWebhook: fleet.HostStatusWebhookSettings{
				Enable:         true,
				DestinationURL: ts.URL,
				HostPercentage: 43,
				DaysCount:      2,
			},
		},
	}

	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return ac, nil
	}

	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, teamID *uint, daysCount int) (int, []uint, error) {
		assert.Equal(t, 2, daysCount)
		return 10, []uint{1, 2, 3, 4, 5, 6}, nil
	}

	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		return nil, nil
	}

	require.NoError(t, TriggerHostStatusWebhook(context.Background(), ds, kitlog.NewNopLogger()))
	assert.Equal(
		t,
		`{"data":{"days_unseen":2,"host_ids":[1,2,3,4,5,6],"total_hosts":10,"unseen_hosts":6},"text":"More than 60.00% of your hosts have not checked into Fleet for more than 2 days. You've been sent this message because the Host status webhook is enabled in your Fleet instance."}`,
		requestBody,
	)
	assert.Equal(t, 1, count)

	requestBody = ""
	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, teamID *uint, daysCount int) (int, []uint, error) {
		assert.Equal(t, 2, daysCount)
		return 10, []uint{1}, nil
	}

	require.NoError(t, TriggerHostStatusWebhook(context.Background(), ds, kitlog.NewNopLogger()))
	assert.Equal(t, "", requestBody)
	assert.Equal(t, 1, count)
}

func TestTriggerHostStatusWebhookTeam(t *testing.T) {
	ds := new(mock.Store)

	requestBody := ""
	count := 0

	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				requestBodyBytes, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				requestBody = string(requestBodyBytes)
				count++
			},
		),
	)
	defer ts.Close()

	ac := &fleet.AppConfig{
		WebhookSettings: fleet.WebhookSettings{
			HostStatusWebhook: fleet.HostStatusWebhookSettings{
				Enable:         false,
				DestinationURL: ts.URL,
				HostPercentage: 43,
				DaysCount:      3,
			},
		},
	}
	teamSettings := fleet.HostStatusWebhookSettings{
		Enable:         true,
		DestinationURL: ts.URL,
		HostPercentage: 43,
		DaysCount:      2,
	}

	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return ac, nil
	}

	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, teamID *uint, daysCount int) (int, []uint, error) {
		assert.Equal(t, 2, daysCount)
		assert.Equal(t, uint(1), *teamID)
		return 10, []uint{1, 2, 3, 4, 5, 6}, nil
	}

	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		return []*fleet.TeamSummary{{ID: 1}}, nil
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		assert.Equal(t, uint(1), id)
		return &fleet.Team{
			ID: 1,
			Config: fleet.TeamConfig{
				WebhookSettings: fleet.TeamWebhookSettings{
					HostStatusWebhook: &teamSettings,
				},
			},
		}, nil
	}

	require.NoError(t, TriggerHostStatusWebhook(context.Background(), ds, kitlog.NewNopLogger()))
	assert.Equal(
		t,
		`{"data":{"days_unseen":2,"host_ids":[1,2,3,4,5,6],"team_id":1,"total_hosts":10,"unseen_hosts":6},"text":"More than 60.00% of your hosts have not checked into Fleet for more than 2 days. You've been sent this message because the Host status webhook is enabled in your Fleet instance."}`,
		requestBody,
	)
	assert.Equal(t, 1, count)

	requestBody = ""
	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, teamID *uint, daysCount int) (int, []uint, error) {
		assert.Equal(t, 2, daysCount)
		assert.Equal(t, uint(1), *teamID)
		return 10, []uint{1}, nil
	}

	require.NoError(t, TriggerHostStatusWebhook(context.Background(), ds, kitlog.NewNopLogger()))
	assert.Equal(t, "", requestBody)
	assert.Equal(t, 1, count)
}
