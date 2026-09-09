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

// The generated parse() also accepts an options argument (start rule,
// tracer); it is deliberately omitted here — the wrapper never passes one.
export function parse(input: string): ParseResult;

export const StartRules: readonly string[];

export const SyntaxError: {
  new (...args: unknown[]): ParserSyntaxError;
  prototype: ParserSyntaxError;
};
