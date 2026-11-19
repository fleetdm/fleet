# Build DEX dashboards with GitHub Actions

The [DEX dashboards](https://fleetdm.com/guides/build-dex-dashboards-with-fleet-and-claude) turn Fleet data into eight digital employee experience dashboards and an overview. That guide runs them as a Claude Code routine and publishes Claude artifacts. This guide runs the same scripts in a GitHub Actions workflow instead, keeps each dashboard's history on a branch in your repository, and serves the pages with GitHub Pages. You don't need a Claude account, and the pages link to each other in the same browser tab.

## How it works

The workflow runs on a schedule in three jobs:

- **Build** reads the Fleet API with a read-only token, reads yesterday's histories from a `dex-data` branch, and builds the pages. This job can read the repository but can't change it.
- **Save histories** commits the updated histories to `dex-data`. This job never sees the Fleet token.
- **Deploy** publishes the pages to GitHub Pages.

Each dashboard has a history file. Patch velocity uses it to time how long each new version takes to reach every computer, and every page charts its key numbers over time from it, for up to a year. Losing the histories starts those charts over, so the workflow refuses to run without them unless you start it by hand as a first run.

## Prerequisites

- Everything in steps 1 and 2 of [Build DEX dashboards with Fleet and Claude](https://fleetdm.com/guides/build-dex-dashboards-with-fleet-and-claude): the DEX reports in Fleet, and a read-only API-only user limited to five endpoints. Its prerequisites list the Fleet and fleetd versions each dashboard needs.
- A GitHub organization on GitHub Enterprise Cloud, to keep the pages private. See the warning below before you use any other plan.
- A private repository in that organization, and admin access to it.
- A Fleet server that GitHub-hosted runners can reach over the internet. If your Fleet server only accepts connections from an allowlist of IP addresses, use a [self-hosted runner](https://docs.github.com/en/actions/hosting-your-own-runners) on your network instead.
- `git` on your computer.

> **Warning:** The dashboards show computer names, which apps each person opens, and their AI tools. Only GitHub Enterprise Cloud can [publish a Pages site privately](https://docs.github.com/en/enterprise-cloud@latest/pages/getting-started-with-github-pages/about-privately-published-github-pages-sites). On every other plan, a Pages site is public on the internet, even when its repository is private. Without Enterprise Cloud, skip GitHub Pages: remove the workflow's `deploy` job and upload the site to a web host behind your single sign-on instead (see "Host the pages somewhere else").

## Step 1: Add the scripts to your repository

1. Download a sparse copy of Fleet's repository with the scripts:

```bash
git clone --depth 1 --filter=blob:none --sparse https://github.com/fleetdm/fleet.git fleet-dex
```

```bash
git -C fleet-dex sparse-checkout set docs/solutions/dex-dashboards
```

2. In your repository, copy the folder to `dex-dashboards/`, and the workflow to `.github/workflows/`:

```bash
cp -R ../fleet-dex/docs/solutions/dex-dashboards dex-dashboards
```

```bash
mkdir -p .github/workflows && cp dex-dashboards/github-workflow.yml .github/workflows/dex-dashboards.yml
```

3. Review the scripts and the workflow, then commit and push them to `main`. The workflow runs the scripts every day with your Fleet token, so read them the way you'd read any code you give a credential to.

The workflow runs daily at 14:07 UTC. To change that, edit the `cron` line. Pick a schedule from "Choose how often it runs" in the Claude guide: the same fleet-size limits apply, because each run reads every computer from Fleet.

## Step 2: Create the data branch

The `dex-data` branch holds only the histories. It shares no commits with `main`. Create it once, from your repository's folder:

```bash
git switch --orphan dex-data
```

```bash
git commit --allow-empty -m "Start DEX dashboard histories"
```

```bash
git push origin dex-data
```

```bash
git switch main
```

## Step 3: Store the Fleet token in an environment

An [environment](https://docs.github.com/en/actions/managing-workflow-runs-and-deployments/managing-deployments/managing-environments-for-deployment) only hands its secrets to jobs that run from the branches you allow. The workflow's Build job uses one named `dex-dashboards`.

1. In your repository, go to **Settings > Environments** and click **New environment**. Name it `dex-dashboards`.
2. Under **Deployment branches and tags**, select **Selected branches and tags** and add `main`.
3. Under **Environment secrets**, add `FLEET_API_TOKEN` with the API-only user's token.
4. Under **Environment variables**, add `FLEET_URL` with your Fleet server's address, for example `https://fleet.example.com`.
5. Optional: add any of these environment variables. Each one works the same as in the Claude guide.
   - `DEX_WARRANTY_VITAL`: the custom host vital that holds warranty end dates. The default is `AppleCare warranty expiration`.
   - `DEX_GRID_INTENSITY`: grams of CO2 per kWh, for computers with no GeoIP country.
   - `DEX_COMPANY_DOMAINS`: comma-separated email domains that count as company sign-ins for coding agents.
   - `DEX_WORKERS`: how many requests go to Fleet at once. The default is 8. Lower it if your Fleet server is busy during runs.

> **Note:** Don't add required reviewers to this environment. Every scheduled run would wait for someone to approve it.

## Step 4: Turn on GitHub Pages

1. Go to **Settings > Pages**.
2. Under **Build and deployment**, set **Source** to **GitHub Actions**.
3. Under **Visibility**, select **Private**. Only people with read access to the repository can open a private site.

GitHub creates a `github-pages` environment the first time the workflow deploys. It only accepts deployments from `main` by default. Keep it that way.

## Step 5: Run the workflow for the first time

1. Go to **Actions**, select **DEX dashboards**, and click **Run workflow**.
2. Select **First run: start new histories**, and click **Run workflow**.
3. When the run finishes, open the **Deploy** job's summary for the site's address.

A first run is the only time the workflow starts without histories. Scheduled runs always read them from `dex-data`, and fail if any page's history is missing, so a lost branch can't silently restart the trend charts. A first run fails if `dex-data` already has histories.

## Secure the setup

The workflow already splits its permissions: only Build sees the Fleet token, and only Save histories can push. It also pins each action to a commit SHA and turns off every permission a job doesn't need. Lock down the repository around it:

- **Keep the repository private, and its access tight.** Anyone with read access can open the dashboards and read the histories on `dex-data`, which include per-computer details. Give read access only to the people who should see the dashboards.
- **Protect `main`.** Anyone who can change the workflow on `main` can read the Fleet token. Add a [ruleset](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) for `main` that requires a pull request with an approving review. Add a [CODEOWNERS](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners) entry for `.github/workflows/` and `dex-dashboards/`, and require code owner review.
- **Protect `dex-data`.** Add a ruleset for `dex-data` that blocks force pushes and deletions, so nobody rewrites or deletes the histories by accident.
- **Allow only GitHub's actions.** The workflow uses only actions from the `actions` organization. Under **Settings > Actions > General**, select **Allow \<owner\>, and select non-\<owner\>, actions and reusable workflows**, then select **Allow actions created by GitHub**. Under **Workflow permissions**, select **Read repository contents and packages permissions**. The workflow asks for more where a job needs it.
- **Keep the Fleet token read-only.** Use the API-only user from the Claude guide: the global Observer role, limited to five read endpoints.
- **Rotate the token.** To replace it, create a new API-only user the same way, update `FLEET_API_TOKEN`, run the workflow once, then delete the old user in Fleet.
- **Review updates before you copy them.** When the scripts change in Fleet's repository, read the diff before you copy them into yours. To keep the pinned actions current, turn on [Dependabot version updates](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/keeping-your-actions-up-to-date-with-dependabot) for the `github-actions` ecosystem.

## Know the storage limits

These limits come from GitHub. The dashboards stay well inside them on most fleets, but a large fleet run often for a long time can reach the repository size limits.

| What | Limit | How the dashboards fit |
|:--|:--|:--|
| [Pages site size](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits) | 1 GB | A few MB on a small fleet. The pages carry today's data only, with no histories. |
| Pages bandwidth | 100 GB a month, soft limit | Only a concern if many people open the pages every day on a large fleet. |
| Pages deployment | Times out after 10 minutes | Deploying a few MB takes well under a minute. |
| [File size in Git](https://docs.github.com/en/repositories/working-with-files/managing-large-files/about-large-files-on-github) | Warning at 50 MB, blocked at 100 MB | The scripts split any history over about 8 MB into part files. |
| Repository size | Under 1 GB recommended, under 5 GB strongly recommended | Each run commits the day's histories to `dex-data`, so the branch grows every run. Each history keeps up to a year of data. |
| [Actions artifacts](https://docs.github.com/en/actions/reference/limits) | 500 MB on Free, 1 GB on Pro, 2 GB on Team | The workflow keeps its artifacts for one day. |
| Job time | 6 hours on GitHub-hosted runners | The Build job stops after 2 hours. |
| Minutes | Private repositories use the plan's included Actions usage | A run takes minutes on a small fleet, and longer on a large one. Running every 4 hours uses six times as much as once a day. |

To check the repository's size, in KB:

```bash
gh api repos/<owner>/<repo> --jq .size
```

If it nears 1 GB, replace `dex-data`'s history with one commit of its current files. Turn off the ruleset's force push block first, and turn it back on after:

```bash
git fetch origin dex-data && git switch -C dex-data FETCH_HEAD
```

```bash
git reset --soft "$(git commit-tree 'HEAD^{tree}' -m 'Squash DEX dashboard histories')"
```

```bash
git push --force origin dex-data
```

The dashboards keep all their history, since it lives in the files. You lose the older commits, which only matter for restoring an earlier day's files.

> **Note:** GitHub runs scheduled workflows in UTC. A run can start late when GitHub is busy, and under heavy load GitHub may drop a queued run. Each trend chart keeps one value per UTC day, so a late run still counts.

## Verify

1. Open the site. The overview shows each dashboard's numbers for today. Click any card, or any tab in the bar across the top, to open that dashboard in the same tab.
2. Check that each dashboard shows today's date next to **Updated**.
3. Open the `dex-data` branch. It has a `history` folder with one folder per dashboard.
4. Open the site in a private browser window. GitHub should ask you to sign in before it shows the page. If it doesn't, the site is public: turn off GitHub Pages under **Settings > Pages** until you fix its visibility.

## Host the pages somewhere else

To serve the pages from your own web host, remove the `deploy` job and the **Upload the site** step, and add a step to the Build job that uploads `${{ runner.temp }}/site` with your host's tool. For example, for an Amazon S3 bucket behind CloudFront, after configuring credentials with [`aws-actions/configure-aws-credentials`](https://github.com/aws-actions/configure-aws-credentials):

```bash
aws s3 sync "$RUNNER_TEMP/site" s3://<bucket>/dex --delete
```

Put the site behind your single sign-on, and give the upload credential access to that one site and nothing else. If you add an action from outside GitHub, allow it in **Settings > Actions > General**, and pin it to a commit SHA.

## What works differently from the Claude version

- **AI governance policy decisions don't save.** The page shows the inventory and risk findings, but nobody can record a posture or decision, and there's no change log. Those need the artifact's shared store.
- **Theme choices stay in each browser.** Each viewer's Light, Dark, or System choice is saved in their browser only.
- **One site per repository.** GitHub Pages serves one site from each repository. To build dashboards for more than one Fleet server, for example a test server and production, use one repository for each.

## Troubleshoot

**"FATAL: history for \<page\> is missing"**

A scheduled run couldn't find a page's history on `dex-data`. If this is the first run, start it by hand with **First run: start new histories**. Otherwise, check the `dex-data` branch's commits for a change that removed the file, and restore it.

**"\<data branch\> already has histories. A first run would start them over."**

You started a first run, but `dex-data` already has histories. Run the workflow again without **First run: start new histories**.

**"FATAL: FLEET_URL and FLEET_API_TOKEN must be set"**

The Build job couldn't read the environment. Check that the environment is named `dex-dashboards`, that it allows `main`, and that the workflow ran from `main`.

**The Build job fails to check out `dex-data`**

The branch doesn't exist. Create it in step 2.

**The Deploy or Build job says the branch isn't allowed to deploy to the environment**

The workflow ran from a branch other than `main`, and the environment's deployment branch rule stopped it. Run it from `main`.

**Patch velocity says "Couldn't check the latest release"**

The runner couldn't reach `sofafeed.macadmins.io` or `api.msrc.microsoft.com`. GitHub-hosted runners reach them by default, so this usually means a firewall in front of a self-hosted runner. Allow those hosts and `updates.fleetdm.com`.

## Further reading

- [Build DEX dashboards with Fleet and Claude](https://fleetdm.com/guides/build-dex-dashboards-with-fleet-and-claude)
- [The DEX dashboard scripts](https://github.com/fleetdm/fleet/tree/main/docs/solutions/dex-dashboards)
- [About privately published GitHub Pages sites](https://docs.github.com/en/enterprise-cloud@latest/pages/getting-started-with-github-pages/about-privately-published-github-pages-sites)
- [Using custom workflows with GitHub Pages](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages)

<meta name="articleTitle" value="Build DEX dashboards with GitHub Actions">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-11-19">
<meta name="category" value="guides">
<meta name="description" value="Build Fleet's DEX dashboards in a GitHub Actions workflow, keep their history in your repository, and serve them privately with GitHub Pages.">
