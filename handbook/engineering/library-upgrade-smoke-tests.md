# Library upgrade smoke tests

When upgrading a third-party dependency, use this document to find the tests you need to run. Libraries are grouped by functional area. Find your library, run the listed tests, and confirm everything passes before merging.

> **How to use:** Press Ctrl+F / Cmd+F and search for the library name (e.g. `gorilla/mux`). Each section lists the test commands to run. The "Tested by" column shows which test suite covers the library.

> **Maintained as part of:** [#48943](https://github.com/fleetdm/fleet/issues/48943). When adding a new dependency, add it to the appropriate section below.

### Legend

| Tested by | Meaning |
|-----------|---------|
| Test command (e.g. `go test ./...`) | Covered by the named test suite |
| Indirect | Exercised through integration tests but no dedicated unit test |
| Build | Verified by successful compilation only |
| Gap | No automated test coverage -- manual verification required |

---

## Server + fleetctl + Orbit (`go.mod`)

### HTTP / Routing / Middleware

go-kit and gorilla/mux are the backbone of every API request. If either breaks, nothing works.

```
go test ./server/service/...
go test ./server/platform/...
MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/integration_*_test.go
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/go-kit/kit` | v0.12.0 | Critical | `go test ./server/service/...` (123 test files) |
| `github.com/gorilla/mux` | v1.8.1 | Critical | `go test ./server/service/...` (integration tests hit all routes) |
| `github.com/gorilla/websocket` | v1.5.1 | High | `go test ./server/service/ -run Campaign` |
| `github.com/igm/sockjs-go/v3` | v3.0.2 | High | `go test ./server/service/ -run Campaign` |
| `github.com/throttled/throttled/v2` | v2.8.0 | High | `go test ./server/platform/middleware/ratelimit/...` |
| `github.com/realclientip/realclientip-go` | v1.0.0 | Medium | `go test ./server/platform/endpointer/...` |
| `github.com/e-dard/netbug` | v0.0.0 | Low | Build |

### Database / SQL

go-sql-driver/mysql and jmoiron/sqlx underpin every database operation.

```
MYSQL_TEST=1 go test ./server/datastore/mysql/...
go test ./server/platform/mysql/...
go test ./server/vulnerabilities/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/go-sql-driver/mysql` | v1.9.3 | Critical | `MYSQL_TEST=1 go test ./server/datastore/mysql/...` (112 test files) |
| `github.com/jmoiron/sqlx` | v1.3.5 | Critical | `MYSQL_TEST=1 go test ./server/datastore/mysql/...` (112 test files) |
| `github.com/doug-martin/goqu/v9` | v9.18.0 | High | `MYSQL_TEST=1 go test ./server/datastore/mysql/ -run TestSoftware` |
| `github.com/ngrok/sqlmw` | v0.0.0 | Medium | `go test ./server/platform/mysql/...` |
| `github.com/VividCortex/mysqlerr` | v0.0.0 | Medium | `go test ./server/datastore/mysql/ -run TestDuplicate` |
| `github.com/ziutek/mymysql` | v1.5.4 | Low | Build (`go build ./server/goose/cmd/goose/`) |
| `github.com/mattn/go-sqlite3` | v1.14.22 | High | `go test ./server/vulnerabilities/...` (74 test files) |
| `github.com/lib/pq` | v1.10.9 | Low | Build |
| `github.com/shogo82148/rdsmysql/v2` | v2.5.0 | Medium | Build (AWS RDS deployments only) |
| `github.com/XSAM/otelsql` | v0.39.0 | Low | Build (active when OTel enabled) |
| `github.com/DATA-DOG/go-sqlmock` | v1.5.0 | Low | `go test ./server/platform/mysql/...` (test-only lib) |

### Redis / Caching

redigo is used by virtually every cross-server feature: live queries, SSO, host check-ins, MDM, automations.

```
REDIS_TEST=1 go test ./server/datastore/redis/...
go test ./server/datastore/cached_mysql/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/gomodule/redigo` | v1.8.9 | Critical | `REDIS_TEST=1 go test ./server/datastore/redis/...` (4 test files) |
| `github.com/mna/redisc` | v1.3.2 | High | `REDIS_TEST=1 go test ./server/datastore/redis/...` |
| `github.com/patrickmn/go-cache` | v2.1.0 | High | `go test ./server/datastore/cached_mysql/...` (1 test file) |

### MDM (Apple / Microsoft / SCEP)

MDM libraries are security-critical. Always test enrollment end-to-end after upgrading.

```
go test ./server/mdm/...
MYSQL_TEST=1 REDIS_TEST=1 go test ./ee/server/service/integration_mdm_test.go
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/micromdm/micromdm` | v1.9.0 | High | `go test ./server/mdm/...` (87 test files) |
| `github.com/micromdm/nanolib` | v0.2.0 | Medium | `go test ./server/mdm/nanomdm/...` |
| `github.com/micromdm/plist` | v0.2.3 | High | `go test ./server/mdm/nanomdm/...` |
| `github.com/groob/plist` | v0.0.0 | Low | Indirect (Orbit extension table) |
| `howett.net/plist` | v1.0.1 | High | `go test ./server/mdm/...` + `go test ./orbit/pkg/...` |
| `github.com/smallstep/scep` | v0.0.0 | Critical | `go test ./server/mdm/scep/...` + MDM integration tests |
| `github.com/smallstep/pkcs7` | v0.0.0 | Critical | `go test ./server/mdm/...` |
| `go.step.sm/crypto` | v0.77.1 | High | `go test ./server/mdm/acme/...` |
| `software.sslmate.com/src/go-pkcs12` | v0.4.0 | Medium | `go test ./ee/server/service/ -run DigiCert` |
| `github.com/RobotsAndPencils/buford` | v0.14.0 | Critical | MDM integration tests (APNs push) |
| `github.com/fxamacker/cbor/v2` | v2.9.1 | Medium | `go test ./server/mdm/acme/...` |

### Authentication / Security / Crypto

These handle SSO, JWT tokens, password hashing, and host identity verification.

```
go test ./server/sso/...
go test ./server/mdm/microsoft/...
go test ./ee/server/service/ -run SSO
go test ./ee/server/service/ -run CondAccess
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/crewjam/saml` | v0.5.1 | Critical | `go test ./server/sso/...` (6 test files) + SSO integration tests |
| `github.com/russellhaering/goxmldsig` | v1.6.0 | High | `go test ./server/sso/...` (via SAML) |
| `github.com/golang-jwt/jwt/v4` | v4.5.2 | High | `go test ./server/mdm/microsoft/...` + license tests |
| `github.com/MicahParks/jwkset` | v0.11.0 | Medium | `go test ./server/mdm/microsoft/...` |
| `github.com/gomodule/oauth1` | v0.2.0 | High | MDM integration tests (DEP API auth) |
| `golang.org/x/oauth2` | v0.35.0 | High | `go test ./ee/server/calendar/...` + `go test ./server/datastore/s3/...` |
| `golang.org/x/crypto` | v0.52.0 | Critical | `go test ./server/fleet/...` (bcrypt) + MDM tests (PBKDF2) |
| `github.com/sethvargo/go-password` | v0.3.0 | Low | Indirect (fleetctl only) |
| `github.com/remitly-oss/httpsig-go` | v1.2.0 | High | `go test ./ee/server/service/hostidentity/...` + `go test ./pkg/fleethttpsig/...` |
| `github.com/sassoftware/relic/v8` | v8.0.1 | Medium | `go test ./pkg/file/...` (8 test files) |

### SCIM

```
go test ./ee/server/scim/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/elimity-com/scim` | v0.0.0 | High | `go test ./ee/server/scim/...` (5 test files) |
| `github.com/scim2/filter-parser/v2` | v2.2.0 | Medium | `go test ./ee/server/scim/...` |

### Vulnerability scanning / NVD / OPA

OPA powers ALL authorization -- every API endpoint. pandatix/nvdapi drives CVE sync.

```
go test ./server/authz/...
go test ./server/vulnerabilities/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/pandatix/nvdapi` | v0.6.4 | High | `go test ./server/vulnerabilities/nvd/...` |
| `github.com/open-policy-agent/opa` | v1.4.2 | Critical | `go test ./server/authz/...` (Rego policy tests) |
| `github.com/oschwald/geoip2-golang` | v1.8.0 | Low | Indirect (optional GeoIP enrichment) |
| `github.com/saferwall/pe` | v1.5.5 | Medium | `go test ./pkg/file/...` |

### Osquery

osquery-go is the core IPC layer between Fleet and osquery. If it breaks, no host data flows.

```
go test ./orbit/pkg/table/...
go test ./server/launcher/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/osquery/osquery-go` | v0.0.0 | Critical | `go test ./orbit/pkg/...` (86 test files) |
| `github.com/AbGuthrie/goquery/v2` | v2.0.1 | Low | Indirect (fleetctl goquery REPL) |
| `github.com/kolide/launcher` | v1.0.12 | Medium | `go test ./server/launcher/...` (1 test file) |
| `github.com/macadmins/osquery-extension` | v1.4.1 | Medium | Indirect (Orbit extension tables) |

### Observability / Telemetry

All OpenTelemetry packages are used as a bundle initialized in `cmd/fleet/otel.go`. Upgrade them together. Elastic APM is a separate stack.

```
go test ./server/platform/tracing/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/getsentry/sentry-go` | v0.18.0 | Low | Indirect (optional error sink) |
| `github.com/prometheus/client_golang` | v1.21.1 | Medium | `go test ./server/service/ -run Metrics` |
| `github.com/rs/zerolog` | v1.32.0 | Medium | `go test ./orbit/pkg/...` (Orbit logger) |
| `go.elastic.co/apm/v2` + modules (4 pkgs) | v2.6.2-v2.7.0 | Low | Indirect (optional APM) |
| `go.opentelemetry.io/*` (14 pkgs) | v0.16.0-v1.43.0 | Medium | `go test ./server/platform/tracing/...` + Build |

### NATS

```
go test ./server/logging/ -run NATS
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/nats-io/nats-server/v2` | v2.12.6 | Low | `go test ./server/logging/ -run NATS` (test-only) |
| `github.com/nats-io/nats.go` | v1.49.0 | Medium | `go test ./server/logging/ -run NATS` (1 test file) |

### Cloud / Infrastructure (AWS, GCP)

Each AWS service maps to a distinct Fleet feature. Test the specific feature you use.

```
go test ./server/logging/...
go test ./server/datastore/s3/...
go test ./server/config/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `cloud.google.com/go/pubsub` | v1.50.1 | Medium | `go test ./server/logging/...` + `go test ./server/mdm/android/...` |
| `github.com/aws/aws-sdk-go-v2` + core (4 pkgs) | v1.19.12-v1.41.5 | High | Indirect (all AWS services below) |
| `github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign` | v1.8.3 | Medium | `go test ./server/datastore/s3/...` |
| `github.com/aws/aws-sdk-go-v2/feature/rds/auth` | v1.6.16 | Medium | Build (AWS RDS only) |
| `github.com/aws/aws-sdk-go-v2/feature/s3/manager` | v1.17.81 | Medium | `go test ./server/datastore/s3/...` |
| `github.com/aws/aws-sdk-go-v2/service/firehose` | v1.37.7 | Medium | `go test ./server/logging/...` |
| `github.com/aws/aws-sdk-go-v2/service/kinesis` | v1.43.5 | Medium | `go test ./server/logging/...` |
| `github.com/aws/aws-sdk-go-v2/service/lambda` | v1.88.5 | Medium | `go test ./server/logging/...` |
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.97.3 | High | `go test ./server/datastore/s3/...` (5 test files) |
| `github.com/aws/aws-sdk-go-v2/service/secretsmanager` | v1.35.8 | Medium | `go test ./server/config/...` |
| `github.com/aws/aws-sdk-go-v2/service/ses` | v1.30.4 | Medium | Indirect (email tests in integration suite) |
| `github.com/aws/aws-sdk-go-v2/service/sts` | v1.41.9 | Medium | Indirect (via AWS common) |
| `github.com/aws/smithy-go` | v1.24.2 | High | Indirect (all AWS services) |
| `google.golang.org/api` | v0.269.0 | High | `go test ./server/mdm/android/...` + `go test ./ee/server/calendar/...` |
| `google.golang.org/grpc` | v1.79.3 | Medium | Indirect (OTel + Google APIs) |

### Packaging / Build

Used at `fleetctl package` time, not agent runtime.

```
go test ./orbit/pkg/packaging/...
go test ./pkg/file/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/goreleaser/nfpm/v2` | v2.20.0 | High | `go test ./orbit/pkg/packaging/...` |
| `github.com/cavaliergopher/rpm` | v1.2.0 | Medium | `go test ./pkg/file/...` |
| `github.com/blakesmith/ar` | v0.0.0 | Medium | `go test ./pkg/file/...` |
| `github.com/josephspurrier/goversioninfo` | v1.4.0 | Low | Build |
| `github.com/mitchellh/gon` | v0.2.6 | Medium | Build (macOS notarization) |
| `github.com/ulikunitz/xz` | v0.5.15 | Low | Build |
| `github.com/xi2/xz` | v0.0.0 | Low | Build |
| `github.com/klauspost/compress` | v1.18.4 | Medium | Indirect (compression in NATS, packaging) |

### CLI / Terminal

cobra (Fleet server) and urfave/cli (fleetctl) are the two CLI frameworks.

```
go test ./cmd/fleetctl/...
go test ./cmd/fleet/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/spf13/cobra` | v1.9.1 | High | `go test ./cmd/fleet/...` (12 test files) |
| `github.com/spf13/pflag` | v1.0.6 | High | Indirect (via cobra) |
| `github.com/spf13/viper` | v1.20.1 | High | `go test ./cmd/fleet/...` (config binding) |
| `github.com/spf13/cast` | v1.7.1 | Low | Indirect (via viper) |
| `github.com/urfave/cli/v2` | v2.27.7 | High | `go test ./cmd/fleetctl/...` (34 test files) |
| `github.com/briandowns/spinner` | v1.23.1 | Low | Build (terminal UX) |
| `github.com/manifoldco/promptui` | v0.9.0 | Low | Build (interactive prompts) |
| `github.com/olekukonko/tablewriter` | v0.0.5 | Low | `go test ./cmd/fleetctl/...` |
| `github.com/gosuri/uilive` | v0.0.4 | Low | Build (terminal UX) |
| `github.com/fatih/color` | v1.16.0 | Low | Build |
| `github.com/skratchdot/open-golang` | v0.0.0 | Low | Build |

### Integrations (Jira, Zendesk, GitHub)

```
go test ./server/worker/...
go test ./server/service/externalsvc/...
go test ./server/vulnerabilities/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/andygrunwald/go-jira` | v1.16.0 | High | `go test ./server/worker/...` (2 test files) + integration tests |
| `github.com/nukosuke/go-zendesk` | v0.13.1 | High | `go test ./server/worker/...` + integration tests |
| `github.com/google/go-github/v37` | v37.0.0 | High | `go test ./server/vulnerabilities/...` |

### XML / YAML / CSV / Data formats

```
go test ./server/fleet/...
go test ./cmd/fleetctl/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/beevik/etree` | v1.6.0 | Medium | `go test ./server/sso/...` (SAML XML) |
| `github.com/antchfx/xmlquery` | v1.3.14 | Medium | `go test ./server/mdm/...` |
| `github.com/clbanning/mxj` | v1.8.4 | Low | Build |
| `github.com/ghodss/yaml` | v1.0.0 | Medium | `go test ./cmd/fleetctl/...` (gitops) |
| `gopkg.in/yaml.v2` | v2.4.0 | Medium | `go test ./cmd/fleetctl/...` |
| `github.com/gocarina/gocsv` | v0.0.0 | Medium | `go test ./server/service/...` (CSV export) |
| `github.com/go-json-experiment/json` | v0.0.0 | Low | Build |
| `github.com/go-ini/ini` | v1.67.0 | Low | Build |
| `gopkg.in/ini.v1` | v1.67.0 | Low | Build |

### Semver / Versioning

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/Masterminds/semver` | v1.5.0 | Medium | Indirect (used in version comparisons) |
| `github.com/Masterminds/semver/v3` | v3.3.1 | Medium | Indirect (used in version comparisons) |

### System / OS / Hardware

Many are platform-specific. Test on the relevant platform.

```
go test ./orbit/pkg/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/shirou/gopsutil/v4` | v4.26.2 | High | `go test ./orbit/pkg/platform/...` |
| `github.com/digitalocean/go-smbios` | v0.0.0 | Low | Build |
| `github.com/go-ole/go-ole` | v1.2.6 | Medium | `go test ./orbit/pkg/windows/...` (Windows) |
| `github.com/godbus/dbus/v5` | v5.1.0 | Low | Build (Linux fleet-desktop) |
| `github.com/hectane/go-acl` | v0.0.0 | Low | Build (Windows) |
| `github.com/hillu/go-ntdll` | v0.0.0 | Low | Build (Windows) |
| `github.com/scjalliance/comshim` | v0.0.0 | Low | Build (Windows) |
| `github.com/mitchellh/go-ps` | v1.0.0 | Low | Indirect |
| `github.com/siderolabs/go-blockdevice/v2` | v2.0.3 | Low | Build |
| `fyne.io/systray` | v1.10.1 | High | Gap (GUI -- manual test: tray icon shows) |
| `github.com/danieljoos/wincred` | v1.2.1 | Low | Build (Windows) |
| `github.com/Azure/go-ntlmssp` | v0.1.1 | Low | Build |

### TPM (Trusted Platform Module)

```
go test ./ee/orbit/pkg/securehw/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/google/go-tpm` | v0.9.8 | High | `go test ./ee/orbit/pkg/securehw/...` |
| `github.com/foxboron/go-tpm-keyfiles` | v0.0.0 | High | `go test ./ee/orbit/pkg/securehw/...` |

### KV store / Embedded DB

```
go test ./orbit/pkg/update/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/dgraph-io/badger/v2` | v2.2007.4 | Medium | `go test ./orbit/pkg/update/...` (13 test files) |
| `github.com/boltdb/bolt` | v1.3.1 | Low | Indirect (TUF file store) |
| `go.etcd.io/bbolt` | v1.3.10 | Low | Indirect |

### Docker / Containers

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/containerd/containerd` | v1.7.33 | Low | Build |
| `github.com/docker/docker` | v28.0.0 | Low | Build |
| `github.com/docker/go-units` | v0.5.0 | Low | Build |

### TUF (The Update Framework)

```
go test ./orbit/pkg/update/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/theupdateframework/go-tuf` | v0.5.2 | Critical | `go test ./orbit/pkg/update/...` (13 test files) |

### Image / Assets

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/nfnt/resize` | v0.0.0 | Low | Indirect |
| `golang.org/x/image` | v0.42.0 | Low | Indirect |
| `github.com/elazarl/go-bindata-assetfs` | v1.0.1 | Medium | Build (serves Fleet UI static assets) |

### Expression / Pattern matching

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/expr-lang/expr` | v1.17.7 | Low | `go test ./server/logging/...` |
| `github.com/bmatcuk/doublestar/v4` | v4.10.0 | Low | `go test ./cmd/fleetctl/...` (gitops) |

### Log shipping

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/facebookincubator/flog` | v0.0.0 | Low | Build |
| `gopkg.in/natefinch/lumberjack.v2` | v2.0.0 | Low | Indirect |

### Data structures / Utilities

These are low-level libraries used broadly. A full test suite run is the best verification.

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/RoaringBitmap/roaring` | v1.9.4 | Medium | `REDIS_TEST=1 go test ./server/datastore/redis/...` |
| `github.com/agnivade/levenshtein` | v1.2.1 | Low | Build |
| `github.com/google/uuid` | v1.6.0 | Medium | Indirect (used everywhere) |
| `github.com/google/go-cmp` | v0.7.0 | Low | Test-only lib |
| `github.com/hashicorp/go-multierror` | v1.1.1 | Low | Indirect |
| `github.com/cenkalti/backoff` | v2.2.1 | Low | Indirect |
| `github.com/cenkalti/backoff/v4` | v4.3.0 | Low | Indirect |
| `github.com/oklog/run` | v1.1.0 | Low | Build |
| `github.com/WatchBeam/clock` | v0.0.0 | Low | Test-only lib |
| `github.com/beevik/ntp` | v0.3.0 | Low | Indirect (Orbit sntp table) |
| `github.com/gofrs/flock` | v0.12.1 | Low | Indirect |
| `github.com/golang/snappy` | v0.0.4 | Low | Build |
| `gopkg.in/guregu/null.v3` | v3.5.0 | Low | Indirect |
| `pgregory.net/rapid` | v1.2.0 | Low | Test-only lib |

### Test-only

These are only compiled into `_test.go` files. The smoke test is: the test suite passes.

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `github.com/stretchr/testify` | v1.11.1 | Low | `go test ./...` (test infrastructure) |
| `github.com/tj/assert` | v0.0.3 | Low | `go test ./...` |
| `github.com/davecgh/go-spew` | v1.1.1 | Low | `go test ./...` |
| `github.com/pmezard/go-difflib` | v1.0.0 | Low | `go test ./...` |
| `github.com/quasilyte/go-ruleguard/dsl` | v0.3.22 | Low | `make lint-go` |
| `github.com/groob/finalizer` | v0.0.0 | Low | `go test ./...` |

### Stdlib extensions (`golang.org/x`)

```
go test ./...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `golang.org/x/exp` | v0.0.0 | Low | Indirect |
| `golang.org/x/mod` | v0.36.0 | Low | Build |
| `golang.org/x/net` | v0.55.0 | Medium | Indirect (HTTP/2, SAML, HTML) |
| `golang.org/x/sync` | v0.21.0 | Medium | Indirect (errgroup, singleflight) |
| `golang.org/x/sys` | v0.45.0 | Medium | Indirect (OS syscalls) |
| `golang.org/x/term` | v0.43.0 | Low | Indirect (fleetctl password prompt) |
| `golang.org/x/text` | v0.38.0 | Low | Indirect (unicode/encoding) |
| `golang.org/x/tools` | v0.45.0 | Low | `make lint-go` + `make generate` |

---

## Inlined third-party code

These libraries have been copied into Fleet's codebase. Tracked in `third_party/vuln-check/go.mod` for vulnerability scanning.

| Library | Inlined location | Version | Risk | Tested by |
|---------|-----------------|---------|------|-----------|
| `github.com/micromdm/nanomdm` | `server/mdm/nanomdm/` | v0.9.0 | Critical | MDM integration tests |
| `github.com/micromdm/nanodep` | `server/mdm/nanodep/` | v0.4.0 | Critical | MDM integration tests |
| `github.com/micromdm/scep/v2` | `server/mdm/scep/` | v2.3.0 | Critical | `go test ./server/mdm/scep/...` |
| `github.com/pressly/goose/v3` | `server/goose/` | v3.17.0 | High | `go test ./server/datastore/mysql/migrations/...` |
| `github.com/facebookincubator/nvdtools` | `server/vulnerabilities/nvd/tools/` | v0.1.5 | High | `go test ./server/vulnerabilities/...` |
| `github.com/virtuald/go-paniclog` | `orbit/pkg/go-paniclog/` | v0.0.0 | Low | Build |
| `github.com/josharian/impl` | `server/mock/mockimpl/` | v1.4.0 | Low | `go generate ./server/mock/...` |
| `github.com/mitchellh/gon` | `orbit/pkg/packaging/` (partial) | v0.2.3 | Low | Build |

---

## Goval-dictionary (OVAL vuln data)

Module: `third_party/goval-dictionary/go.mod`. The shared smoke test is: OVAL vulnerability scanning completes and detects known vulnerabilities on Linux hosts.

```
go test ./third_party/goval-dictionary/...
go test ./server/vulnerabilities/oval/...
go test ./server/vulnerabilities/goval_dictionary/...
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `gorm.io/gorm` + drivers (3 pkgs) | v1.5.5-v1.25.7 | Medium | `go test ./server/vulnerabilities/oval/...` |
| `github.com/hashicorp/go-version` | v1.8.0 | Medium | Indirect (version matching) |
| `github.com/knqyf263/go-rpm-version` | v0.0.0 | Medium | Indirect (RPM version matching) |
| Other 15 libraries | various | Low | Build or Indirect |

---

## Tools

All tools are internal. Shared smoke test: `go build` succeeds and the tool runs.

| Tool | Module | Test command | Libs |
|------|--------|-------------|------|
| Fleet MCP server | `tools/fleet-mcp/go.mod` | `go build ./tools/fleet-mcp/` | 5 (all Low) |
| Terraform provider | `tools/terraform/go.mod` | `go test ./tools/terraform/...` | 5 (Medium) |
| CI linter plugins | `tools/ci/*/go.mod` | `make lint-go` | 2 (Low) |
| GitHub management TUI | `tools/github-manage/go.mod` | `go build ./tools/github-manage/` | 7 (Low) |
| QA check | `tools/qacheck/go.mod` | `go build ./tools/qacheck/` | 2 (Low) |
| Snapshot tool | `tools/snapshot/go.mod` | `go build ./tools/snapshot/` | 2 (Low) |
| Dibble seeder | `tools/dibble/go.mod` | `go build ./tools/dibble/` | 6 (Low) |
| Hangar desktop app | `tools/hangar/go.mod` | `go build ./tools/hangar/` | 3 (Low) |
| Screencap tool | `tools/screencap/go.mod` | `go build ./tools/screencap/` | 2 (Low) |

---

## Frontend (runtime)

From `package.json` `dependencies`. **330 test files** exist across the frontend.

```
yarn test
```

| Library | Version | Risk | Tested by |
|---------|---------|------|-----------|
| `react` | 18.3.1 | Critical | `yarn test` (330 test files use React) |
| `react-dom` | 18.2.0 | Critical | `yarn test` |
| `react-query` | 3.39.3 | Critical | `yarn test` (6 test files) |
| `react-router` | 3.2.6 | Critical | `yarn test` (6 test files) |
| `axios` | 1.16.1 | Critical | `yarn test` (3 test files via MSW mocks) |
| `lodash` | 4.18.1 | High | `yarn test` (62 test files) |
| `react-table` | 7.7.0 | High | Gap (0 test files -- tested via component render tests) |
| `react-ace` / `ace-builds` | 9.3.0 / 1.4.14 | High | `yarn test` (1 test file) |
| `react-select` / `react-select-5` | 1.3.0 / 5.4.0 | Medium | `yarn test` (5 test files) |
| `cmdk` | 1.1.1 | Medium | `yarn test` (6 test files -- CommandPalette) |
| `dompurify` | 3.4.11 | High | `yarn test -- ClickableUrls_xss` (XSS sanitization tests) |
| `validator` | 13.15.22 | Medium | `yarn test` (5 test files -- form validators) |
| `@sgress454/node-sql-parser` | 5.4.0-fork.2 | Medium | `yarn test -- sql_tools` (2 test files) |
| `date-fns` / `date-fns-tz` | 3.6.0 / 3.1.3 | Medium | `yarn test -- date_format` (1 test file) |
| `js-yaml` | 4.2.0 | Medium | `yarn test -- yaml` (yaml.tests.ts + validate_yaml.tests.js) |
| `js-cookie` | 3.0.7 | High | `yarn test` (1 test file -- auth_token) |
| `file-saver` | 1.3.8 | Medium | `yarn test` (1 test file) |
| `recharts` | 3.8.1 | Medium | `yarn test` (1 test file) |
| `sockjs-client` | 1.6.1 | High | Gap (WebSocket -- manual test: live query in UI) |
| `react-markdown` / `remark-gfm` | 10.1.0 / 4.0.1 | Medium | Gap (manual test: view markdown description) |
| `sonner` | 2.0.7 | Low | Gap (manual test: trigger a toast) |
| `react-tabs` | 3.2.3 | Low | Indirect |
| `react-tooltip` / `react-tooltip-5` | 4.2.21 / 5.29.1 | Low | Indirect |
| `react-error-boundary` | 3.1.4 | Low | Indirect |
| `react-router-transition` | 1.2.1 | Low | Indirect |
| `react-accessible-accordion` | 3.3.5 | Low | Unused (consider removing) |
| `uuid` | 14.0.0 | Low | Unused (consider removing) |
| `content-disposition` | 0.5.4 | Low | Indirect |
| `core-js` / `es6-*` / `isomorphic-fetch` | various | Low | Build (polyfills) |
| `history` | 2.1.0 | Medium | Indirect (react-router) |
| `js-md5` | 0.7.3 | Low | Indirect (Gravatar) |
| `memoize-one` / `normalizr` / `when` | various | Low | Indirect |
| `postcss` / `sass` | 8.5.10 / 1.83.4 | Medium | `make build` |
| `prop-types` / `select` / `use-debounce` | various | Low | Indirect |
| `proxy-middleware` / `rc-pagination` | various | Low | Build / Indirect |

---

## Frontend (dev / build toolchain)

From `package.json` `devDependencies`. These do not ship to users.

```
make build        # Verifies webpack/babel/typescript/sass pipeline
yarn test         # Verifies jest/testing-library pipeline
yarn lint         # Verifies eslint pipeline
yarn storybook    # Verifies storybook pipeline
```

| Library group | Risk | Tested by |
|---------------|------|-----------|
| `webpack` + `webpack-cli` + loaders | High | `make build` |
| `typescript` + `ts-loader` + `fork-ts-checker-webpack-plugin` | High | `make build` |
| `@babel/*` (21 packages) + `babel-*` helpers | High | `make build` + `yarn test` |
| `sass-loader` + `css-loader` + `postcss-loader` + `mini-css-extract-plugin` | Medium | `make build` |
| `jest` + `jest-environment-jsdom` + `@testing-library/*` + `msw` | Medium | `yarn test` |
| `eslint` + `@typescript-eslint/*` + plugins (12 packages) | Low | `yarn lint` |
| `prettier` + `eslint-config-prettier` | Low | `yarn prettier:check` |
| `@storybook/*` (10 packages) | Low | `yarn storybook` |
| `classnames` / `compare-versions` / `bourbon` / other utilities | Low | `make build` |

---

## Summary

| Component | Libs | Test coverage |
|-----------|------|---------------|
| Server + fleetctl + Orbit (Go) | 182 | Excellent (669+ test files) |
| Inlined third-party | 8 | Good (MDM + vuln integration tests) |
| Goval-dictionary | 20 | Good (OVAL vuln tests) |
| Tools | ~34 | Build-verified |
| Frontend (runtime) | 49 | Good (330 test files, 4 gaps) |
| Frontend (dev) | ~60 | Pipeline-verified (`make build` / `yarn test` / `yarn lint`) |

### Identified gaps

| Library | Risk | Why it's a gap | Mitigation |
|---------|------|----------------|------------|
| `fyne.io/systray` | High | GUI code, no unit test possible | Manual: verify tray icon on macOS/Win/Linux |
| `sockjs-client` | High | WebSocket, needs live server | Manual: run live query from Fleet UI |
| `react-table` | High | Component rendering, no dedicated test | Indirectly tested via page-level tests |
| `react-markdown` | Medium | Markdown rendering | Manual: view a policy with markdown description |
| `sonner` | Low | Toast notifications | Manual: perform an action, verify toast appears |
