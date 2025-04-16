package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func generateGitopsCommand() *cli.Command {
	return &cli.Command{
		Name:        "generate-gitops",
		Usage:       "Generate GitOps configuration files for Fleet.",
		Description: "This command generates GitOps configuration files for Fleet.",
		Action:      generateGitopsAction,
	}
}

func generateGitopsAction(c *cli.Context) error {
	fmt.Println("Generating GitOps configuration files...")
	return nil
}
