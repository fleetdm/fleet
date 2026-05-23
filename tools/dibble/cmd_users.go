package main

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/seed"
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

func reportErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	for _, e := range errs {
		warnf("%v", e)
	}
	return errs[0]
}
