#!/usr/bin/env node
/**
 * Checks the invariants that keep Fleet-maintained-app icons out of the JS
 * bundle. They are served as individual static files, so a page downloads only
 * the icons it renders — which holds as long as every icon stays a small,
 * quantized PNG that the barrel actually references.
 *
 * Findings are warnings and do not fail the build — an icon that is merely
 * unoptimized should not block a PR. The one exception is a base64 image
 * inlined into a .tsx: that puts an icon back into the entry bundle for every
 * page load, silently undoing the reason these are static files, so it fails.
 * Pass --strict to fail on every finding.
 *
 * Under GitHub Actions the findings are also emitted as warning annotations, so
 * they appear on the offending file in the PR diff rather than only in the log.
 *
 * Run: yarn lint:icons
 */
const fs = require("fs");
const path = require("path");

const REPO_ROOT = path.join(__dirname, "../../..");
const ICONS = path.join(
  REPO_ROOT,
  "frontend/pages/SoftwarePage/components/icons"
);
const PNG_DIR = path.join(ICONS, "png");
const FRONTEND = path.join(REPO_ROOT, "frontend");
const BARREL = path.join(ICONS, "index.ts");

const EXPECTED = { width: 128, height: 128 };
const MAX_BYTES = 16 * 1024;

// Vendors that ship nothing larger. Upscaling would only add blur at the same
// byte cost, so these are allowed below the standard size rather than faked up.
const SMALL_SOURCE_ALLOWLIST = new Set([
  "Gnupg.png",
  "SonicwallNetextender.png",
  "MicrosoftOleDbDriver19.png",
]);

const STRICT = process.argv.includes("--strict");
const ANNOTATE = Boolean(process.env.GITHUB_ACTIONS);

const findings = [];
const rel = (abs) => path.relative(REPO_ROOT, abs).split(path.sep).join("/");
const report = (where, message, blocking = false, line = null) =>
  findings.push({ where, message, blocking, line });
const pngPath = (file) => rel(path.join(PNG_DIR, file));
const barrelPath = rel(BARREL);

const src = fs.readFileSync(BARREL, "utf8");

// --- barrel <-> disk --------------------------------------------------------
const pngImports = new Map();
for (const m of src.matchAll(/^import (\w+) from "\.\/png\/([^"]+)";$/gm)) {
  pngImports.set(m[1], m[2]);
}
const vectorImports = new Set(
  [...src.matchAll(/^import (\w+) from "\.\/([^"/]+)";$/gm)].map((m) => m[1])
);
const onDisk = fs.readdirSync(PNG_DIR).filter((f) => f.endsWith(".png"));

for (const [ident, file] of pngImports) {
  if (!onDisk.includes(file)) {
    report(
      barrelPath,
      `${file}: imported as ${ident} but the file does not exist. ` +
        `The webpack build and tsc fail on this regardless of this check.`
    );
  }
}
const referenced = new Set(pngImports.values());
for (const file of onDisk) {
  if (!referenced.has(file)) {
    report(
      pngPath(file),
      `${file}: on disk but never imported by index.ts, so webpack never ` +
        `emits it — dead weight in git.`
    );
  }
}

// Every icon map value must resolve to one of those imports.
const known = new Set([...pngImports.keys(), ...vectorImports]);
for (const map of src.matchAll(
  /export const \w+_TO_ICON_MAP = \{([\s\S]*?)\n\} as const;/g
)) {
  for (const e of map[1].matchAll(
    /^\s*(?:"[^"]*"|[\w$]+)\s*:\s*(\w+),\s*$/gm
  )) {
    if (!known.has(e[1])) {
      report(
        barrelPath,
        `map value ${e[1]} is not imported. tsc fails on this regardless.`
      );
    }
  }
}

// --- each PNG --------------------------------------------------------------
const SIG = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
const PALETTE = 3;

