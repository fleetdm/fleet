# dibble

```
⠀⠀⠀⠀⠀⣀⣀⣤⣤⣤⣤⣤⠀⣀⣀⣀⠀⠀⠀⠀⠀⠀⡀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⣠⣴⣿⣿⣿⣿⣿⣿⣿⣿⡆⠸⣿⣿⣿⣷⣶⣤⣄⣾⣷⡄⠀⠀⠀⠀⠀⠀
⠀⢰⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠀⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣤⡀⠀⠀⠀
⠀⣤⣤⣤⣈⡉⠛⢿⣿⣿⣿⣿⣿⡆⢸⣿⣿⣿⣿⣿⣿⣿⣿⣧⣽⣿⣷⣄⠀⠀
⠀⢿⠿⣿⣿⣿⣷⣤⡈⢻⣿⣿⣿⣇⠈⣿⣿⣿⣿⣿⣿⠿⣿⣿⣿⣿⣿⣿⡄⠀
⠀⠈⠀⢸⣿⣿⣿⣿⠇⠀⠛⠛⠛⠋⠀⢻⣿⣿⡟⢉⠀⠀⠈⠙⠛⠿⠏⣿⣷⠀
⠀⠀⢠⣿⣿⡿⠟⢁⡄⠀⠀⠀⠀⠀⠀⠈⣿⣿⡇⣾⡀⠀⠀⠀⠀⠀⠀⠸⠿⠀
⠀⠀⠸⣿⣿⠀⢸⣿⣇⠀⠀⠀⠀⠀⠀⠀⢹⣿⡇⠸⣧⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠙⠛⠃⠀⠛⠛⠀⠀⠀⠀⠀⠀⠀⠘⠛⠛⠀⠙⠃⠀⠀⠀⠀⠀⠀⠀⠀

🌱  dibble — Fleet's seed slinger — Dibble the Tapir
```

> A dibble (n.) is a pointed wooden tool for poking holes in soil and planting
> seeds. This dibble is a pointed CLI for poking holes in your Fleet server and
> planting test data.

`dibble` is the one-stop tool for populating a Fleet server with everything
you'd want for dev or testing — users, teams (fleets), policies, reports,
labels, scripts, MDM profiles, software, enroll secrets, certificate authorities,
and vulnerable software inventory.

**Hosts are intentionally out of scope.** Use [`cmd/osquery-perf`](../../cmd/osquery-perf)
for those — duplicating its work here would be a waste of perfectly good seeds.

## Quickstart

```bash
# Build
make dibble                  # writes ./tools/dibble/dibble

# Run the wizard (recommended first time)
./tools/dibble/dibble

# Or skip the wizard and use flags
./tools/dibble/dibble all \
  --fleet-url http://localhost:8080 \
  --api-token "$FLEET_API_TOKEN"
```

The wizard asks for any missing config (Fleet URL, API token), offers to save
it to `~/.dibble.yaml`, lets you pick a theme and which entities to seed, and
then plants the seeds.

## Subcommands

| Command            | What it plants                                           |
| ------------------ | -------------------------------------------------------- |
| `dibble all`       | Everything below, with idempotent defaults               |
| `dibble users`     | Themed users with rotating roles (observer → admin)      |
| `dibble teams`     | Teams (aka fleets) with themed names                     |
| `dibble policies`  | Global + per-team policies, mixed platforms              |
| `dibble reports`   | Saved reports (formerly "queries"), various intervals    |
| `dibble labels`    | Dynamic, query-based labels                              |
| `dibble scripts`        | Saved scripts (.sh, .ps1, .zsh) global + per-team        |
| `dibble profiles`       | Apple `.mobileconfig` + Windows `.xml` MDM profiles      |
| `dibble software`       | Software titles (custom-package upload is a v1 TODO)     |
| `dibble enroll-secrets` | Per-team enroll secrets — the credential fleetd uses to join a team. Distinct from "Fleet secrets" (secret variables). Global enroll secret is left alone. |
| `dibble cas`            | Certificate Authorities (placeholder for now)            |
| `dibble vulns`     | Vulnerable software, written directly to MySQL           |
| `dibble activities` | Fake activity rows, written directly to MySQL — **non-idempotent**, marked with `*` |
| `dibble ping`      | Sanity-check `--fleet-url` and `--api-token`             |
| `dibble version`   | Print version + signature line                           |

Aliases: `dibble fleets` ↔ `dibble teams`, `dibble queries` ↔ `dibble reports`.

## Configuration

Three layers, highest precedence first:

1. **CLI flags:** `--fleet-url`, `--api-token`, `--theme`, `--insecure`, `--suffix`, `--dry-run`, `-v`
2. **Environment:** `FLEET_URL`, `FLEET_API_TOKEN`, `DIBBLE_THEME`
3. **Config file:** `~/.dibble.yaml` (written by the wizard)

Example `~/.dibble.yaml`:

```yaml
fleet_url: https://localhost:8080
api_token: abc123...
theme: mix
insecure: true   # set when targeting a Fleet with a self-signed cert
```

