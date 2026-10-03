# Local Setup

## Prerequisites

- Go toolchain
- Node.js (^24.10.0) + yarn
- `kubectl` with access to the tenant cluster
- `jq` for test scripts

## 1. Bring up MySQL and Redis

The repo ships both, which is the quickest path and keeps you off tenant data:

```bash
docker compose up -d mysql redis   # mysql on 127.0.0.1:3306 (fleet/insecure, db "fleet"), redis on 6379
```

> A tenant cluster is **not** a substitute. Under shared multi-tenancy Fleet talks to Cloud SQL over
> private service access and to a managed Redis, both over TLS — neither is a pod, so the
> `kubectl port-forward fleetmdm-mysql-0` / `fleetmdm-redis-0` that older setups used no longer
> applies. Reaching them needs a relay pod or the Cloud SQL Auth Proxy.

## 2. Build Fleet

Generate frontend assets and compile the binary with embedded assets:

```bash
yarn install --ignore-engines
yarn run --ignore-engines webpack --progress
make generate-go
go build -tags full -o build/fleet ./cmd/fleet
```

## 3. Run database migrations

```bash
./build/fleet prepare db --no-prompt \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=fleet \
  --mysql_password=insecure
```

## 4. Start Fleet

```bash
FLEET_OPENFRAME_MODE=1 ./build/fleet serve \
  --dev \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=fleet \
  --mysql_password=insecure \
  --redis_address=127.0.0.1:6379 \
  --server_address=0.0.0.0:8080 \
  --server_tls=false
```

Fleet UI will be available at `http://localhost:8080/login`.

> **Note:** `FLEET_OPENFRAME_MODE=1` is required for openframe-specific endpoints (policy/query host assignments).

## 5. Reproducing shared multi-tenancy locally

Deployed tenants run in **shared mode**, and none of its fences exist without the flag — so a bug
that only shows there will not reproduce under the plain `serve` above. Add:

```bash
FLEET_OPENFRAME_MULTI_TENANCY_ENABLED=true \
FLEET_OPENFRAME_TENANT_UUID= \
FLEET_OPENFRAME_SUPERUSER_EMAILS=research@flamingo.cx \
FLEET_OPENFRAME_MODE=1 ./build/fleet serve --dev ...
```

An empty `FLEET_OPENFRAME_TENANT_UUID` is what selects shared mode (a value pins the process to one
tenant instead); startup confirms it with
`OpenFrame multitenancy: shared per-request mode (no process pin)`. Every `/api/**` call then needs
`X-Tenant-Id: <any uuid>` — the team is created on first use — and the quickest sanity check is:

```bash
curl -o /dev/null -w '%{http_code}\n' http://localhost:8080/api/v1/fleet/results/info               # 401
curl -o /dev/null -w '%{http_code}\n' -H "X-Tenant-Id: $UUID" http://localhost:8080/api/v1/fleet/results/info  # 200
```

`FLEET_OPENFRAME_SUPERUSER_EMAILS` names the accounts allowed to run unpinned — see
[mysql-multitenancy-feature.md](./mysql-multitenancy-feature.md#superuser-shared-mode-only).

## Troubleshooting

### `--dev` flag overrides MySQL credentials

`applyDevFlags` in `cmd/fleet/main.go` sets default MySQL username/password/database. If your credentials differ from the defaults, pass them explicitly via flags or env vars (`FLEET_MYSQL_USERNAME`, etc.) — the patched version only applies defaults when values are empty.

### Cookies are shared across ports

Cookies ignore the port, so a session from another Fleet on `localhost` (a `kubectl port-forward` of
a deployed one, say) is sent to the local server too. It will not validate against a different
database, but it does make "logged in / logged out" confusing. Use a different host (`127.0.0.1`
vs `localhost`) or a private window to keep them apart.

### gcloud auth expired

If port-forward fails with `Reauthentication failed`, run:

```bash
gcloud auth login
```

### S3 bucket errors

The `failed to create test software installer bucket` / `failed to create test carve bucket` warnings are harmless in local dev — they require a local MinIO instance on port 9000.
