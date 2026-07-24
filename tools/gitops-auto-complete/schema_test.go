package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	ghodss "github.com/ghodss/yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const schemaFile = "generated-schema.json"

// compileSchema loads the committed schema and compiles it. The test validates
// against the committed artifact. TestSchemaUpToDate separately guarantees that
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
// tool is designed to catch (unknown keys, wrong-typed required keys, an item
// missing its required key).
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

	// Required keys: these defs gate on an anyOf of single-key branches.
	t.Run("required keys", func(t *testing.T) {
		for _, name := range []string{"SoftwarePackageSpec", "TeamSpecAppStoreApp", "MaintainedAppSpec"} {
			if _, ok := def(name)["anyOf"].([]any); !ok {
				t.Errorf("%s: expected an anyOf of required-key branches", name)
			}
		}
	})

	// Typed strict-string keys survive relaxNulls as strict strings.
	t.Run("typed strict-string keys", func(t *testing.T) {
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

	// agent_options.config.options is populated and closed, while config stays open.
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
		optKeys := props(opts)
		if len(optKeys) == 0 {
			t.Error("options has no properties")
		}
		// allow_unsafe comes from an embedded per-OS struct, so it guards embed handling.
		if _, ok := optKeys["allow_unsafe"]; !ok {
			t.Error("options missing embedded per-OS option 'allow_unsafe'")
		}
		if _, closed := cfg["additionalProperties"]; closed {
			t.Error("config should stay open (no additionalProperties)")
		}
	})

	// command_line_flags is populated and closed, including per-OS flags pulled up from
	// the embedded structs (users_service_delay isn't in the base flag struct).
	t.Run("command_line_flags", func(t *testing.T) {
		clf, _ := props(def("AgentOptions"))["command_line_flags"].(map[string]any)
		if clf == nil {
			t.Fatal("AgentOptions.command_line_flags missing")
		}
		if clf["additionalProperties"] != false {
			t.Errorf("command_line_flags should be closed (additionalProperties:false), got %v", clf["additionalProperties"])
		}
		flags := props(clf)
		if _, ok := flags["verbose"]; !ok {
			t.Error("command_line_flags missing base flag 'verbose'")
		}
		if _, ok := flags["users_service_delay"]; !ok {
			t.Error("command_line_flags missing embedded per-OS flag 'users_service_delay'")
		}
	})

	// Path refs: path-only defs carry `path` but not `paths`, and path+paths defs carry both.
	t.Run("path refs", func(t *testing.T) {
		for _, name := range []string{"ControlsWithTypes", "SoftwarePackageSpec"} {
			p := props(def(name))
			if _, ok := p["path"]; !ok {
				t.Errorf("%s: missing 'path'", name)
			}
			if _, ok := p["paths"]; ok {
				t.Errorf("%s: unexpected 'paths' on a path-only def", name)
			}
		}
		for _, name := range []string{"GitOpsPolicySpec", "LabelSpec"} {
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
		p := props(def("ControlsWithTypes"))
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

// TestControlsKeysCoverSpec fails when spec.GitOpsControls gains a controls key that
// the hand-written ControlsWithTypes hasn't mirrored. It's the guard that would have
// caught the missing name_template. The embedded fleet.BaseItem's path/paths are added
// separately by addPathReferences, and the tag-less Defined field is internal, so both
// are excluded from the comparison.
func TestControlsKeysCoverSpec(t *testing.T) {
	specKeys := jsonTagSet(reflect.TypeFor[spec.GitOpsControls]())
	delete(specKeys, "path")
	delete(specKeys, "paths")

	toolKeys := jsonTagSet(reflect.TypeFor[ControlsWithTypes]())
	for key := range specKeys {
		if !toolKeys[key] {
			t.Errorf("ControlsWithTypes is missing controls key %q from spec.GitOpsControls; add it", key)
		}
	}
}

// jsonTagSet returns the json key names of a struct, pulling names up through embedded
// structs and skipping fields with no json tag or "-".
func jsonTagSet(t reflect.Type) map[string]bool {
	keys := map[string]bool{}
	for field := range t.Fields() {
		if field.Anonymous {
			for key := range jsonTagSet(field.Type) {
				keys[key] = true
			}
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "" && name != "-" {
			keys[name] = true
		}
	}
	return keys
}
