package processes

import "testing"

func TestIContains(t *testing.T) {
	cases := []struct {
		hay, needle string
		want        bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello", "xyz", false},
		{"abc", "abcd", false}, // needle longer
		{"anything", "", true}, // empty needle
		{"level=ERROR foo", "level=error", true},
	}
	for _, c := range cases {
		if got := icontains(c.hay, c.needle); got != c.want {
			t.Errorf("icontains(%q,%q) = %v, want %v", c.hay, c.needle, got, c.want)
		}
	}
}

func TestDetectLevel(t *testing.T) {
	cases := []struct {
		msg, want string
	}{
		{`ts=2026 level=error msg="boom"`, "error"},
		{`level=err something`, "error"},
		{`level=warn x`, "warn"},
		{`level=debug x`, "debug"},
		{`level=info x`, "info"},
		{`2026-01-01 ERROR failed to connect`, "error"}, // " ERROR " token in head
		{`error: could not bind`, "error"},              // head starts with "error"
		{`[2026] WARN disk low`, "warn"},
		{`something WARNING happened`, "warn"},
		{`[svc] DEBUG tick`, "debug"},
		{`[svc] INFO ready`, "info"},
		{`a perfectly ordinary line`, ""},
	}
	for _, c := range cases {
		if got := detectLevel(c.msg); got != c.want {
			t.Errorf("detectLevel(%q) = %q, want %q", c.msg, got, c.want)
		}
	}
}

func TestDetectLevelOnlyScansHead(t *testing.T) {
	// A " error " token past byte 64 should NOT be detected (head-only scan).
	pad := ""
	for len(pad) < 80 {
		pad += "x"
	}
	if got := detectLevel(pad + " error here"); got != "" {
		t.Errorf("token past 64 bytes should not match, got %q", got)
	}
}

func TestScrubSecrets(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Authorization: Bearer abc123def", "Authorization: Bearer [redacted]"},
		{"BEARER xyz789", "Bearer [redacted]"}, // case-insensitive prefix
		{"token=supersecret&next=1", "token=[redacted]&next=1"},
		{"password=hunter2 done", "password=[redacted] done"},
		{"authtoken=2abc", "authtoken=[redacted]"},
		{"nothing to see here", "nothing to see here"},
	}
	for _, c := range cases {
		if got := scrubSecrets(c.in); got != c.want {
			t.Errorf("scrubSecrets(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSignalName(t *testing.T) {
	if signalName(9) != "SIGKILL" || signalName(15) != "SIGTERM" || signalName(11) != "SIGSEGV" {
		t.Error("known signal names wrong")
	}
	if signalName(99) != "?" {
		t.Error("unknown signal should be ?")
	}
}
