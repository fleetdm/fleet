package main

import (
	"time"

	"github.com/shurcooL/githubv4"
)

type Item struct {
	ID githubv4.ID
	// UpdatedAt is the Project item timestamp (used for stale Awaiting QA detection).
	UpdatedAt githubv4.DateTime

	Content struct {
		Issue struct {
			Number    githubv4.Int
			Title     githubv4.String
			Body      githubv4.String
			URL       githubv4.URI
			Assignees struct {
				Nodes []struct {
					Login githubv4.String
				}
			} `graphql:"assignees(first: 30)"`
			Labels struct {
				Nodes []struct {
					Name githubv4.String
				}
			} `graphql:"labels(first: 100)"`
			Milestone struct {
				Title githubv4.String
			} `graphql:"milestone"`
		} `graphql:"... on Issue"`

		PullRequest struct {
			Number githubv4.Int
			Title  githubv4.String
			Body   githubv4.String
			URL    githubv4.URI
		} `graphql:"... on PullRequest"`
	} `graphql:"content"`

	FieldValues struct {
		Nodes []struct {
			SingleSelectValue struct {
				Name  githubv4.String
				Field struct {
					Common struct {
						ID   githubv4.ID
						Name githubv4.String
					} `graphql:"... on ProjectV2FieldCommon"`
				} `graphql:"field"`
			} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
			IterationValue struct {
				IterationID githubv4.String `graphql:"iterationId"`
				Title       githubv4.String `graphql:"title"`
				Field       struct {
					Common struct {
						ID   githubv4.ID
						Name githubv4.String
					} `graphql:"... on ProjectV2FieldCommon"`
				} `graphql:"field"`
			} `graphql:"... on ProjectV2ItemFieldIterationValue"`
		}
	} `graphql:"fieldValues(first: 20)"`
}

type DraftingCheckViolation struct {
	Item      Item
	Unchecked []string
	Status    string
}

type StaleAwaitingViolation struct {
	Item        Item
	StaleDays   int
	LastUpdated time.Time
	ProjectNum  int
}

type MissingMilestoneIssue struct {
	Item                Item
	ProjectNum          int
	RepoOwner           string
	RepoName            string
	SuggestedMilestones []MilestoneOption
}

type MilestoneOption struct {
	Number int
	Title  string
}

type MissingAssigneeIssue struct {
	Item               Item
	ProjectNum         int
	RepoOwner          string
	RepoName           string
	CurrentAssignees   []string
	SuggestedAssignees []AssigneeOption
}

type AssigneeOption struct {
	Login string
}

type ReleaseLabelIssue struct {
	Item          Item
	ProjectNum    int
	RepoOwner     string
	RepoName      string
	HasProduct    bool
	HasRelease    bool
	CurrentLabels []string
}
