package s3

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanAccess(t *testing.T) {
	ctx := context.Background()
	store := setupInstallerStore(t, "", "")

	cases := []struct {
		name      string
		installer Installer
		expected  bool
	}{{}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := store.CanAccess(ctx, c.installer)
			require.Equal(t, c.expected, out)
		})
	}
}
