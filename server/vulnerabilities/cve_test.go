package vulnerabilities

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func TestTranslateCPEToCVE(t *testing.T) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "TestTranslateCPEToCVE-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ds := new(mock.Store)
	ctx := context.Background()

	ds.AllCPEsFunc = func() ([]string, error) {
		return []string{}, nil
	}

	ds.BulkInsertCVEsFunc = func(cves *sync.Map) error {
		panic("not implemented")
	}

	err = TranslateCPEToCVE(ctx, ds, tempDir, kitlog.NewNopLogger())
	require.NoError(t, err)
}
