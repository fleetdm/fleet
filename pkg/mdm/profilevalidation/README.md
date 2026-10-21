This package contains types and functions for validating MDM profiles on Apple and Windows devices.

It works by parsing the respective sources with [`tools/profile-schema-gen`](../../../tools/profile-schema-gen), applying Fleet's overrides from `inputs/overrides.yml`, and writing the result as a Fleet-defined schema.

The output of this process lands in `outputs/profile_schema.json`, which is embedded into the Go code and parsed into a JSON struct on startup.

## Sources

| Section | Upstream | Pin |
|---|---|---|
| `apple_envelope`, `apple_profiles` | `mdm/profiles/*.yaml` in [apple/device-management](https://github.com/apple/device-management), `release` branch | `version.apple` in the output |
| `apple_declaration_envelope`, `apple_declarations` | `declarative/declarations/{configurations,assets,management,activations}/*.yaml`, same repo | same |
| `windows`, `windows_xsds` | Microsoft's DDF v2 zip, the drop linked from the [Learn DDF page](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf) | `version.windows` |
| `windows` (`source: admx_template`) | `website/profile-generator/schema/windows-admx-templates.json` in this repo: the AI profile generator's third-party ADMX templates | `version.windows_templates` |

The pins are the output's own `version` block:
- **`apple`:** `release` (Apple's commit title, e.g. `Release-v27.0`) and `commit`.
- **`windows`:** `release` (the zip's name, e.g. `DDFv2Feb2026`), `url`, and the zip's `sha256`.
- **`windows_templates`:** the template file's `sha256` and its own `generated_at`.

## Regenerating

Run from the repository root:

```sh
go run ./tools/profile-schema-gen -update   # move to the latest release-branch commit
go run ./tools/profile-schema-gen           # regenerate at the pinned commit, offline
```

- **`-update`** moves both platforms to the latest upstream, into `tools/profile-schema-gen/downloads/` (ignored by git):
  - **Apple:** one `api.github.com` call finds the head of `release` (60 per hour unauthenticated), then that commit's tarball comes from `codeload.github.com`, which isn't API rate limited.
  - **Windows:** Microsoft has no API or public source repo for DDF drops, so the tool reads the Learn DDF page and takes the first `DDFv2*.zip` link, which is the current drop. If the page stops linking one, generation fails. The zip's sha256 becomes the pin. If the bytes change at an unchanged URL, the tool logs a warning.
- **Without `-update`** the tool makes no network calls. It needs `downloads/` to hold what the output pins. Add `-fetch-pinned` to download it when it's missing: the Apple commit from codeload with no API call, and the pinned DDF URL, which must still match the pinned sha256.
- **ADMX templates** are read from the working tree on every run, never downloaded. Changing that file means regenerating.
- **Other flags:** `-overrides <path>` picks the overrides file (default `inputs/overrides.yml`; `""` skips overrides). `-out <path>` writes somewhere else. `-v` logs every file and decision.

CI, in `.github/workflows/profile-validation-schema.yml`:
- **On PRs** that touch this package, the generator, or the ADMX template file, it runs `-fetch-pinned` and fails if the committed schema differs.
- **Daily**, it runs `-update` and opens or updates a PR when Apple's release or Microsoft's DDF drop changes the schema. Tests and the generator's warnings run on that PR through the PR check. If the update itself fails, nothing is written, so there's no PR; the job posts the generator's errors to #help-engineering instead.

Every run is checked:
- **Self-check before writing:** the generator parses its own output with this package's `Parse`. If the floors or must-haves below fail, nothing is written. A reorganized upstream therefore fails generation instead of shrinking the schema.
- **Stable reruns:** `generated_at` only moves when the content does, so rerunning against the same pin is byte-identical.
- **Atomic write:** the output is written to a temp file and renamed, so a failed run never leaves a partial file.

## Output shape

```jsonc
{
  "format_version": 1,
  "version": {
    "apple": {"release": "Release-v27.0", "commit": "<sha>"},
    "windows": {"release": "DDFv2Feb2026", "url": "https://download.microsoft.com/…/DDFv2Feb2026.zip", "sha256": "<sha256>"},
    "windows_templates": {"sha256": "<sha256>", "generated_at": "<RFC 3339>"},
    "generated_at": "<RFC 3339>"
  },
  "apple_envelope": {"TopLevel": {…}, "CommonPayloadKeys": {…}},   // the profile itself
  "apple_declaration_envelope": {…},                              // keys every declaration carries
  "apple_profiles": {"<PayloadType>": {…}},
  "apple_declarations": {"<declaration Type>": {…}},
  "windows": {"<LocURI>": {…}},                                     // DDF nodes and ADMX template policies
  "windows_xsds": {"<sha256>": "<xs:schema …>"}                     // XSD documents chr nodes reference
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

**Windows nodes** (each entry under `windows`, keyed by its LocURI as declared, with `*` for every dynamic segment) have:
- `format`: one of `int`, `chr`, `bool`, `b64`, `xml`, `node`, `null`, `bin`, `time`, `date`, `float`, or `any`; a list when an override widened it.
- `scope`: `user` under `./User/`, otherwise `device`.
- `source`: `windows` for Microsoft's DDF nodes, `admx_template` for template policies.
- At most one allowed-values kind: `enum`, `flag` (bit values), `range` (`min`/`max`), `regex`, `xsd` (a key into `windows_xsds`), or `admx_backed`.
- `delimiter`, when the value is a list of allowed values joined by it.
- `min_build`: the first build per servicing branch, as a list.
- `deprecated`: `{"build": "…"}`, or `{}` when the DDF names no build.
- `override`.

How the data is derived from Apple's YAML:
- **Grouping:** Apple's payload types are keyed by the `payloadtype` inside each file, not the filename. Files that share a type (the six `com.apple.MCX` files, the two `com.apple.extensiblesso` files) are merged into one entry. Where they disagree, the looser definition wins.
- **What's dropped:** descriptions and defaults. `ServerToken` is also dropped from the declaration envelope, because Fleet's server sets it.
- **Constraints:** recorded even where no validator enforces them yet.
- **Unknown values:** carried as declared.

How the Windows data is derived:
- **Every DDF node is kept,** settings and interior nodes alike. Interior nodes have `format: node`. Profiles `<Add>` them to create a dynamic instance before setting its children, e.g. Fleet's SCEP profiles add `ClientCertificateInstall/SCEP/<id>`.
- **Paths are kept as declared.** The Firewall CSP is rooted at bare `./Vendor/MSFT`, and policies at `./Device` or `./User`. Treating those roots as equivalent is the validator's job.
- **Applicability is inherited:** a node without `OsBuildVersion` takes its nearest ancestor's, as the DDF schema specifies. Placeholder builds such as `99.9.99999` are carried as-is.
- **XSDs** are stored once in `windows_xsds`, keyed by sha256, and nodes reference them.
- **`JSON`, `SDDL`, and `None` allowed values** leave the node unconstrained. Unknown DDF elements or allowed-value types are logged as warnings, which the PR check comments on the PR.
- **ADMX template policies** come from the website's template file. Each one is added as a `chr` node with `source: admx_template`, under each scope it lists, at the path built from the AppName the AI profile generator installs it under (e.g. `GoogleChrome~Policy~googlechrome~Startup/HomepageLocation`).

### Compatibility rule

Servers run several versions behind and will be handed newer schemas, so newer data must never make an older validator stricter.

- **Same `format_version`:** a new optional field keeps it. Unknown fields are ignored when parsing, unknown values pass, and `type` lists only widen.
- **New `format_version`:** required when changing or removing what a field means. `Parse` rejects any `format_version` other than the one it knows, so an old server refuses a document it can't read correctly instead of misreading it.

## Floors and must-haves

`Parse` also rejects a document that:
- has fewer than **100** Apple payload types;
- has fewer than **40** `com.apple.configuration.*` declarations;
- is missing `com.apple.applicationaccess` with its `allowCamera` key, or `com.apple.configuration.passcode.settings`;
- is missing either envelope;
- has fewer than **4,000** Windows settings (`source: windows`, not `node`; template policies don't count, so they can't hide a shrunken DDF drop);
- is missing `./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock`;
- has a Windows node whose `xsd` isn't in `windows_xsds`.

Release-v27.0 has 119 payload types and 48 configuration declarations. DDFv2Feb2026 has 5,146 settings and 809 interior nodes; the templates add 2,570 policies across both scopes. Old servers apply these checks to every schema they're later handed, so keep the floors well below today's counts and change them only with evidence.

## Using the schema

- **At startup:** `fleet serve` calls `ParseProfileValidationSchema()`, which parses the embedded copy. It refuses to start if that fails.
- **Reading it:** callers use `Current()`, then the lookups, which return nil for anything unknown:
  - `AppleProfile(payloadType)` and `AppleDeclaration(declarationType)`.
  - `WindowsNode(locURI)` returns the exact node, or else the node whose `*` segments match, preferring the fewest `*`. Paths are compared as written.
- **Swapping it:** `Replace(schema)` swaps in a newer schema from `Parse`, for example a downloaded copy. New `Current()` calls see it, and a schema a caller already holds never changes. Don't modify a schema you get from `Current()`.
- **Dependencies:** the package doesn't import `server/fleet`.

## Overrides

Apple's and Microsoft's published schemas don't match what devices accept in every case. `inputs/overrides.yml` records Fleet's corrections. The generator applies them when it writes `outputs/profile_schema.json`, so the overrides travel with the schema, including newer copies downloaded by older servers.

After editing the file, regenerate from the repository root and commit both files:

```sh
go run ./tools/profile-schema-gen
```

This doesn't need the network as long as `tools/profile-schema-gen/downloads/` holds the pinned Apple commit and DDF drop. On a fresh checkout, add `-fetch-pinned`.

### Rules for every entry

- **`kind`** names the section of the schema the entry targets, and the field that names the target:

  | `kind` | Target field | Targets |
  |---|---|---|
  | `apple_profile` | `payload_type` | A .mobileconfig payload type, e.g. `com.apple.MCX`. Use `TopLevel` for the profile's outer dictionary and `CommonPayloadKeys` for keys every payload carries. |
  | `apple_declaration` | `declaration_type` | A DDM declaration type, e.g. `com.apple.configuration.passcode.settings`. Use `DeclarationBase` for the keys every declaration carries. |
  | `windows` | `loc_uri` | A node in `windows`, matched exactly. Target a dynamic node as written, e.g. `./Vendor/MSFT/Firewall/MdmStore/FirewallRules/*/Action/Type`. `loosen` only. |

- **`reason`** is required. Say what devices accept and where it was seen, for example "CIS benchmark setting; used in it-and-security prevent-autologon profile." It's copied into the output as `override` on the entry it changed, so validators and the UI can explain why something passes.
- **Overrides only widen.** None of them can make the schema reject something the upstream schema accepts.
- **Mistakes fail generation**, and nothing is written until they're fixed. The generator checks in two passes:
  - First the file itself: unknown or misspelled fields, a misspelled kind, a missing reason, and two entries for the same target. Every problem in the file is reported together.
  - Then each entry against the schema, for example a target that doesn't exist.
- **Key paths** use dots between dictionary keys and `[]` for an array's items, e.g. `PayloadContent.SubjectAltName.uniformResourceIdentifier` or `Dock[].Type`. Apple key names can contain dots or spaces (`com.apple.login.mcx.DisableAutoLoginClient`, `Key Type`), so each step matches the longest key name that exists. `ANY` steps into Apple's `ANY` subkey (the `values` of an `open` dictionary) when no key by that name exists, e.g. `AllowedSystemExtensions.ANY[]`.

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

```yaml
loosen:
  - kind: windows
    loc_uri: ./Device/Vendor/MSFT/Policy/Config/Example/Setting   # no key_path
    action: types
    types: [chr]
    reason: The DDF says int, but devices accept a string.
```

- The target must exist upstream: the key at `key_path` for Apple, or the node at `loc_uri` for Windows. If it's renamed or removed, generation fails so someone can look.
- A Windows target must be a setting. Interior nodes (`format: node`) can't be loosened.
- `action: types` adds the listed types to the key's type or the node's format. It never replaces it, so the target still accepts what upstream allows. The values must come from the entry's own vocabulary:
  - Apple: `string`, `integer`, `real`, `boolean`, `date`, `data`, `array`, `dictionary`.
  - Windows: `int`, `chr`, `bool`, `b64`, `xml`, `null`, `bin`, `time`, `date`, `float`.
- `action: any` makes the target accept any value and clears its value constraints. A Windows node keeps its `scope`, `min_build`, `deprecated`, and `source: windows`.

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
- If Apple later ships the type, generation keeps the `allow` definition and replaces Apple's, so no profile that passed gets stricter. The entry then counts as Apple's toward the floor (`source: apple`) and keeps its `override` reason. It logs a warning: replace the entry with `add` and `loosen` entries to adopt Apple's schema.

### What isn't an override

- **Keys Fleet sets itself:** don't add them here. The generator already drops `ServerToken` from the declaration envelope, since Fleet's server sets it when serving a declaration.
- **Typos in a profile:** fix the profile, not the schema.

## Notes for validators

Things the schema records but doesn't decide; the validator (#55792) owns them.

- **Fleet variables:** substitute or skip `$FLEET_VAR_*` and `${FLEET_SECRET_*}` before checking a value. Otherwise `${FLEET_SECRET_LENGTH}` fails an `int` range.
- **Root equivalence:** the DDF roots some CSPs at bare `./Vendor/MSFT` and others at `./Device/Vendor/MSFT`; decide which spellings are equivalent.
- **Interior `*` nodes:** a `regex` on a `node` entry constrains the instance's name, not a value. For example, `FirewallRules/*` names can't contain `|` or `/`.
- **ADMX policies** under `./{Device|User}/Vendor/MSFT/Policy/Config/{area}/{policy}`:
  - **An area without `~`** is a built-in Policy CSP area (none of the 262 contain `~`). It must match a DDF node exactly. Don't fall back to `Policy/Config/*/*`, a generic placeholder Microsoft declares as `null`, or a typo in a built-in setting would pass.
  - **An area with `~`** is an ingested ADMX: `{AppName}~{SettingType}~{Category…}`, created on the device by installing an ADMX through `Policy/ConfigOperations/ADMXInstall/{AppName}/…`. Its policies are always `chr`.
    - **Strict mode** (AI-generated profiles) matches template policies exactly, AppName included.
    - **Lenient mode** ignores the AppName and matches on `{SettingType}~{Category…}/{policy}`, because admins pick their own AppName (Fleet's docs install Chrome as `Chrome`, not `GoogleChrome`). An unknown policy or app isn't an error: it may be a custom ADMX, or a newer vendor release than the template data.
- **Future:** the template file also lists each policy's ADMX elements (`admxElements`: element ids, types, and enum values) for 642 of its 1,575 policies. The schema doesn't carry them yet; they could validate the `<data id="…" value="…"/>` payload of known policies.
