package datastore

import (
	"reflect"
	"testing"

	"github.com/kolide/kolide/server/datastore/internal/appstate"
	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOptions(t *testing.T, ds kolide.Datastore) {
	require.Nil(t, ds.MigrateData())
	// were options pre-loaded?
	opts, err := ds.ListOptions()
	require.Nil(t, err)
	assert.Len(t, opts, len(appstate.Options()))

	opt, err := ds.OptionByName("aws_access_key_id")
	require.Nil(t, err)
	require.NotNil(t, opt)
	opt2, err := ds.Option(opt.ID)
	require.Nil(t, err)
	require.NotNil(t, opt2)
	assert.True(t, reflect.DeepEqual(opt, opt2))

	opt.SetValue("somekey")
	err = ds.SaveOptions([]kolide.Option{*opt})
	require.Nil(t, err)
	opt, err = ds.Option(opt.ID)
	require.Nil(t, err)
	assert.Equal(t, "somekey", opt.GetValue())

	// can't change a read only option
	opt, err = ds.OptionByName("disable_distributed")
	require.Nil(t, err)
	opt.SetValue(true)
	err = ds.SaveOptions([]kolide.Option{*opt})
	require.NotNil(t, err)

	opt, _ = ds.OptionByName("aws_profile_name")
	assert.False(t, opt.OptionSet())
	opt.SetValue("zip")
	opt2, _ = ds.OptionByName("disable_distributed")
	assert.Equal(t, false, opt2.GetValue())
	opt2.SetValue(true)
	modList := []kolide.Option{*opt, *opt2}
	// The aws access key option can be saved but because the disable_events can't
	// be we want to verify that the whole transaction is rolled back
	err = ds.SaveOptions(modList)
	assert.NotNil(t, err)
	opt, _ = ds.OptionByName("aws_profile_name")
	assert.False(t, opt.OptionSet())
	opt2, err = ds.OptionByName("disable_distributed")
	require.Nil(t, err)
	assert.Equal(t, false, opt2.GetValue())
}

func testOptionsToConfig(t *testing.T, ds kolide.Datastore) {
	require.Nil(t, ds.MigrateData())
	resp, err := ds.GetOsqueryConfigOptions()
	require.Nil(t, err)
	assert.Len(t, resp, 10)
	assert.Equal(t, "/api/v1/osquery/distributed/read", resp["distributed_tls_read_endpoint"])
	opt, _ := ds.OptionByName("aws_profile_name")
	assert.False(t, opt.OptionSet())
	opt.SetValue("zip")
	err = ds.SaveOptions([]kolide.Option{*opt})
	require.Nil(t, err)
	resp, err = ds.GetOsqueryConfigOptions()
	require.Nil(t, err)
	assert.Len(t, resp, 11)
	assert.Equal(t, "zip", resp["aws_profile_name"])
}
