# OpenAPI generator (PoC)

A Node.js tool that generates an [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0) document
from Fleet's canonical REST API Markdown reference at
[`docs/REST API/rest-api.md`](../../docs/REST%20API/rest-api.md).

This is the **proof-of-concept** for
[fleetdm/fleet#45279](https://github.com/fleetdm/fleet/issues/45279). The
Markdown remains the canonical source of truth; the OpenAPI spec is a derived,
versioned artifact.

## PoC scope

- All 37 Hosts endpoints from the Markdown reference.
- Markdown parser, endpoint allowlist, OpenAPI 3.1 emitter, validation step.
- Golden-file test (`npm test`) that asserts the generated spec matches a
  committed snapshot (`spec.yml`).
- Output written to `build/openapi.yml` (gitignored) or to stdout.

## What is intentionally NOT in the PoC

- CI hook that runs the generator on every PR — deferred to the full story.
- Release-workflow attachment of `openapi.yml` as a downloadable artifact —
  deferred. The generator is invocable as a single command (`npm run generate`
  or `node src/index.js`) so wiring it into a release job will be mechanical.
- A hosted Swagger UI — Fleet explicitly does not host one. See
  [Why this way / not continuously generated reference docs](https://fleetdm.com/handbook/company/why-this-way#why-not-continuously-generate-rest-api-reference-docs-from-javadoc-style-code-comments).

## Running it locally

From this directory:

```sh
cd tools/openapi
npm install
npm run generate
```

This writes `<repo-root>/build/openapi.yml`. To emit to stdout instead:

```sh
npm run generate:stdout > /tmp/openapi.yml
```

Or directly:

```sh
node src/index.js --out /tmp/openapi.yml
```

The generator validates the produced document against the OpenAPI 3.1 schema
before writing. If validation fails, it exits non-zero (exit code `2`) and
prints the validator's error.

## Testing

A golden-file test compares the generator's output against the committed
`spec.yml` snapshot:

```sh
npm test
```

If you intentionally change the generator or the source Markdown in a way that
affects the output, update the snapshot:

```sh
npm run test:update
```

Then commit the updated `spec.yml` alongside your changes.

## Viewing the spec

For a UI view of the generated spec, run:

```sh
cd tools/openapi
npm install
npm run view
```

This regenerates the spec to `tools/openapi/spec.yml`, starts a local HTTP
server, and opens [Redoc](https://redocly.com/redoc) in your browser pointed
at the spec. The Redoc bundle is loaded from a CDN; nothing is hosted by Fleet.

## Library choices

| Concern | Library | Why |
|---|---|---|
| OpenAPI 3.1 validation | [`@readme/openapi-parser`](https://www.npmjs.com/package/@readme/openapi-parser) | Actively-maintained fork of `@apidevtools/swagger-parser` with explicit OpenAPI 3.1 support. |
| YAML emission | [`yaml`](https://www.npmjs.com/package/yaml) (Eemeli Aro) | Canonical Node YAML library; supports the comment-preserving and line-width controls we want. |

The runtime has **no other production dependencies**.

## How the pieces fit together

```
docs/REST API/rest-api.md          (canonical, hand-written)
            │
            ▼
src/markdown.js                    (parse a section into structured data)
            │
            ▼
src/endpoints.js  ──▶  src/openapi.js  (assemble OpenAPI document)
            │
            ▼
src/schema.js                      (infer JSON Schema from example payloads)
            │
            ▼
src/validate.js                    (OpenAPI 3.1 schema validation)
            │
            ▼
build/openapi.yml                  (output artifact, gitignored)
```

The parser is intentionally narrow and tied to the conventions used in
Fleet's Markdown reference: a section heading, a backticked request line, a
`#### Parameters` table, and a `##### Default response` block with a fenced
`json` example. This is documented in `src/markdown.js`.

## Adding another endpoint

Adding a new endpoint is a data-only change. Append an entry to the `endpoints`
array in [`src/endpoints.js`](./src/endpoints.js):

```js
{
  markdownHeading: '### Get host',
  operationId: 'getHost',
  tag: 'Hosts',
  summary: 'Get host',
  pathParameters: [
    { name: 'id', type: 'integer', description: 'Host ID.' },
  ],
},
```

Then re-run `npm run generate`. If the corresponding Markdown section
violates the parser's conventions, the generator will fail with an explicit
error pointing to the heading — fix the parser (or, more rarely, the
Markdown) rather than special-casing in the manifest.

After adding endpoints, update the golden file: `npm run test:update`.

## Out of scope by design

This PoC deliberately does NOT:

- Modify any documentation files.
- Introduce CI jobs or GitHub Actions.
- Add a hosted Swagger UI.
- Replace any existing API tooling.
