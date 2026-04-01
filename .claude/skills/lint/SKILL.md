---
name: lint
description: Run linters on recently changed files with the correct tools for each language. Use when asked to "lint", "check style", or "run linters".
allowed-tools: Bash(golangci-lint *), Bash(go vet*), Bash(yarn lint*), Bash(yarn --cwd *), Bash(npx eslint*), Bash(npx prettier*), Bash(git diff*), Bash(git status*), Read, Grep, Glob
effort: low
---

# Lint recent changes

Run the appropriate linters on files changed in the current branch.

## Process

### 1. Detect changed files

```bash
git diff --name-only HEAD~1
git diff --name-only --cached
git diff --name-only
```

Combine all three to get the full set of changed files.

### 2. Run linters by language

**Go files** (`*.go`):
Identify which packages have changes, then lint them:
```bash
golangci-lint run ./path/to/changed/package/...
```
If `golangci-lint` isn't available, fall back to `go vet ./path/...`.

**TypeScript/JavaScript files** (`*.ts`, `*.tsx`, `*.js`, `*.jsx`):
```bash
npx eslint frontend/path/to/changed/files
npx prettier --check frontend/path/to/changed/files
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
- `go` — only Go linters
- `js` or `frontend` — only frontend linters
- A file path — lint that specific file/package
