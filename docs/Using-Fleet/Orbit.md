# Orbit

- [Introduction](#introduction)
- [Try Orbit](#try-orbit)
    + [With [`fleetctl preview` already running](https://github.com/fleetdm/fleet#try-fleet):](#with---fleetctl-preview--already-running--https---githubcom-fleetdm-fleet-try-fleet--)
- [Capabilities](#capabilities)
- [Usage](#usage)
  * [Permissions](#permissions)
    + [macOS/Linux](#macos-linux)
    + [Windows](#windows)
  * [Osquery shell](#osquery-shell)
  * [Connect to a Fleet server](#connect-to-a-fleet-server)
  * [Osquery flags](#osquery-flags)
- [Packaging](#packaging)
  * [Dependencies](#dependencies)
  * [Packaging support](#packaging-support)
  * [Building packages](#building-packages)
    + [Update channels](#update-channels)
    + [macOS signing & Notarization](#macos-signing---notarization)
    + [Orbit Osquery Result and Status Logs](#orbit-osquery-result-and-status-logs)
    + [Orbit Development](#orbit-development)
      - [Run Orbit From Source](#run-orbit-from-source)
      - [Generate Installer Packages from Orbit Source](#generate-installer-packages-from-orbit-source)
  * [Troubleshooting](#troubleshooting)
    + [Logs](#logs)
    + [Debug](#debug)
  * [Uninstall](#uninstall)
    + [Windows](#windows-1)
    + [Linux](#linux)
    + [macOS](#macos)
- [Bugs](#bugs)


## Introduction

Orbit is an [osquery](https://github.com/osquery/osquery) runtime and autoupdater. With Orbit, it's easy to deploy osquery, manage configurations, and stay up to date. Orbit eases the deployment of osquery connected with a [Fleet server](https://github.com/fleetdm/fleet), and is a (near) drop-in replacement for osquery in a variety of deployment scenarios.

Orbit is the recommended agent for Fleet. But Orbit can be used with or without Fleet, and Fleet can
be used with or without Orbit.

## Try Orbit

#### With [`fleetctl preview` already running](https://github.com/fleetdm/fleet#try-fleet):

```bash
# With fleetctl in your $PATH
# Generate a macOS installer pointed at your local Fleet
fleetctl package --type=pkg --fleet-url=localhost:8412 --insecure --enroll-secret=YOUR_FLEET_ENROLL_SECRET_HERE
```

> With fleetctl preview running, you can find your Fleet enroll secret by selecting the "Add new host" button on the Hosts page in the Fleet UI.

An installer configured to point at your Fleet instance has now been generated.

Now run that installer (double click, on a Mac) to enroll your own computer as a host in Fleet. Refresh after several seconds (≈30s), and you should now see your local computer as a new host in Fleet.

## Capabilities

| Capability                           | Status |
| ------------------------------------ | ------ |
| Secure autoupdate for osquery        | ✅     |
| Secure autoupdate for Orbit          | ✅     |
| Configurable update channels         | ✅     |
| Full osquery flag customization      | ✅     |
| Package tooling for macOS `.pkg`     | ✅     |
| Package tooling for Linux `.deb`     | ✅     |
| Package tooling for Linux `.rpm`     | ✅     |
| Package tooling for Windows `.msi`   | ✅     |
| Manage/update osquery extensions     | 🔜     |
| Manage cgroups for Linux performance | 🔜     |

## Usage

General information and flag documentation can be accessed by running `orbit --help`.

### Permissions

Orbit generally expects root permissions to be able to create and access it's working files.

To get root level permissions:

#### macOS/Linux

Prefix `orbit` commands with `sudo` (`sudo orbit ...`) or run in a root shell.

#### Windows

Run Powershell or cmd.exe with "Run as administrator" and start `orbit` commands from that shell.

### Osquery shell

Run an `osqueryi` shell with `orbit osqueryi` or `orbit shell`.

### Connect to a Fleet server

Use the `--fleet-url` and `--enroll-secret` flags to connect to a Fleet server.

For example:

```sh
orbit --fleet-url=https://localhost:8080 --enroll-secret=the_secret_value
```

Use `--fleet_certificate` to provide a path to a certificate bundle when necessary for osquery to verify the authenticity of the Fleet server (typically when using a Windows client or self-signed certificates):

```sh
orbit --fleet-url=https://localhost:8080 --enroll-secret=the_secret_value --fleet-certificate=cert.pem
```

Add the `--insecure` flag for connections using otherwise invalid certificates:

```sh
orbit --fleet-url=https://localhost:8080 --enroll-secret=the_secret_value --insecure
```

### Osquery flags

Orbit can be used as near drop-in replacement for `osqueryd`, enhancing standard osquery with autoupdate capabilities. Orbit passes through any options after `--` directly to the `osqueryd` instance.

For example, the following would be a typical drop-in usage of Orbit:

```sh
orbit -- --flagfile=flags.txt
```

## Packaging

Orbit, like standalone osquery, is typically deployed via OS-specific packages. Tooling is provided with this repository to generate installation packages.

### Dependencies

Orbit currently supports building packages on macOS and Linux.

Before building packages, clone or download this repository and [install Go](https://golang.org/doc/install).

Building Windows packages requires Docker to be installed.

### Packaging support

- **macOS** - `.pkg` package generation with (optional) [Notarization](https://developer.apple.com/documentation/xcode/notarizing_macos_software_before_distribution) and codesigning - Persistence via `launchd`.

- **Linux** - `.deb` (Debian, Ubuntu, etc.) & `.rpm` (RHEL, CentOS, etc.) package generation - Persistence via `systemd`.

- **Windows** - `.msi` package generation - Persistence via Services.

### Building packages

Use `fleetctl package` to run the packaging tools.

The only required parameter is `--type`, use one of `deb`, `rpm`, `pkg`, or `msi`.

Configure osquery to connect to a Fleet (or other TLS) server with the `--fleet-url` and `--enroll-secret` flags.

A minimal invocation for communicating with Fleet:

```sh
fleetctl package --type deb --fleet-url=fleet.example.com --enroll-secret=notsosecret
```

This will build a `.deb` package configured to communicate with a Fleet server at `fleet.example.com` using the enroll secret `notsosecret`.

When the Fleet server uses a self-signed (or otherwise invalid) TLS certificate, package with the `--insecure` or `--fleet-certificate` options.

See `fleetctl package` for the full range of packaging options.

#### Fleet Desktop

[Fleet Desktop](./Fleet-desktop.md) is a menu bar icon available on macOS, Windows, and Linux that gives your end users visibility into the security posture of their machine.

You can include Fleet Desktop in the orbit package by including the `--fleet-desktop`option. 

#### Update channels

Orbit uses the concept of "update channels" to determine the version of Orbit, Fleet Desktop, osquery, and any extensions (extension support coming soon) to run. This concept is modeled from the common versioning convention for Docker containers.

Configure update channels for Orbit and osqueryd with the `--orbit-channel`, `--desktop-channel` and `--osqueryd-channel` flags when packaging.

| Channel | Versions |
| ------- | -------- |
| `4`     | 4.x.x    |
| `4.6`   | 4.6.x    |
| `4.6.0` | 4.6.0    |

Additionally `stable` and `edge` are special channel names. The `stable` channel will provide the most recent osquery version that Fleet deems to be stable. When a new version of osquery is released, it is added to the `edge` channel for beta testing. Fleet then provides input to the osquery TSC based on testing. After the version is declared stable by the osquery TSC, Fleet will promote the version to `stable` ASAP.

#### macOS signing & Notarization

Orbit's packager can automate the codesigning and Notarization steps to allow the resulting package to generate packages that appear "trusted" when install on macOS hosts. Signing & notarization are supported only on macOS hosts.

For signing, a "Developer ID Installer" certificate must be available on the build machine ([generation instructions](https://help.apple.com/xcode/mac/current/#/dev154b28f09)). Use `security find-identity -v` to verify the existence of this certificate and make note of the identifier provided in the left column.

For Notarization, valid App Store Connect credentials must be available on the build machine. Set these in the environment variables `AC_USERNAME` and `AC_PASSWORD`. It is common to configure this via [app-specific passwords](https://support.apple.com/en-ca/HT204397). Some organizations (notably those with Apple Enterprise Developer Accounts) may also need to specify `AC_TEAM_ID`. This value can be found on the [Apple Developer "Membership" page](https://developer.apple.com/account/#!/membership) under "Team ID".

Build a signed and notarized macOS package with an invocation like the following:

```sh
AC_USERNAME=zach@example.com AC_PASSWORD=llpk-sije-kjlz-jdzw fleetctl package --type=pkg --fleet-url=fleet.example.com --enroll-secret=63SBzTT+2UyW --sign-identity 3D7260BF99539C6E80A94835A8921A988F4E6498 --notarize
```

This process may take several minutes to complete as the Notarization process completes on Apple's servers.

After successful notarization, the generated "ticket" is automatically stapled to the package.

#### Orbit Osquery Result and Status Logs

If the `logger_path` configuration is set to `filesystem`, Orbit will store osquery's "result" and
"status" logs to the following directories:
  - Windows: C:\Program Files\Orbit\osquery_log
  - macOS: /opt/orbit/osquery_log
  - Linux: /opt/orbit/osquery_log

#### Orbit Development

##### Run Orbit From Source

To execute orbit from source directly, run the following command:

```sh
go run github.com/fleetdm/fleet/v4/orbit/cmd/orbit \
    --dev-mode \
    --disable-updates \
    --root-dir /tmp/orbit \
    --fleet-url https://localhost:8080 \
    --insecure \
    --enroll-secret Pz3zC0NMDdZfb3FtqiLgwoexItojrYh/ \
    -- --verbose
```

Or, using a `flagfile.txt` for osqueryd:
```sh 
go run github.com/fleetdm/fleet/v4/orbit/cmd/orbit \
    --dev-mode \
    --disable-updates \
    --root-dir /tmp/orbit \
    -- --flagfile=flagfile.txt --verbose
```

##### Generate Installer Packages from Orbit Source

The `fleetctl package` command generates installers by fetching the targets/executables from a [TUF](https://theupdateframework.io/) repository.
To generate an installer that contains an Orbit built from source you need to setup a local TUF repository.
The following document explains how you can generate a TUF repository, and installers that use it [tools/tuf/test](../tools/tuf/test/README.md).

### Troubleshooting

#### Logs

Orbit captures and streams osqueryd's stdout/stderr into its own stdout/stderr output.
These are the log destinations for each platform:
- Linux: Orbit and osqueryd stdout/stderr output is sent to syslog (`/var/log/syslog` on Debian systems and `/var/log/messages` on CentOS).
- macOS: `/private/var/log/orbit/orbit.std{out|err}.log`.
- Windows: `C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log` (the log file is rotated).
 Users will need administrative permissions on the host to access these log destinations.
#### Debug

You can use the `--debug` option in `fleetctl package` to generate installers in "debug mode." This mode increases the verbosity of logging for orbit and osqueryd (log DEBUG level).

### Uninstall
#### Windows

Use the "Add or remove programs" dialog to remove Orbit.

#### Linux

Uninstall the package with the corresponding package manager:

- Ubuntu
```sh
sudo apt remove fleet-osquery -y
```
- CentOS
```sh
sudo rpm -e fleet-osquery-X.Y.Z.x86_64
```

#### macOS

Run the [cleanup script](./tools/cleanup/cleanup_macos.sh).

## Bugs

To report a bug or request a feature, [click here](https://github.com/fleetdm/fleet/issues).