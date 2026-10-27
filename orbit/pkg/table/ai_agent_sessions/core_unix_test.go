//go:build unix

package ai_agent_sessions

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFIFODoesNotBlock(t *testing.T) {
	home := setup(t)
	dir := filepath.Join(home, ".claude", "projects", "-tmp-proj")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "fifo.jsonl"), 0o644))
	require.NoError(t, syscall.Mkfifo(filepath.Join(home, ".claude.json"), 0o644))
	writeClaudePrompts(t, home, "real", "hi")

	done := make(chan []map[string]string, 1)
	go func() {
		rows, _ := GenerateSessions(t.Context(), qctx())
		done <- rows
	}()
	select {
	case rows := <-done:
		require.Len(t, rows, 1)
		require.Equal(t, "real", rows[0]["session_id"])
	case <-time.After(5 * time.Second):
		t.Fatal("generate blocked on a FIFO")
	}
}
