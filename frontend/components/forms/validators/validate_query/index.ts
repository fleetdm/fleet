import { astify } from "utilities/osquery_sql_parser";
import type { ParserSyntaxError } from "utilities/osquery_sql_parser";

export const EMPTY_QUERY_ERR = "Query text must be present";
export const INVALID_SYNTAX_ERR = "Syntax error. Please review before saving.";

export const invalidSyntaxOnLineErr = (line: number, column: number) =>
  `Syntax error on line ${line}, column ${column}. Please review before saving.`;
export const expectedSelectErr = (line: number) =>
  `Expected a SELECT statement on line ${line}. osquery only supports SELECT statements.`;

const invalidQueryResponse = (message: string) => {
  return { valid: false, error: message };
};
const validQueryResponse = { valid: true, error: null };

const describeSyntaxError = (err: unknown): string => {
  const { location, expected } = err as Partial<ParserSyntaxError>;
  if (!location) {
    return INVALID_SYNTAX_ERR;
  }

  // When the parser would have accepted a SELECT keyword and, alternatively,
  // the end of the input, the error is at a statement position — meaning the
  // input is something other than the SELECT statements osquery executes
  // (e.g. INSERT, DROP, or free text) rather than a typo inside one.
  const expectsSelect = (expected ?? []).some(
    (e) => e.type === "literal" && e.text?.toUpperCase() === "SELECT"
  );
  const expectsEndOfInput = (expected ?? []).some((e) => e.type === "end");
  if (expectsSelect && expectsEndOfInput) {
    return expectedSelectErr(location.start.line);
  }

  return invalidSyntaxOnLineErr(location.start.line, location.start.column);
};

export const validateQuery = (queryText?: string) => {
  if (!queryText?.trim()) {
    return invalidQueryResponse(EMPTY_QUERY_ERR);
  }

  try {
    astify(queryText);
    return validQueryResponse;
  } catch (err) {
    return invalidQueryResponse(describeSyntaxError(err));
  }
};
