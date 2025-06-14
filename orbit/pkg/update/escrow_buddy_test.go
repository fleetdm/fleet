package update

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestEscrowBuddy(t *testing.T) {
	testingSuite := new(escrowBuddyTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type escrowBuddyTestSuite struct {
	suite.Suite
	withTUF
}

func (s *escrowBuddyTestSuite) TestUpdatesDisabled() {
	t := s.T()
	cfg := &fleet.OrbitConfig{}
	cfg.Notifications.RotateDiskEncryptionKey = true
	r := NewEscrowBuddyRunner(nil, time.Second)
	err := r.Run(cfg)
	require.NoError(t, err)
}

func (s *escrowBuddyTestSuite) TestEscrowBuddyRotatesKey() {
	t := s.T()
	updater := &Updater{
		client: s.client,
		opt:    Options{Targets: make(map[string]TargetInfo), RootDirectory: t.TempDir()},
	}
	runner := &Runner{updater: updater, localHashes: make(map[string][]byte)}
	escrowBuddyPath := "escrowBuddy/macos/stable/escrowBuddy.pkg"

	cfg := &fleet.OrbitConfig{}
	r := &EscrowBuddyRunner{updateRunner: runner, interval: time.Millisecond}
	// mock the command to run the defaults cli
	cmdCalls := []map[string]any{}
	r.runCmdFunc = func(cmd string, args ...string) error {
		cmdCalls = append(cmdCalls, map[string]any{"cmd": cmd, "args": args})
		return nil
	}

	// no new target added if the notification is not set
	err := r.Run(cfg)
	require.NoError(t, err)
	targets := runner.updater.opt.Targets
	require.Len(t, targets, 0)
	require.Empty(t, cmdCalls)

	// there's an error when the remote repo doesn't have the target yet
	cfg.Notifications.RotateDiskEncryptionKey = true
	err = r.Run(cfg)
	require.ErrorContains(t, err, "tuf: file not found")
	require.Empty(t, cmdCalls)

	// add escrow buddy to the remote
	s.addRemoteTarget(escrowBuddyPath)

	err = r.Run(cfg)
	require.NoError(t, err)
	require.Len(t, cmdCalls, 2)
	require.Equal(t, cmdCalls[0]["cmd"], "sh")
	require.Equal(t, cmdCalls[0]["args"], []string{"-c", "/Library/Security/SecurityAgentPlugins/Escrow\\ Buddy.bundle/Contents/Resources/AuthDBSetup.sh"})
	require.Equal(t, cmdCalls[1]["cmd"], "sh")
	require.Equal(t, cmdCalls[1]["args"], []string{"-c", "defaults write /Library/Preferences/com.netflix.Escrow-Buddy.plist GenerateNewKey -bool true"})

	targets = runner.updater.opt.Targets
	require.Len(t, targets, 1)
	ti, ok := targets["escrowBuddy"]
	require.True(t, ok)
	require.EqualValues(t, EscrowBuddyMacOSTarget, ti)

	time.Sleep(3 * time.Millisecond)
	cfg.Notifications.RotateDiskEncryptionKey = false
	cmdCalls = []map[string]any{}
	err = r.Run(cfg)
	require.NoError(t, err)
	// only one call to set the GenerateNewKey to false
	require.Len(t, cmdCalls, 1)
	require.Equal(t, cmdCalls[0]["cmd"], "sh")
	require.Equal(t, cmdCalls[0]["args"], []string{"-c", "defaults write /Library/Preferences/com.netflix.Escrow-Buddy.plist GenerateNewKey -bool false"})

}
