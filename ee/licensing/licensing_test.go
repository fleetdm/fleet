package licensing

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPublicKey(t *testing.T) {
	t.Parallel()

	key, err := loadPublicKey()
	require.NoError(t, err)
	require.NotNil(t, key)
}

func TestLoadLicense(t *testing.T) {
	t.Parallel()

	key := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjQwOTk1MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjEwMCwibm90ZSI6ImZvciBkZXZlbG9wbWVudCBvbmx5IiwidGllciI6ImJhc2ljIiwiaWF0IjoxNjIyNDI2NTg2fQ.WmZ0kG4seW3IrNvULCHUPBSfFdqj38A_eiXdV_DFunMHechjHbkwtfkf1J6JQJoDyqn8raXpgbdhafDwv3rmDw"
	license, err := LoadLicense(key)
	require.NoError(t, err)
	assert.Equal(t,
		&kolide.LicenseInfo{
			Tier:         kolide.TierBasic,
			Organization: "development",
			DeviceCount:  100,
			Expiration:   time.Unix(1640995200, 0),
			Note:         "for development only",
		},
		license,
	)
	assert.Equal(t, kolide.TierBasic, license.Tier)
}
