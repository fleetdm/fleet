package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mock"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	kitlog "github.com/go-kit/log"
)

func TestNewMDMProfileManagerWithoutConfig(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mdmmock.MDMAppleStore{}
	ds := new(mock.Store)
	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, nil)
	logger := kitlog.NewNopLogger()

	sch, err := newMDMProfileManager(ctx, "foo", ds, cmdr, logger)
	require.NotNil(t, sch)
	require.NoError(t, err)
}
