package cloudstorage

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/stretchr/testify/require"
)

type vendorErr struct{}

func (vendorErr) Error() string { return "vendor failure" }

func TestMisconfiguredError(t *testing.T) {
	vendor := &vendorErr{}

	cases := []struct {
		name     string
		vendor   string
		code     string
		message  string
		resource string
		want     string
	}{
		{
			name:     "all fields",
			vendor:   "S3",
			code:     "NoSuchBucket",
			message:  "The specified bucket does not exist.",
			resource: "bucket: my-bucket",
			want:     "Cloud storage is misconfigured. S3: api error NoSuchBucket: The specified bucket does not exist. (bucket: my-bucket)",
		},
		{
			name:     "only the first message line is kept",
			vendor:   "Azure",
			code:     "ContainerNotFound",
			message:  "The specified container does not exist.\nRequestId:abc\nTime:2026-01-01T00:00:00.000Z",
			resource: "container: c",
			want:     "Cloud storage is misconfigured. Azure: api error ContainerNotFound: The specified container does not exist. (container: c)",
		},
		{
			name:   "no message and no resource",
			vendor: "Azure",
			code:   "403",
			want:   "Cloud storage is misconfigured. Azure: api error 403",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ctxerr.Wrap(t.Context(), NewMisconfigured(vendor, c.vendor, c.code, c.message, c.resource), "uploading")
			require.Equal(t, c.want, ctxerr.Cause(err).Error())

			// The vendor error must stay reachable.
			var got *vendorErr
			require.ErrorAs(t, err, &got)
			require.Same(t, vendor, got)

			var mis *MisconfiguredError
			require.ErrorAs(t, err, &mis)
		})
	}
}
