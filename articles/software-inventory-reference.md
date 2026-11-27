# Software inventory

## Apps

| Type | Name | Version | Vendor | Identifier | Install path | File hashes | Last opened | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| macOS app (`source`: `apps`) | ✅ | ✅ | ✅ Apple Developer Team ID | ✅ Bundle ID | ✅ | ✅ cdhash and executable SHA-256 | ✅ | ✅ | — |
| Windows app (`source`: `programs`) | ✅ | ✅ | ✅ | ✅ Upgrade code | ✅ | ❌ | ✅ | ✅ | Includes Microsoft Store (MSIX/Appx) apps on recent versions of Fleet's agent. Older Microsoft Office versions (like Office 2013) appear in inventory, but Fleet doesn't detect their vulnerabilities. Learn more in [vulnerability processing](https://fleetdm.com/guides/vulnerability-processing#limitations). |
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
| Windows 10, 11, and Server | ✅ | ✅ | ✅ | ✅ | ✅ Matched against Microsoft security bulletins. | Also reports display version, such as 23H2, and installation type. Windows IoT Enterprise is untested. |
| Linux | ✅ | ✅ | ✅ | ✅ | ✅ Kernel vulnerabilities, on supported distributions. | Collected on recognized distributions only. See the table below. |
| ChromeOS | ✅ | ✅ | ✅ | ✅ | ❌ | The kernel version repeats the OS version. |
| Android | ✅ | ✅ | ❌ | ✅ | ✅ Matched against the Android security patch level. | — |
| iOS and iPadOS | ✅ | ✅ | ❌ | ❌ | ❌ | — |
| tvOS | ❌ | ❌ | ❌ | ❌ | ❌ | Enrolling Apple TV hosts is [coming soon](https://github.com/fleetdm/fleet/issues/38791). |
| Fire OS, and Android (AOSP) desk phones and room devices | ❌ | ❌ | ❌ | ❌ | ❌ | These devices don't include Google services. Fleet enrolls Android hosts through Google's Android Management API. AOSP support is requested in [#32525](https://github.com/fleetdm/fleet/issues/32525). |
| Samsung Tizen and LG webOS | ❌ | ❌ | ❌ | ❌ | ❌ | Smart TV and signage OSs that don't let you install Fleet's agent. |
| Windows Embedded and Windows CE | ❌ | ❌ | ❌ | ❌ | ❌ | Windows Embedded 7 and 8.1 are older than Fleet's agent supports. Windows CE isn't based on desktop Windows. |
| Windows IoT Core | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet's agent doesn't support Windows IoT Core. |
| AIX, Solaris, and HP-UX | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet's agent has no build for these. |
| VMware ESXi | ❌ | ❌ | ❌ | ❌ | ❌ | ESXi isn't Linux, and it doesn't let you install Fleet's agent. |
| Network and security appliances | ❌ | ❌ | ❌ | ❌ | ❌ | Includes Cisco IOS XE, Cisco Meraki (MR, MS, and MX), Junos, ArubaOS, Huawei VRP, PAN-OS, FortiOS, and SonicOS. These OSs are locked by the vendor. Use the vendor's API or SNMP. |
| Conferencing devices, printers, and real-time OSs | ❌ | ❌ | ❌ | ❌ | ❌ | Includes Cisco RoomOS, HP FutureSmart, Xerox ConnectKey, and real-time OSs. Their firmware is locked by the vendor. |

### Linux distributions

Fleet treats Linux as a first-class platform, not an afterthought. It's open source, API-first, and built so every Linux device gets the same management as your Macs and PCs. Fleet supports these distributions, starting at these minimum versions:

| <img src="../website/assets/images/os-icon-ubuntu-40x40@2x.png" alt="Ubuntu logo" width="40" /> | <img src="../website/assets/images/os-icon-debian-40x40@2x.png" alt="Debian logo" width="40" /> | <img src="../website/assets/images/os-icon-rhel-40x40@2x.png" alt="RHEL logo" width="40" /> | <img src="../website/assets/images/os-icon-fedora-40x40@2x.png" alt="Fedora logo" width="40" /> | <img src="../website/assets/images/os-icon-centos-40x40@2x.png" alt="CentOS logo" width="40" /> |
| :---: | :---: | :---: | :---: | :---: |
| Ubuntu<br>20.04+ | Debian<br>11+ | RHEL<br>7+ | Fedora<br>38+ | CentOS<br>7.1+ |
| <img src="../website/assets/images/os-icon-amazon-linux-40x40@2x.png" alt="Amazon Linux logo" width="40" /> | <img src="../website/assets/images/os-icon-opensuse-40x40@2x.png" alt="openSUSE logo" width="40" /> | <img src="../website/assets/images/os-icon-arch-linux-40x40@2x.png" alt="Arch Linux logo" width="40" /> | <img src="../website/assets/images/os-icon-omarchy-40x40@2x.png" alt="Omarchy logo" width="40" /> | <img src="../website/assets/images/os-icon-nvidia-dgx-40x40@2x.png" alt="NVIDIA DGX OS logo" width="40" /> |
| Amazon Linux<br>2+ | openSUSE<br>15.6+ | Arch<br>Rolling | Omarchy<br>Rolling | NVIDIA DGX OS<br>6+ |
| <img src="../website/assets/images/os-icon-nixos-40x40@2x.png" alt="NixOS logo" width="40" /> | | | | |
| NixOS<br>Unstable | | | | |

✅\* means Fleet collects software on the distribution, with a caveat in the notes below the table. Kernel vulnerability coverage varies by distribution. See [Linux coverage](https://fleetdm.com/guides/vulnerability-processing#linux-coverage). Untested means Fleet hasn't installed its agent on the distribution or confirmed that it collects software.

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
| Parrot OS | Untested |
| UniFi OS | Untested |
| Arista EOS | Untested |
| Check Point Gaia | Untested |
| Raspberry Pi OS 32-bit | ❌ |
| Alpine | ❌ |
| elementary OS | ❌ |
| Deepin | ❌ |
| Garuda | ❌ |
| Clear Linux | ❌ |
| Photon OS | ❌ |
| Solus | ❌ |
| Vanilla OS | ❌ |
| openSUSE MicroOS and Aeon | ❌ |
| SteamOS | ❌ |
| Bazzite, Bluefin, and Aurora | ❌ |
| Fedora Atomic desktops (Silverblue, Kinoite, and others) | ❌ |
| Ubuntu Core | ❌ |
| AXIS OS | ❌ |
| BrightSign OS | ❌ |
| IGEL OS | ❌ |
| Sailfish OS (Jolla Phone) | ❌ |
| Any other distribution | ❌ |

- Fedora, Rocky Linux, AlmaLinux, and Oracle Linux report as RHEL because they ship `/etc/redhat-release`. Fleet shows them under RHEL on the **Software** > **OS** page.
- NVIDIA DGX OS reports as Ubuntu (DGX OS 7 as Ubuntu 24.04, DGX OS 6 as Ubuntu 22.04) because its `/etc/os-release` is Ubuntu's. Fleet doesn't read `/etc/dgx-release`, so the host appears as Ubuntu on the **Software** > **OS** page and the DGX OS version isn't reported. DGX Spark is arm64; use the `arm64` fleetd package.
- NixOS hosts enroll with the NixOS module instead of a package from `fleetctl package`, and Fleet collects their software as [Nix packages](#packages). Learn how in [Enroll NixOS hosts](https://fleetdm.com/guides/enroll-hosts#enroll-nixos-hosts).
- Raspberry Pi OS 64-bit reports as Debian and is covered. Raspberry Pi OS 32-bit reports as `raspbian`, which Fleet doesn't recognize.
- Alpine, elementary OS, Deepin, Garuda, Clear Linux, Photon OS, Solus, Vanilla OS, openSUSE MicroOS, Aeon, and any distribution not listed above: Fleet recognizes Linux by matching the platform value Fleet reads from `/etc/os-release` against a [fixed list](https://github.com/fleetdm/fleet/blob/main/server/fleet/hosts.go). Hosts with any other value enroll, but Fleet doesn't collect an OS entry or software inventory for them.
- Fedora Atomic desktops, Bazzite, Bluefin, and Aurora use rpm-ostree, which keeps the OS read-only. Fleet's standard agent package can't be installed on them, but you can repackage it. Learn how in [Enroll rpm-ostree hosts](https://fleetdm.com/guides/enroll-hosts#enroll-rpm-ostree-hosts). On these hosts, Fleet collects RPM packages only after you add the `nobody` user to `/etc/passwd`, which is a step in that guide. Learn more in [#58969](https://github.com/fleetdm/fleet/issues/58969). Native rpm-ostree support is tracked in [#48694](https://github.com/fleetdm/fleet/issues/48694).
- SteamOS also has a read-only OS, and Fleet's agent package can't be installed on it.
- Fleet hasn't tested UniFi OS, Arista EOS, Check Point Gaia, or Parrot OS. Fleet collects software only if the platform value in `/etc/os-release` is on Fleet's [fixed list](https://github.com/fleetdm/fleet/blob/main/server/fleet/hosts.go). `parrot` is on the list. Arista EOS installs third-party software as EOS extensions.
- Check Point Gaia is based on Red Hat Enterprise Linux. Check with Check Point before you install third-party software on a Gaia appliance.
- AXIS OS 12 and later doesn't give root access, and BrightSign OS runs only BrightSign-approved code, so Fleet's agent can't be installed on them.
- Ubuntu Core runs snaps only. Fleet's agent isn't packaged as a snap, so Ubuntu Core hosts can't enroll.
- [IGEL OS](https://www.igel.com/secure-endpoint-os/) uses an immutable root filesystem, and software is delivered as IGEL apps through IGEL's App Portal. Fleet's agent isn't available as an IGEL app, so IGEL OS isn't supported.
- [Sailfish OS](https://sailfishos.org/), used on the [Jolla Phone](https://commerce.jolla.com/products/jolla-phone-october-2026), reports its platform as `sailfishos`, which Fleet doesn't recognize. Fleet has no MDM support for Sailfish OS, and Android apps that run through Jolla AppSupport aren't collected.

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
| Nix package (`source`: `nix_packages`) | Linux (NixOS only) | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ When the NVD has a matching entry. A CVE that nixpkgs patched without changing the package version may still be reported. | Includes packages in the NixOS system profile and users' Nix and Home Manager profiles, plus their runtime dependencies. The install path is the Nix store path. A package with several outputs (for example, `openssl-3.0.14` and `openssl-3.0.14-bin`) is one software item with several install paths. Nix installed on another distribution isn't collected, because that distribution's packages are already reported. |
| Python package (`source`: `python_packages`) | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | System and per-user site-packages, plus pipx, uv tools, conda, pipenv, pyenv, and mise. Project-level `.venv` directories aren't collected. |
| npm package (`source`: `npm_packages`) | macOS, Linux | ✅ | ✅ | ❌ | ✅ | ❌ | ✅ | Global packages only, including nvm, fnm, asdf, mise, and Volta locations. Windows isn't queried yet. Project-level `node_modules` directories aren't collected. |
| macOS package (.pkg) (`source`: `pkg_packages`), Tarball (.tar.gz) (`source`: `tgz_packages`), Script-only package (.sh) (`source`: `sh_packages`), Script-only package (.py) (`source`: `py_packages`), and Script-only package (.ps1) (`source`: `ps1_packages`) | ❌ Not collected from hosts | N/A | N/A | N/A | N/A | N/A | N/A | Fleet sets these types for custom packages added to the software library, not for software it finds on hosts. A `.pkg` that installs an app appears as a macOS app instead. |
| Scoop, winget portable packages, snap, Flatpak, AppImage, Nix on distributions other than NixOS, pnpm, Yarn, Bun, RubyGems, cargo, and Nim | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet doesn't read them. |
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

Fleet Premium marks AI tools in software inventory. To see only AI tools, go to **Software**, select **Filters**, and turn on **AI tools**. In the API, use the `ai_tool` filter on [List software](https://fleetdm.com/docs/rest-api/rest-api#list-software). Fleet finds AI tools with the [ai_tools](https://github.com/fleetdm/fleet/blob/main/orbit/pkg/table/ai_tools/README.md) table in Fleet's agent.

If Fleet already collects an AI tool as another type, such as a macOS app or an npm package, it keeps that type. Fleet adds three types for AI tools it doesn't collect any other way: **MCP server**, **AI skill**, and **AI CLI tool**.

| Type | Collected on | Name | Version | Vendor | Install path | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- |
| AI desktop apps | macOS, Windows | ✅ | ✅ | ✅ | ✅ | ✅ | Keep their existing type, such as macOS app or Windows app. Covers apps such as ChatGPT, Claude, Cursor, LM Studio, Ollama, and Jan. |
| AI mobile apps | iOS/iPadOS, Android | ✅ | ✅ | ❌ | ❌ | ❌ | Collected as iOS, iPadOS, or Android apps, but not marked as AI tools. The `ai_tools` table runs in Fleet's agent, which doesn't run on mobile devices. |
| AI CLI tools | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ Only when installed with a package manager. | CLIs installed with a package manager keep that type, such as npm package, Homebrew package, or Go binary. Learn more in [Packages](#packages) and [Binaries and other](#binaries-and-other). CLIs installed any other way, such as Claude Code's native installer, appear as **AI CLI tool** (`source`: `ai_clis`). |
| MCP servers | macOS, Windows, Linux | ✅ | N/A | ❌ | ✅ | ❌ | Appear as **MCP server** (`source`: `mcp_servers`). Most MCP servers don't have a version, because they update on the fly, locally or remotely. The install path is the config file that declares the server. |
| AI skills | macOS, Windows, Linux | ✅ | N/A | N/A | ✅ | N/A | Appear as **AI skill** (`source`: `ai_skills`). Covers two kinds of markdown files that steer AI agents. Agent instructions, such as `AGENTS.md`, `CLAUDE.md`, `.cursorrules`, and `.windsurfrules`, load in every session. Skills, such as Claude Code's `SKILL.md` or a project's `.agents/skills` directory, load only when a task needs them. Skills are [coming soon](https://github.com/fleetdm/fleet/issues/51288). Neither has a version or vulnerabilities. |
| AI IDE extensions | macOS, Windows, Linux | ✅ | ✅ | ✅ | ✅ | ✅ | Keep their [IDE extensions](#ide-extensions) type, such as VS Code extension. |
| AI browser extensions | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ | Keep their [browser extensions](#browser-extensions) type, such as Chrome extension. |
| Live AI and MCP sockets | ❌ Not in inventory | ❌ | ❌ | ❌ | ❌ | ❌ | Open network ports for running AI tools, such as a local model server's API port. They aren't software, so they don't appear in software inventory. You can report on them with the `ai_tools` table's `sockets` type. |
| AI models | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet doesn't collect installed model weights or which model a local runner has loaded. Only the runner app is visible, under AI desktop apps. |
| AI sessions | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | Fleet doesn't track AI agent sessions, such as which user ran Claude Code, Codex, or Cursor, when, or in which project. A `sessions` or `memory` directory only counts as evidence of an agent workspace; its contents aren't read. Seeing AI sessions on the **Host details** page is [coming soon](https://github.com/fleetdm/fleet/issues/51825). |

- Fleet marks an AI tool collected as another type by matching install paths. Windows apps whose registry entry has no install location, such as some per-user installs, aren't marked. Linux desktop apps aren't marked, because Linux packages don't report an install path.
- On hosts running an older version of Fleet's agent, AI CLI tools from a native installer, such as Claude Code's, show no version. AI CLI tools installed with Homebrew also appear as a separate **AI CLI tool** with no version, and the Homebrew package isn't marked as an AI tool. Update Fleet's agent to fix both.


## Binaries and other

| Type | Collected on | Name | Version | Vendor | Install path | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Go binary (`source`: `go_binaries`) | macOS, Windows, Linux | ✅ | ✅ | ❌ | ✅ | ✅ When the NVD has a matching entry. | Collected from each user's `~/go/bin` directory. Needs Fleet's agent. Learn more in the [go_binaries](https://github.com/fleetdm/fleet/tree/main/orbit/pkg/table/go_binaries) reference. |
| macOS widgets (`source`: `apps`) | macOS, as part of their app | ✅ | ✅ | ✅ | ✅ | ✅ | Collected under Apps. Widgets are WidgetKit extensions that ship inside apps that already appear there. |
| Shortcuts and Android ringtones | ❌ Not collected | ❌ | ❌ | ❌ | ❌ | ❌ | No MDM API exposes them. |

## Drivers

Fleet doesn't list drivers in software inventory, but you can report on them. ✅ in **Shown in Fleet** means Fleet stores the value and shows it in the UI or API. **Available in reports** lists the tables to use in a report.

| Type | Shown in Fleet | Available in reports | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- |
| Windows drivers | ❌ | ✅ Windows: [drivers](https://fleetdm.com/tables/drivers), [amcache_driver_packages](https://fleetdm.com/tables/amcache_driver_packages), and [amcache_driver_binaries](https://fleetdm.com/tables/amcache_driver_binaries) | ❌ | `drivers` lists drivers in use, with name, version, provider, manufacturer, date, image path, and signing status. `amcache_driver_binaries` adds SHA-1 hashes. The Amcache tables need Fleet's agent 1.64.0 or later. |
| macOS kernel and system extensions | ❌ | ✅ macOS: [kernel_extensions](https://fleetdm.com/tables/kernel_extensions) and [system_extensions](https://fleetdm.com/tables/system_extensions) | ❌ | System extensions report the Apple Developer Team ID. |
| Linux kernel modules | ❌ | ✅ Linux: [kernel_modules](https://fleetdm.com/tables/kernel_modules) | ❌ | No version column. Kernel vulnerabilities are matched under [Operating systems](#operating-systems), on supported distributions. |

- Fleet doesn't match drivers against vulnerabilities. Attackers can load a legitimate, signed driver that has a known vulnerability to run code in the Windows kernel. To find these drivers, write a policy that compares `amcache_driver_binaries` hashes against a blocklist, such as Microsoft's [recommended driver block rules](https://learn.microsoft.com/en-us/windows/security/application-security/application-control/app-control-for-business/design/microsoft-recommended-driver-block-rules).
- Fleet doesn't show which driver updates are available or pending. Windows driver updates are requested in [#29903](https://github.com/fleetdm/fleet/issues/29903) and [#54997](https://github.com/fleetdm/fleet/issues/54997).

## BIOS and firmware

Fleet doesn't list firmware in software inventory, but you can report on most of it. ✅ in **Shown in Fleet** means Fleet stores the value and shows it in the UI or API. **Available in reports** lists the tables to use in a report.

| Type | Shown in Fleet | Available in reports | Vulnerabilities | Caveats |
| --- | --- | --- | --- | --- |
| System firmware (BIOS or UEFI) | ❌ | ✅ macOS, Windows, Linux: [platform_info](https://fleetdm.com/tables/platform_info) | ❌ | Reports vendor, version, release date, and firmware type (UEFI, BIOS, or iBoot). Fleet doesn't check firmware for known vulnerabilities, such as LogoFAIL and PKfail ([CVE-2024-8105](https://nvd.nist.gov/vuln/detail/CVE-2024-8105)). |
| SMBIOS structures | ❌ | ✅ macOS, Linux: [smbios_tables](https://fleetdm.com/tables/smbios_tables) | N/A | Needs an Intel-compatible host. Not available on Windows. |
| CPU microcode | ❌ | ✅ macOS, Windows, Linux: `cpu_microcode` in [system_info](https://fleetdm.com/tables/system_info) | ❌ | On Linux, microcode packages appear under [Packages](#packages) and are matched like other packages on supported distributions. Fleet doesn't match the running microcode version against CPU vulnerabilities, such as Downfall ([CVE-2022-40982](https://nvd.nist.gov/vuln/detail/CVE-2022-40982)). |
| BIOS settings | ❌ | ✅ Windows: [wmi_bios_info](https://fleetdm.com/tables/wmi_bios_info) | N/A | Reads setting names and values on Dell, HP, and Lenovo PCs. On some Dell PCs, it needs Dell Command \| Monitor. Config-defined WMI tables, which could read PC makers' own BIOS data such as Dell Trusted Device results, are requested in [#59971](https://github.com/fleetdm/fleet/issues/59971). |
| Firmware and BIOS passwords | ✅ macOS (Apple silicon): Recovery Lock password. Requires Fleet Premium and MDM. | ✅ macOS (Intel): [firmwarepasswd](https://fleetdm.com/tables/firmwarepasswd) | N/A | Fleet sets, escrows, and rotates Recovery Lock passwords. Learn more in [Recovery Lock passwords](https://fleetdm.com/guides/recovery-lock-password). Setting a BIOS password on Windows is [coming soon](https://github.com/fleetdm/fleet/issues/57800). Escrowing a unique BIOS password per Windows host is requested in [#59972](https://github.com/fleetdm/fleet/issues/59972). Fleet doesn't report whether a BIOS password is set on Windows or Linux hosts. |
| Secure Boot state | ❌ | ✅ macOS, Windows, Linux: [secureboot](https://fleetdm.com/tables/secureboot) | N/A | Columns vary by platform. |
| Secure Boot certificates (db and dbx) | ❌ | ✅ Linux: [secureboot_certificates](https://fleetdm.com/tables/secureboot_certificates) | ❌ | Shows each certificate's issuer, expiry date, and whether it's revoked. Doesn't show hash entries in dbx, and most revocations are hashes. Not collected on Windows. See the notes below the table. |
| TPM | ❌ | ✅ Windows: [tpm_info](https://fleetdm.com/tables/tpm_info) | ❌ | Reports manufacturer, spec and firmware version, and whether the TPM is activated, enabled, and owned. Not collected on Linux. Macs don't have a TPM. |
| Intel Management Engine (ME) | ❌ | ✅ Windows, Linux: [intel_me_info](https://fleetdm.com/tables/intel_me_info) | ❌ | Version only. |
| AMD Secure Processor (PSP) | ❌ | ❌ | ❌ | — |
| Mac security chip and component firmware | ❌ | ✅ macOS: [ibridge_info](https://fleetdm.com/tables/ibridge_info) and [device_firmware](https://fleetdm.com/tables/device_firmware) | ❌ | `ibridge_info` reports T1 and T2 chip firmware. `device_firmware` is best effort, and can include SSD controller firmware. |
| Mac firmware integrity | ❌ | ✅ Intel Macs without a T2 chip: [firmware_eficheck_integrity_check](https://fleetdm.com/tables/firmware_eficheck_integrity_check) | N/A | Runs Apple's `eficheck`. Needs Fleet's agent. |
| iPhone and iPad modem firmware | ✅ iOS, iPadOS | N/A | ❌ | On the **Host details** page, under **Vitals** > **View all**. Not collected on personally-owned (BYOD) hosts. |
| Android bootloader | ✅ Android | N/A | ❌ | Fleet also shows the security update version and Google's security posture. The security update version is matched under [Operating systems](#operating-systems). Bootloader lock state isn't collected. |
| USB and peripheral devices | ❌ | ✅ macOS, Linux: [usb_devices](https://fleetdm.com/tables/usb_devices). Windows: [amcache_device_pnp](https://fleetdm.com/tables/amcache_device_pnp) | N/A | Reports device vendor, model, and version, not firmware. `amcache_device_pnp` also lists devices that were connected before, and needs Fleet's agent 1.64.0 or later. USB inventory on Windows is requested in [#26953](https://github.com/fleetdm/fleet/issues/26953). |
| PCI devices | ❌ | ✅ macOS, Linux: [pci_devices](https://fleetdm.com/tables/pci_devices) | N/A | Reports each device's vendor, model, and driver. Doesn't report firmware versions. Not available on Windows. |
| Network adapter, SSD, Thunderbolt, and embedded controller firmware on PCs | ❌ | ❌ | ❌ | Fleet doesn't read firmware versions for these components. |
| Baseboard management controllers (BMCs), such as iDRAC and iLO | ❌ | ❌ | ❌ | BMCs run separately from the host OS. An AMI MegaRAC BMC vulnerability, [CVE-2024-54085](https://nvd.nist.gov/vuln/detail/CVE-2024-54085), is on CISA's Known Exploited Vulnerabilities list. |

- Fleet doesn't show which BIOS or firmware updates are available or pending on a host.
  - Windows: Fleet's agent lists pending Windows updates, but not driver or firmware offers. Fleet can install Dell Command Update, Lenovo System Update, and Lenovo Dock Manager as Fleet-maintained apps, but doesn't run their scans or read the results. HP Image Assistant isn't a Fleet-maintained app. Running Dell Command Update and HP Image Assistant is requested in [#51145](https://github.com/fleetdm/fleet/issues/51145). Managing Windows driver and firmware updates and BIOS settings is requested in [#60037](https://github.com/fleetdm/fleet/issues/60037).
  - macOS: [software_update](https://fleetdm.com/tables/software_update) lists pending macOS updates and needs Fleet's agent 1.63.0 or later. macOS updates include Mac firmware.
  - Linux: Fleet doesn't read pending firmware updates from fwupd.
- Update history: [windows_update_history](https://fleetdm.com/tables/windows_update_history) shows whether Windows Update installs succeeded or failed. Updates installed by PC makers' tools or fwupd aren't recorded.
- Fleet doesn't report whether a Windows host supports DFCI, the firmware interface Intune uses to manage UEFI settings.
- Secure Boot certificates: Microsoft UEFI CA 2011, which signs Linux boot loaders, expired on June 27, 2026. Hosts with expired certificates still boot, but can't boot future boot loaders signed only with Microsoft UEFI CA 2023. Use `secureboot_certificates` to find Linux hosts that don't have the 2023 certificate yet.
- On Windows, a table that reports each host's progress through Microsoft's 2023 Secure Boot certificate update is [coming soon](https://github.com/fleetdm/fleet/issues/51382). It doesn't list db or dbx entries. Until then, follow [Microsoft is rotating every Windows PC's Secure Boot keys](https://fleetdm.com/articles/microsoft-is-rotating-every-windows-pcs-secure-boot-keys).
- Fleet doesn't compare firmware against known-good versions or detect firmware implants. The only exception is `eficheck` on Intel Macs without a T2 chip.
- Fleet doesn't build a firmware software bill of materials (SBOM).

### Processor and hardware details

Fleet stores more hardware details than the **Host details** page shows. The rest are in the API.

| Detail | Host details page | API | Available in reports | Caveats |
| --- | --- | --- | --- | --- |
| Processor type (architecture), such as `x86_64` or `arm64e` | ✅ | ✅ `cpu_type` | ✅ [system_info](https://fleetdm.com/tables/system_info) | Not shown for iOS and iPadOS hosts. |
| CPU model, such as Apple M3 Pro or Intel Core i7 | ❌ | ✅ `cpu_brand` | ✅ [system_info](https://fleetdm.com/tables/system_info) | — |
| CPU physical and logical cores | ❌ | ✅ `cpu_physical_cores` and `cpu_logical_cores` | ✅ [system_info](https://fleetdm.com/tables/system_info) | — |
| Hardware model and serial number | ✅ | ✅ `hardware_model` and `hardware_serial` | ✅ [system_info](https://fleetdm.com/tables/system_info) | — |
| Hardware vendor and hardware version | ❌ | ✅ `hardware_vendor` and `hardware_version` | ✅ [system_info](https://fleetdm.com/tables/system_info) | — |

- A CSV exported from the **Hosts** page includes only the columns shown in the table. To export every field, call the hosts report endpoint without the `columns` parameter. Learn more in the [REST API](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-report-in-csv) reference.

## Data collected

The tables above list what Fleet stores for each software type. Drivers, BIOS, firmware, and hardware details aren't software types, so the rest of this section doesn't apply to them. Two things apply to every software type:

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
<meta name="articleTitle" value="Software inventory">
<meta name="description" value="Fleet collects the apps, operating systems, browser extensions, packages, IDE extensions, plugins, and binaries installed on your hosts.">