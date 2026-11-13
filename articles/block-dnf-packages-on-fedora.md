# Block dnf packages on Fedora

Users with `sudo` rights can install any package from Fedora's repositories with dnf. Use Fleet to block packages you don't trust, remove ones that are already installed, and keep the block in place.

## Prerequisites

- Hosts enrolled in Fleet.
- Scripts enabled. Linux hosts don't use Fleet's MDM features, so pass the `--enable-scripts` flag when you generate the `fleetd` package.
- Fleet Premium, to run the scripts automatically when a host fails a policy.
- dnf5, the default on Fedora 41 and later. The blocking script uses a dnf5 settings folder.

## How dnf blocks packages

dnf blocks packages listed in its `excludepkgs` setting. The blocking script adds the setting in its own file, so it doesn't change `/etc/dnf/dnf.conf`:

```ini
[main]
excludepkgs=sshpass,telnet
```

An excluded package can't be installed, reinstalled, or downloaded with dnf. Any package that depends on it can't be installed either.

> **Note:** This is a guardrail, not a hard control. It stops installs through dnf. Users with `sudo` rights can still turn off excludes for one command with `--setopt=disable_excludes='*'`, install an `.rpm` file with `rpm`, or edit the blocking file. Users without `sudo` rights can't install packages with dnf at all.

## Set the package list

Store the blocked packages in a [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles), so you don't need to edit the scripts when the list changes.

1. In Fleet, go to **Controls > Variables** and select **Add variable**.
2. Add `DNF_BLOCKED_PACKAGES` with the packages to block, like `sshpass telnet`.

