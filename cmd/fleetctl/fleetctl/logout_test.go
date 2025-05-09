package fleetctl

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestLogout(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	ds.SessionByIDFunc = func(ctx context.Context, id uint) (*fleet.Session, error) {
		return &fleet.Session{
			ID:         333,
			AccessedAt: time.Now(),
			UserID:     123,
			Key:        "12344321",
		}, nil
	}
	ds.DestroySessionFunc = func(ctx context.Context, session *fleet.Session) error {
		return nil
	}

	assert.Equal(t, "", RunAppForTest(t, []string{"logout"}))
	assert.True(t, ds.DestroySessionFuncInvoked)
}
