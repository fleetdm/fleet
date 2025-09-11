package fleet

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestTempFileReader(t *testing.T) {
	content1And2 := "Hello, World!"
	tfr1, err := NewTempFileReader(strings.NewReader(content1And2), t.TempDir)
	require.NoError(t, err)
	tfr2, err := NewTempFileReader(strings.NewReader(content1And2), t.TempDir)
	require.NoError(t, err)

	content3 := "Hello, Temp!"
	keepFile, err := os.CreateTemp(t.TempDir(), "test")
	require.NoError(t, err)
	_, err = io.Copy(keepFile, strings.NewReader(content3))
	require.NoError(t, err)
	err = keepFile.Close()
	require.NoError(t, err)
	tfr3, err := NewKeepFileReader(keepFile.Name())
	require.NoError(t, err)

	b, err := io.ReadAll(tfr1)
	require.NoError(t, err)
	require.Equal(t, content1And2, string(b))
	b, err = io.ReadAll(tfr2)
	require.NoError(t, err)
	require.Equal(t, content1And2, string(b))

	// rewind and read again gets the same content
	err = tfr1.Rewind()
	require.NoError(t, err)
	b, err = io.ReadAll(tfr1)
	require.NoError(t, err)
	require.Equal(t, content1And2, string(b))

	// tfr2 is at EOF, so it reads nothing
	b, err = io.ReadAll(tfr2)
	require.NoError(t, err)
	require.Equal(t, "", string(b))

	b, err = io.ReadAll(tfr3)
	require.NoError(t, err)
	require.Equal(t, content3, string(b))

	// closing deletes the file
	err = tfr1.Close()
	require.NoError(t, err)
	_, err = os.Stat(tfr1.Name())
	require.True(t, os.IsNotExist(err))

	// tfr2 still exists
	_, err = os.Stat(tfr2.Name())
	require.False(t, os.IsNotExist(err))

	// tfr3 still exists even after Close
	err = tfr3.Close()
	require.NoError(t, err)
	_, err = os.Stat(tfr3.Name())
	require.False(t, os.IsNotExist(err))
}

func TestForMyDevicePage(t *testing.T) {
	var iconUrl *string
	var hostSoftwareInstaller HostSoftwareWithInstaller

	testCases := []struct {
		name     string
		before   func()
		testFunc func(*testing.T)
	}{
		{
			name: "no icon",
			before: func() {
				iconUrl = nil
				hostSoftwareInstaller = HostSoftwareWithInstaller{
					IconUrl: iconUrl,
					ID:      1,
				}
			},
			testFunc: func(t *testing.T) {
				hostSoftwareInstaller.ForMyDevicePage("token")
				require.Nil(t, hostSoftwareInstaller.IconUrl)
			},
		},
		{
			name: "not fleet custom icon url",
			before: func() {
				iconUrl = ptr.String("https://example.com/icon.png")
				hostSoftwareInstaller = HostSoftwareWithInstaller{
					IconUrl: iconUrl,
					ID:      1,
				}
			},
			testFunc: func(t *testing.T) {
				hostSoftwareInstaller.ForMyDevicePage("token")
				require.NotNil(t, hostSoftwareInstaller.IconUrl)
				require.Equal(t, *iconUrl, *hostSoftwareInstaller.IconUrl)
			},
		},
		{
			name: "matching custom icon url",
			before: func() {
				iconUrl = ptr.String("/api/latest/fleet/software/titles/42/icon?team_id=7")
				hostSoftwareInstaller = HostSoftwareWithInstaller{
					IconUrl: iconUrl,
					ID:      1,
				}
			},
			testFunc: func(t *testing.T) {
				auth := "71f0c624-497c-4dc4-aedf-6cedddcc643d"
				expectedIconUrl := fmt.Sprintf("/api/latest/fleet/device/%s/software/titles/%d/icon", auth, hostSoftwareInstaller.ID)
				hostSoftwareInstaller.ForMyDevicePage(auth)
				require.NotNil(t, hostSoftwareInstaller.IconUrl)
				require.Equal(t, expectedIconUrl, *hostSoftwareInstaller.IconUrl)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			tc.testFunc(t)
		})
	}
}
