package main

import "testing"

// TestReplaceUncheckedChecklistLine provides scrumcheck behavior for this unit.
func TestReplaceUncheckedChecklistLine(t *testing.T) {
	t.Parallel()

	text := "I have been confirmed to consistently lead to reproduction in multiple Fleet instances."

	t.Run("updates unchecked markdown item", func(t *testing.T) {
		t.Parallel()
		body := "- [ ] " + text + "\nother"
		gotBody, updated, alreadyChecked := replaceUncheckedChecklistLine(body, text)
		if !updated || alreadyChecked {
			t.Fatalf("expected updated=true alreadyChecked=false, got updated=%v alreadyChecked=%v", updated, alreadyChecked)
		}
		if gotBody == body {
			t.Fatalf("expected body to change")
		}
	})

	t.Run("reports already checked", func(t *testing.T) {
		t.Parallel()
		body := "- [x] " + text
		_, updated, alreadyChecked := replaceUncheckedChecklistLine(body, text)
		if updated || !alreadyChecked {
			t.Fatalf("expected updated=false alreadyChecked=true, got updated=%v alreadyChecked=%v", updated, alreadyChecked)
		}
	})

	t.Run("not found remains unchanged", func(t *testing.T) {
		t.Parallel()
		body := "- [ ] something else"
		gotBody, updated, alreadyChecked := replaceUncheckedChecklistLine(body, text)
		if updated || alreadyChecked {
			t.Fatalf("expected updated=false alreadyChecked=false, got updated=%v alreadyChecked=%v", updated, alreadyChecked)
		}
		if gotBody != body {
			t.Fatalf("expected body unchanged")
		}
	})
}

// TestIsValidRepoSlug provides scrumcheck behavior for this unit.
func TestIsValidRepoSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want bool
	}{
		{in: "fleetdm/fleet", want: true},
		{in: "fleet-dm/re_po.test", want: true},
		{in: "fleetdm/fleet/extra", want: false},
		{in: "../fleetdm/fleet", want: false},
		{in: "fleetdm", want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got := isValidRepoSlug(tc.in)
			if got != tc.want {
				t.Fatalf("isValidRepoSlug(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
