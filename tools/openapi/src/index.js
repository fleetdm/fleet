#!/usr/bin/env node
// PoC OpenAPI generator for Fleet's REST API.
//
// Reads docs/REST API/rest-api.md, extracts the endpoints listed in the
// allowlist (src/endpoints.js), and emits an OpenAPI 3.1 YAML document.
// Validates the output against the OpenAPI 3.1 schema before writing.
//
// Usage:
//   node src/index.js                                   # write build/openapi.yml
//   node src/index.js --out path/to/openapi.yml
//   node src/index.js --stdout                          # write YAML to stdout
//   node src/index.js --markdown path/to/rest-api.md
//
// Exit codes:
//   0 — success
//   1 — generic error (parse failure, IO error, unknown flag, etc.)
//   2 — validation failed (output is not valid OpenAPI 3.1)
'use strict';

const fs = require('fs');
const path = require('path');
const yaml = require('yaml');

const { endpoints } = require('./endpoints');
const { parseEndpoint } = require('./markdown');
const { buildDocument } = require('./openapi');
const { validate } = require('./validate');

const REPO_ROOT = path.resolve(__dirname, '../../..');
const DEFAULT_MARKDOWN = path.join(REPO_ROOT, 'docs', 'REST API', 'rest-api.md');
const DEFAULT_OUT = path.join(REPO_ROOT, 'build', 'openapi.yml');

function parseArgs(argv) {
  const args = { out: DEFAULT_OUT, stdout: false, markdown: DEFAULT_MARKDOWN };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    switch (a) {
      case '--stdout':
        args.stdout = true;
        break;
      case '--out':
        args.out = argv[++i];
        if (!args.out) throw new Error('--out requires a path');
        break;
      case '--markdown':
        args.markdown = argv[++i];
        if (!args.markdown) throw new Error('--markdown requires a path');
        break;
      case '-h':
      case '--help':
        printUsage();
        process.exit(0);
        break;
      default:
        throw new Error(`unknown argument: ${a}`);
    }
  }
  return args;
}

function printUsage() {
  process.stdout.write(
    [
      'fleet-openapi — PoC generator: REST API Markdown → OpenAPI 3.1 YAML',
      '',
      'Usage:',
      '  node src/index.js [--out <path>] [--markdown <path>] [--stdout]',
      '',
      'Flags:',
      '  --out <path>        Output file path (default: <repo>/build/openapi.yml).',
      '  --stdout            Write YAML to stdout instead of a file.',
      '  --markdown <path>   Source Markdown file (default: docs/REST API/rest-api.md).',
      '  -h, --help          Show this message.',
      '',
    ].join('\n'),
  );
}

async function main() {
  let args;
  try {
    args = parseArgs(process.argv.slice(2));
  } catch (err) {
    process.stderr.write(`error: ${err.message}\n`);
    printUsage();
    process.exit(1);
  }

  let md;
  try {
    md = fs.readFileSync(args.markdown, 'utf8');
  } catch (err) {
    process.stderr.write(
      `error: could not read Markdown source at ${args.markdown}: ${err.message}\n`,
    );
    process.exit(1);
  }

  // Parse each allowlisted endpoint, keeping the spec entry alongside.
  const endpointResults = [];
  for (const spec of endpoints) {
    try {
      const parsed = parseEndpoint(md, spec.markdownHeading);
      endpointResults.push({ spec, parsed });
    } catch (err) {
      process.stderr.write(
        `error: failed to parse endpoint ${JSON.stringify(spec.markdownHeading)}: ${err.message}\n`,
      );
      process.exit(1);
    }
  }

  // Resolve API version. For now we use a static placeholder; the full story
  // will wire this to the Fleet release version (e.g. via a tag or a flag).
  const doc = buildDocument(endpointResults, { version: '0.0.1-poc' });

  try {
    await validate(doc);
  } catch (err) {
    process.stderr.write(`error: generated document failed OpenAPI 3.1 validation\n`);
    process.stderr.write(`${err.stack || err.message || String(err)}\n`);
    process.exit(2);
  }

  const yamlOut = yaml.stringify(doc, {
    // Quote strings that look like YAML reserved words / numbers so the spec
    // round-trips losslessly through tooling.
    defaultStringType: 'PLAIN',
    defaultKeyType: 'PLAIN',
    lineWidth: 0, // don't wrap long description lines
  });

  if (args.stdout) {
    process.stdout.write(yamlOut);
    return;
  }

  fs.mkdirSync(path.dirname(args.out), { recursive: true });
  fs.writeFileSync(args.out, yamlOut, 'utf8');
  process.stdout.write(
    `wrote ${endpointResults.length} endpoint(s) to ${args.out}\n`,
  );
}

main().catch((err) => {
  process.stderr.write(`unexpected error: ${err.stack || err.message || err}\n`);
  process.exit(1);
});
