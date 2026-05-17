package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "Fleet ship is currently macOS-only.")
		fmt.Fprintln(os.Stderr, "For Linux/Windows, follow docs/Contributing/getting-started/building-fleet.md.")
		os.Exit(1)
	}

	reconfigure := flag.Bool("reconfigure", false, "Re-run the first-time setup wizard.")
	flag.Parse()

	cfg, cfgExists, err := LoadConfig()
	if err != nil {
		fail(err)
	}
	priv, keyExists, err := LoadPrivateKey()
	if err != nil {
		fail(err)
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		fail(err)
	}

	// Wizard runs on first launch (no config file yet), when the server
	// private key file is missing (user lost it and needs to paste from
	// 1Password again), or when the user explicitly asked for it.
	needWiz := !cfgExists || !keyExists || *reconfigure

	p := tea.NewProgram(
		initialModel(cfg, priv, keyExists, needWiz, repoRoot),
		tea.WithAltScreen(),
	)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fleet ship exited with error: %v\n", err)
		os.Exit(1)
	}

	// Tear down the engine after the TUI has released the terminal, so the
	// user sees a clean prompt with shutdown progress instead of a frozen
	// alt-screen while docker-compose-down runs.
	if final, ok := finalModel.(model); ok && final.eng != nil {
		fmt.Fprintln(os.Stderr, "Shutting down Fleet ship...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		final.eng.Stop(ctx)
	}
	_ = ClearActiveSession()
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "fleet ship: %v\n", err)
	os.Exit(1)
}
