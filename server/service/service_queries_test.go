package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestNewQueryAttach(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	name := "bad"
	query := "attach '/nope' as bad"
	_, err := svc.NewQuery(
		context.Background(),
		fleet.QueryPayload{Name: &name, Query: &query},
	)
	require.Error(t, err)
}
