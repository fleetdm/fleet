# osquery_sql_parser

A standalone parser for the SQLite SQL flavor that osquery executes, used to
validate queries and extract table names in the browser (see
`frontend/utilities/sql_tools.ts` and
`frontend/components/forms/validators/validate_query/`).

The grammar is tightened to what osquery actually runs: **SELECT statements
only** (including CTEs, UNION, and VALUES inside CTEs). INSERT, UPDATE,
DELETE, CREATE, DROP, ALTER, ATTACH, etc. are parse errors by design.

## Provenance

`osquery_sql.pegjs` derives from
[sgress454/node-sql-parser](https://github.com/sgress454/node-sql-parser)
(Fleet's fork of
[taozhi8833998/node-sql-parser](https://github.com/taozhi8833998/node-sql-parser),
Apache-2.0 — see `LICENSE`) at tag `5.4.0-fork.2`, with three changes:

- the top-level statement rule accepts only SELECT/union statements
  (`crud_stmt = union_stmt / empty_stmt`)
- `CROSS JOIN` support added to `join_op`
- all rules made unreachable by the `crud_stmt` gate (the DDL/DML machinery:
  `create_table_stmt`, `update_stmt`, ...) are deleted. The pruned grammar
  compiles to a byte-identical parser, since peggy never emitted code for
  them anyway.

Because of the pruning, this file diffs against upstream's `sqlite.pegjs`
with large deletion hunks. When porting an upstream fix, diff upstream
against tag `5.4.0-fork.2`, then apply only the hunks that touch rules still
present here; a hunk that only touches deleted DDL/DML rules is irrelevant
by construction. Don't re-add statement types to `crud_stmt`.

Unlike the previously inlined copy of this parser (removed in #39076, which
was an unpatchable prebuilt blob), this directory vendors the *source
grammar*, so parser bugs are fixed here with a normal Fleet PR.

## Files

- `osquery_sql.pegjs` — the grammar. **This is the only file you edit.**
- `osquery_sql_parser.generated.js` — parser generated from the grammar.
  **Gitignored, never checked in.** It is regenerated automatically before
  every consumer runs: the `pretest`/`pretest:ci`/`prelint`/`prestorybook`
  yarn hooks and the `generate-js`/`generate-ci`/`generate-dev` make targets.
  To (re)create it manually — e.g. to read or debug the generated code — run
  `make generate-osquery-sql-parser` (or `yarn generate:osquery-sql-parser`).
- `osquery_sql_parser.generated.d.ts` — hand-written types for the generated
  module (peggy emits no types). Checked in.
- `index.ts` — the public wrapper (`astify`).

## Making a change

```
reproduce → find the rule → edit osquery_sql.pegjs → regenerate → test
```

1. **Reproduce.** Run the failing SQL through the grammar directly:
   ```sh
   yarn --silent peggy -t "SELECT ... the failing query ..." frontend/utilities/osquery_sql_parser/osquery_sql.pegjs
   ```
   On failure this prints a `SyntaxError` with a `location` (line/column) and
   an expected-token list that identify the grammar rule to widen. First
   confirm the SQL is valid in real SQLite (`sqlite3` CLI) — if SQLite
   rejects it too, fix the query, not the grammar.
2. **Edit the grammar.** Rules read like BNF: `rule = alt1 / alt2 { action }`.
   The action blocks build the AST, so the grammar is also the AST-shape
   documentation. Notes: `'FOO'i` matches case-insensitively, `__` is optional
   whitespace/comments, and alternation order matters (longest match first).
   Do not re-add DDL/DML alternatives to `crud_stmt`. If a change produces a
   new AST node shape, check whether `sql_tools.ts` needs to learn it.
3. **Test.** `yarn test frontend/utilities/osquery_sql_parser frontend/utilities/sql_tools.tests.ts frontend/components/forms/validators/validate_query`
   (`yarn test` regenerates the parser from the grammar first, via the
   `pretest` hook — always through the pinned peggy from the lockfile. The
   one thing that does NOT regenerate it is invoking jest directly, e.g.
   from an IDE test runner: after a grammar edit, run
   `make generate-osquery-sql-parser` first or your run uses the stale
   parser.) Add the fixed SQL to the tests in this directory as a regression
   case. If it was listed in the corpus test's known failures, remove it
   there.
4. **Commit** the grammar. The generated parser is gitignored — there is
   nothing else to commit.

## Pulling fixes from upstream

Diff upstream's `pegjs/sqlite.pegjs` against tag `5.4.0-fork.2` in the fork
repo and port the hunks manually onto `osquery_sql.pegjs` here. Upstream CVE
advisories remain visible via the `node-sql-parser` entry in
`third_party/vuln-check/package.json`.
