package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

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
		fmt.Fprintf(os.Stderr, "fleet ship: %v\n", err)
		os.Exit(1)
	}
	_, keyExists, err := LoadPrivateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fleet ship: %v\n", err)
		os.Exit(1)
	}

	// Wizard runs on first launch (no config file yet), when the MDM key file
	// is missing (user lost it and needs to paste from 1Password again), or
	// when the user explicitly asked for it.
	needWiz := !cfgExists || !keyExists || *reconfigure

	p := tea.NewProgram(
		initialModel(cfg, keyExists, needWiz),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fleet ship exited with error: %v\n", err)
		os.Exit(1)
	}
}
