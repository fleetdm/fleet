# App catalog parity

Tracking work to ensure Fleet has a [Fleet-maintained app (FMA)](../README.md) for every macOS app on [appcatalog.cloud](https://appcatalog.cloud/apps) that ships a Homebrew cask.

## Snapshot

| Metric | Count |
|---|---:|
| Apps on appcatalog.cloud (sitemap) | 1,653 |
| …matched to a Homebrew cask | 1117 slugs → 1092 unique casks |
| …already a Fleet-maintained app | 213 |
| …cask exists, FMA missing (work queue) | 879 |
| **…added as FMAs (A–Z + #)** | **760** |
| …since added to `main` independently (deduplicated on merge) | 385 |
| **…net-new FMAs contributed by this PR** | **375** |
| …removed after CI validation failure | 78 |
| …deferred (see below) | 41 |
| Apps with no Homebrew cask (handoff) | 533 |

## Status: complete

All 879 work-queue casks have been processed, one letter per commit (A–Z plus the `#` numeric-leading bucket). **838 were initially added as Fleet-maintained apps; 41 were deferred.** A subsequent run of the Darwin FMA validator (CI) flagged 78 of the added apps as failing, so they were **removed for now** — leaving **760 added**.

**Merge with `main` (deduplication).** While this branch was open, **385** of those 760 casks were independently added to `main` by other PRs (including the macOS "R" batch, [#47370](https://github.com/fleetdm/fleet/pull/47370)). On merging `main`, those casks were dropped from this branch in favor of `main`'s versions, so this PR now contributes **375 net-new FMAs**. The 385 remain Fleet-maintained apps — they're just sourced from `main` rather than this branch — so they stay checked in [to-add.md](to-add.md). (`sonos` is one of these: `main` ships it as "Sonos" under `com.sonos.macController2`, so this branch's divergent "Sonos S2" copy was reconciled to `main`'s version; the net-new `sonos-s1-controller` stays.) One pre-existing app, Dell Display Manager, was **removed on `main`** ([#47420](https://github.com/fleetdm/fleet/pull/47420)) and that removal is honored here. `libreoffice` is kept on `main`'s `libreoffice-still` cask (this branch had switched it to the `libreoffice` fresh channel).

Remaining open items:

- **78 apps removed after CI validation failure** — unchecked in [to-add.md](to-add.md) and annotated with the failure category. Categories: uninstall didn't remove the app, version not found by osquery (identity/version mismatch), download blocked (4xx), installer failed (bad dmg/script), invalid character in app name, download server error (5xx), download timed out, and download host unresolved (DNS). Several (5xx, timeout, DNS) may be transient and worth re-validating before re-adding. Note: the same CI run also suffered an infrastructure failure (the `osquery` app's uninstall removed `osqueryi` from PATH, then the runner ran out of disk) that produced ~160 spurious "osqueryi not found" / "no space left on device" errors for unrelated apps — those apps were **not** removed and should be re-validated.

- **Descriptions are auto-generated** from each cask's Homebrew `desc` and need a human review/polish pass before these ship in a PR.
- **Icons are generated** for the added apps by pulling each app's icon from [appcatalog.cloud](https://appcatalog.cloud/apps) via `tools/software/icons/fetch-appcatalog-icons-batch.sh` (see that directory's README). A handful of apps have no appcatalog icon and still need one generated from the `.app` bundle — they're marked `-` in `tools/software/icons/appcatalog-slug-overrides.tsv`. These are third-party-hosted vendor icons, so they warrant a review pass before shipping.
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
