# Block Safari extensions on macOS

Safari extensions can read page content, cookies, and browsing history. Unlike Chrome, Edge, and Firefox, Safari doesn't use a browser policy file. Use a custom declarative device management (DDM) declaration in Fleet to turn extensions off, allow only the ones you approve, or restrict the sites they can access on macOS.

## Prerequisites

- macOS 15 or later.
- Macs that are supervised. This includes Macs that enrolled through [Automated Device Enrollment (ADE)](https://support.apple.com/en-us/102300).
- The extension's host app installed on the Mac. Safari extensions ship inside apps, and Safari only applies the declaration to extensions whose app is present.

> **Note:** On macOS, Apple only supports this declaration in the user scope. Fleet delivers declarations to the device channel unless `PayloadScope` is `User`, so the declarations in this guide include `"PayloadScope": "User"`. See [Device and user scope](https://fleetdm.com/guides/custom-os-settings#device-and-user-scope).

## Find the extension identifier

Safari identifies each extension with a composed identifier in the format `Identifier (TeamIdentifier)`, like `com.example.WebExtension (ABCDE12345)`.

1. On a Mac with the app installed, find the extension bundle in the app's `PlugIns` folder.
2. Run `codesign` against it:

```bash
codesign -dv "/Applications/Example.app/Contents/PlugIns/Example Extension.appex" 2>&1 | grep -E "^(Identifier|TeamIdentifier)="
```

3. Combine the two values from the output:

```
Identifier=com.example.WebExtension
TeamIdentifier=ABCDE12345
```

Here, the composed identifier is `com.example.WebExtension (ABCDE12345)`.

## Block extensions

1. Save the following as `safari-extension-blocklist.json`. Replace the composed identifier with your own. Keep the `Identifier` to 64 bytes or fewer, and unique among your declarations.

```json
{
  "Type": "com.apple.configuration.safari.extensions.settings",
  "Identifier": "com.example.safari.extensions.blocklist",
  "PayloadScope": "User",
  "Payload": {
    "ManagedExtensions": {
      "com.example.WebExtension (ABCDE12345)": {
        "State": "AlwaysOff"
      }
    }
  }
}
```

2. In Fleet, go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
3. Choose the `.json` file and select **Add profile** to upload it.

Each key under `ManagedExtensions` is a composed identifier, and `State` is one of:

| `State` | What it does |
| ------- | ------------ |
| `Allowed` | Users can turn the extension on or off. |
| `AlwaysOn` | The extension is always on. |
| `AlwaysOff` | The extension is always off. |

### Allow only specific extensions

Use `*` to match every extension, then turn on the ones you approve:

```json
{
  "Type": "com.apple.configuration.safari.extensions.settings",
  "Identifier": "com.example.safari.extensions.allowlist",
  "PayloadScope": "User",
  "Payload": {
    "ManagedExtensions": {
      "*": {
        "State": "AlwaysOff"
      },
      "com.example.WebExtension (ABCDE12345)": {
        "State": "AlwaysOn"
      }
    }
  }
}
```

To turn off every extension, use only the `*` entry.

### Turn off an extension in Private Browsing

To keep an extension on but turn it off in Private Browsing, set `PrivateBrowsing` to `AlwaysOff`:

```json
"com.example.WebExtension (ABCDE12345)": {
  "State": "AlwaysOn",
  "PrivateBrowsing": "AlwaysOff"
}
```

### Deny an extension access to specific sites

To keep an extension on but deny it access to a site and its subdomains, use `DeniedDomains`:

```json
"com.example.WebExtension (ABCDE12345)": {
  "DeniedDomains": ["*example.org"]
}
```

`*example.org` matches `example.org` and its subdomains, like `www.example.org`. Safari also supports `AllowedDomains`. If a domain is in both lists, Safari denies it.

## Manage with GitOps

To manage the declaration with [GitOps](https://fleetdm.com/docs/configuration/yaml-files), save the JSON file in your GitOps repository and reference it in your fleet's YAML file:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/profiles/safari-extension-blocklist.json
```

Keep `"PayloadScope": "User"` in the JSON file. Then run GitOps to apply the change.

## What users see

- **Turned-off extension (`AlwaysOff`):** The extension is off, and users can't turn it on.
- **Turned-on extension (`AlwaysOn`):** The extension is on, and users can't turn it off.
- **Managed settings:** In **Safari > Settings > Extensions**, the controls for a managed extension are greyed out. Safari also shows a note that device management configured them.
- **Extensions without an entry:** Users keep control of them.

## Verify

1. In Fleet, go to **Hosts** and select a host.
2. Select the **OS settings** tab.
3. Confirm the profile shows **Verified**.
4. On the Mac, open Safari, go to **Settings > Extensions**, and confirm the extension is off.

## Troubleshoot

**The profile shows Failed or Pending**

Confirm the Mac runs macOS 15 or later and is supervised. Confirm the declaration includes `"PayloadScope": "User"`. User-scoped profiles go to the account that enrolled the Mac. If that account was deleted, see [Custom OS settings](https://fleetdm.com/guides/custom-os-settings).

**The extension is still on**

Confirm the composed identifier matches exactly, including the space and parentheses. Confirm the extension's app is installed on the Mac.

**Users can turn on an extension you didn't list**

Extensions without an entry stay under the user's control. Add a `*` entry set to `AlwaysOff` to turn off everything you didn't list.

## Further reading

- [Apple: Safari extension settings declaration](https://github.com/apple/device-management/blob/release/declarative/declarations/configurations/safari.extensions.settings.yaml)
- [Custom OS settings in Fleet](https://fleetdm.com/guides/custom-os-settings)

<meta name="articleTitle" value="Block Safari extensions on macOS">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-09-29">
<meta name="category" value="guides">
<meta name="description" value="Turn off, allow only specific, or restrict Safari extensions on macOS with a custom DDM declaration in Fleet.">
