# Go binary vulnerability detection

This package detects vulnerabilities in Go binaries (`software.source = 'go_binaries'`) using the
[Go vulnerability database](https://vuln.go.dev), the same data source `govulncheck` uses.

Go binaries are excluded from NVD CPE matching. A binary's name says nothing about the module it
was built from, so the generic CPE path gave every binary named `air` the Adobe AIR CPE and all
188 of its CVEs. The Go database is keyed on module path instead, which is unambiguous.

## What is matched

Two independent lookups run per software row:

| Lookup | Key | Compared against | Source column |
|---|---|---|---|
| Module advisories | Go module path | the binary's own version | `software.extension_id` |
| Standard-library advisories | `stdlib` | the Go toolchain the binary was built with | `software.release` |

Standard-library advisories apply regardless of the binary's own version: the fix is a rebuild
with a newer toolchain. The match is toolchain-level. The `go_binaries` table does not report
which standard-library packages a binary links, so a binary built with an affected toolchain is
flagged whether or not it uses the affected package, and stdlib matches over-report compared to
`govulncheck`'s symbol-level analysis. The toolchain version is
canonicalized the way govulncheck does it: a GOEXPERIMENT suffix (`go1.21.12 X:boringcrypto`) is
dropped, and release candidates and betas become semver prereleases (`go1.22rc1` is
`v1.22.0-rc.1`), which the database's `1.22.0-0` lower bounds are written to order against.
Module versions keep their pseudo-version form (`v0.0.0-20240101000000-abcdef123456`); an
`introduced` bound of `0` covers them.

`resolved_in_version` is set only for module advisories, where the fix really is a newer release
of that software. Standard-library matches leave it unset — the UI renders it as the version to
upgrade the software to, and a Go toolchain version is not that.

## What is skipped

- **Toolchain advisories.** They affect the `go` command on the build machine, not the binary.
- **Reports with no CVE alias.** `software_cve.cve` and every CVE metadata join are keyed on a
  CVE ID. Go reports that only carry a GHSA alias are not represented.
- **`(devel)` versions.** A binary built from a working tree has no module release to compare.
- **Empty module paths.** Binaries built outside module mode report no `Main.Path`.
- **Development toolchains** (`devel go1.27-abcdef ...`). There is no version to compare.
- **Dependency-level advisories.** They need the binary's `Deps` list, which the fleetd
  `go_binaries` table does not emit.

CVE metadata (scores, descriptions, KEV) comes from the existing NVD feed. A Go report whose CVE
NVD has not published yet appears without scores until it catches up.

## Data flow

Fleet never fetches `vuln.go.dev` directly. A publisher job in
[fleetdm/vulnerabilities](https://github.com/fleetdm/vulnerabilities) pulls the database and
publishes it as a release asset, which keeps a single egress point for restricted deployments and
matches how OSV data is mirrored. `Refresh` downloads the newest asset into the server's
`databases_path`, verifies its digest, and deletes the ones it supersedes.

## Artifact contract

The publisher writes one gzipped JSON file per database snapshot, named
`govulndb-YYYY-MM-DD.json.gz`:

```json
{
  "schema_version": "1",
  "generated": "2026-09-09T00:00:00Z",
  "modules": {
    "github.com/example/tool": [
      {
        "id": "GO-2024-1234",
        "cves": ["CVE-2024-1234"],
        "ranges": [{ "introduced": "0", "fixed": "1.49.0" }]
      }
    ],
    "stdlib": [
      {
        "id": "GO-2024-2963",
        "cves": ["CVE-2024-24791"],
        "ranges": [
          { "introduced": "0", "fixed": "1.21.12" },
          { "introduced": "1.22.0-0", "fixed": "1.22.5" }
        ]
      }
    ]
  }
}
```

Requirements on the publisher:

- Keys are module paths exactly as the Go database reports them, including any `/v2` suffix, plus
  `stdlib`. Do not emit `toolchain`.
- `cves` holds the report's CVE aliases. Omit reports that have none.
- `ranges` pairs up the OSV `affected[].ranges[].events[]` list: each `introduced` with the
  `fixed` that follows it, and no `fixed` when the report has no fix yet. A range is half-open:
  `introduced <= v < fixed`.
- Versions carry **no** leading `v`, matching the database (`1.21.12`, not `v1.21.12`).
- `schema_version` is `"1"`. The analyzer rejects any other value, so a format change needs a
  new version and a matching server release before it is published.
- An artifact with an empty `modules` map is rejected by the analyzer rather than read as every
  advisory having been remediated, so a partial run must not be published.

## Testing

```bash
go install helm.sh/helm/v3/cmd/helm@v3.6.0    # module advisories (GO-2022-0384, fixed in 3.6.1)
go install github.com/air-verse/air@v1.48.0   # the known false positive: no Adobe AIR CVEs
go build -o ~/go/bin/devtool ./cmd/fleetctl   # (devel) row with a module path
fleetctl trigger --name vulnerabilities
```

Any binary built with an out-of-date toolchain exercises the standard-library path; check the
Go version a binary reports with `go version -m <path>`.

To see what the upstream database says about a module:

```bash
curl -s https://vuln.go.dev/index/modules.json | jq '.[] | select(.path=="helm.sh/helm/v3")'
curl -s https://vuln.go.dev/ID/GO-2022-0384.json | jq '{aliases, affected}'
```

```sql
SELECT s.name, s.version, s.release, sc.cve, sc.source, sc.resolved_in_version
FROM software_cve sc JOIN software s ON s.id = sc.software_id
WHERE s.source = 'go_binaries';
```
