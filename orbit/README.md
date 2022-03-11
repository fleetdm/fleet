<img width="200" alt="Fleet logo, landscape, dark text, transparent background" src="https://user-images.githubusercontent.com/618009/103300491-9197e280-49c4-11eb-8677-6b41027be800.png">

# Orbit osquery

Orbit is an [osquery](https://github.com/osquery/osquery) runtime and autoupdater. With Orbit, it's easy to deploy osquery, manage configurations, and stay up to date. Orbit eases the deployment of osquery connected with a [Fleet server](https://github.com/fleetdm/fleet), and is a (near) drop-in replacement for osquery in a variety of deployment scenarios.

Orbit is the recommended agent for Fleet. But Orbit can be used with or without Fleet, and Fleet can be used with or without Orbit.

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

## Bugs

To report a bug or request a feature, [click here](https://github.com/fleetdm/fleet/issues).

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

#### Update channels

Orbit uses the concept of "update channels" to determine the version of Orbit, osquery, and any extensions (extension support coming soon) to run. This concept is modeled from the common versioning convention for Docker containers.

Configure update channels for Orbit and osqueryd with the `--orbit-channel` and `--osqueryd-channel` flags when packaging.

| Channel | Versions |
| ------- | -------- |
| `4`     | 4.x.x    |
| `4.6`   | 4.6.x    |
| `4.6.0` | 4.6.0    |

Additionally `stable` and `edge` are special channel names. `stable` will always return the version Fleet deems to be stable, while `edge` will provide newer releases for beta testing.

#### macOS signing & Notarization

Orbit's packager can automate the codesigning and Notarization steps to allow the resulting package to generate packages that appear "trusted" when install on macOS hosts. Signing & notarization are supported only on macOS hosts.

For signing, a "Developer ID Installer" certificate must be available on the build machine ([generation instructions](https://help.apple.com/xcode/mac/current/#/dev154b28f09)). Use `security find-identity -v` to verify the existence of this certificate and make note of the identifier provided in the left column.

For Notarization, valid App Store Connect credentials must be available on the build machine. Set these in the environment variables `AC_USERNAME` and `AC_PASSWORD`. It is common to configure this via [app-specific passwords](https://support.apple.com/en-ca/HT204397).

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
  - macOS: /var/lib/orbit/osquery_log
  - Linux: /var/lib/orbit/osquery_log

#### Orbit Development

For ease of development of Orbit, `fleetctl package` allows the generation of a package with a
custom orbit executable using the `FLEETCTL_ORBIT_DEV_BUILD_PATH` environment variable:
```sh
FLEETCTL_ORBIT_DEV_BUILD_PATH=$(pwd)/orbit.exe ./build/fleetctl package --type=msi --fleet-url=https://localhost:8080 --enroll-secret=the_secret_value
Generating your osquery installer...
2022/01/03 20:31:10 root pinning is not supported in Spec 1.0.19
WARNING: You are attempting to override orbit with a dev build.
Press Enter to continue, or Control-c to exit.
[...]
```

If you want to run orbit from source directly, you can do the following:

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


### Troubleshooting

#### Logs

Orbit captures and streams osqueryd's stdout/stderr into its own stdout/stderr output.
Following are the destination of logs for each platform (to access such locations the user will need administrative permissions on the host):
- Linux: Orbit and osqueryd stdout/stderr output is sent to syslog (`/var/log/syslog` on Debian systems and `/var/log/messages` on CentOS).
- macOS: `/private/var/log/orbit/orbit.std{out|err}.log`.
- Windows: `C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.lg` (the log file is rotated).

#### Debug

You can use the `--debug` option in `fleetctl package` to generate installers in "debug mode". Such mode increases the verbosity of logging for orbit and osqueryd (log DEBUG level).

### Uninstall
#### Windows

Use the "Add or remove programs" dialog to remove Orbit.

#### Linux

Run the [cleanup script](./tools/cleanup/cleanup_linux.sh).

#### macOS

Run the [cleanup script](./tools/cleanup/cleanup_macos.sh).

## FAQs

### How does Orbit compare with Kolide Launcher?

Orbit is inspired by the success of [Kolide Launcher](https://github.com/kolide/launcher), and approaches a similar problem domain with new strategies informed by the challenges encountered in real world deployments. Orbit does not share any code with Launcher.

- Both Orbit and Launcher use [The Update Framework](https://theupdateframework.com/) specification for managing updates. Orbit utilizes the official [go-tuf](https://github.com/theupdateframework/go-tuf) library, while Launcher has it's own implementation of the specification.
- Orbit can be deployed as a (near) drop-in replacement for osquery, supporting full customization of the osquery flags. Launcher heavily manages the osquery flags making deployment outside of Fleet or Kolide's SaaS difficult.
- Orbit prefers the battle-tested plugins of osquery. Orbit uses the built-in logging, configuration, and live query plugins, while Launcher uses custom implementations.
- Orbit prefers the built-in osquery remote APIs. Launcher utilizes a custom gRPC API that has led to issues with character encoding, load balancers/proxies, and request size limits.
- Orbit encourages use of the osquery performance Watchdog, while Launcher disables the Watchdog.

Additionally, Orbit aims to tackle problems out of scope for Launcher:

- Configure updates via release channels, providing more granular control over agent versioning.
- Support for deploying and updating osquery extensions (🔜).
- Manage osquery versions and startup flags from a remote (Fleet) server (🔜).
- Further control of osquery performance via cgroups (🔜).

### Is Orbit Free?

Yes! Orbit is licensed under an MIT license and all uses are encouraged.

### How does orbit update osquery? And how do the stable and edge channels get triggered to update osquery on a self hosted Fleet instance?

Orbit uses a configurable update server. We expect that many folks will just use the update server we manage (similar to what Kolide does with Launcher's update server). We are also offering [tooling for self-managing an update server](https://fleetdm.com/docs/deploying/fleetctl-agent-updates) as part of Fleet Premium (the subscription offering).

## Community

#### Chat

Please join us in the #fleet channel on [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/).

<a href="https://fleetdm.com"><img alt="Banner featuring a futuristic cloud city with the Fleet logo" src="https://user-images.githubusercontent.com/618009/98254443-eaf21100-1f41-11eb-9e2c-63a0545601f3.jpg"/></a>
