# Overview

This directory contains the `gitops-migrate` tool, designed leading up to the `4.74` Fleet release to automate necessary GitOps YAML transformations.

# 4.74 YAML Changes

The `4.74` release moves GitOps YAML keys: `self_service`, `categories`, `labels_exclude_any`, `labels_include_any` and `setup_experience` from the software files ([example](https://github.com/fleetdm/fleet/blob/c9a02741950f6510f9f1be48a2c19bc524417f70/cmd/fleetctl/gitops-migrate/testdata/mozilla-firefox.yml#L2-L9)) to the team files ([example](https://github.com/fleetdm/fleet/blob/c9a02741950f6510f9f1be48a2c19bc524417f70/it-and-security/teams/workstations.yml#L47-L70)).

# Installation

## Method 1: Download the Binary (Recommended)

Download the appropriate binary for your operating system and architecture:

| Operating System | Architecture | Download Link |
| ---------------- | ------------ | ------------- |
| `macos`          | `arm64`      | [TODO](TODO)  |
| `windows`        | `amd64`      | [TODO](TODO)  |
| `windows`        | `arm64`      | [TODO](TODO)  |
| `linux`          | `amd64`      | [TODO](TODO)  |
| `linux`          | `arm64`      | [TODO](TODO)  |

## Method 2: Go Install

[Install Go](https://go.dev/doc/install) and install `gitops-migrate` by running:

```shell
$ go install github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate@latest
```

You can verify the installation was successful by running `gitops-migrate usage` which should display the help text.

> [!NOTE]
> If the `go install` is successful but you're not able to run `gitops-migrate`, you may need to add your `GOBIN` directory to `PATH` in the way appropriate for your operating system ([Windows](https://www.architectryan.com/2018/03/17/add-to-the-path-on-windows-10/), [Mac](https://medium.com/@B-Treftz/macos-adding-a-directory-to-your-path-fe7f19edd2f7), [Linux](https://pimylifeup.com/ubuntu-add-to-path/)). The path to add can be found by running: `go env GOBIN`.

# Running the Migration

The migration will unfold in two primary steps: `format` and `migrate`.

> [!IMPORTANT]
> If your GitOps files are version-controlled (stored in GitHub or similar) it is recommended to perform these steps in order, opening a pull request, moving through your standard review process and merging that pull request **before** moving to the next step.

## Step 1: Format

### Overview

When manipulating YAML files with this tool, for reasons we won't get into here, the output will always alphabetize the keys. This means the following YAML file:
```yaml
a: []
c: []
b: []
```
Will become:
```yaml
a: []
b: []
c: []
```

This _can_ mean, if your GitOps files are version-controlled (stored in GitHub or similar), you could see a very large number of changed lines which might make it more difficult to spot the **actual** transformations.

Considering the above, we recommend running the `gitops-migrate` `format` command **before** performing the migration.

### Steps

Run the `gitops-migrate` tool, specifying the `format` command followed by the path to your GitOps YAML files.

**Linux/Mac:**
```bash
# If 'gitops-migrate' is in the current working directory.
$ ./gitops-migrate format ./fleet_gitops
# If 'gitops-migrate' is on PATH.
$ gitops-migrate format ./fleet_gitops
```

**Windows:**
```powershell
# If 'gitops-migrate' is in the current working directory.
PS> .\gitops-migrate format .\fleet_gitops
# If 'gitops-migrate' is on PATH.
PS> gitops-migrate format .\fleet_gitops
```

Your YAML files should now all be alphabetized!

> [!TIP]
> It's recommended to pause here, open a pull request for the formatting changes _only_, then move onto the next section once that pull request has been reviewed and merged.

## Step 2: Migrate

### Overview

Now we'll run the `gitops-migrate` `migrate` command which will:
- Perform a backup of your GitOps files, outputting an archive to your operating system's `TEMP` directory (**the path to this backup will be shown at the start of the output of the command, be sure to take note of it**).
- Migrate all YAML files in the provided directory (changes outlined [above](#474-yaml-changes)).

### Steps

Run the `gitops-migrate` tool, specifying the `migrate` command and the path to your GitOps YAML files.

**Linux/Mac**:
```bash
# If 'gitops-migrate' is in the current working directory.
$ ./gitops-migrate migrate ./gitops_files
# If 'gitops-migrate' is on PATH.
$ gitops-migrate migrate ./gitops_files
```

**Windows:**
```powershell
# If 'gitops-migrate' is in the current working directory.
PS> .\gitops-migrate migrate .\gitops_files
# If 'gitops-migrate' is on PATH.
PS> gitops-migrate migrate .\gitops_files
```

### Did it work?

**In the command output,** you should see messages like the following:
```shell
> Successfully applied transforms to team file.
┣━ [Team File]=>[it-and-security/teams/workstations.yml]
┗━ [Count]=>[39]
```

In cases where the _team_ file previously contained software packages which referenced software files containing the fields [described above](#474-yaml-changes), you can spot-check the results by confirming these fields are now present in the software packages array items, right alongside the `path` key(s).

**When looking at a `git diff`** you should see changes similar to the following:

**Software file (`slack.yml`):**
```diff
url: https://downloads.slack-edge.com/desktop-releases/linux/x64/4.41.105/slack-desktop-4.41.105-amd64.deb
- self_service: true
- categories:
-  - Productivity
-  - Communication
- labels_include_any:
-  - "Debian-based Linux hosts"
```

**Team File (`my_team.yml`):**
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

### Help, something has gone wrong!

In the event you've attempted the migration and encounter any issues, you can quickly revert your GitOps file states by simply restoring the backup taken automatically during the `migrate` process.

To do this, locate the backup archive path in the log output:

```bash
> Performing Fleet GitOps file backup.
┣━ [Source]=>[fleet_gitops]
┗━ [Destination]=>[/tmp/fleet-gitops-1916163188/fleet-gitops-backup-8-31-2025_4-47-29.tar.gz] # <-- Here
```

Then simply run the `gitops-migrate` `restore` command to restore this backup, specifying the **archive** path as the first arg and the path to restore the archive **to** as the second arg:

**Linux/Mac**:
```bash
# If 'gitops-migrate' is in the current working directory.
$ ./gitops-migrate restore /tmp/fleet-gitops-1916163188/fleet-gitops-backup-8-31-2025_4-47-29.tar.gz ./fleet_gitops
# If 'gitops-migrate' is on PATH.
$ gitops-migrate restore /tmp/fleet-gitops-1916163188/fleet-gitops-backup-8-31-2025_4-47-29.tar.gz ./fleet_gitops
```

**Windows:**
```powershell
# If 'gitops-migrate' is in the current working directory.
PS> .\gitops-migrate restore 'C:\Users\am\AppData\Local\Temp\fleet-gitops-1916163188/fleet-gitops-backup-8-31-2025_4-47-29.tar.gz' .\fleet_gitops
# If 'gitops-migrate' is on PATH.
PS> gitops-migrate restore 'C:\Users\am\AppData\Local\Temp\fleet-gitops-1916163188/fleet-gitops-backup-8-31-2025_4-47-29.tar.gz' .\fleet_gitops
```
