This package contains types and functions for validating MDM profiles on Apple and Windows devices.

It works by parsing the respective sources with [`tools/profile-schema-gen`](../../../tools/profile-schema-gen), applying Fleet's overrides from `inputs/overrides.yml`, and writing the result as a Fleet-defined schema.

The output of this process lands in `outputs/profile_schema.json`, which is embedded into the Go code and parsed into a JSON struct on startup.

## Sources

| Section | Upstream | Pin |
|---|---|---|
| `apple_envelope`, `apple_profiles` | `mdm/profiles/*.yaml` in [apple/device-management](https://github.com/apple/device-management), `release` branch | `version.apple` in the output |
| `apple_declaration_envelope`, `apple_declarations` | `declarative/declarations/{configurations,assets,management,activations}/*.yaml`, same repo | same |
| `windows`, `windows_xsds` | Empty for now. #55790 fills them. | |

The pin is the output's own `version` block: `release` (Apple's commit title, e.g. `Release-v27.0`) and `commit`.

## Regenerating

Run from the repository root:

```sh
go run ./tools/profile-schema-gen -update   # move to the latest release-branch commit
go run ./tools/profile-schema-gen           # regenerate at the pinned commit, offline
```

- **`-update`** makes one `api.github.com` call to find the head of `release`. That's 60 per hour unauthenticated. It then downloads that commit's tarball from `codeload.github.com`, which isn't API rate limited, into `tools/profile-schema-gen/downloads/` (ignored by git).
- **Without `-update`** the tool makes no network calls. It needs `downloads/` to hold the commit the output pins. Add `-fetch-pinned` to download that commit when it's missing; this uses codeload only, with no API call.
- **Other flags:** `-overrides <path>` picks the overrides file (default `inputs/overrides.yml`; `""` skips overrides). `-out <path>` writes somewhere else. `-v` logs every file and decision.

CI, in `.github/workflows/profile-validation-schema.yml`:
- **On PRs** that touch this package or the generator, it runs `-fetch-pinned` and fails if the committed schema differs.
- **Daily**, it runs `-update` and opens or updates a PR when Apple's release changes the schema.

Every run is checked:
- **Self-check before writing:** the generator parses its own output with this package's `Parse`. If the floors or must-haves below fail, nothing is written. A reorganized upstream therefore fails generation instead of shrinking the schema.
- **Stable reruns:** `generated_at` only moves when the content does, so rerunning against the same pin is byte-identical.
- **Atomic write:** the output is written to a temp file and renamed, so a failed run never leaves a partial file.

## Output shape

```jsonc
{
  "format_version": 1,
  "version": {"apple": {"release": "Release-v27.0", "commit": "<sha>"}, "generated_at": "<RFC 3339>"},
  "apple_envelope": {"TopLevel": {…}, "CommonPayloadKeys": {…}},   // the profile itself
  "apple_declaration_envelope": {…},                              // keys every declaration carries
  "apple_profiles": {"<PayloadType>": {…}},
  "apple_declarations": {"<declaration Type>": {…}},
  "windows": {}, "windows_xsds": {}
}
```

**Payload types** (each entry under `apple_profiles` or `apple_declarations`, and each envelope entry) have:
- `platforms`
- `source`: `apple`, or `override` for `allow` entries
- `keys`
- `open` and `values`, for types that accept any key at the top level
- `override`: the reason, if an override touched the type

**Keys** have:
- `type`: a string, or a list when an override widened it. One of `string`, `integer`, `real`, `boolean`, `date`, `data`, `array`, `dictionary`, `any`.
- `required`, `rangelist`, `range`, `format`, `repetition`, `valuetype`
- `items` (array elements), `keys` (dictionary subkeys)
- `open` and `values`: Apple's `ANY` subkey. Any key name is accepted, and `values` describes the values.
- `ref`: points back at an enclosing definition, for types that contain themselves, e.g. `"ref": "Dock[]"`
- `platforms`: recorded only where they differ from the parent's
- `override`

How the data is derived from Apple's YAML:
- **Grouping:** Apple's payload types are keyed by the `payloadtype` inside each file, not the filename. Files that share a type (the six `com.apple.MCX` files, the two `com.apple.extensiblesso` files) are merged into one entry. Where they disagree, the looser definition wins.
- **What's dropped:** descriptions and defaults. `ServerToken` is also dropped from the declaration envelope, because Fleet's server sets it.
- **Constraints:** recorded even where no validator enforces them yet.
- **Unknown values:** carried as declared.

### Compatibility rule

Servers run several versions behind and will be handed newer schemas, so newer data must never make an older validator stricter.

- **Same `format_version`:** a new optional field keeps it. Unknown fields are ignored when parsing, unknown values pass, and `type` lists only widen.
- **New `format_version`:** required when changing or removing what a field means. `Parse` rejects any `format_version` other than the one it knows, so an old server refuses a document it can't read correctly instead of misreading it.

## Floors and must-haves

`Parse` also rejects a document that:
- has fewer than **100** Apple payload types;
- has fewer than **40** `com.apple.configuration.*` declarations;
- is missing `com.apple.applicationaccess` with its `allowCamera` key, or `com.apple.configuration.passcode.settings`;
- is missing either envelope.

Release-v27.0 has 119 payload types and 48 configuration declarations. Old servers apply these checks to every schema they're later handed, so keep the floors well below today's counts and change them only with evidence.

## Using the schema

- **At startup:** `fleet serve` calls `ParseProfileValidationSchema()`, which parses the embedded copy. It refuses to start if that fails.
- **Reading it:** callers use `Current()`, then the lookups `AppleProfile(payloadType)` and `AppleDeclaration(declarationType)`, which return nil for unknown types.
- **Swapping it:** `Replace(schema)` swaps in a newer schema from `Parse`, for example a downloaded copy. New `Current()` calls see it, and a schema a caller already holds never changes. Don't modify a schema you get from `Current()`.
- **Dependencies:** the package doesn't import `server/fleet`.

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
