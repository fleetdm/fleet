package service

import (
	"context"
	"testing"
	"time"

	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/inmem"
	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpectedCheckinInterval(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	require.Nil(t, ds.MigrateData())
	svc, err := newTestService(ds, nil)
	require.Nil(t, err)
	ctx := context.Background()

	var distributedInterval uint
	var distributedIntervalID uint
	var loggerTlsPeriod uint
	var loggerTlsPeriodID uint

	updateLocalOptionValues := func(opts []kolide.Option) {
		for _, option := range opts {
			if option.Name == "distributed_interval" {
				distributedInterval = uint(option.Value.Val.(int))
				distributedIntervalID = option.ID
			}
			if option.Name == "logger_tls_period" {
				loggerTlsPeriod = uint(option.Value.Val.(int))
				loggerTlsPeriodID = option.ID
			}
		}
	}

	options, err := svc.GetOptions(ctx)
	require.Nil(t, err)
	updateLocalOptionValues(options)
	require.Equal(t, 10, int(distributedInterval))
	require.Equal(t, 10, int(loggerTlsPeriod))
	interval, err := svc.ExpectedCheckinInterval(ctx)
	require.Nil(t, err)
	assert.Equal(t, 10*time.Second*expectedCheckinIntervalMultiplier, interval)

	options, err = svc.ModifyOptions(ctx, kolide.OptionRequest{
		Options: []kolide.Option{
			kolide.Option{
				ID:   distributedIntervalID,
				Name: "distributed_interval",
				Value: kolide.OptionValue{
					Val: 50,
				},
				Type:     kolide.OptionTypeInt,
				ReadOnly: false,
			},
			kolide.Option{
				ID:   loggerTlsPeriodID,
				Name: "logger_tls_period",
				Value: kolide.OptionValue{
					Val: 100,
				},
				Type:     kolide.OptionTypeInt,
				ReadOnly: false,
			},
		},
	},
	)
	require.Nil(t, err)

	options, err = svc.GetOptions(ctx)
	require.Nil(t, err)
	updateLocalOptionValues(options)
	require.Equal(t, 50, int(distributedInterval))
	require.Equal(t, 100, int(loggerTlsPeriod))
	interval, err = svc.ExpectedCheckinInterval(ctx)
	require.Nil(t, err)
	assert.Equal(t, 50*time.Second*expectedCheckinIntervalMultiplier, interval)

	options, err = svc.ModifyOptions(ctx, kolide.OptionRequest{
		Options: []kolide.Option{
			kolide.Option{
				ID:   loggerTlsPeriodID,
				Name: "logger_tls_period",
				Value: kolide.OptionValue{
					Val: 20,
				},
				Type:     kolide.OptionTypeInt,
				ReadOnly: false,
			},
		},
	},
	)
	require.Nil(t, err)

	options, err = svc.GetOptions(ctx)
	require.Nil(t, err)
	updateLocalOptionValues(options)
	require.Equal(t, 50, int(distributedInterval))
	require.Equal(t, 20, int(loggerTlsPeriod))
	interval, err = svc.ExpectedCheckinInterval(ctx)
	require.Nil(t, err)
	assert.Equal(t, 20*time.Second*expectedCheckinIntervalMultiplier, interval)

	// Set the interval low enough to hit the minimum threshold
	options, err = svc.ModifyOptions(ctx, kolide.OptionRequest{
		Options: []kolide.Option{
			kolide.Option{
				ID:   loggerTlsPeriodID,
				Name: "logger_tls_period",
				Value: kolide.OptionValue{
					Val: 2,
				},
				Type:     kolide.OptionTypeInt,
				ReadOnly: false,
			},
		},
	},
	)
	require.Nil(t, err)

	options, err = svc.GetOptions(ctx)
	require.Nil(t, err)
	updateLocalOptionValues(options)
	require.Equal(t, 50, int(distributedInterval))
	require.Equal(t, 2, int(loggerTlsPeriod))
	interval, err = svc.ExpectedCheckinInterval(ctx)
	require.Nil(t, err)
	assert.Equal(t, minimumExpectedCheckinInterval, interval)
}
