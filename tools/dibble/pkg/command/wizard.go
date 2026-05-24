package command

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/themes"
)

// runWizard drives the interactive no-arg flow. The cobra root command is
// passed in so the wizard can invoke `runAll` against the same client setup
// the subcommands use.
func runWizard(root *cobra.Command) error {
	// Cobra hasn't run yet (we bypassed Execute), so initialize viper manually.
	initConfig()

	fmt.Fprint(stdoutish, themes.TapirSmall)
	fmt.Fprintln(stdoutish, "Welcome to dibble. Let's plant some seeds.")

	// Step 1: resolve config (URL + token), prompting only for what's missing.
	urlAsked, tokenAsked, err := promptConfig()
	if err != nil {
		return err
	}

	// Step 2: offer to persist whatever was just typed in.
	if urlAsked || tokenAsked {
		save := true
		if err := survey.AskOne(
			&survey.Confirm{Message: fmt.Sprintf("Save these to %s?", configPath()), Default: true},
			&save,
		); err != nil {
			return err
		}
		if save {
			if err := writeConfigFile(); err != nil {
				warnf("could not save config: %v", err)
			} else {
				printf("config saved to %s", configPath())
			}
		}
	}

	// Step 3: ping. Fail fast so the user knows their config is wrong before
	// they spend time picking themes. If a TLS verification error trips us
	// up against a self-signed dev cert, offer to flip --insecure.
	client, err := newClientFromViper()
	if err != nil {
		return err
	}
	var ver struct {
		Version string `json:"version"`
	}
	pingErr := client.Get("/api/latest/fleet/version", &ver)
	if pingErr != nil && isTLSVerificationError(pingErr) && !viper.GetBool(keyInsecure) {
		retry := true
		if err := survey.AskOne(
			&survey.Confirm{Message: "TLS cert isn't trusted. Skip verification (insecure)?", Default: true},
			&retry,
		); err != nil {
			return err
		}
		if retry {
			viper.Set(keyInsecure, true)
			client, err = newClientFromViper()
			if err != nil {
				return err
			}
			pingErr = client.Get("/api/latest/fleet/version", &ver)
		}
	}
	if pingErr != nil {
		return fmt.Errorf("could not reach Fleet at %s: %w", viper.GetString(keyFleetURL), pingErr)
	}
	printf("connected to Fleet %s ✓", ver.Version)

	// Step 4: theme picker.
	chosenTheme := viper.GetString(keyTheme)
	if err := survey.AskOne(
		&survey.Select{
			Message: "Theme?",
			Options: themes.Names(),
			Default: chosenTheme,
		},
		&chosenTheme,
	); err != nil {
		return err
	}
	viper.Set(keyTheme, chosenTheme)
	theme, err := currentTheme()
	if err != nil {
		return err
	}

	// Step 5: entity multi-select.
	type entity struct {
		Label   string
		Key     string
		Slow    bool
		Default bool
	}
	entities := []entity{
		{"users", "users", false, true},
		{"teams", "teams", false, true},
		{"enroll-secrets", "enroll-secrets", false, true},
		{"labels", "labels", false, true},
		{"policies", "policies", false, true},
		{"reports (queries)", "reports", false, false},
		{"scripts", "scripts", false, false},
		{"profiles (Apple + Windows MDM)", "profiles", false, false},
		{"software (titles, no upload)", "software", false, false},
		{"vulns (direct MySQL, slow)", "vulns", true, false},
		{"cas (placeholder)", "cas", false, false},
	}
	labels := make([]string, len(entities))
	defaults := []string{}
	for i, e := range entities {
		labels[i] = e.Label
		if e.Slow {
			labels[i] += "  ⚠"
		}
		if e.Default {
			defaults = append(defaults, labels[i])
		}
	}
	var picked []string
	if err := survey.AskOne(
		&survey.MultiSelect{Message: "What would you like to seed?", Options: labels, Default: defaults},
		&picked,
	); err != nil {
		return err
	}
	wanted := map[string]bool{}
	for _, p := range picked {
		for i, e := range entities {
			if p == labels[i] {
				wanted[e.Key] = true
			}
		}
	}

	// Step 6: counts.
	counts := defaultAllCounts()
	customise := false
	if err := survey.AskOne(
		&survey.Confirm{Message: "Use default counts?", Default: true},
		&customise,
	); err != nil {
		return err
	}
	customise = !customise // confirm says "use defaults yes" → don't customise

	if customise {
		counts = promptCounts(wanted, counts)
	}

	// Step 7: run.
	return runWizardSelection(client, theme, wanted, counts)
}

