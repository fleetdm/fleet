# Block Chrome extensions on macOS, Windows, and Linux

Chrome extensions can read page content, cookies, and browsing history. Use Fleet to block extensions you don't trust, allow only the ones you do, and force-install the ones your team needs. This guide covers Google Chrome on macOS, Windows, and Linux.

## Prerequisites

- Hosts enrolled in Fleet.
- The ID of each extension. Find it in the Chrome Web Store URL, or on `chrome://extensions` with **Developer mode** turned on. IDs are 32 lowercase letters, like `abcdefghijklmnopabcdefghijklmnop`.
- Windows only: the Chrome ADMX file deployed to your hosts. See [Managing Google Chrome on Windows with Fleet](https://fleetdm.com/guides/managing-chrome-with-fleet).

Chrome offers three extension policies:

| Policy | What it does |
| ------ | ------------ |
| `ExtensionInstallBlocklist` | Blocks the listed extensions. Use `*` to block everything except the allowlist, including all unpacked extensions. |
| `ExtensionInstallAllowlist` | Exempts the listed extensions from the blocklist. |
| `ExtensionInstallForcelist` | Installs the listed extensions and prevents users from removing them. |

## macOS

1. Save the following as `chrome-extension-blocklist.mobileconfig`. Replace the extension IDs with your own.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.google.Chrome</string>
      <key>PayloadIdentifier</key>
      <string>com.example.chrome.extensions.68027BF3</string>
      <key>PayloadUUID</key>
      <string>68027BF3-B326-4AF9-97F1-2596AD34FA3B</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>ExtensionInstallBlocklist</key>
      <array>
        <string>abcdefghijklmnopabcdefghijklmnop</string>
        <string>ponmlkjihgfedcbaponmlkjihgfedcba</string>
      </array>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Chrome extension blocklist</string>
  <key>PayloadIdentifier</key>
  <string>com.example.chrome.extensions</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>F9F12859-E39A-467C-8A76-2E3A83C3AC73</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

2. In Fleet, go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
3. Choose the `.mobileconfig` file and select **Add profile** to upload it.

To allow only specific extensions, block everything with `*` and add the extensions you approve to `ExtensionInstallAllowlist`:

```xml
<key>ExtensionInstallBlocklist</key>
<array>
  <string>*</string>
</array>
<key>ExtensionInstallAllowlist</key>
<array>
  <string>abcdefghijklmnopabcdefghijklmnop</string>
</array>
```

To force-install an extension, add it to `ExtensionInstallForcelist` as `<extension ID>;<update URL>`. For extensions in the Chrome Web Store, the update URL is `https://clients2.google.com/service/update2/crx`:

```xml
<key>ExtensionInstallForcelist</key>
<array>
  <string>abcdefghijklmnopabcdefghijklmnop;https://clients2.google.com/service/update2/crx</string>
</array>
```

## Windows

1. Save the following as `chrome-extension-blocklist.xml`. Replace the extension IDs with your own.

```xml
<Replace>
  <Item>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/Chrome~Policy~googlechrome~Extensions/ExtensionInstallBlocklist</LocURI>
    </Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>&lt;enabled/&gt;&lt;data id=&quot;ExtensionInstallBlocklistDesc&quot; value=&quot;1&#xF000;abcdefghijklmnopabcdefghijklmnop&#xF000;2&#xF000;ponmlkjihgfedcbaponmlkjihgfedcba&quot;/&gt;</Data>
  </Item>
</Replace>
```

2. In Fleet, go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
3. Choose the `.xml` file and select **Add profile** to upload it.

> **Note:** The `value` is a list of numbered entries separated by the `&#xF000;` character: `1&#xF000;first-id&#xF000;2&#xF000;second-id`.

To allow only specific extensions or force-install extensions, use the same format with a different `LocURI` and data `id`:

| Policy | `LocURI` ending | Data `id` |
| ------ | --------------- | --------- |
| Allowlist | `Chrome~Policy~googlechrome~Extensions/ExtensionInstallAllowlist` | `ExtensionInstallAllowlistDesc` |
| Force-install | `Chrome~Policy~googlechrome~Extensions/ExtensionInstallForcelist` | `ExtensionInstallForcelistDesc` |

For the blocklist wildcard, use `*` as the only entry: `value="1&#xF000;*"`. For force-install entries, use `<extension ID>;<update URL>`.

## Linux

Linux has no MDM configuration profiles. Instead, use a Fleet script to write a managed policy file. Chrome reads every JSON file in `/etc/opt/chrome/policies/managed/`.

1. Save the following as `chrome-extension-blocklist.sh`. Replace the extension IDs with your own. Scripts require `fleetd` with scripts enabled.

```bash
#!/bin/bash

mkdir -p /etc/opt/chrome/policies/managed

cat > /etc/opt/chrome/policies/managed/fleet-extensions.json <<'EOF'
{
  "ExtensionInstallBlocklist": [
    "abcdefghijklmnopabcdefghijklmnop",
    "ponmlkjihgfedcbaponmlkjihgfedcba"
  ]
}
EOF

chmod 644 /etc/opt/chrome/policies/managed/fleet-extensions.json
```

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. Run the script on a host from its **Details** page, or [run it automatically](https://fleetdm.com/guides/policy-automation-run-script) on Linux hosts that fail a policy.

To allow only specific extensions or force-install extensions, add `ExtensionInstallAllowlist` or `ExtensionInstallForcelist` to the same JSON object:

```json
{
  "ExtensionInstallBlocklist": ["*"],
  "ExtensionInstallAllowlist": ["abcdefghijklmnopabcdefghijklmnop"],
  "ExtensionInstallForcelist": [
    "abcdefghijklmnopabcdefghijklmnop;https://clients2.google.com/service/update2/crx"
  ]
}
```

> **Note:** This path applies to Google Chrome. Chromium reads `/etc/chromium/policies/managed/`, and some distributions use a different directory. Ubuntu's Chromium package, for example, reads `/etc/chromium-browser/policies`.

## Manage with GitOps

To manage these settings with [GitOps](https://fleetdm.com/docs/configuration/yaml-files), save the files in your GitOps repository and reference them in your fleet's YAML file:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/profiles/chrome-extension-blocklist.mobileconfig
  windows_settings:
    configuration_profiles:
      - path: ../lib/windows/profiles/chrome-admx.xml
      - path: ../lib/windows/profiles/chrome-extension-blocklist.xml
  scripts:
    - path: ../lib/linux/scripts/chrome-extension-blocklist.sh
```

The `chrome-admx.xml` profile is the Chrome ADMX profile from [Managing Google Chrome on Windows with Fleet](https://fleetdm.com/guides/managing-chrome-with-fleet).

To run the Linux script automatically, add a policy that runs it when a host fails the check. This requires Fleet Premium.

```yaml
policies:
  - name: Linux - Chrome extension blocklist is present
    query: SELECT 1 FROM file WHERE path = '/etc/opt/chrome/policies/managed/fleet-extensions.json';
    platform: linux
    run_script:
      path: ../lib/linux/scripts/chrome-extension-blocklist.sh
```

> **Note:** This policy only checks that the file exists. After you change the extension list, run the script again from each host's **Details** page.

## What users see

- **Blocked extension:** If a user tries to install a blocked extension, Chrome tells them the administrator blocked it: `<name> (extension ID "<id>") is blocked by the administrator.` A blocked extension that's already installed is disabled, and users can't turn it back on.
- **Force-installed extension:** Chrome installs it without a prompt. Users can't remove or modify it. Chrome tells them the administrator of the machine requires the extension.
- **Custom message:** To show your own text in the Chrome Web Store when an install is blocked, such as a link to request an exception, use the `blocked_install_message` field of the [`ExtensionSettings` policy](https://chromeenterprise.google/policies/extension-settings/).

## Verify

1. On a host, open `chrome://policy` in Chrome.
2. Select **Reload policies**.
3. Confirm the extension policies appear with the IDs you set and the status **OK**.

To check across all your hosts, run this report to list installed extensions:

```sql
SELECT users.username, chrome_extensions.name, chrome_extensions.identifier
FROM users
CROSS JOIN chrome_extensions USING (uid);
```

## Troubleshoot

**Policies don't appear on `chrome://policy`**

Restart Chrome and select **Reload policies**. On Windows, confirm the Chrome ADMX file is deployed and the profile shows **Verified** in Fleet.

**The extension is still installed after you block it**

Chrome disables blocked extensions but doesn't always remove them right away. Confirm the ID matches exactly, and check `chrome://policy` for errors on the policy.

**A force-installed extension doesn't install**

Confirm the update URL is reachable from the host and the entry uses the `<extension ID>;<update URL>` format.

## Further reading

- [Chrome Enterprise: Manage Chrome extensions](https://support.google.com/chrome/a/answer/9296680)
- [Chrome Enterprise policy list](https://chromeenterprise.google/policies/)
- [Managing Google Chrome on Windows with Fleet](https://fleetdm.com/guides/managing-chrome-with-fleet)

<meta name="articleTitle" value="Block Chrome extensions on macOS, Windows, and Linux">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-09-29">
<meta name="category" value="guides">
<meta name="description" value="Block, allow, or force-install Chrome extensions on macOS, Windows, and Linux hosts with Fleet using configuration profiles and scripts.">
