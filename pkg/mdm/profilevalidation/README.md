This package contains types and functions for validating MDM profiles on Apple and Windows devices.

It works by parsing the respective sources with [`tools/profile-schema-gen`](../../../tools/profile-schema-gen), applying Fleet's overrides from `inputs/overrides.yml`, and writing the result as a Fleet-defined schema.

The output of this process lands in `outputs/profile_schema.json`, which is embedded into the Go code and parsed into a JSON struct on startup.

## Overrides

Apple's published schema doesn't match what devices accept in every case. `inputs/overrides.yml` records Fleet's corrections. The generator applies them when it writes `outputs/profile_schema.json`, so the overrides travel with the schema, including newer copies downloaded by older servers.

After editing the file, regenerate from the repository root and commit both files:

```sh
go run ./tools/profile-schema-gen
```

This doesn't need the network as long as `tools/profile-schema-gen/downloads/` holds the pinned Apple commit. On a fresh checkout, run with `-update` once.

### Rules for every entry

- **`kind`** names the section of the schema the entry targets, and the field that names the target:

  | `kind` | Target field | Targets |
  |---|---|---|
  | `apple_profile` | `payload_type` | A .mobileconfig payload type, e.g. `com.apple.MCX`. Use `TopLevel` for the profile's outer dictionary and `CommonPayloadKeys` for keys every payload carries. |
  | `apple_declaration` | `declaration_type` | A DDM declaration type, e.g. `com.apple.configuration.passcode.settings`. Use `DeclarationBase` for the keys every declaration carries. |
  | `windows` | `loc_uri` | Accepted, but has no effect until the Windows sections exist. |

- **`reason`** is required. Say what devices accept and where it was seen, for example "CIS benchmark setting; used in it-and-security prevent-autologon profile." It's copied into the output as `override` on the entry it changed, so validators and the UI can explain why something passes.
- **Overrides only widen.** None of them can make the schema reject something Apple's schema accepts.
- **Mistakes fail generation**, and nothing is written until they're fixed. The generator checks in two passes:
  - First the file itself: unknown or misspelled fields, a misspelled kind, a missing reason, and two entries for the same target. Every problem in the file is reported together.
  - Then each entry against the schema, for example a target that doesn't exist.
- **Key paths** use dots between dictionary keys and `[]` for an array's items, e.g. `PayloadContent.SubjectAltName.uniformResourceIdentifier` or `Dock[].Type`. Apple key names can contain dots or spaces (`com.apple.login.mcx.DisableAutoLoginClient`, `Key Type`), so each step matches the longest key name that exists.

### `loosen`: a key type is too strict

```yaml
loosen:
  - kind: apple_profile
    payload_type: com.apple.security.scep
    key_path: PayloadContent.SubjectAltName.uniformResourceIdentifier
    action: types        # or: any
    types: [array]
    reason: Apple's schema says string, but devices accept a list of URIs.
```

- The key at `key_path` must exist in Apple's schema. If Apple renames or removes it, generation fails so someone can look.
- `action: types` adds the listed types to the key's type. It never replaces the type, so the key still accepts what Apple's schema allows.
- `action: any` makes the key accept any value.

### `add`: a key Apple's schema doesn't list

```yaml
add:
  - kind: apple_profile
    payload_type: com.apple.MCX
    name: forceInternetSharingOff
    key: {type: boolean}
    reason: CIS benchmark setting; used in it-and-security disable-internet-sharing profile.
  - kind: apple_profile
    payload_type: com.apple.security.scep
    key_path: PayloadContent          # the dictionary to add to; leave out for the top level
    name: ExtendedKeyUsage
    key: {type: array, items: {type: string}}
    reason: Extended key usage OIDs for the SCEP certificate.
```

- `key_path` is the dictionary the key goes into. It must exist and be a dictionary. Leave it out to add the key at the payload's top level.
- `name` is the new key's name, and it can contain dots.
- `key` uses the same shape as a key in `outputs/profile_schema.json`: `type` (required; a string, or a list of types), and optionally `keys`, `items`, `values`, `open`, `rangelist`, `range`, `format`.
- If Apple later adds the same key, generation keeps going. It merges the two definitions to the looser one and logs a warning. Remove the entry when you see that warning.

### `allow`: a payload type outside Apple's schema

```yaml
allow:
  - kind: apple_profile
    payload_type: com.google.Chrome
    reason: Vendor preference domain written by the AI profile generator.
    keys:                 # optional
      HomepageLocation: {type: string}
```

- `allow` is for `apple_profile` only. Third-party preference domains exist only in profiles, and every declaration type is Apple's.
- The type is marked `"source": "override"`. It stays open, so keys not listed under `keys` still pass. `allow` says the type is legitimate, not that its key list is complete.
- `TopLevel` and `CommonPayloadKeys` can't be allowed; use `add` for their keys.
- If Apple later ships the type, generation keeps Apple's definition, merges the listed keys into it loosely, and logs a warning. Remove the entry when you see that warning.

### What isn't an override

- **Keys Fleet sets itself:** don't add them here. The generator already drops `ServerToken` from the declaration envelope, since Fleet's server sets it when serving a declaration.
- **Typos in a profile:** fix the profile, not the schema.
