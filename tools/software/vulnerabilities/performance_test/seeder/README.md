> [!NOTE]
> **Prefer [`dibble`](../../../../dibble/README.md) for seeding vulnerable
> software at volume.** The equivalent is:
> ```bash
> ./tools/dibble/dibble vulns --macos 100000 --ubuntu 100000 --windows 100000
> ```
> This script is kept for backwards compatibility. It's a higher-volume
> sibling of `tools/software/vulnerabilities/seed_data/seed_vuln_data.go`
> with deadlock-retry logic specific to vuln performance testing.

# Vulnerability performance-test seeder

Bulk MySQL seeder used by the vulnerability performance test harness
(`tools/software/vulnerabilities/performance_test/tester/`). Writes very large
numbers of software rows directly to MySQL with deadlock retries.

## Usage

```bash
go run ./tools/software/vulnerabilities/performance_test/seeder
```

See `volume_vuln_seeder.go` for configuration constants.
