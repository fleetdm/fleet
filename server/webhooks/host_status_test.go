package webhooks

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerHostStatusWebhook(t *testing.T) {
	ds := new(mock.Store)

	requestBody := ""

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBodyBytes, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(requestBodyBytes)
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

	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, daysCount int) (int, int, error) {
		assert.Equal(t, 2, daysCount)
		return 10, 6, nil
	}

	require.NoError(t, TriggerHostStatusWebhook(context.Background(), ds, kitlog.NewNopLogger(), ac))
	assert.Equal(
		t,
		`{"data":{"days_unseen":2,"total_hosts":10,"unseen_hosts":6},"text":"More than 60.00% of your hosts have not checked into Fleet for more than 2 days. You've been sent this message because the Host status webhook is enabled in your Fleet instance."}`,
		requestBody,
	)
	requestBody = ""

	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, daysCount int) (int, int, error) {
		assert.Equal(t, 2, daysCount)
		return 10, 1, nil
	}

	require.NoError(t, TriggerHostStatusWebhook(context.Background(), ds, kitlog.NewNopLogger(), ac))
	assert.Equal(t, "", requestBody)
}
