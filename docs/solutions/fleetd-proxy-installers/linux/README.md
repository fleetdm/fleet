# Build proxy-aware fleetd Linux packages (DEB and RPM)

This builds fleetd `.deb` and `.rpm` packages that take the Fleet URL, enroll secret and proxy at install time. It's for headless Linux hosts that can only reach Fleet through a forward proxy.

```
sudo env FLEET_URL=https://fleet.example.com FLEET_SECRET=<enroll-secret> ORBIT_PROXY_URL=http://proxy.example.internal:3128 \
  apt-get install -y ./fleetd-proxy_1.60.0_amd64.deb
```

Files:

- `build-linux-in-container.sh`: runs the build in a throwaway container. Use this one.
- `build-fleetd-proxy-linux.sh`: the build itself. `build-linux-in-container.sh` runs it for you.


## The install values

Linux packages can't take install-time properties the way msiexec does. Instead, the package reads the values from environment variables on the install command:

- `FLEET_URL`: the Fleet server, for example `https://fleet.example.com`.
- `FLEET_SECRET`: the enroll secret. A team's secret puts the host in that team.
- `ORBIT_PROXY_URL`: the proxy as a URL with its port, for example `http://proxy.example.internal:3128`. Use `http://` even though Fleet is HTTPS: the scheme describes the connection to the proxy, and the HTTPS traffic is tunneled through it with CONNECT.
- `OSQUERY_PROXY`: optional. osquery's proxy is taken from `ORBIT_PROXY_URL`, so set this only if osquery should use a different proxy. Give it as host:port with no scheme.
- `NO_PROXY`: optional. It lists destinations Orbit should reach directly instead of through the proxy, and defaults to `localhost,127.0.0.1`. Most deployments don't need it. osquery has no bypass list, so this doesn't affect osquery.

The proxy needs to allow CONNECT to the Fleet server on 443 and to `updates.fleetdm.com:443`. The proxy resolves those names, so hosts don't need DNS for them.


## Install

Debian and Ubuntu:

```
sudo env FLEET_URL=https://fleet.example.com FLEET_SECRET=<enroll-secret> ORBIT_PROXY_URL=http://proxy.example.internal:3128 \
  apt-get install -y ./fleetd-proxy_1.60.0_amd64.deb
```

RHEL, Oracle Linux, Rocky and Alma:

```
sudo env FLEET_URL=https://fleet.example.com FLEET_SECRET=<enroll-secret> ORBIT_PROXY_URL=http://proxy.example.internal:3128 \
  dnf install -y ./fleetd-proxy-1.60.0.x86_64.rpm
```

`dpkg -i` works the same way as `apt-get install`. `rpm -i` and `yum install` should too, but weren't tested. Put `env` after `sudo` as shown. Some sudo configurations drop variables that are set in front of the command, but `env` sets them for the installer itself.

If a deployment tool can't set environment variables, write the values to `/etc/default/fleetd-proxy` before installing, one `KEY=value` per line. Make the file readable by root only, since it holds the secret. Environment variables win if both are set.

```
FLEET_URL=https://fleet.example.com
FLEET_SECRET=<enroll-secret>
ORBIT_PROXY_URL=http://proxy.example.internal:3128
```

To change a value later, reinstall the same package with just the new value. Values you don't pass keep their current setting:

```
sudo env ORBIT_PROXY_URL=http://new-proxy.example.internal:3128 apt-get install --reinstall -y ./fleetd-proxy_1.60.0_amd64.deb
```

The package is still named `fleet-osquery`, like Fleet's standard package. To uninstall, run `sudo apt-get purge fleet-osquery` or `sudo dnf remove fleet-osquery`.


## What the script does

The script builds Fleet's stock packages with `fleetctl package --type deb` and `--type rpm`, then changes only their install scripts. It checks that everything else in the package (binaries, systemd unit, `/etc/default/orbit`) is identical to stock.

The added install step:

- writes the Fleet URL, enroll secret and `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY` to `/etc/default/orbit-proxy` (root-only).
- adds a systemd drop-in, `/etc/systemd/system/orbit.service.d/10-fleetd-proxy.conf`. It loads that file after `/etc/default/orbit`, so its values win, and it adds `-- --proxy_hostname=...` to Orbit's command line, which Orbit passes through to osqueryd.

The osquery proxy goes on the command line rather than in `osquery.flags`, because Orbit rewrites `osquery.flags` whenever a team's agent options set `command_line_flags`.

None of the package's own files are edited, so package upgrades keep the settings. With no values supplied (for example, an upgrade), the existing settings are left alone. Uninstalling removes both files but leaves `/etc/default/fleetd-proxy` in place.


## What's been tested

