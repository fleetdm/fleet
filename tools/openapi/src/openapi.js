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
  // "integer or string" → oneOf
  const orMatch = t.match(/^(\w+)\s+or\s+(\w+)$/);
  if (orMatch) {
    const types = [orMatch[1], orMatch[2]].map((x) => TYPE_MAP[x]).filter(Boolean);
    if (types.length === 2) {
      return { oneOf: types.map((x) => ({ type: x })) };
    }
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
  // OpenAPI requires (in, name) to be unique across an operation's parameters.
  // The manifest's pathParameters and the Markdown table can overlap once
  // path-templated endpoints (e.g. `Get host`) are added — dedupe defensively.
  const seen = new Set();

  for (const pp of endpointSpec.pathParameters || []) {
    const key = `path:${pp.name}`;
    if (seen.has(key)) continue;
    seen.add(key);
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
      // Body params are handled by buildRequestBody — only warn for truly unknown values.
      if (where !== 'body') {
        process.stderr.write(
          `warning: skipping parameter ${JSON.stringify(p.name)} with unsupported "in": ${JSON.stringify(p.in)}\n`,
        );
      }
      continue;
    }
    const key = `${where}:${p.name}`;
    if (seen.has(key)) continue;
    seen.add(key);
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
 * Build an OpenAPI requestBody from body-typed parameters and an optional
 * example parsed from the Markdown.
 *
 * @param {Array<{ name: string, type: string, in: string, description: string }>} parsedParams
 * @param {any | null} exampleRequestBody
 * @returns {any | undefined}
 */
function buildRequestBody(parsedParams, exampleRequestBody) {
  const bodyParams = parsedParams.filter((p) => p.in.toLowerCase() === 'body');
  if (bodyParams.length === 0 && !exampleRequestBody) return undefined;

  let schema;
  if (exampleRequestBody) {
    schema = inferSchema(exampleRequestBody);
  } else if (bodyParams.length > 0) {
    // Build schema from parameter declarations.
    const properties = {};
    for (const p of bodyParams) {
      properties[p.name] = mapParameterSchema(p.type);
      if (p.description) {
        properties[p.name].description = collapseWhitespace(p.description);
      }
    }
    schema = { type: 'object', properties };
  }

  const result = {
    required: true,
    content: {
      'application/json': { schema },
    },
  };

  if (exampleRequestBody) {
    result.content['application/json'].example = exampleRequestBody;
  }

  return result;
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

  const requestBody = buildRequestBody(parsed.parameters, parsed.exampleRequestBody);
  if (requestBody) {
    operation.requestBody = requestBody;
  }

  // Build responses from the example payload.
  if (parsed.exampleResponse && parsed.exampleResponse.body) {
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
  } else if (parsed.exampleResponse) {
    // Status code present but no body (e.g. 204 No Content).
    operation.responses = {
      [String(parsed.exampleResponse.status)]: {
        description: 'Successful response.',
      },
    };
  } else {
    // No example response in the Markdown — emit a minimal 200 placeholder.
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
        'Auto-generated from [Fleet\'s REST API docs](https://fleetdm.com/docs/rest-api/rest-api). ' +
        'To submit edits or fixes, update the [source docs](https://github.com/fleetdm/fleet/blob/main/docs/REST%20API/rest-api.md).',
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
    security: [{ bearerAuth: [] }],
    components: {
      securitySchemes: {
        bearerAuth: {
          type: 'http',
          scheme: 'bearer',
          description: 'API token passed as "Authorization: Bearer <token>".',
        },
      },
    },
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

module.exports = { buildDocument, buildPathItem, buildParameters, buildRequestBody };
