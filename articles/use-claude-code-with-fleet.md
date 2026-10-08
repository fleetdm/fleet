# Use Claude Code with Fleet

Claude Code is an AI coding agent you can use from the Claude Desktop app or your terminal. Point it at a Fleet GitOps repository and it can write policies, reports, and configuration profiles for you, checked against Fleet's schema and Apple, Microsoft, and Google references. This guide covers signing in to Claude Desktop and using Claude Code with Fleet, whether or not you manage Fleet with GitOps.

Support for Codex and GitHub Copilot is coming soon.

## Prerequisites

- A Claude account on a plan that includes Claude Code. See [Claude Code's setup docs](https://docs.claude.com/en/docs/claude-code/setup) for current options.
- [Claude Desktop](https://claude.ai/download) installed on your computer.
- To use GitOps: a repository created with `fleetctl new`. [Install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first if you haven't.

## Sign in to Claude Desktop

1. Open Claude Desktop.
2. Sign in with your Claude account.
3. Select the **Code** tab.

Prefer the terminal? You can [install Claude Code](https://docs.claude.com/en/docs/claude-code/setup) and run `claude` in any folder instead. The steps below work the same way.

## Use Claude Code in a GitOps repository

Repositories created with [`fleetctl new`](https://github.com/fleetdm/fleet/blob/main/cmd/fleetctl/fleetctl/templates/new/README.md) include a `CLAUDE.md` file and a `fleet-gitops` skill. `CLAUDE.md` tells Claude how the repository is laid out. The skill makes Claude validate what it writes before writing it.

1. In the **Code** tab, choose your GitOps repository's folder.
2. Describe what you want. For example: "Add a policy that checks FileVault is on for macOS hosts" or "Create a configuration profile that disables the camera on macOS."
3. Review the files Claude adds or changes under `platforms/`.
4. Commit your changes and open a pull request. Your CI runs a dry run on the pull request and shows whether it passed.

## Use Claude Code without GitOps

If you manage Fleet in the UI, you can still use the `fleet-gitops` skill to generate policies, reports, and configuration profiles, then add them to Fleet yourself.

1. In the **Code** tab, choose an empty folder, or create a new one.
2. Ask Claude: "Create `.claude/skills/fleet-gitops/SKILL.md` in this folder using the contents of https://github.com/fleetdm/fleet/blob/main/cmd/fleetctl/fleetctl/templates/new/.claude/skills/fleet-gitops/SKILL.md."
3. Run `/fleet-gitops` followed by what you want, for example: `/fleet-gitops Write a Windows policy that checks BitLocker is on.`
4. Copy the result into Fleet:
   - **Policies and reports:** Add a new policy or report in the Fleet UI and paste in the SQL query.
   - **Configuration profiles:** Upload the file as a custom OS setting in the Fleet UI.

> **Note:** The skill's instructions mention "this repository," but it works in any folder. The agent still validates against the same references.

## Troubleshoot

**Claude writes a query that fails in Fleet**

Ask Claude to check each table and column against the [Fleet osquery schema](https://fleetdm.com/tables) for the target platform, then try again.

**`/fleet-gitops` isn't found**

Make sure you chose the folder that contains `.claude/skills/fleet-gitops/SKILL.md`.

## Further reading

- [Build and validate configuration profiles with AI instead of a GUI](https://fleetdm.com/guides/build-configuration-profiles-with-ai): a deeper walkthrough for Apple, Windows, and Android profiles.
- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)

<meta name="articleTitle" value="Use Claude Code with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-07">
<meta name="description" value="Sign up for Claude Code and use it to generate Fleet policies, reports, and configuration profiles, with or without GitOps.">
