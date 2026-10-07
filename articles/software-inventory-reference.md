# Software inventory reference

Software inventory in Fleet collects the apps, operating systems, browser extensions, packages, IDE extensions, plugins, and binaries installed on your hosts. [Vulnerability (CVE) processing](https://fleetdm.com/guides/vulnerability-processing#coverage) runs against this inventory, so software that isn't collected here can't be checked for vulnerabilities.

Each software type below lists its `source`, and browser and IDE extensions also list `extension_for` (the browser or IDE the extension belongs to). Use these values to filter software in [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#list-software).

## Apps

| Type | Name | Version | Vendor | Identifier | Install path | File hashes | Last opened | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| macOS app (`source`: `apps`) | ✅ | ✅ | ✅ Apple Developer Team ID | ✅ Bundle ID | ✅ | ✅ cdhash and executable SHA-256 | ✅ | ✅ | — |
| Windows app (`source`: `programs`) | ✅ | ✅ | ✅ | ✅ Upgrade code | ✅ | ❌ | ✅ | ✅ | Includes Microsoft Store (MSIX/Appx) apps on recent versions of Fleet's agent. |
| Linux apps (`source`: `deb_packages`, `rpm_packages`) | ✅ | ✅ | ✅ rpm only | ❌ | ❌ | ❌ | ✅ deb and rpm | ✅ | Apps install as [packages](#packages) on Linux. |
| Android app (`source`: `android_apps`) | ✅ | ✅ | ❌ | ✅ Application ID | ❌ | ❌ | ❌ | ❌ | BYOD hosts report work profile apps only. Fully-managed hosts report all apps. |
| iOS app (`source`: `ios_apps`) and iPadOS app (`source`: `ipados_apps`) | ✅ | ✅ | ❌ | ✅ Bundle ID | ❌ | ❌ | ❌ | ❌ | BYOD hosts report only the apps Fleet installed. |
| ChromeOS: browser extensions (`source`: `chrome_extensions`, `extension_for`: `chrome`) | ✅ | ✅ | ❌ | ✅ Extension ID | ❌ | ❌ | ❌ | ✅ | Collected under [browser extensions](#browser-extensions). |
| ChromeOS: progressive web apps (PWAs) and Isolated Web Apps (IWAs) | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | No extension API returns installed web apps. |
| ChromeOS: Android apps | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Android apps run in a container the browser can't see. Distinct from the Android apps row above, which covers Android hosts. |
| ChromeOS: Linux apps and packages (Crostini) | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Crostini runs in a separate Linux container, outside the browser. |
| ChromeOS: Steam games (Borealis) and Parallels Desktop Windows apps | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Both run in separate containers or virtual machines, outside the browser. |
| Personal-side apps on BYOD hosts | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Apple User Enrollment and Android work profiles don't expose them. |

- Alternative app store and sideloaded apps should appear in the same MDM app lists on fully-managed hosts. Fleet hasn't verified this yet.
- Native apps on iOS and iPadOS: built-in and user-installed apps aren't included on BYOD hosts. Learn more in [Enrolling BYOD iPad/iOS devices](https://fleetdm.com/guides/enroll-byod-ios-ipados-hosts).
- On ChromeOS, Fleet's agent runs as a browser extension, so it only sees software installed in the browser. Collecting web apps, Android apps, and software in Linux containers would need an integration with Google's [Chrome Management API](https://developers.google.com/chrome/management/guides/reports_api) instead of the agent.
- Chrome Apps aren't listed above because Google is removing them from ChromeOS. Support for user-installed Chrome Apps ended in July 2025, and kiosk mode ended in July 2026. Learn more in [End of support for Chrome apps](https://support.google.com/chrome/a/answer/15950395).

## Operating systems

| OS | Name | Version | Architecture | Kernel version | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- |
| macOS | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| Windows 10, 11, and Server | ✅ | ✅ | ✅ | ✅ | ✅ Matched against Microsoft security bulletins. | Also reports display version, such as 23H2, and installation type. |
| Linux | ✅ | ✅ | ✅ | ✅ | ✅ Kernel vulnerabilities, on supported distributions. | Collected on recognized distributions only. See the table below. |
| ChromeOS | ✅ | ✅ | ✅ | ✅ | ❌ | The kernel version repeats the OS version. |
| Android | ✅ | ✅ | ❌ | ❌ | ✅ Matched against the Android security patch level. | — |
| iOS and iPadOS | ✅ | ✅ | ❌ | ❌ | ❌ | — |

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

| Browser | Collected on | Name | Version | Vendor | Extension ID | Install path | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Chrome extension (`source`: `chrome_extensions`, `extension_for`: `chrome`) | macOS, Windows, Linux, ChromeOS | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ChromeOS reports no install path. |
| Chromium, Brave, Edge, Edge Beta, Opera, and Yandex extensions (`source`: `chrome_extensions`, `extension_for`: `chromium`, `brave`, `edge`, `edge_beta`, `opera`, `yandex`) | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | — |
| Chrome Beta, Dev, and Canary, Brave Beta and Nightly, Edge Dev and Canary, and Vivaldi (`source`: `chrome_extensions`) | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | The **Type** column shows the raw browser value, such as "Chrome Beta". You can filter by `source`: `chrome_extensions`. Filtering by these browsers individually (`extension_for`) is [coming soon](https://github.com/fleetdm/fleet/issues/54596). |
| Arc (`source`: `chrome_extensions`) | macOS | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | Not collected on Windows. Shows the raw browser value. You can filter by `source`: `chrome_extensions`. Filtering by these browsers individually (`extension_for`) is [coming soon](https://github.com/fleetdm/fleet/issues/54596). |
| Firefox extension (`source`: `firefox_addons`, `extension_for`: `firefox`) | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | On Linux, the snap profile is collected and the Flatpak profile isn't. |
| Safari extension (`source`: `safari_extensions`) | macOS | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ When the NVD has a matching entry. | Fleet doesn't store the extension identifier yet. |
| Internet Explorer extension (`source`: `ie_extensions`) | Windows | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ When the NVD has a matching entry. | Fleet doesn't store the extension identifier yet. |
| Perplexity Comet and Dia | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Support ships in an upcoming version of Fleet's agent. |

- Fleet doesn't hash browser extensions on any platform, and none of them report a last opened time.
- Safari on iOS and Firefox, Edge, and Yandex on Android support extensions, but no MDM API exposes them.

## Packages

| Type | Collected on | Name | Version | Vendor | Install path | Last opened | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Homebrew package (`source`: `homebrew_packages`) | macOS | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | Formulae, plus casks that don't install a `.app` bundle. Those appear under Apps. Homebrew on Linux isn't collected. |
| Chocolatey package (`source`: `chocolatey_packages`) | Windows | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | — |
| deb package (`source`: `deb_packages`) | Linux | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ Ubuntu. Debian covers the kernel only. | — |
| RPM package (`source`: `rpm_packages`) | Linux | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ RHEL, CentOS, Fedora, and Amazon Linux. | Also reports architecture and release. |
| pacman package (`source`: `pacman_packages`) | Linux | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ Matched against the NVD by version, which can produce false positives. | Needs Fleet's agent. Also reports architecture. |
| Portage package (`source`: `portage_packages`) | Linux | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ When the NVD has a matching entry. | — |
| Python package (`source`: `python_packages`) | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | System and per-user site-packages, plus pipx, uv tools, conda, pipenv, pyenv, and mise. Project-level `.venv` directories aren't collected. |
| npm package (`source`: `npm_packages`) | macOS, Linux | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | Global packages only, including nvm, fnm, asdf, mise, and Volta locations. Windows isn't queried yet. Project-level `node_modules` directories aren't collected. |
| macOS package (.pkg) (`source`: `pkg_packages`), Tarball (.tar.gz) (`source`: `tgz_packages`), Script-only package (.sh) (`source`: `sh_packages`), Script-only package (.py) (`source`: `py_packages`), and Script-only package (.ps1) (`source`: `ps1_packages`) | ❌ Not collected from hosts | N/A | N/A | N/A | N/A | N/A | N/A | Fleet sets these types for custom packages added to the software library, not for software it finds on hosts. A `.pkg` that installs an app appears as a macOS app instead. |
| Scoop, winget portable packages, snap, Flatpak, AppImage, Nix, pnpm, Yarn, Bun, RubyGems, cargo, and Nim | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet doesn't read them. |
| Atom | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Removed from inventory in December 2023 after the editor was sunset. |

- On Ubuntu desktop, Firefox and Chromium install as snaps by default, so those browsers are missing from inventory even though their extensions are collected. Follow the snap request in [#22658](https://github.com/fleetdm/fleet/issues/22658).

## IDE extensions

| Type | Collected on | Name | Version | Vendor | Extension ID | Install path | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| VS Code, VS Code Insiders, VSCodium, VSCodium Insiders, Cursor, Windsurf, and Trae extensions (`source`: `vscode_extensions`, `extension_for`: `vscode`, `vscode_insiders`, `vscodium`, `vscodium_insiders`, `cursor`, `windsurf`, `trae`) | macOS, Windows, Linux | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Remote-server variants are collected too. Learn more in the [vscode_extensions](https://fleetdm.com/tables/vscode_extensions) reference. |
| JetBrains: CLion, DataGrip, GoLand, IntelliJ IDEA, IntelliJ IDEA Community Edition, PhpStorm, PyCharm, PyCharm Community Edition, ReSharper, Rider, RubyMine, RustRover, and WebStorm extensions (`source`: `jetbrains_plugins`, `extension_for`: `CLion`, `DataGrip`, `GoLand`, `IntelliJIdea`, `IntelliJIdeaCommunityEdition`, `PhpStorm`, `PyCharm`, `PyCharmCommunityEdition`, `ReSharper`, `Rider`, `RubyMine`, `RustRover`, `WebStorm`) | macOS, Windows, Linux | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | Fleet reads 13 products installed under the `JetBrains` directory. Learn more in the [jetbrains_plugins](https://fleetdm.com/tables/jetbrains_plugins) reference. |
| Google Antigravity, Kiro, and code-server | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | These use the VS Code extension format, but Fleet doesn't read their directories yet. |
| JetBrains: Android Studio, DataSpell, Aqua, and Writerside | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Android Studio installs under a `Google` directory. The rest aren't in Fleet's list. |
| Zed, Sublime Text, Neovim/Vim, and Emacs | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet doesn't read them. |

## Plugins

| Type | Collected on | Name | Version | Vendor | Plugin ID | Install path | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Adobe plugin (`source`: `adobe_plugins`): CEP extensions for Photoshop, Illustrator, Premiere Pro, After Effects, InDesign, InCopy, Animate, Dreamweaver, Audition, Bridge, Lightroom, Lightroom Classic, XD, and Prelude | macOS, Windows | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ Adobe files CVEs against the host application, which Fleet detects under Apps. | Adobe ships no Linux applications. The **Type** column shows "Plugin (Adobe)" for every host application, because the host application is read from the manifest but not stored. |
| Adobe plugin (`source`: `adobe_plugins`): UXP plugins for Photoshop, XD, InDesign, InCopy, and Premiere Pro | macOS, Windows | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | Fleet scans the shared `Adobe/UXP/extensions` directory. Plugins that Creative Cloud installs under `Adobe/UXP/PluginsStorage` haven't been verified. |
| Adobe native plug-ins for Photoshop, Premiere Pro, After Effects, and Illustrator | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Reported only by a deep-scan mode Fleet doesn't ingest. Native plug-ins have no manifest, so they have no version. |
| Adobe native plug-ins for Acrobat, InDesign, Lightroom Classic, and Substance 3D, and MediaCore shared plug-ins | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet doesn't scan these locations. MediaCore is where third-party effects such as Boris FX, Red Giant, Sapphire, and Neat Video install. |
| Xcode (`source`: `apps`) | macOS, as an app | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | Collected under Apps, with everything a macOS app reports. Xcode source editor extensions ship inside apps that already appear under Apps, and Xcode has no separate plugin system. |
| Figma | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Plugins are tied to the user's Figma account and run from Figma's servers. Nothing is installed on the device. |
| Sketch, Affinity Photo, Designer, and Publisher, GIMP, Inkscape, Krita, Blender, darktable, OFX plug-ins for DaVinci Resolve, Nuke, and Natron, Final Cut Pro and Motion (FxPlug and Motion templates), audio plug-ins (AU, VST/VST3, AAX, and LV2), Autodesk Maya, 3ds Max, Cinema 4D, and Houdini, Unity, Unreal Engine, and Godot, AutoCAD, SolidWorks, Microsoft Office add-ins, Obsidian, LibreOffice and OpenOffice, Notepad++, Android SDK, and Arduino SDK | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet doesn't read them. Affinity loads Photoshop-compatible plug-ins from a folder the user chooses. Unity and Godot plugins are installed per project, not per host. Notepad++ has no native macOS or Linux build. |

- Adobe applications (Photoshop, Acrobat, Substance 3D, and others) appear under Apps. That's also where Fleet matches Adobe vulnerabilities. Learn more in the [adobe_plugins](https://github.com/fleetdm/fleet/tree/main/orbit/pkg/table/adobe_plugins) reference.

## AI tools

Fleet Premium marks AI tools in software inventory. To see only AI tools, go to **Software**, select **Filters**, and turn on **AI tools**. Fleet finds AI tools with the [ai_tools](https://github.com/fleetdm/fleet/blob/main/orbit/pkg/table/ai_tools/README.md) table in Fleet's agent.

If Fleet already collects an AI tool as another type, such as a macOS app or an npm package, it keeps that type. Fleet adds three types for AI tools it doesn't collect any other way: **MCP server**, **AI skill**, and **AI CLI tool**.

| Type | Collected on | Name | Version | Vendor | Install path | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- |
| AI desktop apps | macOS, Windows | ✅ | ✅ | ✅ | ✅ | ✅ | Keep their existing type, such as macOS app or Windows app. Covers apps such as ChatGPT, Claude, Cursor, LM Studio, and Ollama. |
| AI mobile apps | iOS/iPadOS, Android | ✅ | ✅ | ❌ | ❌ | ❌ | Collected as iOS, iPadOS, or Android apps, but not marked as AI tools. The `ai_tools` table runs in Fleet's agent, which doesn't run on mobile devices. |
| AI CLI tools | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ Only when installed with a package manager. | CLIs installed with a package manager keep that type, such as npm package, Homebrew package, or Go binary. Learn more in [Packages](#packages) and [Binaries and other](#binaries-and-other). CLIs installed any other way, such as Claude Code's native installer, appear as **AI CLI tool**. |
| MCP servers | macOS, Windows, Linux | ✅ | N/A | ❌ | ✅ | ❌ | Appear as **MCP server**. Most MCP servers don't have a version, because they update on the fly, locally or remotely. The install path is the config file that declares the server. |
| AI skills | macOS, Windows, Linux | ✅ | N/A | N/A | ✅ | N/A | Appear as **AI skill**. Covers two kinds of markdown files that steer AI agents. Agent instructions, such as `AGENTS.md`, `CLAUDE.md`, `.cursorrules`, and `.windsurfrules`, load in every session. Skills, such as Claude Code's `SKILL.md` or a project's `.agents/skills` directory, load only when a task needs them. Skills are [coming soon](https://github.com/fleetdm/fleet/issues/51288). Neither has a version or vulnerabilities. |
| AI IDE extensions | macOS, Windows, Linux | ✅ | ✅ | ✅ | ✅ | ✅ | Keep their [IDE extensions](#ide-extensions) type, such as VS Code extension. |
| AI browser extensions | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ | Keep their [browser extensions](#browser-extensions) type, such as Chrome extension. |
| Live AI and MCP sockets | ❌ Not in inventory | ❌ | ❌ | ❌ | ❌ | ❌ | Open network ports for running AI tools, such as a local model server's API port. They aren't software, so they don't appear in software inventory. You can report on them with the `ai_tools` table's `sockets` type. |
| AI models | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet doesn't collect installed model weights or which model a local runner has loaded. Only the runner app is visible, under AI desktop apps. |

## Binaries and other

| Type | Collected on | Name | Version | Vendor | Install path | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Go binary (`source`: `go_binaries`) | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ When the NVD has a matching entry. | Collected from each user's `~/go/bin` directory. Needs Fleet's agent. Learn more in the [go_binaries](https://github.com/fleetdm/fleet/tree/main/orbit/pkg/table/go_binaries) reference. |
| macOS widgets (`source`: `apps`) | macOS, as part of their app | ✅ | ✅ | ✅ | ✅ | ✅ | Collected under Apps. Widgets are WidgetKit extensions that ship inside apps that already appear there. |
| Shortcuts and Android ringtones | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | No MDM API exposes them. |

## Data collected

The tables above list what Fleet stores for each software type. Two things apply to every type:

- **Install count:** Fleet counts how many hosts in your Fleet run each title and each version.
- **Vulnerability details:** Fleet Premium adds the CVE publish date, CVSS score, EPSS probability, and CISA KEV status.

Fleet doesn't collect these for any software type:

- **Title and version release dates.** Fleet reports the version a host is running, not when that version shipped.
- **Popularity outside your Fleet.** Install counts cover your hosts only.
- **Publisher reputation.** Fleet reports the publisher name, and the Apple Developer Team ID on macOS, but not a publisher's first release date or install totals across their other titles.

## How Fleet collects software

- macOS, Windows, Linux, and ChromeOS: [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd) runs software inventory reports ([macOS example](https://fleetdm.com/reports/get-installed-mac-os-software)). Fleet refreshes software inventory every hour like other host details ([configurable](#configuration)).
- iOS and iPadOS: Fleet asks the device for its installed apps over MDM, using Apple's [InstalledApplicationList command](https://developer.apple.com/documentation/devicemanagement/installedapplicationlistcommand).
- Android: Fleet reads app reports from the [Android Management API](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices).

## Limitations

Fleet collects software from known package databases and install locations. Software installed outside of those, such as tarballs extracted to `/opt`, portable Windows executables, and AppImage files, isn't collected.

Fleet collects a name and version for each item. What else it collects varies by software type, as the tables above show.

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
<meta name="articleTitle" value="Software inventory reference">
<meta name="description" value="Find out how Fleet collects software inventory and what software it covers on each platform.">