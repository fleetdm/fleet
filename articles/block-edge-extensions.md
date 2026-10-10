# Block Microsoft Edge extensions on macOS, Windows, and Linux

Edge extensions can read page content, cookies, and browsing history. Use Fleet to block extensions you don't trust, allow only the ones you do, and force-install the ones your team needs. This guide covers Microsoft Edge on macOS, Windows, and Linux.

## Prerequisites

- Hosts enrolled in Fleet.
- The ID of each extension. Find it at the end of the extension's URL on the Microsoft Edge Add-ons website, or on `edge://extensions` with **Developer mode** turned on. IDs are 32 lowercase letters, like `abcdefghijklmnopabcdefghijklmnop`.
- Windows only: the Edge ADMX file (`msedge.admx`) deployed to your hosts. See [Deploy the Edge ADMX file](#deploy-the-edge-admx-file).

Edge offers three extension policies:

| Policy | What it does |
| ------ | ------------ |
| `ExtensionInstallBlocklist` | Blocks the listed extensions. Use `*` to block everything except the allowlist. |
| `ExtensionInstallAllowlist` | Exempts the listed extensions from the blocklist. |
| `ExtensionInstallForcelist` | Installs the listed extensions and prevents users from removing or disabling them. Overrides the blocklist. |

> **Note:** Blocking an extension that's already installed disables it and prevents users from re-enabling it. It doesn't uninstall the extension.

## macOS

1. Save the following as `edge-extension-blocklist.mobileconfig`. Replace the extension IDs with your own.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.microsoft.Edge</string>
      <key>PayloadIdentifier</key>
      <string>com.example.edge.extensions.3C1F6A52</string>
      <key>PayloadUUID</key>
      <string>3C1F6A52-8D7B-4E0A-9B34-6F2A1C5D7E90</string>
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
  <string>Edge extension blocklist</string>
  <key>PayloadIdentifier</key>
  <string>com.example.edge.extensions</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>B7E4D209-51AC-4F86-A0C3-92D8E6F1B473</string>
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

To force-install an extension, add it to `ExtensionInstallForcelist` as `<extension ID>;<update URL>`. For extensions on the Microsoft Edge Add-ons website, the update URL is `https://edge.microsoft.com/extensionwebstorebase/v1/crx`:

```xml
<key>ExtensionInstallForcelist</key>
<array>
  <string>abcdefghijklmnopabcdefghijklmnop;https://edge.microsoft.com/extensionwebstorebase/v1/crx</string>
</array>
```

## Windows

Edge policies on Windows are ADMX-backed, so deploy the Edge ADMX file before you set any extension policy.

### Deploy the Edge ADMX file

1. Download the Edge policy templates from the [Microsoft Edge Enterprise landing page](https://aka.ms/EdgeEnterprise) and extract `msedge.admx`.
2. Save the following as `edge-admx.xml`. Paste the full contents of `msedge.admx` in place of the comment.

```xml
<Add>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
      <Type>text/plain</Type>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/Edge/Policy/EdgeAdmx</LocURI>
    </Target>
    <Data><![CDATA[
      <!-- Paste the full contents of msedge.admx here. Don't modify the file. -->
    ]]></Data>
  </Item>
</Add>
```

3. In Fleet, go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
4. Choose `edge-admx.xml` and select **Add profile** to upload it.

> **Note:** The `Edge` in the `LocURI` above must match the `Edge` in the extension policy `LocURI` below. To learn more about ADMX ingestion, see [Creating Windows CSPs](https://fleetdm.com/guides/creating-windows-csps#ingesting-custom-admx-templates-admxinstall).

### Block extensions

Wait for the ADMX profile to show **Verified** before you add the extension profile.

1. Save the following as `edge-extension-blocklist.xml`. Replace the extension IDs with your own.

```xml
<Replace>
  <Item>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/Edge~Policy~microsoft_edge~Extensions/ExtensionInstallBlocklist</LocURI>
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
| Allowlist | `Edge~Policy~microsoft_edge~Extensions/ExtensionInstallAllowlist` | `ExtensionInstallAllowlistDesc` |
| Force-install | `Edge~Policy~microsoft_edge~Extensions/ExtensionInstallForcelist` | `ExtensionInstallForcelistDesc` |

For the blocklist wildcard, use `*` as the only entry: `value="1&#xF000;*"`. For force-install entries, use `<extension ID>;<update URL>`, like `1&#xF000;abcdefghijklmnopabcdefghijklmnop;https://edge.microsoft.com/extensionwebstorebase/v1/crx`.

## Linux

Linux has no MDM configuration profiles. Instead, use a Fleet script to write a managed policy file. Edge reads every JSON file in `/etc/opt/edge/policies/managed/`.

1. Save the following as `edge-extension-blocklist.sh`. Replace the extension IDs with your own. Scripts require `fleetd` with scripts enabled.

```bash
#!/bin/bash

mkdir -p /etc/opt/edge/policies/managed

cat > /etc/opt/edge/policies/managed/fleet-extensions.json <<'EOF'
{
  "ExtensionInstallBlocklist": [
    "abcdefghijklmnopabcdefghijklmnop",
    "ponmlkjihgfedcbaponmlkjihgfedcba"
  ]
}
EOF

chmod 644 /etc/opt/edge/policies/managed/fleet-extensions.json
```

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. Run the script on a host from its **Details** page, or [run it automatically](https://fleetdm.com/guides/policy-automation-run-script) on Linux hosts that fail a policy.

To allow only specific extensions or force-install extensions, add `ExtensionInstallAllowlist` or `ExtensionInstallForcelist` to the same JSON object:

```json
{
  "ExtensionInstallBlocklist": ["*"],
  "ExtensionInstallAllowlist": ["abcdefghijklmnopabcdefghijklmnop"],
  "ExtensionInstallForcelist": [
    "abcdefghijklmnopabcdefghijklmnop;https://edge.microsoft.com/extensionwebstorebase/v1/crx"
  ]
}
```

## Manage with GitOps

To manage these settings with [GitOps](https://fleetdm.com/docs/configuration/yaml-files), save the files in your GitOps repository and reference them in your fleet's YAML file:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/profiles/edge-extension-blocklist.mobileconfig
  windows_settings:
    configuration_profiles:
      - path: ../lib/windows/profiles/edge-admx.xml
      - path: ../lib/windows/profiles/edge-extension-blocklist.xml
  scripts:
    - path: ../lib/linux/scripts/edge-extension-blocklist.sh
```

To run the Linux script automatically, add a policy that runs it when a host fails the check. This requires Fleet Premium.

```yaml
policies:
  - name: Linux - Edge extension blocklist is present
    query: SELECT 1 FROM file WHERE path = '/etc/opt/edge/policies/managed/fleet-extensions.json';
    platform: linux
    run_script:
      path: ../lib/linux/scripts/edge-extension-blocklist.sh
```

> **Note:** This policy only checks that the file exists. After you change the extension list, run the script again from each host's **Details** page.

## What users see

- **Blocked extension:** A blocked extension that's already installed is disabled, and users can't turn it back on. It isn't uninstalled. If a user tries to install a blocked extension from the Microsoft Edge Add-ons website, Edge shows an error.
- **Force-installed extension:** Edge installs it without a prompt. Users can't uninstall or disable it.
- **Custom message:** To add your own text to the error on the Microsoft Edge Add-ons website, such as who to contact for an exception, use the `blocked_install_message` field of the [`ExtensionSettings` policy](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-manage-extensions-ref-guide).

## Verify

1. On a host, open `edge://policy` in Edge.
2. Select **Reload policies**.
3. Confirm the extension policies appear with the IDs you set and the status **OK**.

On Windows, you can also check `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Edge` in the registry.

To check across all your hosts, run this report to list installed Edge extensions:

```sql
SELECT users.username, chrome_extensions.browser_type, chrome_extensions.name, chrome_extensions.identifier
FROM users
CROSS JOIN chrome_extensions USING (uid)
WHERE chrome_extensions.browser_type IN ('edge', 'edge_beta');
```

## Troubleshoot

**Policies don't appear on `edge://policy`**

Restart Edge and select **Reload policies**. On Windows, confirm the ADMX profile shows **Verified** in Fleet and the name in the `LocURI` (`Edge`) matches in both profiles.

**The extension is still installed after you block it**

Edge disables blocked extensions but doesn't uninstall them. Confirm the ID matches exactly, and check `edge://policy` for errors on the policy.

**A force-installed extension doesn't install**

Confirm the update URL is reachable from the host and the entry uses the `<extension ID>;<update URL>` format.

## Further reading

- [Use group policies to manage Microsoft Edge extensions](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-manage-extensions-policies)
- [Configure Microsoft Edge using Mobile Device Management](https://learn.microsoft.com/en-us/deployedge/configure-edge-with-mdm)
- [Microsoft Edge browser policy reference](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-policies)

<meta name="articleTitle" value="Block Microsoft Edge extensions on macOS, Windows, and Linux">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-09-29">
<meta name="category" value="guides">
<meta name="description" value="Block, allow, or force-install Microsoft Edge extensions on macOS, Windows, and Linux hosts with Fleet using configuration profiles and scripts.">
