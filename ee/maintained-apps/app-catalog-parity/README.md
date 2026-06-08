# App catalog parity

Tracking work to ensure Fleet has a [Fleet-maintained app (FMA)](../README.md) for every macOS app on [appcatalog.cloud](https://appcatalog.cloud/apps) that ships a Homebrew cask.

## Snapshot

| Metric | Count |
|---|---:|
| Apps on appcatalog.cloud (sitemap) | 1,653 |
| …matched to a Homebrew cask | 1117 slugs → 1092 unique casks |
| …already a Fleet-maintained app | 213 |
| …cask exists, FMA missing (work queue) | 879 |
| **…added as FMAs (A–Z + #)** | **838** |
| …deferred (see below) | 41 |
| Apps with no Homebrew cask (handoff) | 533 |

## Status: complete

All 879 work-queue casks have been processed, one letter per commit (A–Z plus the `#` numeric-leading bucket). **838 were added as Fleet-maintained apps; 41 were deferred.** Remaining open items:

- **Descriptions are auto-generated** from each cask's Homebrew `desc` and need a human review/polish pass before these ship in a PR.
- **Icons are not yet generated** for the added apps (see `tools/software/icons`); FMAs work without them and the icon appears the following release.
- The 41 deferrals are checkbox-unchecked in [to-add.md](to-add.md), each annotated with a reason. Categories:
  - **Unsupported installer format** (`.tar.gz`, 7z/xz, etc.) — e.g. alfred, gitbutler, vivaldi, cap, codex.
  - **No app bundle in the installer** (CLI binary, nested `.pkg`, EULA-only) — e.g. 1password-cli, mailmate, textmate, rawtherapee, busycal, rode-central.
  - **Installer-stub / bootstrapper** apps — autodesk-fusion, blockblock, soapui.
  - **Ingester limitation**: casks using an `uninstall.signal` string field the generator can't parse — karabiner-elements, keybase, wire (fixable in `ee/maintained-apps/ingesters/homebrew`).
  - **Transient** (retryable): qgis (download timeout), scroll-reverser (cask API miss).

## Files

- [to-add.md](to-add.md) — the FMA work queue, grouped A–Z. Checked = added; unchecked = deferred (with reason).
- [no-homebrew-cask.md](no-homebrew-cask.md) — apps lacking a Homebrew cask, for separate upstream submission.

## Methodology

1. All app URLs pulled from `https://appcatalog.cloud/sitemap.xml` (the live page only prerenders 100).
2. The full Homebrew cask list (`https://formulae.brew.sh/api/cask.json`) is the source of truth for whether a cask exists — not the catalog's own labels.
3. A catalog slug matches a cask by: exact token, token with a trailing 5-hex dedupe suffix stripped, normalized display-name match, or version-suffix stripped (`acorn-7` → `acorn`).
4. The work queue is the matched casks **not** already present in `inputs/homebrew/`.

FMAs are added one alphabet letter per commit. Identity fields are derived from cask metadata (fast path) and confirmed by the CI validator; bundle IDs that could not be derived from cask metadata are flagged in the relevant PR.
