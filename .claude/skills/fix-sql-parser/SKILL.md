---
name: fix-sql-parser
description: Fix the osquery SQL parser when it rejects a valid query (or accepts an invalid one). Use when a query is flagged as invalid in the query editor but runs fine in osquery, when asked to "fix the SQL parser", "fix the grammar", "this query is valid but Fleet says syntax error", or when a corpus knownFailure needs closing.
allowed-tools: Bash(yarn *), Bash(npx *), Bash(sqlite3 *), Bash(make generate-osquery-sql-parser), Bash(node *), Bash(git diff*), Bash(git status*), Read, Edit, Grep, Glob
effort: high
---

# Fix an osquery SQL parser bug

Query to fix: $ARGUMENTS

The parser lives in `frontend/utilities/osquery_sql_parser/`. Read that
directory's `README.md` first — it is the source of truth for this workflow and
for what the grammar deliberately excludes. This skill is the executable
version of its "Making a change" section.

## First: is it actually a bug?

Do not skip this. Three things are rejected **on purpose**, and "fixing" them
would undo deliberate decisions:

1. **Non-SELECT statements.** osquery is read-only. `INSERT`, `UPDATE`,
   `DELETE`, `CREATE`, `DROP`, `ALTER`, `ATTACH`, `PRAGMA`, `EXPLAIN`,
   `BEGIN` are parse errors by design. Never re-add statement types to
   `crud_stmt`.
2. **Bind parameters** (`?`, `:name`, `@name`, `$name`). Valid SQLite, but
   nothing in Fleet supplies bindings, so an unbound parameter silently matches
   nothing. Rejecting them is intentional.
3. **MySQL/Postgres dialect** osquery has no equivalent for: `FOR UPDATE`,
   `INTERVAL`, `@@variables`, `RLIKE`, `FROM DUAL`, `#` comments, `@>` and
   other jsonb operators, `COLLATE =`. See the README's Provenance section for
   the full list.

Then confirm the query is valid for the engine, using real SQLite:

```bash
sqlite3 :memory: "<the query>"
```

A syntax error here means **the query is wrong, not the grammar** — say so and
stop. (Errors about missing tables are expected and fine; osquery's tables
don't exist in a bare sqlite3. Only syntax errors matter.)

If the query is valid SQLite and isn't in one of the three excluded categories,
it is a real bug. Continue.

## 1. Reproduce against the grammar

```bash
yarn --silent peggy -t "<the query>" frontend/utilities/osquery_sql_parser/osquery_sql.pegjs
```

The `SyntaxError` it prints carries a `location` (line/column) and an
`expected` list. Together these tell you which rule gave up and what it wanted
— that is the rule to widen. Note the column and read the SQL at that offset;
the failure is usually one token past the construct the grammar can't handle.

## 2. Add the failing case to `corpus.json` and watch it fail

`frontend/utilities/osquery_sql_parser/corpus.json` is hand-maintained. Add:

```json
{"name": "short description of the construct", "query": "<the query>"}
```

No `expect` field means "must parse". If you are instead fixing the opposite
problem — the parser accepts something it shouldn't — add
`"expect": "reject"` and a `note` saying why.

**If the query is already in the corpus as a `knownFailure`, delete its
`knownFailure` and `expect` fields now, before touching the grammar.** Left
alone, that entry asserts the gap still exists, so it passes until your fix
lands and then fails — backwards. Removing the fields is what makes it red.

Now run the tests and **confirm they fail, naming your entry**:

```bash
yarn test frontend/utilities/osquery_sql_parser
```

You should see `✕ parses every query expected to parse` with your entry's name
in the reported array. A red test for the right reason before any grammar edit
is what proves the fix actually did something. If it passes here, either the
query already parses or the entry is wrong — investigate before continuing.

## 3. Edit the grammar

Edit only `osquery_sql.pegjs`. **Never** edit
`osquery_sql_parser.generated.js`; it is gitignored and rebuilt from the
grammar.

Rules read like BNF: `rule = alt1 / alt2 { action }`. Notes:

- `'FOO'i` matches case-insensitively
- `__` is optional whitespace/comments
- alternation order matters — longest match first
- action blocks build the AST, so the grammar doubles as AST documentation

Prefer the smallest widening that fixes the case: add an alternative to the
rule that failed, rather than restructuring. If your change alters the shape of
an AST node, check whether `frontend/utilities/sql_tools.ts`'s visitor needs to
learn it — that file walks the AST to extract table names.

## 4. Test

```bash
yarn test frontend/utilities/osquery_sql_parser frontend/utilities/sql_tools.tests.ts frontend/components/forms/validators/validate_query
```

`yarn test` regenerates the parser first via the `pretest` hook, always through
the lockfile-pinned peggy. Invoking jest directly does not — if you do, run
`make generate-osquery-sql-parser` first or you test a stale parser.

Your new entry should now pass.

## 5. Check you did not widen too far

This is the step that catches real mistakes. A loosened rule can start
accepting dialect syntax the grammar deliberately rejects, and the corpus is
the guard: it holds ~57 `expect: "reject"` entries covering exactly that.

```bash
yarn test frontend/utilities/osquery_sql_parser   # corpus: parse + reject + gaps
npx tsc --noEmit -p tsconfig.json
yarn lint
```

Then spot-check the exclusions by hand — each of these must still be rejected:

```bash
for q in "INSERT INTO users (name) VALUES ('x')" "SELECT @@version" \
         "SELECT * FROM users FOR UPDATE" "SELECT 1 FROM users WHERE uid = ?"; do
  printf '%-45s ' "$q"
  yarn --silent peggy -t "$q" frontend/utilities/osquery_sql_parser/osquery_sql.pegjs \
    >/dev/null 2>&1 && echo "ACCEPTED - BUG, you widened too far" || echo "rejected (correct)"
done
```

Finally, run the whole frontend suite, since `sql_tools` and the query
validator feed several pages:

```bash
yarn test
```

## 6. Report

Tell the user:

- the rule you changed and why that was the right one
- the corpus entry you added
- confirmation that the reject cases and full suite still pass

Commit the grammar and `corpus.json` together. The generated parser is
gitignored, so there is nothing else to commit. Do not add issue or PR numbers
to comments in either file.

## Rules

- Verify against real `sqlite3` before changing the grammar. If SQLite rejects
  the query, the query is the bug.
- Red test first, always: corpus entry before grammar edit.
- Only `osquery_sql.pegjs` is edited. The generated parser is never edited or
  committed.
- `crud_stmt` stays SELECT-only.
- Do not delete or restructure unrelated rules while you are in there.
- If the fix would require re-admitting a deliberate exclusion, stop and ask
  rather than deciding for the user.
