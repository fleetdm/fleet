package spec

import (
	"strings"
	"testing"
)

func TestValidateAcceptsBuiltDocument(t *testing.T) {
	doc, err := Build(sampleResult())
	if err != nil {
		t.Fatal(err)
	}
	b, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(b); err != nil {
		t.Fatalf("built document should validate: %v", err)
	}
}

func TestValidateRejectsGarbage(t *testing.T) {
	err := Validate([]byte("openapi: 3.1.0\ninfo:\n  title: x\npaths: []\n"))
	if err == nil {
		t.Fatal("want validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "valid") {
		t.Logf("error text: %v", err)
	}
}
