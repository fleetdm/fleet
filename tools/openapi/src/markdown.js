// Markdown parser for Fleet's canonical REST API reference.
//
// This module is intentionally narrow: given the full Markdown text and a
// heading line (e.g. "### List hosts"), it extracts the structured data we
// need to emit an OpenAPI operation. It does NOT try to be a general-purpose
// Markdown parser; it relies on the conventions used in
// docs/REST API/rest-api.md:
//
//   ### <Section name>                          (operation heading)
//
//   `<METHOD> <path>`                            (request line, backticked)
//
//   #### Parameters
//
//   | Name | Type | In | Description |
//   | ---- | ---- | -- | ----------- |
//   | ...                                         (parameter rows)
//
//   #### Example
//
//   `<METHOD> <path>?<query>`
//
//   ##### Default response
//
//   `Status: <code>`
//
//   ```json
//   { ... }                                      (example response body)
//   ```
//
// The next operation heading is taken as the terminator for the current
// section.
'use strict';

/**
 * Extract the section of Markdown that documents one endpoint.
 *
 * @param {string} md - the full Markdown contents of rest-api.md
 * @param {string} heading - the exact heading line, e.g. "### List hosts"
 * @returns {string} - the slice of Markdown from the heading (inclusive) up to
 *   the next sibling heading at the same level (exclusive). Throws if not found.
 */
