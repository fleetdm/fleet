# Plan: `dibble` — Fleet's One-Stop Test Data Seeder

## Context

Fleet's repo has accumulated ~8 ad-hoc seeding tools — one per data category, each with its own flag conventions, its own auth wiring, and varying README quality. There is no single command that says "give me a populated dev Fleet I can poke at." This makes onboarding slow and forces engineers to read source to figure out which tool seeds what.

`dibble` consolidates all non-host seeding into one cobra-based CLI. (Hosts are intentionally excluded; `cmd/osquery-perf` already covers that surface, and re-implementing it here would duplicate work.) The tool is named after the gardening dibble — a pointed stick for planting seeds — and the README leans into the gardening pun. Seeded data uses character names from a rotating set of media themes (Hitchhiker's Guide, The Good Place, Parks & Rec, Star Trek: TNG, LOTR, DBZ, Robin Williams characters, Studio Ghibli, Sanderson's Cosmere, Sailor Moon) so that test data is visually distinct from real data and fun to work with in the UI.

**Mascot.** dibble's mascot is **Dibble the Tapir** — chosen because tapirs are famous for rooting around in the dirt with their snouts (perfect for a seed-planting tool), are underrated, and look great in ASCII. Dibble the Tapir appears in:
- The README header (full-size ASCII tapir, a short bio line).
- The wizard's welcome banner (smaller ASCII rendering above the prompts).
- `dibble version` output (signature line: `— Dibble the Tapir 🌱`).
- A tiny snout glyph 🐾 (or `>·)~` if no emoji) as the bullet character on progress lines: `>·)~ users… 5 planted.`

ASCII art lives in `tools/dibble/themes/mascot.go` as `const TapirLarge`, `const TapirSmall`, `const TapirSnout` — kept in code (not in `data/`) so it ships with the binary and renders without filesystem reads.

Outcome: one binary, `dibble`, that any engineer can run against a local Fleet to populate every seedable entity in seconds. Legacy seeding tools are deleted after dibble achieves parity. Anyone arriving at an old path via stale docs lands in dibble.

## Scope of entities seeded

In-scope (each gets a `dibble <entity>` subcommand):

