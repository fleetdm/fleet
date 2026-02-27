package main

import (
	"strings"
	"testing"
)

// TestEmbeddedReportTemplateLoaded ensures report HTML is sourced from the embedded frontend file.
func TestEmbeddedReportTemplateLoaded(t *testing.T) {
	t.Parallel()

	if strings.TrimSpace(htmlReportTemplate) == "" {
		t.Fatal("embedded report template is empty")
	}
	if !strings.Contains(htmlReportTemplate, "<title>scrumcheck report</title>") {
		t.Fatalf("embedded report template missing expected title, got prefix: %.120q", htmlReportTemplate)
	}
	if !strings.Contains(htmlReportTemplate, "<script src=\"/assets/app.js\"></script>") {
		t.Fatal("embedded report template missing app.js asset include")
	}
	jsRaw := string(mustReadEmbeddedUIAsset("app.js"))
	if !strings.Contains(jsRaw, "function renderDraftingFromState(state)") {
		t.Fatal("embedded app.js missing drafting render function")
	}
}
