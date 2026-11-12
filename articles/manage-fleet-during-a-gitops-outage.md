# Manage Fleet during a GitOps outage

If your CI provider (like GitHub Actions) goes down, you can still make urgent changes to Fleet through the UI. This guide covers how to do that safely, and how to bring your git repository back in sync once CI is available again.

## Prerequisites

Check these before you start:

- Fleet Premium, with [GitOps mode](https://fleetdm.com/guides/gitops-mode) enabled
- Admin access to the Fleet UI

> **Warning:** GitOps is your source of truth. Any change you make in the UI during the outage will be deleted or overwritten the next time GitOps runs, unless you also commit that change to your repository. Treat every UI change as temporary until it's mirrored in git.

## Turn off GitOps mode

1. Go to **Settings > Integrations > Change management**.
2. Turn off **GitOps mode**.

This unlocks the UI sections that GitOps mode normally makes read-only, so you can make the change you need.

## Make the change in the Fleet UI

Make only the change required to resolve the urgent issue. Since this change isn't in your repository yet, GitOps will revert it on its next run unless you also make it in git.

## Mirror the change in your repository

As soon as you can, commit the same change to your GitOps repository. This keeps your repository and your live Fleet instance in sync, so the next GitOps run doesn't undo the fix you just made.

> **Note:** If CI is still down, you can commit the change to your repository now and run `fleetctl gitops` manually once you're able to, or wait until CI recovers.

## Verify GitOps is applying successfully

Before you turn GitOps mode back on, confirm that a GitOps run has completed successfully with your mirrored change, either through CI or by running `fleetctl gitops` manually. Check the run's output for errors, and confirm the change is still in place in the Fleet UI.

## Turn GitOps mode back on

Once you've confirmed GitOps is applying successfully, go back to **Settings > Integrations > Change management** and turn **GitOps mode** on.

## Further reading

- [GitOps mode](https://fleetdm.com/guides/gitops-mode)
- [Preventing mistakes with GitOps](https://fleetdm.com/guides/preventing-mistakes-with-gitops)
- [YAML file reference](https://fleetdm.com/docs/configuration/yaml-files)

<meta name="articleTitle" value="Manage Fleet during a GitOps outage">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-08-06">
<meta name="category" value="guides">
<meta name="description" value="What to do when your CI provider is down and you need to make an urgent change to Fleet outside of GitOps.">
