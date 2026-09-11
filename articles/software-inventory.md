# Software inventory

Software inventory in Fleet collects the apps, operating systems, browser extensions, packages, IDE extensions, plugins, and binaries installed on your hosts. [Vulnerability (CVE) processing](https://fleetdm.com/guides/vulnerability-processing#coverage) runs against this inventory, so software that isn't collected here can't be checked for vulnerabilities.

Fleet collects software inventory for the software types below. Each table says where Fleet collects that software, what it stores beyond a name and a version, and whether it matches that software to vulnerabilities (CVEs). To see exactly which software Fleet checks, see [Vulnerability processing](https://fleetdm.com/guides/vulnerability-processing#coverage).

> If Fleet isn't collecting software that's installed on your hosts, please file a [feature request](https://github.com/fleetdm/fleet/issues/new?template=feature-request.md). If Fleet is collecting software incorrectly (wrong name, version, or type), please file a [bug](https://github.com/fleetdm/fleet/issues/new?template=bug-report.md).

## Apps

| Type | Collected on | Data collected | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- |
| Native apps | macOS, Windows, Android, iOS, iPadOS | Name and version everywhere. Bundle ID on Apple platforms, application ID on Android. macOS adds the install path, the Apple Developer Team ID, file hashes, and last opened. Windows adds the publisher, install path, upgrade code, and last opened. | ✅ macOS and Windows. ❌ Android, iOS, and iPadOS. | On Linux, apps install as [packages](#packages). Not collected on ChromeOS. |
| Microsoft Store (MSIX/Appx) apps | Windows | Name, version, publisher, install path, upgrade code, and last opened. | ✅ | Per-user and machine-wide provisioned packages need a recent version of Fleet's agent. |
| Alternative app store and sideloaded apps | Android, iOS, iPadOS | Expected to match native apps on the same platform. | ❌ | Fleet hasn't verified that these appear in MDM app lists. |
| Personal-side apps on BYOD hosts | ❌ Not collected | — | ❌ | Apple User Enrollment and Android work profiles don't expose them. |

- Native apps on ChromeOS: Fleet's ChromeOS agent runs as a browser extension. It has no API for installed Android apps or progressive web apps (PWAs).
- Native apps on Android: personally-owned (BYOD) hosts report apps in the work profile only. Fully-managed hosts report all installed apps.
- Native apps on iOS and iPadOS: personally-owned (BYOD) hosts report only the apps Fleet installed. Built-in and user-installed apps aren't included. Learn more in [Enrolling BYOD iPad/iOS devices](https://fleetdm.com/guides/enroll-byod-ios-ipados-hosts).
- Microsoft Store apps: hosts on fleetd's `stable` channel update automatically and pick up support for these as it ships.
- Personal-side apps on BYOD hosts: Apple User Enrollment and Android work profiles don't expose them.

## Operating systems

| Platform | Collected | Data collected | Vulnerabilities |
| --- | --- | --- | --- |
| macOS | ✅ | Name, version, architecture, and kernel version. | ✅ |
| Windows 10, 11, and Server | ✅ | Name, version, display version (such as 23H2), architecture, and installation type. | ✅ Matched against Microsoft security bulletins. |
| ChromeOS | ✅ | Name, version, and architecture. | ❌ |
| Android | ✅ | Name and version. | ✅ Matched against the Android security patch level. |
| iOS and iPadOS | ✅ | Name and version. | ❌ |
| Linux | ✅ On recognized distributions. See the table below. | Name, version, architecture, and kernel version. | ✅ Kernel vulnerabilities, on supported distributions. |

### Linux distributions

✅\* means Fleet collects software on the distribution, with a caveat in the notes below the table. Kernel vulnerability coverage varies by distribution. See [Linux coverage](https://fleetdm.com/guides/vulnerability-processing#linux-coverage).

| Distribution | Collected |
| --- | --- |
| Ubuntu | ✅ |
| Debian | ✅ |
| Red Hat Enterprise Linux (RHEL) | ✅ |
| CentOS and CentOS Stream | ✅ |
| Fedora | ✅\* |
| Rocky Linux | ✅\* |
| AlmaLinux | ✅\* |
| Oracle Linux | ✅\* |
| Amazon Linux | ✅ |
| SLES | ✅ |
| openSUSE Leap | ✅ |
| openSUSE Tumbleweed | ✅ |
| Arch Linux | ✅ |
| Arch Linux ARM | ✅ |
| Omarchy | ✅ |
| CachyOS | ✅ |
| EndeavourOS | ✅ |
| Manjaro | ✅ |
| Manjaro ARM | ✅ |
| Gentoo | ✅ |
| Kali | ✅ |
| Pop!_OS | ✅ |
| Linux Mint | ✅ |
| Zorin OS | ✅ |
| Void | ✅ |
| NixOS | ✅ |
| Tuxedo OS | ✅ |
| KDE neon | ✅ |
| Flatcar | ✅ |
| CoreOS | ✅ |
| AMD Ryzen AI Developer Platform | ✅ |
| NVIDIA DGX OS (DGX Spark, DGX Station, and DGX servers) | ✅\* |
| Raspberry Pi OS 64-bit | ✅\* |
| Raspberry Pi OS 32-bit | ❌ |
| Alpine | ❌ |
| elementary OS | ❌ |
| Deepin | ❌ |
| Garuda | ❌ |
| Clear Linux | ❌ |
| Photon OS | ❌ |
| Solus | ❌ |
| Parrot OS | ❌ |
| Vanilla OS | ❌ |
| openSUSE MicroOS and Aeon | ❌ |
| SteamOS | ❌ |
| Bazzite, Bluefin, and Aurora | ❌ |
| Ubuntu Core | ❌ |
| Any other distribution | ❌ |

- Fedora, Rocky Linux, AlmaLinux, and Oracle Linux report as RHEL because they ship `/etc/redhat-release`. Fleet shows them under RHEL on the **Software** > **OS** page.
- NVIDIA DGX OS reports as Ubuntu (DGX OS 7 as Ubuntu 24.04, DGX OS 6 as Ubuntu 22.04) because its `/etc/os-release` is Ubuntu's. Fleet doesn't read `/etc/dgx-release`, so the host appears as Ubuntu on the **Software** > **OS** page and the DGX OS version isn't reported. DGX Spark is arm64; use the `arm64` fleetd package.
- Raspberry Pi OS 64-bit reports as Debian and is covered. Raspberry Pi OS 32-bit reports as `raspbian`, which Fleet doesn't recognize.
- Alpine, elementary OS, Deepin, Garuda, Clear Linux, Photon OS, Solus, Parrot OS, Vanilla OS, openSUSE MicroOS, Aeon, and any distribution not listed above: Fleet recognizes Linux by matching the platform value Fleet reads from `/etc/os-release` against a [fixed list](https://github.com/fleetdm/fleet/blob/main/server/fleet/hosts.go). Hosts with any other value enroll, but Fleet doesn't collect an OS entry or software inventory for them.
- SteamOS, Bazzite, Bluefin, and Aurora: these use immutable root filesystems. Installing Fleet's agent is unsupported, and their platform values aren't in Fleet's list.
- Ubuntu Core runs snaps only. Fleet's agent isn't packaged as a snap, so Ubuntu Core hosts can't enroll.

## Browser extensions

| Browser | Collected on | Data collected | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- |
| Chrome | macOS, Windows, Linux, ChromeOS | Name, version, extension ID, browser, and install path. | ✅ | ChromeOS reports no install path. |
| Chromium, Brave, Edge, Edge Beta, Opera, and Yandex | macOS, Windows, Linux | Name, version, extension ID, browser, and install path. | ✅ | — |
| Chrome Beta, Dev, and Canary, Brave Beta and Nightly, Edge Dev and Canary, and Vivaldi | macOS, Windows, Linux | Name, version, extension ID, browser, and install path. | ✅ | The **Type** column shows the raw browser value, such as "Chrome Beta". |
| Arc | macOS | Name, version, extension ID, browser, and install path. | ✅ | Not collected on Windows. Shows the raw browser value. |
| Firefox | macOS, Windows, Linux | Name, version, extension ID, and install path. | ✅ | On Linux, the snap profile is collected and the Flatpak profile isn't. |
| Safari | macOS | Name and version only. | ✅ When the NVD has a matching entry. | Fleet doesn't store the extension identifier yet. |
| Internet Explorer | Windows | Name and version only. | ✅ When the NVD has a matching entry. | Fleet doesn't store the extension identifier yet. |
| Perplexity Comet and Dia | ❌ Not collected | — | ❌ | Support ships in an upcoming version of Fleet's agent. |

- Fleet doesn't hash browser extensions on any platform.
- Safari on iOS and Firefox, Edge, and Yandex on Android support extensions, but no MDM API exposes them.

## Packages

| Type | Collected on | Data collected | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- |
| Homebrew | macOS | Name, version, and install path. | ✅ | Formulae, plus casks that don't install a `.app` bundle. Those appear under Apps. Homebrew on Linux isn't collected. |
| Chocolatey | Windows | Name, version, and install path. | ✅ | — |
| deb | Linux | Name, version, and last opened. | ✅ Ubuntu. Debian covers the kernel only. | No install path, architecture, or publisher. |
| rpm | Linux | Name, version, publisher, architecture, release, and last opened. | ✅ RHEL, CentOS, Fedora, and Amazon Linux. | No install path. |
| pacman | Linux | Name, version, and architecture. | ✅ Matched against the NVD by version, which can produce false positives. | Needs Fleet's agent. |
| Portage | Linux | Name and version. | ✅ When the NVD has a matching entry. | — |
| Python | macOS, Windows, Linux | Name, version, and install path. | ✅ | System and per-user site-packages, plus pipx, uv tools, conda, pipenv, pyenv, and mise. Project-level `.venv` directories aren't collected. |
| npm | macOS, Linux | Name, version, and install path. | ✅ | Global packages only, including nvm, fnm, asdf, mise, and Volta locations. Windows isn't queried yet. Project-level `node_modules` directories aren't collected. |
| Scoop, winget portable packages, snap, Flatpak, AppImage, Nix, pnpm, Yarn, Bun, RubyGems, cargo, and Nim | ❌ Not collected | — | ❌ | Fleet doesn't read them. |
| Atom | ❌ Not collected | — | ❌ | Removed from inventory in December 2023 after the editor was sunset. |

- On Ubuntu desktop, Firefox and Chromium install as snaps by default, so those browsers are missing from inventory even though their extensions are collected. Follow the snap request in [#22658](https://github.com/fleetdm/fleet/issues/22658).

## IDE extensions

| Type | Collected on | Data collected | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- |
| VS Code, VS Code Insiders, VSCodium, VSCodium Insiders, Cursor, Windsurf (Devin), and Trae | macOS, Windows, Linux | Name, version, publisher, extension ID, edition, and install path. | ✅ | Remote-server variants are collected too. Learn more in the [vscode_extensions](https://fleetdm.com/tables/vscode_extensions) reference. |
| JetBrains: CLion, DataGrip, GoLand, IntelliJ IDEA (and Community Edition), PhpStorm, PyCharm (and Community Edition), ReSharper, Rider, RubyMine, RustRover, and WebStorm | macOS, Windows, Linux | Name, version, publisher, product, and install path. | ✅ | Fleet reads 13 products installed under the `JetBrains` directory. Learn more in the [jetbrains_plugins](https://fleetdm.com/tables/jetbrains_plugins) reference. |
| Google Antigravity, Kiro, and code-server | ❌ Not collected | — | ❌ | These use the VS Code extension format, but Fleet doesn't read their directories yet. |
| JetBrains: Android Studio, DataSpell, Aqua, and Writerside | ❌ Not collected | — | ❌ | Android Studio installs under a `Google` directory. The rest aren't in Fleet's list. |
| Zed, Sublime Text, Neovim/Vim, and Emacs | ❌ Not collected | — | ❌ | Fleet doesn't read them. |

## Plugins

| Type | Collected on | Data collected | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- |
| Adobe CEP extensions for Photoshop, Illustrator, Premiere Pro, After Effects, InDesign, InCopy, Animate, Dreamweaver, Audition, Bridge, Lightroom, Lightroom Classic, XD, and Prelude | macOS, Windows | Name, version, vendor, plugin ID, and install path. | ❌ Adobe files CVEs against the host application, which Fleet detects under Apps. | Adobe ships no Linux applications. The **Type** column shows "Plugin (Adobe)" for every host application, because the host application is read from the manifest but not stored. |
| Adobe UXP plugins for Photoshop, XD, InDesign, InCopy, and Premiere Pro | macOS, Windows | Name, version, vendor, plugin ID, and install path. | ❌ | Fleet scans the shared `Adobe/UXP/extensions` directory. Plugins that Creative Cloud installs under `Adobe/UXP/PluginsStorage` haven't been verified. |
| Adobe native plug-ins for Photoshop, Premiere Pro, After Effects, and Illustrator | ❌ Not collected | — | ❌ | Reported only by a deep-scan mode Fleet doesn't ingest. Native plug-ins have no manifest, so they have no version. |
| Adobe native plug-ins for Acrobat, InDesign, Lightroom Classic, and Substance 3D, and MediaCore shared plug-ins | ❌ Not collected | — | ❌ | Fleet doesn't scan these locations. MediaCore is where third-party effects such as Boris FX, Red Giant, Sapphire, and Neat Video install. |
| Xcode | macOS, as an app | Collected under Apps, with everything a macOS app reports. | ✅ As an app. | Xcode source editor extensions ship inside apps that already appear under Apps. Xcode has no separate plugin system. |
| Figma | ❌ Not collected | — | ❌ | Plugins are tied to the user's Figma account and run from Figma's servers. Nothing is installed on the device. |
| Sketch, Affinity Photo, Designer, and Publisher, GIMP, Inkscape, Krita, Blender, darktable, OFX plug-ins for DaVinci Resolve, Nuke, and Natron, Final Cut Pro and Motion (FxPlug and Motion templates), audio plug-ins (AU, VST/VST3, AAX, and LV2), Autodesk Maya, 3ds Max, Cinema 4D, and Houdini, Unity, Unreal Engine, and Godot, AutoCAD, SolidWorks, Microsoft Office add-ins, Obsidian, LibreOffice and OpenOffice, Notepad++, Android SDK, and Arduino SDK | ❌ Not collected | — | ❌ | Fleet doesn't read them. Affinity loads Photoshop-compatible plug-ins from a folder the user chooses. Unity and Godot plugins are installed per project, not per host. Notepad++ has no native macOS or Linux build. |

- Adobe applications (Photoshop, Acrobat, Substance 3D, and others) appear under Apps. That's also where Fleet matches Adobe vulnerabilities. Learn more in the [adobe_plugins](https://github.com/fleetdm/fleet/tree/main/orbit/pkg/table/adobe_plugins) reference.

## Binaries, AI tools, and other

| Type | Collected on | Data collected | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- |
| Go binaries | macOS, Windows, Linux | Name, version, and install path. | ✅ When the NVD has a matching entry. | Collected from each user's `~/go/bin` directory. Needs Fleet's agent. Learn more in the [go_binaries](https://github.com/fleetdm/fleet/tree/main/orbit/pkg/table/go_binaries) reference. |
| AI desktop apps, agent CLIs, MCP servers, and agent instruction files | ❌ Not in inventory | — | ❌ | Fleet's agent includes an [ai_tools](https://github.com/fleetdm/fleet/blob/main/orbit/pkg/table/ai_tools/README.md) table you can report on, but its results don't feed software inventory yet. |
| macOS widgets | macOS, as part of their app | Collected under Apps, with everything a macOS app reports. | ✅ As an app. | Widgets are WidgetKit extensions that ship inside apps that already appear under Apps. |
| Shortcuts and Android ringtones | ❌ Not collected | — | ❌ | No MDM API exposes them. |

## Data collected

Fleet stores a name and a version for every item in inventory. What else it stores depends on the software type and the platform.

| Data | Collected |
| --- | --- |
| Name (title) and version | ✅ Every software type. |
| Publisher (vendor) | ✅ Windows programs, rpm packages, IDE extensions, and Adobe plugins. macOS apps report the Apple Developer Team ID from the app's code signature instead. deb packages report no publisher. |
| Unique identifier | ✅ Bundle ID (macOS, iOS, and iPadOS), application ID (Android), upgrade code (Windows apps only), and extension ID (browser extensions, VS Code extensions, and Adobe plugins). |
| Install path | ✅ Apps, browser extensions, IDE extensions, plugins, Go binaries, and Homebrew, Chocolatey, Python, and npm packages. Linux system packages (deb, rpm, pacman, and Portage) and ChromeOS extensions don't report a path. |
| File hashes | ✅ macOS apps only. fleetd collects the code directory hash (cdhash) and the SHA-256 hash of the app's executable. Not collected on Windows or Linux, or for packages, extensions, and plugins. |
| Last opened | ✅ macOS apps, Windows programs, and deb and rpm packages. No other software type reports usage. |
| Architecture and release | ✅ rpm packages report both. pacman packages report architecture. No other software type reports either. |
| Install count | ✅ The number of hosts in your Fleet running each title and each version. |
| Vulnerabilities (CVEs) | ✅ For the software types marked above. Fleet Premium adds the CVE publish date, CVSS score, EPSS probability, and CISA KEV status. |
| Title and version release dates | ❌ Not collected. Fleet reports the version a host is running, not when that version shipped. |
| Popularity outside your Fleet | ❌ Not collected. Install counts cover your hosts only. |
| Publisher reputation | ❌ Not collected. Fleet reports the publisher name, and the Team ID on macOS, but not a publisher's first release date or install totals across their other titles. |

## How Fleet collects software

- macOS, Windows, Linux, and ChromeOS: [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd) runs software inventory reports ([macOS example](https://fleetdm.com/reports/get-installed-mac-os-software)). Fleet refreshes software inventory every hour like other host details ([configurable](#configuration)).
- iOS and iPadOS: Fleet asks the device for its installed apps over MDM, using Apple's [InstalledApplicationList command](https://developer.apple.com/documentation/devicemanagement/installedapplicationlistcommand).
- Android: Fleet reads app reports from the [Android Management API](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices).

## Limitations

Fleet collects software from known package databases and install locations. Software installed outside of those, such as tarballs extracted to `/opt`, portable Windows executables, and AppImage files, isn't collected.

Fleet collects a name and version for each item. What else it collects varies by software type. See [Data collected](#data-collected).

On personally-owned (BYOD) iOS, iPadOS, and Android hosts, the platform limits what MDM can see. Fleet reports only Fleet-installed apps (Apple) or work-profile apps (Android). This is a platform restriction, not a Fleet setting.

On Linux, Fleet only collects software from distributions it recognizes. See [Linux distributions](#linux-distributions) above.

## Configuration

Software inventory is on by default for all fleets. To turn it off, set `enable_software_inventory` to `false` in the `features` section of your GitOps configuration. You can set this for all fleets (`org_settings`) or for a specific fleet (`settings`).

```yaml
org_settings:
  features:
    enable_software_inventory: false
```

Fleet refreshes software inventory on the same schedule as other host details. To change the interval, set `FLEET_OSQUERY_DETAIL_UPDATE_INTERVAL` on the Fleet server (default: `1h`). Lowering this value increases load on the Fleet server. Learn more in the [Fleet server configuration](https://fleetdm.com/docs/configuration/fleet-server-configuration#osquery-detail-update-interval) reference.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="karmine05">
<meta name="authorFullName" value="Dhruv Majumdar">
<meta name="publishedOn" value="2026-09-11">
<meta name="articleTitle" value="Software inventory">
<meta name="description" value="Find out how Fleet collects software inventory and what software it covers on each platform.">