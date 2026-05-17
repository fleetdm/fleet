// Allowlist of pilot endpoints to include in the generated OpenAPI 3.1 spec.
//
// The PoC ships ONE entry. Adding the remaining pilot endpoints from
// https://github.com/fleetdm/fleet/issues/45279 is a data-only change: append
// to this array. The Markdown parser keys off `markdownHeading` and extracts:
//   - the request line (`HTTP_METHOD /api/v1/...`)
//   - the `#### Parameters` table directly under the heading
//   - the first ```json``` block under `##### Default response`
//
// Each entry may also override or supplement what the parser detects. Hand-
// authored overrides are explicitly marked so a future contributor knows what
// to delete if/when parser support catches up to the Markdown reality.
'use strict';

/** @typedef {{
 *   markdownHeading: string,          // exact "### List hosts" style line, no trailing punctuation
 *   operationId: string,              // OpenAPI operationId — must be unique
 *   tag: string,                      // OpenAPI tag (groups related endpoints)
 *   summary: string,                  // one-line summary (used as the operation summary)
 *   description?: string,             // optional longer description
 *   pathOverride?: string,            // when the doc's request line uses :id, force /{id} form
 *   pathParameters?: Array<{ name: string, type: string, description: string }>,
 * }} EndpointSpec */

/** @type {EndpointSpec[]} */
const endpoints = [
  {
    markdownHeading: '### List hosts',
    operationId: 'listHosts',
    tag: 'Hosts',
    summary: 'List hosts',
    description:
      'Returns a paginated list of hosts enrolled in Fleet. Supports filtering, ' +
      'ordering, and optional inclusion of related data (software, policies, ' +
      'users, labels, device status).',
    // No path parameters for this endpoint; left empty for shape parity with
    // future endpoints like `### Get host` which take an `:id`.
    pathParameters: [],
  },
];

module.exports = { endpoints };
