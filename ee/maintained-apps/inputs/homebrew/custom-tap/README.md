# Fleet custom-tap casks

This directory is a **self-contained source of truth** for Fleet-maintained apps that
don't exist in `Homebrew/homebrew-cask` and therefore can't be ingested from
`https://formulae.brew.sh/api/`.

It is laid out like a Homebrew tap:

```
custom-tap/
├── Casks/                # Cask DSL sources (.rb). Edit these.
│   ├── fleet-desktop.rb
│   └── druva-insync.rb
├── api/                  # Generated cask metadata (.json). Do not hand-edit.
│   ├── fleet-desktop.json
│   └── druva-insync.json
├── regenerate.sh         # Regenerates api/*.json from Casks/*.rb.
└── README.md             # You are here.
```

It is **not** a real Homebrew tap — the Fleet repo isn't named
`homebrew-<something>` and `Casks/` isn't at the repo root, so `brew tap` /
`brew install` against it won't work. It exists solely to feed Fleet's FMA
ingester, and keeping both the source (`.rb`) and the built artifact (`.json`)
in the same repo means the PR that changes a cask also tests the change in
CI.

## How it hooks into the FMA ingester

Each app here has an input manifest one directory up
(`../<token>.json`) with a `cask_path` field pointing at the generated JSON.
Example — [`../fleet-desktop.json`](../fleet-desktop.json):

```json
{
  "name": "Fleet Desktop",
  "token": "fleet-desktop",
  "cask_path": "ee/maintained-apps/inputs/homebrew/custom-tap/api/fleet-desktop.json",
  ...
}
```

When `go run cmd/maintained-apps/main.go` runs, the ingester sees `cask_path`,
reads the local file, and skips the brew API entirely. Apps without
`cask_path` continue to use `https://formulae.brew.sh/api/` as before.

## Adding a new cask

1. Write the cask DSL in `Casks/<token>.rb`. Use an existing file or
   <https://docs.brew.sh/Cask-Cookbook> as a reference.
2. Run `./regenerate.sh` in this directory to produce `api/<token>.json`.
3. Create an input manifest at
   `ee/maintained-apps/inputs/homebrew/<token>.json` that points
   `cask_path` at the new JSON. Follow the template in
   `../fleet-desktop.json`.
4. Generate the FMA output manifest:
   `go run cmd/maintained-apps/main.go --slug="<token>/darwin"` from the repo
   root.
5. Follow the rest of the FMA contributor flow in
   [`../../../README.md`](../../../README.md) (apps.json description, icon,
   PR).

## Updating an existing cask

1. Edit the stanza you care about in `Casks/<token>.rb`.
2. Run `./regenerate.sh` to refresh `api/<token>.json`.
3. Regenerate the FMA output manifest:
   `go run cmd/maintained-apps/main.go --slug="<token>/darwin"`.
4. Commit all three changes together: the `.rb`, the `.json`, and the
   `outputs/<token>/darwin.json`.

## Why `regenerate.sh` strips fields

`brew info --cask --json=v2` includes several fields that depend on the
developer's machine or the throwaway tap the script uses internally —
`installed`, `installed_time`, `outdated`, `full_token`, `tap`,
`tap_git_head`, `generated_date`. None of these are read by the FMA
ingester, so the script strips them to keep committed JSON stable across
machines.

## Requirements

- macOS (the `.rb` DSL is parsed by Homebrew).
- Homebrew installed (`brew` on PATH).
- `jq` (`brew install jq`).
