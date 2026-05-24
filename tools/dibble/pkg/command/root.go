package command

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/themes"
)

// Build-time variables (set via -ldflags).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Execute is the entry point used by cmd/dibble/main.go. It constructs the
// root cobra command, routes no-arg invocations through the wizard, and runs
// whichever path was chosen. Returns a non-nil error on failure.
func Execute(args []string) error {
	rootCmd := newRootCmd()
	if shouldRunWizard(args) {
		return runWizard(rootCmd)
	}
	return rootCmd.Execute()
}

// shouldRunWizard reports whether `dibble` was invoked with no subcommand,
// in which case we drop into the interactive wizard. Flags-only invocations
// (e.g. `dibble --fleet-url X`) still trigger the wizard so it can fill in
// any missing config.
func shouldRunWizard(args []string) bool {
	for _, a := range args {
		switch a {
		case "help", "--help", "-h", "completion", "--version":
			return false
		}
		// First non-flag positional → a subcommand was given.
		if len(a) > 0 && a[0] != '-' {
			return false
		}
		if a == "--no-wizard" {
			return false
		}
	}
	return true
}

// Config keys (viper).
const (
	keyFleetURL = "fleet_url"
	keyAPIToken = "api_token"
	keyTheme    = "theme"
	keyInsecure = "insecure"
	keySuffix   = "suffix"
)

// newRootCmd builds the root cobra command and wires viper.
//
// Configuration precedence (highest first):
//  1. Command-line flag
//  2. Environment variable (FLEET_URL, FLEET_API_TOKEN, DIBBLE_THEME)
//  3. ~/.dibble.yaml
//  4. Built-in defaults
func newRootCmd() *cobra.Command {
	cobra.OnInitialize(initConfig)

	root := &cobra.Command{
		Use:   "dibble",
		Short: "Fleet's seed slinger — plant test data into a Fleet server",
		Long: `dibble is the one-stop CLI for seeding test data into a Fleet server.
Run it with no arguments for an interactive wizard, or pass a subcommand to
script it.

Hosts are intentionally out of scope — use cmd/osquery-perf for that.`,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	// Persistent flags (apply to every subcommand).
	root.PersistentFlags().String("fleet-url", "", "Fleet server URL (env FLEET_URL)")
	root.PersistentFlags().String("api-token", "", "Fleet API token (env FLEET_API_TOKEN)")
	root.PersistentFlags().String("config", "", "Config file (default $HOME/.dibble.yaml)")
	root.PersistentFlags().String("theme", "mix", "Easter-egg theme: mix, hitchhikers, goodplace, parksrec, tng, lotr, dbz, robin_williams, ghibli, cosmere, sailor_moon")
	root.PersistentFlags().Bool("dry-run", false, "Print actions without calling the Fleet API")
	root.PersistentFlags().BoolP("verbose", "v", false, "Verbose logging")
	root.PersistentFlags().Bool("no-wizard", false, "Disable the interactive wizard (always error on missing config)")
	root.PersistentFlags().Bool("insecure", false, "Skip TLS certificate verification (for self-signed dev certs)")
	root.PersistentFlags().String("suffix", "", `Append to every generated name to avoid collisions on re-run. Use "auto" for a random 4-char suffix per run.`)

	// Bind to viper so env / config-file values flow through.
	_ = viper.BindPFlag(keyFleetURL, root.PersistentFlags().Lookup("fleet-url"))
	_ = viper.BindPFlag(keyAPIToken, root.PersistentFlags().Lookup("api-token"))
	_ = viper.BindPFlag(keyTheme, root.PersistentFlags().Lookup("theme"))
	_ = viper.BindPFlag(keyInsecure, root.PersistentFlags().Lookup("insecure"))
	_ = viper.BindPFlag(keySuffix, root.PersistentFlags().Lookup("suffix"))

	// Env mapping. FLEET_URL / FLEET_API_TOKEN feel natural to people coming
	// from fleetctl, even though the dibble prefix would be more consistent.
	viper.SetEnvPrefix("DIBBLE")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv(keyFleetURL, "FLEET_URL", "DIBBLE_FLEET_URL")
	_ = viper.BindEnv(keyAPIToken, "FLEET_API_TOKEN", "DIBBLE_API_TOKEN")

	// Subcommands.
	root.AddCommand(newVersionCmd())
	root.AddCommand(newPingCmd())
	root.AddCommand(newAllCmd())
	root.AddCommand(newUsersCmd())
	root.AddCommand(newTeamsCmd())
	root.AddCommand(newPoliciesCmd())
	root.AddCommand(newReportsCmd())
	root.AddCommand(newLabelsCmd())
	root.AddCommand(newScriptsCmd())
	root.AddCommand(newProfilesCmd())
	root.AddCommand(newSoftwareCmd())
	root.AddCommand(newEnrollSecretsCmd())
	root.AddCommand(newCAsCmd())
	root.AddCommand(newVulnsCmd())

	return root
}

func initConfig() {
	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".dibble")
		viper.SetConfigType("yaml")
	}
	_ = viper.ReadInConfig() // missing config file is fine — wizard will help.
}

// configPath returns the path used to persist wizard answers.
func configPath() string {
	if p := viper.ConfigFileUsed(); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".dibble.yaml"
	}
	return filepath.Join(home, ".dibble.yaml")
}

func requireConfig() error {
	if viper.GetString(keyFleetURL) == "" {
		return fmt.Errorf("missing --fleet-url (or FLEET_URL, or ~/.dibble.yaml). Run `dibble` with no args for the wizard")
	}
	if viper.GetString(keyAPIToken) == "" {
		return fmt.Errorf("missing --api-token (or FLEET_API_TOKEN, or ~/.dibble.yaml). Run `dibble` with no args for the wizard")
	}
	return nil
}

// currentTheme returns the theme selected by viper with the suffix applied.
// "auto" expands to a 4-character random hex suffix that's stable for the
// life of this process so every entity in the run shares the same tag.
func currentTheme() (themes.Theme, error) {
	t, err := themes.Get(viper.GetString(keyTheme))
	if err != nil {
		return themes.Theme{}, err
	}
	suffix := viper.GetString(keySuffix)
	if suffix == "auto" {
		suffix = autoSuffix()
	}
	t.Suffix = suffix
	return t, nil
}

var cachedAutoSuffix string

func autoSuffix() string {
	if cachedAutoSuffix != "" {
		return cachedAutoSuffix
	}
	var b [2]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		cachedAutoSuffix = "x"
		return cachedAutoSuffix
	}
	cachedAutoSuffix = hex.EncodeToString(b[:])
	return cachedAutoSuffix
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print dibble version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dibble %s  (commit %s, built %s)\n", version, commit, date)
			fmt.Println("— Dibble the Tapir 🌱")
		},
	}
}
