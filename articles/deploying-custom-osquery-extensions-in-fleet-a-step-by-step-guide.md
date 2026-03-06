# Deploying custom osquery extensions in Fleet: A step-by-step guide

### Links to article series:

- Part 1: [Deploying custom osquery extensions in Fleet](https://fleetdm.com/articles/deploying-custom-osquery-extensions-in-fleet)
- Part 2: Deploying custom osquery extensions in Fleet: A step-by-step guide

## Step 1: Deploy the extension binary

Deploy your custom extension binary to any location on the target filesystem. This can be accomplished through:

- Package installers (`.pkg` on macOS, `.msi` on Windows, `.deb` or `.rpm` on Linux)
- Scripts
- Manual deployment for testing

### Critical requirements

#### Ownership

Ensure the extension file is owned by `root:wheel` (macOS) or `root:admin` (some Linux systems)

#### Permissions

Set appropriate execute permissions (typically `755`)

## Step 2: Configure the extensions loader

Create a text file named `extensions.load` and place it in the `/var/osquery/` directory. For some operating systems, this is `/etc/osquery/`. This file should contain the full path to your custom extension binary, with one extension path per line.

Example `extensions.load` file:

```
/usr/local/bin/my-custom-extension.ext
/opt/security/monitoring-extension.ext
```

## Step 3: Restart the orbit agent

After placing the extension and configuration file, restart the Orbit agent to load the new extension:

### macOS

```
sudo launchctl stop com.fleetdm.orbit
sudo launchctl start com.fleetdm.orbit
```

### Use systemctl on systemd systems

```
sudo systemctl restart orbit
```

#### For complete examples, see Fleet's repository:

- [Example Script](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/scripts/install-macos-compatibility-extension.sh)
- [Example Policy](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/policies/install-macos-compatibility-extension.yml)

## Considerations and best practices

### Security

When selecting custom extensions to deploy, prioritize open source solutions whenever possible. Open source extensions provide transparency, allowing you to audit the code for security vulnerabilities and ensure trustworthiness before deployment.

### Version management

Implement versioning in your detection policies and extension names to handle updates.

### Testing

Always test extensions locally before deploying through Fleet. You can do this by running `orbit/osqueryi` locally using a command similar to: 

```
$ sudo /path/to/orbit shell -- --extension /path/to/extension.ext
```

## Custom extension examples

Here are some examples of the custom extensions we use at Fleet: 

- [macos_compatibility](https://github.com/harrisonravazzolo/macos-compatibility-ext): Get a snapshot of what version of macOS your MacBooks are compatible with.
- [snap_packages](https://github.com/allenhouchins/fleet-stuff/tree/main/linux-mdm-snap-packages): Collect packages installed by snap in a similar syntax to what you are used to with deb_packages or rpm_packages.

About the author: Allen Houchins is Head of IT & Solutions Consulting at Fleet Device Management.

<meta name="articleTitle" value="Deploying custom osquery extensions in Fleet: A step-by-step guide">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-06">
<meta name="description" value="Learn how to deploy custom osquery extensions directly from Fleet.">
