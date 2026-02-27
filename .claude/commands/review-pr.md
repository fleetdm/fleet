Review the pull request: $ARGUMENTS

Use `gh pr view` and `gh pr diff` to get the full context.

Review the changes focusing on:
1. **Correctness** — logic errors, edge cases, nil pointer risks
2. **Go idioms** — error handling with ctxerr, proper context usage, slog logging
3. **SQL safety** — injection risks, missing indexes for new queries, migration correctness
4. **Test coverage** — are new code paths tested? Are integration tests needed?
5. **Fleet conventions** — matches patterns in surrounding code

For each issue found, cite the specific file and line. Categorize findings as:
- **Must fix** — bugs, security issues, data loss risks
- **Should fix** — convention violations, missing error handling
- **Nit** — style preferences, minor improvements

Be concise. Don't comment on things that are fine.