1. **users** — admin / maintainer / observer / observer+ / gitops roles. Invite flow optional.
2. **teams** (alias: `fleets`) — with team config + per-team enroll secrets.
3. **policies** — global + per-team, platforms (macOS/Windows/Linux/ChromeOS).
4. **reports** (alias: `queries`) — saved queries with interval, observer_can_run, platforms. Subsumes `tools/seed_data/queries`.
5. **labels** — dynamic (query-based) + manual (host-assigned). Subsumes `tools/loadtest/fleetd_labels`.
6. **scripts** — saved scripts per team / global. Sample bash + powershell + zsh.
7. **profiles** — MDM Apple `.mobileconfig`, Apple DDM, Windows `.xml`. Subsumes `tools/loadtest/scripts_and_profiles` + `tools/mdm/apple/loadtest` MDM-profile-creation portion.
8. **software** — custom packages + Fleet-maintained app references. Subsumes `tools/loadtest/unified_queue` install/script enqueue.
9. **secrets** — global + team enroll secrets, plus secret variables.
10. **cas** — Certificate Authorities (SCEP/DigiCert/SmallStep mock entries).
11. **vulns** — vulnerable software rows. Subsumes `tools/software/vulnerabilities/seed_data` + `…/performance_test/seeder`. Direct-DB mode (it's bulk-only and bypasses real scanners by design).
12. **all** — runs the lot with sensible defaults; idempotent.

Out of scope (called out in README): hosts (osquery-perf), activities/audit log (auto-emitted), real ABM/VPP tokens (require Apple credentials), app config (single global doc — not "seed-able").

## Architecture

### Layout
```
tools/dibble/
├── README.md              # Gardening-pun intro, command map, theme list, legacy-tool migration table
├── dibble-plan.md         # Mirror of this plan committed to repo
├── main.go                # cobra root, registers subcommands
├── cmd_root.go            # persistent flags, viper bindings, client init
├── cmd_all.go             # `dibble all`
├── cmd_users.go           # one file per entity, thin cobra wrapper
├── cmd_teams.go
├── cmd_policies.go
├── cmd_reports.go
├── cmd_labels.go
├── cmd_scripts.go
├── cmd_profiles.go
├── cmd_software.go
├── cmd_secrets.go
├── cmd_cas.go
├── cmd_vulns.go
├── client.go              # net/http wrapper around Fleet API, bearer auth
├── seed/                  # business logic per entity (importable, testable)
│   ├── users.go
│   ├── teams.go
│   ├── ...                # one per entity
│   └── vulns.go           # MySQL path; reuses CSVs from old seed_data tool
├── themes/                # easter-egg name registries
│   ├── theme.go           # registry, Pick(kind) helper
│   ├── hitchhikers.go
│   ├── goodplace.go
│   ├── parksrec.go
│   ├── tng.go
│   ├── lotr.go
│   ├── dbz.go
│   ├── robin_williams.go
│   ├── ghibli.go
│   ├── cosmere.go
│   └── sailor_moon.go
└── data/                  # //go:embed
    ├── sample.mobileconfig
    ├── sample-windows.xml
    ├── sample-ddm.json
    ├── software-macos.csv      # carried over from tools/software/vulnerabilities/seed_data
    ├── software-ubuntu.csv
    └── software-win.csv
```

Single flat `main` package for cobra commands; `seed/` and `themes/` are subpackages so tests can target them directly. No tool-local `go.mod` — cobra (`v1.9.1`) and viper (`v1.20.1`) are already in root `go.mod`. Shares the root module → no version drift, no double-vendoring.

### Interactive wizard (no-arg mode)

Running `dibble` with no arguments launches a friendly wizard. This is the recommended entry point for humans; flags exist for scripting and CI.

Flow:

1. **Resolve config.** Load `~/.dibble.yaml` if present, then env vars. For any required value still missing (`fleet_url`, `api_token`), prompt for it inline with validation (URL must parse; token must be non-empty). Token is masked at the prompt.
2. **Offer to persist.** If any value was supplied interactively, confirm `Save these to ~/.dibble.yaml?` (default yes). Writes only the values the user just entered — never overwrites existing fields silently.
3. **Ping.** Hit `/api/v1/fleet/version` to sanity-check the URL + token before asking what to seed. Fail fast with a clear message if auth fails.
4. **Pick theme.** Single-select list of all themes plus `mix` (default highlighted).
5. **Pick entities.** Multi-select checkbox list of every seedable entity. Pre-checked: `users`, `teams`, `secrets`, `policies`, `labels`. The slow / expensive ones (`vulns`, large `reports`) are unchecked by default with a `(slow)` suffix on the label.
6. **Tune counts (optional).** A final confirm: `Use default counts? (y) / customise (n)`. If `n`, loop through chosen entities and prompt for an integer override; otherwise skip straight to execution.
7. **Run.** Stream progress to stdout: `🌱 users… 5 created. 🌱 teams… 3 created.` Final summary with elapsed time.

Library: `github.com/AlecAivazis/survey/v2` — well-suited for this exact pattern (Input, Password, Confirm, Select, MultiSelect prompts) and significantly simpler than Bubble Tea for a linear flow. Add to root `go.mod` if not already present.

Behaviour rules:
- Any flag set on the command line skips the corresponding wizard prompt. Running `dibble --fleet-url X` still launches the wizard but doesn't re-ask for the URL.
- `dibble all`, `dibble users`, etc. (any explicit subcommand) **never** triggers the wizard — those are scripting paths and must remain non-interactive. Wizard only fires when `os.Args` has no subcommand.
- `--no-wizard` flag forces non-interactive mode (errors out on missing required config). Useful for CI.
- A banner at the top featuring the tapir mascot:
  ```
       _.._
     .'    `-.
    /        \\___    🌱  dibble — Fleet's seed slinger.
   (o  o     `   `>     theme: mix   target: http://localhost:8080
    `--'             /     — Dibble the Tapir
        `~~~~~~~~~~~~~
  ```
  (Final ASCII to be tuned during implementation; lives in `themes/mascot.go`.)

### CLI surface

```
dibble                                # → wizard
dibble [global flags] <command> [flags]   # → non-interactive

Global flags (persistent, viper-bound):
  --fleet-url string      Fleet server URL (env: FLEET_URL, config: fleet_url)
  --api-token string      Fleet API token  (env: FLEET_API_TOKEN, config: api_token)
  --config string         Config file path (default: $HOME/.dibble.yaml)
  --theme string          One of: hitchhikers, goodplace, parksrec, tng, lotr,
                          dbz, robin_williams, ghibli, cosmere, sailor_moon, mix
                          (default: mix)
  --dry-run               Print what would be created, don't call the API
  -v, --verbose

Per-entity flag pattern:
  --count N               How many to create (default varies, e.g. 5 for users)
  --team string           Scope to a team by name (where applicable)

Examples:
  dibble all --fleet-url http://localhost:8080 --api-token $T
  dibble users --count 20 --theme cosmere
  dibble policies --team "Heart of Gold" --count 10
  dibble vulns --macos 100 --ubuntu 100 --windows 100
  dibble reports --count 1000000     # bulk path uses direct DB
```

Config precedence (viper layering): flag > env > `$HOME/.dibble.yaml` > defaults. Bonus: if `--api-token` is unset, auto-discover from `~/.fleet/config` (where fleetctl writes it).

### API client

Thin `client.go` over `net/http`. Bearer auth from viper. Provides:
- `Post(path string, body any, out any) error`
- `Get(path string, out any) error`
- `Patch(path string, body any, out any) error`
- `PostMultipart(path string, fields map[string]string, files map[string]io.Reader, out any) error` — for installer/profile uploads
- `EnsureNoConflict` — idempotency helper: if create returns 409/"already exists", treat as success unless `--strict`.

Deliberately not using `server/service/client.go` — that's an internal client coupled to server internals. A small bespoke client keeps dibble decoupled and matches the convention used by `tools/loadtest/*` and `tools/team-builder`.

### Themes

`themes/theme.go` exposes:
```go
type Theme struct {
    Name      string
    Users     []Person   // First, Last, Handle
    Teams     []string
    Policies  []NamedItem
    Software  []NamedItem
    Labels    []string
    Scripts   []NamedItem
}
func All() []Theme
func Get(name string) (Theme, error)   // "mix" returns a blended Theme
func Pick(t Theme, kind string, i int) string  // deterministic rotation, no dup collisions
```

Each theme file is ~60 lines of curated `Theme{...}` literal. Curating is the fun part — examples:

- **hitchhikers**: users Arthur Dent, Ford Prefect, Zaphod Beeblebrox, Trillian, Marvin; teams Heart of Gold, Magrathea, Vogon Constructor Fleet; policy "Towel readiness check"; software "Babel Fish", "Pan Galactic Gargle Blaster"; label "knows-where-towel-is".
- **goodplace**: Eleanor Shellstrop, Chidi Anagonye, Tahani Al-Jamil, Jason Mendoza, Michael, Janet; teams "Good Place", "Bad Place", "Medium Place"; policy "Ethical compliance check"; software "Janet Void Browser".
- **cosmere**: Kaladin, Shallan, Dalinar, Vin, Kelsier, Hoid; teams "Roshar Bridge Four", "Scadrial Survivors"; software "Stormlight Manager"; label "windrunner-eligible".
- **ghibli**: Totoro, Chihiro, Howl, Sophie, Kiki, San; teams "Spirited Bathhouse", "Moving Castle"; software "Catbus Transit"; label "soot-sprite-detected".
- **sailor_moon**: Usagi, Ami, Rei, Makoto, Minako, Mamoru, Chibiusa, Luna; teams "Inner Senshi", "Outer Senshi"; software "Moon Tiara Action".
- **lotr**, **tng**, **parksrec**, **dbz**, **robin_williams** — same structure.

Default `--theme mix` interleaves entries from every theme so a single `dibble all` run produces a riot of references. Deterministic with `--seed N` if needed for tests.

### Build

Add to root `Makefile`:
```makefile
dibble:
	go build -o tools/dibble/dibble ./tools/dibble
```

Binary lives next to the source at `tools/dibble/dibble` (not `build/`). Add `tools/dibble/dibble` to `.gitignore`. Document `go build -o tools/dibble/dibble ./tools/dibble` and `go run ./tools/dibble all …` in the README.

## Legacy tool retirement

**Revised 2026-05-23:** Leave the legacy tools on disk and add a deprecation
banner pointing at dibble to each one's README. We'll remove them in a
follow-up once nothing references them. The original list below now reads as
"add deprecation banner" rather than "delete":

**Deprecate (add banner pointing at dibble equivalent):**
- `tools/team-builder/` → `dibble teams`
- `tools/loadtest/fleetd_labels/` → `dibble labels`
- `tools/loadtest/scripts_and_profiles/` → `dibble scripts` + `dibble profiles`
- `tools/loadtest/unified_queue/` → `dibble software --enqueue-runs`
- `tools/mdm/apple/loadtest/` → `dibble profiles --platform apple --count N` + `dibble teams`
- `tools/software/vulnerabilities/seed_data/` → `dibble vulns` (CSVs migrate to `tools/dibble/data/`)
- `tools/software/vulnerabilities/performance_test/seeder/` → `dibble vulns --volume`
- `tools/seed_data/queries/` → `dibble reports --count N`

**Keep (out of dibble scope):**
- `tools/loadtest/osquery/` — shell scripts + gnuplot for osqueryd CPU/mem profiling. Not API seeding.
- `tools/mdm/assets/` — export/import of MDM-encrypted assets. Backup tool, not seeder.
- `tools/saml/` — SimpleSAMLPHP fixture. Auth test infra.
- `cmd/osquery-perf` — hosts, out of scope.

If a parent directory becomes empty after deletion (e.g. `tools/seed_data/`), delete the parent too. Don't delete shared parents (`tools/loadtest/`, `tools/mdm/`, `tools/software/`) — other things live there.

For any external doc that links to a deleted path, leave the path in a "Legacy tools → dibble" table at the top of `tools/dibble/README.md`, so a grep for the old name lands somewhere helpful.

## Files to create

- `tools/dibble/main.go`, `cmd_root.go`, `cmd_<entity>.go` (×12)
- `tools/dibble/wizard.go` — interactive no-arg flow using `survey/v2`
- `tools/dibble/client.go`
- `tools/dibble/seed/*.go` (one per entity)
- `tools/dibble/themes/theme.go` + `themes/<theme>.go` (×10) + `themes/mascot.go` (Dibble the Tapir ASCII art)
- `tools/dibble/data/sample.mobileconfig`, `sample-windows.xml`, `sample-ddm.json`, the three vuln CSVs
- `tools/dibble/README.md` (gardening intro + command map + theme table + legacy migration table)
- `tools/dibble/dibble-plan.md` (this plan, committed for posterity per user request)
- Test files: `seed/*_test.go` for non-trivial helpers, `themes/theme_test.go` for `Pick` determinism

## Files to modify

- `Makefile` — add `dibble:` target
- `tools/README.md` — add a `dibble` entry pointing at the new tool

## Files to delete (after parity verified)

As listed above.

## Implementation steps (suggested execution order)

1. Scaffold `tools/dibble/` with `main.go`, `cmd_root.go`, viper/cobra wiring, and `dibble version`. Confirm it builds.
2. Implement `client.go` + auth. Add a `dibble ping` subcommand that hits `/api/v1/fleet/version` to validate config.
3. Implement `themes/` package and `theme_test.go`. No API yet, pure data.
4. Implement entity seeders in this order, each behind its own subcommand: users → teams → secrets → labels → policies → reports → scripts → profiles → software → cas → vulns.
5. Implement `dibble all` (sequences the above, idempotent).
6. Implement the interactive wizard in `wizard.go`. Triggers when `main()` sees no subcommand. Uses `survey/v2` prompts; reuses the same seed functions as the subcommands so wizard and CLI paths can't drift.
7. Write `README.md` with gardening pun, wizard demo (asciinema or screenshot block), command table, theme list, legacy-tool migration table.
8. Write `dibble-plan.md` (this plan, committed alongside).
9. Migrate vuln CSVs into `data/`. Run `dibble vulns` and confirm it produces the same row counts as the old tool.
10. Verify parity for each legacy tool (see verification below). Once green, delete legacy tools and their empty parents.
11. Add `make dibble` target and `.gitignore` entry.

## Verification

End-to-end smoke test:
```bash
make deps && make dibble
make serve   # in another terminal; uses local MySQL+Redis
# create admin user via the UI bootstrap or fleetctl, grab API token
./tools/dibble/dibble ping --fleet-url http://localhost:8080 --api-token $T
./tools/dibble/dibble all --fleet-url http://localhost:8080 --api-token $T
```

Open the Fleet UI and confirm: users page shows themed accounts, fleets/teams page shows themed teams, policies page lists global + per-team policies, labels page shows both dynamic and manual, software page shows seeded titles, mdm → profiles shows both Apple and Windows entries. Run `dibble vulns --macos 100` and confirm vulnerable software appears in the host detail of an existing host.

Wizard smoke test:
```bash
rm -f ~/.dibble.yaml      # start clean
./tools/dibble/dibble     # no args → wizard
# answer prompts; confirm save-to-yaml; pick "mix" theme; pick users+teams; default counts; run
./tools/dibble/dibble     # second run → wizard skips URL/token prompts (loaded from ~/.dibble.yaml)
./tools/dibble/dibble --no-wizard   # errors out cleanly if config is missing
```

Parity tests (run before deleting any legacy tool):
- For each legacy tool, run its old invocation against a fresh Fleet, snapshot the resulting row counts in MySQL.
- Reset Fleet (`make db-reset`). Run the equivalent `dibble` command. Compare row counts. They should match within ±1 (timestamps may shift ordering).

Automated tests:
- `go test ./tools/dibble/...` for theme determinism, idempotency helpers, CSV parsing.
- No integration tests against a live Fleet in CI — this is dev-only tooling.

Lint:
- `make lint-go-incremental` after the implementation lands.
