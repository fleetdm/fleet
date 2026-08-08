// Validate a generated OpenAPI document against the OpenAPI 3.1 schema.
//
// Library choice: @readme/openapi-parser
//   - Actively maintained fork of @apidevtools/swagger-parser
//   - Explicit support for OpenAPI 3.1 (the parent project lags 3.1)
//   - Returns parsed/dereferenced API on success; throws on validation errors
//
// We deliberately call .validate() on a deep-cloned doc to avoid the parser
// mutating our in-memory object (it dereferences $refs in place).
'use strict';

const OpenAPIParser = require('@readme/openapi-parser');

/**
 * @param {object} doc - the OpenAPI document as a JS object
 * @returns {Promise<void>} - resolves if valid, rejects with the validator's
 *   error if invalid
 */
async function validate(doc) {
  // Defensive clone — the parser dereferences in place.
  const clone = JSON.parse(JSON.stringify(doc));
  // Force the validator to require an explicit `openapi` version — refuse
  // anything that's not 3.1.x.
  if (typeof clone.openapi !== 'string' || !clone.openapi.startsWith('3.1')) {
    throw new Error(
      `expected OpenAPI 3.1.x; got ${JSON.stringify(clone.openapi)}`,
    );
  }
  await OpenAPIParser.validate(clone);
}

module.exports = { validate };
