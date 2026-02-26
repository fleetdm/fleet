package main

import (
	"fmt"
	"strings"
)

type bridgePolicy struct {
	ChecklistByIssue  map[string]map[string]bool
	MilestonesByIssue map[string]map[int]bool
	SprintsByItemID   map[string]sprintApplyTarget
}

type sprintApplyTarget struct {
	ProjectID   string
	FieldID     string
	IterationID string
}

func buildBridgePolicy(drafting []DraftingCheckViolation, missing []MissingMilestoneIssue, missingSprints []MissingSprintViolation) bridgePolicy {
	p := bridgePolicy{
		ChecklistByIssue:  make(map[string]map[string]bool),
		MilestonesByIssue: make(map[string]map[int]bool),
		SprintsByItemID:   make(map[string]sprintApplyTarget),
	}

	for _, v := range drafting {
		owner, repo := parseRepoFromIssueURL(getURL(v.Item))
		if owner == "" || repo == "" {
			continue
		}
		key := issueKey(owner+"/"+repo, getNumber(v.Item))
		if p.ChecklistByIssue[key] == nil {
			p.ChecklistByIssue[key] = make(map[string]bool)
		}
		for _, text := range v.Unchecked {
			if text == "" {
				continue
			}
			p.ChecklistByIssue[key][text] = true
		}
	}

	for _, v := range missing {
		key := issueKey(v.RepoOwner+"/"+v.RepoName, getNumber(v.Item))
		if p.MilestonesByIssue[key] == nil {
			p.MilestonesByIssue[key] = make(map[int]bool)
		}
		for _, m := range v.SuggestedMilestones {
			if m.Number <= 0 {
				continue
			}
			p.MilestonesByIssue[key][m.Number] = true
		}
	}

	for _, v := range missingSprints {
		itemID := strings.TrimSpace(fmt.Sprintf("%v", v.ItemID))
		projectID := strings.TrimSpace(fmt.Sprintf("%v", v.ProjectID))
		fieldID := strings.TrimSpace(fmt.Sprintf("%v", v.SprintFieldID))
		iterationID := strings.TrimSpace(v.CurrentSprintID)
		if itemID == "" || projectID == "" || fieldID == "" || iterationID == "" {
			continue
		}
		p.SprintsByItemID[itemID] = sprintApplyTarget{
			ProjectID:   projectID,
			FieldID:     fieldID,
			IterationID: iterationID,
		}
	}

	return p
}
