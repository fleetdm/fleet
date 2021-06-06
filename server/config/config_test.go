package config

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestConfigRoundtrip(t *testing.T) {
	// This test verifies that a config can be roundtripped through yaml.
	// Doing so ensures that config_dump will provide the correct config.
	// Newly added config values will automatically be tested in this
	// function because of the reflection on the config struct.

	cmd := &cobra.Command{}
	// Leaving this flag unset means that no attempt will be made to load
	// the config file
	cmd.PersistentFlags().StringP("config", "c", "", "Path to a configuration file")
	man := NewManager(cmd)

	// Use reflection magic to walk the config struct, setting unique
	// values to be verified on the roundtrip. Note that bools are always
	// set to true, which could false positive if the default value is
	// true.
	original := &FleetConfig{}
	v := reflect.ValueOf(original)
	for conf_index := 0; conf_index < v.Elem().NumField(); conf_index++ {
		conf_v := v.Elem().Field(conf_index)
		for key_index := 0; key_index < conf_v.NumField(); key_index++ {
			key_v := conf_v.Field(key_index)
			switch key_v.Interface().(type) {
			case string:
				switch conf_v.Type().Field(key_index).Name {
				case "TLSProfile":
					// we have to explicitly set value for this key as it will only
					// accept intermediate or modern
					key_v.SetString(TLSProfileModern)
				default:
					key_v.SetString(v.Elem().Type().Field(conf_index).Name + "_" + conf_v.Type().Field(key_index).Name)
				}
			case int:
				key_v.SetInt(int64(conf_index*100 + key_index))
			case bool:
				key_v.SetBool(true)
			case time.Duration:
				d := time.Duration(conf_index*100 + key_index)
				key_v.Set(reflect.ValueOf(d))
			}
		}
	}

	// Marshal the generated config
	buf, err := yaml.Marshal(original)
	require.Nil(t, err)

	// Manually load the serialized config
	man.viper.SetConfigType("yaml")
	err = man.viper.ReadConfig(bytes.NewReader(buf))
	require.Nil(t, err)

	// Ensure the read config is the same as the original
	assert.Equal(t, *original, man.LoadConfig())
}
