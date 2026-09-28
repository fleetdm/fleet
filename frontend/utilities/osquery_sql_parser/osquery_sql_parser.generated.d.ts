// Types for the peggy-generated parser (osquery_sql_parser.generated.js).

export interface ParserSyntaxErrorLocation {
  offset: number;
  line: number;
  column: number;
}

export interface ParserExpectation {
  type: "literal" | "class" | "any" | "end" | "other";
  // Set for type "literal": the exact text that would have been accepted.
  text?: string;
  ignoreCase?: boolean;
  // Set for type "other": a human-readable description.
  description?: string;
}

export interface ParserSyntaxError extends Error {
  expected: ParserExpectation[] | null;
  found: string | null;
  location: {
    start: ParserSyntaxErrorLocation;
    end: ParserSyntaxErrorLocation;
  };
}

export interface ParseResult {
  tableList: string[];
  columnList: string[];
  // The AST built by the grammar's action blocks. Callers walk it
  // generically (see sql_tools.ts), so it is intentionally untyped.
  ast: unknown;
}

// Omit startRule to parse with the default "start" rule. "no_stmt" (enabled by
// --allowed-start-rules in the generate script) returns nothing useful; callers
// only check whether it throws.
export interface ParseOptions {
  startRule?: "no_stmt";
}

export function parse(input: string, options?: ParseOptions): ParseResult;

export const StartRules: readonly string[];

export const SyntaxError: {
  new (...args: unknown[]): ParserSyntaxError;
  prototype: ParserSyntaxError;
};
