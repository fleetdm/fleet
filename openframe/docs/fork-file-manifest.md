# Fork File Manifest — what diverges from upstream Fleet

A path-level inventory of everything this fork **creates, modifies, and deletes**
relative to upstream [`fleetdm/fleet`](https://github.com/fleetdm/fleet). Use it as
the index for "what did the fork touch"; the per-feature docs in this directory
explain *why*.

## How this was computed (and its limits)

The fork does **not** clean-merge upstream — it periodically squash-rebases
upstream code into its own commits (e.g. `CI to build Fleet` re-imported entire
upstream trees in one commit). So there is no single upstream commit to diff
against, and a raw `git diff upstream/main HEAD` is dominated by the ~8,200
upstream commits the fork has not pulled.

This manifest is therefore built by:

1. Taking the files touched by the **genuine fork commits** (`upstream/main..HEAD`,
   non-merge), **excluding** the squashed re-sync commit `01a6bb8a63` (4,837 files
   of pure upstream churn).
2. Classifying each path by whether it exists in the fork tree vs. the upstream
   universe (current `upstream/main` ∪ the fork's base-era tags `v4.81.2` /
   `v4.80.0`).

**Baseline:** the fork tracks Fleet **~v4.81.2** (`charts/fleet/Chart.yaml`
`appVersion`). **As of** fork commit `cb06c5d367`.

> Caveat: a handful of files dated between upstream releases (e.g. two
> `migrations/tables/2025122*_*.go`) were added by upstream, inherited by the
> fork, and later renamed upstream — they are counted under *modified migrations*,
> not fork creations.

## Summary

| Class | Files | Where |
|-------|------:|-------|
| **Created** | ~30 | `openframe/`, `server/service/openframe/`, `migrations/openframe/`, `redis/keyprefix.go`, fork CI, 4 chart templates |
| **Modified** | ~516 | 473 upstream migrations made idempotent + ~43 server/orbit/chart/CI files |
| **Deleted** | ~84 | entirely under `.github/` (upstream CI/CD, issue templates, scripts) |

**No top-level directory was deleted.** The fork keeps all of upstream's
`website/`, `ee/`, `frontend/`, `articles/`, `android/`, etc. Deletions are
confined to `.github/`. New directories are purely additive.

## Directory-level view

| Directory | Status | Notes |
|-----------|--------|-------|
| `openframe/` | **created** | Fork docs (`docs/`) + scripts (`scripts/`). Net-new top-level dir. |
| `server/service/openframe/` | **created** | Agent token-auth pipeline (4 files). |
| `server/datastore/mysql/migrations/openframe/` | **created** | Separate goose migration pipeline (3 files). |
| `.github/steps/` | **created** | macOS/Windows signing composite actions. |
| `server/datastore/mysql/migrations/tables/` + `…/data/` | **modified (bulk)** | 473 upstream migrations rewritten idempotent. |
| `charts/fleet/` | **modified + created files** | Fork-owned chart; 4 new templates (`configmap.yaml`, `secret.yaml`, `vulnprocessing/pvc.yaml`, `vulnprocessing/bind-job.yaml`), 8 modified; vuln-processing templates live under `templates/vulnprocessing/`. |
| `server/fleet/`, `server/datastore/`, `server/service/`, `server/mock/`, `cmd/fleet/` | **modified** | Host-assignment, Redis-prefix, TTL-cleanup, osquery-id features. |
| `orbit/cmd/orbit/`, `orbit/pkg/osquery/` | **modified** | Agent OpenFrame mode. |
| `.github/workflows/`, `.github/ISSUE_TEMPLATE/`, `.github/scripts/`, `.github/actions/` | **mostly deleted** | Upstream CI/community automation stripped; replaced by the fork's lean pipeline. |

## Created

### New directories / fork-only code

```
openframe/                                         # net-new top-level dir
├── docs/                                           # this documentation set
└── scripts/
    └── test_host_assignments.sh

server/service/openframe/                           # agent token-auth pipeline
├── openframe-encryption-service.go
├── openframe-token-extractor.go
├── openframe_authorization_manager.go
└── openframe_token_refresher.go

server/datastore/mysql/migrations/openframe/        # separate goose client
├── migration.go
├── 20260301000001_AddPolicyHostsJoinTable.go
├── 20260301000002_AddQueryHostsJoinTable.go
├── 20260818000001_AddPoliciesOpenframeManagedColumn.go   # policies.openframe_managed
├── 20260818000002_AddQueriesOpenframeManagedColumn.go    # queries.openframe_managed
└── 20260831000001_SeedGlobalAppConfigRow.go              # instance app_config row (id=1)

server/datastore/redis/keyprefix.go                 # per-tenant Redis prefix
server/fleet/openframe.go                           # IsOpenframeMode() gate
```

### New CI / packaging files

```
.github/steps/sign-macos-package/action.yml
.github/steps/sign-windows-package/action.yml
.github/workflows/release.yml
.github/workflows/test.yml
.github/workflows/changes.yaml
.github/workflows/sync-upstream.yml
charts/fleet/templates/configmap.yaml
charts/fleet/templates/secret.yaml
charts/fleet/templates/vulnprocessing/bind-job.yaml
charts/fleet/templates/vulnprocessing/pvc.yaml
```

### Documentation (`openframe/docs/`)

All fork documentation lives here (the `docs/node-key-management.md` that upstream
never had was relocated into this directory):

```
README.md                       agent-openframe-mode.md
architecture-host-assignments.md  api-host-assignments.md
api-expose-osquery-host-id.md   query-results-ttl-cleanup.md
api-osquery-schema-search.md
redis-key-prefix.md             migrations.md
helm-chart.md                   ci-cd-release-pipeline.md
node-key-management.md          local-setup.md
test_host_assignments.md        fork-file-manifest.md   (this file)
```

## Modified

### Upstream migrations made idempotent (~473 files)

Every file under `server/datastore/mysql/migrations/tables/` and
`server/datastore/mysql/migrations/data/` that contained `CREATE TABLE`,
`INSERT`, or `DROP TABLE` was rewritten with `IF NOT EXISTS` / `INSERT IGNORE` /
`IF EXISTS` and tagged `// Idempotent migration.`. See
[migrations.md](migrations.md). This is the largest single category by file count
and the heaviest standing rebase cost.

### Server / agent feature code (~43 files)

| Area | Files |
|------|-------|
| Host assignments | `server/fleet/{policies,queries,hosts,datastore,service}.go`, `server/datastore/mysql/{policies,queries,hosts}.go`, `server/service/{global_policies,queries,handler,labels_util}.go`, `server/mock/{datastore,datastore_mock}.go`, `server/mock/service/service_mock.go`, `server/datastore/mysql/mysql.go`, `cmd/fleet/prepare.go` |
| osquery schema search and refresh | `schema/osquery_{search,refresh}.go`, `cmd/fleet/osquery_schema_refresh_openframe.go`, `cmd/fleet/serve.go`, `server/config/config.go`, `server/service/{osquery_schema_openframe,handler}.go`, `server/api_endpoints/api_endpoints.yml`, `charts/fleet/values.yaml` |
| Managed queries / policies | flag: `server/fleet/{queries,policies}.go`, `server/service/{queries,global_policies,team_policies}.go`, `server/datastore/mysql/{queries,policies}.go`, `schema.sql`; queries listing opt-in (`include_openframe_managed`): `server/fleet/{app,api_queries,service}.go`, `server/service/{global_schedule,team_schedule,queries_test}.go`, `server/mock/service/service_mock.go`; tests: `server/datastore/mysql/{queries,policies}_openframe_managed_test.go` — see [managed-queries.md](managed-queries.md), [managed-policies.md](managed-policies.md) |
| Shared-mode superuser | `server/fleet/openframe.go` (allowlist), `server/service/openframe_middleware.go`, `server/service/endpoint_campaigns.go`, `cmd/fleet/serve.go`, `charts/fleet/{values.yaml,templates/deployment.yaml}`; tests: `server/fleet/openframe_test.go`, `server/service/openframe_middleware_test.go` — see [mysql-multitenancy-feature.md](mysql-multitenancy-feature.md#superuser-shared-mode-only) |
| osquery host id | `server/fleet/hosts.go` |
| Query-results TTL cleanup | `server/config/config.go`, `server/fleet/{cron_schedules,datastore}.go`, `server/datastore/mysql/query_results.go`, `cmd/fleet/{cron,serve}.go` |
| Redis key prefix | `server/datastore/redis/redis.go`, `server/config/config.go`, `cmd/fleet/serve.go` |
| Agent OpenFrame mode | `orbit/cmd/orbit/orbit.go`, `orbit/pkg/osquery/osquery.go`, `server/service/orbit_client.go`, `server/service/base_client.go` |
| Agent options kept | `cmd/fleet/serve.go` (starter-library skip under multitenancy), `server/fleet/openframe.go` (trimmed seed/fallback defaults), `server/datastore/mysql/teams_openframe_test.go`, `server/fleet/openframe_test.go` |
| Agent JSON content-type | `client/orbit_client.go`, `client/device_client.go`, `orbit/cmd/fetch_cert/main.go`, `client/orbit_client_content_type_test.go` |
| Agent skip setup experience | `orbit/cmd/orbit/orbit.go`, `orbit/pkg/setup_experience/setup_experience.go`, `orbit/cmd/orbit/setup_experience_openframe_test.go`, `orbit/pkg/setup_experience/setup_experience_openframe_test.go` |
| Build / meta | `go.mod`, `go.sum`, `.gitignore`, `README.md`, `.github/pull_request_template.md`, `server/archtest/*` |

### Helm chart (~9 files)

`charts/fleet/values.yaml`, `Chart.yaml`, and templates
`deployment.yaml`, `job-migration.yaml`, `vulnprocessing/cronjob.yaml`,
`_helpers.tpl`, `rbac.yaml`, `sa.yaml` (+ `charts/example-tuf-skaffold.yaml`).
See [helm-chart.md](helm-chart.md).

## Deleted (~84 files, all under `.github/`)

The fork removed upstream Fleet's heavy CI/CD and community automation, which it
does not run, and replaced it with the lean pipeline in
[ci-cd-release-pipeline.md](ci-cd-release-pipeline.md).

| Group | Count | Examples |
|-------|------:|----------|
| `.github/workflows/` | 65 | `build-binaries.yaml`, `build-orbit.yaml`, `goreleaser-fleet.yaml`, `release-fleetd-base.yml`, `deploy-fleet-website.yml`, `dogfood-*.yml`, `fleetd-tuf.yml`, `test-go.yaml`, `test-js.yml`, `scorecards-analysis.yml`, `trivy-scan.yml`, `codeql-analysis.yml`, … |
| `.github/ISSUE_TEMPLATE/` | 11 | all upstream issue/story/timebox templates |
| `.github/scripts/` | 6 | dogfood / infra helper scripts |
| `.github/actions/` | 1 | `r2-upload` |
| `.github/dependabot.yml` | 1 | upstream dependabot config |

> Some of these names (e.g. `code-sign-windows.yml`, `goreleaser-*`) overlap
> conceptually with the fork's own pipeline, but the fork's signing/release logic
> lives in the **new** files listed under *Created*, not these upstream ones.

## Reproducing this manifest

```bash
# genuine fork commits (exclude the squashed re-sync)
git log --no-merges --format=%H upstream/main..HEAD | grep -v 01a6bb8a63

# classify a path: present in fork but not in any upstream baseline => created
git cat-file -e HEAD:<path>          && echo in-fork
git cat-file -e upstream/main:<path> || git cat-file -e v4.81.2:<path> || echo upstream-only-absent
```

---

## Appendix: full file lists

Computed from the fork working tree vs the upstream baseline
(`upstream/main` ∪ `v4.81.2` ∪ `v4.80.0`). **Excludes**
`server/datastore/mysql/migrations/tables/` and
`server/datastore/mysql/migrations/data/` (the ~473 idempotent upstream migrations —
see [migrations.md](migrations.md)). Paths are repo-root-relative.

### Added (49)

```
.github/steps/sign-macos-package/action.yml
.github/steps/sign-windows-package/action.yml
.github/workflows/changes.yaml
.github/workflows/release.yml
.github/workflows/sync-upstream.yml
.github/workflows/test.yml
CLAUDE.md
charts/fleet/templates/configmap.yaml
charts/fleet/templates/secret.yaml
charts/fleet/templates/vulnprocessing/bind-job.yaml
charts/fleet/templates/vulnprocessing/pvc.yaml
cmd/fleet/osquery_schema_refresh_openframe.go
cmd/fleet/osquery_schema_refresh_openframe_test.go
openframe/docs/README.md
openframe/docs/agent-openframe-mode.md
openframe/docs/api-expose-osquery-host-id.md
openframe/docs/api-host-assignments.md
openframe/docs/api-osquery-schema-search.md
openframe/docs/architecture-host-assignments.md
openframe/docs/ci-cd-release-pipeline.md
openframe/docs/fork-file-manifest.md
openframe/docs/helm-chart.md
openframe/docs/local-setup.md
openframe/docs/migrations.md
openframe/docs/node-key-management.md
openframe/docs/query-results-ttl-cleanup.md
openframe/docs/redis-key-prefix.md
openframe/docs/test_host_assignments.md
openframe/docs/upstream-sync-conflict-resolution.md
openframe/scripts/test_host_assignments.sh
openframe/scripts/verify.sh
server/datastore/mysql/migrations/openframe/20260301000001_AddPolicyHostsJoinTable.go
server/datastore/mysql/migrations/openframe/20260301000002_AddQueryHostsJoinTable.go
server/datastore/mysql/migrations/openframe/20260722000001_AddTeamIdToCdcTables.go
server/datastore/mysql/migrations/openframe/migration.go
server/datastore/mysql/migrations_openframe_test.go
server/datastore/redis/keyprefix.go
server/datastore/redis/keyprefix_test.go
server/fleet/openframe.go
schema/osquery_search.go
schema/osquery_search_test.go
schema/osquery_refresh.go
schema/osquery_refresh_test.go
server/service/osquery_schema_openframe.go
server/service/osquery_schema_openframe_test.go
server/service/openframe/openframe-encryption-service.go
server/service/openframe/openframe-token-extractor.go
server/service/openframe/openframe_authorization_manager.go
server/service/openframe/openframe_token_refresher.go
```

### Modified (52)

```
.github/pull_request_template.md
.gitignore
Makefile
README.md
charts/example-tuf-skaffold.yaml
charts/fleet/Chart.yaml
charts/fleet/templates/_helpers.tpl
charts/fleet/templates/vulnprocessing/cronjob.yaml
charts/fleet/templates/deployment.yaml
charts/fleet/templates/job-migration.yaml
charts/fleet/templates/rbac.yaml
charts/fleet/templates/sa.yaml
charts/fleet/values.yaml
cmd/fleet/cron.go
cmd/fleet/prepare.go
cmd/fleet/serve.go
cmd/osquery-perf/agent.go
go.mod
go.sum
orbit/cmd/orbit/orbit.go
orbit/pkg/osquery/osquery.go
server/activity/internal/mysql/new_activity.go
server/archtest/README.md
server/archtest/test_files/dependency/dependency.go
server/config/config.go
server/datastore/mysql/hosts.go
server/datastore/mysql/mysql.go
server/datastore/mysql/policies.go
server/datastore/mysql/queries.go
server/datastore/mysql/query_results.go
server/datastore/redis/redis.go
server/fleet/api_queries.go
server/fleet/app.go
server/fleet/cron_schedules.go
server/fleet/datastore.go
server/fleet/hosts.go
server/fleet/policies.go
server/fleet/queries.go
server/fleet/service.go
server/mock/datastore.go
server/mock/datastore_mock.go
server/mock/service/service_mock.go
server/service/base_client.go
server/service/global_policies.go
server/service/global_schedule.go
server/service/handler.go
server/service/labels_util.go
server/service/orbit_client.go
server/service/osquery_utils/queries.go
server/service/queries.go
server/service/team_schedule.go
server/vulnerabilities/nvd/cpe.go
```
