package command

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

// categoryHelp tells reviewers what each subcommand actually plants. Keep
// the keys in sync with seed.ActivityCategories.
var categoryHelp = map[string]string{
	seed.CategorySettings:     "global config / feature toggles (agent options, GitOps mode, MDM enable/disable, disk encryption defaults, OS minimum versions, VPP, org logo, enroll secrets, conditional access integrations).",
	seed.CategoryProfiles:     "configuration profile lifecycle (macOS / Windows / declaration / Android), resent profiles, bootstrap packages, setup assistant, enrollment profile renewal failures.",
	seed.CategoryScripts:      "script management and execution: ran, added, updated, deleted, edited, canceled, plus batch script scheduled / canceled.",
	seed.CategorySoftware:     "software and app store apps lifecycle: install, uninstall, add, edit, delete, canceled installs, setup experience software.",
	seed.CategoryHosts:        "per-host actions: lock / unlock / wipe, enroll / unenroll, disk encryption key access, recovery lock, managed local accounts, cleared passcode, conditional access bypass.",
	seed.CategoryUsers:        "auth and user management: SSO, login, failed login, create / delete user, global and team role changes.",
	seed.CategoryTeams:        "team (fleet) lifecycle: created, deleted, applied spec, transferred hosts.",
	seed.CategoryPolicies:     "policy CRUD plus applied_spec_policy.",
	seed.CategoryQueries:      "saved + live queries (and legacy packs): created, edited, deleted, applied spec, live query.",
	seed.CategoryLabels:       "label CRUD: created, edited, deleted.",
	seed.CategoryCertificates: "certificate authorities and proxies: NDES, custom SCEP, DigiCert, Hydrant, custom EST, Smallstep, plus add / delete / install / resend certificate.",
}

// newActivitiesCmd wires `dibble activities` and its per-category
// subcommands. The seeder writes directly to MySQL (activity_past +
// activity_host_past), bypassing the service layer. It is intentionally
// non-idempotent: every invocation inserts a fresh batch tagged with the
// current run id so seeded rows are easy to spot in the UI.
//
// Subcommands:
//
//	dibble activities all          # one row per activity type (~161)
//	dibble activities settings     # global config / feature toggle rows
//	dibble activities profiles     # profile CRUD + renewal failures
//	dibble activities scripts      # script management & execution
//	dibble activities software     # software / app store apps lifecycle
//	dibble activities hosts        # per-host actions (lock, wipe, enroll, ...)
//	dibble activities users        # auth + user management
//	dibble activities teams        # team / fleet lifecycle
//	dibble activities policies     # policy CRUD
//	dibble activities queries      # saved + live queries (and packs)
//	dibble activities labels       # label CRUD
//	dibble activities certificates # cert proxies and CAs
//
// Invoking `dibble activities` with no subcommand prints the help, mirroring
// the convention from `kubectl get` etc.
func newActivitiesCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "activities",
		Short: "Seed fake activities directly into MySQL (non-idempotent)",
		Long: `Activities aren't seed-able via the Fleet API — they're written by
NewActivity inside the service layer as a side effect of every state-changing
endpoint. dibble shortcuts that by writing rows directly to MySQL, one per
activity type per batch.

Every name-like value (team names, host names, software titles, profiles,
labels, users, scripts, etc.) is prefixed with "*" so faked rows are obvious
in the activity feed and host activity card. Each run also stamps a unique
tag onto names so re-running dibble produces fresh, distinguishable entries.

Requires direct access to the Fleet MySQL instance — the default DSN matches
the local docker-compose dev environment.`,
		// Show subcommand help when invoked bare.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	addCommonFlags := func(cmd *cobra.Command) {
		cmd.Flags().String("dsn", "fleet:insecure@tcp(localhost:3306)/fleet", "MySQL DSN")
		cmd.Flags().Uint("actor-id", 1, "user_id stamped on inserted rows (must exist in users)")
		cmd.Flags().String("actor-name", "*Dibble Admin", "user_name stamped on inserted rows")
		cmd.Flags().String("actor-email", "*admin@example.com", "user_email stamped on inserted rows")
		cmd.Flags().Uint("host-id", 1, "host_id used for host-scoped activities and activity_host_past links")
		cmd.Flags().Int("batches", 1, "Number of full passes to insert (each pass writes one row per template with a fresh run tag)")
	}

	runCategory := func(category string) func(cmd *cobra.Command, args []string) error {
		return func(cmd *cobra.Command, args []string) error {
			dsn, _ := cmd.Flags().GetString("dsn")
			actorID, _ := cmd.Flags().GetUint("actor-id")
			actorName, _ := cmd.Flags().GetString("actor-name")
			actorEmail, _ := cmd.Flags().GetString("actor-email")
			hostID, _ := cmd.Flags().GetUint("host-id")
			batches, _ := cmd.Flags().GetInt("batches")

			res := seed.Activities(context.Background(), seederLogger{}, seed.ActivitiesOptions{
				DSN:        dsn,
				ActorID:    actorID,
				ActorName:  actorName,
				ActorEmail: actorEmail,
				HostID:     hostID,
				Batches:    batches,
				Category:   category,
			})
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		}
	}

	// `all` is the equivalent of the original `dibble activities` command.
	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Seed one row of every activity type (~161 rows per batch)",
		RunE:  runCategory(seed.CategoryAll),
	}
	addCommonFlags(allCmd)
	root.AddCommand(allCmd)

	for _, category := range seed.ActivityCategories {
		category := category // capture
		sub := &cobra.Command{
			Use:   category,
			Short: fmt.Sprintf("Seed %s-related activities", category),
			Long:  fmt.Sprintf("Seed %s-related activities:\n\n  %s", category, categoryHelp[category]),
			RunE:  runCategory(category),
		}
		addCommonFlags(sub)
		root.AddCommand(sub)
	}

	return root
}
