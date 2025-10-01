# Overview

This directory contains the `gitops-migrate` tool, designed leading up to the `4.74` Fleet release to automate necessary GitOps YAML transformations.

# 4.74 YAML Changes

The `4.74` release moves GitOps YAML keys: `self_service`, `categories`, `labels_exclude_any`, and `labels_include_any` from the software files ([example](https://github.com/fleetdm/fleet/blob/c9a02741950f6510f9f1be48a2c19bc524417f70/cmd/fleetctl/gitops-migrate/testdata/mozilla-firefox.yml#L2-L9)) to the team files ([example](https://github.com/fleetdm/fleet/blob/c9a02741950f6510f9f1be48a2c19bc524417f70/it-and-security/teams/workstations.yml#L47-L70)).

# Installation

## Download the Binary

1. Download the appropriate binary for your operating system and architecture:

| Operating System | Architecture | Download Link                                                                                                                                                       |
| ---------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `macos`          | `arm64`      | [Download](https://download.fleetdm.com/tools/gitops-migrate-darwin-arm64)([Hash](https://download.fleetdm.com/tools/gitops-migrate-darwin-arm64.sha256))           |
| `macos`          | `amd64`      | [Download](https://download.fleetdm.com/tools/gitops-migrate-darwin-amd64)([Hash](https://download.fleetdm.com/tools/gitops-migrate-darwin-amd64.sha256))           |
| `windows`        | `amd64`      | [Download](https://download.fleetdm.com/tools/gitops-migrate-windows-amd64.exe)([Hash](https://download.fleetdm.com/tools/gitops-migrate-windows-amd64.exe.sha256)) |
| `windows`        | `arm64`      | [Download](https://download.fleetdm.com/tools/gitops-migrate-windows-arm64.exe)([Hash](https://download.fleetdm.com/tools/gitops-migrate-windows-arm64.exe.sha256)) |
| `linux`          | `amd64`      | [Download](https://download.fleetdm.com/tools/gitops-migrate-linux-amd64)([Hash](https://download.fleetdm.com/tools/gitops-migrate-linux-amd64.sha256))             |
| `linux`          | `arm64`      | [Download](https://download.fleetdm.com/tools/gitops-migrate-linux-arm64)([Hash](https://download.fleetdm.com/tools/gitops-migrate-linux-arm64.sha256))             |

2. Rename the file `gitops-migrate` with no extension.

3. Move the file to the root of your Fleet GitOps directory. For example, the root of our GitOps directory is [/it-and-security](https://github.com/fleetdm/fleet/tree/main/it-and-security).

4. Open terminal and navigate to your Fleet GitOps directory. 

5. Make the binary file executable. For example, on Linux and macOS use `chmod +x gitops-migrate`.

6. Verify the installation was successful by running `./gitops-migrate usage` which should display the help text.

> If you're on macOS Tahoe (26) and you see a warning that Apple could not verify "gitops-migrate" is free of malware, select the Apple icon in the top left corner of the screen and select **Settings > Privacy & Security**. Then, next to "gitops-migrate" select **Allow Anyway** and run the migration tool again.

# Running the migration

When manipulating YAML files with this tool, the output will always alphabetize the keys and remove all comments.

The migration will unfold in two primary steps: `format` and `migrate`.

## Step 1: Format

1. Run `./gitops-migrate format ./`.
2. Commit the resulting changes to your repo. 

## Step 2: Migrate

1. Run `./gitops-migrate migrate ./`.

In the command output, you should see messages like the following:
```shell
> Successfully applied transforms to team file.
┣━ [Team File]=>[it-and-security/teams/workstations.yml]
┗━ [Count]=>[39]
```

2. Commit the resulting changes to your repo. 

## Confirm

In cases where the _team_ file previously contained software packages which referenced software files containing the fields [described above](#474-yaml-changes), you can spot-check the results by confirming these fields are now present in the software packages array items, right alongside the `path` key(s).

When looking at a `git diff` you should see changes similar to the following:

**Software file (`./slack.yml`):**
```diff
url: https://downloads.slack-edge.com/desktop-releases/linux/x64/4.41.105/slack-desktop-4.41.105-amd64.deb
- self_service: true
- categories:
-  - Productivity
-  - Communication
- labels_include_any:
-  - "Debian-based Linux hosts"
```

**Team File (`./my_team.yml`):**
```diff
software:
  packages:
    - path: slack.yml
+     self_service: true
+     categories:
+       - Productivity
+       - Communication
+     labels_include_any:
+       - "Debian-based Linux hosts"
```
