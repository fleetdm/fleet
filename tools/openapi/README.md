# OpenAPI spec generator

Generates Fleet's [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0) spec
from the canonical REST API reference at
[`docs/REST API/rest-api.md`](../../docs/REST%20API/rest-api.md). The Markdown
is the source of truth; `openapi.yml` is a generated artifact, not committed
to the repo. Story: [#45279](https://github.com/fleetdm/fleet/issues/45279).

> **Beta.** The generated spec is a best-effort artifact produced
> automatically from the API reference. It is not manually validated
> against the API: it's only as accurate as the reference docs, and
> generator errors are possible. The Markdown reference is the official
> API documentation. If the spec disagrees with actual API behavior,
> [file an issue](https://github.com/fleetdm/fleet/issues).

## Usage

Generate the spec locally after editing the REST API docs:

```sh
make openapi
```

This writes `tools/openapi/openapi.yml`, which is gitignored. Every PR that
touches the docs or this tool gets CI generation and validation, and the
result is uploaded as a workflow artifact named `openapi-spec` (see the
"Actions" tab on the PR). There's no committed file to keep in sync.
Attaching the spec to releases lands in a follow-up PR (see DESIGN.md).

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

## Design

See [DESIGN.md](./DESIGN.md).
