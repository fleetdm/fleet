# Set up Fleet GitOps for staging and production

Rolling out every configuration change straight to production is risky. This guide covers how to structure your Fleet GitOps repository and secrets so you can test changes in staging before they reach production hosts.

## Prerequisites

- An existing Fleet GitOps setup ([Fleet's GitOps reference](https://fleetdm.com/docs/configuration/yaml-files))
- Admin access to your GitHub repository settings
- A staging Fleet instance and a production Fleet instance, each with its own API token

## Choose one repo with branches, or two separate repos

Fleet's GitOps templates don't include a built-in staging/production structure, so you have to decide how to split them. Two approaches work well:

### Option 1: Two separate repos

[Fork](https://docs.github.com/en/pull-requests/how-tos/work-with-forks/fork-a-repo) a `fleet-gitops-staging` repo from your `fleet-gitops-production` repo, each with its own secrets, collaborators, and workflow. On GitHub, promote changes by using the "Contribute" button to [open a PR from the fork](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request-from-a-fork) against the production repo. If production ever gets out of sync with staging (try to avoid this by always making changes in staging first and promoting them up to production), pull production changes back down into staging by [syncing the fork](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/syncing-a-fork).

- Clean secret isolation. There's no shared workflow file that could accidentally reference the wrong environment's secret.
- The ability to add branches to separate proposed changes and test them one at a time. This is great if you're working on the repo with multiple users, and working on multiple changes at a time that need to be tested individually.
- Twice the repo administration: two sets of [branch protection rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/managing-a-branch-protection-rule), two workflow files to keep in sync as Fleet's GitOps schema evolves.

### Option 2: One repo, two branches

Use a `staging` branch that deploys to your staging instance and a `main` branch that deploys to production. Promote a change by opening a PR from `staging` into `main` after it's been validated.

- One set of collaborators, PR templates, and workflow files to maintain.
- Requires careful secret scoping at the environment level (see the next section) so a workflow running on `staging` can never read production secrets.
- A little more complicated to stage and test multiple changes, as you're stuck with syncing two branches to your servers.

> **Note:** If you're not sure which to pick, start with two repos. If you're the only one making changes and have a good understanding of git, you may prefer using one repo with two branches.

## Scope secrets when using one repo with two branches

If you went with [Option 1](#option-1-two-separate-repos) above, you already get secret isolation for free: each repo has its own set of secrets, so there's no way for a staging run to see the production token. This section is for [Option 2](#option-2-one-repo-two-branches), where staging and production share a repo.

Don't rely on a single set of [repository secrets](https://docs.github.com/en/actions/security-for-github-actions/security-guides/using-secrets-in-github-actions) (**Settings > Secrets and variables > Actions > Repository secrets**) for both branches. A repository secret is available to every workflow run in the repo, regardless of which branch triggered it, so a typo in a branch condition could point a staging run at your production API token. There are two ways to avoid that.

### GitHub environments (recommended)

Use [GitHub environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments) to scope secrets to a specific branch, so a workflow running on the wrong branch can't read the secret at all, even if the conditional logic has a bug:

1. Go to **Settings > Environments** in your GitHub repo and create two environments, for example `staging` and `production`.
2. Under each environment's [**Deployment branches and tags**](https://docs.github.com/en/actions/reference/deployments-and-environments#deployment-branches-and-tags), restrict it to the matching branch (`staging` or `main`).
3. Add each environment's `FLEET_URL` and `FLEET_API_TOKEN` as [environment secrets](https://docs.github.com/en/actions/security-for-github-actions/security-guides/using-secrets-in-github-actions#creating-secrets-for-an-environment), scoped to that environment only.
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

### Repo secrets, chosen by branch

Add separate secrets for each environment, for example `STAGING_FLEET_API_TOKEN` and `PROD_FLEET_API_TOKEN`, and pick between them in your workflow with the [`github.ref` context](https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/accessing-contextual-information-about-workflow-runs#github-context), based on which branch triggered the run:

```yaml
env:
  FLEET_URL: ${{ github.ref == 'refs/heads/main' && secrets.PROD_FLEET_URL || secrets.STAGING_FLEET_URL }}
  FLEET_API_TOKEN: ${{ github.ref == 'refs/heads/main' && secrets.PROD_FLEET_API_TOKEN || secrets.STAGING_FLEET_API_TOKEN }}
```

This works on any GitHub plan, but it depends on that conditional being correct in every job. A typo in the branch check silently points a staging run at the production secret.

## Enable secrets for forked repo pull requests

If you went with [Option 1](#option-1-two-separate-repos) above, your staging repo is a fork of your production repo. GitHub treats a fork's pull requests as coming from an outside contributor by default, even when the fork lives in the same GitHub org. The "Contribute" PR [won't have access](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#workflows-in-forked-repositories) to your Fleet API token, so any dry-run step in that workflow will fail until you enable one of the options below. This same default also protects you if you have actual external contributors: it stops someone opening a PR that exfiltrates your secrets through a modified workflow file. GitHub covers this attack, and how to defend against it, in [Security hardening for GitHub Actions](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions).

> **Warning:** Only enable this if you understand the risk. GitHub masks a secret's value if a workflow prints it directly to the log, but that doesn't stop someone who controls the workflow's behavior: a modified workflow step can encode the secret before printing it (defeating the mask) or send it to an external server in a network request, which never touches the log at all. Any workflow that has access to your secrets and runs against fork PR code can do this, since the fork's author controls that workflow at PR time.

If you need fork PR workflows to run against a real Fleet instance (for example, to dry-run GitOps changes before merge), keep in mind that `fleetctl gitops --dry-run` still authenticates to Fleet and validates the change against its current state. It only skips the final apply step, so it still needs a valid API token. There's no secret-free way to run it.

Two safer options instead of exposing your production secrets broadly:

- Point fork PR dry runs at your staging instance's token instead of production's. A leaked staging token exposes less than a leaked production one.
- Require approval before secrets are exposed. Under **Settings > Environments**, enable [**Required reviewers**](https://docs.github.com/en/actions/reference/deployments-and-environments#required-reviewers) on the environment. A maintainer then has to approve the run before it can access that environment's secrets, even from a fork PR.

Only as a last resort, under [**Settings > Actions > General**](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository#enabling-workflows-for-forks-of-private-repositories) **> Fork pull request workflows from outside collaborators**, enable "Send secrets and variables to workflows from fork pull requests." Treat this the same as giving every fork contributor direct access to your secrets.

This setting exists at both the repository and organization level, and the organization-level setting can override the repo-level one. If it's disabled org-wide, enabling it on the repo alone won't be enough. An org owner will also need to check [**Organization settings > Actions > General**](https://docs.github.com/en/organizations/managing-organization-settings/disabling-or-limiting-github-actions-for-your-organization).

## Further reading

- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Preventing mistakes with GitOps](https://fleetdm.com/guides/preventing-mistakes-with-gitops)
- [GitHub docs: manage environments for deployment](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)
- [GitHub docs: using secrets in GitHub Actions](https://docs.github.com/en/actions/security-for-github-actions/security-guides/using-secrets-in-github-actions)
- [GitHub docs: security hardening for GitHub Actions](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions)
- [GitHub docs: about forks](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/about-forks)

<meta name="articleTitle" value="Set up Fleet GitOps for staging and production">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-31">
<meta name="description" value="How to structure your Fleet GitOps repo and secrets to safely test changes in staging before they reach production.">
