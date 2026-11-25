# How to run Fleet GitOps at enterprise scale

*A GitOps repo that works for one admin and fifty hosts needs more structure once a dozen engineers, five teams, and fifty thousand hosts depend on it. Here's how to lay it out and protect it.*

## Key takeaways

- **One repo, organized by fleet, scales further than you'd expect.** A file per fleet and shared folders per platform let several teams work side by side, and Fleet runs its own devices this way.

- **Exactly one repo should own org-wide settings.** Anything missing from your YAML gets reset, and Fleet's starter workflow deletes fleets it doesn't see by default. If configuration ever spans more than one repo, decide up front which repo is authoritative for what.

- **CODEOWNERS turns a directory structure into an ownership model.** Map each fleet file and platform folder to the team responsible for it, and let your security team own policies everywhere, so the right people review every change automatically.

- **Branch protection makes review mandatory, not polite.** Require pull requests, approvals from code owners, and fresh approvals after new commits, and block direct and force pushes to your default branch.

- **A dry run that must pass catches mistakes before any device sees them.** Mark the `fleetctl gitops --dry-run` check as required, keep branches up to date before merge, and a broken profile or invalid policy can't reach production.

- **Least-privilege tokens and GitOps mode close the side doors.** A dedicated API-only GitOps user per pipeline limits the blast radius of a leaked token, and GitOps mode stops anyone from quietly undoing a reviewed change in the UI.

<a purpose="cta-button" href="https://fleetdm.com/docs/configuration/yaml-files">See the GitOps reference</a>

Fleet's [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files) and the `fleetctl new` starter repo are built around a single team managing a single Fleet instance. That's the right place to start, and it's not where most enterprises stay.

Once a second team starts managing its own devices, a security team wants a say in compliance policies, and a separate infrastructure group shows up with Linux servers, the question changes. It's no longer "how do I write this YAML." It's "how do several teams change the same source of truth safely, with the right reviewers, and without any one person able to push a mistake to every device." That's a repository and process problem, and Git already has good answers for it.

## Start from one repo, organized by fleet

`fleetctl new` scaffolds a repo with `default.yml` for org-wide settings, one file per fleet under `fleets/`, shared `labels/`, and a folder per platform and artifact type under `platforms/` (`policies/`, `reports/`, `software/`, `scripts/`, `configuration-profiles/`, and so on). Each fleet file pulls from those folders with `paths:` glob patterns like `../platforms/macos/configuration-profiles/*.mobileconfig`, so a policy, script, or profile is defined once and reused by every fleet that references it.

```
fleet-gitops/
├── default.yml
├── fleets/
│   ├── workstations.yml
│   ├── servers.yml
│   └── testing-and-qa.yml
├── labels/
├── platforms/
│   ├── macos/
│   ├── windows/
│   └── linux/
└── .github/
    ├── CODEOWNERS
    └── workflows/
```

