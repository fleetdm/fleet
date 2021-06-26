package licensing

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
		&fleet.LicenseInfo{
			Tier:         fleet.TierBasic,
			Organization: "development",
			DeviceCount:  100,
			Expiration:   time.Unix(1640995200, 0),
			Note:         "for development only",
		},
		license,
	)
	assert.Equal(t, fleet.TierBasic, license.Tier)
}

func TestLoadLicenseExpired(t *testing.T) {
	t.Parallel()

	key := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjA5NDU5MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjQyLCJ0aWVyIjoiYmFzaWMiLCJpYXQiOjE2MjI0Mjk1MTB9.pvmgQ2_6GWbGcdlm3JbNTbxFF8V6-xs2pC6zO8P96TF806W0y1TjF5G2ZjzEWCkNMk3dydaRoMHIzE7WgCaK5w"
	_, err := LoadLicense(key)
	require.Error(t, err)
}

func TestLoadLicenseNotIssuedYet(t *testing.T) {
	t.Parallel()

	// iat (issued at) is in the year 2480
	key := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjA5NDU5MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjQyLCJ0aWVyIjoiYmFzaWMiLCJpYXQiOjE2MDk0NTkyMDAwfQ.3UCxwT-kbm8OBIBylI9wXq4yStcVLaB3tYQvkmK8VNL7NQ-GrW4pjx8Ie3gS21Ub4iJtfFmessoC9lMKF5i5gw"
	_, err := LoadLicense(key)
	require.Error(t, err)
}

func TestLoadLicenseSignatureError(t *testing.T) {
	t.Parallel()

	// signature doesn't match
	key := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjA5NDU5MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRmdmljZXMiOjQyLCJ0aWVyIjoiYmFzaWMiLCJpYXQiOjE2MjI0Mjk1MTB9.pvmgQ2_6GWbGcdlm3JbNTbxFF8V6-xs2pC6zO8P96TF806W0y1TjF5G2ZjzEWCkNMk3dydaRoMHIzE7WgCaK5w"
	_, err := LoadLicense(key)
	require.Error(t, err)
}

func TestLoadLicenseIncorrectAlgorithm(t *testing.T) {
	t.Parallel()

	// signature doesn't match
	key := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjA5NDU5MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjQyLCJ0aWVyIjoiYmFzaWMiLCJpYXQiOjE2MDk0NTkyMDB9.AAAAAAAAAAAAAAAAAAAAAPi2EbMBWwhCQnCDGptBsE6E1wa4Ql42xOfuWKDzx7v-AAAAAAAAAAAAAAAAAAAAAHmQCJSjvujpV9QpY9d86v4-_OvaTnttE_ry3Xxeua84"
	_, err := LoadLicense(key)
	require.Error(t, err)
}
