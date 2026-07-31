# OpenAPI spec generator

Generates Fleet's pilot [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0)
spec from the canonical REST API reference at
[`docs/REST API/rest-api.md`](../../docs/REST%20API/rest-api.md). The Markdown
is the source of truth; `openapi.yml` is a generated artifact, not committed
to the repo. Pilot story: [#45279](https://github.com/fleetdm/fleet/issues/45279).

## Usage

Generate the spec locally after editing docs for a pilot endpoint:

```sh
make openapi
```

This writes `tools/openapi/openapi.yml`, which is gitignored. Every PR that
touches the docs or this tool gets CI generation and validation, and the
result is uploaded as a workflow artifact named `openapi-spec` (see the
"Actions" tab on the PR). Releases generate and attach `openapi.yml`
automatically, no committed file to keep in sync.

Pilot endpoints live in [`allowlist.yml`](./allowlist.yml). Expanding coverage
is config-only: add a method and path that exist in the Markdown, run
`make openapi` to confirm it generates and validates, and commit the
allowlist change. The generate command prints a coverage report showing how
much of the full reference currently parses.

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
