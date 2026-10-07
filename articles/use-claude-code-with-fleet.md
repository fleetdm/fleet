# Use Claude Code with Fleet

Claude Code is an AI coding agent that runs in your terminal. Point it at a Fleet GitOps repository and it can write policies, reports, and configuration profiles for you, checked against Fleet's schema and Apple, Microsoft, and Google references. This guide covers signing up, installing Claude Code, and using it with Fleet, whether or not you manage Fleet with GitOps.

Support for Codex and GitHub Copilot is coming soon.

## Prerequisites

- A Claude account on a plan that includes Claude Code, or an Anthropic Console account with API billing. See [Claude Code's setup docs](https://docs.claude.com/en/docs/claude-code/setup) for current options.
- A terminal on macOS, Linux, or Windows.
- To use GitOps: a repository created with `fleetctl new`. [Install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first if you haven't.

## Install Claude Code and sign in

1. Follow the [Claude Code setup docs](https://docs.claude.com/en/docs/claude-code/setup) to install it.
2. Open your terminal and run `claude`.
3. When prompted, sign in with your Claude account (or Console account).

## Use Claude Code in a GitOps repository

Repositories created with `fleetctl new` include a `CLAUDE.md` file and a `fleet-gitops` skill. `CLAUDE.md` tells Claude how the repository is laid out. The skill makes Claude validate what it writes before writing it.

1. In your terminal, go to your GitOps repository.
2. Run `claude`.
3. Describe what you want. For example: "Add a policy that checks FileVault is on for macOS hosts" or "Create a configuration profile that disables the camera on macOS."
4. Review the files Claude adds or changes under `platforms/`.
5. To dry-run the changes the way CI does, ask Claude to run `fleetctl gitops --dry-run` on your files. `CLAUDE.md` has the exact command.
6. Open a pull request. Merging deploys the change to Fleet.

> **Note:** A commit to the default branch is a deploy. Always read what Claude wrote before you merge.

## Use Claude Code without GitOps

If you manage Fleet in the UI, you can still use the `fleet-gitops` skill to generate policies, reports, and configuration profiles, then add them to Fleet yourself.

1. Create a folder on your computer, and in it create `.claude/skills/fleet-gitops/`.
2. Copy [`SKILL.md`](https://github.com/fleetdm/fleet/blob/main/cmd/fleetctl/fleetctl/templates/new/.claude/skills/fleet-gitops/SKILL.md) from the `fleetctl new` template into that folder.
3. In your terminal, go to the folder you created and run `claude`.
4. Run `/fleet-gitops` followed by what you want, for example: `/fleet-gitops Write a Windows policy that checks BitLocker is on.`
5. Copy the result into Fleet:
   - **Policies and reports:** Add a new policy or report in the Fleet UI and paste in the SQL query.
   - **Configuration profiles:** Upload the file as a custom OS setting in the Fleet UI.

> **Note:** The skill's instructions mention "this repository," but it works in any folder. The agent still validates against the same references.

## Troubleshoot

**Claude writes a query that fails in Fleet**

Ask Claude to check each table and column against the [Fleet osquery schema](https://fleetdm.com/tables) for the target platform, then try again.

**`/fleet-gitops` isn't found**

Make sure you started `claude` from the folder that contains `.claude/skills/fleet-gitops/SKILL.md`.

## Further reading

- [Build and validate configuration profiles with AI instead of a GUI](https://fleetdm.com/guides/build-configuration-profiles-with-ai): a deeper walkthrough for Apple, Windows, and Android profiles.
- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)

<meta name="articleTitle" value="Use Claude Code with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-07">
<meta name="description" value="Sign up for Claude Code and use it to generate Fleet policies, reports, and configuration profiles, with or without GitOps.">
