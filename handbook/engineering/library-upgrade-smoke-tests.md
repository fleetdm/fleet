# Library upgrade smoke tests

When upgrading a third-party dependency, use this document to find the smoke tests you need to run. Libraries are grouped by functional area. Find your library, run the listed smoke tests, and confirm everything works before merging.

> **How to use:** Press Ctrl+F / Cmd+F and search for the library name (e.g. `gorilla/mux`). The table tells you what to test and how risky the upgrade is.

> **Smoke test status:** `TODO` means the smoke test has not been defined yet. These will be filled in as part of [#48943](https://github.com/fleetdm/fleet/issues/48943).

## Table of contents

- [Server + fleetctl + Orbit (go.mod)](#server--fleetctl--orbit-gomod)
  - [Cloud / Infrastructure (AWS, GCP)](#cloud--infrastructure-aws-gcp)
  - [HTTP / Routing / Middleware](#http--routing--middleware)
  - [Database / SQL](#database--sql)
  - [Redis / Caching](#redis--caching)
  - [MDM (Apple / Microsoft / SCEP)](#mdm-apple--microsoft--scep)
  - [Authentication / Security / Crypto](#authentication--security--crypto)
  - [SCIM](#scim)
  - [Vulnerability scanning / NVD / OPA](#vulnerability-scanning--nvd--opa)
  - [Osquery](#osquery)
  - [Observability / Telemetry](#observability--telemetry)
  - [NATS (Live query)](#nats-live-query)
  - [Packaging / Build](#packaging--build)
  - [CLI / Terminal](#cli--terminal)
  - [XML / YAML / CSV / Data formats](#xml--yaml--csv--data-formats)
  - [Integrations (Jira, Zendesk, GitHub)](#integrations-jira-zendesk-github)
  - [Semver / Versioning](#semver--versioning)
  - [System / OS / Hardware](#system--os--hardware)
  - [TPM (Trusted Platform Module)](#tpm-trusted-platform-module)
  - [KV store / Embedded DB](#kv-store--embedded-db)
  - [Docker / Containers](#docker--containers)
  - [TUF (The Update Framework)](#tuf-the-update-framework)
  - [Image / Assets](#image--assets)
  - [Expression / Pattern matching](#expression--pattern-matching)
  - [Log shipping](#log-shipping)
  - [Data structures / Utilities](#data-structures--utilities)
  - [Test-only](#test-only)
  - [Stdlib extensions (golang.org/x)](#stdlib-extensions-golangorgx)
- [Inlined third-party code](#inlined-third-party-code)
- [Goval-dictionary (OVAL vuln data)](#goval-dictionary-oval-vuln-data)
- [Tools](#tools)
  - [Fleet MCP server](#fleet-mcp-server)
  - [Terraform provider](#terraform-provider)
  - [CI linter plugins](#ci-linter-plugins)
  - [GitHub management TUI](#github-management-tui)
  - [QA check](#qa-check)
  - [Snapshot tool](#snapshot-tool)
  - [Dibble test-data seeder](#dibble-test-data-seeder)
  - [Hangar desktop app](#hangar-desktop-app)
  - [Screencap tool](#screencap-tool)
- [Frontend (runtime)](#frontend-runtime)
- [Frontend (dev / build toolchain)](#frontend-dev--build-toolchain)
- [Summary](#summary)

---

## Server + fleetctl + Orbit (`go.mod`)

The main Go module covers the Fleet server, the `fleetctl` CLI, and the Orbit agent.

### Cloud / Infrastructure (AWS, GCP)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `cloud.google.com/go/pubsub` | v1.50.1 | TODO | |
| `github.com/aws/aws-sdk-go-v2` | v1.41.5 | TODO | |
| `github.com/aws/aws-sdk-go-v2/config` | v1.32.12 | TODO | |
| `github.com/aws/aws-sdk-go-v2/credentials` | v1.19.12 | TODO | |
| `github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign` | v1.8.3 | TODO | |
| `github.com/aws/aws-sdk-go-v2/feature/rds/auth` | v1.6.16 | TODO | |
| `github.com/aws/aws-sdk-go-v2/feature/s3/manager` | v1.17.81 | TODO | |
| `github.com/aws/aws-sdk-go-v2/service/firehose` | v1.37.7 | TODO | |
| `github.com/aws/aws-sdk-go-v2/service/kinesis` | v1.43.5 | TODO | |
| `github.com/aws/aws-sdk-go-v2/service/lambda` | v1.88.5 | TODO | |
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.97.3 | TODO | |
| `github.com/aws/aws-sdk-go-v2/service/secretsmanager` | v1.35.8 | TODO | |
| `github.com/aws/aws-sdk-go-v2/service/ses` | v1.30.4 | TODO | |
| `github.com/aws/aws-sdk-go-v2/service/sts` | v1.41.9 | TODO | |
| `github.com/aws/smithy-go` | v1.24.2 | TODO | |
| `google.golang.org/api` | v0.269.0 | TODO | |
| `google.golang.org/grpc` | v1.79.3 | TODO | |

### HTTP / Routing / Middleware

> **go-kit/kit** and **gorilla/mux** are the backbone of every API request. If either breaks, nothing works. Test broadly.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/go-kit/kit` | v0.12.0 | Every endpoint is built on go-kit's endpoint/transport layer. **1.** Server starts and `/healthz` responds. **2.** Login returns a valid token. **3.** List hosts API returns results. **4.** Create/edit a policy via API. **5.** MDM endpoints respond (e.g. SCEP, MDM checkin). **6.** Error responses have correct JSON structure. **7.** `fleetctl` CLI commands (get hosts, apply config) work. | Critical |
| `github.com/gorilla/mux` | v1.8.1 | All Fleet routes are registered on this router. **1.** Hit 5+ distinct API routes and verify 200 responses. **2.** URL path params work (e.g. `/api/v1/fleet/hosts/:id`). **3.** Unknown routes return 404. **4.** SCEP and MDM sub-routes respond. **5.** SCIM routes respond (EE). | Critical |
| `github.com/gorilla/websocket` | v1.5.1 | Powers the WebSocket transport for live queries. **1.** Run a live query from the Fleet UI -- verify results stream in real time. **2.** Run `fleetctl query` from CLI -- verify results return. **3.** Cancel a live query mid-flight and verify clean disconnect. | High |
| `github.com/igm/sockjs-go/v3` | v3.0.2 | SockJS fallback when raw WebSocket is blocked. **1.** Run a live query from the Fleet UI (standard path). **2.** If possible, test behind a proxy that blocks WebSocket upgrades -- verify live query still works via XHR fallback. | High |
| `github.com/throttled/throttled/v2` | v2.8.0 | Rate-limits login, SSO, forgot-password, MFA endpoints. **1.** Attempt 10+ rapid login failures -- verify rate-limit response (HTTP 429). **2.** Verify SSO initiate endpoint is rate-limited. **3.** Verify forgot-password endpoint is rate-limited. **4.** Normal login still works after rate-limit window passes. | High |
| `github.com/realclientip/realclientip-go` | v1.0.0 | Extracts true client IP for rate limiting behind proxies. **1.** Configure `trusted_proxies` and deploy behind a reverse proxy. **2.** Verify rate limiting keys on the correct client IP (not the proxy IP). **3.** Without `trusted_proxies`, verify `RemoteAddr` is used. | Medium |
| `github.com/e-dard/netbug` | v0.0.0 | Exposes `/debug/` pprof endpoints behind auth token. **1.** Set `debug_token` in server config. **2.** Hit `/debug/pprof/` with the token -- verify pprof index page renders. **3.** Without the token, verify 403/401. | Low |

### Database / SQL

> **go-sql-driver/mysql** and **jmoiron/sqlx** underpin every database operation. If either breaks, the server cannot function. Test the full request lifecycle.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/go-sql-driver/mysql` | v1.9.3 | The MySQL wire-protocol driver -- every DB query goes through it. **1.** Server starts and connects to MySQL. **2.** Login/create user works. **3.** Enroll a host -- verify it appears in the host list. **4.** Create a policy -- verify it persists across server restart. **5.** Run `MYSQL_TEST=1 go test ./server/datastore/mysql/...` | Critical |
| `github.com/jmoiron/sqlx` | v1.3.5 | Struct scanning and named queries for all data access. **1.** Same as `go-sql-driver/mysql` above (sqlx wraps every query). **2.** Additionally: list hosts with filters (exercises struct scanning). **3.** Edit app config and re-read it (exercises named params). **4.** Run `MYSQL_TEST=1 go test ./server/datastore/mysql/...` | Critical |
| `github.com/doug-martin/goqu/v9` | v9.18.0 | Query builder for complex dynamic queries in software/host listing. **1.** List software with filters (name, version, vulnerable). **2.** List hosts with multiple filters (platform, label, status). **3.** Vulnerability scanning: run a vuln scan and verify CVEs matched. **4.** Run `go test ./server/datastore/mysql/ -run TestSoftware` | High |
| `github.com/ngrok/sqlmw` | v0.0.0 | SQL middleware that wraps the MySQL driver for intercepting queries. **1.** Server starts without errors (the wrapped driver registers at init). **2.** If using a custom interceptor (tracing), verify SQL spans appear. **3.** Run `go test ./server/platform/mysql/...` | Medium |
| `github.com/VividCortex/mysqlerr` | v0.0.0 | MySQL error code constants for duplicate-entry, FK violations, deadlock retry. **1.** Create a duplicate entity (e.g. same email) -- verify 409 Conflict response (not 500). **2.** Delete an entity referenced by a FK -- verify correct error. **3.** Run `go test ./server/datastore/mysql/ -run TestDuplicate` | Medium |
| `github.com/ziutek/mymysql` | v1.5.4 | Only used by the standalone `goose` migration CLI (not the Fleet server binary). **1.** Run `go build ./server/goose/cmd/goose/` -- verify it compiles. **2.** Run a migration up/down with the goose CLI. | Low |
| `github.com/mattn/go-sqlite3` | v1.14.22 | CGo SQLite driver for vulnerability scanning (NVD CPE database, OVAL). **1.** Run vulnerability scanning -- verify CVEs are detected for known-vulnerable software. **2.** OVAL-based Linux vuln scanning completes without errors. **3.** Run `go test ./server/vulnerabilities/...` | High |
| `github.com/lib/pq` | v1.10.9 | PostgreSQL driver, only in standalone CLI tools (goose, nanomdm CLI). Not used by the Fleet server. **1.** Run `go build ./server/goose/cmd/goose/` -- verify it compiles. | Low |
| `github.com/shogo82148/rdsmysql/v2` | v2.5.0 | AWS RDS IAM auth TLS certificate registration. Only active in AWS RDS deployments. **1.** Configure Fleet with RDS IAM authentication. **2.** Verify the server connects to the RDS instance and serves requests. **3.** If no AWS environment, verify `go build` succeeds. | Medium |
| `github.com/XSAM/otelsql` | v0.39.0 | OpenTelemetry SQL instrumentation (tracing spans + DB pool metrics). **1.** Enable OTel tracing and point to a collector. **2.** Make API requests -- verify SQL query spans appear in the trace backend. **3.** Verify DB pool metrics are exported. | Low |
| `github.com/DATA-DOG/go-sqlmock` | v1.5.0 | SQL mock for tests only -- not compiled into production. **1.** Run `go test ./server/platform/mysql/... ./server/datastore/mysql/migrations/...` | Low |

### Redis / Caching

> **redigo** is used by virtually every feature that coordinates across Fleet server instances: live queries, SSO, host check-ins, MDM, automations. Test broadly.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/gomodule/redigo` | v1.8.9 | Fleet's foundational Redis client -- used by live query targeting, pub/sub, SSO session storage, distributed locks, async host-seen ingestion, policy automations, MDM profile matching, host auth caching, and error dedup. **1.** Run a live query -- verify results stream back. **2.** SSO login flow completes successfully. **3.** Enroll a host -- verify check-in succeeds (host-seen written to Redis). **4.** Trigger a failing-policy automation -- verify it fires. **5.** In a multi-server deployment, verify distributed lock prevents duplicate cron runs. **6.** Run `REDIS_TEST=1 go test ./server/datastore/redis/...` | Critical |
| `github.com/mna/redisc` | v1.3.2 | Redis Cluster support -- wraps redigo for slot routing, retries, and replica reads. Only active in Redis Cluster deployments. **1.** Configure Fleet against a Redis Cluster. **2.** Run a live query -- verify results return. **3.** SSO login works. **4.** If no cluster environment, verify `go test ./server/datastore/redis/...` passes. | High |
| `github.com/patrickmn/go-cache` | v2.1.0 | In-process memory cache for hot DB reads (app config 1s TTL, team settings 1m, packs 1m, MDM config assets 15m). **1.** Change app config -- verify the change takes effect within ~1s. **2.** Update team agent options -- verify hosts pick up the change within ~1m. **3.** Upload an MDM config asset -- verify it's served correctly. **4.** Run `go test ./server/datastore/cached_mysql/...` | High |

### MDM (Apple / Microsoft / SCEP)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/micromdm/micromdm` | v1.9.0 | TODO | |
| `github.com/micromdm/nanolib` | v0.2.0 | TODO | |
| `github.com/micromdm/plist` | v0.2.3 | TODO | |
| `github.com/groob/plist` | v0.0.0 | TODO | |
| `howett.net/plist` | v1.0.1 | TODO | |
| `github.com/smallstep/scep` | v0.0.0 | TODO | |
| `github.com/smallstep/pkcs7` | v0.0.0 | TODO | |
| `go.step.sm/crypto` | v0.77.1 | TODO | |
| `software.sslmate.com/src/go-pkcs12` | v0.4.0 | TODO | |
| `github.com/RobotsAndPencils/buford` | v0.14.0 | TODO | |
| `github.com/fxamacker/cbor/v2` | v2.9.1 | TODO | |

### Authentication / Security / Crypto

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/crewjam/saml` | v0.5.1 | TODO | |
| `github.com/russellhaering/goxmldsig` | v1.6.0 | TODO | |
| `github.com/golang-jwt/jwt/v4` | v4.5.2 | TODO | |
| `github.com/MicahParks/jwkset` | v0.11.0 | TODO | |
| `github.com/gomodule/oauth1` | v0.2.0 | TODO | |
| `golang.org/x/oauth2` | v0.35.0 | TODO | |
| `golang.org/x/crypto` | v0.52.0 | TODO | |
| `github.com/sethvargo/go-password` | v0.3.0 | TODO | |
| `github.com/remitly-oss/httpsig-go` | v1.2.0 | TODO | |
| `github.com/sassoftware/relic/v8` | v8.0.1 | TODO | |

### SCIM

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/elimity-com/scim` | v0.0.0 | TODO | |
| `github.com/scim2/filter-parser/v2` | v2.2.0 | TODO | |

### Vulnerability scanning / NVD / OPA

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/pandatix/nvdapi` | v0.6.4 | TODO | |
| `github.com/open-policy-agent/opa` | v1.4.2 | TODO | |
| `github.com/oschwald/geoip2-golang` | v1.8.0 | TODO | |
| `github.com/saferwall/pe` | v1.5.5 | TODO | |

### Osquery

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/osquery/osquery-go` | v0.0.0 | TODO | |
| `github.com/AbGuthrie/goquery/v2` | v2.0.1 | TODO | |
| `github.com/kolide/launcher` | v1.0.12 | TODO | |
| `github.com/macadmins/osquery-extension` | v1.4.1 | TODO | |

### Observability / Telemetry

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/getsentry/sentry-go` | v0.18.0 | TODO | |
| `github.com/prometheus/client_golang` | v1.21.1 | TODO | |
| `github.com/rs/zerolog` | v1.32.0 | TODO | |
| `go.elastic.co/apm/v2` | v2.7.0 | TODO | |
| `go.elastic.co/apm/module/apmgorilla/v2` | v2.6.2 | TODO | |
| `go.elastic.co/apm/module/apmhttp/v2` | v2.7.1 | TODO | |
| `go.elastic.co/apm/module/apmsql/v2` | v2.6.2 | TODO | |
| `go.opentelemetry.io/otel` | v1.43.0 | TODO | |
| `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc` | v0.16.0 | TODO | |
| `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc` | v1.40.0 | TODO | |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace` | v1.40.0 | TODO | |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.40.0 | TODO | |
| `go.opentelemetry.io/otel/log` | v0.16.0 | TODO | |
| `go.opentelemetry.io/otel/metric` | v1.43.0 | TODO | |
| `go.opentelemetry.io/otel/sdk` | v1.43.0 | TODO | |
| `go.opentelemetry.io/otel/sdk/log` | v0.16.0 | TODO | |
| `go.opentelemetry.io/otel/sdk/metric` | v1.43.0 | TODO | |
| `go.opentelemetry.io/otel/trace` | v1.43.0 | TODO | |
| `go.opentelemetry.io/contrib/bridges/otelslog` | v0.15.0 | TODO | |
| `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux` | v0.60.0 | TODO | |
| `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` | v0.61.0 | TODO | |

### NATS (Live query)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/nats-io/nats-server/v2` | v2.12.6 | TODO | |
| `github.com/nats-io/nats.go` | v1.49.0 | TODO | |

### Packaging / Build

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/goreleaser/nfpm/v2` | v2.20.0 | TODO | |
| `github.com/cavaliergopher/rpm` | v1.2.0 | TODO | |
| `github.com/blakesmith/ar` | v0.0.0 | TODO | |
| `github.com/josephspurrier/goversioninfo` | v1.4.0 | TODO | |
| `github.com/mitchellh/gon` | v0.2.6 | TODO | |
| `github.com/ulikunitz/xz` | v0.5.15 | TODO | |
| `github.com/xi2/xz` | v0.0.0 | TODO | |
| `github.com/klauspost/compress` | v1.18.4 | TODO | |

### CLI / Terminal

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/spf13/cobra` | v1.9.1 | TODO | |
| `github.com/spf13/pflag` | v1.0.6 | TODO | |
| `github.com/spf13/viper` | v1.20.1 | TODO | |
| `github.com/spf13/cast` | v1.7.1 | TODO | |
| `github.com/urfave/cli/v2` | v2.27.7 | TODO | |
| `github.com/briandowns/spinner` | v1.23.1 | TODO | |
| `github.com/manifoldco/promptui` | v0.9.0 | TODO | |
| `github.com/olekukonko/tablewriter` | v0.0.5 | TODO | |
| `github.com/gosuri/uilive` | v0.0.4 | TODO | |
| `github.com/fatih/color` | v1.16.0 | TODO | |
| `github.com/skratchdot/open-golang` | v0.0.0 | TODO | |

### XML / YAML / CSV / Data formats

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/beevik/etree` | v1.6.0 | TODO | |
| `github.com/antchfx/xmlquery` | v1.3.14 | TODO | |
| `github.com/clbanning/mxj` | v1.8.4 | TODO | |
| `github.com/ghodss/yaml` | v1.0.0 | TODO | |
| `gopkg.in/yaml.v2` | v2.4.0 | TODO | |
| `github.com/gocarina/gocsv` | v0.0.0 | TODO | |
| `github.com/go-json-experiment/json` | v0.0.0 | TODO | |
| `github.com/go-ini/ini` | v1.67.0 | TODO | |
| `gopkg.in/ini.v1` | v1.67.0 | TODO | |

### Integrations (Jira, Zendesk, GitHub)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/andygrunwald/go-jira` | v1.16.0 | TODO | |
| `github.com/nukosuke/go-zendesk` | v0.13.1 | TODO | |
| `github.com/google/go-github/v37` | v37.0.0 | TODO | |

### Semver / Versioning

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/Masterminds/semver` | v1.5.0 | TODO | |
| `github.com/Masterminds/semver/v3` | v3.3.1 | TODO | |

### System / OS / Hardware

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/shirou/gopsutil/v4` | v4.26.2 | TODO | |
| `github.com/digitalocean/go-smbios` | v0.0.0 | TODO | |
| `github.com/go-ole/go-ole` | v1.2.6 | TODO | |
| `github.com/godbus/dbus/v5` | v5.1.0 | TODO | |
| `github.com/hectane/go-acl` | v0.0.0 | TODO | |
| `github.com/hillu/go-ntdll` | v0.0.0 | TODO | |
| `github.com/scjalliance/comshim` | v0.0.0 | TODO | |
| `github.com/mitchellh/go-ps` | v1.0.0 | TODO | |
| `github.com/siderolabs/go-blockdevice/v2` | v2.0.3 | TODO | |
| `fyne.io/systray` | v1.10.1 | TODO | |
| `github.com/danieljoos/wincred` | v1.2.1 | TODO | |
| `github.com/Azure/go-ntlmssp` | v0.1.1 | TODO | |

### TPM (Trusted Platform Module)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/google/go-tpm` | v0.9.8 | TODO | |
| `github.com/foxboron/go-tpm-keyfiles` | v0.0.0 | TODO | |

### KV store / Embedded DB

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/dgraph-io/badger/v2` | v2.2007.4 | TODO | |
| `github.com/boltdb/bolt` | v1.3.1 | TODO | |
| `go.etcd.io/bbolt` | v1.3.10 | TODO | |

### Docker / Containers

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/containerd/containerd` | v1.7.33 | TODO | |
| `github.com/docker/docker` | v28.0.0 | TODO | |
| `github.com/docker/go-units` | v0.5.0 | TODO | |

### TUF (The Update Framework)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/theupdateframework/go-tuf` | v0.5.2 | TODO | |

### Image / Assets

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/nfnt/resize` | v0.0.0 | TODO | |
| `golang.org/x/image` | v0.42.0 | TODO | |
| `github.com/elazarl/go-bindata-assetfs` | v1.0.1 | TODO | |

### Expression / Pattern matching

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/expr-lang/expr` | v1.17.7 | TODO | |
| `github.com/bmatcuk/doublestar/v4` | v4.10.0 | TODO | |

### Log shipping

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/facebookincubator/flog` | v0.0.0 | TODO | |
| `gopkg.in/natefinch/lumberjack.v2` | v2.0.0 | TODO | |

### Data structures / Utilities

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/RoaringBitmap/roaring` | v1.9.4 | TODO | |
| `github.com/agnivade/levenshtein` | v1.2.1 | TODO | |
| `github.com/google/uuid` | v1.6.0 | TODO | |
| `github.com/google/go-cmp` | v0.7.0 | TODO | |
| `github.com/hashicorp/go-multierror` | v1.1.1 | TODO | |
| `github.com/cenkalti/backoff` | v2.2.1 | TODO | |
| `github.com/cenkalti/backoff/v4` | v4.3.0 | TODO | |
| `github.com/oklog/run` | v1.1.0 | TODO | |
| `github.com/WatchBeam/clock` | v0.0.0 | TODO | |
| `github.com/beevik/ntp` | v0.3.0 | TODO | |
| `github.com/gofrs/flock` | v0.12.1 | TODO | |
| `github.com/golang/snappy` | v0.0.4 | TODO | |
| `gopkg.in/guregu/null.v3` | v3.5.0 | TODO | |
| `pgregory.net/rapid` | v1.2.0 | TODO | |

### Test-only

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/stretchr/testify` | v1.11.1 | TODO | |
| `github.com/tj/assert` | v0.0.3 | TODO | |
| `github.com/davecgh/go-spew` | v1.1.1 | TODO | |
| `github.com/pmezard/go-difflib` | v1.0.0 | TODO | |
| `github.com/quasilyte/go-ruleguard/dsl` | v0.3.22 | TODO | |
| `github.com/groob/finalizer` | v0.0.0 | TODO | |

### Stdlib extensions (`golang.org/x`)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `golang.org/x/exp` | v0.0.0 | TODO | |
| `golang.org/x/mod` | v0.36.0 | TODO | |
| `golang.org/x/net` | v0.55.0 | TODO | |
| `golang.org/x/sync` | v0.21.0 | TODO | |
| `golang.org/x/sys` | v0.45.0 | TODO | |
| `golang.org/x/term` | v0.43.0 | TODO | |
| `golang.org/x/text` | v0.38.0 | TODO | |
| `golang.org/x/tools` | v0.45.0 | TODO | |

---

## Inlined third-party code

These libraries have been copied into Fleet's codebase. They are not pulled via `go get` but their versions are tracked in `third_party/vuln-check/go.mod` for vulnerability scanning.

| Library | Inlined location | Version | Smoke tests | Risk |
|---------|-----------------|---------|-------------|------|
| `github.com/micromdm/nanomdm` | `server/mdm/nanomdm/` | v0.9.0 | TODO | |
| `github.com/micromdm/nanodep` | `server/mdm/nanodep/` | v0.4.0 | TODO | |
| `github.com/micromdm/scep/v2` | `server/mdm/scep/` | v2.3.0 | TODO | |
| `github.com/pressly/goose/v3` | `server/goose/` | v3.17.0 | TODO | |
| `github.com/facebookincubator/nvdtools` | `server/vulnerabilities/nvd/tools/` | v0.1.5 | TODO | |
| `github.com/virtuald/go-paniclog` | `orbit/pkg/go-paniclog/` | v0.0.0 | TODO | |
| `github.com/josharian/impl` | `server/mock/mockimpl/` | v1.4.0 | TODO | |
| `github.com/mitchellh/gon` | `orbit/pkg/packaging/` (partial) | v0.2.3 | TODO | |

---

## Goval-dictionary (OVAL vuln data)

Module: `third_party/goval-dictionary/go.mod`. Used for processing OVAL vulnerability data feeds.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/cheggaaa/pb/v3` | v3.1.7 | TODO | |
| `github.com/glebarez/sqlite` | v1.11.0 | TODO | |
| `github.com/go-redis/redis/v8` | v8.11.5 | TODO | |
| `github.com/google/go-cmp` | v0.7.0 | TODO | |
| `github.com/hashicorp/go-version` | v1.8.0 | TODO | |
| `github.com/inconshreveable/log15` | v3.0.0 | TODO | |
| `github.com/k0kubun/pp` | v3.0.1 | TODO | |
| `github.com/klauspost/compress` | v1.18.3 | TODO | |
| `github.com/knqyf263/go-rpm-version` | v0.0.0 | TODO | |
| `github.com/labstack/echo/v4` | v4.15.0 | TODO | |
| `github.com/mitchellh/go-homedir` | v1.1.0 | TODO | |
| `github.com/spf13/cobra` | v1.10.2 | TODO | |
| `github.com/spf13/viper` | v1.21.0 | TODO | |
| `github.com/ulikunitz/xz` | v0.5.15 | TODO | |
| `golang.org/x/net` | v0.48.0 | TODO | |
| `golang.org/x/xerrors` | v0.0.0 | TODO | |
| `gopkg.in/yaml.v2` | v2.4.0 | TODO | |
| `gorm.io/driver/mysql` | v1.5.5 | TODO | |
| `gorm.io/driver/postgres` | v1.5.7 | TODO | |
| `gorm.io/gorm` | v1.25.7 | TODO | |

---

## Tools

### Fleet MCP server

Module: `tools/fleet-mcp/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/gorilla/websocket` | v1.5.1 | TODO | |
| `github.com/joho/godotenv` | v1.5.1 | TODO | |
| `github.com/mark3labs/mcp-go` | v0.44.0 | TODO | |
| `github.com/sirupsen/logrus` | v1.9.3 | TODO | |
| `golang.org/x/time` | v0.15.0 | TODO | |

### Terraform provider

Module: `tools/terraform/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/hashicorp/terraform-plugin-framework` | v1.7.0 | TODO | |
| `github.com/hashicorp/terraform-plugin-go` | v0.22.1 | TODO | |
| `github.com/hashicorp/terraform-plugin-log` | v0.9.0 | TODO | |
| `github.com/hashicorp/terraform-plugin-testing` | v1.7.0 | TODO | |
| `github.com/stretchr/testify` | v1.8.1 | TODO | |

### CI linter plugins

Modules: `tools/ci/apiparamcheck/go.mod`, `tools/ci/setboolcheck/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/golangci/plugin-module-register` | v0.1.2 | TODO | |
| `golang.org/x/tools` | v0.42.0 | TODO | |

### GitHub management TUI

Module: `tools/github-manage/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/charmbracelet/bubbles` | v0.21.0 | TODO | |
| `github.com/charmbracelet/bubbletea` | v1.3.6 | TODO | |
| `github.com/charmbracelet/glamour` | v0.10.0 | TODO | |
| `github.com/charmbracelet/lipgloss` | v1.1.1 | TODO | |
| `github.com/mattn/go-isatty` | v0.0.20 | TODO | |
| `github.com/spf13/cobra` | v1.9.1 | TODO | |
| `github.com/spf13/pflag` | v1.0.6 | TODO | |

### QA check

Module: `tools/qacheck/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/shurcooL/githubv4` | v0.0.0 | TODO | |
| `golang.org/x/oauth2` | v0.35.0 | TODO | |

### Snapshot tool

Module: `tools/snapshot/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/manifoldco/promptui` | v0.9.0 | TODO | |
| `golang.org/x/sys` | v0.28.0 | TODO | |

### Dibble test-data seeder

Module: `tools/dibble/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/AlecAivazis/survey/v2` | v2.3.7 | TODO | |
| `github.com/go-sql-driver/mysql` | v1.10.0 | TODO | |
| `github.com/google/uuid` | v1.6.0 | TODO | |
| `github.com/spf13/cobra` | v1.10.2 | TODO | |
| `github.com/spf13/viper` | v1.21.0 | TODO | |
| `gopkg.in/yaml.v3` | v3.0.1 | TODO | |

### Hangar desktop app

Module: `tools/hangar/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/Masterminds/semver/v3` | v3.5.0 | TODO | |
| `github.com/wailsapp/wails/v3` | v3.0.0-alpha.98 | TODO | |
| `gopkg.in/yaml.v3` | v3.0.1 | TODO | |

### Screencap tool

Module: `tools/screencap/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/chromedp/cdproto` | v0.0.0 | TODO | |
| `github.com/chromedp/chromedp` | v0.14.2 | TODO | |

---

## Frontend (runtime)

From `package.json` `dependencies`. These ship to users in the browser.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `@sgress454/node-sql-parser` | 5.4.0-fork.2 | TODO | |
| `ace-builds` | 1.4.14 | TODO | |
| `axios` | 1.16.1 | TODO | |
| `cmdk` | 1.1.1 | TODO | |
| `content-disposition` | 0.5.4 | TODO | |
| `core-js` | 3.25.1 | TODO | |
| `date-fns` | 3.6.0 | TODO | |
| `date-fns-tz` | 3.1.3 | TODO | |
| `dompurify` | 3.4.11 | TODO | |
| `es6-object-assign` | 1.1.0 | TODO | |
| `es6-promise` | 4.2.8 | TODO | |
| `file-saver` | 1.3.8 | TODO | |
| `history` | 2.1.0 | TODO | |
| `isomorphic-fetch` | 3.0.0 | TODO | |
| `js-cookie` | 3.0.7 | TODO | |
| `js-md5` | 0.7.3 | TODO | |
| `js-yaml` | 4.2.0 | TODO | |
| `lodash` | 4.18.1 | TODO | |
| `memoize-one` | 5.2.1 | TODO | |
| `normalizr` | 3.6.2 | TODO | |
| `postcss` | 8.5.10 | TODO | |
| `prop-types` | 15.8.1 | TODO | |
| `proxy-middleware` | 0.15.0 | TODO | |
| `rc-pagination` | 1.16.3 | TODO | |
| `react` | 18.3.1 | TODO | |
| `react-accessible-accordion` | 3.3.5 | TODO | |
| `react-ace` | 9.3.0 | TODO | |
| `react-dom` | 18.2.0 | TODO | |
| `react-error-boundary` | 3.1.4 | TODO | |
| `react-markdown` | 10.1.0 | TODO | |
| `react-query` | 3.39.3 | TODO | |
| `react-router` | 3.2.6 | TODO | |
| `react-router-transition` | 1.2.1 | TODO | |
| `react-select` | 1.3.0 | TODO | |
| `react-select-5` (npm:react-select) | 5.4.0 | TODO | |
| `react-table` | 7.7.0 | TODO | |
| `react-tabs` | 3.2.3 | TODO | |
| `react-tooltip` | 4.2.21 | TODO | |
| `react-tooltip-5` (npm:react-tooltip) | 5.29.1 | TODO | |
| `recharts` | 3.8.1 | TODO | |
| `remark-gfm` | 4.0.1 | TODO | |
| `sass` | 1.83.4 | TODO | |
| `select` | 1.1.2 | TODO | |
| `sockjs-client` | 1.6.1 | TODO | |
| `sonner` | 2.0.7 | TODO | |
| `use-debounce` | 9.0.4 | TODO | |
| `uuid` | 14.0.0 | TODO | |
| `validator` | 13.15.22 | TODO | |
| `when` | 3.7.8 | TODO | |

---

## Frontend (dev / build toolchain)

From `package.json` `devDependencies`. These affect the build pipeline and test suite, not the runtime product.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `@babel/cli` | 7.17.6 | TODO | |
| `@babel/core` | 7.18.10 | TODO | |
| `@babel/plugin-proposal-*` (13 plugins) | 7.16.7 - 7.17.6 | TODO | |
| `@babel/plugin-syntax-dynamic-import` | 7.8.3 | TODO | |
| `@babel/plugin-syntax-import-meta` | 7.10.4 | TODO | |
| `@babel/preset-env` | 7.21.5 | TODO | |
| `@babel/preset-react` | 7.18.6 | TODO | |
| `@babel/preset-typescript` | 7.21.5 | TODO | |
| `@storybook/*` (10 packages) | 8.0.4 - 8.6.17 | TODO | |
| `@testing-library/jest-dom` | 6.4.2 | TODO | |
| `@testing-library/react` | 15.0.2 | TODO | |
| `@testing-library/user-event` | 14.5.2 | TODO | |
| `@tsconfig/recommended` | 1.0.13 | TODO | |
| `@typescript-eslint/eslint-plugin` | 5.58.0 | TODO | |
| `@typescript-eslint/parser` | 5.58.0 | TODO | |
| `autoprefixer` | 10.4.19 | TODO | |
| `babel-core` | 7.0.0-bridge.0 | TODO | |
| `babel-eslint` | 9.0.0 | TODO | |
| `babel-jest` | 29.2.0 | TODO | |
| `babel-loader` | 8.2.3 | TODO | |
| `babel-plugin-dynamic-import-node` | 2.3.3 | TODO | |
| `bourbon` | 5.1.0 | TODO | |
| `classnames` | 2.2.5 | TODO | |
| `compare-versions` | 6.1.1 | TODO | |
| `css-loader` | 6.7.3 | TODO | |
| `esbuild-loader` | 4.4.2 | TODO | |
| `eslint` | 7.32.0 | TODO | |
| `eslint-config-airbnb` | 15.1.0 | TODO | |
| `eslint-config-prettier` | 8.5.0 | TODO | |
| `eslint-import-resolver-webpack` | 0.10.0 | TODO | |
| `eslint-plugin-import` | 2.25.4 | TODO | |
| `eslint-plugin-jest` | 20.0.3 | TODO | |
| `eslint-plugin-jsx-a11y` | 5.1.1 | TODO | |
| `eslint-plugin-prettier` | 3.4.1 | TODO | |
| `eslint-plugin-react` | 7.29.4 | TODO | |
| `eslint-plugin-react-hooks` | 4.3.0 | TODO | |
| `eslint-plugin-storybook` | 0.9.0 | TODO | |
| `expect` | 1.20.2 | TODO | |
| `fork-ts-checker-webpack-plugin` | 9.1.0 | TODO | |
| `html-webpack-plugin` | 5.5.0 | TODO | |
| `identity-obj-proxy` | 3.0.0 | TODO | |
| `ignore-styles` | 5.0.1 | TODO | |
| `jest` | 29.2.0 | TODO | |
| `jest-environment-jsdom` | 30.0.5 | TODO | |
| `jest-fixed-jsdom` | 0.0.8 | TODO | |
| `jsdom` | 26.1.0 | TODO | |
| `json-loader` | 0.5.7 | TODO | |
| `mini-css-extract-plugin` | 2.7.5 | TODO | |
| `msw` | 2.5.1 | TODO | |
| `node-bourbon` | 4.2.8 | TODO | |
| `node-sass-glob-importer` | 5.3.3 | TODO | |
| `postcss-loader` | 4.3.0 | TODO | |
| `prettier` | 2.2.1 | TODO | |
| `react-docgen-typescript-plugin` | 1.0.5 | TODO | |
| `regenerator-runtime` | 0.13.9 | TODO | |
| `sass-loader` | 13.2.2 | TODO | |
| `storybook` | 8.6.17 | TODO | |
| `trace-unhandled` | 2.0.1 | TODO | |
| `ts-loader` | 6.2.2 | TODO | |
| `typescript` | 6.0.2 | TODO | |
| `webpack` | 5.105.0 | TODO | |
| `webpack-cli` | 5.0.1 | TODO | |
| `webpack-notifier` | 1.12.0 | TODO | |

---

## Summary

| Component | Module | Direct deps |
|-----------|--------|-------------|
| Server + fleetctl + Orbit | `go.mod` | 182 |
| Inlined third-party | `third_party/vuln-check/go.mod` | 8 |
| Goval-dictionary | `third_party/goval-dictionary/go.mod` | 20 |
| Fleet MCP server | `tools/fleet-mcp/go.mod` | 5 |
| Terraform provider | `tools/terraform/go.mod` | 5 |
| CI linter plugins | `tools/ci/*/go.mod` | 2 |
| GitHub management TUI | `tools/github-manage/go.mod` | 7 |
| QA check | `tools/qacheck/go.mod` | 2 |
| Snapshot tool | `tools/snapshot/go.mod` | 2 |
| Dibble seeder | `tools/dibble/go.mod` | 6 |
| Hangar desktop app | `tools/hangar/go.mod` | 3 |
| Screencap tool | `tools/screencap/go.mod` | 2 |
| Frontend (runtime) | `package.json` dependencies | 49 |
| Frontend (dev) | `package.json` devDependencies | ~60 (grouped) |
| **Total** | | **~353 direct dependencies** |
