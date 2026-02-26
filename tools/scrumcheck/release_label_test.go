package main

import (
	"testing"

	"github.com/shurcooL/githubv4"
)

// TestLabelsContain provides scrumcheck behavior for this unit.
func TestLabelsContain(t *testing.T) {
	t.Parallel()

	labels := []string{":product", "bug"}
	if !labelsContain(labels, ":PRODUCT") {
		t.Fatal("expected case-insensitive match")
	}
	if labelsContain(labels, ":release") {
		t.Fatal("did not expect release label")
	}
}

// TestIssueLabels provides scrumcheck behavior for this unit.
func TestIssueLabels(t *testing.T) {
	t.Parallel()

	var it Item
	it.Content.Issue.Labels.Nodes = []struct{ Name githubv4.String }{
		{Name: githubv4.String("  :release ")},
		{Name: githubv4.String("bug")},
		{Name: githubv4.String(":Release")},
		{Name: githubv4.String("")},
	}

	got := issueLabels(it)
	if len(got) != 2 {
		t.Fatalf("expected 2 unique labels, got %d (%v)", len(got), got)
	}
}
