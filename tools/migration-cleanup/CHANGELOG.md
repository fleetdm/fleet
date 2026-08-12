# Changelog

## 2026-08-07

- Add `--since-commit <ref>` to scan `main` from a given commit forward.
- Add `--commit <ref>` to inspect a single commit for renames.
- Exactly one of `--branch`, `--since-commit`, or `--commit` is now required;
  providing more than one is an error.
