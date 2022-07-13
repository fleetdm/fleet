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
		err       error
	}{{}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, outerr := store.Exists(ctx, c.installer)
			require.Equal(t, out, c.expected)
			require.Equal(t, outerr, c.err)
		})
	}
}