// promptConfig prompts for any missing URL/token and updates viper. Returns
// whether each value was actually asked for (so we know whether to offer
// persisting).
func promptConfig() (bool, bool, error) {
	urlAsked, tokenAsked := false, false

	if viper.GetString(keyFleetURL) == "" {
		var u string
		err := survey.AskOne(
			&survey.Input{Message: "Fleet URL?", Default: "http://localhost:8080"},
			&u,
			survey.WithValidator(func(in any) error {
				s, _ := in.(string)
				if s == "" {
					return errors.New("required")
				}
				if _, err := url.Parse(s); err != nil {
					return err
				}
				return nil
			}),
		)
		if err != nil {
			return false, false, err
		}
		viper.Set(keyFleetURL, u)
		urlAsked = true
	}

	if viper.GetString(keyAPIToken) == "" {
		var t string
		err := survey.AskOne(
			&survey.Password{Message: "Fleet API token?"},
			&t,
			survey.WithValidator(func(in any) error {
				s, _ := in.(string)
				if strings.TrimSpace(s) == "" {
					return errors.New("required")
				}
				return nil
			}),
		)
		if err != nil {
			return false, false, err
		}
		viper.Set(keyAPIToken, t)
		tokenAsked = true
	}
	return urlAsked, tokenAsked, nil
}

// writeConfigFile persists only the keys dibble manages. Existing fields in
// the file are preserved on best-effort: we read first, then merge.
func writeConfigFile() error {
	path := configPath()
	existing := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}
	existing[keyFleetURL] = viper.GetString(keyFleetURL)
	existing[keyAPIToken] = viper.GetString(keyAPIToken)
	existing[keyTheme] = viper.GetString(keyTheme)
	if viper.GetBool(keyInsecure) {
		existing[keyInsecure] = true
	}
	data, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func promptCounts(wanted map[string]bool, defaults allCounts) allCounts {
	out := defaults
	ask := func(label string, dst *int) {
		var s string
		_ = survey.AskOne(
			&survey.Input{Message: label, Default: fmt.Sprintf("%d", *dst)},
			&s,
		)
		var v int
		_, _ = fmt.Sscanf(s, "%d", &v)
		if v > 0 {
			*dst = v
		}
	}
	if wanted["users"] {
		ask("users count", &out.Users)
	}
	if wanted["teams"] {
		ask("teams count", &out.Teams)
	}
	if wanted["policies"] {
		ask("policies count", &out.Policies)
	}
	if wanted["reports"] {
		ask("reports count", &out.Reports)
	}
	if wanted["labels"] {
		ask("labels count", &out.Labels)
	}
	if wanted["scripts"] {
		ask("scripts count", &out.Scripts)
	}
	if wanted["profiles"] {
		ask("profiles count", &out.Profiles)
	}
	if wanted["software"] {
		ask("software count", &out.Software)
	}
	if wanted["cas"] {
		ask("CAs count", &out.CAs)
	}
	return out
}

// isTLSVerificationError reports whether the error came from an untrusted
// TLS certificate — the common pain point with self-signed dev Fleets.
func isTLSVerificationError(err error) bool {
	if err == nil {
		return false
	}
	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "x509:") || strings.Contains(msg, "certificate signed by unknown authority")
}

func runWizardSelection(c *Client, theme themes.Theme, wanted map[string]bool, counts allCounts) error {
	// Zero out entities the user didn't pick — runAll uses count > 0 as
	// the implicit "do this" signal for everything except teams, which we
	// gate explicitly to keep downstream seeders sensible.
	if !wanted["users"] {
		counts.Users = 0
	}
	if !wanted["teams"] {
		counts.Teams = 0
	}
	if !wanted["policies"] {
		counts.Policies = 0
	}
	if !wanted["reports"] {
		counts.Reports = 0
	}
	if !wanted["labels"] {
		counts.Labels = 0
	}
	if !wanted["scripts"] {
		counts.Scripts = 0
	}
	if !wanted["profiles"] {
		counts.Profiles = 0
	}
	if !wanted["software"] {
		counts.Software = 0
	}
	if !wanted["cas"] {
		counts.CAs = 0
	}
	counts.EnrollSecrets = wanted["enroll-secrets"]
	return runAll(c, theme, counts)
}
