package service

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestNewUserSSO(t *testing.T) {
	sampleSize := 10000000
	keySize := 14
	extraTries := 0

	for i := 1; i < sampleSize; i++ {
		fakePassword := ""
		tries := 0
		for fakePassword == "" && tries < 10 {
			str, err := server.GenerateRandomText(keySize)
			require.NoError(t, err)
			if err = fleet.ValidatePasswordRequirements(str); err == nil {
				fakePassword = str
			}
			tries++
		}
		if tries > 2 {
			extraTries++
		}
		require.NotEmpty(t, fakePassword)
	}
	fmt.Println(float64(extraTries) / float64(sampleSize))
}
