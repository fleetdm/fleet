package fleetctl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerCompose(t *testing.T) {
	t.Parallel()
	t.Run("returns the right command according to the version", func(t *testing.T) {
		v1 := dockerCompose{dockerComposeV1}
		cmd1 := v1.Command("up")
		require.Equal(t, []string{"docker-compose", "up"}, cmd1.Args)

		v2 := dockerCompose{dockerComposeV2}
		cmd2 := v2.Command("up")
		require.Equal(t, []string{"docker", "compose", "up"}, cmd2.Args)
	})

	t.Run("strings according to the version", func(t *testing.T) {
		v1 := dockerCompose{dockerComposeV1}
		require.Equal(t, v1.String(), "`docker-compose`")

		v2 := dockerCompose{dockerComposeV2}
		require.Equal(t, v2.String(), "`docker compose`")
	})
}
