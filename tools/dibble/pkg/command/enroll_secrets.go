package command

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

func newEnrollSecretsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "enroll-secrets",
		Aliases: []string{"enrollsecrets"},
		Short:   "Seed per-team enroll secrets (the credential fleetd uses to join a team)",
		Long: `Seeds an enroll secret for every team. These are the credentials a fleetd
agent presents to register with a team — NOT the "Fleet secrets" (a.k.a.
secret variables) used as template variables in profiles and scripts.

Find seeded values in the UI under Settings → [team] → Add hosts → Show enroll secret.
The global enroll secret is intentionally left alone.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}
			c, err := newClientFromViper()
			if err != nil {
				return err
			}
			teams, _ := listExistingTeams(c)
			res := seed.EnrollSecrets(c, seederLogger{}, teams)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
}
