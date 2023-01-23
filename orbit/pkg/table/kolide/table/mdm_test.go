//go:build darwin
// +build darwin

package table

import (
	"context"
	"testing"

	"github.com/kolide/kit/env"
	"github.com/stretchr/testify/require"
)

func TestMDMProfileStatus(t *testing.T) {
	t.Parallel()

	if env.Bool("SKIP_TEST_MDM", true) {
		t.Skip("Skipping MDM Test")
	}

	_, err := getMDMProfileStatus(context.TODO())
	require.Nil(t, err)
}
