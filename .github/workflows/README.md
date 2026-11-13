# Github Actions

Fleet uses Github Actions for continuous integration (CI). This document describes best practices
and at patterns for writing and maintaining Fleet's Github Actions workflows.

## Bash

By default, Github Actions sets the shell to `bash -e` for linux and MacOS runners. To help write
safer bash scripts in run jobs and avoid common issues, override the default by adding the following
to the workflow file

```
defaults:
  run:
    # fail-fast using bash -eo pipefail. See https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#exit-codes-and-error-action-preference
    shell: bash
```

By specifying the default shell to `bash`, some extra flags are set. The option `pipefail` changes
the behaviour when using the pipe `|` operator such that if any command in a pipeline fails, that
commands return code will be used a the return code for the whole pipeline. Consider the following
example in `test-go.yaml`

```
    - name: Run Go Tests
      run: |
        # omitted ...
          make test-go 2>&1 | tee /tmp/gotest.log
```

If the `pipefail` option was *not* set, this job would always succeed because `tee` would always
return success. This is not the intended behavior.  Instead, we want the job to fail if `make
test-go` fails.

## Concurrency

Github Action runners are limited. If a lot of workflows are queued, they will wait in pending until
a runner becomes available. This has caused issue in the past where workflows take an excessively long
time to start. To help with this issue, use the following in workflows

```
# This allows a subsequently queued workflow run to interrupt previous runs
concurrency:
  group: ${{ github.workflow }}-${{ github.head_ref || github.run_id}}
  cancel-in-progress: true
```

When a workflow is triggered via a pull request, it will cancel previous running workflows for that
pull request. This is especially useful when changes are pushed to a pull request frequently.
Manually triggered workflows, workflows that run on a schedule, and workflows triggered by pushes to
`main` are unaffected.

## Go fuzz testing

The [Go fuzz workflow](./test-go-fuzz.yml) runs native Go fuzz tests without databases, credentials, or live devices.
Pull requests and pushes to `main` run each target for 60 seconds. Nightly and manual runs use 15 minutes per target.
The matrix runs one job per package, with at most four jobs running concurrently.
Each job discovers its fuzz targets and runs them sequentially, with four workers and a 1 GiB soft memory budget per process.
Failures do not stop the remaining targets. The job fails after all targets finish.
The job timeout is three hours to accommodate nightly runs of packages with multiple targets.
The targets cover:

- Windows CSRs, MSI tables, macOS package Distribution XML, and RPM version comparison.
- Apple device info, MDM signatures, BER depth limits, and certificate-binding extensions.
- Platform SSO party info, EC keys, encrypted assertions, nonces, and embedded passwords.
- ACME JWS prechecks and decoding, CMS recovery keys, SCEP challenges and enrollment, and ABM tokens.
- PKCS#12 certificate inventory.

Run all committed seed inputs from the repository root:

```sh
go test ./server/mdm/microsoft ./pkg/file ./server/vulnerabilities/utils \
  ./server/mdm ./server/mdm/apple ./server/mdm/nanomdm/cryptoutil \
  ./server/mdm/apple/psso/pssocrypto ./server/mdm/acme/api/http \
  ./server/mdm/scep/x509util ./server/mdm/scep/server \
  ./server/mdm/nanodep/tokenpki ./orbit/pkg/cryptoinfo -run '^Fuzz'
```

Run mutation fuzzing for one target:

```sh
go test ./server/mdm/microsoft -run '^$' \
  -fuzz '^FuzzParseCertificateRequestFromWindowsDevice$' \
  -fuzztime=60s -fuzzminimizetime=1s -parallel=2 -timeout=3m
```

Repeat with a package listed in the workflow matrix and a target discovered by `go test <package> -list '^Fuzz'`.
Regular `go test` replays seed inputs but does not generate mutations.
CI limits minimization to one second per input so large seeds do not consume the mutation budget.
Keep new targets fast and deterministic. Use synthetic inputs or existing public fixtures.
Check useful properties, not just whether parsing succeeds. Malformed input may return an error, but must not crash or hang.

The workflow limits workers and runtime. `GOMEMLIMIT` is a soft per-process memory budget, not an isolation boundary.
Harnesses also bound input sizes. Compressed formats need separate decompression limits.
The PKCS#12 target mutates certificate DER and passwords inside generated bundles with fixed, low KDF iteration counts.
It does not mutate arbitrary PFX KDF parameters, which could turn a small input into unbounded work.
Do not fuzz filesystem extraction or execute uploaded programs without disposable container or VM isolation.

Runs on `main` save generated corpora from `GOCACHE/fuzz` after the fuzz step succeeds or fails.
Caches are separate for each package and contain the coverage inputs for all its targets.
Cancelled runs and runs that skip the fuzz step do not save corpora.
Pull requests can restore those corpora but cannot save them.
Build and dependency caches are not included. Logs and failing inputs are uploaded as workflow artifacts for seven days.
Fuzz step failures on `main` notify `#help-engineering` with the failed targets, run link, and reproduction and fix instructions.
Pull request failures remain in GitHub checks and do not send Slack notifications.

When a run fails, download its artifacts and copy the minimized input into the affected package's `testdata/fuzz/<target>/` directory.
Reproduce it with `go test <package> -run '<target>/<input filename>'`.
Fix the cause and commit the minimized input with the fix so regular Go tests replay it.
Route failures to the package's owners in [CODEOWNERS](../../CODEOWNERS).

When adding a target, use the conventional `func FuzzXxx(f *testing.F)` signature in a `*_test.go` file.
Targets in listed packages are discovered automatically.
For a new package, add it to the workflow matrix and seed replay command above.
