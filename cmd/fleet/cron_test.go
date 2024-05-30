package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fleetdm/fleet/v4/server/config"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
)

func TestNewMDMProfileManagerWithoutConfig(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mock.MDMAppleStore{}
	ds := new(mock.Store)
	mdmConfig := config.MDMConfig{}
	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, nil, mdmConfig)
	logger := kitlog.NewNopLogger()

	sch, err := newMDMProfileManager(ctx, "foo", ds, cmdr, logger, false, mdmConfig)
	require.NotNil(t, sch)
	require.NoError(t, err)
}
