package main

import (
	"fmt"
	"log"

	"fleetdm/gm/pkg/ghapi"
	"fleetdm/gm/pkg/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func main() {
	// Initialize logger
	if err := logger.Init(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	rootCmd := &cobra.Command{
		Use:   "gm",
		Short: "GitHub Manage CLI",
		Long:  "A CLI tool to manage GitHub repositories and workflows.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cmdSent := fmt.Sprintf("Command: %s Args: %v Flags: ", cmd.CommandPath(), args)
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				if flag.Changed {
					cmdSent = fmt.Sprintf("%s --%s: %s", cmdSent, flag.Name, flag.Value.String())
				}
			})
			logger.Debugf("%s", cmdSent)
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Placeholder for the default command behavior
			logger.Info("Welcome to GitHub Manage CLI!")
			cmd.Usage()
		},
	}

	rootCmd.AddCommand(issuesCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(estimatedCmd)
	rootCmd.AddCommand(sprintCmd)
	rootCmd.AddCommand(milestoneCmd)

	// Test command to test SetCurrentSprint functionality
	rootCmd.AddCommand(&cobra.Command{
		Use:   "test",
		Short: "Test SetCurrentSprint functionality",
		Run: func(cmd *cobra.Command, args []string) {
			// Useful to test new custom functions
			testIssueNumber := 31541
			ghapi.GetIssues(fmt.Sprintf("%d", testIssueNumber))
		},
	})

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
