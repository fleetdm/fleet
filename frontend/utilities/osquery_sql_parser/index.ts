import { parse, SyntaxError } from "./osquery_sql_parser.generated";

/**
 * Parses a string of osquery SQL (SQLite flavor, SELECT statements only) and
 * returns its AST. Throws a SyntaxError — with `location` and `expected`
 * details — for anything the parser cannot accept, including well-formed SQL
 * that osquery does not execute (INSERT, CREATE, etc.).
 *
 * The input is parsed as-is (the grammar tolerates surrounding whitespace and
 * comments), so error locations always match the original text's line and
 * column numbers.
 */
export const astify = (sql: string): unknown => {
  const result = parse(sql);

  // The grammar's start rule always returns { tableList, columnList, ast }, so
  // this only trips if a grammar edit changes that contract. Checking it here
  // fails loudly at the boundary instead of leaking an undefined AST into
  // sql_tools' visitor, where it would surface as an unrelated error.
  if (!result || typeof result !== "object" || !("ast" in result)) {
    throw new Error(
      "osquery_sql_parser: expected the parser to return an object with an `ast` property"
    );
  }

  return result.ast;
};

export { SyntaxError };
export type {
  ParserExpectation,
  ParserSyntaxError,
} from "./osquery_sql_parser.generated";
