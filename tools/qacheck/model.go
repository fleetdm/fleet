package main

import "github.com/shurcooL/githubv4"

type Item struct {
	ID githubv4.ID

	Content struct {
		Issue struct {
			Number githubv4.Int
			Title  githubv4.String
			Body   githubv4.String
			URL    githubv4.URI
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
				Name githubv4.String
			} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
		}
	} `graphql:"fieldValues(first: 20)"`
}

type DraftingCheckViolation struct {
	Item      Item
	Unchecked []string
	Status    string
}
