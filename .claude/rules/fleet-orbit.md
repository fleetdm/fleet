---
paths:
  - "orbit/**/*.go"
---

# Fleet Orbit conventions

Orbit is Fleet's lightweight agent that manages osquery, handles updates, and provides device-level functionality. It runs on end-user devices, so reliability and security are critical.

## Architecture
- **Entry point**: `orbit/cmd/orbit/` — main binary
- **Packages**: `orbit/pkg/` — modular packages for each concern
- **Update system**: `orbit/pkg/update/` — TUF-based auto-update for osquery, orbit, and desktop
- **Packaging**: `orbit/pkg/packaging/` — builds installers for macOS (.pkg), Windows (.msi), and Linux (.deb/.rpm)
- **Platform-specific code**: use build tags (`_darwin.go`, `_windows.go`, `_linux.go`) and `_stub.go` for unsupported platforms

## Key patterns
- **Keystore**: `orbit/pkg/keystore/` — platform-specific secure key storage (macOS Keychain, Windows DPAPI, Linux file-based). Always use the keystore abstraction, never raw file I/O for secrets.
- **osquery management**: `orbit/pkg/osquery/` — launching, monitoring, and communicating with osquery. Orbit owns the osquery lifecycle.
- **Token management**: `orbit/pkg/token/` — orbit enrollment token read/write with file locking
- **Platform executables**: `orbit/pkg/execuser/` — run commands as the logged-in user (not root). Critical for UI prompts and desktop app.

## Security considerations
- Orbit runs as root/SYSTEM — every input must be validated
- Never log enrollment tokens, orbit keys, or device identifiers at info level
- File operations on device should use restrictive permissions (0600/0700)
- TUF update verification must never be bypassed
- Use `orbit/pkg/insecure/` only for intentionally insecure test configurations

## Testing
- Unit tests don't need special env vars (no MySQL/Redis)
- Platform-specific tests may need build tags: `go test -tags darwin ./orbit/pkg/...`
- Use `_stub.go` files for cross-platform test compatibility
- Packaging tests may require signing certificates or specific tools (notarytool, WiX)

## Build and packaging
- macOS: `.pkg` built with `pkgbuild`, optional notarization via `notarytool` or `rcodesign`
- Windows: `.msi` built with WiX toolset, templates in `orbit/pkg/packaging/windows_templates.go`
- Linux: `.deb` and `.rpm` via `nfpm`
- Cross-compilation: orbit supports `GOOS`/`GOARCH` targeting
