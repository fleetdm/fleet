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
- **Security hardening** for running in-process in the root/SYSTEM orbit daemon:
  regular-file-only reads that never follow symlinks or block on FIFOs/devices
  (`internal/fsutil`), path-traversal containment for attacker-controlled
  config/manifest/plist fields, removal of outbound DNS resolution of untrusted
  MCP hostnames (`internal/netsock`), owner-based (not name-based) uid/username
  attribution (`internal/homes`), and panic recovery at the `Generate` boundary.

## License

⚠️ At the vendored commit (`7c942d0`), the upstream repository contained **no
LICENSE file** and no license header in its source. Redistributing it therefore
has no explicit grant. This must be resolved with the author before this code
ships (e.g. obtain an explicit license, or a written contribution grant).
