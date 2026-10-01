# fleetd proxy installers

Build scripts for fleetd installers that take the Fleet URL, enroll secret and proxy at install time, for hosts that can only reach Fleet through a forward proxy (for example, Squid). One build can be deployed behind any proxy, so different teams can use different proxies and enroll secrets with the same package.

fleetd's two components read proxy settings differently: Orbit uses `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`, and osquery only uses its `--proxy_hostname` flag. These installers set both from one set of install-time values.


## Platforms

- [Windows (MSI)](windows/README.md): values are passed as msiexec properties.
- [Linux (DEB and RPM)](linux/README.md): values are passed as environment variables on the install command, or in `/etc/default/fleetd-proxy`.

Each README covers how to build the installer and how to install it. Both builds run in a throwaway container, so nothing is installed on the build machine.


## Limitations

- Unauthenticated proxies only. fleetd has no supported authenticated-proxy path (#35957).
- SSL-inspecting proxies haven't been tested.
- Fleet Desktop doesn't use these proxy settings (#49351), so leave it off.
- The installers aren't signed. Sign them with your own certificate if your environment requires it.
