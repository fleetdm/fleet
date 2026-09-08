package evidence

import "testing"

func TestScoreCatalog(t *testing.T) {
	s := Signals{"catalog": {}, "binary": {}}
	if Score(s) != 100 {
		t.Fatalf("catalog score=%d want 100", Score(s))
	}
	if !ShouldEmit(s) {
		t.Fatal("catalog should emit")
	}
}

func TestNameAloneNeverEmits(t *testing.T) {
	s := Signals{"name_match": {}}
	if ShouldEmit(s) {
		t.Fatal("name_match alone must not emit")
	}
	if Score(s) != WeightNameMatch {
		t.Fatalf("score=%d", Score(s))
	}
}

func TestStrongShapeEmitsOffline(t *testing.T) {
	s := Signals{"workspace_shape": {}, "mcp_config": {}, "instructions": {}}
	if !ShouldEmit(s) {
		t.Fatal("strong shape must emit")
	}
	if Score(s) < 40 {
		t.Fatalf("score=%d want >=40", Score(s))
	}
}

func TestWeakInstructionsAloneNoEmit(t *testing.T) {
	s := Signals{"instructions": {}}
	if ShouldEmit(s) {
		t.Fatal("instructions alone must not emit agents candidate")
	}
}

func TestFrameworkAloneEmits(t *testing.T) {
	s := Signals{"framework:crewai": {}}
	if !ShouldEmit(s) {
		t.Fatal("framework must emit")
	}
	if Score(s) != WeightFramework {
		t.Fatalf("score=%d want %d", Score(s), WeightFramework)
	}
}

func TestGrokLikeSignals(t *testing.T) {
	s := Signals{"tool_home": {}, "binary": {}, "running": {}}
	if !ShouldEmit(s) {
		t.Fatal("tool_home+binary+running should emit")
	}
	want := WeightToolHome + WeightBinary + WeightRunning
	if Score(s) != want {
		t.Fatalf("score=%d want %d", Score(s), want)
	}
}

func TestEgressAloneNoEmit(t *testing.T) {
	s := Signals{"ai_egress": {}}
	if ShouldEmit(s) {
		t.Fatal("egress alone must not emit")
	}
}

func TestEvidenceCSVStable(t *testing.T) {
	s := Signals{"running": {}, "binary": {}, "tool_home": {}}
	if s.CSV() != "binary,running,tool_home" {
		t.Fatalf("csv=%q", s.CSV())
	}
}
