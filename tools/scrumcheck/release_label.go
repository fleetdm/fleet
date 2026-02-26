package main

import (
	"context"
	"sort"
	"strings"

	"github.com/shurcooL/githubv4"
)

const (
	productLabel       = ":product"
	releaseLabel       = ":release"
	unreleasedBugLabel = "unreleased bug"
)

func runReleaseLabelChecks(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	projectNums []int,
	limit int,
) []ReleaseLabelIssue {
	out := make([]ReleaseLabelIssue, 0)
	for _, projectNum := range projectNums {
		if projectNum == draftingProjectNum {
			continue
		}
		projectID := fetchProjectID(ctx, client, org, projectNum)
		items := fetchItems(ctx, client, projectID, limit)
		for _, it := range items {
			if it.Content.Issue.Number == 0 {
				continue
			}
			labels := issueLabels(it)
			hasProduct := labelsContain(labels, productLabel)
			hasRelease := labelsContain(labels, releaseLabel)
			if inDoneColumn(it) && !hasProduct {
				continue
			}
			if !hasProduct && hasRelease {
				continue
			}
			owner, repo := parseRepoFromIssueURL(getURL(it))
			if owner == "" || repo == "" {
				continue
			}
			out = append(out, ReleaseLabelIssue{
				Item:          it,
				ProjectNum:    projectNum,
				RepoOwner:     owner,
				RepoName:      repo,
				HasProduct:    hasProduct,
				HasRelease:    hasRelease,
				CurrentLabels: labels,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ProjectNum != out[j].ProjectNum {
			return out[i].ProjectNum < out[j].ProjectNum
		}
		return getNumber(out[i].Item) < getNumber(out[j].Item)
	})
	return out
}

func issueLabels(it Item) []string {
	out := make([]string, 0, len(it.Content.Issue.Labels.Nodes))
	seen := make(map[string]bool)
	for _, n := range it.Content.Issue.Labels.Nodes {
		name := strings.TrimSpace(string(n.Name))
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, name)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func labelsContain(labels []string, wanted string) bool {
	wantedNorm := normalizeLabelName(wanted)
	for _, l := range labels {
		if normalizeLabelName(l) == wantedNorm {
			return true
		}
	}
	return false
}
