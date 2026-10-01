# ai_tools (vendored)

This package provides the fleetd `ai_tools` osquery table: a unified inventory
of AI software (desktop apps, IDE plugins, agent CLIs, MCP servers, live AI/MCP
sockets, agent instruction files, and browser extensions) with a `type`
discriminator and per-row `risk_flags`, `sha256`, and JSON `detail` columns.

## Provenance

The source under this directory is **vendored** (copied into the tree), not
imported as a Go module dependency.

- Upstream: https://github.com/karmine05/agentic-detector
- Version: tag `v0.3.0`, commit `7c942d0`
- Imported: 2026-07 (into `orbit/pkg/table/ai_tools/`)

The upstream `tables` package was renamed to `ai_tools`, and the import prefix
`github.com/karmine05/agentic-detector/` was rewritten to
`github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/`. `ai_tools.go` adds the
exported `Columns()`/`Generate()` wrappers used to register the table in
`orbit/pkg/table/extension.go`.

### Modifications beyond the mechanical import

- **Lint compliance** with Fleet's linters (set types, modernize idioms,
  defensive nil guards); all behavior-preserving.
- **Windows app detection** rewritten for a daemon running as SYSTEM, where
  upstream's `HKEY_CURRENT_USER` read resolves to SYSTEM's own empty hive and
  finds nothing: the apps collector walks real users' loaded hives under
  `HKEY_USERS` for per-user uninstall entries, and additionally scans the
  MSIX/Appx install root (`%ProgramFiles%\WindowsApps`), which no uninstall key
  covers. Per-user package directories are read only to attribute scope, never to
  report an app: they outlive an uninstall, so a row sourced from one would
  assert an install that is no longer there.
- **Security hardening** for running in-process in the root/SYSTEM orbit daemon:
  regular-file-only reads that never follow symlinks or block on FIFOs/devices
  (`internal/fsutil`), path-traversal containment for attacker-controlled
  config/manifest/plist fields, removal of outbound DNS resolution of untrusted
  MCP hostnames (`internal/netsock`), OS-attested (never name-based) uid/username
  attribution cross-checked against on-disk ownership (`internal/homes`), and
  panic recovery at the `Generate` boundary.

## License

Upstream is MIT licensed (`Copyright (c) 2026 Karmine`), added upstream in commit
`e494314`. The license is vendored alongside this code at
`orbit/pkg/table/ai_tools/LICENSE`.
