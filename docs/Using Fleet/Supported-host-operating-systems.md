# Supported host operating systems

Fleet supports the following operating system versions on hosts. 

| OS      | Supported version(s)                    |
| :------ | :-------------------------------------  |
| macOS   | 13+ (Ventura)                           |
| Windows | Pro and Enterprise 10+, Server 2012+    |
| Linux   | CentOS 7.1+,  Ubuntu 20.04+, Fedora 38+ |
| ChromeOS | 112.0.5615.134+                        |

While Fleet may still function partially or fully with OS versions older than those above, Fleet does not actively test against unsupported versions and does not pursue bugs on them. 

## Some notes on compatibility

### Tables

Not all osquery tables are available for every OS. Please check out the [osquery schema](https://fleetdm.com/tables) for detailed information. 

If a table is not available for your host, Fleet will generally handle things behind the scenes for you. 

### Linux

Fleet Desktop is supported on Ubuntu and Fedora. 

Fedora requires a [gnome extension](https://extensions.gnome.org/extension/615/appindicator-support/) and Google Chrome for Fleet Desktop.

On Ubuntu, Fleet Desktop currently supports Xorg as X11 server, Wayland is currently not supported. Ubuntu 24.04 comes with Wayland enabled by default. To use X11 instead of Wayland you can set `WaylandEnable=false` in `/etc/gdm3/custom.conf` and reboot.

The `fleetctl package` command is not supported on DISA-STIG distribution.

<meta name="pageOrderInSection" value="1200">
<meta name="description" value="This page contains information about operating systems that are compatible with Fleet's agent (fleetd).">
<meta name="navSection" value="The basics">
