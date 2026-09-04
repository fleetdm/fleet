/**
 * Regression test driven by corpus.json, a frozen corpus of osquery SQL: one
 * query per distinct SQL shape found in this repo's queries (standard query
 * library, CIS packs, dogfood GitOps), plus hand-written grammar cases.
 *
 * An entry with no `expect` must parse -- that is the default and the vast
 * majority. `expect: "reject"` means the parser must not accept it, and carries
 * either `knownFailure` (valid osquery SQL we cannot parse yet, so fixing it
 * means updating the entry) or a `note` explaining the rejection: broken SQL,
 * or a deliberate exclusion such as non-SELECT statements and bound parameters
 * nothing in Fleet can bind. `name` appears only on hand-written cases whose
 * intent is not obvious from the SQL.
 *
 * corpus.json is hand-maintained: add a case to it when you fix or find a
 * parser bug.
 */

import { astify } from ".";
import corpus from "./corpus.json";

interface ICorpusQuery {
  query: string;
  name?: string;
  expect?: "reject";
  knownFailure?: string;
  note?: string;
}

const queries = corpus.queries as ICorpusQuery[];

const parses = (sql: string) => {
  try {
    astify(sql);
    return true;
  } catch {
    return false;
  }
};

// A failing entry is reported by its label when it has one, otherwise by the
// SQL itself -- which is what you need to debug a parse error.
const label = ({ name, query }: ICorpusQuery) =>
  name ?? query.replace(/\s+/g, " ").trim().slice(0, 90);

describe("osquery_sql_parser query corpus", () => {
  it("corpus is populated and every entry has a query", () => {
    expect(queries.length).toBeGreaterThan(400);
    expect(queries.filter(({ query }) => !query?.trim())).toEqual([]);
  });

  it("parses every query expected to parse", () => {
    const failures = queries
      .filter(({ expect: e }) => !e)
      .filter(({ query }) => !parses(query))
      .map(label);
    expect(failures).toEqual([]);
  });

  it("rejects every query expected to be rejected", () => {
    const accepted = queries
      .filter(({ expect: e, knownFailure }) => e === "reject" && !knownFailure)
      .filter(({ query }) => parses(query))
      .map(label);
    expect(accepted).toEqual([]);
  });

  // Separate from the assertion above so the message says what to do: these
  // are valid queries we could not parse, so parsing one is an improvement
  // that needs its corpus entry updated, not a regression.
  it("known grammar gaps still fail (set expect to parse once one is fixed)", () => {
    const nowParsing = queries
      .filter(({ knownFailure }) => knownFailure)
      .filter(({ query }) => parses(query))
      .map(label);
    expect(nowParsing).toEqual([]);
  });
});
