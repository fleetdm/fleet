# Library upgrade smoke tests

When upgrading a third-party dependency, use this document to find the smoke tests you need to run. Libraries are grouped by functional area. Find your library, run the listed smoke tests, and confirm everything works before merging.

> **How to use:** Press Ctrl+F / Cmd+F and search for the library name (e.g. `gorilla/mux`). The table tells you what to test and how risky the upgrade is.

> **Maintained as part of:** [#48943](https://github.com/fleetdm/fleet/issues/48943). When adding a new dependency, add it to the appropriate section below.

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

> These libraries are used for log shipping, file storage, email, secrets, and Google integrations. Each AWS service has a distinct Fleet feature. Test the specific feature that uses the service you're upgrading.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `cloud.google.com/go/pubsub` | v1.50.1 | Osquery log shipping to GCP Pub/Sub + Android MDM push notifications. **1.** Configure GCP Pub/Sub as a log destination -- verify osquery logs arrive. **2.** Enroll an Android device -- verify compliance events arrive from Google. | Medium |
| `github.com/aws/aws-sdk-go-v2` | v1.41.5 | Core AWS SDK used by all AWS services below. **1.** If upgrading this alone, run the smoke tests for every AWS service you use (S3, SES, Kinesis, etc.). **2.** Verify `go build` succeeds. | High |
| `github.com/aws/aws-sdk-go-v2/config` | v1.32.12 | AWS credential/config loading. Same as core SDK above. | High |
| `github.com/aws/aws-sdk-go-v2/credentials` | v1.19.12 | AWS credential providers. Same as core SDK above. | High |
| `github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign` | v1.8.3 | Signed CloudFront URLs for software installer and bootstrap package downloads. **1.** Configure CloudFront in front of S3 bucket. **2.** Download a software installer via the UI -- verify the signed URL works. | Medium |
| `github.com/aws/aws-sdk-go-v2/feature/rds/auth` | v1.6.16 | RDS IAM authentication. **1.** Configure Fleet with RDS IAM auth. **2.** Verify DB connection works and server serves requests. | Medium |
| `github.com/aws/aws-sdk-go-v2/feature/s3/manager` | v1.17.81 | Multipart S3 uploads for large files. **1.** Upload a large software installer (>5 MB). **2.** Verify it downloads correctly. | Medium |
| `github.com/aws/aws-sdk-go-v2/service/firehose` | v1.37.7 | Osquery log shipping to AWS Kinesis Data Firehose. **1.** Configure Firehose as osquery log destination. **2.** Enroll a host and generate result logs. **3.** Verify logs arrive in the Firehose delivery stream. | Medium |
| `github.com/aws/aws-sdk-go-v2/service/kinesis` | v1.43.5 | Osquery log shipping to AWS Kinesis Data Streams. **1.** Configure Kinesis as osquery log destination. **2.** Enroll a host and generate result logs. **3.** Verify logs arrive in the Kinesis stream. | Medium |
| `github.com/aws/aws-sdk-go-v2/service/lambda` | v1.88.5 | Osquery log shipping to AWS Lambda. **1.** Configure Lambda as osquery log destination. **2.** Verify logs trigger the Lambda function. | Medium |
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.97.3 | File storage: carves, MDM bootstrap packages, software installers, org logos. **1.** Upload a software installer to S3. **2.** Download it via the UI. **3.** Upload an MDM bootstrap package. **4.** Trigger a file carve and verify the result is stored. | High |
| `github.com/aws/aws-sdk-go-v2/service/secretsmanager` | v1.35.8 | Resolves `secret://arn:aws:...` URIs in Fleet config. **1.** Configure a DB password via Secrets Manager ARN. **2.** Verify Fleet starts and connects using the resolved secret. | Medium |
| `github.com/aws/aws-sdk-go-v2/service/ses` | v1.30.4 | Transactional email via AWS SES (invitations, password resets). **1.** Configure SES as email backend. **2.** Trigger a password reset email. **3.** Verify the email arrives. | Medium |
| `github.com/aws/aws-sdk-go-v2/service/sts` | v1.41.9 | Cross-account AWS access via AssumeRole. **1.** Configure `sts_assume_role_arn`. **2.** Verify Fleet can access S3/Kinesis/etc. in the target account. | Medium |
| `github.com/aws/smithy-go` | v1.24.2 | Low-level AWS protocol layer. Same as core SDK -- test any AWS integration you use. | High |
| `google.golang.org/api` | v0.269.0 | Google APIs: Android Management (Android MDM), Calendar (compliance windows), Admin Directory (Workspace user sync). **1.** Android MDM: enroll an Android device. **2.** Google Calendar: trigger a calendar event from a failing policy. **3.** Google Workspace: verify user sync from directory. | High |
| `google.golang.org/grpc` | v1.79.3 | gRPC transport for OTel exporters and Google Cloud APIs. **1.** If OTel is enabled, verify traces/metrics export successfully. **2.** Verify Google API calls (Android MDM, Calendar) work. | Medium |

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

> MDM libraries are security-critical and affect device enrollment, profile delivery, and push notifications. Always test enrollment end-to-end after upgrading any of these.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/micromdm/micromdm` | v1.9.0 | App manifest generation and mobileconfig profile signing. **1.** Deploy a configuration profile to a macOS host -- verify it is signed and installs. **2.** Upload a macOS app package -- verify the manifest is generated correctly. | High |
| `github.com/micromdm/nanolib` | v0.2.0 | Structured logging interface for the embedded NanoMDM server. **1.** Apple MDM check-in succeeds. **2.** Server logs show structured MDM events. | Medium |
| `github.com/micromdm/plist` | v0.2.3 | Plist encoding/decoding inside NanoMDM for MDM commands and check-in payloads. **1.** Send an MDM command (e.g. Lock) to a macOS device -- verify it executes. **2.** Apple MDM check-in succeeds and device info is recorded. | High |
| `github.com/groob/plist` | v0.0.0 | Plist parsing for the macOS user profiles osquery extension table. **1.** Query `macos_user_profiles` on a macOS host -- verify installed profiles are returned. | Low |
| `howett.net/plist` | v1.0.1 | Most widely used plist library -- dataflatten (Darwin osquery tables), fleetctl MDM commands, VPP app parsing. **1.** Query any Darwin-specific osquery extension table (e.g. `battery`). **2.** `fleetctl mdm` commands work. **3.** VPP apps sync correctly. | High |
| `github.com/smallstep/scep` | v0.0.0 | SCEP certificate enrollment for Apple MDM and host identity (EE). **1.** Enroll a macOS/iOS device via DEP -- verify SCEP cert is issued. **2.** (EE) Verify fleetd host identity certificate is issued via SCEP. | Critical |
| `github.com/smallstep/pkcs7` | v0.0.0 | PKCS7 signature parsing for both Apple and Windows MDM message validation. **1.** Apple MDM: device sends signed check-in -- verify it's accepted. **2.** Windows MDM: WSTEP enrollment completes (CSR is PKCS7-wrapped). | Critical |
| `go.step.sm/crypto` | v0.77.1 | JWT/JWK operations for the ACME device attestation subsystem. **1.** Apple MDM ACME enrollment flow completes on a supported device. | High |
| `software.sslmate.com/src/go-pkcs12` | v0.4.0 | PKCS12 cert bundle encoding (DigiCert EE) and parsing (cryptoinfo osquery table). **1.** (EE) DigiCert CA integration issues a .pfx cert. **2.** Query `cryptoinfo` table on a host with PKCS12 files. | Medium |
| `github.com/RobotsAndPencils/buford` | v0.14.0 | APNs HTTP/2 push notifications to wake Apple devices. **1.** Send an MDM command -- verify the device wakes up and fetches it (push notification fires). **2.** Bulk push to multiple devices works. | Critical |
| `github.com/fxamacker/cbor/v2` | v2.9.1 | CBOR decoding for WebAuthn/FIDO2 attestation in ACME device attestation. **1.** Apple ACME device attestation challenge completes. | Medium |

### Authentication / Security / Crypto

> These libraries handle SSO, JWT tokens, password hashing, and host identity verification. Security-critical -- always test auth flows end-to-end.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/crewjam/saml` | v0.5.1 | SAML SSO login and (EE) Okta Conditional Access IdP. **1.** SSO login flow: redirect to IdP, callback, session created. **2.** SSO logout works. **3.** (EE) Conditional Access: Fleet acts as IdP for Okta -- verify response is accepted. | Critical |
| `github.com/russellhaering/goxmldsig` | v1.6.0 | XML digital signature validation on SAML assertions. **1.** SSO login with a real IdP -- verify signature validation passes. **2.** Tamper with a SAML response -- verify it's rejected. | High |
| `github.com/golang-jwt/jwt/v4` | v4.5.2 | JWTs for Windows MDM enrollment and license validation. **1.** Windows MDM enrollment completes (issues STS auth token). **2.** Fleet starts with a valid license key. **3.** Fleet rejects an invalid license key. | High |
| `github.com/MicahParks/jwkset` | v0.11.0 | Azure AD JWKS token verification during Windows MDM enrollment. **1.** Windows MDM enrollment with Azure AD Conditional Access completes. | Medium |
| `github.com/gomodule/oauth1` | v0.2.0 | OAuth 1.0a for Apple DEP API authentication. **1.** DEP sync fetches device serials from Apple. **2.** DEP profile assignment works. | High |
| `golang.org/x/oauth2` | v0.35.0 | OAuth2 for GCS storage, Google Calendar, Google Workspace directory sync. **1.** If using GCS for installers: upload and download a file. **2.** (EE) Google Calendar compliance event is created. **3.** (EE) Google Workspace user sync runs. | High |
| `golang.org/x/crypto` | v0.52.0 | bcrypt password hashing, PBKDF2 for MDM keys, ASN.1 parsing for Windows MDM CSRs. **1.** Create a user with password -- login works. **2.** Change password -- old password rejected, new accepted. **3.** Windows MDM enrollment (CSR parsing). **4.** `fleetctl` password prompt works in terminal. | Critical |
| `github.com/sethvargo/go-password` | v0.3.0 | Random password generation in `fleetctl user create --api-only`. **1.** Run `fleetctl user create --api-only` -- verify generated password works for API auth. | Low |
| `github.com/remitly-oss/httpsig-go` | v1.2.0 | HTTP signature signing (orbit) and verification (server) for host identity auth. **1.** (EE) Enroll a host with host identity enabled. **2.** Verify signed requests from the host are accepted by the server. **3.** Verify unsigned/tampered requests are rejected. | High |
| `github.com/sassoftware/relic/v8` | v8.0.1 | OLE/CFB reader for extracting MSI installer metadata (name, version). **1.** Upload a `.msi` software installer -- verify name and version are extracted correctly. | Medium |

### SCIM

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/elimity-com/scim` | v0.0.0 | (EE) SCIM 2.0 server for user/group provisioning from IdPs (Okta, etc.). **1.** Configure SCIM with an IdP. **2.** Provision a user -- verify it appears in Fleet. **3.** Deprovision a user -- verify it's removed. **4.** Provision a group -- verify members are assigned. | High |
| `github.com/scim2/filter-parser/v2` | v2.2.0 | Parses SCIM filter expressions (e.g. `userName eq "alice"`). **1.** IdP queries SCIM endpoint with a filter -- verify correct results. **2.** Test multiple filter operators (eq, co, sw). | Medium |

### Vulnerability scanning / NVD / OPA

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/pandatix/nvdapi` | v0.6.4 | NVD API 2.0 client for CVE and CPE data sync. **1.** Vulnerability sync cron runs without errors. **2.** CVEs appear for known-vulnerable software (e.g. old Chrome version). **3.** Run `go test ./server/vulnerabilities/nvd/...` | High |
| `github.com/open-policy-agent/opa` | v1.4.2 | The entire Fleet authorization system -- every API endpoint checks permissions via OPA/Rego. **1.** Login as admin -- verify full access. **2.** Login as observer -- verify write endpoints return 403. **3.** Login as GitOps user -- verify restricted access. **4.** Run `go test ./server/authz/...` | Critical |
| `github.com/oschwald/geoip2-golang` | v1.8.0 | MaxMind GeoIP lookups for host location display. **1.** Configure a MaxMind DB path. **2.** View a host with a public IP -- verify country/city appear. **3.** Without MaxMind config, verify no errors (graceful no-op). | Low |
| `github.com/saferwall/pe` | v1.5.5 | PE file parsing for Windows `.exe` metadata extraction (software inventory). **1.** Upload a Windows `.exe` installer -- verify product name and version are extracted. | Medium |

### Osquery

> **osquery-go** is the core IPC layer between Fleet and osquery. If it breaks, no host data flows.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/osquery/osquery-go` | v0.0.0 | Thrift IPC for all osquery extension tables in Orbit + Launcher bridge. **1.** Enroll a host with Orbit -- verify host data appears. **2.** Query a Fleet-specific extension table (e.g. `puppet_info`, `sntp_request`). **3.** Launcher-enrolled host checks in successfully (if applicable). | Critical |
| `github.com/AbGuthrie/goquery/v2` | v2.0.1 | Interactive osquery REPL in `fleetctl goquery`. **1.** Run `fleetctl goquery` targeting a live host. **2.** Execute a SQL query -- verify results return. | Low |
| `github.com/kolide/launcher` | v1.0.12 | Compatibility shim for Kolide Launcher gRPC clients. **1.** If using Launcher: verify enrollment, config fetch, and log submission. **2.** If not using Launcher, verify `go build` succeeds. | Medium |
| `github.com/macadmins/osquery-extension` | v1.4.1 | Puppet, Chrome profiles, and file_lines extension tables. **1.** Query `puppet_info` on a macOS host with Puppet. **2.** Query `google_chrome_profiles`. **3.** Query `file_lines` with a target file. | Medium |

### Observability / Telemetry

> All OpenTelemetry (`go.opentelemetry.io/*`) packages are used as a unified bundle initialized in `cmd/fleet/otel.go`. Upgrade them together and run one shared smoke test. Elastic APM is a separate, independent stack.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/getsentry/sentry-go` | v0.18.0 | Optional error reporting sink. **1.** Configure a Sentry DSN. **2.** Trigger a server error -- verify it appears in Sentry. **3.** Without Sentry config, verify no errors at startup. | Low |
| `github.com/prometheus/client_golang` | v1.21.1 | Metrics endpoint (`/metrics`). Always on. **1.** Hit `/metrics` -- verify Prometheus metrics are returned. **2.** Verify request count/duration histograms update after API calls. | Medium |
| `github.com/rs/zerolog` | v1.32.0 | Structured logger for the Orbit agent (not the server). **1.** Start Orbit -- verify structured JSON logs are emitted. **2.** Verify log levels work (debug/info/warn). | Medium |
| `go.elastic.co/apm/v2` | v2.7.0 | Elastic APM tracing for HTTP requests and error events. **1.** Configure Elastic APM. **2.** Make API requests -- verify APM transactions appear. **3.** Trigger an error -- verify it's recorded as an APM error event. | Low |
| `go.elastic.co/apm/module/apmgorilla/v2` | v2.6.2 | Gorilla mux APM instrumentation. Same smoke test as `go.elastic.co/apm/v2`. | Low |
| `go.elastic.co/apm/module/apmhttp/v2` | v2.7.1 | HTTP transport APM instrumentation. Same smoke test as `go.elastic.co/apm/v2`. | Low |
| `go.elastic.co/apm/module/apmsql/v2` | v2.6.2 | SQL query APM instrumentation. Same smoke test as `go.elastic.co/apm/v2`. | Low |
| `go.opentelemetry.io/otel` | v1.43.0 | **All OTel packages share this smoke test:** **1.** Enable OTel (`FLEET_OTEL_ENABLED=true`) with an OTLP gRPC endpoint. **2.** Make API requests -- verify trace spans appear in the backend. **3.** Verify metrics are exported. **4.** Verify server logs are forwarded (if log export is enabled). **5.** Without OTel config, verify server starts cleanly (no-op). | Medium |
| `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc` | v0.16.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc` | v1.40.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace` | v1.40.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.40.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/log` | v0.16.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/metric` | v1.43.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/sdk` | v1.43.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/sdk/log` | v0.16.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/sdk/metric` | v1.43.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/otel/trace` | v1.43.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/contrib/bridges/otelslog` | v0.15.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux` | v0.60.0 | Same as `go.opentelemetry.io/otel` above. | Medium |
| `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` | v0.61.0 | Same as `go.opentelemetry.io/otel` above. | Medium |

### NATS (Live query)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/nats-io/nats-server/v2` | v2.12.6 | Test-only -- spins up an in-process NATS server for log writer tests. **1.** Run `go test ./server/logging/ -run TestNATS` | Low |
| `github.com/nats-io/nats.go` | v1.49.0 | NATS/JetStream osquery log destination (alternative to Kinesis/Firehose). **1.** Configure NATS as osquery log destination. **2.** Enroll a host -- verify result logs arrive in the NATS subject. **3.** Test with JetStream enabled and with compression options. | Medium |

### Packaging / Build

> These libraries are used at build/package time by `fleetctl package`, not at agent runtime.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/goreleaser/nfpm/v2` | v2.20.0 | Core package builder for `fleetctl package --type=deb/rpm`. **1.** Build a `.deb` package with `fleetctl package --type=deb`. **2.** Build an `.rpm` package. **3.** Install the resulting package on a test VM -- verify Orbit starts. | High |
| `github.com/cavaliergopher/rpm` | v1.2.0 | RPM header parsing for software inventory. **1.** Upload a `.rpm` installer -- verify name and version are extracted. | Medium |
| `github.com/blakesmith/ar` | v0.0.0 | `ar` archive reader for `.deb` metadata extraction. **1.** Upload a `.deb` installer -- verify name and version are extracted. | Medium |
| `github.com/josephspurrier/goversioninfo` | v1.4.0 | Windows version resource embedding during build. **1.** Build a Windows `.msi` or `.exe` -- verify version info is embedded. | Low |
| `github.com/mitchellh/gon` | v0.2.6 | macOS notarization and stapling for fleet-desktop and orbit. **1.** Build a macOS `.pkg` with notarization enabled. **2.** Verify the package is notarized and stapled. | Medium |
| `github.com/ulikunitz/xz` | v0.5.15 | XZ compression/decompression. **1.** `fleetctl package` completes for all package types. **2.** Verify `go build` succeeds. | Low |
| `github.com/xi2/xz` | v0.0.0 | Alternate XZ decompression. Same as above. | Low |
| `github.com/klauspost/compress` | v1.18.4 | General-purpose compression (zstd, gzip, snappy, etc.). **1.** Verify NATS log compression works (if configured). **2.** Package builds complete. **3.** `go build` succeeds. | Medium |

### CLI / Terminal

> **cobra** (Fleet server CLI) and **urfave/cli** (fleetctl) are the two CLI frameworks. They don't overlap.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/spf13/cobra` | v1.9.1 | CLI framework for the `fleet` server binary (`fleet serve`, `fleet prepare db`, etc.). **1.** `fleet serve` starts the server. **2.** `fleet prepare db` runs migrations. **3.** `fleet vuln_process` runs. **4.** `fleet --help` shows all subcommands. | High |
| `github.com/spf13/pflag` | v1.0.6 | Flag parsing for cobra commands. Same smoke tests as cobra. | High |
| `github.com/spf13/viper` | v1.20.1 | Config loading: merges config file + env vars + flags. **1.** Set a config value via env var (`FLEET_MYSQL_ADDRESS`) -- verify it's used. **2.** Set via config file -- verify it's used. **3.** Verify flag overrides env var. | High |
| `github.com/spf13/cast` | v1.7.1 | Type casting for viper config values. Same smoke tests as viper. | Low |
| `github.com/urfave/cli/v2` | v2.27.7 | CLI framework for `fleetctl` (every subcommand). **1.** `fleetctl get hosts` works. **2.** `fleetctl apply -f` works. **3.** `fleetctl query` works. **4.** `fleetctl gitops` works. **5.** `fleetctl --help` shows all subcommands. | High |
| `github.com/briandowns/spinner` | v1.23.1 | Terminal spinner during long operations. **1.** Run `fleetctl query` -- verify spinner displays while waiting. **2.** Terminal UX only -- no functional impact. | Low |
| `github.com/manifoldco/promptui` | v0.9.0 | Interactive prompts in `fleetctl new`. **1.** Run `fleetctl new` -- verify it prompts for org name interactively. | Low |
| `github.com/olekukonko/tablewriter` | v0.0.5 | ASCII table rendering for `fleetctl get` and live query results. **1.** `fleetctl get hosts` -- verify table renders correctly. **2.** `fleetctl query` -- verify results display as a table. | Low |
| `github.com/gosuri/uilive` | v0.0.4 | Live-updating terminal output for streaming live query results. **1.** Run `fleetctl query` -- verify results update in-place as hosts respond. | Low |
| `github.com/fatih/color` | v1.16.0 | Terminal color output. **1.** CLI output has colored text (warnings, errors). No functional impact. | Low |
| `github.com/skratchdot/open-golang` | v0.0.0 | Opens URLs in the default browser. **1.** `fleetctl preview` opens the Fleet UI in the browser. | Low |

### XML / YAML / CSV / Data formats

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/beevik/etree` | v1.6.0 | XML tree manipulation (used by SAML and MDM). **1.** SSO login works. **2.** MDM profile XML is generated correctly. | Medium |
| `github.com/antchfx/xmlquery` | v1.3.14 | XML XPath queries (SCEP, MDM). **1.** MDM check-in XML payloads are parsed. **2.** `go test ./server/mdm/...` passes. | Medium |
| `github.com/clbanning/mxj` | v1.8.4 | XML-to-map conversion. **1.** `go build` succeeds. **2.** Run related integration tests. | Low |
| `github.com/ghodss/yaml` | v1.0.0 | YAML parsing (JSON-compatible). **1.** `fleetctl apply -f config.yml` works. **2.** `fleetctl gitops` processes YAML files. | Medium |
| `gopkg.in/yaml.v2` | v2.4.0 | YAML v2 parsing used throughout. Same as above. | Medium |
| `github.com/gocarina/gocsv` | v0.0.0 | CSV generation for host exports and query results. **1.** Export hosts as CSV -- verify the file is well-formed. **2.** Download query results as CSV. | Medium |
| `github.com/go-json-experiment/json` | v0.0.0 | Experimental JSON library. **1.** `go build` succeeds. **2.** API responses are valid JSON. | Low |
| `github.com/go-ini/ini` | v1.67.0 | INI file parsing. **1.** `go build` succeeds. | Low |
| `gopkg.in/ini.v1` | v1.67.0 | INI file parsing (alternate). Same as above. | Low |

### Integrations (Jira, Zendesk, GitHub)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/andygrunwald/go-jira` | v1.16.0 | Jira automation: creates tickets for vulnerabilities and failing policies. **1.** Configure Jira integration. **2.** Trigger a failing-policy automation -- verify a Jira issue is created. **3.** Trigger a vulnerability automation -- verify a Jira issue is created with CVE details. | High |
| `github.com/nukosuke/go-zendesk` | v0.13.1 | Zendesk automation: creates tickets for vulnerabilities and failing policies. **1.** Configure Zendesk integration. **2.** Trigger a failing-policy automation -- verify a Zendesk ticket is created. **3.** Trigger a vulnerability automation -- verify a ticket is created. | High |
| `github.com/google/go-github/v37` | v37.0.0 | GitHub API for vulnerability data downloads (NVD, MSRC, OVAL, Office) + maintained app ingestion. **1.** Vulnerability sync runs -- verify it downloads data from `fleetdm/nvd` GitHub releases. **2.** (EE) Maintained apps: verify winget manifests are fetched from GitHub. **3.** `fleetctl preview` works (downloads releases). | High |

### Semver / Versioning

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/Masterminds/semver` | v1.5.0 | Semver parsing used across the codebase. **1.** Software inventory version comparisons work. **2.** `go test` passes for packages that use semver. | Medium |
| `github.com/Masterminds/semver/v3` | v3.3.1 | Semver v3 parsing. Same as above. | Medium |

### System / OS / Hardware

> Many of these are platform-specific (Windows, macOS, Linux). Test on the relevant platform.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/shirou/gopsutil/v4` | v4.26.2 | Process listing/killing in Orbit (manages osquery/desktop subprocesses). **1.** Orbit starts and manages osquery lifecycle. **2.** Orbit restarts osquery after a crash. **3.** Cross-platform: test on macOS, Linux, Windows. | High |
| `github.com/digitalocean/go-smbios` | v0.0.0 | SMBIOS/DMI data for hardware inventory. **1.** Host detail page shows hardware model/serial. **2.** `go build` succeeds. | Low |
| `github.com/go-ole/go-ole` | v1.2.6 | Windows COM/OLE for Windows Update and BitLocker APIs. **1.** (Windows) Query `windows_updates` table. **2.** (Windows) BitLocker encryption status is reported. | Medium |
| `github.com/godbus/dbus/v5` | v5.1.0 | Linux D-Bus for detecting the active GUI session (fleet-desktop). **1.** (Linux) Fleet Desktop system tray icon appears for the logged-in user. | Low |
| `github.com/hectane/go-acl` | v0.0.0 | Windows file ACL management. **1.** (Windows) Orbit writes files with correct permissions. | Low |
| `github.com/hillu/go-ntdll` | v0.0.0 | Windows NT API. **1.** (Windows) `go build` succeeds. **2.** Orbit runs on Windows. | Low |
| `github.com/scjalliance/comshim` | v0.0.0 | Windows COM initialization. **1.** (Windows) COM-dependent features work (Windows Update, BitLocker). | Low |
| `github.com/mitchellh/go-ps` | v1.0.0 | Process listing. **1.** Orbit can detect running osquery processes. | Low |
| `github.com/siderolabs/go-blockdevice/v2` | v2.0.3 | Block device enumeration for disk encryption status. **1.** Host reports disk encryption status correctly. | Low |
| `fyne.io/systray` | v1.10.1 | System tray icon for Fleet Desktop (macOS/Windows/Linux). **1.** Fleet Desktop shows the tray icon. **2.** Clicking the icon opens the menu. **3.** "My device" link opens the browser. | High |
| `github.com/danieljoos/wincred` | v1.2.1 | Windows Credential Manager for secure storage. **1.** (Windows) Orbit stores/retrieves credentials correctly. | Low |
| `github.com/Azure/go-ntlmssp` | v0.1.1 | NTLM authentication. **1.** `go build` succeeds. | Low |

### TPM (Trusted Platform Module)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/google/go-tpm` | v0.9.8 | (EE) TPM 2.0 hardware-backed host identity keys. **1.** On a host with a TPM: enroll with host identity -- verify TPM-backed key is created. **2.** Verify HTTP signature auth uses the TPM key. **3.** Without a TPM, verify graceful fallback. | High |
| `github.com/foxboron/go-tpm-keyfiles` | v0.0.0 | TPM key file serialization. Same smoke tests as `go-tpm`. | High |

### KV store / Embedded DB

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/dgraph-io/badger/v2` | v2.2007.4 | Embedded KV store used by TUF update metadata in Orbit. **1.** Orbit auto-update check runs without errors. **2.** TUF metadata is persisted across Orbit restarts. | Medium |
| `github.com/boltdb/bolt` | v1.3.1 | Bolt KV store (legacy, used by TUF file store). **1.** Same as badger above -- Orbit TUF operations work. | Low |
| `go.etcd.io/bbolt` | v1.3.10 | BBolt (maintained fork of bolt). Same smoke tests as bolt. | Low |

### Docker / Containers

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/containerd/containerd` | v1.7.33 | Container runtime client (used for ChromeOS and container-based features). **1.** `go build` succeeds. **2.** Test container-related features if applicable. | Low |
| `github.com/docker/docker` | v28.0.0 | Docker client. **1.** `fleetctl preview` spins up Docker containers. **2.** `go build` succeeds. | Low |
| `github.com/docker/go-units` | v0.5.0 | Human-readable size/duration parsing. **1.** `go build` succeeds. | Low |

### TUF (The Update Framework)

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/theupdateframework/go-tuf` | v0.5.2 | Orbit auto-update: securely fetches and verifies updates from `updates.fleetdm.com`. **1.** Orbit checks for updates without errors. **2.** Simulate an update -- verify the new binary is downloaded, verified, and applied. **3.** Verify a tampered update is rejected. **4.** `fleetctl updates` commands work (EE). | Critical |

### Image / Assets

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/nfnt/resize` | v0.0.0 | Image resizing (org logo processing). **1.** Upload a custom org logo -- verify it renders at the correct size. | Low |
| `golang.org/x/image` | v0.42.0 | Image format support. Same as above. | Low |
| `github.com/elazarl/go-bindata-assetfs` | v1.0.1 | Serves embedded static assets (Fleet UI). **1.** Fleet UI loads in the browser (CSS, JS, images). | Medium |

### Expression / Pattern matching

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/expr-lang/expr` | v1.17.7 | Expression evaluation (NATS log subject routing, dynamic templates). **1.** If using NATS log subject templates, verify routing works. **2.** `go build` succeeds. | Low |
| `github.com/bmatcuk/doublestar/v4` | v4.10.0 | Glob pattern matching (GitOps file matching, etc.). **1.** `fleetctl gitops` with glob patterns works. **2.** `go build` succeeds. | Low |

### Log shipping

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/facebookincubator/flog` | v0.0.0 | Structured logging utility. **1.** `go build` succeeds. **2.** Server logs are emitted correctly. | Low |
| `gopkg.in/natefinch/lumberjack.v2` | v2.0.0 | Log file rotation for filesystem log destinations. **1.** Configure filesystem log output. **2.** Verify logs rotate when size limit is reached. | Low |

### Data structures / Utilities

> These are low-level utility libraries used broadly. Upgrading them rarely breaks specific features. A full test suite run is the best smoke test.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/RoaringBitmap/roaring` | v1.9.4 | Roaring bitmaps for live query host targeting in Redis. **1.** Run a live query targeting a label -- verify correct hosts respond. | Medium |
| `github.com/agnivade/levenshtein` | v1.2.1 | String distance for fuzzy matching. **1.** `go build` succeeds. | Low |
| `github.com/google/uuid` | v1.6.0 | UUID generation used everywhere. **1.** Create entities (hosts, policies, users) -- verify UUIDs are generated. **2.** `go test ./...` passes. | Medium |
| `github.com/google/go-cmp` | v0.7.0 | Deep equality comparison (primarily in tests). **1.** `go test ./...` passes. | Low |
| `github.com/hashicorp/go-multierror` | v1.1.1 | Multi-error aggregation. **1.** `go build` succeeds. **2.** Batch operations surface aggregated errors correctly. | Low |
| `github.com/cenkalti/backoff` | v2.2.1 | Exponential backoff for retries. **1.** `go build` succeeds. | Low |
| `github.com/cenkalti/backoff/v4` | v4.3.0 | Exponential backoff v4. Same as above. | Low |
| `github.com/oklog/run` | v1.1.0 | Goroutine lifecycle manager. **1.** Server starts and shuts down cleanly (all goroutines exit). | Low |
| `github.com/WatchBeam/clock` | v0.0.0 | Mock-friendly clock interface (used in tests). **1.** `go test ./...` passes. | Low |
| `github.com/beevik/ntp` | v0.3.0 | NTP time sync (Orbit `sntp_request` extension table). **1.** Query `sntp_request` table -- verify it returns NTP data. | Low |
| `github.com/gofrs/flock` | v0.12.1 | File-based locking. **1.** Orbit starts without lock contention errors. | Low |
| `github.com/golang/snappy` | v0.0.4 | Snappy compression. **1.** `go build` succeeds. | Low |
| `gopkg.in/guregu/null.v3` | v3.5.0 | Nullable types for JSON/SQL. **1.** API responses with nullable fields serialize correctly. | Low |
| `pgregory.net/rapid` | v1.2.0 | Property-based testing. **1.** `go test ./...` passes. | Low |

### Test-only

> These libraries are only compiled into `_test.go` files. The smoke test is: the test suite passes.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/stretchr/testify` | v1.11.1 | Assertion/mock framework used in virtually every test file. **1.** `go test ./server/fleet/...` passes. **2.** `go test ./server/service/...` passes. | Low |
| `github.com/tj/assert` | v0.0.3 | Thin wrapper around testify. **1.** `go test ./...` passes. | Low |
| `github.com/davecgh/go-spew` | v1.1.1 | Deep pretty-printer for test output (testify dependency). **1.** `go test ./...` passes. | Low |
| `github.com/pmezard/go-difflib` | v1.0.0 | Diff output for test assertions (testify dependency). **1.** `go test ./...` passes. | Low |
| `github.com/quasilyte/go-ruleguard/dsl` | v0.3.22 | Custom linter rule DSL. **1.** `make lint-go` passes. | Low |
| `github.com/groob/finalizer` | v0.0.0 | Test cleanup helper. **1.** `go test ./...` passes. | Low |

### Stdlib extensions (`golang.org/x`)

> These are quasi-standard library packages. They are broadly used and rarely break in isolation. Run the full test suite.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `golang.org/x/exp` | v0.0.0 | Experimental stdlib functions (maps, slices, etc.). **1.** `go build` succeeds. **2.** `go test ./...` passes. | Low |
| `golang.org/x/mod` | v0.36.0 | Go module version parsing. **1.** `go build` succeeds. | Low |
| `golang.org/x/net` | v0.55.0 | HTTP/2, HTML parsing, network utilities. **1.** Server starts and serves HTTPS. **2.** API calls over HTTP/2 work. **3.** SSO (SAML) works (uses x/net HTML parsing). | Medium |
| `golang.org/x/sync` | v0.21.0 | errgroup, singleflight, semaphore. **1.** Concurrent operations don't deadlock or race. **2.** `go test -race ./...` passes. | Medium |
| `golang.org/x/sys` | v0.45.0 | OS-level syscalls (all platforms). **1.** Orbit starts on all platforms. **2.** `go build` succeeds for all targets. | Medium |
| `golang.org/x/term` | v0.43.0 | Terminal handling (password input in fleetctl). **1.** `fleetctl login` password prompt works. | Low |
| `golang.org/x/text` | v0.38.0 | Unicode, encoding, language tags. **1.** Hosts with non-ASCII names display correctly. **2.** `go build` succeeds. | Low |
| `golang.org/x/tools` | v0.45.0 | Go tooling (used by linters and code generators). **1.** `make lint-go` passes. **2.** `make generate` succeeds. | Low |

---

## Inlined third-party code

These libraries have been copied into Fleet's codebase. They are not pulled via `go get` but their versions are tracked in `third_party/vuln-check/go.mod` for vulnerability scanning.

| Library | Inlined location | Version | Smoke tests | Risk |
|---------|-----------------|---------|-------------|------|
| `github.com/micromdm/nanomdm` | `server/mdm/nanomdm/` | v0.9.0 | Core Apple MDM server. **1.** Apple MDM check-in succeeds. **2.** Send MDM command (Lock, Erase) -- verify execution. **3.** DEP enrollment completes. | Critical |
| `github.com/micromdm/nanodep` | `server/mdm/nanodep/` | v0.4.0 | Apple DEP API client. **1.** DEP sync fetches device serials. **2.** DEP profile assignment works. **3.** ABM token renewal works. | Critical |
| `github.com/micromdm/scep/v2` | `server/mdm/scep/` | v2.3.0 | SCEP certificate authority. **1.** Apple MDM enrollment issues SCEP cert. **2.** Certificate renewal works. | Critical |
| `github.com/pressly/goose/v3` | `server/goose/` | v3.17.0 | Database migration framework. **1.** `fleet prepare db` runs all migrations. **2.** `goose up/down` works for individual migrations. **3.** Verify migration idempotency. | High |
| `github.com/facebookincubator/nvdtools` | `server/vulnerabilities/nvd/tools/` | v0.1.5 | NVD data format parsing for vulnerability matching. **1.** Vulnerability scan detects CVEs for known-vulnerable software. **2.** CPE matching works correctly. | High |
| `github.com/virtuald/go-paniclog` | `orbit/pkg/go-paniclog/` | v0.0.0 | Captures panic output in Orbit. **1.** Orbit starts without errors. **2.** `go build ./orbit/...` succeeds. | Low |
| `github.com/josharian/impl` | `server/mock/mockimpl/` | v1.4.0 | Mock stub generator (dev tooling). **1.** `go generate ./server/mock/...` succeeds. | Low |
| `github.com/mitchellh/gon` | `orbit/pkg/packaging/` (partial) | v0.2.3 | macOS notarization status parsing (partial copy). **1.** `go build ./orbit/...` succeeds. | Low |

---

## Goval-dictionary (OVAL vuln data)

Module: `third_party/goval-dictionary/go.mod`. Used for processing OVAL vulnerability data feeds.

> These libraries are internal to the goval-dictionary tool. The shared smoke test is: OVAL vulnerability scanning completes without errors and detects known vulnerabilities on Linux hosts.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/cheggaaa/pb/v3` | v3.1.7 | Progress bar for OVAL data download. **1.** OVAL data sync runs. | Low |
| `github.com/glebarez/sqlite` | v1.11.0 | SQLite backend for OVAL data storage. **1.** OVAL vulnerability scan completes. **2.** Linux host CVEs are detected. | Medium |
| `github.com/go-redis/redis/v8` | v8.11.5 | Redis client (optional caching). **1.** `go build` succeeds. | Low |
| `github.com/google/go-cmp` | v0.7.0 | Test comparisons. **1.** `go test` passes. | Low |
| `github.com/hashicorp/go-version` | v1.8.0 | Version comparison for OVAL matching. **1.** OVAL vuln scan matches correct versions. | Medium |
| `github.com/inconshreveable/log15` | v3.0.0 | Structured logging. **1.** OVAL sync runs with visible logs. | Low |
| `github.com/k0kubun/pp` | v3.0.1 | Pretty-printer for debugging. **1.** `go build` succeeds. | Low |
| `github.com/klauspost/compress` | v1.18.3 | Compression for OVAL data feeds. **1.** OVAL data download and extraction works. | Low |
| `github.com/knqyf263/go-rpm-version` | v0.0.0 | RPM version comparison for RHEL/CentOS OVAL matching. **1.** OVAL scan detects vulns on RHEL hosts. | Medium |
| `github.com/labstack/echo/v4` | v4.15.0 | HTTP framework (goval-dictionary server mode). **1.** `go build` succeeds. | Low |
| `github.com/mitchellh/go-homedir` | v1.1.0 | Home directory resolution. **1.** `go build` succeeds. | Low |
| `github.com/spf13/cobra` | v1.10.2 | CLI framework. **1.** goval-dictionary CLI runs. | Low |
| `github.com/spf13/viper` | v1.21.0 | Config loading. Same as cobra. | Low |
| `github.com/ulikunitz/xz` | v0.5.15 | XZ decompression for OVAL feeds. **1.** OVAL data sync works. | Low |
| `golang.org/x/net` | v0.48.0 | Network utilities. **1.** OVAL data downloads work. | Low |
| `golang.org/x/xerrors` | v0.0.0 | Error wrapping (legacy). **1.** `go build` succeeds. | Low |
| `gopkg.in/yaml.v2` | v2.4.0 | YAML parsing. **1.** Config files parse correctly. | Low |
| `gorm.io/driver/mysql` | v1.5.5 | MySQL GORM driver. **1.** OVAL data stored in MySQL works. | Medium |
| `gorm.io/driver/postgres` | v1.5.7 | PostgreSQL GORM driver. **1.** `go build` succeeds (not used in production). | Low |
| `gorm.io/gorm` | v1.25.7 | ORM framework for OVAL data storage. **1.** OVAL scan reads/writes data correctly. | Medium |

---

## Tools

### Fleet MCP server

Module: `tools/fleet-mcp/go.mod`

> All tools are internal. The shared smoke test for each is: `go build` succeeds and the tool's primary function works.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/gorilla/websocket` | v1.5.1 | WebSocket client for MCP live queries. **1.** Fleet MCP server starts. **2.** Execute a live query via MCP -- verify results. | Low |
| `github.com/joho/godotenv` | v1.5.1 | `.env` file loading. **1.** MCP server reads `.env` config. | Low |
| `github.com/mark3labs/mcp-go` | v0.44.0 | MCP protocol implementation. **1.** MCP server starts and responds to tool calls. | Low |
| `github.com/sirupsen/logrus` | v1.9.3 | Structured logging for MCP server. **1.** Logs are emitted. | Low |
| `golang.org/x/time` | v0.15.0 | Rate limiting. **1.** `go build` succeeds. | Low |

### Terraform provider

Module: `tools/terraform/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/hashicorp/terraform-plugin-framework` | v1.7.0 | Core Terraform provider framework. **1.** `terraform plan` with the Fleet provider works. **2.** `terraform apply` creates/updates Fleet resources. | Medium |
| `github.com/hashicorp/terraform-plugin-go` | v0.22.1 | Low-level Terraform plugin protocol. Same as above. | Medium |
| `github.com/hashicorp/terraform-plugin-log` | v0.9.0 | Provider logging. **1.** `TF_LOG=DEBUG terraform plan` shows Fleet provider logs. | Low |
| `github.com/hashicorp/terraform-plugin-testing` | v1.7.0 | Provider test framework. **1.** `go test ./tools/terraform/...` passes. | Low |
| `github.com/stretchr/testify` | v1.8.1 | Test assertions. **1.** `go test ./tools/terraform/...` passes. | Low |

### CI linter plugins

Modules: `tools/ci/apiparamcheck/go.mod`, `tools/ci/setboolcheck/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/golangci/plugin-module-register` | v0.1.2 | golangci-lint plugin registration. **1.** `make lint-go` passes. | Low |
| `golang.org/x/tools` | v0.42.0 | Go analysis framework. **1.** `make lint-go` passes. | Low |

### GitHub management TUI

Module: `tools/github-manage/go.mod`

> All charmbracelet libraries are used together for the TUI. Upgrade them as a group.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/charmbracelet/bubbles` | v0.21.0 | TUI components. **1.** `go build ./tools/github-manage/` succeeds. **2.** Tool launches and displays UI. | Low |
| `github.com/charmbracelet/bubbletea` | v1.3.6 | TUI framework. Same as above. | Low |
| `github.com/charmbracelet/glamour` | v0.10.0 | Markdown rendering. Same as above. | Low |
| `github.com/charmbracelet/lipgloss` | v1.1.1 | Terminal styling. Same as above. | Low |
| `github.com/mattn/go-isatty` | v0.0.20 | TTY detection. Same as above. | Low |
| `github.com/spf13/cobra` | v1.9.1 | CLI framework. Same as above. | Low |
| `github.com/spf13/pflag` | v1.0.6 | Flag parsing. Same as above. | Low |

### QA check

Module: `tools/qacheck/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/shurcooL/githubv4` | v0.0.0 | GitHub GraphQL API client. **1.** `go build ./tools/qacheck/` succeeds. **2.** Tool queries GitHub issues. | Low |
| `golang.org/x/oauth2` | v0.35.0 | GitHub auth. Same as above. | Low |

### Snapshot tool

Module: `tools/snapshot/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/manifoldco/promptui` | v0.9.0 | Interactive prompts. **1.** `go build ./tools/snapshot/` succeeds. | Low |
| `golang.org/x/sys` | v0.28.0 | OS syscalls. Same as above. | Low |

### Dibble test-data seeder

Module: `tools/dibble/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/AlecAivazis/survey/v2` | v2.3.7 | Interactive prompts. **1.** `go build ./tools/dibble/` succeeds. **2.** Dibble seeds test data into a local Fleet instance. | Low |
| `github.com/go-sql-driver/mysql` | v1.10.0 | MySQL connectivity. Same as above. | Low |
| `github.com/google/uuid` | v1.6.0 | UUID generation. Same as above. | Low |
| `github.com/spf13/cobra` | v1.10.2 | CLI framework. Same as above. | Low |
| `github.com/spf13/viper` | v1.21.0 | Config loading. Same as above. | Low |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing. Same as above. | Low |

### Hangar desktop app

Module: `tools/hangar/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/Masterminds/semver/v3` | v3.5.0 | Version parsing. **1.** `go build ./tools/hangar/` succeeds. | Low |
| `github.com/wailsapp/wails/v3` | v3.0.0-alpha.98 | Desktop app framework. **1.** Hangar app builds and launches. | Low |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing. Same as above. | Low |

### Screencap tool

Module: `tools/screencap/go.mod`

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `github.com/chromedp/cdproto` | v0.0.0 | Chrome DevTools Protocol. **1.** `go build ./tools/screencap/` succeeds. **2.** Screencap captures a page screenshot. | Low |
| `github.com/chromedp/chromedp` | v0.14.2 | Headless Chrome automation. Same as above. | Low |

---

## Frontend (runtime)

From `package.json` `dependencies`. These ship to users in the browser.

> **react**, **react-dom**, **react-query**, **react-router**, and **axios** are foundational. If any breaks, the entire UI is down. Test broadly after upgrading these.

| Library | Version | Smoke tests | Risk |
|---------|---------|-------------|------|
| `@sgress454/node-sql-parser` | 5.4.0-fork.2 | SQL parser for query editor "compatible platforms" detection. **1.** Open the query editor. **2.** Type a query using a platform-specific table -- verify the platform badge updates. | Medium |
| `ace-builds` | 1.4.14 | Code editor engine (SQL editor, YAML agent options editor). **1.** SQL query editor loads with syntax highlighting. **2.** Agent options YAML editor loads and is editable. | High |
| `axios` | 1.16.1 | HTTP client for ALL API calls from the frontend. **1.** Login works. **2.** Navigate to hosts, policies, software pages -- all data loads. **3.** Create/update operations work (add policy, etc.). | Critical |
| `cmdk` | 1.1.1 | Command palette (Cmd+K). **1.** Press Cmd+K -- palette opens. **2.** Search for a host by name -- verify results. **3.** Navigate to a result -- verify page loads. | Medium |
| `content-disposition` | 0.5.4 | File download header parsing. **1.** Download a CSV export -- verify filename is correct. | Low |
| `core-js` | 3.25.1 | Polyfills for older browsers. **1.** UI loads in all supported browsers. **2.** `make build` succeeds. | Medium |
| `date-fns` | 3.6.0 | Date formatting throughout the UI. **1.** Activity feed shows human-readable timestamps. **2.** Host detail "last seen" time is formatted. **3.** Check multiple date displays across pages. | Medium |
| `date-fns-tz` | 3.1.3 | Timezone-aware date formatting. Same as `date-fns`. | Medium |
| `dompurify` | 3.4.11 | XSS sanitization for rendered HTML (clickable URLs). **1.** View a host with a URL in a field -- verify it renders as a link. **2.** Inject a `<script>` tag in a field -- verify it's sanitized. | High |
| `es6-object-assign` | 1.1.0 | Object.assign polyfill. **1.** `make build` succeeds. | Low |
| `es6-promise` | 4.2.8 | Promise polyfill. **1.** `make build` succeeds. | Low |
| `file-saver` | 1.3.8 | Browser file downloads (CSV exports, scripts, profiles). **1.** Export hosts as CSV -- file downloads. **2.** Download a script from the scripts page. **3.** Download a configuration profile. | Medium |
| `history` | 2.1.0 | Browser history management for react-router. **1.** Navigate between pages -- back/forward buttons work. | Medium |
| `isomorphic-fetch` | 3.0.0 | Fetch polyfill. **1.** `make build` succeeds. API calls work. | Low |
| `js-cookie` | 3.0.7 | Session token cookie management. **1.** Login -- verify session cookie is set. **2.** Logout -- verify cookie is cleared. **3.** Page refresh preserves session. | High |
| `js-md5` | 0.7.3 | MD5 hashing (Gravatar). **1.** User avatar loads based on email hash. | Low |
| `js-yaml` | 4.2.0 | YAML parsing in agent options editor. **1.** Edit agent options YAML -- verify validation works. **2.** Save -- verify the YAML is applied. | Medium |
| `lodash` | 4.18.1 | Utility functions used in 234+ files. **1.** UI loads without errors. **2.** Navigate all major pages. **3.** `yarn test` passes. | High |
| `memoize-one` | 5.2.1 | Memoization for expensive computations. **1.** UI is responsive (no performance regression). | Low |
| `normalizr` | 3.6.2 | API response normalization. **1.** Data loads correctly on all pages. | Low |
| `postcss` | 8.5.10 | CSS processing. **1.** `make build` succeeds. **2.** UI styles render correctly. | Medium |
| `prop-types` | 15.8.1 | React prop type checking (legacy). **1.** `make build` succeeds. | Low |
| `proxy-middleware` | 0.15.0 | Dev server proxy. **1.** `make generate-dev` works. | Low |
| `rc-pagination` | 1.16.3 | Pagination component. **1.** Navigate paginated tables (hosts, software). **2.** Page numbers update correctly. | Low |
| `react` | 18.3.1 | Core React framework. **1.** UI loads. **2.** Navigate all major pages. **3.** Interactive components work (forms, modals, dropdowns). **4.** `yarn test` passes. | Critical |
| `react-accessible-accordion` | 3.3.5 | Appears unused -- no imports found. **1.** `make build` succeeds. Consider removing. | Low |
| `react-ace` | 9.3.0 | Code editor component wrapping ace-builds. Same tests as `ace-builds`. | High |
| `react-dom` | 18.2.0 | React DOM rendering. Same tests as `react`. | Critical |
| `react-error-boundary` | 3.1.4 | Error boundary for graceful crash handling. **1.** UI handles component errors without full-page crash. | Low |
| `react-markdown` | 10.1.0 | Markdown rendering (policy/query descriptions). **1.** View a policy with a markdown description -- verify it renders (bold, links, lists). | Medium |
| `react-query` | 3.39.3 | Server-state management for ALL data fetching. **1.** All pages load data. **2.** Mutations work (create, update, delete). **3.** Data refetches after mutations. **4.** Navigate away and back -- cached data shows. | Critical |
| `react-router` | 3.2.6 | Client-side routing for ALL pages. **1.** Navigate between hosts, policies, software, settings. **2.** Deep links work (paste a URL directly). **3.** Back/forward browser buttons work. **4.** 404 page shows for unknown routes. | Critical |
| `react-router-transition` | 1.2.1 | Page transition animations. **1.** Page transitions are smooth (no visual glitches). | Low |
| `react-select` | 1.3.0 | Dropdown/select controls (legacy). **1.** Team selector works. **2.** Label filter works. **3.** User role dropdown works. | Medium |
| `react-select-5` (npm:react-select) | 5.4.0 | Dropdown/select controls (v5). Same as above. | Medium |
| `react-table` | 7.7.0 | Data tables on every list page. **1.** Hosts table renders with sorting/filtering. **2.** Software table renders. **3.** Policies table renders. **4.** Column sorting works. | High |
| `react-tabs` | 3.2.3 | Tabbed UI (Add Hosts modal, MDM dashboard, label selector). **1.** Add Hosts modal tabs switch correctly. **2.** Dashboard MDM solution tabs work. | Low |
| `react-tooltip` | 4.2.21 | Tooltips (legacy). **1.** Hover over a truncated cell -- tooltip shows full text. | Low |
| `react-tooltip-5` (npm:react-tooltip) | 5.29.1 | Tooltips (v5). Same as above. | Low |
| `recharts` | 3.8.1 | Dashboard charts (hosts enrolled bar chart, historical line chart). **1.** Dashboard loads. **2.** Bar chart shows hosts by platform. **3.** Line chart shows historical host counts. | Medium |
| `remark-gfm` | 4.0.1 | GitHub Flavored Markdown (tables, strikethrough). Same tests as `react-markdown`. | Low |
| `sass` | 1.83.4 | SCSS compilation. **1.** `make build` succeeds. **2.** UI styles render correctly. | Medium |
| `select` | 1.1.2 | Text selection utility. **1.** Copy buttons work. | Low |
| `sockjs-client` | 1.6.1 | WebSocket client for live query result streaming in the UI. **1.** Run a live query from the UI. **2.** Verify results stream in real time. **3.** Verify query completes and stops. | High |
| `sonner` | 2.0.7 | Toast notifications. **1.** Perform an action (save, delete) -- verify toast appears. **2.** Toast auto-dismisses. | Low |
| `use-debounce` | 9.0.4 | Input debouncing. **1.** Type in search fields -- results update after a brief pause (not on every keystroke). | Low |
| `uuid` | 14.0.0 | Listed in package.json but no direct imports found. **1.** `make build` succeeds. Consider removing. | Low |
| `validator` | 13.15.22 | Form input validation (email, URL, hostname, UUID). **1.** Enter an invalid email in a form -- verify error message. **2.** Enter an invalid URL -- verify error. | Medium |
| `when` | 3.7.8 | Promise utilities. **1.** `make build` succeeds. | Low |

---

## Frontend (dev / build toolchain)

From `package.json` `devDependencies`. These affect the build pipeline and test suite, not the runtime product. Grouped by function to reduce noise.

> **Shared smoke test for all build toolchain libraries:** **1.** `make build` succeeds (webpack compiles without errors). **2.** The built UI loads in the browser and renders correctly. **3.** `yarn test` passes. **4.** `yarn lint` passes.

| Library group | Versions | Smoke tests | Risk |
|---------------|----------|-------------|------|
| `@babel/*` (core, CLI, presets, plugins -- 21 packages) | 7.8.3 - 7.21.5 | Transpilation pipeline. **1.** `make build` succeeds. **2.** UI loads in all supported browsers. **3.** `yarn test` passes (Jest uses Babel). | High |
| `babel-core`, `babel-loader`, `babel-jest`, `babel-eslint`, `babel-plugin-dynamic-import-node` | various | Babel integration with webpack/Jest/ESLint. Same as `@babel/*` above. | High |
| `webpack`, `webpack-cli`, `webpack-notifier` | 5.105.0 / 5.0.1 / 1.12.0 | Module bundler. **1.** `make build` succeeds. **2.** `make generate-dev` (watch mode) works. **3.** Built bundle loads in browser. | High |
| `typescript` | 6.0.2 | TypeScript compiler. **1.** `make build` succeeds. **2.** No new type errors. | High |
| `ts-loader`, `fork-ts-checker-webpack-plugin` | 6.2.2 / 9.1.0 | TypeScript webpack integration. Same as `typescript`. | Medium |
| `esbuild-loader` | 4.4.2 | Fast JS/TS minification in webpack. **1.** `make build` succeeds. **2.** Bundle size is reasonable. | Medium |
| `css-loader`, `sass-loader`, `postcss-loader`, `mini-css-extract-plugin` | various | CSS/SCSS processing pipeline. **1.** `make build` succeeds. **2.** UI styles render correctly. | Medium |
| `sass` (also in runtime), `bourbon`, `node-bourbon`, `node-sass-glob-importer`, `autoprefixer` | various | SCSS tooling. Same as CSS pipeline above. | Medium |
| `html-webpack-plugin` | 5.5.0 | HTML template generation. **1.** `make build` produces `index.html`. | Low |
| `json-loader` | 0.5.7 | JSON file loading. **1.** `make build` succeeds. | Low |
| `eslint`, `eslint-config-*`, `eslint-plugin-*`, `@typescript-eslint/*` (12 packages) | various | Linting pipeline. **1.** `yarn lint` passes. **2.** No new lint errors. | Low |
| `prettier`, `eslint-config-prettier`, `eslint-plugin-prettier` | 2.2.1 / various | Code formatting. **1.** `yarn prettier:check` passes. | Low |
| `jest`, `babel-jest`, `jest-environment-jsdom`, `jest-fixed-jsdom`, `jsdom` | 29.2.0 / various | Test runner. **1.** `yarn test` passes. **2.** Test coverage report generates. | Medium |
| `@testing-library/jest-dom`, `@testing-library/react`, `@testing-library/user-event` | various | React testing utilities. **1.** `yarn test` passes. | Medium |
| `msw` | 2.5.1 | API mocking for tests. **1.** `yarn test` passes (tests that mock API calls work). | Low |
| `expect`, `identity-obj-proxy`, `ignore-styles`, `regenerator-runtime`, `trace-unhandled` | various | Test utilities and polyfills. **1.** `yarn test` passes. | Low |
| `@storybook/*`, `storybook`, `react-docgen-typescript-plugin` (11 packages) | 8.0.4 - 8.6.17 | Component documentation. **1.** `yarn storybook` starts. **2.** Stories render correctly. | Low |
| `classnames` | 2.2.5 | CSS class name composition (used in runtime components via devDeps). **1.** UI renders correctly. | Low |
| `compare-versions` | 6.1.1 | Version string comparison. **1.** `make build` succeeds. | Low |
| `@tsconfig/recommended` | 1.0.13 | TypeScript config base. **1.** `make build` succeeds. | Low |

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
