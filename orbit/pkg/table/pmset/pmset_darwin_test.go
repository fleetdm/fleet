//go:build darwin
// +build darwin

package pmset

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePMSetOutput(t *testing.T) {
	// pmset -g
	sampleOutput := []byte(`System-wide power settings:
 SleepDisabled          0
Currently in use:
 lidwake              1
 lowpowermode         0
 standbydelayhigh     86400
 proximitywake        1
 standby              1
 standbydelaylow      0
 ttyskeepawake        1
 hibernatemode        3
 powernap             0
 gpuswitch            2
 hibernatefile        /var/vm/sleepimage
 highstandbythreshold 50
 displaysleep         10
 womp                 1
 networkoversleep     0
 sleep                1 (sleep prevented by bluetoothd, coreaudiod)
 acwake               0
 halfdim              1
 tcpkeepalive         1
 disksleep            10`)
	result := parsePMSetOutput(sampleOutput)

	systemWide := result["System-wide power settings:"].(map[string]string)
	require.NotNil(t, systemWide)
	require.Equal(t, systemWide["SleepDisabled"], "0")

	currInUse := result["Currently in use:"].(map[string]string)
	require.Equal(t, currInUse["powernap"], "0")
	require.Equal(t, currInUse["hibernatefile"], "/var/vm/sleepimage")
	require.Equal(t, currInUse["highstandbythreshold"], "50")
	require.Equal(t, currInUse["sleep"], "1 (sleep prevented by bluetoothd, coreaudiod)")

	// pmset -g custom
	sampleOutput = []byte(`Battery Power:
 lidwake              1
 lowpowermode         1
 standbydelayhigh     86400
 proximitywake        0
 standby              1
 standbydelaylow      10800
 ttyskeepawake        1
 hibernatemode        3
 gpuswitch            2
 powernap             0
 hibernatefile        /var/vm/sleepimage
 highstandbythreshold 50
 displaysleep         2
 womp                 0
 networkoversleep     0
 sleep                1
 lessbright           1
 halfdim              1
 tcpkeepalive         1
 acwake               0
 disksleep            10
AC Power:
 lidwake              1
 lowpowermode         0
 standbydelayhigh     86400
 proximitywake        1
 standby              1
 standbydelaylow      0
 ttyskeepawake        1
 hibernatemode        3
 powernap             0
 gpuswitch            2
 hibernatefile        /var/vm/sleepimage
 highstandbythreshold 50
 displaysleep         10
 womp                 1
 networkoversleep     0
 sleep                1
 acwake               0
 halfdim              1
 tcpkeepalive         1
 disksleep            10
`)
	result = parsePMSetOutput(sampleOutput)

	batteryPower := result["Battery Power:"].(map[string]string)
	require.NotNil(t, batteryPower)
	require.Equal(t, batteryPower["powernap"], "0")
	require.Equal(t, batteryPower["displaysleep"], "2")
	require.Equal(t, batteryPower["highstandbythreshold"], "50")
	require.Equal(t, batteryPower["hibernatefile"], "/var/vm/sleepimage")
	require.Equal(t, batteryPower["disksleep"], "10")

	acPower := result["AC Power:"].(map[string]string)
	require.Equal(t, acPower["powernap"], "0")
	require.Equal(t, acPower["displaysleep"], "10")
	require.Equal(t, acPower["highstandbythreshold"], "50")
	require.Equal(t, acPower["hibernatefile"], "/var/vm/sleepimage")
	require.Equal(t, acPower["disksleep"], "10")
}
