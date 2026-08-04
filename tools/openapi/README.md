# OpenAPI spec generator

Generates Fleet's [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0) spec
from the canonical REST API reference at
[`docs/REST API/rest-api.md`](../../docs/REST%20API/rest-api.md). The Markdown
is the source of truth; `openapi.yml` is a generated artifact, not committed
to the repo. Story: [#45279](https://github.com/fleetdm/fleet/issues/45279).

## Usage

Generate the spec locally after editing the REST API docs:

```sh
make openapi
```

This writes `tools/openapi/openapi.yml`, which is gitignored. Every PR that
touches the docs or this tool gets CI generation and validation, and the
result is uploaded as a workflow artifact named `openapi-spec` (see the
"Actions" tab on the PR). Releases generate and attach `openapi.yml`
automatically, no committed file to keep in sync.

The generator covers every documented endpoint: every `### ` section in
`rest-api.md` that has a request line becomes an operation in the spec.
There's no allowlist to update. Expanding coverage means fixing a docs
section so it parses, not adding config.

Any endpoint section that fails to parse (bad JSON in an example, a
malformed parameters table, a missing response block, and so on) fails the
build: `generate` exits 1 and lists every offending heading and reason.
Sections with no request line at all (for example "Retrieve your API
token", which documents UI steps, not an endpoint) are not endpoints and are
tolerated; they still show up in the coverage report printed to stderr.

## Verifying against a live server

```sh
cd tools/openapi
go run . verify --server https://localhost:8080 --email admin@example.com --password '...'
```

Verify is a hand-built contract test covering a fixed set of 10 commonly
integrated endpoints (see DESIGN.md), not every endpoint in the spec. It calls each
one and validates the response against the spec.
It seeds data it can create over the API (a policy, a report, a fleet) using
`openapi-verify-` prefixed names, and does not clean them up. Use a disposable
dev instance. Endpoints that need real hosts (`GET /hosts/{id}`) or an
MDM-enrolled host (`POST /commands/run`) report as partially verified when the
server has none; pass `--strict` to fail on partials for release sign-off.

## Design

See [DESIGN.md](./DESIGN.md).
