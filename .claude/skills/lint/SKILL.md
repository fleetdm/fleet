---
name: lint
description: Run linters on recently changed files with the correct tools for each language. Use when asked to "lint", "check style", or "run linters".
allowed-tools: Bash(make lint*), Bash(golangci-lint *), Bash(go vet*), Bash(yarn lint*), Bash(yarn --cwd *), Bash(npx eslint*), Bash(npx prettier*), Bash(git diff*), Bash(git status*), Read, Grep, Glob
effort: low
---

# Lint recent changes

Run the appropriate linters on files changed in the current branch. Use the project's own make targets when available.

## Process

### 1. Detect changed files

Find recently changed files (last commit, staged, and unstaged):

```bash
git diff --name-only HEAD~1   # Last commit
git diff --name-only --cached # Staged but not committed
git diff --name-only          # Unstaged changes
```

Combine all three and deduplicate to get the full set.

### 2. Run linters by language

**Go files** (`*.go`):
Use the project's incremental linter — it only checks changes since branching from main:
```bash
make lint-go-incremental
```
This uses `.golangci-incremental.yml` with `--new-from-merge-base=origin/main`. It's faster and more relevant than linting entire packages.

For a full lint (e.g., before committing), use:
```bash
make lint-go
```

**TypeScript/JavaScript files** (`*.ts`, `*.tsx`, `*.js`, `*.jsx`):
```bash
npx eslint frontend/path/to/changed/files
npx prettier --check frontend/path/to/changed/files
```

Or use the make target:
```bash
make lint-js
```

**SCSS files** (`*.scss`):
```bash
npx prettier --check frontend/path/to/changed/files.scss
```

### 3. Report results

For each linter run, show:
- Which packages/files were linted
- Any errors or warnings found
- Suggested fixes (if the linter provides them)

If everything passes, confirm which linters ran and on which files.

If an argument is provided, use it to filter: $ARGUMENTS
- `go` — only Go linters (uses `make lint-go-incremental`)
- `full` — full Go lint (uses `make lint-go`)
- `js` or `frontend` — only frontend linters (uses `make lint-js`)
- A file path — lint that specific file/package
