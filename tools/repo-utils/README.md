# Backport

Automatically cherry-pick and create a PR to a target branch.

```
$ ./tools/repo-utils/backport.sh <commit-to-pick> <target-branch>
```

For example, to cherry pick commit `3c28b7f` to the branch `minor-1.1.1`

```
$ ./tools/repo-utils/backport.sh 3c28b7f minor-1.1.1
```


### Setup

- Install `jq` https://jqlang.github.io/jq/
- Setup a GitHub token, you have two options:
    a. Install the `gh` CLI and do `gh auth login`, the script will pick up the token from there.
    b. Set a `$GITHUB_TOKEN` environment variable with a token with read/write access to the repo.
