package main

import (
	"context"
	"sort"
	"strings"

	"github.com/shurcooL/githubv4"
)

// runProductBoardMilestoneCheck finds project-67 issue items that match at
// least one provided group label and still have any milestone set.
func runProductBoardMilestoneCheck(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	limit int,
	groupLabelFilter map[string]struct{},
) []ProductBoardMilestoneViolation {
	if len(groupLabelFilter) == 0 {
		return nil
	}

	projectID := fetchProjectID(ctx, client, org, draftingProjectNum)
	items := fetchItems(ctx, client, projectID, limit)
	statusNeedles := strings.Split(productBoardMilestoneStatusNeedle, ",")
	out := make([]ProductBoardMilestoneViolation, 0)
	for _, it := range items {
		if it.Content.Issue.Number == 0 {
			continue
		}
		if _, ok := matchedStatus(it, statusNeedles); !ok {
			continue
		}
		if !matchesLabelFilter(it, groupLabelFilter) {
			continue
		}
		milestone := strings.TrimSpace(string(it.Content.Issue.Milestone.Title))
		if milestone == "" {
			continue
		}
		owner, repo := parseRepoFromIssueURL(getURL(it))
		if owner == "" || repo == "" {
			continue
		}
		out = append(out, ProductBoardMilestoneViolation{
			Item:       it,
			ProjectNum: draftingProjectNum,
			RepoOwner:  owner,
			RepoName:   repo,
			Milestone:  milestone,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return getNumber(out[i].Item) < getNumber(out[j].Item)
	})
	return out
}
