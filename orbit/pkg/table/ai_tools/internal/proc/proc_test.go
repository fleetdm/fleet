package proc

import (
	"os"
	"os/user"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectionsLoadedOnFirstUse(t *testing.T) {
	loads := 0
	s := NewSnapshot(map[int]Process{7: {PID: 7, Name: "node"}}, func() []Conn {
		loads++
		return []Conn{{PID: 7, Status: "LISTEN", LocalPort: 3000}}
	})
	require.Zero(t, loads)

	require.Equal(t, 3000, s.ListenPort(7))
	require.Len(t, s.Connections(), 1)
	require.Equal(t, 1, loads)
}

func TestTakeResolvesFieldsOnAccess(t *testing.T) {
	s := Take(t.Context())
	p, ok := s.Procs[os.Getpid()]
	require.True(t, ok)
	require.NotEmpty(t, p.Name)
	require.NotEmpty(t, p.Cmdline)
	require.NotEmpty(t, p.ExePath())
	require.NotEmpty(t, p.Argv())
	// The owner lookup can legitimately be empty where the uid has no account
	// entry (a bare container), so compare only when the lookup works here.
	if u, err := user.Current(); err == nil {
		require.Equal(t, u.Username, p.User())
	}
}

func TestMatchesBin(t *testing.T) {
	for _, tc := range []struct {
		name, proc, exe, cmdline, bin string
		want                          bool
	}{
		{"exact name", "claude", "/usr/local/bin/claude", "claude --help", "claude", true},
		{"windows exe name", "claude.exe", `C:\bin\claude.exe`, "claude.exe", "claude", true},
		{"path token in command line", "node", "/usr/bin/node", "node /x/bin/claude --help", "claude", true},
		{"executable base name", "Amazon Q", "/opt/homebrew/bin/q", "Amazon Q", "q", true},
		{"name suffix", "myclaude", "/usr/bin/myclaude", "myclaude", "claude", false},
		{"short bin suffix", "icq", "/usr/bin/icq", "/usr/bin/icq", "q", false},
		{"short bin suffix, no path", "sq", "/bin/sq", "sq", "q", false},
		{"empty bin", "q", "/bin/q", "q", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := Process{Name: tc.proc, Exe: tc.exe, Cmdline: tc.cmdline}
			require.Equal(t, tc.want, p.MatchesBin(tc.bin))
		})
	}
}

// The executable is resolved only when the name and command line don't match,
// since resolving it costs a syscall per process.
func TestMatchesBinResolvesExeLast(t *testing.T) {
	resolved := false
	exe := func() string { resolved = true; return "/usr/local/bin/claude" }
	require.True(t, matchesBin("claude", exe, "/usr/local/bin/claude", "claude"))
	require.False(t, resolved, "name matched")
	require.True(t, matchesBin("node", exe, "/x/bin/claude --help", "claude"))
	require.False(t, resolved, "command line matched")
	require.True(t, matchesBin("node", exe, "node cli.js", "claude"))
	require.True(t, resolved)
}

// Deferred reads share one deadline set by Take, so however many of them hang,
// a snapshot stops reading at that deadline.
func TestDeferredReadsStopAtSnapshotDeadline(t *testing.T) {
	window := snapshotReadWindow
	snapshotReadWindow = 0
	t.Cleanup(func() { snapshotReadWindow = window })

	s := Take(t.Context())
	p, ok := s.Procs[os.Getpid()]
	require.True(t, ok)
	require.NotEmpty(t, p.Name, "read with the process list, not deferred")
	require.Empty(t, p.ExePath())
	require.Empty(t, p.User())
	require.Empty(t, p.Argv())
	require.Empty(t, s.Connections())
}