Separate names with spaces. The scripts convert the list to the comma-separated format dnf uses. To look up a package's exact name, run `dnf search <keyword>`. To see what's already installed on your hosts, run the report in [Verify](#verify).

Scripts reference the variable as `$FLEET_SECRET_DNF_BLOCKED_PACKAGES`. Fleet fills in the value when it sends a script to a host.

> **Note:** Custom variables apply to all fleets, and Fleet hides their values in the UI. To block different packages on different fleets, create a variable for each list, like `DNF_BLOCKED_PACKAGES_SERVERS`, and a copy of each script that uses it.

## Block packages

1. Save the following as `dnf-blocked-packages.sh`.

```bash
#!/bin/bash

read -ra PACKAGES <<< "$FLEET_SECRET_DNF_BLOCKED_PACKAGES"

CONF_DIR=/etc/dnf/libdnf5.conf.d
CONF_FILE=$CONF_DIR/90-fleet-blocked.conf
LIST_FILE=/etc/dnf/blocked-packages.txt

mkdir -p "$CONF_DIR"

LIST=$(IFS=,; echo "${PACKAGES[*]}")
printf '[main]\nexcludepkgs=%s\n' "$LIST" > "$CONF_FILE"
printf '%s\n' "${PACKAGES[@]}" > "$LIST_FILE"

chown root:root "$CONF_FILE" "$LIST_FILE"
chmod 644 "$CONF_FILE" "$LIST_FILE"
```

> **Note:** Fleet replaces `$FLEET_SECRET_DNF_BLOCKED_PACKAGES` before the script reaches the host. To test the script outside Fleet, replace the variable with package names first.

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. To run the script on a host, open the host's page and select **Actions > Run script**. To run it automatically on hosts that fail a policy, see [Run automatically with policies](#run-automatically-with-policies).

The script writes the `excludepkgs` setting to `/etc/dnf/libdnf5.conf.d/90-fleet-blocked.conf`. It also writes the package names to `/etc/dnf/blocked-packages.txt`, one per line, for the [policy](#run-automatically-with-policies) that checks for blocked packages. Both files are owned by root, so only users with `sudo` rights can change them.

Blocking a package doesn't remove copies that are already installed. They keep working, but dnf won't reinstall them. To remove them, see [Remove installed packages](#remove-installed-packages).

## Remove installed packages

When dnf removes a package, it also removes every package that depends on it. To avoid removing other software, the script first checks which installed packages need each blocked package and skips any that are still needed.

1. Save the following as `dnf-package-removal.sh`. It uses the same variable as the blocking script.

```bash
#!/bin/bash

read -ra PACKAGES <<< "$FLEET_SECRET_DNF_BLOCKED_PACKAGES"

DNF_OPTS=(--setopt=disable_excludes='*')
STATUS=0

for pkg in "${PACKAGES[@]}"; do
  if ! rpm -q "$pkg" >/dev/null 2>&1; then
    echo "Not installed: $pkg"
    continue
  fi

  OTHERS=$(dnf "${DNF_OPTS[@]}" repoquery --installed --whatrequires "$pkg" --qf '%{name}\n' 2>/dev/null \
    | grep -v -e '^$' -e "^${pkg}$" | sort -u | tr '\n' ' ')
  if [ -n "$OTHERS" ]; then
    echo "Skipped: $pkg is required by: $OTHERS"
    STATUS=1
    continue
  fi

  if dnf "${DNF_OPTS[@]}" -y remove "$pkg"; then
    echo "Removed: $pkg"
  else
    echo "Failed: $pkg"
    STATUS=1
  fi
done

exit $STATUS
```

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. To run the script on a host, open the host's page and select **Actions > Run script**. To run it automatically on hosts that fail a policy, see [Run automatically with policies](#run-automatically-with-policies).

Excluded packages are also hidden from `dnf remove`, so the script turns off excludes for its own dnf commands with `--setopt=disable_excludes='*'`. The block stays in place for everyone else.

If other installed packages need a blocked package, the script skips it and lists them, like `Skipped: oniguruma is required by: bat jq`. The script run then shows as failed in Fleet.

> **Note:** When the script skips a package, it exits with an error, so the policy automation retries it, up to 3 attempts in total. If the package is still needed after that, the host keeps failing the policy until you remove or block the packages that depend on it.

## Run automatically with policies

Add two policies that run the scripts when a host fails them. The first puts the blocking files back if either is missing. The second removes blocked packages, including ones installed before you added the block. It reads the names from `/etc/dnf/blocked-packages.txt`, so you don't need to edit it when the list changes.

1. In Fleet, go to **Policies**, select your fleet, and select **Add policy**.
2. Enter the following query:

```sql
SELECT 1 WHERE (
  SELECT COUNT(*) FROM file
  WHERE path IN ('/etc/dnf/libdnf5.conf.d/90-fleet-blocked.conf', '/etc/dnf/blocked-packages.txt')
) = 2;
```

3. Name the policy `Linux - dnf blocklist is present` and set the platform to Linux.
4. Set the automation to run `dnf-blocked-packages.sh`, then save the policy.
5. Repeat steps 1-4 for the second policy. Name it `Linux - Blocked dnf packages are not installed`, set its automation to run `dnf-package-removal.sh`, and use this query:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM rpm_packages
  WHERE name IN (SELECT line FROM file_lines WHERE path = '/etc/dnf/blocked-packages.txt')
);
```

> **Note:** The first policy only checks that the files exist. After you change the variable, run `dnf-blocked-packages.sh` again. To run it on many hosts at once, see [Batch execute scripts](https://fleetdm.com/guides/scripts#batch-execute-scripts).

To learn more, see [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

### Fleets with other Linux distributions

The Linux platform includes every Linux host in the fleet. On hosts that don't use dnf, like Ubuntu, the first policy fails and runs the dnf script. To limit both policies to hosts with dnf, create a dynamic [label](https://fleetdm.com/guides/managing-labels-in-fleet) named `Hosts with dnf` with this query, and target both policies to it:

```sql
SELECT 1 FROM file WHERE path IN ('/usr/bin/dnf', '/usr/bin/dnf5');
```

## Manage with GitOps

To manage these settings with [GitOps](https://fleetdm.com/docs/configuration/yaml-files):

1. Add the package list to your repository's secrets as `FLEET_SECRET_DNF_BLOCKED_PACKAGES`.
2. Pass the secret to your GitOps workflow:

```yaml
env:
  FLEET_SECRET_DNF_BLOCKED_PACKAGES: ${{ secrets.FLEET_SECRET_DNF_BLOCKED_PACKAGES }}
```

> **Note:** If you run `fleetctl gitops` from your own computer, like for a local test, skip steps 1 and 2. Instead, set the variable in the same terminal window before you run `fleetctl gitops`:
>
> ```bash
> export FLEET_SECRET_DNF_BLOCKED_PACKAGES="sshpass telnet"
> ```

3. Save the scripts in your GitOps repository and reference them in your fleet's YAML file:

```yaml
labels:
  - name: Hosts with dnf
    description: Linux hosts that use dnf to install packages.
    label_membership_type: dynamic
    platform: linux
    query: SELECT 1 FROM file WHERE path IN ('/usr/bin/dnf', '/usr/bin/dnf5');

controls:
  scripts:
    - path: ../lib/linux/scripts/dnf-blocked-packages.sh
    - path: ../lib/linux/scripts/dnf-package-removal.sh

policies:
  - name: Linux - dnf blocklist is present
    query: |
      SELECT 1 WHERE (
        SELECT COUNT(*) FROM file
        WHERE path IN ('/etc/dnf/libdnf5.conf.d/90-fleet-blocked.conf', '/etc/dnf/blocked-packages.txt')
      ) = 2;
    platform: linux
    labels_include_any:
      - Hosts with dnf
    run_script:
      path: ../lib/linux/scripts/dnf-blocked-packages.sh
  - name: Linux - Blocked dnf packages are not installed
    query: |
      SELECT 1 WHERE NOT EXISTS (
        SELECT 1 FROM rpm_packages
        WHERE name IN (SELECT line FROM file_lines WHERE path = '/etc/dnf/blocked-packages.txt')
      );
    platform: linux
    labels_include_any:
      - Hosts with dnf
    run_script:
      path: ../lib/linux/scripts/dnf-package-removal.sh
```

> **Note:** Each `path` is relative to the YAML file it's in. Change the paths to match where the scripts are saved in your repository, like `../lib/servers/scripts/`.

> **Warning:** The `labels` key replaces all labels defined in that file. If your file already has a `labels` section, add `Hosts with dnf` to it. If your organization manages labels outside GitOps, leave out the `labels` section and create the label in the Fleet UI instead.

To learn more, see [Custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles#gitops).

## What users see

| Situation | What happens |
| --------- | ------------ |
| Blocked package | dnf stops the install: `Argument 'sshpass' matches only excluded packages.` Reinstalling an excluded package shows the same message. |
| Blocked dependency | If a package depends on a blocked package, dnf stops the install and says the dependency `is filtered out by exclude filtering`. |
| Download | `dnf download` stops with `No package "sshpass" available`. |
| Removed package | The package is removed without a prompt. If users try to install it again, dnf blocks it. |
| User without `sudo` | dnf stops any install and says the operation requires superuser privileges. |

## Verify

1. On a host, run `cat /etc/dnf/libdnf5.conf.d/90-fleet-blocked.conf /etc/dnf/blocked-packages.txt` and confirm the files list the packages you blocked.
2. Run `sudo dnf install <blocked package>` and confirm dnf refuses.

To check across all your hosts, run this report to list installed packages:

```sql
SELECT name, version FROM rpm_packages;
```

To check for one package, add `WHERE name = '<name>'`. If it was removed, the report returns no results.

> **Note:** To see dnf packages on a host's **Software** tab, select **Package (RPM)** in the **Types** filter.

## Troubleshoot

**Users can still install blocked packages**

Check these in order:

1. Confirm the package name matches exactly.
2. Confirm `/etc/dnf/libdnf5.conf.d/90-fleet-blocked.conf` exists and the host uses dnf5. Run `dnf --version` to check.
3. Check whether another script or policy automation also writes to the file, which would overwrite your settings.
4. Users with `sudo` rights can edit or delete the blocking files. The [blocklist policy](#run-automatically-with-policies) puts deleted files back the next time the host checks in. To restore edited files, run `dnf-blocked-packages.sh` again.

**A host doesn't block a package you added**

Changing the variable doesn't update hosts that already have the blocking files. Run `dnf-blocked-packages.sh` again on those hosts. To run it on many hosts at once, see [Batch execute scripts](https://fleetdm.com/guides/scripts#batch-execute-scripts).

**The removal script skips a package**

Other installed packages need it. Fleet retries the script up to 3 times, and the host keeps failing the policy until the dependent packages are gone. Remove the dependent packages first, or add them to your blocked packages.

Some packages are needed by the operating system. For example, on Fedora 45 Workstation, `jq` is required by `clevis`, `dracut-network`, and `lvm2`. Don't block packages that system tools depend on.

**Your own removal script can't remove an excluded package**

dnf hides excluded packages from `dnf remove`. Add `--setopt=disable_excludes='*'` to the script's dnf commands, as in [Remove installed packages](#remove-installed-packages).

**A blocked package is still installed**

Blocking doesn't remove packages that were already installed. Run `dnf-package-removal.sh`, or set it to [run automatically](#run-automatically-with-policies).

**The dnf policies run on hosts that don't use dnf**

Limit both policies to the `Hosts with dnf` label. See [Fleets with other Linux distributions](#fleets-with-other-linux-distributions).

**`fleetctl gitops` fails with `script file does not exist`**

A script `path` in your YAML file doesn't match where the script is saved. Paths are relative to the YAML file, so check the folder names and the file name, then run `fleetctl gitops` again.

## Further reading

- [dnf5 configuration reference](https://dnf5.readthedocs.io/en/latest/dnf5.conf.5.html)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)
- [Labels in Fleet](https://fleetdm.com/guides/managing-labels-in-fleet)
- [Block Homebrew packages on macOS](https://fleetdm.com/guides/block-homebrew-packages-on-macos)

<meta name="articleTitle" value="Block dnf packages on Fedora">
<meta name="authorFullName" value="Matthew Thomas">
<meta name="authorGitHubUsername" value="adriant172">
<meta name="publishedOn" value="2026-11-13">
<meta name="category" value="guides">
<meta name="description" value="Block, remove, and keep dnf packages off Fedora hosts with Fleet using scripts and policies.">
