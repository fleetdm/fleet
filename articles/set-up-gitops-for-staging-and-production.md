# Set up Fleet GitOps for staging and production

Rolling out every configuration change straight to production is risky. This guide covers how to structure your Fleet GitOps repository and secrets so you can test changes in staging before they reach production hosts.

## Prerequisites

- An existing Fleet GitOps setup ([Fleet's GitOps reference](https://fleetdm.com/docs/configuration/yaml-files))
- Admin access to your GitHub repository settings
- A staging Fleet instance and a production Fleet instance, each with its own API token

## Choose one repo with branches, or two separate repos

Fleet's GitOps templates don't include a built-in staging/production structure, you have to decide how to split them. Two approaches work well:

### Option 1: One repo, two branches

Use a `staging` branch that deploys to your staging instance and a `main` branch that deploys to production. Promote a change by opening a PR from `staging` into `main` after it's been validated.

- Easier to promote changes, since the diff between `staging` and `main` shows exactly what's about to ship to production.
- One set of collaborators, PR templates, and workflow files to maintain.
- Requires careful secret scoping (see the next section) so a workflow running on `staging` can never read production secrets.

### Option 2: Two separate repos

Keep a `fleet-gitops-staging` repo and a `fleet-gitops-production` repo, each with its own secrets, collaborators, and workflow.

- Clean secret isolation. There's no shared workflow file that could accidentally reference the wrong environment's secret.
- Promoting a change means copying files between repos, or scripting a sync, since there's no single diff to review.
- Twice the repo administration: two sets of branch protection rules, two workflow files to keep in sync as Fleet's GitOps schema evolves.

> **Note:** If you're not sure which to pick, start with one repo and two branches. It's the lower-maintenance option, and GitHub environments (below) give you the secret isolation you'd otherwise get from splitting repos.

## Scope secrets to each environment

Whichever structure you choose, don't rely on a single set of repository secrets for both staging and production. A repository secret (**Settings > Secrets and variables > Actions > Repository secrets**) is available to every workflow run in that repo, regardless of which branch triggered it. That means a typo in a branch condition could point a staging run at your production API token.

Instead, use [GitHub environments](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment) to scope secrets to a specific branch:

1. Go to **Settings > Environments** in your GitHub repo and create two environments, for example `staging` and `production`.
2. Under each environment's **Deployment branches and tags**, restrict it to the matching branch (`staging` or `main`).
3. Add each environment's `FLEET_URL` and `FLEET_API_TOKEN` as environment secrets, scoped to that environment only.
4. In your GitOps workflow file, reference the environment for each job so it pulls the right secrets:

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: ${{ github.ref == 'refs/heads/main' && 'production' || 'staging' }}
    steps:
      - uses: actions/checkout@v4
      - name: Run fleetctl gitops
        env:
          FLEET_URL: ${{ secrets.FLEET_URL }}
          FLEET_API_TOKEN: ${{ secrets.FLEET_API_TOKEN }}
        run: fleetctl gitops --config ./default.yml
```

This way, a run triggered from `staging` can only ever see the staging environment's secrets, even if the workflow file has a bug.

## Enable secrets for forked repo pull requests

If contributors work from forks (common if you're managing a large or open GitOps repo), GitHub doesn't pass secrets to workflows triggered by a fork's pull request by default. This is a deliberate security measure: it stops an external contributor from opening a PR that exfiltrates your secrets through a modified workflow file.

> **Warning:** Only enable this if you understand the risk. Any workflow that has access to your secrets and runs against fork PR code can be used to leak those secrets, since the fork's author controls the workflow's behavior at PR time.

If you need fork PR workflows to run against a real Fleet instance (for example, to dry-run GitOps changes before merge), keep in mind that `fleetctl gitops --dry-run` still authenticates to Fleet and validates the change against its current state. It only skips the final apply step, so it still needs a valid API token. There's no secret-free way to run it.

Two safer options instead of exposing your production secrets broadly:

- Point fork PR dry runs at your staging instance's token instead of production's. A leaked staging token exposes less than a leaked production one.
- Require approval before secrets are exposed. Under **Settings > Environments**, enable **Required reviewers** on the environment. A maintainer then has to approve the run before it can access that environment's secrets, even from a fork PR.

Only as a last resort, under **Settings > Actions > General > Fork pull request workflows from outside collaborators**, enable "Send secrets and variables to workflows from fork pull requests." Treat this the same as giving every fork contributor direct access to your secrets.

## Further reading

- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Preventing mistakes with GitOps](https://fleetdm.com/guides/preventing-mistakes-with-gitops)
- [GitHub docs: using environments for deployment](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment)

<meta name="articleTitle" value="Set up Fleet GitOps for staging and production">
<meta name="authorFullName" value="TODO: fill in before publishing">
<meta name="authorGitHubUsername" value="TODO: fill in before publishing">
<meta name="category" value="guides">
<meta name="publishedOn" value="TODO: fill in before publishing">
<meta name="description" value="How to structure your Fleet GitOps repo and secrets to safely test changes in staging before they reach production.">
