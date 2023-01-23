//go:build darwin
// +build darwin

package macos_software_update

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func Test_generateRecommendedUpdatesHappyPath(t *testing.T) {
	t.Parallel()
	table := Table{logger: log.NewNopLogger()}

	_, err := table.generate(context.Background(), tablehelpers.MockQueryContext(nil))

	// Since the output is dynamic and can be empty, just verify no error
	require.NoError(t, err)
}