We tested the fleetctl 4.91.1 packages on Ubuntu 24.04 (environment variables and apt-get), Debian 13 (`/etc/default/fleetd-proxy` and dpkg) and Rocky Linux 9 (environment variables and dnf). Each host sat behind Squid with all other outbound traffic blocked. All three enrolled, and everything went through the proxy: Orbit's enrollment, check-ins and TUF updates, and osquery's enrollment, query results and software inventory. Reinstalling, changing the proxy by reinstalling, and uninstalling were also tested.

Oracle Linux and RHEL weren't tested directly. They're binary-compatible with Rocky and use the same RPM.

New builds bundle whatever Orbit and osquery are on the stable channel (Orbit 1.61.0 as of 2026-09-23). The tested builds had Orbit 1.60.0. Orbit updates itself to stable either way, so after the first update hosts run the same version regardless. To build exactly the tested versions, set `ORBIT_CHANNEL=1.60.0 OSQUERYD_CHANNEL=5.23.1`.

Only plain CONNECT proxying has been tested. SSL-inspecting and authenticated proxies haven't.


## Build

Build the packages once and deploy them everywhere; the Fleet URL, enroll secret and proxy are supplied at install time.


### Where to run it

The recommended way runs the whole build inside a throwaway Fedora container, so the host's distro doesn't matter and nothing gets installed on it. Fedora is used because it packages the three tools the build needs: `dpkg-deb`, `rpm` and `rpmrebuild`. You need an amd64 (x86_64) machine with Docker or Podman:

- Debian or Ubuntu: `sudo apt-get install -y docker.io`
- RHEL, Rocky, Alma, Oracle Linux or Fedora: `sudo dnf install -y podman`, then run the build with `ENGINE=podman`.
- Windows (x64) or an Intel Mac: Docker Desktop. On Windows, run the script from a WSL2 shell. This should work but hasn't been tested.

The container build has been run with Docker on Debian 13 and with Podman on Rocky Linux 9 (RHEL-compatible). Both produced passing builds.

You can also run `build-fleetd-proxy-linux.sh` directly on Fedora, where dnf installs what it needs. On RHEL-family distros, `rpmrebuild` comes from EPEL. Running it directly hasn't been tested; the container is the tested path.


### Steps

1. Put both scripts in a directory and make them executable: `chmod +x *.sh`
2. Run the build. Use `sudo` unless your user is in the `docker` group:

   ```
   ./build-linux-in-container.sh
   ```

   With Podman: `ENGINE=podman ./build-linux-in-container.sh`

3. The script ends with a `VERIFY PASS` or `VERIFY FAIL` block. Don't ship a FAIL.
4. The packages are in `./fleetd-proxy-linux-out/`: `fleetd-proxy_<version>_amd64.deb` and `fleetd-proxy-<version>.x86_64.rpm`. The version is the Orbit version bundled at build time. The examples in this README use 1.60.0.

The first run takes several minutes while dnf downloads Fedora's package metadata inside the container.

The build needs outbound access to GitHub (the fleetctl download), Docker Hub (the Fedora image), the Fedora package mirrors and `updates.fleetdm.com` (the Orbit and osquery binaries). If the build machine is behind a proxy too, set `HTTPS_PROXY`, `HTTP_PROXY` and their lowercase forms in the shell (`build-linux-in-container.sh` passes them into the container), and give Docker or Podman proxy settings so it can pull the image. The build doesn't need any Fleet server credentials.

Options, all set as env vars:

- `FLEET_VER`: fleetctl release that builds the stock packages. Default `v4.91.1`.
- `FLEET_URL`: bakes a default Fleet URL into the packages, so installs only need the secret and proxy. An install-time `FLEET_URL` still wins.
- `ORBIT_CHANNEL`, `OSQUERYD_CHANNEL`: `stable` (default), `edge`, or a version such as `1.60.0` or `5.23.1`. A version sets what's bundled and also what the host auto-updates to, until you change it with `update_channels` in agent options.
- `ARCH=arm64`: ARM build. Untested.
- `TYPES`: `deb`, `rpm`, or both (default `"deb rpm"`).
- `OUT_DIR`: output directory.
- `ENGINE`: `docker` or `podman`. Defaults to Docker if it's installed.


## Verify on a host

```
systemctl status orbit
cat /etc/systemd/system/orbit.service.d/10-fleetd-proxy.conf
ps -eo args | grep '[o]squeryd' | grep -o -- '--proxy_hostname=[^ ]*'
```

In the proxy's access log, expect CONNECTs to the Fleet server and to `updates.fleetdm.com:443`. Orbit's CONNECTs carry a `Go-http-client` user agent. osquery's carry none.


## Limitations and tips

- Unauthenticated proxies only. fleetd has no supported authenticated-proxy path (#35957).
- SSL-inspecting proxies haven't been tested and need extra certificate setup.
- `FLEET_SECRET` can show up in deployment-tool logs, so use a team-scoped enroll secret.
- Like Fleet's standard packages, these aren't signed. If dnf enforces signature checks on local packages, add `--nogpgcheck`.
