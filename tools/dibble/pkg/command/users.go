package command

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

func newUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Seed Fleet users with themed names and rotating roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}
			c, err := newClientFromViper()
			if err != nil {
				return err
			}
			theme, err := currentTheme()
			if err != nil {
				return err
			}
			count, _ := cmd.Flags().GetInt("count")
			res := seed.Users(c, seederLogger{}, theme, count)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().Int("count", 5, "How many users to seed")
	return cmd
}

// ErrSeederFailed is returned by reportErrors when one or more seeders
// produced errors. The errors have already been written to stderr by
// reportErrors, so main.go skips re-printing them and just exits non-zero.
var ErrSeederFailed = errors.New("dibble: one or more seeders had errors")

func reportErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	for _, e := range errs {
		warnf("%v", e)
	}
	return ErrSeederFailed
}
