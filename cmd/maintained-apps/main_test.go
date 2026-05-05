package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONEncoderPreservesHTML(t *testing.T) {
	testData := struct {
		Description string `json:"description"`
	}{
		Description: `Test with HTML: <a href="https://example.com">link</a> & special chars < >`,
	}

	// Test with SetEscapeHTML(false)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(testData); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	result := buf.String()

	// Verify HTML characters are preserved, not escaped
	if strings.Contains(result, `\u003c`) {
		t.Error("Found escaped '<' character (\\u003c) - HTML escaping is still enabled")
	}
	if strings.Contains(result, `\u003e`) {
		t.Error("Found escaped '>' character (\\u003e) - HTML escaping is still enabled")
	}
	if strings.Contains(result, `\u0026`) {
		t.Error("Found escaped '&' character (\\u0026) - HTML escaping is still enabled")
	}

	// Verify HTML characters are present (note: quotes inside JSON are still escaped)
	if !strings.Contains(result, `<a href=\"https://example.com\">`) {
		t.Error("HTML anchor tag was not preserved correctly")
	}
	if !strings.Contains(result, ` & `) {
		t.Error("Ampersand character was not preserved correctly")
	}

	t.Logf("Successfully preserved HTML in JSON output: %s", result)
}
