# Block Firefox extensions on macOS, Windows, and Linux

Firefox extensions can read page content, cookies, and browsing history. Use Fleet to block extensions you don't trust, allow only the ones you do, and force-install the ones your team needs. This guide covers Mozilla Firefox on macOS, Windows, and Linux.

## Prerequisites

- Hosts enrolled in Fleet.
- The ID of each extension. On a host with the extension installed, open `about:support` and find the extension under **Extensions**. IDs look like `uBlock0@raymondhill.net`. Some IDs are a UUID in braces, like `{446900e4-71c2-419f-a6a7-df9c091e268b}`.
- Windows only: the Firefox ADMX file (`firefox.admx`) deployed to your hosts. See [Deploy the Firefox ADMX file](#deploy-the-firefox-admx-file).

Unlike Chrome and Edge, Firefox has one policy for all of this: `ExtensionSettings`. It maps each extension ID to an `installation_mode`:

| `installation_mode` | What it does |
| ------------------- | ------------ |
| `blocked` | Prevents installation and removes the extension if it's already installed. |
| `allowed` | Lets users install the extension. This is the default. |
| `force_installed` | Installs the extension and prevents users from removing or disabling it. Set `install_url` too. |
| `normal_installed` | Installs the extension, but users can disable it. Set `install_url` too. |

Use the ID `*` to set a default for every extension you don't list. To allow only specific extensions, set `*` to `blocked` and list the extensions you approve as `allowed`. Settings for a specific ID override `*`.

Newer Firefox versions make `install_url` optional for extensions hosted on addons.mozilla.org. Setting it works on all versions.

> **Note:** Unlike Chrome and Edge, which disable blocked extensions, Firefox removes a blocked extension that's already installed.

## macOS

1. Save the following as `firefox-extension-blocklist.mobileconfig`. Replace the extension IDs with your own.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>org.mozilla.firefox</string>
      <key>PayloadIdentifier</key>
      <string>com.example.firefox.extensions.5D2A9E14</string>
      <key>PayloadUUID</key>
      <string>5D2A9E14-3B6C-4F71-8A0E-C4B9D7126F35</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>EnterprisePoliciesEnabled</key>
      <true/>
      <key>ExtensionSettings</key>
      <dict>
        <key>blocked-extension@example.com</key>
        <dict>
          <key>installation_mode</key>
          <string>blocked</string>
        </dict>
        <key>{446900e4-71c2-419f-a6a7-df9c091e268b}</key>
        <dict>
          <key>installation_mode</key>
          <string>blocked</string>
        </dict>
      </dict>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Firefox extension blocklist</string>
  <key>PayloadIdentifier</key>
  <string>com.example.firefox.extensions</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>A8F03C6B-92D5-4E1B-B7A4-1E5C90D2F68A</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

2. In Fleet, go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
3. Choose the `.mobileconfig` file and select **Add profile** to upload it.

> **Warning:** Firefox ignores enterprise policies on macOS unless `EnterprisePoliciesEnabled` is `true`. Keep it in the profile.

To allow only specific extensions, block everything with `*` and mark the extensions you approve as `allowed`. Replace the `ExtensionSettings` dictionary with:

```xml
<key>ExtensionSettings</key>
<dict>
  <key>*</key>
  <dict>
    <key>installation_mode</key>
    <string>blocked</string>
  </dict>
  <key>uBlock0@raymondhill.net</key>
  <dict>
    <key>installation_mode</key>
    <string>allowed</string>
  </dict>
</dict>
```

To force-install an extension, set `installation_mode` to `force_installed` and add an `install_url`. For extensions on addons.mozilla.org, the URL is `https://addons.mozilla.org/firefox/downloads/latest/<add-on slug>/latest.xpi`, where the slug is the last part of the extension's page URL:

```xml
<key>uBlock0@raymondhill.net</key>
<dict>
  <key>installation_mode</key>
  <string>force_installed</string>
  <key>install_url</key>
  <string>https://addons.mozilla.org/firefox/downloads/latest/ublock-origin/latest.xpi</string>
</dict>
```

## Windows

Firefox policies on Windows are ADMX-backed, so deploy the Firefox ADMX file before you set any extension policy.

### Deploy the Firefox ADMX file

1. Download `firefox.admx` from the [Mozilla policy-templates repository](https://github.com/mozilla/policy-templates/tree/master/windows).
2. Save the following as `firefox-admx.xml`. Paste the full contents of `firefox.admx` in place of the comment.

```xml
<Add>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
      <Type>text/plain</Type>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/Firefox/Policy/FirefoxAdmx</LocURI>
    </Target>
    <Data><![CDATA[
      <!-- Paste the full contents of firefox.admx here. Don't modify the file. -->
    ]]></Data>
  </Item>
</Add>
```

3. In Fleet, go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
4. Choose `firefox-admx.xml` and select **Add profile** to upload it.

> **Note:** The `Firefox` in the `LocURI` above must match the `Firefox` in the extension policy `LocURI` below. To learn more about ADMX ingestion, see [Creating Windows CSPs](https://fleetdm.com/guides/creating-windows-csps#ingesting-custom-admx-templates-admxinstall).

### Block extensions

Wait for the ADMX profile to show **Verified** before you add the extension profile.

1. Save the following as `firefox-extension-blocklist.xml`. Replace the extension IDs with your own.

```xml
<Replace>
  <Item>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/Firefox~Policy~firefox~Extensions/ExtensionSettingsOneLine</LocURI>
    </Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data><![CDATA[<enabled/><data id="JSONOneLine" value='{"blocked-extension@example.com":{"installation_mode":"blocked"},"{446900e4-71c2-419f-a6a7-df9c091e268b}":{"installation_mode":"blocked"}}'/>]]></Data>
  </Item>
</Replace>
```

2. In Fleet, go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
3. Choose the `.xml` file and select **Add profile** to upload it.

> **Note:** The JSON goes on one line, inside single quotes, and is limited to 16,384 characters. Avoid apostrophes in the JSON, because they end the value early.

To allow only specific extensions or force-install extensions, change the JSON. The following allows only uBlock Origin and blocks everything else:

```json
{"*":{"installation_mode":"blocked"},"uBlock0@raymondhill.net":{"installation_mode":"allowed"}}
```

The following force-installs uBlock Origin:

```json
{"uBlock0@raymondhill.net":{"installation_mode":"force_installed","install_url":"https://addons.mozilla.org/firefox/downloads/latest/ublock-origin/latest.xpi"}}
```

## Linux

Linux has no MDM configuration profiles. Instead, use a Fleet script to write a policy file. Firefox reads `policies.json` from `/etc/firefox/policies/`.

1. Save the following as `firefox-extension-blocklist.sh`. Replace the extension IDs with your own. Scripts require `fleetd` with scripts enabled.

```bash
#!/bin/bash

mkdir -p /etc/firefox/policies

cat > /etc/firefox/policies/policies.json <<'EOF'
{
  "policies": {
    "ExtensionSettings": {
      "blocked-extension@example.com": {
        "installation_mode": "blocked"
      },
      "{446900e4-71c2-419f-a6a7-df9c091e268b}": {
        "installation_mode": "blocked"
      }
    }
  }
}
EOF

chmod 644 /etc/firefox/policies/policies.json
```

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. Run the script on a host from its **Details** page, or [run it automatically](https://fleetdm.com/guides/policy-automation-run-script) on Linux hosts that fail a policy.

> **Warning:** Firefox reads a single `policies.json`, and this script replaces it. If you already manage other Firefox policies in this file, add `ExtensionSettings` to your existing JSON instead.

To allow only specific extensions or force-install extensions, use the same `ExtensionSettings` shapes shown in the macOS section, written as JSON:

```json
{
  "policies": {
    "ExtensionSettings": {
      "*": { "installation_mode": "blocked" },
      "uBlock0@raymondhill.net": {
        "installation_mode": "force_installed",
        "install_url": "https://addons.mozilla.org/firefox/downloads/latest/ublock-origin/latest.xpi"
      }
    }
  }
}
```

> **Note:** `/etc/firefox/policies/` works for Firefox installed from `deb` and `rpm` packages and for the Ubuntu snap. The Flatpak version can't read the host's `/etc`. For Flatpak, write the file to `/var/lib/flatpak/extension/org.mozilla.firefox.systemconfig/<arch>/<branch>/policies/policies.json`.

## Manage with GitOps

To manage these settings with [GitOps](https://fleetdm.com/docs/configuration/yaml-files), save the files in your GitOps repository and reference them in your fleet's YAML file:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/profiles/firefox-extension-blocklist.mobileconfig
  windows_settings:
    configuration_profiles:
      - path: ../lib/windows/profiles/firefox-admx.xml
      - path: ../lib/windows/profiles/firefox-extension-blocklist.xml
  scripts:
    - path: ../lib/linux/scripts/firefox-extension-blocklist.sh
```

To run the Linux script automatically, add a policy that runs it when a host fails the check. This requires Fleet Premium.

```yaml
policies:
  - name: Linux - Firefox extension policy is present
    query: SELECT 1 FROM file_lines WHERE path = '/etc/firefox/policies/policies.json' AND line LIKE '%ExtensionSettings%';
    platform: linux
    run_script:
      path: ../lib/linux/scripts/firefox-extension-blocklist.sh
```

> **Note:** This policy only checks that the file mentions `ExtensionSettings`. After you change the extension list, run the script again from each host's **Details** page.

## What users see

- **Blocked extension:** If a user tries to install a blocked extension, Firefox tells them it's blocked by their organization: `<name> (<id>) is blocked by your organization.` A blocked extension that's already installed is removed when Firefox applies the policy.
- **Force-installed extension:** Firefox installs it without a prompt. Users can't remove or disable it.
- **Normally installed extension:** Firefox installs it without a prompt. Users can disable it, but not remove it.

## Verify

1. On a host, open `about:policies` in Firefox.
2. Select the **Active** tab.
3. Confirm `ExtensionSettings` appears with the IDs you set.
4. Select the **Errors** tab to check for problems applying the policy.

To check across all your hosts, run this report to list installed Firefox extensions:

```sql
SELECT users.username, firefox_addons.name, firefox_addons.identifier, firefox_addons.disabled
FROM users
CROSS JOIN firefox_addons USING (uid);
```

## Troubleshoot

**`about:policies` says the Enterprise Policies service is inactive**

On macOS, confirm the profile includes `EnterprisePoliciesEnabled` set to `true`. On Linux, confirm the file is at `/etc/firefox/policies/policies.json` and is valid JSON.

**`ExtensionSettings` doesn't appear on Windows**

Confirm the ADMX profile shows **Verified** in Fleet and the name in the `LocURI` (`Firefox`) matches in both profiles. Confirm the JSON is on one line and inside single quotes.

**The extension is still blocked after you allow it**

A setting for a specific ID overrides `*`. Confirm the ID matches exactly, including any braces or `@` domain.

**A force-installed extension doesn't install**

Confirm the `install_url` is reachable from the host and points to the extension's `.xpi` file.

## Further reading

- [Firefox `ExtensionSettings` policy](https://firefox-admin-docs.mozilla.org/reference/policies/extensionsettings/)
- [Firefox policy templates](https://github.com/mozilla/policy-templates)
- [Firefox ADMX OMA-URIs for Intune](https://github.com/mozilla/policy-templates/blob/master/docs/oma-uris.md)

<meta name="articleTitle" value="Block Firefox extensions on macOS, Windows, and Linux">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-09-29">
<meta name="category" value="guides">
<meta name="description" value="Block, allow, or force-install Firefox extensions on macOS, Windows, and Linux hosts with Fleet using configuration profiles and scripts.">
