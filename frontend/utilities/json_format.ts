/**
 * Indents already-valid JSON by inserting whitespace between tokens. Tokens
 * themselves are copied verbatim, so the result stays byte-for-byte faithful to
 * the input -- reserializing via JSON.parse/stringify would silently truncate
 * integers past 2^53, reorder integer-like keys and rewrite number and escape
 * formatting, which matters when the JSON came from a device rather than from
 * us.
 */
const indentJson = (value: string): string => {
  const indentUnit = "  ";
  let out = "";
  let depth = 0;
  let inString = false;

  const newline = () => `\n${indentUnit.repeat(depth)}`;

  for (let i = 0; i < value.length; i += 1) {
    const char = value[i];

    if (inString) {
      out += char;
      if (char === "\\") {
        // an escape sequence can contain a quote, so copy the escaped
        // character with it rather than letting it end the string
        i += 1;
        out += value[i] ?? "";
      } else if (char === '"') {
        inString = false;
      }
    } else if (char === '"') {
      inString = true;
      out += char;
    } else if (char === "{" || char === "[") {
      const close = char === "{" ? "}" : "]";
      let next = i + 1;
      while (next < value.length && /\s/.test(value[next])) {
        next += 1;
      }
      if (value[next] === close) {
        // keep empty objects and arrays on one line
        out += char + close;
        i = next;
      } else {
        depth += 1;
        out += char + newline();
      }
    } else if (char === "}" || char === "]") {
      depth -= 1;
      out += newline() + char;
    } else if (char === ",") {
      out += char + newline();
    } else if (char === ":") {
      out += ": ";
    } else if (!/\s/.test(char)) {
      out += char;
    }
  }

  return out;
};

/**
 * Pretty-prints a string that may or may not be JSON, for display in a
 * read-only box. Only syntactically valid JSON objects and arrays are
 * reformatted, so plists, XML, plain text and bare scalars are returned
 * unchanged rather than mangled.
 */
const formatJsonForDisplay = (value: string): string => {
  let parsed;
  try {
    parsed = JSON.parse(value);
  } catch {
    return value;
  }
  if (parsed === null || typeof parsed !== "object") {
    return value;
  }
  return indentJson(value);
};

export default formatJsonForDisplay;
