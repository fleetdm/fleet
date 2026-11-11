package ghapi

import "testing"

func TestIssueRefKey(t *testing.T) {
	if got := IssueRefKey("fleetdm/fleet", 123); got != "fleetdm/fleet#123" {
		t.Errorf("IssueRefKey = %q", got)
	}
	// Case-insensitive: GitHub repo names are, so the key must normalize.
	if IssueRefKey("FleetDM/Fleet", 123) != IssueRefKey("fleetdm/fleet", 123) {
		t.Error("IssueRefKey should be case-insensitive on the repo")
	}
	if got := IssueRefKey("fleetdm/fleet", 0); got != "" {
		t.Errorf("number 0 (draft) should have no key, got %q", got)
	}
}

func TestProjectItemRepoFullNameAndKey(t *testing.T) {
	cases := []struct {
		name     string
		item     ProjectItem
		wantRepo string
		wantKey  string
	}{
		{
			"from content URL",
			ProjectItem{Content: ProjectItemContent{Number: 5, URL: "https://github.com/fleetdm/confidential/issues/5"}},
			"fleetdm/confidential", "fleetdm/confidential#5",
		},
		{
			"from Repository URL field",
			ProjectItem{Repository: "https://github.com/fleetdm/fleet", Content: ProjectItemContent{Number: 7}},
			"fleetdm/fleet", "fleetdm/fleet#7",
		},
		{
			"from bare owner/name Repository field",
			ProjectItem{Repository: "fleetdm/fleet", Content: ProjectItemContent{Number: 7}},
			"fleetdm/fleet", "fleetdm/fleet#7",
		},
		{
			"unknown repo still keys by number",
			ProjectItem{Content: ProjectItemContent{Number: 9}},
			"", "#9",
		},
		{
			"draft issue has no key",
			ProjectItem{Content: ProjectItemContent{Title: "draft"}},
			"", "",
		},
	}
	for _, c := range cases {
		if got := c.item.RepoFullName(); got != c.wantRepo {
			t.Errorf("%s: RepoFullName = %q, want %q", c.name, got, c.wantRepo)
		}
		if got := c.item.IssueKey(); got != c.wantKey {
			t.Errorf("%s: IssueKey = %q, want %q", c.name, got, c.wantKey)
		}
	}
}