This layout holds up at scale. Fleet manages its own laptops, mobile devices, and servers from a single repo, with a fleet file apiece for workstations, servers, testing and QA, and company-owned and personal mobile devices, all drawing from shared folders. Allen Houchins, who leads IT at Fleet, explains how that setup evolved in [What I have learned from managing devices with GitOps](https://fleetdm.com/guides/what-i-have-learned-from-managing-devices-with-gitops).

The pattern generalizes: one fleet file per group of hosts a team owns, and shared folders for everything that isn't specific to one fleet. When a CVE requires a policy change, you fix it in one file and every fleet that references it picks up the change on the next merge. You don't need a repo per team to separate ownership. You need a directory structure that reflects it, and the governance described below.

## Decide which repo owns what

Fleet GitOps is declarative. In the words of the reference docs, "any settings not defined in your YAML files will be reset to the default values or deleted." That's what makes GitOps trustworthy, and it's also why ownership has to be unambiguous before a second repo or pipeline enters the picture.

Two rules keep multi-team setups out of trouble:

First, only one repo holds `default.yml`. Org-wide settings, global labels, and integrations are defined in a single place. Two pipelines applying different versions of org-wide settings to the same Fleet instance will overwrite each other on every run.

Second, know what `--delete-other-fleets` does before you split. `fleetctl gitops` on its own only touches the fleets in the files you pass it with `-f`. The GitHub Action and GitLab pipeline that `fleetctl new` generates, however, pass `--delete-other-fleets` by default whenever the repo has a `default.yml`, which deletes any fleet in Fleet that isn't defined in that repo. In a single repo that's exactly what you want: removing a fleet file removes the fleet. If some fleets live in a second repo, set the action's `delete-other-fleets` input to `false` in the repo that holds `default.yml`, or every apply will delete the other repo's fleets. The starter script already turns deletion off in repos that contain only fleet files.

For most organizations, one repo with clear CODEOWNERS is simpler and safer than several repos. Reach for separate repos when a group needs genuinely separate administration, such as a business unit with its own Git organization, and give each of those repos only its own fleet files and a fleet-scoped token.

## Route every change to the right reviewers with CODEOWNERS

A `CODEOWNERS` file maps paths to the people or teams responsible for them. GitHub and GitLab both request those owners as reviewers automatically on any pull request that touches their paths, and branch protection (next section) can require their approval before merge. Fleet's own repo uses this for its `it-and-security` directory, which holds the GitOps configuration for Fleet's own devices.

In a multi-team repo, split ownership along the same lines as your directory structure:

```
# Fallback owner for anything not matched below
*                             @acme/endpoint-platform

# Org-wide settings, CI, and this file
/default.yml                  @acme/endpoint-platform
/.github/                     @acme/endpoint-platform

# Fleets
/fleets/workstations.yml      @acme/client-engineering
/fleets/testing-and-qa.yml    @acme/client-engineering
/fleets/servers.yml           @acme/infrastructure

# Platforms
/platforms/macos/             @acme/mac-engineering
/platforms/windows/           @acme/windows-engineering
/platforms/linux/             @acme/infrastructure

# Security reviews compliance policies on every platform
/platforms/*/policies/        @acme/security
```

A few details make the difference between CODEOWNERS that works and CODEOWNERS that quietly doesn't:

- The last matching pattern wins. Put broad rules first and specific ones after. In the example above, macOS policies belong to `@acme/security`, not `@acme/mac-engineering`, because the policies rule comes later.
- Multiple owners on one line means any one of them can approve. If both the platform team and security need to sign off on a policy, CODEOWNERS alone won't enforce that. Add the second check as a required reviewer team or a team agreement.
- Own the guardrails themselves. Assign `/.github/` (which contains `CODEOWNERS` and your workflow files) to a small platform team, so nobody can loosen review requirements or edit the pipeline without that team's approval.
- Owners need write access. On GitHub, a team or user listed in CODEOWNERS must have write permission to the repo, or their approval won't count.

On GitLab, code owner approvals on protected branches require GitLab Premium or Ultimate.

## Protect the default branch

CODEOWNERS decides who should review. Branch protection makes the review mandatory. On GitHub, configure a branch protection rule or ruleset for `main` (**Settings** > **Branches** or **Settings** > **Rules** > **Rulesets**) with at least these settings:

- **Require a pull request before merging**, with **Require approvals** set to at least one. Many enterprises use two for production repos.
- **Require review from Code Owners**, so each changed file needs approval from its owning team.
- **Dismiss stale pull request approvals when new commits are pushed**, so an approval covers the code that's merging.
- **Require approval of the most recent reviewable push**, so the person who pushed the last change can't be the one who approves it.
- **Require status checks to pass before merging** and **Require branches to be up to date before merging**, covered in the next section.
- Leave **Allow force pushes** and **Allow deletions** off, and turn on **Do not allow bypassing the above settings** so admins follow the same rules. If you need an emergency path, grant bypass to a small, named group and audit its use.

GitLab offers the same controls through protected branches, merge request approval rules, and the **Pipelines must succeed** merge check.

The point isn't bureaucracy. Every change to your devices now has an author, at least one reviewer who owns the affected configuration, and a permanent record in Git history. When something does go wrong, [reverting a bad change](https://fleetdm.com/guides/revert-a-bad-change-in-gitops) is a pull request away.

## Make the dry run a required check

Review catches intent problems. A dry run catches everything else: invalid YAML, a profile Fleet will reject, a policy referencing a label that doesn't exist. The workflow `fleetctl new` generates already runs `fleetctl gitops --dry-run` on every pull request and runs a real apply only when changes merge to `main`. What it can't do on its own is block a merge.

To make the dry run a gate, add its check (the generated GitHub workflow's job is named `fleet-gitops`) under **Require status checks to pass before merging**. Then a failing dry run blocks the merge button instead of showing a red X someone can click past.

A few details matter here:

- Require branches to be up to date. A dry run validates the pull request against the repo as it was when the check ran. If another pull request merges first, the combined result may differ. Requiring an up-to-date branch reruns the check against what will merge.
- Don't add path filters to the required workflow. On GitHub, a required check that's skipped because of a `paths:` filter stays pending and blocks the merge. The generated workflow runs on every change for this reason.
- The dry run needs a real token. `fleetctl gitops --dry-run` authenticates to Fleet and validates against its current state. It only skips the final apply. Plan your secrets accordingly, especially for pull requests from forks.
- Keep runs from overlapping. The generated GitHub workflow uses a `concurrency` group, and the GitLab pipeline uses a `resource_group`, so two applies never race each other. Keep that in place as you customize the workflow.
- Let the nightly run catch drift. The generated GitHub workflow also reapplies `main` on a nightly schedule, which reverts any change made outside the repo.

## Give automation least privilege

The token your pipeline uses can change every device in its scope, so treat it accordingly. Fleet has a dedicated `gitops` role for this. A user with the `gitops` role must be API-only, so it can apply configuration but can't sign in to the Fleet UI. Create one with `fleetctl user create --api-only`, then assign the role in Fleet.

On any Fleet tier, the `gitops` role can be granted globally. On Fleet Premium, it can also be scoped to a single fleet, so a pipeline that only manages `fleets/servers.yml` can't reconfigure workstations even if its workflow is misconfigured or its token leaks. Give each pipeline its own token rather than sharing one global token across every team.

Store tokens as CI secrets, never in the repo. Reference other credentials in your YAML as environment variables, which `fleetctl` fills in from CI secrets at run time, so they never appear in Git history. On GitHub, scoping secrets to a [deployment environment](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments) with required reviewers adds one more checkpoint: the production apply waits for an approver before it can read the token.

## Keep teams and environments as separate concerns

It's tempting to solve "multiple teams" and "staging versus production" with the same directory split. Resist that. Team ownership answers who reviews and merges a change. Environment answers where that change lands before it reaches every device. [Set up Fleet GitOps for staging and production](https://fleetdm.com/guides/set-up-gitops-for-staging-and-production) walks through the two approaches that work, separate repos or separate branches, and how to scope secrets for each.

Your team structure (fleet files, shared folders, and CODEOWNERS) stays the same whichever environment approach you choose.

Inside a single instance, a test fleet gives you a lightweight canary. Fleet's own repo has a `testing-and-qa` fleet for exactly this. Roll out a new profile or software package by referencing it from the test fleet's file first, confirm it behaves on real devices, then add it to production fleets in a follow-up pull request.

## Make the repo the only way in

All of these guardrails assume changes go through Git. [GitOps mode](https://fleetdm.com/guides/gitops-mode) enforces that. Set `gitops_mode_enabled: true` and a `repository_url` under `org_settings.gitops` in `default.yml`, and Fleet's UI shows every GitOps-managed section as read-only, with a tooltip pointing people to your repository. GitOps mode requires Fleet Premium.

Without it, an admin with UI access can undo a change that three reviewers approved, and nothing in the repo shows it happened until the next apply overwrites it. With it, the pull request is the only way in.

## Structure first, then scale

None of this requires exotic tooling. It's a directory layout, a CODEOWNERS file, a few branch protection settings, and a workflow Fleet already generates for you. The difference at enterprise scale is whether those guardrails exist before ten engineers start merging changes that reach every device, or get added after the first incident shows why they matter.

## See it live

Read [Set up Fleet GitOps for staging and production](https://fleetdm.com/guides/set-up-gitops-for-staging-and-production) to add environment separation on top of this structure.

- **Get a demo** of Fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Join a free GitOps workshop**: [fleetdm.com/gitops-workshop](https://fleetdm.com/gitops-workshop)

<meta name="articleTitle" value="How to run Fleet GitOps at enterprise scale">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-11-25">
<meta name="description" value="Repo layout, CODEOWNERS, branch protection, and required dry runs for running Fleet GitOps across many teams and fleets.">
