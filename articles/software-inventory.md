# Software inventory

Software inventory in Fleet collects the apps, operating systems, browser extensions, packages, IDE extensions, plugins, and binaries installed on your hosts. Vulnerability processing runs against this inventory, so software that isn't collected here can't be checked for vulnerabilities.

To see what software is covered, check out the [Coverage section](#coverage). To see which of that software Fleet checks for vulnerabilities, see [Vulnerability processing](https://fleetdm.com/guides/vulnerability-processing).

## How Fleet collects software

- macOS, Windows, Linux, and ChromeOS: Fleet's agent (fleetd) runs osquery queries and reports the results to the Fleet server. Fleet refreshes software inventory on the same schedule as other host details (default: every hour).
- iOS and iPadOS: Fleet asks the device for its installed apps over MDM.
- Android: Fleet reads app reports from the Android Management API.

Software inventory is on by default. See [Configuration](#configuration) to turn it off or change the refresh interval.

## Coverage

Fleet collects software inventory for these software types. Items marked with an asterisk have a caveat in the [Details](#details) section.

| Symbol | Meaning |
| ------ | ------- |
| ✅ | Collected |
| ✅\* | Collected, with a caveat (see Details) |
| ❌ | Not collected (see Details) |
| ❔ | Not yet verified |
| N/A | Not applicable on this platform |

| Category | Type | macOS | Windows | Linux | ChromeOS | Android | iOS/iPadOS |
| -------- | ---- | ----- | ------- | ----- | -------- | ------- | ---------- |
| Apps | Native apps | ✅ | ✅ | On Linux, apps are installed as "Packages". | ❌ | ✅\* | ✅\* |
| | Microsoft Store (MSIX/Appx) apps | N/A | ✅\* | N/A | N/A | N/A | N/A |
| | Alternative app store and sideloaded apps | N/A | N/A | N/A | N/A | ❔ | ❔ |
| | Personal-side apps on BYOD hosts | N/A | N/A | N/A | N/A | ❌ | ❌ |
| Operating system | macOS | ✅ | N/A | N/A | N/A | N/A | N/A |
| | Windows 10, 11, and Server | N/A | ✅ | N/A | N/A | N/A | N/A |
| | ChromeOS | N/A | N/A | N/A | ✅ | N/A | N/A |
| | Android | N/A | N/A | N/A | N/A | ✅ | N/A |
| | iOS and iPadOS | N/A | N/A | N/A | N/A | N/A | ✅ |
| | Ubuntu | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Debian | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Red Hat Enterprise Linux (RHEL) | N/A | N/A | ✅ | N/A | N/A | N/A |
| | CentOS and CentOS Stream | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Fedora | N/A | N/A | ✅\* | N/A | N/A | N/A |
| | Rocky Linux | N/A | N/A | ✅\* | N/A | N/A | N/A |
| | AlmaLinux | N/A | N/A | ✅\* | N/A | N/A | N/A |
| | Oracle Linux | N/A | N/A | ✅\* | N/A | N/A | N/A |
| | Amazon Linux | N/A | N/A | ✅ | N/A | N/A | N/A |
| | SLES | N/A | N/A | ✅ | N/A | N/A | N/A |
| | openSUSE Leap | N/A | N/A | ✅ | N/A | N/A | N/A |
| | openSUSE Tumbleweed | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Arch Linux | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Arch Linux ARM | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Omarchy | N/A | N/A | ✅ | N/A | N/A | N/A |
| | CachyOS | N/A | N/A | ✅ | N/A | N/A | N/A |
| | EndeavourOS | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Manjaro | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Manjaro ARM | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Gentoo | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Kali | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Pop!_OS | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Linux Mint | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Zorin OS | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Void | N/A | N/A | ✅ | N/A | N/A | N/A |
| | NixOS | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Tuxedo OS | N/A | N/A | ✅ | N/A | N/A | N/A |
| | KDE neon | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Flatcar | N/A | N/A | ✅ | N/A | N/A | N/A |
| | CoreOS | N/A | N/A | ✅ | N/A | N/A | N/A |
| | AMD Ryzen AI Developer Platform | N/A | N/A | ✅ | N/A | N/A | N/A |
| | NVIDIA DGX OS (DGX Spark, DGX Station, and DGX servers) | N/A | N/A | ✅\* | N/A | N/A | N/A |
| | Raspberry Pi OS 64-bit | N/A | N/A | ✅\* | N/A | N/A | N/A |
| | Raspberry Pi OS 32-bit | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Alpine | N/A | N/A | ❌ | N/A | N/A | N/A |
| | elementary OS | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Deepin | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Garuda | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Clear Linux | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Photon OS | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Solus | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Parrot OS | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Vanilla OS | N/A | N/A | ❌ | N/A | N/A | N/A |
| | openSUSE MicroOS and Aeon | N/A | N/A | ❌ | N/A | N/A | N/A |
| | SteamOS | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Bazzite, Bluefin, and Aurora | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Ubuntu Core | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Any other distribution | N/A | N/A | ❌ | N/A | N/A | N/A |
| Browser extensions | Chrome | ✅ | ✅ | ✅ | ✅ | N/A | N/A |
| | Chrome Beta, Dev, and Canary | ✅\* | ✅\* | ✅\* | N/A | N/A | N/A |
| | Chromium | ✅ | ✅ | ✅ | N/A | N/A | N/A |
| | Brave | ✅ | ✅ | ✅ | N/A | N/A | N/A |
| | Brave Beta and Nightly | ✅\* | ✅\* | ✅\* | N/A | N/A | N/A |
| | Edge | ✅ | ✅ | ✅ | N/A | ❌ | N/A |
| | Edge Beta | ✅ | ✅ | ✅ | N/A | N/A | N/A |
| | Edge Dev and Canary | ✅\* | ✅\* | ✅\* | N/A | N/A | N/A |
| | Opera | ✅ | ✅ | ✅ | N/A | N/A | N/A |
| | Yandex | ✅ | ✅ | ✅ | N/A | ❌ | N/A |
| | Vivaldi | ✅\* | ✅\* | ✅\* | N/A | N/A | N/A |
| | Arc | ✅\* | ❌ | N/A | N/A | N/A | N/A |
| | Perplexity Comet and Dia | ❌ | ❌ | N/A | N/A | N/A | N/A |
| | Firefox | ✅ | ✅ | ✅\* | N/A | ❌ | N/A |
| | Safari | ✅\* | N/A | N/A | N/A | N/A | ❌ |
| | Internet Explorer | N/A | ✅\* | N/A | N/A | N/A | N/A |
| Packages | Homebrew | ✅\* | N/A | ❌ | N/A | N/A | N/A |
| | Chocolatey | N/A | ✅ | N/A | N/A | N/A | N/A |
| | Scoop | N/A | ❌ | N/A | N/A | N/A | N/A |
| | winget portable packages | N/A | ❌ | N/A | N/A | N/A | N/A |
| | deb | N/A | N/A | ✅ | N/A | N/A | N/A |
| | rpm | N/A | N/A | ✅ | N/A | N/A | N/A |
| | pacman | N/A | N/A | ✅ | N/A | N/A | N/A |
| | Portage | N/A | N/A | ✅ | N/A | N/A | N/A |
| | snap | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Flatpak | N/A | N/A | ❌ | N/A | N/A | N/A |
| | AppImage | N/A | N/A | ❌ | N/A | N/A | N/A |
| | Nix | ❌ | N/A | ❌ | N/A | N/A | N/A |
| | Python | ✅\* | ✅\* | ✅\* | N/A | N/A | N/A |
| | npm | ✅\* | ❌ | ✅\* | N/A | N/A | N/A |
| | pnpm, Yarn, and Bun (global) | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | RubyGems | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Rust (cargo) | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Nim | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Atom | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| IDE extensions | VS Code, VS Code Insiders, VSCodium, VSCodium Insiders, Cursor, Windsurf (Devin), and Trae | ✅ | ✅ | ✅ | N/A | N/A | N/A |
| | Google Antigravity, Kiro, and code-server | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | JetBrains: CLion, DataGrip, GoLand, IntelliJ IDEA (and Community Edition), PhpStorm, PyCharm (and Community Edition), ReSharper, Rider, RubyMine, RustRover, and WebStorm | ✅ | ✅ | ✅ | N/A | N/A | N/A |
| | JetBrains: Android Studio, DataSpell, Aqua, and Writerside | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Zed, Sublime Text, Neovim/Vim, and Emacs | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| Adobe plugins | CEP extensions for Photoshop, Illustrator, Premiere Pro, After Effects, InDesign, InCopy, Animate, Dreamweaver, Audition, Bridge, Lightroom, Lightroom Classic, XD, and Prelude | ✅\* | ✅\* | N/A | N/A | N/A | N/A |
| | UXP plugins for Photoshop, XD, InDesign, InCopy, and Premiere Pro | ✅\* | ✅\* | N/A | N/A | N/A | N/A |
| | Native plug-ins for Photoshop, Premiere Pro, After Effects, and Illustrator | ❌ | ❌ | N/A | N/A | N/A | N/A |
| | Native plug-ins for Acrobat, InDesign, Lightroom Classic, and Substance 3D | ❌ | ❌ | N/A | N/A | N/A | N/A |
| | MediaCore shared plug-ins (third-party effects for Premiere Pro, After Effects, and Audition) | ❌ | ❌ | N/A | N/A | N/A | N/A |
| Design and creative plugins | Sketch | ❌ | N/A | N/A | N/A | N/A | N/A |
| | Figma | N/A | N/A | N/A | N/A | N/A | N/A |
| | Affinity Photo, Designer, and Publisher | ❌ | ❌ | N/A | N/A | N/A | N/A |
| | GIMP | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Inkscape | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Krita | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Blender | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | darktable | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | OFX plug-ins for DaVinci Resolve, Nuke, and Natron | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Final Cut Pro and Motion (FxPlug and Motion templates) | ❌ | N/A | N/A | N/A | N/A | N/A |
| | Audio plug-ins (AU, VST/VST3, AAX, and LV2) for Logic Pro, Ableton Live, Pro Tools, and Reaper | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Autodesk Maya, 3ds Max, Cinema 4D, and Houdini | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Unity, Unreal Engine, and Godot | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | AutoCAD | ❌ | ❌ | N/A | N/A | N/A | N/A |
| | SolidWorks | N/A | ❌ | N/A | N/A | N/A | N/A |
| Productivity and other plugins | Microsoft Office add-ins (Excel, Word, Outlook, and PowerPoint) | ❌ | ❌ | N/A | N/A | N/A | N/A |
| | Xcode | ✅\* | N/A | N/A | N/A | N/A | N/A |
| | Obsidian | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | LibreOffice and OpenOffice | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| | Notepad++ (Windows only) | N/A | ❌ | N/A | N/A | N/A | N/A |
| | Android SDK and Arduino SDK | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| Binaries | Go | ✅\* | ✅\* | ✅\* | N/A | N/A | N/A |
| AI tools | AI desktop apps, agent CLIs, MCP servers, and agent instruction files | ❌ | ❌ | ❌ | N/A | N/A | N/A |
| Other | macOS widgets | ✅\* | N/A | N/A | N/A | N/A | N/A |
| | Shortcuts | ❌ | N/A | N/A | N/A | N/A | ❌ |
| | Ringtones | N/A | N/A | N/A | N/A | ❌ | N/A |

> If Fleet isn't collecting software that's installed on your hosts, please file a [feature request](https://github.com/fleetdm/fleet/issues/new?template=feature-request.md). If Fleet is collecting software incorrectly (wrong name, version, or type), please file a [bug](https://github.com/fleetdm/fleet/issues/new?template=bug-report.md).

### Details

#### Apps

- Native apps on ChromeOS: Fleet's ChromeOS agent runs as a browser extension. It has no API for installed Android apps or progressive web apps (PWAs).
- Native apps on Android: personally-owned (BYOD) hosts report apps in the work profile only. Fully-managed hosts report all installed apps.
- Native apps on iOS and iPadOS: personally-owned (BYOD) hosts report only the apps Fleet installed. Built-in and user-installed apps aren't included. Learn more in [Enrolling BYOD iPad/iOS devices](https://fleetdm.com/guides/enroll-byod-ios-ipados-hosts).
- Microsoft Store apps: requires osquery 5.17 or later for per-user packages and 5.22.1 or later for machine-wide provisioned packages. Hosts on fleetd's `stable` channel receive osquery updates automatically.
- Alternative app store and sideloaded apps: these should appear in the same MDM app lists on fully-managed hosts. Fleet hasn't verified this yet.
- Personal-side apps on BYOD hosts: Apple User Enrollment and Android work profiles don't expose them.

#### Operating system

- Fedora, Rocky Linux, AlmaLinux, and Oracle Linux report as RHEL because they ship `/etc/redhat-release`. Fleet shows them under RHEL on the **Software** > **OS** page.
- NVIDIA DGX OS reports as Ubuntu (DGX OS 7 as Ubuntu 24.04, DGX OS 6 as Ubuntu 22.04) because its `/etc/os-release` is Ubuntu's. Fleet doesn't read `/etc/dgx-release`, so the host appears as Ubuntu on the **Software** > **OS** page and the DGX OS version isn't reported. DGX Spark is arm64; use the `arm64` fleetd package.
- Raspberry Pi OS 64-bit reports as Debian and is covered. Raspberry Pi OS 32-bit reports as `raspbian`, which Fleet doesn't recognize.
- Alpine, elementary OS, Deepin, Garuda, Clear Linux, Photon OS, Solus, Parrot OS, Vanilla OS, openSUSE MicroOS, Aeon, and any distribution not listed above: Fleet recognizes Linux by matching the platform value osquery reads from `/etc/os-release` against a [fixed list](https://github.com/fleetdm/fleet/blob/main/server/fleet/hosts.go). Hosts with any other value enroll, but Fleet doesn't collect an OS entry or software inventory for them.
- SteamOS, Bazzite, Bluefin, and Aurora: these use immutable root filesystems. Installing Fleet's agent is unsupported, and their platform values aren't in Fleet's list.
- Ubuntu Core runs snaps only. Fleet's agent isn't packaged as a snap, so Ubuntu Core hosts can't enroll.

#### Browser extensions

- Chrome Beta, Dev, and Canary, Brave Beta and Nightly, Edge Dev and Canary, Vivaldi, and Arc: osquery 5.15 or later collects these and Fleet stores them, but the **Type** column shows the raw browser value (for example, "Chrome Beta" or "Vivaldi") because Fleet doesn't have a display name for them yet. Arc on Windows isn't collected.
- Perplexity Comet and Dia: support was added to osquery after 5.23.1. Fleet will collect them once the next osquery release ships in fleetd.
- Firefox on Linux: the snap profile location is collected. The Flatpak profile location isn't.
- Safari and Internet Explorer: osquery reports an identifier for each extension, but Fleet doesn't store it yet. Only the name and version are collected.
- Fleet doesn't hash browser extensions on any platform.
- Safari on iOS and Firefox, Edge, and Yandex on Android support extensions, but no MDM API exposes them.

#### Packages

- Homebrew on macOS: Fleet collects formulae, plus casks that don't install a `.app` bundle (those appear under Apps). Homebrew on Linux isn't collected because osquery's `homebrew_packages` table only runs on macOS.
- Scoop, winget portable packages, snap, Flatpak, AppImage, Nix, pnpm, Yarn, Bun, RubyGems, cargo, and Nim: no osquery or fleetd table reads them. On Ubuntu desktop, Firefox and Chromium are installed as snaps by default, so those browsers are missing from inventory even though their extensions are collected. Follow the snap request in [#22658](https://github.com/fleetdm/fleet/issues/22658).
- Python: with osquery 5.23 or later, Fleet collects system and per-user site-packages, plus packages installed with pipx, uv tools, conda, pipenv, pyenv, and mise. Packages inside project-level `.venv` directories aren't collected.
- npm: Fleet collects globally installed packages, including nvm, fnm, asdf, mise, and Volta locations. Scoped (`@org/name`) and nested packages require osquery 5.23 or later. On Windows, osquery supports npm packages but Fleet doesn't query them yet. Project-level `node_modules` directories aren't collected.
- Atom packages were removed from inventory in December 2023 after the editor was sunset.

#### IDE extensions

- VS Code and its forks: osquery reads the extension directories for VS Code, VS Code Insiders, VSCodium, VSCodium Insiders, Cursor, Windsurf (including its Devin rename), and Trae, plus their remote-server variants. Learn more in the [vscode_extensions](https://fleetdm.com/tables/vscode_extensions) table. Google Antigravity, Kiro, and code-server use the same format but aren't in osquery's list yet.
- JetBrains: osquery enumerates 13 products installed under the `JetBrains` directory. Learn more in the [jetbrains_plugins](https://fleetdm.com/tables/jetbrains_plugins) table. Android Studio installs under a `Google` directory and isn't collected. DataSpell, Aqua, and Writerside aren't in osquery's list.
- Zed, Sublime Text, Neovim/Vim, and Emacs: no osquery or fleetd table reads them.

#### Plugins

- Adobe applications (Photoshop, Acrobat, Substance 3D, and others) appear under Apps. That's also where Fleet matches Adobe vulnerabilities. Adobe doesn't ship applications for Linux.
- Adobe CEP extensions: Fleet collects the name, version, and vendor. The host application is read from the manifest but not stored, so the **Type** column shows "Plugin (Adobe)" for every host application. Learn more in the [adobe_plugins](https://github.com/fleetdm/fleet/tree/main/orbit/pkg/table/adobe_plugins) table.
- Adobe UXP plugins: Fleet scans the shared `Adobe/UXP/extensions` directory. Plugins that Creative Cloud installs under `Adobe/UXP/PluginsStorage` haven't been verified.
- Adobe native plug-ins for Photoshop, Premiere Pro, After Effects, and Illustrator: the `adobe_plugins` table can report these in its deep-scan mode, but Fleet doesn't ingest that mode. Native plug-ins don't have a manifest, so they have no version.
- Adobe native plug-ins for Acrobat, InDesign, Lightroom Classic, and Substance 3D, and MediaCore shared plug-ins: the `adobe_plugins` table doesn't scan these locations. MediaCore is where third-party effects such as Boris FX, Red Giant, Sapphire, and Neat Video install.
- Figma plugins are tied to the user's Figma account and run from Figma's servers. Nothing is installed on the device to collect.
- Sketch, Affinity, GIMP, Inkscape, Krita, Blender, darktable, OFX, FxPlug, audio plug-ins, 3D suites, game engines, AutoCAD, SolidWorks, Microsoft Office add-ins, Obsidian, LibreOffice, OpenOffice, Notepad++, Android SDK, and Arduino SDK: no osquery or fleetd table reads them. Affinity loads Photoshop-compatible plug-ins from a folder the user chooses. Unity and Godot plugins are installed per project, not per host. Notepad++ has no native macOS or Linux build.
- Xcode source editor extensions ship inside apps that already appear under Apps. Xcode has no separate plugin system.

#### Binaries, AI tools, and other

- Go binaries: Fleet collects binaries in each user's `~/go/bin` directory. Learn more in the [go_binaries](https://github.com/fleetdm/fleet/tree/main/orbit/pkg/table/go_binaries) table.
- AI tools: fleetd includes an [ai_tools](https://github.com/fleetdm/fleet/blob/main/orbit/pkg/table/ai_tools/README.md) table that you can query, but Fleet doesn't add its results to software inventory yet.
- macOS widgets are WidgetKit extensions that ship inside apps that already appear under Apps.
- Shortcuts and Android ringtones: no MDM API exposes them.

## Limitations

Fleet collects software from known package databases and install locations. Software installed outside of those, such as tarballs extracted to `/opt`, portable Windows executables, and AppImage files, isn't collected.

Fleet collects a name and version for each item. It doesn't collect file hashes for apps, packages, or extensions.

On personally-owned (BYOD) iOS, iPadOS, and Android hosts, the platform limits what MDM can see. Fleet reports only Fleet-installed apps (Apple) or work-profile apps (Android). This is a platform restriction, not a Fleet setting.

On Linux, Fleet only collects software from distributions it recognizes. See the [Operating system](#operating-system) details above.

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