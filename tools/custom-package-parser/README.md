# Custom packages tests

This document aims to provide us with some stats for the most used apps with respect to what Fleet extracts from the installers and what osquery reports for the installed applications.
The goal is to improve the accuracy of automatically generated policy queries for installers.

## Results

The results have been calculated using many of the apps in the current list of FMA apps for macOS (as of December 2024).

### pkg

11 `pkg`s were tested:
- Matching of extracted bundle identifier and osquery's reported `bundle_identifier`: 100% (11/11)
- Matching of extracted package title/name and osquery's reported `apps.name`: 100% (11/11)

### msi

10 `msi`s were tested:
- Matching of extracted GUID and osquery's reported `programs.identifying_number`: 90% (9/10)
- Matching of extracted package title/name and osquery's reported `programs.name`: 90% (9/10)

### exe

13 `exe`s were tested:
- Matching of extracted package title/name and osquery's reported `programs.name`: ~30% (4/13)

### deb

6 `deb`s were tested:
- Matching of extracted package title/name and osquery's reported `deb_packages.name`: ~100% (6/6)

### rpm

6 `rpm`s were tested:
- Matching of extracted package title/name and osquery's reported `deb_packages.name`: ~100% (6/6)

## Tests

### 1Password

#### pkg

✅ https://downloads.1password.com/mac/1Password.pkg
- Bundle Identifier: 'com.1password.1password'
- Name: '1Password.app' (matches osquery's apps.name)
- Package IDs: 'com.1password.1password'

#### exe

✅ https://downloads.1password.com/win/1PasswordSetup-latest.exe
- Default installer script didn't work.
- Running `1PasswordSetup-latest.exe --silent` on the `cmd` works, but not via Fleet because the installer is per-user, whereas the MSI is system-wide, see https://support.1password.com/deploy-1password/.
Extracted metadata:
- Name: '1Password' (matches osquery's `programs.name`)
- Package IDs: '1Password'

#### msi

✅ https://downloads.1password.com/win/1PasswordSetup-latest.msi
- Name: '1Password' (matches osquery's `programs.name`)
- Package IDs: '{321BD799-2490-40D7-8A88-6888809FA681}' (matches osquery's `programs.identifying_number`)

#### deb

✅ https://downloads.1password.com/linux/debian/amd64/stable/1password-latest.deb
- Name: '1password' (matches osquery's `deb_packages.name`)
- Package IDs: '1password'

#### rpm

✅ https://downloads.1password.com/linux/rpm/stable/x86_64/1password-latest.rpm
- Name: '1password' (matches osquery's `rpm_packages.name`)
- Package IDs: '1password'

### Adobe Acrobat Reader

#### pkg

N/A (they have .dmg)

#### exe

❌ https://ardownload2.adobe.com/pub/adobe/acrobat/win/AcrobatDC/2400520320/AcroRdrDCx642400520320_en_US.exe
- Name: 'Adobe Self Extractor' (osquery reports `Adobe Acrobat (64-bit)` in `programs.name`)
- Package IDs: 'Adobe Self Extractor'

#### msi

N/A

#### deb

N/A

#### rpm

N/A

### Box Drive

#### pkg

✅ https://e3.boxcdn.net/desktop/releases/mac/BoxDrive.pkg
- Name: 'Box.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'com.box.desktop'
- Package IDs: 'com.box.desktop.installer.autoupdater,com.box.desktop.installer.desktop,com.box.desktop.installer.local.appsupport'

#### msi

✅ https://e3.boxcdn.net/desktop/releases/win/BoxDrive.msi
- Name: 'Box' (matches osquery's `programs.name`)
- Package IDs: '{9ACD1AAB-DCE9-480D-A7A4-5470D5E4E10F}' (matches osquery's `programs.identifying_number`)

#### exe

N/A

#### deb

N/A

#### rpm

N/A

### Brave Browser

#### pkg

✅ https://github.com/brave/brave-browser/releases/download/v1.73.101/Brave-Browser-universal.pkg
- Name: 'Brave Browser.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'com.brave.Browser'
- Package IDs: 'com.brave.Browser,com.brave.Browser.helper.renderer,com.brave.Updater,com.brave.Keystone,com.brave.Browser.framework,com.brave.Browser.helper,com.brave.Browser.helper.plugin,org.sparkle-project.Sparkle,org.sparkle-project.Sparkle.Autoupdate,com.brave.Keystone.Agent,com.brave.Browser.framework.AlertNotificationService'

#### exe

❌ https://referrals.brave.com/latest/BraveBrowserSetup.exe
- Default installer script doesn't work.
- Name: 'BraveSoftware Update' (does not match osquery's `programs.name`, which is 'Brave')
- Package IDs: 'BraveSoftware Update'

#### msi

N/A

#### deb

✅ https://github.com/brave/brave-browser/releases/download/v1.73.101/brave-browser_1.73.101_amd64.deb
- Default installer script doesn't work.
- Name: 'brave-browser' (matches osquery's `deb_packages.name`)
- Package IDs: 'brave-browser'

#### rpm

✅ https://github.com/brave/brave-browser/releases/download/v1.73.101/brave-browser-1.73.101-1.x86_64.rpm
- Default installer script doesn't work.
- Name: 'brave-browser' (matches osquery's `rpm_packages.name`)
- Package IDs: 'brave-browser'

### Cloudflare WARP

#### pkg

✅ https://appcenter-filemanagement-distrib5ede6f06e.azureedge.net/e638644a-02a2-4a21-aa30-8a9a1bf774ce/Cloudflare_WARP_2024.11.309.0.pkg
- Name: 'Cloudflare WARP.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'com.cloudflare.1dot1dot1dot1.macos' 
- Package IDs: 'com.cloudflare.1dot1dot1dot1.macos'

#### msi

✅ https://appcenter-filemanagement-distrib3ede6f06e.azureedge.net/679d20da-1684-49df-89e5-e976ec1c010c/Cloudflare_WARP_2024.11.309.0.msi
- Name: 'Cloudflare WARP' (matches osquery's `programs.name`)
- Package IDs: '{2BC6DCCB-7E9D-44D7-A525-6F6C6E83C419}' (matches osquery's `programs.identifying_number`)

#### exe

N/A

#### deb

✅ https://pkg.cloudflareclient.com/pool/focal/main/c/cloudflare-warp/cloudflare-warp_2024.11.309.0_amd64.deb
- Name: 'cloudflare-warp' (matches osquery's `deb_packages.name`)
- Package IDs: 'cloudflare-warp'

#### rpm

N/A

### Docker

#### pkg

N/A (has dmg, pkg requires admin account in app.docker.com)

#### msi

N/A (msi requires admin account in app.docker.com)

#### exe

❌ https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe
- Default installer script doesn't work.
- Name: 'Docker Desktop Installer' (doesn't match osquery's `programs.name`)
- Package IDs: 'Docker Desktop Installer'

#### deb

✅ https://desktop.docker.com/linux/main/amd64/docker-desktop-amd64.deb
- Name: 'docker-desktop' (matches osquery's `deb_packages.name`)
- Package IDs: 'docker-desktop'

#### rpm

❌ https://desktop.docker.com/linux/main/amd64/docker-desktop-x86_64.rpm
- Default installer script doesn't work on my Fedora 38 VM.

### Figma

#### pkg

✅ https://desktop.figma.com/mac-universal/Figma-124.6.5.pkg
- Name: 'Figma.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'com.figma.Desktop'
- Package IDs: 'com.figma.Desktop'

#### msi

✅ https://desktop.figma.com/win/Figma-124.6.5.msi
- Name: 'Figma (Machine - MSI)' (matches osquery's `programs.name`)
- Package IDs: '{6332AF99-9139-41D1-98FC-BA21B9D6DE2E}' (matches osquery's `programs.identifying_number`)

#### exe

❌ https://desktop.figma.com/win/FigmaSetup.exe
- Default installer script doesn't work.
- Name: 'Figma Desktop' (doesnt match osquery's `programs.name`)
- Package IDs: 'Figma Desktop'

#### deb

✅ https://github.com/Figma-Linux/figma-linux/releases/download/v0.11.5/figma-linux_0.11.5_linux_amd64.deb
- Name: 'figma-linux'
- Package IDs: 'figma-linux'

#### rpm

✅ https://github.com/Figma-Linux/figma-linux/releases/download/v0.11.5/figma-linux_0.11.5_linux_x86_64.rpm
- Name: 'figma-linux'
- Package IDs: 'figma-linux'

### Firefox

#### pkg

✅ https://ftp.mozilla.org/pub/firefox/releases/129.0.2/mac/en-US/Firefox%20129.0.2.pkg
- Name: 'Firefox.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'org.mozilla.firefox'
- Package IDs: 'org.mozilla.firefox'

#### msi

❌ https://ftp.mozilla.org/pub/firefox/releases/129.0.2/win64/en-US/Firefox%20Setup%20129.0.2.msi
- Name: 'Mozilla Firefox 129.0.2 x64 en-US' (doesn't match osquery's `programs.name`, `Mozilla Firefox (x64 en-US)`)
- Package IDs: '{1294A4C5-9977-480F-9497-C0EA1E630130}' (osquery returns empty `programs.identifying_number`)
- Default uninstall script doesn't work because it seems the installer doesn't set the GUID on the system registry.

#### exe

❌ https://download-installer.cdn.mozilla.net/pub/firefox/releases/133.0.3/win32/en-US/Firefox%20Installer.exe
- Default installer script succeeds but doesn't install Firefox
- Name: 'Firefox' (doesn't match osquery's `programs.name`, `Mozilla Firefox (x64 en-US`)
- Package IDs: 'Firefox'

#### deb

✅ https://ftp.mozilla.org/pub/firefox/releases/129.0.2/linux-x86_64/en-US/firefox-129.0.2.deb
- Name: 'firefox' (matches osquery's `deb_packages.name`)
- Package IDs: 'firefox'

#### rpm

Skipped.

### Chrome

#### pkg

✅ https://dl.google.com/dl/chrome/mac/universal/stable/gcem/GoogleChrome.pkg
- Name: 'Google Chrome.app' (matches osquery's apps.name)
- Bundle Identifier: 'com.google.Chrome'
- Package IDs: 'com.google.Chrome'

#### msi

✅ https://dl.google.com/tag/s/appguid%3D%7B8A69D345-D564-463C-AFF1-A69D9E530F96%7D%26iid%3D%7BDAD35779-DEEF-9D60-7F91-7A3EEC3B65A9%7D%26lang%3Den%26browser%3D4%26usagestats%3D0%26appname%3DGoogle%2520Chrome%26needsadmin%3Dtrue%26ap%3Dx64-stable-statsdef_0%26brand%3DGCEA/dl/chrome/install/googlechromestandaloneenterprise64.msi
- Name: 'Google Chrome' (matches osquery's `programs.name`)
- Package IDs: '{D9596C6B-431E-3638-ACB7-B4B0D24D2D1B}' (matches osquery's `programs.identifying_number`)

#### exe

❌ https://dl.google.com/tag/s/appguid%3D%7B8A69D345-D564-463C-AFF1-A69D9E530F96%7D%26iid%3D%7B8CCBCFA1-CE41-77DB-B8C4-98742A89BC8D%7D%26lang%3Des-419%26browser%3D5%26usagestats%3D1%26appname%3DGoogle%2520Chrome%26needsadmin%3Dprefers%26ap%3Dx64-statsdef_1%26brand%3DUEAD%26installdataindex%3Dempty/update2/installers/ChromeSetup.exe
- Name: 'Google Installer' (doesn't match osquery's `programs.name`)
- Package IDs: 'Google Installer'

#### deb

Skipped.

#### rpm

✅ https://dl.google.com/linux/chrome/rpm/stable/x86_64/google-chrome-stable-129.0.6668.70-1.x86_64.rpm
- Name: 'google-chrome-stable' (matches osquery's `rpm_packages.name`)
- Package IDs: 'google-chrome-stable'

### Microsoft Edge

#### pkg

✅ https://msedge.sf.dl.delivery.mp.microsoft.com/filestreamingservice/files/8613322a-2386-49ce-a73f-0b718af56cfe/MicrosoftEdge-131.0.2903.99.pkg
- Name: 'Microsoft Edge.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'com.microsoft.edgemac'
- Package IDs: 'com.microsoft.edgemac'

#### msi

✅ https://msedge.sf.dl.delivery.mp.microsoft.com/filestreamingservice/files/249fe233-1b7c-4b8d-93bc-e64ba81a0c02/MicrosoftEdgeEnterpriseX64.msi
- Name: 'Microsoft Edge' (matches osquery's `programs.name`)
- Package IDs: '{5DFDE950-0D8C-30AC-966B-EED2E340F09B}' (matches osquery's `programs.identifying_number`)

#### exe

N/A

#### deb

✅ https://packages.microsoft.com/repos/edge/pool/main/m/microsoft-edge-stable/microsoft-edge-stable_131.0.2903.99-1_amd64.deb
- Name: 'microsoft-edge-stable' (matches osquery's `deb_packages.name`)
- Package IDs: 'microsoft-edge-stable'

#### rpm

✅ https://packages.microsoft.com/yumrepos/edge/microsoft-edge-stable-131.0.2903.99-1.x86_64.rpm
- Name: 'microsoft-edge-stable'
- Package IDs: 'microsoft-edge-stable'

### Microsoft Excel

Skipped (not easy to get ahold of installers)

### Microsoft Teams

#### pkg

✅ https://statics.teams.cdn.office.net/production-osx/enterprise/webview2/lkg/MicrosoftTeams.pkg
- Name: 'Microsoft Teams.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'com.microsoft.teams2'
- Package IDs: 'com.microsoft.MSTeamsAudioDevice,com.microsoft.teams2,com.microsoft.package.Microsoft_AutoUpdate.app'

#### msi

✅ https://statics.teams.cdn.office.net/production-windows-x64/1.7.00.33761/Teams_windows_x64.msi
- Default installer script doesn't work.
- Name: 'Teams Machine-Wide Installer' (matches osquery's `programs.name`)
- Package IDs: '{731F6BAA-A986-45A4-8936-7C3AAAAA760B}' (matches osquery's `programs.identifying_number`)

#### exe

❌ https://statics.teams.cdn.office.net/evergreen-assets/DesktopClient/MSTeamsSetup.exe
- Name: 'Microsoft Teams' (osquery does not return the entry for the installed Microsoft Teams on this setup, maybe a osquery bug?)
- Package IDs: 'Microsoft Teams'

#### deb

Skipped.

#### rpm

Skipped.

### Microsoft Word

Skipped (not easy to get ahold of installers)

### Notion

#### pkg

N/A

#### msi

N/A

#### exe

✅ https://desktop-release.notion-static.com/Notion%20Setup%204.2.0.exe
- Name: 'Notion 4.2.0' (matches osquery's `programs.name`)
- Package IDs: 'Notion'

#### deb

Skipped.

#### rpm

Skipped.

### Postman

#### pkg

N/A (they have a zip:app)

#### msi

N/A

#### exe

✅ https://dl.pstmn.io/download/latest/win64
- Name: 'Postman' (matches osquery's `programs.name`)
- Package IDs: 'Postman'

#### deb

N/A (installer is just a tar.gz)

#### rpm

N/A (installer is just a tar.gz)

### Slack

#### pkg

✅ https://downloads.slack-edge.com/desktop-releases/mac/x64/4.41.105/Slack-4.41.105-macOS.pkg
- Name: 'Slack.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'com.tinyspeck.slackmacgap'
- Package IDs: 'com.tinyspeck.slackmacgap'

#### msi

✅ https://downloads.slack-edge.com/desktop-releases/windows/x64/4.41.105/slack-standalone-4.41.105.0.msi
- Name: 'Slack (Machine - MSI)' (matches osquery's `programs.name`)
- Package IDs: '{D1458C20-B783-4E0C-B9D9-FAC9F56F94DB}' (matches osquery's `programs.identifying_number`)

#### exe

❌ https://downloads.slack-edge.com/desktop-releases/windows/x64/4.41.105/SlackSetup.exe
- Name: 'Slack Desktop' (doesn't match osquery's `programs.name`, `Slack`)
- Package IDs: 'Slack Desktop'

#### deb

Skipped.

#### rpm

✅ https://downloads.slack-edge.com/desktop-releases/linux/x64/4.39.95/slack-4.39.95-0.1.el8.x86_64.rpm
- Name: 'slack' (matches osquery's `rpm_packages.name`)
- Package IDs: 'slack'

### Team Viewer

#### pkg

N/A (needs an admin license)

#### msi

N/A (needs an admin license)

#### exe

N/A (their exes are executables, not installers)

#### deb

Skipped.

#### rpm

Skipped.

### Visual Studio Code

#### pkg

N/A

#### msi

N/A

#### exe

❌ https://vscode.download.prss.microsoft.com/dbazure/download/stable/fabdb6a30b49f79a7aba0f2ad9df9b399473380f/VSCodeSetup-x64-1.96.2.exe
- Name: 'Visual Studio Code' (doesn't match osquery's `programs.name`, `Microsoft Visual Studio Code`)
- Package IDs: 'Visual Studio Code'

#### deb

Skipped.

#### rpm

Skipped.

### WhatsApp

#### pkg

N/A (they have a zip:app)

#### msi

N/A (from app store)

#### exe

N/A (from app store)

#### deb

N/A

#### rpm

N/A

### Zoom for IT admins

#### pkg

✅ https://cdn.zoom.us/prod/6.3.0.44805/ZoomInstallerIT.pkg
- Name: 'zoom.us.app' (matches osquery's `apps.name`)
- Bundle Identifier: 'us.zoom.xos'
- Package IDs: 'us.zoom.pkg.videomeeting'

#### msi

✅ https://cdn.zoom.us/prod/6.3.0.52884/x64/ZoomInstallerFull.msi
- Name: 'Zoom Workplace (64-bit)' (matches osquery's `programs.name`)
- Package IDs: '{9BF959AB-C61A-460F-BA37-7D3DABB1388B}' (matches osquery's `programs.identifying_number`)

#### exe

Skipped.

#### deb

Skipped.

#### rpm

Skipped.

### Tailscale

#### exe

✅ https://dl.tailscale.com/stable/tailscale-setup-1.72.0.exe
- Name: 'Tailscale' (matches osquery's `programs.name`).
- Package IDs: 'Tailscale'
