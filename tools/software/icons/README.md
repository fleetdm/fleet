# Creating Icons for Fleet-maintained Apps

App icons for the Fleet server and [fleetdm.com software catalog](https://fleetdm.com/software-catalog) can be generated using the following script on macOS using an associated `.app` bundle.

## Usage

```bash
bash tools/software/icons/generate-icons.sh -s slug-name [-a /path/to/App.app | -i /path/to/icon.png]
```

- `-s`: Slug name for the Fleet-maintained app (required). The portion before the slash will be used in the output filenames.
- `-a`: Path to the `.app` bundle (e.g. `/Applications/Safari.app`). Required if `-i` is not provided.
- `-i`: Path to a PNG icon file. Required if `-a` is not provided. The icon will be resized to 128x128 if larger.

## Examples

Using an app bundle:
```bash
bash tools/software/icons/generate-icons.sh -a /Applications/Google\ Chrome.app -s "google-chrome/darwin"
```

Using a PNG file directly:
```bash
bash tools/software/icons/generate-icons.sh -i /path/to/icon.png -s "company-portal/windows"
```

This will generate two files:

- `frontend/pages/SoftwarePage/components/icons/png/GoogleChrome.png` – the 128x128 icon served to the app
- `website/assets/images/app-icon-google-chrome-60x60@2x.png` – the 128x128 PNG used on the website

## Notes

- App icons are plain PNGs that webpack emits as individual static files, so a page downloads only
  the icons it renders. Do not add them as base64-in-TSX components — that puts every icon in the
  entry bundle for every page load.
- Icons are quantized with `pngquant` if it is installed (`brew install pngquant`).
- The filename is derived from the app's name (e.g. `Google Chrome.app` → `GoogleChrome.png`).
  Keep it alphanumeric — the filename becomes part of a URL.
- The script ensures consistent formatting and naming conventions across icon entries.
- **The script automatically adds the import statement and map entry to `frontend/pages/SoftwarePage/components/icons/index.ts`**, so you don't need to manually update the index file. The app name used in the map is extracted from the app's `Info.plist` (`CFBundleName` or `CFBundleDisplayName`).

## Icon requirements

`yarn lint:icons` checks these. Locally it runs as part of `make lint-js`; in CI it is its own
`lint-icons` job, and reports findings as annotations. Findings are **warnings and do not fail the build** — an
unoptimized icon is worth flagging, not worth blocking a PR over. Pass `--strict` to fail on any
finding locally.

It is deliberately not part of the CI `lint-js` job as well — running it in both places annotates
every finding twice and fails two jobs for one problem.

The single exception is the last row below: a base64 image inlined into a `.tsx` **does** fail CI.
Everything else costs some extra bytes on the pages that show that icon; an inlined icon costs
every page load, and nothing else in the build would catch it.

| Requirement | Why |
|---|---|
| **128×128** | Twice the largest size the UI renders (`large`, 64 px), so it stays sharp on retina. |
| **Palette PNG** (color type 3) | This is what `pngquant` produces, so it is the signal that quantization ran. An unquantized 128×128 RGBA icon is ~12 KB instead of ~3 KB. |
| **Under 16 KB** | Every icon is a separate request. The current set averages 3.3 KB and peaks at 10.3 KB. |
| **Imported by `index.ts`** | An icon on disk that nothing imports is never emitted — it is dead weight in git. |
| **No base64-inlined images** — *fails CI inside the icon directory* | Guards the whole arrangement: one inlined icon is one icon in every page load. Scanned recursively across `frontend/`; a hit outside the icon directory warns rather than fails, since it may be deliberate there. |

Two of these also fail the build on their own regardless of this check: an import with no matching
file, and a map value that is not imported. Those break webpack and `tsc`.

If a vendor genuinely ships nothing larger than 128×128, add the filename to
`SMALL_SOURCE_ALLOWLIST` in `check-icons.js` rather than upscaling — an upscaled icon is just a
blurrier icon at the same byte cost. Two are allowlisted today (`Gnupg`, `SonicwallNetextender`,
both 48×48).

### A note on color profiles

`pngquant` converts wide-gamut icons (Display P3, an `iCCP`/`cICP` chunk) to sRGB — it does this
even with `--strip`. About 80 icons came from P3 sources, and on a P3 display they render very
slightly less saturated than the original. This is deliberate: the icon no longer depends on the
browser honoring an embedded profile, and preserving P3 for those would cost ~850 KB across the
set.
