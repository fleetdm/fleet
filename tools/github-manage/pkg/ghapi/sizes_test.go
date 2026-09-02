package ghapi

import "testing"

func mkSized(id string, num int, size string) ProjectItem {
	return ProjectItem{ID: id, Title: id, Size: size, Content: ProjectItemContent{Number: num}}
}

func mkTShirtSized(id string, num int, size string) ProjectItem {
	return ProjectItem{ID: id, Title: id, TShirtSize: size, Content: ProjectItemContent{Number: num}}
}

func TestSizeValue(t *testing.T) {
	if got := mkSized("a", 1, "M").SizeValue(); got != "M" {
		t.Errorf("Size field: got %q", got)
	}
	if got := mkTShirtSized("a", 1, "L").SizeValue(); got != "L" {
		t.Errorf("T-shirt size field: got %q", got)
	}
	if got := (ProjectItem{}).SizeValue(); got != "" {
		t.Errorf("unset: got %q", got)
	}
}

func TestPlanSizeSync(t *testing.T) {
	local := []ProjectItem{
		mkSized("in-sync", 1, "M"),
		mkSized("needs-local", 2, ""),
		mkSized("needs-remote", 3, "L"),
		mkSized("conflict", 4, "S"),
		mkSized("only-local", 5, "XL"),
		mkSized("both-unset", 6, ""),
	}
	remote := []ProjectItem{
		mkTShirtSized("r1", 1, "M"),
		mkTShirtSized("r2", 2, "XS"),
		mkTShirtSized("r3", 3, ""),
		mkTShirtSized("r4", 4, "L"),
		mkTShirtSized("r6", 6, ""),
		mkTShirtSized("only-remote", 7, "M"),
	}
	plan := PlanSizeSync(local, remote)

	if len(plan.SetLocal) != 1 || plan.SetLocal[0].Local.ID != "needs-local" || plan.SetLocal[0].Remote.SizeValue() != "XS" {
		t.Fatalf("SetLocal = %+v", plan.SetLocal)
	}
	if len(plan.SetRemote) != 1 || plan.SetRemote[0].Local.ID != "needs-remote" || plan.SetRemote[0].Remote.ID != "r3" {
		t.Fatalf("SetRemote = %+v", plan.SetRemote)
	}
	if len(plan.Conflicts) != 1 || plan.Conflicts[0].Local.ID != "conflict" {
		t.Fatalf("Conflicts = %+v", plan.Conflicts)
	}
}

func TestPlanSizeSyncCaseInsensitiveMatch(t *testing.T) {
	plan := PlanSizeSync(
		[]ProjectItem{mkSized("a", 1, "m")},
		[]ProjectItem{mkTShirtSized("r", 1, "M")},
	)
	if len(plan.SetLocal)+len(plan.SetRemote)+len(plan.Conflicts) != 0 {
		t.Fatalf("expected sizes differing only by case to count as in sync, got %+v", plan)
	}
}
