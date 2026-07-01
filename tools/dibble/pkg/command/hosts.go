package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newHostsCmd wires `dibble hosts` — a thin convenience over running
// cmd/osquery-perf. dibble doesn't seed hosts (osquery-perf does that), but
// looking up the right team's enroll secret and stringing the command
// together by hand is enough friction that this subcommand pays for itself.
//
// Flow:
//  1. List teams (and the global / no-team scope) from the Fleet API.
//  2. Radio-button pick — single-select via survey.
//  3. Fetch that team's enroll secret.
//  4. Build the `go run cmd/osquery-perf/agent.go -enroll_secret … -server_url …`
//     command, print it, and offer to run it inline. While the child runs,
//     Ctrl-C is forwarded to it so the user can shut it down cleanly.
func newHostsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hosts",
		Short: "Pick a team and run osquery-perf against it (single-select radio + inline run)",
		Long: `Lists existing teams (fleets), lets you pick one with a radio button, fetches
that team's enroll secret, and either prints the matching osquery-perf command
or runs it for you.

Picking the global / no-team option uses the global enroll secret from
/api/latest/fleet/spec/enroll_secret.

While osquery-perf runs inline, Ctrl-C is forwarded to it so you can stop the
simulated hosts cleanly.

dibble doesn't seed hosts itself — this is a convenience around osquery-perf,
which is the canonical host simulator (see cmd/osquery-perf).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}
			c, err := newClientFromViper()
			if err != nil {
				return err
			}

			teamName, _ := cmd.Flags().GetString("team-name")
			teamID, _ := cmd.Flags().GetInt("team-id")
			run, _ := cmd.Flags().GetBool("run")
			printOnly, _ := cmd.Flags().GetBool("print-only")
			hostCount, _ := cmd.Flags().GetInt("host-count")

			picked, err := pickHostsTeam(c, teamID, teamName)
			if err != nil {
				return err
			}

			secret, err := fetchEnrollSecret(c, picked)
			if err != nil {
				return err
			}

			repoRoot, err := findFleetRepoRoot()
			if err != nil {
				return err
			}

			fullCmd, displayCmd := buildOsqueryPerfCmd(repoRoot, secret, viper.GetString(keyFleetURL), hostCount)
			scope := picked.scopeDescription()
			printf("osquery-perf command for %s:", scope)
			fmt.Fprintln(stdoutish, "\n"+displayCmd+"\n")

			switch {
			case printOnly:
				return nil
			case run:
				return runOsqueryPerf(fullCmd)
			default:
				return maybeRunOsqueryPerf(fullCmd)
			}
		},
	}
	cmd.Flags().Int("team-id", -1, "Team to enroll into. -1 (default) = interactive pick. 0 = no team / global.")
	cmd.Flags().String("team-name", "", "Team to enroll into, by name. Overrides --team-id.")
	cmd.Flags().Int("host-count", 1, "Number of simulated hosts (passed to osquery-perf -host_count)")
	cmd.Flags().Bool("run", false, "Skip the confirm prompt and run osquery-perf immediately")
	cmd.Flags().Bool("print-only", false, "Skip the confirm prompt and just print the command")
	return cmd
}

// pickedTeam is the outcome of the team picker. ID == 0 means the global /
// no-team scope.
type pickedTeam struct {
	ID   uint
	Name string
}

func (p pickedTeam) scopeDescription() string {
	if p.ID == 0 {
		return "no team (global)"
	}
	return fmt.Sprintf("team %q (id=%d)", p.Name, p.ID)
}

// pickHostsTeam resolves which team to enroll into. Precedence mirrors
// resolveSoftwareTeamID in software.go:
//
//  1. --team-name (looked up against Fleet)
//  2. --team-id (0 = global, ≥1 = that team)
//  3. Interactive single-select via survey
func pickHostsTeam(c *Client, flagID int, flagName string) (pickedTeam, error) {
	teams, err := listExistingTeams(c)
	if err != nil {
		return pickedTeam{}, fmt.Errorf("list teams: %w", err)
	}

	if flagName != "" {
		for _, t := range teams {
			if strings.EqualFold(t.Name, flagName) {
				return pickedTeam{ID: t.ID, Name: t.Name}, nil
			}
		}
		return pickedTeam{}, fmt.Errorf("no team named %q (try --team-id or run `dibble teams` first)", flagName)
	}
	if flagID == 0 {
		return pickedTeam{ID: 0, Name: ""}, nil
	}
	if flagID > 0 {
		for _, t := range teams {
			if t.ID == uint(flagID) {
				return pickedTeam{ID: t.ID, Name: t.Name}, nil
			}
		}
		// Honor explicit ID even if we couldn't resolve a name.
		return pickedTeam{ID: uint(flagID), Name: ""}, nil
	}

	options := []string{"(no team — global)"}
	for _, t := range teams {
		options = append(options, fmt.Sprintf("%s (id=%d)", t.Name, t.ID))
	}
	def := options[0]
	if len(options) > 1 {
		def = options[1]
	}
	var picked string
	if err := survey.AskOne(&survey.Select{
		Message: "Which team should the simulated hosts enroll into?",
		Options: options,
		Default: def,
	}, &picked); err != nil {
		return pickedTeam{}, err
	}
	if picked == options[0] {
		return pickedTeam{ID: 0, Name: ""}, nil
	}
	for _, t := range teams {
		if picked == fmt.Sprintf("%s (id=%d)", t.Name, t.ID) {
			return pickedTeam{ID: t.ID, Name: t.Name}, nil
		}
	}
	return pickedTeam{}, errors.New("no team selected")
}

// fetchEnrollSecret returns the first secret for the given team, or the
// first global secret when picked.ID == 0.
func fetchEnrollSecret(c *Client, picked pickedTeam) (string, error) {
	if picked.ID == 0 {
		var resp struct {
			Spec struct {
				Secrets []struct {
					Secret string `json:"secret"`
				} `json:"secrets"`
			} `json:"spec"`
		}
		if err := c.Get("/api/latest/fleet/spec/enroll_secret", &resp); err != nil {
			return "", fmt.Errorf("fetch global enroll secret: %w", err)
		}
		if len(resp.Spec.Secrets) == 0 {
			return "", errors.New("global enroll secret is empty — set one in Fleet settings or run `dibble enroll-secrets`")
		}
		return resp.Spec.Secrets[0].Secret, nil
	}

	var resp struct {
		Secrets []struct {
			Secret string `json:"secret"`
		} `json:"secrets"`
	}
	if err := c.Get(fmt.Sprintf("/api/latest/fleet/fleets/%d/secrets", picked.ID), &resp); err != nil {
		return "", fmt.Errorf("fetch enroll secret for team %d: %w", picked.ID, err)
	}
	if len(resp.Secrets) == 0 {
		return "", fmt.Errorf("team %d has no enroll secret — run `dibble enroll-secrets` to add one", picked.ID)
	}
	return resp.Secrets[0].Secret, nil
}

// findFleetRepoRoot walks up from the current working directory until it
// finds the parent fleet checkout (the one with cmd/osquery-perf/agent.go).
// Returned path is absolute.
func findFleetRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "cmd", "osquery-perf", "agent.go")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find fleet repo root (cmd/osquery-perf/agent.go) walking up from %s", cwd)
		}
		dir = parent
	}
}

// buildOsqueryPerfCmd assembles the slice of arguments passed to exec.Cmd
// alongside a human-readable rendering for printing back to the user.
func buildOsqueryPerfCmd(repoRoot, secret, serverURL string, hostCount int) (full []string, display string) {
	if serverURL == "" {
		serverURL = "https://localhost:8080"
	}
	if hostCount < 1 {
		hostCount = 1
	}
	agentPath := filepath.Join(repoRoot, "cmd", "osquery-perf", "agent.go")
	full = []string{
		"go", "run", agentPath,
		"-enroll_secret", secret,
		"-server_url", serverURL,
		"-host_count", fmt.Sprintf("%d", hostCount),
	}

	// Quote the secret for display so copy-paste works in shells.
	display = fmt.Sprintf("go run %s -enroll_secret %q -server_url %s -host_count %d",
		agentPath, secret, serverURL, hostCount)
	return full, display
}

// maybeRunOsqueryPerf prompts the user, and if confirmed runs osquery-perf
// inline with Ctrl-C forwarded. Non-interactive sessions (no TTY, survey
// returns error) default to printing only.
func maybeRunOsqueryPerf(args []string) error {
	wantRun := true
	if err := survey.AskOne(&survey.Confirm{
		Message: "Run osquery-perf now? (Ctrl-C to stop)",
		Default: true,
	}, &wantRun); err != nil {
		// Non-interactive — fall back to print-only behavior.
		printf("not running osquery-perf (interactive prompt failed: %v)", err)
		return nil
	}
	if !wantRun {
		return nil
	}
	return runOsqueryPerf(args)
}

// runOsqueryPerf executes the command with stdout / stderr passthrough and
// forwards SIGINT / SIGTERM to the child so Ctrl-C stops the simulated
// hosts cleanly. Returns whatever exit error the child raises, except that
// a signal-initiated shutdown is treated as success.
//
// Used to use exec.CommandContext + signal.Notify with manual Process.Signal
// calls so the child gets the signal directly rather than the cancel-style
// kill exec.CommandContext does by default.
func runOsqueryPerf(args []string) error {
	if len(args) < 2 {
		return errors.New("runOsqueryPerf: empty command")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Disable the default ctx-cancel kill so we control shutdown ourselves.
	// (Go 1.20+ exec wires Cancel to SIGKILL; we want SIGINT so osquery-perf
	// can run its own deferred cleanup.)
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return cmd.Process.Signal(syscall.SIGINT)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start osquery-perf: %w", err)
	}

	// Forward the first signal we receive; further signals from the user
	// keep getting forwarded so a hung child can be killed harder by
	// repeated Ctrl-C.
	go func() {
		for sig := range sigCh {
			if cmd.Process == nil {
				return
			}
			_ = cmd.Process.Signal(sig)
		}
	}()

	err := cmd.Wait()
	// A SIGINT-induced exit looks like an error to exec.Wait; treat it as
	// success when we asked the child to stop.
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() && (status.Signal() == syscall.SIGINT || status.Signal() == syscall.SIGTERM) {
					return nil
				}
			}
		}
		return err
	}
	return nil
}