for (const file of onDisk) {
  const buf = fs.readFileSync(path.join(PNG_DIR, file));
  if (!buf.subarray(0, 8).equals(SIG)) {
    report(pngPath(file), `${file}: not a PNG.`);
    continue;
  }
  const width = buf.readUInt32BE(16);
  const height = buf.readUInt32BE(20);
  const colorType = buf[25];

  if (colorType !== PALETTE) {
    report(
      pngPath(file),
      `${file}: color type ${colorType}, expected ${PALETTE} (palette) — ` +
        `pngquant was not run, so this icon is roughly 4x larger than it ` +
        `needs to be. See tools/software/icons/README.md`
    );
  }
  if (buf.length > MAX_BYTES) {
    report(
      pngPath(file),
      `${file}: ${(buf.length / 1024).toFixed(1)} KB exceeds the ${
        MAX_BYTES / 1024
      } KB ceiling — every icon is fetched individually.`
    );
  }
  if (
    (width !== EXPECTED.width || height !== EXPECTED.height) &&
    !SMALL_SOURCE_ALLOWLIST.has(file)
  ) {
    report(
      pngPath(file),
      `${file}: ${width}x${height}, expected ${EXPECTED.width}x${EXPECTED.height} ` +
        `(2x the largest size the UI renders). If the vendor ships nothing ` +
        `bigger, add it to SMALL_SOURCE_ALLOWLIST in this script.`
    );
  }
}

// --- no going back to base64 -----------------------------------------------
// Scanned recursively, and across the whole frontend rather than just the icon
// directory: an inlined image is bundle weight wherever it lives. Only the icon
// directory blocks, because that is unambiguously an FMA icon regression —
// elsewhere an inlined image may be deliberate, so it warns instead.
const SOURCE_EXT = /\.(tsx|ts|jsx|js)$/;
const sourceFiles = fs
  .readdirSync(FRONTEND, { recursive: true })
  .map((f) => path.join(FRONTEND, String(f)))
  .filter((f) => SOURCE_EXT.test(f));

for (const abs of sourceFiles) {
  const body = fs.readFileSync(abs, "utf8");
  const hit = /data:image\/[a-z+]+;base64,/.exec(body);
  if (hit) {
    const file = path.basename(abs);
    const inIconDir = abs.startsWith(ICONS + path.sep);
    // Source files have real lines, so point at the offending one — that lets
    // the annotation render inline on the diff, not just in the run summary.
    const line = body.slice(0, hit.index).split("\n").length;
    report(
      rel(abs),
      inIconDir
        ? `${file}: embeds a base64 image. App icons belong in ` +
            `${rel(PNG_DIR)} as static files — inlining one puts it back ` +
            `into the entry bundle for every page load, which is the thing ` +
            `moving them out of JS fixed.`
        : `${file}: embeds a base64 image, so it ships in the entry bundle ` +
            `on every page load. Import the file instead and let webpack ` +
            `emit it, so only pages that render it pay for it.`,
      inIconDir,
      line
    );
  }
}

// ---------------------------------------------------------------------------
const summary = `${onDisk.length} icons, ${vectorImports.size} vector components`;

if (!findings.length) {
  console.log(`icon check passed: ${summary}`);
  process.exit(0);
}

const blocking = findings.filter((f) => f.blocking);

if (ANNOTATE) {
  for (const { where, message, blocking: b, line } of findings) {
    const at = line ? `file=${where},line=${line}` : `file=${where}`;
    console.log(`::${b ? "error" : "warning"} ${at}::${message}`);
  }
}

console.log(`\nicon check: ${findings.length} finding(s) across ${summary}\n`);
for (const { where, message, blocking: b, line } of findings) {
  console.log(`  ${b ? "x" : "!"} ${message}`);
  console.log(`    ${where}${line ? `:${line}` : ""}`);
}

const failing = blocking.length > 0 || STRICT;
if (blocking.length) {
  console.log(
    `\n${blocking.length} finding(s) marked x block the build: an inlined ` +
      `icon silently undoes the reason these are static files. The rest are ` +
      `warnings.\n`
  );
} else {
  console.log(
    STRICT
      ? "\nFailing because --strict was passed.\n"
      : "\nThese are warnings and do not block the PR. Fixing them keeps each " +
          "page load paying only for the icons it actually shows.\n"
  );
}
process.exit(failing ? 1 : 0);
