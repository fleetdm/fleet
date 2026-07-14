# gitops-auto-complete

Generates a JSON schema from Fleet's GitOps Go structs so editors (yaml-language-server)
can offer completion and validation for GitOps YAML.

Not tracked in git (yet). Kept as its own Go module so it doesn't touch the root
`go.mod`/`go.sum`.

## Run

```bash
cd tools/gitops-auto-complete
go run .                       # print schema to stdout
go run . generated.schema.json # write to a file
```

Then point yaml-language-server at the output, e.g. a modeline at the top of a YAML file:

```yaml
# yaml-language-server: $schema=/absolute/path/to/generated.schema.json
```

## How it works

Reflects a local `GitOpsSpec` struct (in `main.go`) that carries the real top-level
GitOps YAML keys (`name`, `org_settings`, `settings`, `agent_options`, `controls`,
`policies`, `reports`, `software`, `labels`) via json tags, reusing Fleet's existing
types for each section (`spec.GitOpsOrgSettings` which embeds `fleet.AppConfig`,
`spec.GitOpsSoftware`, `fleet.AgentOptions`, etc.).

Uses `github.com/invopop/jsonschema` with:
- `RequiredFromJSONSchemaTags: true` — nothing is required unless explicitly tagged
  (otherwise yamlls auto-scaffolds every field on completion).
- `KeyNamer: toSnake` — reused Fleet structs whose fields lack json tags
  (e.g. `GitOpsSoftware.Packages`) would otherwise reflect as PascalCase keys.
- `Mapper: typeMapper` — collapses `optjson.*` wrappers to the scalar/array they
  marshal to, and `fleet.Duration` to a string (it embeds `time.Duration`, which
  otherwise produces a self-referential `$def` that overflows yamlls' resolver).
- `ExpandedStruct: true` — inlines the root struct's properties so yamlls offers
  root-level key completion.

## Post-processing (in `main.go`)

It also types `agent_options.config.options.*` by parsing the AST of Fleet's
generated `server/fleet/agent_options_generated.go` (the unexported `osqueryOptions`
struct — 110 osquery options). This gives completion, types, and strict unknown-key
validation matching Fleet, without importing the unexported type. The rest of
`config` (schedule, decorators, ...) stays open. Because of this the tool must run
from its own directory (the path is relative: `../../server/fleet/...`).

After reflecting, the generated schema is walked to match real GitOps files:
- `addTypeDescriptions` — sets each property's `description` to its type (e.g. `string`,
  `AppleOSUpdateSettings`, `array<string>`) so yamlls shows it on hover (`K`). Hover
  renders `description`, not the bare type, so this is required for hover to show
  anything. Runs first, before relaxNulls strips scalar types.
- `addRenameAliases` — adds each `renameto` key alongside its json-tag name so both
  the old and new spellings validate (Fleet accepts both).
- `addPathRefs` — adds `path`/`paths` to the `pathRefDefs` set (top-level section
  values and list-item types) where GitOps supports file references, rather than to
  every object (keeps completion uncluttered).
- `relaxNulls` — empty placeholder keys (`minimum_version:`) are common in GitOps and
  parse as null. yamlls can't validate null without also suggesting it in completion,
  so scalar leaves drop their `type` (empty stays valid, no `null` suggestion, but the
  value is no longer type-checked); objects/arrays keep `[type, null]` so their
  structure/key completion survives.

Real-file validation went from ~30 diagnostics to 0–3 on the canonical
`it-and-security/default.yml` and `my-gitops/**` files.

## Known limitations

- `agent_options.config`, `secrets`, `certificate_authorities` are open (`json.RawMessage`
  / `any`) so they accept anything.
- A few keys still flag: `controls.macos_setup.end_user_license_agreement` (gitops-only
  field not on `fleet.MacOSSetup`), `label_membership_type` (a `uint` enum that marshals
  as a string), and `integrations.google_calendar` (array shape). Each is a one-off
  custom-marshal/wrapper mismatch.
- Slices of structs inside `optjson.Slice` become `array<object>` (open items) rather
  than a `$ref`.
