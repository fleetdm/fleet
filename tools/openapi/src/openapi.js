// Assemble the OpenAPI 3.1 document from the parsed Markdown sections plus
// each endpoint's manifest entry.
//
// Layering: this module knows nothing about Markdown. It receives already-
// parsed endpoint structures and produces the OpenAPI JS object that gets
// serialized to YAML by index.js.
'use strict';

const { inferSchema } = require('./schema');

// JSON Schema "type" → OpenAPI 3.1 parameter schema. The Markdown parameter
// table uses Fleet's own type labels (mostly the obvious set, plus `array`).
const TYPE_MAP = {
  string: 'string',
  integer: 'integer',
  number: 'number',
  boolean: 'boolean',
  array: 'array',
  object: 'object',
};

/**
 * Map a Markdown table type cell to an OpenAPI parameter schema.
 *
 * Example inputs we expect: "integer", "string", "boolean", "array (string)".
 * Unknown values fall back to `{ type: 'string' }` so we never emit a hard
 * failure for a typo in the docs — but we log a warning so it's visible.
 */
function mapParameterSchema(rawType) {
  if (!rawType) return { type: 'string' };
  const t = rawType.trim().toLowerCase();
  if (TYPE_MAP[t]) return { type: TYPE_MAP[t] };

  // "array (string)" → array of strings
  const arrayMatch = t.match(/^array\s*\(\s*(\w+)\s*\)$/);
  if (arrayMatch && TYPE_MAP[arrayMatch[1]]) {
    return { type: 'array', items: { type: TYPE_MAP[arrayMatch[1]] } };
  }
  process.stderr.write(
    `warning: unknown parameter type ${JSON.stringify(rawType)}; defaulting to string\n`,
  );
  return { type: 'string' };
}

/**
 * Build the `parameters` array for an OpenAPI operation, combining params
 * parsed from the Markdown table with any path parameters declared in the
 * endpoint manifest entry.
 */
function buildParameters(parsedParams, endpointSpec) {
  /** @type {any[]} */
  const out = [];

  for (const pp of endpointSpec.pathParameters || []) {
    out.push({
      name: pp.name,
      in: 'path',
      required: true,
      description: pp.description,
      schema: mapParameterSchema(pp.type),
    });
  }

  for (const p of parsedParams) {
    // OpenAPI requires `in` to be one of: path | query | header | cookie.
    // Fleet's table uses these values directly. Reject anything weird.
    const where = p.in.toLowerCase();
    if (!['path', 'query', 'header', 'cookie'].includes(where)) {
      process.stderr.write(
        `warning: skipping parameter ${JSON.stringify(p.name)} with unsupported "in": ${JSON.stringify(p.in)}\n`,
      );
      continue;
    }
    out.push({
      name: p.name,
      in: where,
      required: where === 'path',
      description: collapseWhitespace(p.description),
      schema: mapParameterSchema(p.type),
    });
  }
  return out;
}

function collapseWhitespace(s) {
  return s.replace(/\s+/g, ' ').trim();
}

/**
 * Build a single Path Item Object for one endpoint.
 *
 * @returns {{ pathTemplate: string, pathItem: any }}
 */
function buildPathItem(endpointSpec, parsed) {
  // OpenAPI uses {param} not :param. Convert if needed.
  const pathTemplate = (endpointSpec.pathOverride || parsed.path).replace(
    /:([A-Za-z_][A-Za-z0-9_]*)/g,
    '{$1}',
  );

  /** @type {any} */
  const operation = {
    operationId: endpointSpec.operationId,
    tags: [endpointSpec.tag],
    summary: endpointSpec.summary,
  };
  if (endpointSpec.description) {
    operation.description = endpointSpec.description;
  }

  const parameters = buildParameters(parsed.parameters, endpointSpec);
  if (parameters.length > 0) {
    operation.parameters = parameters;
  }

  // Build responses from the example payload.
  if (parsed.exampleResponse) {
    const { status, body } = parsed.exampleResponse;
    operation.responses = {
      [String(status)]: {
        description: 'Successful response.',
        content: {
          'application/json': {
            schema: inferSchema(body),
            example: body,
          },
        },
      },
    };
  } else {
    // No example response in the Markdown — emit a minimal 200 placeholder.
    // This keeps the document valid and signals "shape unknown" to consumers.
    operation.responses = {
      '200': {
        description: 'Successful response.',
      },
    };
  }

  return {
    pathTemplate,
    pathItem: {
      [parsed.method]: operation,
    },
  };
}

/**
 * Assemble the top-level OpenAPI 3.1 document.
 *
 * @param {Array<{ spec: any, parsed: any }>} endpointResults
 * @param {{ version: string, title?: string }} info
 * @returns {any}
 */
function buildDocument(endpointResults, info) {
  /** @type {Record<string, any>} */
  const paths = {};
  for (const { spec, parsed } of endpointResults) {
    const { pathTemplate, pathItem } = buildPathItem(spec, parsed);
    // Merge into paths: multiple methods may target the same path template.
    paths[pathTemplate] = { ...(paths[pathTemplate] || {}), ...pathItem };
  }

  return {
    openapi: '3.1.0',
    info: {
      title: info.title || 'Fleet REST API',
      version: info.version,
      description:
        'Auto-generated from Fleet\'s canonical REST API Markdown reference ' +
        '(`docs/REST API/rest-api.md`). PoC scope: one endpoint. See ' +
        'tools/openapi/README.md.',
      license: {
        name: 'MIT',
        identifier: 'MIT',
      },
    },
    servers: [
      {
        url: 'https://fleet.example.com',
        description: 'Replace with your Fleet server URL.',
      },
    ],
    tags: collectTags(endpointResults),
    paths,
  };
}

function collectTags(endpointResults) {
  const seen = new Set();
  /** @type {Array<{ name: string }>} */
  const tags = [];
  for (const { spec } of endpointResults) {
    if (!seen.has(spec.tag)) {
      seen.add(spec.tag);
      tags.push({ name: spec.tag });
    }
  }
  return tags;
}

module.exports = { buildDocument, buildPathItem, buildParameters };
