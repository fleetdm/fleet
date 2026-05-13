// Infer a JSON Schema (Draft 2020-12, the dialect OpenAPI 3.1 uses) from an
// example response body. The PoC philosophy: lossy is OK. We capture types
// and structure so consumers see *what to expect*; the canonical truth lives
// in the Markdown and Go handler code.
//
// Notable handling:
//   - For arrays, we use the first element as the prototype for `items`. If
//     the array is empty, we fall back to `items: {}` (any).
//   - For objects, every present key becomes a property; we do NOT mark any
//     properties as `required` since the example may omit optional fields.
//   - For `null` we emit `nullable: true` with `type: "string"` as a fallback;
//     in OpenAPI 3.1 `nullable` is deprecated in favor of `type: ["x","null"]`,
//     so we emit the union form.
//   - We do NOT collapse oneOf/anyOf across siblings in an array; that would
//     require multi-example synthesis which is out of PoC scope.
'use strict';

function inferSchema(value) {
  if (value === null) {
    return { type: ['string', 'null'] };
  }
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return { type: 'array', items: {} };
    }
    return { type: 'array', items: inferSchema(value[0]) };
  }
  switch (typeof value) {
    case 'string': {
      // Detect ISO 8601 timestamps so consumers get format hints.
      if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(value)) {
        return { type: 'string', format: 'date-time' };
      }
      return { type: 'string' };
    }
    case 'number':
      return Number.isInteger(value)
        ? { type: 'integer' }
        : { type: 'number' };
    case 'boolean':
      return { type: 'boolean' };
    case 'object': {
      /** @type {Record<string, any>} */
      const properties = {};
      for (const [k, v] of Object.entries(value)) {
        properties[k] = inferSchema(v);
      }
      return { type: 'object', properties };
    }
    default:
      // Functions, symbols, undefined — shouldn't appear in JSON. Be defensive.
      return {};
  }
}

module.exports = { inferSchema };
