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
export const astify = (sql: string): unknown => parse(sql).ast;

export { SyntaxError };
export type {
  ParserExpectation,
  ParserSyntaxError,
} from "./osquery_sql_parser.generated";
