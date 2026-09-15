package jarvis

import (
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

func TestProjectEmoji(t *testing.T) {
	cases := []struct{ title, want string }{
		{"🍎 #g-apple-at-work", "🍎"},
		{"❤️‍🩹 #g-auto-patching", "❤️‍🩹"},
		{"🗺️ Release planning", "🗺️"},
		{"#g-mdm", ""},
		{"Drafting", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := projectEmoji(c.title); got != c.want {
			t.Errorf("projectEmoji(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}

func TestGroupInfo(t *testing.T) {
	issueWithLabels := func(num int, labels ...string) Item {
		ls := make([]ghapi.Label, len(labels))
		for i, n := range labels {
			ls[i] = ghapi.Label{Name: n}
		}
		return Item{Kind: KindIssue, Number: num, Issue: &ghapi.Issue{Number: num, Labels: ls}}
	}

	m := Model{issueProjects: map[int][]ProjectRef{
		1: {
			{Number: 67, Title: "Drafting", Status: "In review"},
			{Number: 108, Title: "🍎 #g-apple-at-work", Status: "In progress"},
		},
		2: {
			{Number: 109, Title: "❤️‍🩹 #g-auto-patching", Status: "Ready"},
			{Number: 108, Title: "🍎 #g-apple-at-work", Status: "Settled"},
		},
		3: {
			{Number: 87, Title: "🗺️ Release planning", Status: "4.90"},
		},
	}}

	// On the labeled group board: emoji + that board's status (not Drafting's).
	emoji, status := m.groupInfo(issueWithLabels(1, "bug", "#g-apple-at-work"))
	if emoji != "🍎" || status != "In progress" {
		t.Errorf("issue 1: got %q %q", emoji, status)
	}

	// Label picks between multiple group boards.
	emoji, status = m.groupInfo(issueWithLabels(2, "#g-auto-patching"))
	if emoji != "❤️‍🩹" || status != "Ready" {
		t.Errorf("issue 2: got %q %q", emoji, status)
	}

	// No label: fall back to the issue's only group board.
	emoji, status = m.groupInfo(issueWithLabels(1))
	if emoji != "🍎" || status != "In progress" {
		t.Errorf("issue 1 no label: got %q %q", emoji, status)
	}

	// Labeled but not on the board: emoji learned from other issues, no status.
	emoji, status = m.groupInfo(issueWithLabels(3, "#g-apple-at-work"))
	if emoji != "🍎" || status != "" {
		t.Errorf("issue 3: got %q %q", emoji, status)
	}

	// Non-group projects only, no label: nothing.
	emoji, status = m.groupInfo(issueWithLabels(3))
	if emoji != "" || status != "" {
		t.Errorf("issue 3 no label: got %q %q", emoji, status)
	}
}