> Pass `--insecure` (or set `insecure: true` in the config file) to skip TLS
> verification — same convention as `fleetctl --insecure`. The wizard offers
> this automatically when it can't validate the cert on first ping.

### Re-seeding (avoiding "skipped" on repeat runs)

dibble is idempotent: a name like *Heart of Gold* exists in Fleet after your
first run, so the second `dibble all` reports `0 created, N skipped` for
everything global (labels, policies, reports, …). To get fresh entries each
run, append a `--suffix`:

```bash
./tools/dibble/dibble all --suffix auto       # random 4-char tag per run
./tools/dibble/dibble all --suffix demo2      # explicit tag — useful for reruns
```

Names become *Heart of Gold (auto-b3f1)*, *Towel readiness check (demo2)*,
etc. Emails get the suffix as a `+tag` so they remain unique and valid.

## Themes

Each theme is a curated set of character names that get used for users, teams,
policies, software titles, labels, and scripts. Pick one with
`--theme <name>` or let `mix` interleave them all.

| Theme              | Display                                |
| ------------------ | -------------------------------------- |
| `mix` *(default)*  | Interleave every theme                 |
| `hitchhikers`      | Hitchhiker's Guide to the Galaxy       |
| `goodplace`        | The Good Place                         |
| `parksrec`         | Parks and Recreation                   |
| `tng`              | Star Trek: The Next Generation         |
| `lotr`             | The Lord of the Rings                  |
| `dbz`              | Dragon Ball Z                          |
| `robin_williams`   | Robin Williams characters              |
| `ghibli`           | Studio Ghibli                          |
| `cosmere`          | Brandon Sanderson's Cosmere            |
| `sailor_moon`      | Sailor Moon                            |

Adding a theme: drop a new file in `themes/` that calls `Register(Theme{...})`
from `init()`. The wizard, `--theme` flag, and `mix` blend pick it up automatically.

## Legacy tools → dibble

These older tools are deprecated in favor of dibble but **still on disk** —
each has a banner in its README pointing at the dibble equivalent. We'll
remove them once nothing references them. Use dibble for new work.

| Old path                                                 | dibble equivalent                            |
| -------------------------------------------------------- | -------------------------------------------- |
| `tools/team-builder/`                                    | `dibble teams --count N`                     |
| `tools/loadtest/fleetd_labels/`                          | `dibble labels --count N`                    |
| `tools/loadtest/scripts_and_profiles/`                   | `dibble scripts` + `dibble profiles`         |
| `tools/loadtest/unified_queue/`                          | `dibble software` (enqueue path is a TODO)   |
| `tools/mdm/apple/loadtest/`                              | `dibble teams` + `dibble profiles`           |
| `tools/software/vulnerabilities/seed_data/`              | `dibble vulns --macos N --ubuntu N --windows N` |
| `tools/software/vulnerabilities/performance_test/seeder/`| `dibble vulns --macos N --ubuntu N --windows N` (bulk mode) |
| `tools/seed_data/queries/`                               | `dibble reports --count N`                   |

Out-of-scope and **not** absorbed (those tools still live in `tools/`):

- `cmd/osquery-perf` — hosts (intentionally not replicated).
- `tools/loadtest/osquery` — osqueryd CPU/mem profiling shell scripts.
- `tools/mdm/assets` — encrypted-asset export/import (backup, not seeding).
- `tools/saml` — SimpleSAMLPHP fixture for SSO testing.

## Design notes & known TODOs

- **API-first.** Most seeders call the Fleet API as a bearer-authed client.
  The exceptions are `dibble vulns` and `dibble activities`, which write
  directly to MySQL — Fleet has no "create vulnerability" or "create
  activity" endpoint by design.
- **Idempotent.** Re-running `dibble all` against an already-seeded Fleet
  reports "skipped" rather than failing. **Exception:** `dibble activities`
  is intentionally non-idempotent — every run inserts a fresh batch
  prefixed with `*` and tagged with the current run id, so seeded rows are
  obvious in the UI and don't conflate across runs.
- **Custom-package software upload** isn't wired up yet. Today
  `dibble software` only logs intent.
- **Mock CA creation** also isn't wired up — placeholder until a mock CA
  type lands in Fleet.
- **Manual labels and hosts** need a populated host inventory; dibble doesn't
  spin up hosts (see osquery-perf).

## Contributing

dibble follows the standard Go project layout:

```
tools/dibble/
├── cmd/dibble/        # main entry point (just calls command.Execute)
├── pkg/
│   ├── command/       # cobra commands, client, wizard, logging
│   ├── seed/          # per-entity seed logic
│   └── themes/        # character/media datasets + tapir mascot
├── go.mod / go.sum    # dibble has its own module
└── README.md
```

dibble is a standalone Go module (`github.com/fleetdm/fleet/v4/tools/dibble`)
so its dependencies don't bleed into the root Fleet `go.mod`.

- Build: `make dibble` from the repo root, or `go build -o dibble ./cmd/dibble` from `tools/dibble`.
- Tests: `cd tools/dibble && go test ./...`
- Lint: `cd tools/dibble && golangci-lint run` (the repo-level `make lint-go-incremental` skips this module).
