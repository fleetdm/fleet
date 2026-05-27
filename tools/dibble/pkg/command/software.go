package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

// newSoftwareCmd wires `dibble software` and its subcommands. The seeder
// uploads the curated installer fixtures and registers a few Fleet-maintained
// apps. Both paths target one team — the first existing team by default, or
// whichever team --team-id / --team-name selects.
func newSoftwareCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "software",
		Short: "Seed software (custom installers + Fleet-maintained apps)",
		Long: `Seed real software into Fleet for testing. Pick a subcommand:

  dibble software all          — upload custom installers AND add maintained apps
  dibble software custom       — upload curated installers (2-3 per extension)
  dibble software maintained   — add Fleet-maintained apps from the server's catalog

Custom uploads cover .pkg / .deb / .msi / .rpm / .tar.gz / .ipa fixtures bundled
into the dibble binary. Maintained-app entries are read from
/api/latest/fleet/software/fleet_maintained_apps and POSTed in catalog order.

All work scopes to a single team. By default dibble picks the first team Fleet
returns; override with --team-id or --team-name. Use --team-id 0 for the
no-team / global scope.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}

	addCommonFlags := func(cmd *cobra.Command) {
		cmd.Flags().Int("team-id", -1, "Team to upload software to. -1 (default) = first team Fleet returns. 0 = no team / global.")
		cmd.Flags().String("team-name", "", "Team to upload software to, by name. Overrides --team-id if set.")
		cmd.Flags().Int("maintained-count", 3, "How many Fleet-maintained apps to add (used by `all` and `maintained`)")
	}

	mkSub := func(use, short string, runFn func(client *Client, opt seed.SoftwareOptions) error) *cobra.Command {
		sub := &cobra.Command{
			Use:   use,
			Short: short,
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := requireConfig(); err != nil {
					return err
				}
				c, err := newClientFromViper()
				if err != nil {
					return err
				}
				teamID, err := resolveSoftwareTeamID(cmd, c)
				if err != nil {
					return err
				}
				maintained, _ := cmd.Flags().GetInt("maintained-count")
				return runFn(c, seed.SoftwareOptions{
					TeamID:             teamID,
					MaintainedAppCount: maintained,
				})
			},
		}
		addCommonFlags(sub)
		return sub
	}

	root.AddCommand(mkSub("custom", "Upload curated installer fixtures (2-3 per extension)",
		func(c *Client, opt seed.SoftwareOptions) error {
			res := seed.SoftwareCustom(c, seederLogger{}, opt)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		}))

	root.AddCommand(mkSub("maintained", "Add Fleet-maintained apps from the server's catalog",
		func(c *Client, opt seed.SoftwareOptions) error {
			if opt.MaintainedAppCount <= 0 {
				opt.MaintainedAppCount = 3
			}
			res := seed.SoftwareMaintained(c, seederLogger{}, opt)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		}))

	root.AddCommand(mkSub("all", "Upload custom installers AND add maintained apps",
		func(c *Client, opt seed.SoftwareOptions) error {
			if opt.MaintainedAppCount <= 0 {
				opt.MaintainedAppCount = 3
			}
			cRes := seed.SoftwareCustom(c, seederLogger{}, opt)
			printf("%s", cRes.Summary())
			mRes := seed.SoftwareMaintained(c, seederLogger{}, opt)
			printf("%s", mRes.Summary())
			combined := append([]error{}, cRes.Errors...)
			combined = append(combined, mRes.Errors...)
			return reportErrors(combined)
		}))

	return root
}

// resolveSoftwareTeamID determines which team uploads land under. Precedence:
//
//  1. --team-name (looked up against Fleet)
//  2. --team-id (0 means global, -1 means "auto-pick")
//  3. Auto-pick: first team Fleet returns. If none exist, prompt
//     interactively when a TTY is attached; otherwise default to 0 / global.
func resolveSoftwareTeamID(cmd *cobra.Command, c *Client) (uint, error) {
	if name, _ := cmd.Flags().GetString("team-name"); name != "" {
		teams, err := listExistingTeams(c)
		if err != nil {
			return 0, fmt.Errorf("list teams to resolve --team-name=%q: %w", name, err)
		}
		for _, t := range teams {
			if strings.EqualFold(t.Name, name) {
				return t.ID, nil
			}
		}
		return 0, fmt.Errorf("no team named %q (try --team-id or run `dibble teams` first)", name)
	}

	explicit, _ := cmd.Flags().GetInt("team-id")
	if explicit == 0 {
		return 0, nil
	}
	if explicit > 0 {
		return uint(explicit), nil
	}

	// Auto-pick: first team Fleet returns. The wizard's software step (or
	// --team-id / --team-name) is the path for picking a specific team.
	teams, err := listExistingTeams(c)
	if err != nil {
		return 0, fmt.Errorf("list teams for auto-pick: %w", err)
	}
	if len(teams) == 0 {
		printf("no teams found — falling back to no-team / global scope")
		return 0, nil
	}
	printf("software: auto-picked team %q (id=%d) — override with --team-id or --team-name", teams[0].Name, teams[0].ID)
	return teams[0].ID, nil
}
