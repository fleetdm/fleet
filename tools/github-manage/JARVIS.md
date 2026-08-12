# 🎩 Jarvis — your personal work dashboard

> *"Your GitHub work, sorted by leverage."*

`gm jarvis` is an interactive TUI that aggregates **your** open work across
Fleet's GitHub repo into leverage-ordered buckets: what's blocking others,
what you can merge right now, what needs your hands, your review queue, and
what's gone cold. It replaces the seven-tab morning triage with one screen
and a keystroke per decision.

Prereqs (build + `gh auth` with `project` scope) are covered in
[README.md](README.md). Once `./gm` is built and `gh` has the `project`
scope, launch with:

```bash
./gm jarvis
```

## First run

The first launch walks you through:

1. **Role picker** — Developer / Manager / QA / Design. Tailors the Claude
   seed prompt that Start Work fires off, and (for QA) surfaces issues
   awaiting QA with a `t` test action.
2. **Project picker** — fuzzy-search Fleet's open project boards and
   multi-select the ones you work out of. These drive the top **Project View**
   section (issues assigned to you in any status except Done / Ready for
   release, plus a count of unassigned issues sitting in Ready).

Selections are written to `~/.config/gm/jarvis/config.json`. Re-open the
project picker anytime with `P`; re-run onboarding by deleting the file.

## What jarvis pulls

- Pull requests you **authored** (mergeability, CI, review state)
- Pull requests **awaiting your review**
- **Issues assigned to you**, with their project board Status

Fetches are cached at `~/.config/gm/jarvis/cache.json`. Jarvis opens
instantly from a cache younger than 4h — press `R` to refresh everything,
`r` to refresh just the highlighted item, or launch with `--no-cache` to
force a live pull on startup.

## Board view — one keystroke per decision

Navigate with `↑/↓` (`j/k`), jump with `g/G`, open the highlighted item in
your browser with `enter`. The action row drives the whole dev lifecycle:

| Key | Action |
|-----|--------|
| `w` | **Start work** — branch off `main` in a local clone, launch a Claude session, set the issue's project Status to *In progress* |
| `v` | Mark the issue **In review** |
| `m` | **Merge** the linked PR (squash) — advances the issue to *Awaiting QA* |
| `M` | Merge **+ start a Claude cherry-pick session** for the merged PR |
| `a` | Mark **Awaiting QA** |
| `p` | Pin / unpin to **Focus** |
| `t` | *(QA role only)* Spin up a repro/verify Claude session on the PR branch |
| `b` | Open the selected issue's most recently updated **project board** in your browser |
| `c` | Post a **comment** on the highlighted item |
| `s` | **Snooze** (1h / 4h / tomorrow / 1 week) |
| `d` | **Dismiss** (hide until it changes) |
| `x` | Mark **Done** locally · `u` clear local state |
| `H` | Show/hide dismissed items |
| `P` | Re-open the **project picker** |
| `f` | Switch to **Focus view** (see below) · `J` jump between sections |
| `r/R` | Refresh **one** / **everything** |
| `q` | Quit |

## Focus view

Press `f` for an issue-centric card list of the work you've pinned. Each
card shows the issue's project Status, its linked PR (with mergeability
reason), the active Claude session (if any), and the single most useful
next action. Same action keys as the board (`w v m M a`) plus `p` to unpin
and `f` to return.

## Start Work — what actually happens

Pressing `w` on an issue:

1. Prompts for a branch name (prefilled from the issue).
2. Lists local clones of the repo it finds under `clone_base_dirs`
   (default `~/projects`) and shows which are free / dirty / on another
   branch. Pick one with `↑/↓`, confirm with `enter`.
3. Checks out `main`, pulls, creates the branch.
4. Launches a Claude Code session seeded with the role-specific start prompt
   (override per role in `config.json → start_prompts`).
5. Sets the issue's project Status to **In progress**.

If no clone is discovered, jarvis tells you where it looked and how to fix
it (add the parent dir to `clone_base_dirs`).

## Config reference — `~/.config/gm/jarvis/config.json`

```json
{
  "clone_base_dirs": ["~/src", "~/projects"],
  "primary_projects": ["g-mdm", "g-software", 58],
  "role": "developer",
  "start_prompts": {
    "developer": "Let's tackle issue #{{.Issue}}: {{.Title}}\n{{.URL}}",
    "qa": "Reproduce and verify #{{.Issue}} on branch {{.Branch}}."
  }
}
```

- `clone_base_dirs` — dirs scanned one level deep for local clones.
- `primary_projects` — project **numbers**, **gm aliases**, or **names**
  (`"g-apple-at-work"`). Managers of multiple teams can list several; each
  becomes its own selectable row in Project View.
- `role` — `developer` (default), `manager`, `qa`, or `design`.
- `start_prompts` — per-role Go text/template overrides. Available fields:
  `{{.Issue}} {{.Title}} {{.URL}} {{.Branch}}`.

## Troubleshooting

- **"could not resolve to a Node with the global id of …"** or projects
  showing as empty: you're missing the `project` scope. Run
  `gh auth refresh -s project` and relaunch.
- **Stale data:** press `R` for a full refresh, or start with `--no-cache`.
- **Wrong clone offered / none found:** update `clone_base_dirs` in
  `config.json` — jarvis only scans one level deep under each entry.
- **Reset onboarding:** `rm ~/.config/gm/jarvis/config.json` and relaunch.
