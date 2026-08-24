# Changelog

## 2026-08-15

- Replace the minimal id-shift SQL with a full rebuild of row ids into version
  order (rebase above `MAX(id)`, then compact to `1..N`). This fixes down-moves
  whose remapped rows sit at the table tail (the shift offset previously
  computed to NULL and silently no-opped), supports up-moves and down-moves in
  the same scope (previously collapsed into a single direction), and removes
  the stale `AUTO_INCREMENT` collision risk after shifts.
- Process rename commits oldest-first so chained renames (A -> B in one
  commit, B -> C in a later one) resolve rows to the terminal version ID
  instead of an intermediate one.
- Rewrite the dry-run simulator to mirror the generated SQL exactly
  (sequential remaps, dedup, sort-and-renumber) and report how many ordering
  violations the rebuild fixes.

## 2026-08-07

- Add `--since-commit <ref>` to scan `main` from a given commit forward.
- Add `--commit <ref>` to inspect a single commit for renames.
- Exactly one of `--branch`, `--since-commit`, or `--commit` is now required;
  providing more than one is an error.
