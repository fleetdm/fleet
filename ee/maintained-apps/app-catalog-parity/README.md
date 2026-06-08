# App catalog parity

Tracking work to ensure Fleet has a [Fleet-maintained app (FMA)](../README.md) for every macOS app on [appcatalog.cloud](https://appcatalog.cloud/apps) that ships a Homebrew cask.

## Snapshot

| Metric | Count |
|---|---:|
| Apps on appcatalog.cloud (sitemap) | 1,653 |
| …matched to a Homebrew cask | 1117 slugs → 1092 unique casks |
| …already a Fleet-maintained app | 213 |
| **…cask exists, FMA missing (work queue)** | **879** |
| Apps with no Homebrew cask (handoff) | 533 |

## Files

- [to-add.md](to-add.md) — the FMA work queue, grouped A–Z. Worked one letter per commit.
- [no-homebrew-cask.md](no-homebrew-cask.md) — apps lacking a Homebrew cask, for separate upstream submission.

## Methodology

1. All app URLs pulled from `https://appcatalog.cloud/sitemap.xml` (the live page only prerenders 100).
2. The full Homebrew cask list (`https://formulae.brew.sh/api/cask.json`) is the source of truth for whether a cask exists — not the catalog's own labels.
3. A catalog slug matches a cask by: exact token, token with a trailing 5-hex dedupe suffix stripped, normalized display-name match, or version-suffix stripped (`acorn-7` → `acorn`).
4. The work queue is the matched casks **not** already present in `inputs/homebrew/`.

FMAs are added one alphabet letter per commit. Identity fields are derived from cask metadata (fast path) and confirmed by the CI validator; bundle IDs that could not be derived from cask metadata are flagged in the relevant PR.
