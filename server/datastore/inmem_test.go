package datastore

import (
	"testing"

	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/inmem"
	"github.com/stretchr/testify/require"
)

func TestInmem(t *testing.T) {

	for _, f := range testFunctions {
		t.Run(functionName(f), func(t *testing.T) {
			ds, err := inmem.New(config.TestConfig())
			require.Nil(t, err)
			defer func() { require.Nil(t, ds.Drop()) }()
			require.Nil(t, err)
			f(t, ds)
		})
	}
}
