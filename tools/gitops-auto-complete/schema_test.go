package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	ghodss "github.com/ghodss/yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const schemaFile = "generated-schema.json"

// compileSchema loads the committed schema and compiles it. The test validates
// against the committed artifact; TestSchemaUpToDate separately guarantees that
// artifact matches what the generator currently produces.
func compileSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	b, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatalf("read %s: %v", schemaFile, err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", doc); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	schema, err := c.Compile("schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return schema
}

// loadInstance reads a YAML fixture and decodes it into the JSON-compatible value
// the validator expects (ghodss converts YAML->JSON so numbers/bools are typed).
func loadInstance(t *testing.T, path string) any {
	t.Helper()
	y, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	j, err := ghodss.YAMLToJSON(y)
	if err != nil {
		t.Fatalf("yaml->json %s: %v", path, err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(j))
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return inst
}

// TestValidFixtures asserts every comprehensive valid gitops file validates
// cleanly against the schema (so all the keys they use stay covered).
func TestValidFixtures(t *testing.T) {
	schema := compileSchema(t)
	files, err := filepath.Glob("testdata/valid/*.yml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no valid fixtures found: %v", err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			if err := schema.Validate(loadInstance(t, file)); err != nil {
				t.Errorf("expected %s to validate, got:\n%v", file, err)
			}
		})
	}
}

// TestInvalidFixtures asserts the schema still rejects the specific mistakes the
// tool is designed to catch (unknown keys, wrong-typed source keys, a package
// with no source).
func TestInvalidFixtures(t *testing.T) {
	schema := compileSchema(t)
	files, err := filepath.Glob("testdata/invalid/*.yml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no invalid fixtures found: %v", err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			if err := schema.Validate(loadInstance(t, file)); err == nil {
				t.Errorf("expected %s to fail validation, but it passed", file)
			}
		})
	}
}

// TestInvariants pins the post-processing that a refactor could silently break.
func TestInvariants(t *testing.T) {
	b, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatalf("read %s: %v", schemaFile, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	defs, _ := doc["$defs"].(map[string]any)
	if defs == nil {
		t.Fatal("schema has no $defs")
	}
	def := func(name string) map[string]any {
		d, _ := defs[name].(map[string]any)
		if d == nil {
			t.Fatalf("missing $def %q", name)
		}
		return d
	}
	props := func(node map[string]any) map[string]any {
		p, _ := node["properties"].(map[string]any)
		return p
	}

	// Declarative notes: the exact set from declarativeExceptions must appear in
	// the schema, once each. Counting by note string keeps this independent of the
	// walker that placed them.
	t.Run("declarative notes", func(t *testing.T) {
		want := map[string]int{}
		for _, note := range declarativeExceptions {
			want[note]++
		}
		got := map[string]int{}
		for _, desc := range collectDescriptions(doc) {
			for note := range want {
				if containsNote(desc, note) {
					got[note]++
				}
			}
		}
		for note, count := range want {
			if got[note] != count {
				t.Errorf("note %q: want %d occurrence(s), got %d", note, count, got[note])
			}
		}
	})

	// Required source: software defs gate on an anyOf of single-key branches.
	t.Run("required source", func(t *testing.T) {
		for _, name := range []string{"SoftwarePackageSpec", "TeamSpecAppStoreApp", "MaintainedAppSpec"} {
			if _, ok := def(name)["anyOf"].([]any); !ok {
				t.Errorf("%s: expected an anyOf of required-source branches", name)
			}
		}
	})

	// Typed source keys survive relaxNulls as strict strings.
	t.Run("typed source keys", func(t *testing.T) {
		for def, keys := range map[string][]string{
			"SoftwarePackageSpec": {"url", "hash_sha256"},
			"TeamSpecAppStoreApp": {"app_store_id"},
			"MaintainedAppSpec":   {"slug"},
		} {
			p := props(defs[def].(map[string]any))
			for _, key := range keys {
				keyNode, _ := p[key].(map[string]any)
				if keyNode["type"] != "string" {
					t.Errorf("%s.%s: want type string, got %v", def, key, keyNode["type"])
				}
			}
		}
	})

	// agent_options.config.options is populated and closed; config stays open.
	t.Run("config.options", func(t *testing.T) {
		cfg, _ := props(def("AgentOptions"))["config"].(map[string]any)
		if cfg == nil {
			t.Fatal("AgentOptions.config missing")
		}
		opts, _ := props(cfg)["options"].(map[string]any)
		if opts == nil {
			t.Fatal("AgentOptions.config.options missing")
		}
		if opts["additionalProperties"] != false {
			t.Errorf("options should be closed (additionalProperties:false), got %v", opts["additionalProperties"])
		}
		if len(props(opts)) == 0 {
			t.Error("options has no properties")
		}
		if _, closed := cfg["additionalProperties"]; closed {
			t.Error("config should stay open (no additionalProperties)")
		}
	})

	// Path refs added to the file-reference defs.
	t.Run("path refs", func(t *testing.T) {
		for _, name := range []string{"Controls", "SoftwarePackageSpec", "LabelSpec"} {
			p := props(def(name))
			if _, ok := p["path"]; !ok {
				t.Errorf("%s: missing 'path'", name)
			}
			if _, ok := p["paths"]; !ok {
				t.Errorf("%s: missing 'paths'", name)
			}
		}
	})

	// Rename aliases: old key deprecated, new key present and not deprecated.
	t.Run("rename aliases", func(t *testing.T) {
		p := props(def("Controls"))
		old, _ := p["macos_setup"].(map[string]any)
		if old["deprecated"] != true {
			t.Errorf("macos_setup should be deprecated, got %v", old["deprecated"])
		}
		if _, ok := p["setup_experience"]; !ok {
			t.Error("setup_experience alias missing")
		}
	})
}

// TestSchemaUpToDate regenerates the schema and asserts the committed file matches,
// so a refactor that changes the output is caught (and the validation/invariant
// tests above stay meaningful against the committed artifact).
func TestSchemaUpToDate(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "out.json")
	cmd := exec.Command("go", "run", ".", tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go run . failed: %v\n%s", err, out)
	}
	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read regenerated schema: %v", err)
	}
	want, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatalf("read committed schema: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s is stale; run `go run . %s` to update", schemaFile, schemaFile)
	}
}

func collectDescriptions(node any) []string {
	var out []string
	switch node := node.(type) {
	case map[string]any:
		if desc, ok := node["description"].(string); ok {
			out = append(out, desc)
		}
		for _, child := range node {
			out = append(out, collectDescriptions(child)...)
		}
	case []any:
		for _, child := range node {
			out = append(out, collectDescriptions(child)...)
		}
	}
	return out
}

func containsNote(desc string, note string) bool {
	return bytes.Contains([]byte(desc), []byte(note))
}
