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
