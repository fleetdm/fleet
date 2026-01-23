# Vulnerability scanning for inlined dependencies

This directory contains manifest files (`go.mod`, `package.json`) that list the third-party dependencies that have been copied/inlined into Fleet's codebase.

## Purpose

Fleet has several dependencies that were copied directly into the repository rather than imported via Go modules or npm. These inlined dependencies are not automatically scanned by vulnerability detection tools like GitHub Dependabot because they don't appear in the main `go.mod` or `package.json`.

This directory solves that problem by creating "dummy" manifest files that list these dependencies at their copied versions. This allows:

- **GitHub Dependabot** to detect vulnerabilities and create alerts
- **govulncheck** to scan for Go vulnerabilities
- **npm audit** to scan for JavaScript vulnerabilities
- Other security scanning tools to identify issues

## Important notes

1. **This code is NOT compiled into Fleet** - These manifest files exist solely for vulnerability scanning
2. **Keep versions in sync** - When updating an inlined dependency, update the version here to match
3. **No Go code here** - Do not add any `.go` files to this directory

## Tracked dependencies

### Go dependencies (go.mod)

| Dependency                 | Fleet Location                        | Version                            |
|----------------------------|---------------------------------------|------------------------------------|
| micromdm/nanomdm           | server/mdm/nanomdm/                   | v0.9.0                             |
| micromdm/nanodep           | server/mdm/nanodep/                   | v0.4.0                             |
| micromdm/scep/v2           | server/mdm/scep/                      | v2.3.0                             |
| pressly/goose/v3           | server/goose/                         | v3.17.0                            |
| facebookincubator/nvdtools | server/vulnerabilities/nvd/tools/     | v0.1.5                             |
| virtuald/go-paniclog       | orbit/pkg/go-paniclog/                | v0.0.0-20190812204905-43a7fa316459 |
| josharian/impl             | server/mock/mockimpl/                 | v1.4.0                             |
| mitchellh/gon              | orbit/pkg/packaging/macos_notarize.go | v0.2.3                             |
| sassoftware/relic          | pkg/file/xar.go                       | v7.2.1+incompatible                |
| oscartbeaumont/windows_mdm | tools/mdm/windows/poc-mdm-server/     | v0.0.0-20210615145659-e52e28e50db7 |

### npm dependencies (package.json)

| Dependency      | Fleet Location                      | Version |
|-----------------|-------------------------------------|---------|
| node-sql-parser | frontend/utilities/node-sql-parser/ | 5.3.13  |

## Running vulnerability scans locally

### Go vulnerabilities

```bash
cd third_party/vuln-check
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### npm vulnerabilities

```bash
cd third_party/vuln-check
npm audit
```

## Related documentation

- [ADR-0004: Third-party library vendoring](../../docs/Contributing/adr/0004-third-party-vendoring.md)
- [GitHub Issue #31605](https://github.com/fleetdm/fleet/issues/31605)
