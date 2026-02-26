package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteHTMLReport(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	p, err := writeHTMLReport(HTMLReportData{
		GeneratedAt: "now",
		Org:         "fleetdm",
	})
	if err != nil {
		t.Fatalf("writeHTMLReport err: %v", err)
	}
	if !strings.HasSuffix(p, filepath.Join(reportDirName, reportFileName)) {
		t.Fatalf("unexpected report path: %q", p)
	}
	raw, err := os.ReadFile(filepath.Join(tmp, reportDirName, reportFileName))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(raw), "Scrum check") {
		t.Fatalf("report does not contain expected title")
	}
}

func TestUIBridgeReportPathAndURL(t *testing.T) {
	t.Parallel()

	b := &uiBridge{
		baseURL: "http://127.0.0.1:9999",
	}
	b.setReportPath("/tmp/x/index.html")
	if b.reportURL() != "http://127.0.0.1:9999/report" {
		t.Fatalf("unexpected reportURL: %q", b.reportURL())
	}
}