function extractSection(md, heading) {
  const lines = md.split('\n');
  const start = lines.findIndex((l) => l.trim() === heading.trim());
  if (start === -1) {
    throw new Error(`heading not found in Markdown: ${JSON.stringify(heading)}`);
  }
  const level = heading.match(/^#+/)[0].length;
  // Find the next heading at the same OR shallower level (treat shallower as
  // a section boundary too, e.g. ## Hosts → ## Labels).
  const headingRe = new RegExp(`^#{1,${level}} `);
  let end = lines.length;
  for (let i = start + 1; i < lines.length; i++) {
    if (headingRe.test(lines[i])) {
      end = i;
      break;
    }
  }
  return lines.slice(start, end).join('\n');
}

/**
 * Parse the HTTP method + path from the first backticked request line
 * directly under the section heading.
 *
 * @param {string} section
 * @returns {{ method: string, path: string }}
 */
function parseRequestLine(section) {
  // First non-heading, non-blank line is the request line, formatted as
  //   `GET /api/v1/fleet/hosts`
  // We tolerate optional surrounding whitespace.
  const lines = section.split('\n');
  for (let i = 1; i < lines.length; i++) {
    const trimmed = lines[i].trim();
    if (!trimmed) continue;
    if (trimmed.startsWith('#')) continue;
    const m = trimmed.match(/^`(GET|POST|PUT|PATCH|DELETE)\s+(\/[^`?\s]+)`$/);
    if (m) {
      return { method: m[1].toLowerCase(), path: m[2] };
    }
    // If the first non-blank line is something else, this section doesn't
    // follow the convention; surface that explicitly.
    throw new Error(
      `expected request line under heading, got: ${JSON.stringify(trimmed)}`,
    );
  }
  throw new Error('no request line found in section');
}

/**
 * Parse the "#### Parameters" table into a list of param objects.
 *
 * The table format Fleet uses:
 *   | Name | Type | In | Description |
 *   | ---- | ---- | -- | ----------- |
 *   | page | integer | query | Page number ... |
 *
 * Returns [] if no `#### Parameters` section exists (some GETs have none).
 *
 * @param {string} section
 * @returns {Array<{ name: string, type: string, in: string, description: string }>}
 */
function parseParametersTable(section) {
  const lines = section.split('\n');
  const headerIdx = lines.findIndex((l) => l.trim() === '#### Parameters');
  if (headerIdx === -1) return [];

  // Find first table row (`| ... |`) after the header.
  let i = headerIdx + 1;
  while (i < lines.length && !lines[i].trim().startsWith('|')) i++;
  if (i >= lines.length) return [];

  // Skip the header row and the `| --- |` separator row.
  const headerRow = lines[i];
  const headerCols = splitTableRow(headerRow);
  const expected = ['name', 'type', 'in', 'description'];
  const actual = headerCols.map((c) => c.toLowerCase());
  if (
    actual.length < 4 ||
    actual[0] !== expected[0] ||
    actual[1] !== expected[1] ||
    actual[2] !== expected[2] ||
    actual[3] !== expected[3]
  ) {
    throw new Error(
      `unexpected Parameters table header: ${JSON.stringify(headerRow)}`,
    );
  }
  i++; // skip header
  if (i < lines.length && /^\|[\s-:|]+\|$/.test(lines[i].trim())) i++; // skip separator

  /** @type {Array<{ name: string, type: string, in: string, description: string }>} */
  const params = [];
  while (i < lines.length && lines[i].trim().startsWith('|')) {
    const cols = splitTableRow(lines[i]);
    if (cols.length >= 4 && cols[0]) {
      params.push({
        name: cols[0],
        type: cols[1] || 'string',
        in: cols[2] || 'query',
        description: cols[3] || '',
      });
    }
    i++;
  }
  return params;
}

/**
 * Split a Markdown table row into its cells. Strips the leading and trailing
 * `|` and trims each cell. Does NOT support escaped pipes (`\|`) — Fleet's
 * Markdown does not use them in the parameter tables we target.
 */
function splitTableRow(row) {
  const trimmed = row.trim();
  if (!trimmed.startsWith('|') || !trimmed.endsWith('|')) return [];
  const inner = trimmed.slice(1, -1);
  return inner.split('|').map((c) => c.trim());
}

/**
 * Extract the first ```json``` code block that appears under the
 * `##### Default response` heading. Returns the parsed JSON, plus the
 * declared status code if present.
 *
 * Some examples use `// comment` lines (e.g. "// Fleet Premium only"). We
 * strip line-comments before parsing so the JSON is valid.
 *
 * @param {string} section
 * @returns {{ status: number, body: any } | null}
 */
function parseDefaultResponse(section) {
  const lines = section.split('\n');
  const headerIdx = lines.findIndex(
    (l) => l.trim() === '##### Default response',
  );
  if (headerIdx === -1) return null;

  // Find `Status: <code>` line (backticked) and the json fence after it.
  let status = 200;
  let i = headerIdx + 1;
  for (; i < lines.length; i++) {
    const t = lines[i].trim();
    const m = t.match(/^`Status:\s*(\d{3})`$/);
    if (m) {
      status = parseInt(m[1], 10);
      i++;
      break;
    }
    if (t.startsWith('```json')) break;
    if (t.startsWith('#')) return null; // ran off the end of the subsection
  }

  // Find ```json fence.
  while (i < lines.length && !lines[i].trim().startsWith('```json')) i++;
  if (i >= lines.length) return null;
  i++; // step into fence body
  const bodyLines = [];
  while (i < lines.length && !lines[i].trim().startsWith('```')) {
    bodyLines.push(lines[i]);
    i++;
  }
  const raw = bodyLines.join('\n');
  // Strip `// ...` line comments from the JSON example (Fleet uses them to
  // annotate Premium-only fields). We only strip when the `//` is NOT inside
  // a string literal. Use a tolerant approach: walk char-by-char tracking
  // string state.
  const stripped = stripJSONLineComments(raw);
  let body;
  try {
    body = JSON.parse(stripped);
  } catch (err) {
    throw new Error(
      `failed to parse example response JSON: ${err.message}\n--- input ---\n${stripped}\n--- end ---`,
    );
  }
  return { status, body };
}

/**
 * Strip `// ...` line comments from a JSON-ish string, preserving // inside
 * string literals. Trailing commas in the source are NOT corrected here; if
 * the example happens to have one, JSON.parse will surface it.
 */
function stripJSONLineComments(src) {
  let out = '';
  let inString = false;
  let escape = false;
  for (let i = 0; i < src.length; i++) {
    const c = src[i];
    if (inString) {
      out += c;
      if (escape) {
        escape = false;
      } else if (c === '\\') {
        escape = true;
      } else if (c === '"') {
        inString = false;
      }
      continue;
    }
    if (c === '"') {
      inString = true;
      out += c;
      continue;
    }
    if (c === '/' && src[i + 1] === '/') {
      // Skip until newline.
      while (i < src.length && src[i] !== '\n') i++;
      // Preserve the newline so JSON line numbers don't shift.
      if (i < src.length) out += '\n';
      continue;
    }
    out += c;
  }
  return out;
}

/**
 * Parse one endpoint section from rest-api.md.
 *
 * @param {string} md - full Markdown contents
 * @param {string} heading - e.g. "### List hosts"
 * @returns {{
 *   method: string,
 *   path: string,
 *   parameters: Array<{ name: string, type: string, in: string, description: string }>,
 *   exampleResponse: { status: number, body: any } | null,
 * }}
 */
function parseEndpoint(md, heading) {
  const section = extractSection(md, heading);
  const { method, path } = parseRequestLine(section);
  const parameters = parseParametersTable(section);
  const exampleResponse = parseDefaultResponse(section);
  return { method, path, parameters, exampleResponse };
}

module.exports = {
  parseEndpoint,
  // exported for unit-style ad-hoc testing if a future contributor wants them
  extractSection,
  parseRequestLine,
  parseParametersTable,
  parseDefaultResponse,
};
