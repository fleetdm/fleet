package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderHTMLReport verifies the app shell renders expected HTML output.
func TestRenderHTMLReport(t *testing.T) {
	var out bytes.Buffer
	err := renderHTMLReport(&out, HTMLReportData{
		GeneratedAt: "now",
		Org:         "fleetdm",
	})
	if err != nil {
		t.Fatalf("renderHTMLReport err: %v", err)
	}
	if !strings.Contains(out.String(), "Scrum check") {
		t.Fatalf("report does not contain expected title")
	}
}

// TestUIBridgeReportPathAndURL verifies bridge report URL/path helpers.
func TestUIBridgeReportPathAndURL(t *testing.T) {
	b := &uiBridge{
		baseURL: "http://127.0.0.1:9999",
	}
	if b.reportURL() != "http://127.0.0.1:9999/" {
		t.Fatalf("unexpected reportURL: %q", b.reportURL())
	}
}
