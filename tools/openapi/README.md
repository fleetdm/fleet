# OpenAPI spec generator

Generates Fleet's pilot [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0)
spec from the canonical REST API reference at
[`docs/REST API/rest-api.md`](../../docs/REST%20API/rest-api.md). The Markdown
is the source of truth; `openapi.yml` is a derived artifact attached to each
Fleet release. Pilot story: [#45279](https://github.com/fleetdm/fleet/issues/45279).

## Usage

Regenerate the committed spec after editing docs for a pilot endpoint:

```sh
make openapi
```

CI fails if `openapi.yml` is stale (`make openapi-check`). Commit the
regenerated file together with your docs change.

Pilot endpoints live in [`allowlist.yml`](./allowlist.yml). Expanding coverage
is config-only: add a method and path that exist in the Markdown, run
`make openapi`, and commit both files. The generate command prints a coverage
report showing how much of the full reference currently parses.

## Verifying against a live server

```sh
cd tools/openapi
go run . verify --server https://localhost:8080 --email admin@example.com --password '...'
```

Verify calls each pilot endpoint and validates the response against the spec.
It seeds data it can create over the API (a policy, a report, a fleet) using
`openapi-verify-` prefixed names, and does not clean them up. Use a disposable
dev instance. Endpoints that need real hosts (`GET /hosts/{id}`) or an
MDM-enrolled host (`POST /commands/run`) report as partially verified when the
server has none; pass `--strict` to fail on partials for release sign-off.

## Design

See [DESIGN.md](./DESIGN.md).
