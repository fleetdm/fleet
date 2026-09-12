package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

var choreCmd = &cobra.Command{
	Use:   "chore",
	Short: "Board maintenance chores (rank, merge-size, ...)",
}

var mergeSizeCmd = &cobra.Command{
	Use:   "merge-size [project-id-or-alias]",
	Short: "Sync issue sizes between a project and release planning",
	Long: `Goes issue by issue in the selected project and reconciles its size field with
the release planning project: a size set on only one side is copied to the
other, and when the two sides disagree you are asked which value to keep.
Issues present in only one of the two projects are skipped.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID, err := ghapi.ResolveProjectID(args[0])
		if err != nil {
			fmt.Printf("Error resolving project: %v\n", err)
			return
		}
		releaseArg, _ := cmd.Flags().GetString("release-project")
		releaseID, err := ghapi.ResolveProjectID(releaseArg)
		if err != nil {
			fmt.Printf("Error resolving release planning project: %v\n", err)
			return
		}
		if releaseID == projectID {
			fmt.Println("Selected project is the release planning project; nothing to merge.")
			return
		}
		limit, _ := cmd.Flags().GetInt("limit")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		local, localTotal, err := ghapi.GetProjectItemsWithTotal(projectID, limit)
		if err != nil {
			fmt.Printf("Error fetching project %d items: %v\n", projectID, err)
			return
		}
		remote, remoteTotal, err := ghapi.GetProjectItemsWithTotal(releaseID, limit)
		if err != nil {
			fmt.Printf("Error fetching release planning items: %v\n", err)
			return
		}
		if localTotal > len(local) || remoteTotal > len(remote) {
			fmt.Printf("Warning: fetched %d/%d project items and %d/%d release planning items; raise --limit to cover everything.\n",
				len(local), localTotal, len(remote), remoteTotal)
		}

		plan := ghapi.PlanSizeSync(local, remote)
		fmt.Printf("Matched issues needing changes: %d to set here, %d to set in release planning, %d conflicts.\n",
			len(plan.SetLocal), len(plan.SetRemote), len(plan.Conflicts))

		if dryRun {
			for _, p := range plan.SetLocal {
				fmt.Printf("  would set #%d here to %s (%s)\n", p.Local.Content.Number, p.Remote.SizeValue(), p.Local.Title)
			}
			for _, p := range plan.SetRemote {
				fmt.Printf("  would set #%d in release planning to %s (%s)\n", p.Local.Content.Number, p.Local.SizeValue(), p.Local.Title)
			}
			for _, p := range plan.Conflicts {
				fmt.Printf("  conflict on #%d: here=%s, release planning=%s (%s)\n",
					p.Local.Content.Number, p.Local.SizeValue(), p.Remote.SizeValue(), p.Local.Title)
			}
			return
		}

		applied, failed := 0, 0
		apply := func(targetProject int, itemID string, number int, value, where string) {
			if err := ghapi.SetItemSize(targetProject, itemID, value); err != nil {
				fmt.Printf("  error setting #%d %s to %s: %v\n", number, where, value, err)
				failed++
				return
			}
			fmt.Printf("  set #%d %s to %s\n", number, where, value)
			applied++
		}

		for _, p := range plan.SetLocal {
			apply(projectID, p.Local.ID, p.Local.Content.Number, p.Remote.SizeValue(), "here")
		}
		for _, p := range plan.SetRemote {
			apply(releaseID, p.Remote.ID, p.Local.Content.Number, p.Local.SizeValue(), "in release planning")
		}

		reader := bufio.NewReader(os.Stdin)
		for _, p := range plan.Conflicts {
			ls, rs := p.Local.SizeValue(), p.Remote.SizeValue()
			fmt.Printf("\n#%d %s\n  [1] keep %s (this project, updates release planning)\n  [2] keep %s (release planning, updates this project)\n  [s] skip\n> ",
				p.Local.Content.Number, p.Local.Title, ls, rs)
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("\nInput closed, skipping remaining conflicts.")
				break
			}
			switch strings.TrimSpace(line) {
			case "1":
				apply(releaseID, p.Remote.ID, p.Local.Content.Number, ls, "in release planning")
			case "2":
				apply(projectID, p.Local.ID, p.Local.Content.Number, rs, "here")
			default:
				fmt.Println("  skipped")
			}
		}

		fmt.Printf("\nDone. %d updated", applied)
		if failed > 0 {
			fmt.Printf(", %d failed", failed)
		}
		fmt.Println(".")
	},
}

func init() {
	mergeSizeCmd.Flags().IntP("limit", "l", 1000, "Maximum number of items to fetch from each project")
	mergeSizeCmd.Flags().StringP("release-project", "r", "releases", "Release planning project ID or alias")
	mergeSizeCmd.Flags().BoolP("dry-run", "n", false, "Report what would change without writing")

	choreCmd.AddCommand(rankCmd)
	choreCmd.AddCommand(mergeSizeCmd)
}
