# Migrating to GitOps using `fleetctl`

## Introduction

At Fleet, we are strong proponents of using [GitOps](https://fleetdm.com/guides/sysadmin-diaries-gitops-a-strategic-advantage#basic-article) to manage your configuration (you can read more about our rationale [here](https://fleetdm.com/guides/articles/preventing-mistakes-with-gitops)). But what if you already have a Fleet instance with complex configuration or a large numbers of labels, policies, queries or software installers? How can you migrate your configuration management to GitOps while ensuring that nothing is lost in the shuffle?

Enter `fleetctl generate-gitops`.

## What is `generate-gitops`?

The `generate-gitops` command is a migration tool that takes your existing Fleet configuration and transforms it into a series of GitOps-ready files. The format and layout of the files reflects our best-practice recommendations for using GitOps.

## Basic usage

> First ensure that [you have fleetctl installed](https://fleetdm.com/guides/fleetctl) and have logged in via `fleetctl login`.

To generate a new set of GitOps files reflecting your current configuration, open a terminal and run:

`fleetctl generate-gitops --dir /path/to/your/desired/gitops/folder`

If the specified folder already exists, it must be empty, or else the command will exit for safety. If you are sure you'd like to generate your GitOps files in a non-empty folder, you may use the `--force` option:

`fleetctl generate-gitops --dir /path/to/your/desired/gitops/folder --force`

The `--force` option may come in handy if you've already initialized a Git repo in the chosen folder.

## Handling sensitive information

It is generally not recommended to store sensitive information such as Fleet enrollment secrets directly in a version control framework like Git, even when using a private repository on a provider like GitLab or GitHub. By default, the `generate-gitops` command will leave comments in place of sensitive items, and display a list of filenames and keys that will need to be updated manually before the files are ready to be used with GitOps. A typical strategy for dealing with these items is to store their contents in environment variables or "secrets" on a version control provider, and then refer to the variable within your GitOps file. For example:

```yaml
- secrets:
    - secret: $TEAM_ENROLLMENT_SECRET
```

To have `generate-gitops` output sensitive info in plaintext in your files, you may use the `--insecure` option. Caveat emptor!

## Other options

The `generate-gitops` tool includes a few other options to make migrating to GitOps easier:

- `--print` : Print the configuration to `stdout` rather than to files.
- `--team` : **Available in Fleet Premium.** Only output the configuration files of the team with the specified name. Global or "no team" configuration may be output using `--team global` or `--team no-team`. (This option can be useful for testing out GitOps with a "canary" team before rolling it out to your entire organization.)
- `--key` : Display the value of a specific, dot-delimited key, e.g. `agent_options.config.decorators`. Searches for the given key in the global configuration by default; use in conjunction with `--team` to output config from a specific team.

See `fleetctl generate-gitops --help` for all options.

## Known issues

- GitOps cannot currently sync Fleet-maintained app installers. If your current configuration includes FMA-based installers, the migration tool will output a placeholder for them which will cause GitOps to fail (ensuring that your current configuration is not overwritten).
- The migration tool does not output YARA rules at this time. If you have previously used GitOps to apply YARA rules, you will need to manually add them to any output from the tool to ensure that your existing rules are maintained.
- The migration tool does not output the `macos_settings` key configuration at this time. If you have customized configuration for Mac hosts such as a bootstrap package or script, the tool will output a placeholder for you to replace with the correct details. See [the GitOps reference](https://fleetdm.com/docs/configuration/yaml-files#macos-setup) for more information on `macos_settings`.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="sgress454">
<meta name="authorFullName" value="Scott Gress">
<meta name="publishedOn" value="2025-05-22">
<meta name="articleTitle" value="Migrating to GitOps using fleetctl">
<meta name="description" value="Instructions for migrating your Fleet configuration to GitOps.">