# Build and validate configuration profiles with AI instead of a GUI

Most admins build configuration profiles in a GUI. In Intune you pick a setting out of the settings catalog. In Workspace ONE you pick a profile payload. On Apple you open iMazing Profile Editor or ProfileCreator and pick from a payload list. In every case the picker constrains you to settings that exist, with values the platform accepts, and that constraint is the real value of the tool. Everything else about it, the clicking, the exporting, the copying into a repo, is transcription.

Every one of those pickers is a GUI over a published schema. Apple's payload keys come from apple/device-management and ProfileManifests. Intune's settings catalog is generated from Microsoft's CSP Device Description Framework (DDF) files. Android's settings are the fields of the Android Management API `Policy` object, which Google publishes as a discovery document. This guide points an AI agent at those same schemas, so you describe the setting you want in a sentence and get a validated profile, scoped and wired into your repo, as a pull request you review. You do the setup once, per platform you manage. After that the unit of work is intent, not XML.

## Prerequisites

- Fleet, with your configuration in a Git repository. An agent needs that reviewable, schema-legible structure to act against, something a GUI can't give it (see [why AI-powered device management requires GitOps](https://fleetdm.com/articles/why-ai-powered-device-management-requires-gitops)). Start with the [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files), which documents every key you can manage as code. To scaffold a new repo, run `fleetctl new`. To convert an existing Fleet instance, follow [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl).
- An AI coding agent that can run shell commands. Claude Code, Codex, Cursor, and GitHub Copilot all work.
- MDM turned on for each platform you manage. See [Apple](https://fleetdm.com/guides/apple-mdm-setup), [Windows](https://fleetdm.com/guides/windows-mdm-setup), and [Android](https://fleetdm.com/guides/android-mdm-setup) setup.
- [Fleet Premium](https://fleetdm.com/pricing) if you plan to scope profiles to labels.
- For the Apple workflow, a machine running macOS or Linux, since [contour](https://github.com/macadmins/contour) ships builds for those two. There's no Windows build, so author from WSL if that's your desktop, or let CI do the validating. The Windows workflow has no host requirement at all.

## Step 1: Put the schema where the agent can read it

An agent writing XML from memory invents keys. An agent reading a published schema does not. Set up the platforms you manage. The two paths are independent, so skip either one.

### Apple

[Contour](https://github.com/macadmins/contour) is an open-source CLI from the Mac Admins community that generates `.mobileconfig` profiles and DDM JSON declarations from Apple's schema, with the schema embedded in the binary so lookups are offline and free.

1. Download the latest signed `.pkg` from [contour releases](https://github.com/macadmins/contour/releases) and install it.

2. From the root of your repo, create the repo-level config.

```bash
contour init --domain com.acme --name "Acme" --mdm fleet --yes
```

This writes `.contour/config.toml` and a starter `AGENTS.md`. Commit both, so your team and your CI runner generate identical output. Every generated profile gets a `PayloadIdentifier` prefixed with your domain, and UUIDs are deterministic, so regenerating a profile produces byte-identical output instead of a churning diff.

The `--mdm fleet` flag writes Fleet's deploy-time variable catalog into the config as a commented template, covering variables like `FLEET_VAR_HOST_HARDWARE_SERIAL` and `FLEET_VAR_HOST_END_USER_IDP_USERNAME`.

3. Teach the agent to drive it.

```bash
contour setup-agent
```

This installs a Claude Code skill at `.claude/skills/contour/SKILL.md` and appends contour's own usage instructions to both `CLAUDE.md` and `AGENTS.md`. For an agent that reads a different file, tell it to run `contour help-ai` for a command index. You don't need to learn contour's subcommands yourself. The agent discovers them.

> **Note:** contour behavior described in this guide was verified against `0.4.0-beta.4`. Contour is in preview and its flags may change, so re-check the behaviors in step 4 against the version you install.

### Windows

Microsoft publishes the CSP schema as XML, in the DDF files. These are the same source Intune's settings catalog and the Microsoft CSP reference pages are generated from, so they describe every setting those tools expose.

There's nothing to install. Microsoft publishes a DDF page per CSP, so the agent can read the schema for the CSP it needs on demand:

```
https://learn.microsoft.com/en-us/windows/client-management/mdm/<csp>-ddf-file
```

Point the agent at that pattern in `AGENTS.md` or `CLAUDE.md` at the root of your repo:

```markdown
Windows CSP schema comes from Microsoft's per-CSP DDF pages at
https://learn.microsoft.com/en-us/windows/client-management/mdm/<csp>-ddf-file.
Fetch the page for the CSP you're targeting and read the node's `<DFProperties>`
before writing SyncML. Never write a LocURI you haven't confirmed in a DDF.
```

For each node, the DDF declares what a GUI's picker would have enforced for you:

| DDF field | What it constrains |
|---|---|
| `AccessType` | Whether the node accepts `Add` and `Replace`, or is read-only |
| `DFFormat` | The data type, which must match the `<Format>` in your SyncML |
| `MSFT:AllowedValues` | The enum values, numeric range, or regex the node accepts |
| `MSFT:Applicability` | The minimum OS build and the Windows editions the setting applies to |
| `MSFT:DependencyBehavior` | Other nodes that must be set for this one to take effect |
| `MSFT:GpMapping` | The Group Policy name the setting corresponds to |
| `MSFT:AdmxBacked` | The `.admx` file, area path, and policy name backing the node |

Roughly three quarters of CSP nodes declare `AllowedValues`, so most of what you'll set is machine-checkable before it ever reaches a device.

> **Note:** Per-CSP pages are convenient for authoring, but each one is dated independently and some lag the bulk release. If you want a pinned, offline copy you can grep across every CSP at once, and the CI check in step 5, download the [DDF v2 files](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf) into `platforms/windows/schema/` and commit them. The February 2026 drop is 313 files and about 8 MB, covering roughly 5,900 nodes.

### Android

Google publishes the schema as a machine-readable discovery document, so there's nothing to install and nothing to commit. One URL covers every setting:

```
https://androidmanagement.googleapis.com/$discovery/rest?version=v1
```

The `Policy` object in that document is the entire surface: every field an Android configuration profile can set, its type, the shape of every nested object, and for enum fields the exact allowed values with a description of each. Point the agent at it in `AGENTS.md` or `CLAUDE.md`:

```markdown
Android profiles are Android Management API Policy JSON. Read the Policy schema from
https://androidmanagement.googleapis.com/$discovery/rest?version=v1 before writing one.
Copy enum values from the schema exactly, including case.
```

Fleet reserves the Policy fields it manages elsewhere or doesn't support yet, covering software management, kiosk mode, disk encryption, setup experience, and status reporting. It rejects those by name with the reason, so you don't need the list up front. Add a field to your instructions file the first time you see it rejected.

## Step 2: Write down the rules the agent can't infer

Step 1 gave the agent the schema. What the schema can't cover is your environment, and that's the gap you fill.

These go in `AGENTS.md` or `CLAUDE.md` at the root of your repo, the same files step 1 wrote to. Anything true of every profile you'll ever ship belongs here. Anything specific to one profile, like which fleet it targets or which setting you want, belongs in the prompt instead. The test is whether you'd repeat it next time.

An agent can read your repo and infer the directory layout, the naming, and the YAML style. It can't infer the standing decisions behind them:

- **Which delivery method you prefer.** Many Apple settings can ship as either a `.mobileconfig` or a DDM declaration. Say which one you want for those cases and why, because otherwise the agent picks for you, and not always the same way twice.
- **Which Windows CSP you prefer when several would work.** A setting is often reachable through both the Policy CSP and a dedicated CSP, and ADMX-backed nodes are reachable through `ADMXInstall` as well. Pick one and say so, or you'll accumulate three ways of doing the same thing.
- **How a profile gets wired in.** A file committed outside a directory covered by a `paths:` glob does nothing until it has an explicit `path:` entry in the YAML for each fleet that should receive it. Write down which mechanism your repo uses, and the agent will wire new profiles the same way every time. The target fleet itself goes in the prompt, since it changes per profile.
- **The conventions behind your layout.** Which directory each delivery method lives in, how files are named, and which identifier prefix you use. 
- **Anything the agent got wrong once.** Both platforms have a few behaviors where a wrong input still produces a passing result, covered in step 4. Noting them here is the same habit as documenting a gotcha for a new teammate, and it's what keeps the second occurrence from happening.

Write these as sentences with reasoning rather than nested bullets, since a rule with a stated reason is one the agent can apply to a case you didn't anticipate.

## Step 3: Ground the agent in real references

Fleet maintains a [`fleet-gitops` skill](https://github.com/fleetdm/fleet/blob/main/.claude/skills/fleet-gitops/SKILL.md) that points an agent at the reference for each platform:

| What you're building | Reference to validate against |
|---|---|
| First-party Apple payloads (`.mobileconfig`) | [apple/device-management](https://github.com/apple/device-management/tree/release/mdm/profiles) |
| Apple DDM declarations (`.json`) | [apple/device-management declarations](https://github.com/apple/device-management/tree/release/declarative/declarations) |
| Third-party Apple payloads (`.mobileconfig`) | [ProfileManifests](https://github.com/ProfileManifests/ProfileManifests) |
| Windows CSPs (`.xml`) | The DDF files from step 1, plus the [Microsoft CSP reference](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-reference) for prose descriptions |
| Windows ADMX-backed policies | The `.admx` files at `C:\Windows\PolicyDefinitions\` on any Windows host |
| Android profiles (`.json`) | The [Policy discovery document](https://androidmanagement.googleapis.com/$discovery/rest?version=v1) for types and enum values, plus the [Android Management API reference](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies) for prose descriptions |
| osquery tables in reports and policies | [Fleet schema](https://fleetdm.com/tables) |

ProfileManifests is the same community manifest repo that powers ProfileCreator and iMazing Profile Editor, and the DDF files are what Intune's settings catalog is built from. Pointing your agent at both gives you the setting coverage those GUIs have, from the same source they use.

For Claude Code, copy the skill to `.claude/skills/fleet-gitops/SKILL.md` in your repo and invoke it with `/fleet-gitops`. Otherwise, paste the table into your instructions file.

> **Note:** If you're coming from Group Policy, you can search the DDF files by the Group Policy name you already know. Roughly 900 nodes carry a `MSFT:GpMapping` with a `GpEnglishName` attribute, so "find the CSP node for Allow enhanced PINs for startup" is a question the agent can answer against the schema.

## Step 4: Describe what you want

Setup is done. From here the workflow is a sentence.

Open your agent in the repo and state the intent, including the scope:

> Require a 12-character passcode with no simple passcodes on all devices in the workstations fleet. Open a pull request.

The agent finds the setting in the schema, writes the profile into the right directory, validates it, wires it into the workstations fleet, and opens a pull request. You didn't name a payload type, a LocURI, a file path, or a flag. That's the same sentence whether the target is macOS or Windows, and if your workstations fleet has both, the agent produces one profile for each.

The same pattern covers the other delivery methods:

> Ship that as a DDM declaration instead, and tell me which keys don't carry over.

> Block removable storage on Windows workstations. Cite the DDF node and its allowed values in the pull request description.

> Turn on PowerShell script block logging on Windows. It's ADMX-backed, so read the area path out of the DDF and the element values out of the `.admx`.

> Disable the camera on the Android devices in the field-techs fleet, and quote the enum value you used from the Policy schema.

> Normalize every profile under `platforms/macos/configuration-profiles/` to our org identifier and validate them, then summarize what changed.

Ask for the reasoning when the answer matters. "Check whether a DDM equivalent exists for this payload and tell me what it maps to" gets you contour's migration report, including renamed keys, unsupported keys, and any note that Apple stops honoring the legacy payload in a future macOS release.

> **Warning:** Read the agent's command output, not only the file it produced. The most common failure is an agent reporting a successful validation it never ran, or missing that a value was dropped. Step 5 makes that failure impossible to merge.

### Apple traps

Three contour behaviors are worth knowing, because each one turns a wrong input into a passing result:

- **`--set` does not set payload fields.** It substitutes <span v-pre>`{{placeholder}}`</span> variables declared by a recipe. Passing `--set minLength=12` is accepted, silently dropped, and the profile still reports that schema validation passed. Field values belong in a recipe's `[profile.fields]` table. The scaffold lists optional fields as comments directly under `[[profile]]`, and keys left in that position are ignored the same way.
- **Validation without `--strict` accepts unknown keys.** An invented key is written into the profile and validation exits `0`. See step 5.
- **DDM identifiers collide.** Contour derives an `Identifier` from the last component of the declaration type, so `passcode.settings` and `softwareupdate.settings` both produce `com.acme.settings`. Two declarations sharing one is a silent overwrite.

Fleet builds the `com.apple.activation.simple` declaration for each configuration declaration it delivers, so an agent doesn't need to write one even though raw DDM requires it.

### Windows traps

The same class of problem, and the DDF is where you catch each one. A GUI hid these by construction, by greying out a dependent setting or filtering by edition, so they're the ones to write into `AGENTS.md` first:

- **Values are not always the ones you'd guess.** `DeviceLock/DevicePasswordEnabled` takes `0` for enabled and `1` for disabled. The DDF spells that out in a `ValueDescription` per enum value, so an agent that reads it gets this right while an agent reasoning from the node name gets it backwards.
- **Some nodes are read-only.** About 500 leaf nodes accept only `Get`. A `Replace` against one is accepted by Fleet, deploys, and fails on the device. `AccessType` tells you before you ship it.
- **`<Format>` must match `DFFormat`.** The DDF declares `int`, `chr`, or `bool` per node. A mismatch is a device-side failure that nothing upstream catches.
- **Settings have dependencies.** `DeviceLock/MinDevicePasswordLength` has no effect unless `DeviceLock/DevicePasswordEnabled` is also set. Around 130 nodes declare a `DependencyBehavior` group naming exactly what they need.
- **Applicability is per build and per edition.** `MSFT:Applicability` carries a minimum OS build and an allowed edition list. A valid setting aimed at the wrong SKU is a silent no-op, not an error.

For ADMX-backed nodes, the DDF gives you the `.admx` filename, the area path, and the policy name, which is most of what you need. You still read the element IDs and their accepted values out of the `.admx` file itself. [Creating Windows configuration profiles (CSPs)](https://fleetdm.com/guides/creating-windows-csps) walks through that assembly by hand, and it's worth reading once so you can tell when the agent has done it wrong.

Fleet validates what it can on upload. It checks XML well-formedness, requires a supported top-level element, rejects `LocURI` formats that real Windows hosts refuse, and blocks LocURIs that conflict with settings Fleet already manages, such as disk encryption.

> **Note:** Migrating an existing Intune baseline rather than writing new profiles? [Migrating Intune policies to Fleet with the CSP converter](https://fleetdm.com/guides/migrating-intune-policies-to-fleet-csp-converter) covers the bulk path, and the validation in step 5 applies to its output too.

### Android traps

Fleet catches more here than on the other two platforms, because it validates against Google's own `Policy` type rather than a format spec. An unknown top-level key, a value of the wrong type, a field Fleet reserves, and a Premium-only field all fail with a message naming the field. What gets through:

- **Enum values aren't checked locally.** `cameraAccess` is typed as a string, so `"camera_access_disabled"` in the wrong case and an entirely invented value both pass and then fail when Google receives the policy. This is why the instructions in step 1 tell the agent to copy enum values out of the schema rather than infer them.
- **Unknown keys nested inside an object are dropped, not rejected.** A typo at the top level fails by name. The same typo one level down, inside something like `advancedSecurityOverrides`, is silently discarded, so the profile deploys and the setting you wanted just isn't there.

Both are cases where the profile looks right and does nothing, so ask the agent to quote the schema for each field it set. That quote is what you check.

## Step 5: Enforce validation in CI

Instructions make the agent likely to validate. CI makes it impossible to skip. These go in the GitHub Actions workflow that already runs `fleetctl gitops` for your repo, usually `.github/workflows/gitops.yml`, as steps that run on pull requests. Add one per delivery method you use.

### Apple

Contour publishes a Linux build, so this runs on a standard `ubuntu-latest` runner. The schema is embedded in the binary, so validation needs no network access and no Apple credentials:

```yaml
- name: Validate Apple profiles
  run: |
    curl -fsSL -o contour.tar.gz \
      https://github.com/macadmins/contour/releases/download/v0.4.0-beta.4/contour-0.4.0-beta.4-x86_64-unknown-linux-gnu.tar.gz
    tar -xzf contour.tar.gz
    ./contour profile validate ./platforms/macos/configuration-profiles -r --strict
```

This fails on an invented payload key with exit code `1`. Pin the version in the URL rather than tracking the latest release, since contour is in preview.

> **Warning:** The `--strict` flag is not the default and it is the one that matters. A typo'd key is the most likely failure mode for an agent-authored profile, it's the failure the GUI made impossible, and without `--strict` you get no signal at all.

For DDM declarations, run both the schema check and the cross-reference check in the same step, so they reuse the binary you just extracted:

```bash
./contour profile ddm validate ./platforms/macos/declaration-profiles -r
./contour profile ddm verify ./platforms/macos/declaration-profiles -r
```

`ddm validate` catches the same class of problem as `profile validate` does for `.mobileconfig`: missing required keys, wrong types, and unknown fields, checked against the DDM JSON schemas instead of payload schemas.

`ddm verify` checks what schema validation alone can't. It builds a reference graph across the directory and confirms that every asset and activation a declaration points at actually resolves, exiting non-zero on a broken reference.

> **Warning:** Don't add `--strict` to `ddm verify` in a Fleet repo. It promotes an orphaned configuration, meaning a declaration no activation references, into an error. Because Fleet supplies the activations, a Fleet repo has none, so every declaration is orphaned by that definition and the check fails permanently. Contour labels the warning itself as valid on Apple's side.

Add `--json` to any of these commands for machine-readable output.

Neither check catches the identifier collision from step 4. Two declarations sharing `com.acme.settings` verify clean and exit `0`, because a reference graph can't see that one identifier was supposed to be two. Add it as its own step, since a duplicate is just a duplicate:

```bash
dupes=$(jq -r '.Identifier' platforms/macos/declaration-profiles/*.json | sort | uniq -d)
[ -z "$dupes" ] || { printf 'duplicate declaration identifiers:\n%s\n' "$dupes"; exit 1; }
```

A recipe has no key for setting the identifier directly, so the fix is to change the `Identifier` in the generated declaration. Note that in `AGENTS.md` too, since regenerating the file would otherwise undo it.

### Windows

Fleet validates Windows profiles server-side, and `--dry-run` runs that validation without applying anything:

```bash
fleetctl gitops --dry-run -f ./path/to/fleet.yml
```

That covers XML well-formedness, the presence of a supported top-level element, the `LocURI` format rules real Windows hosts enforce, collisions with settings Fleet manages itself such as disk encryption, and profile name conflicts across platforms. Use it as the gate on every pull request.

> **Warning:** Dry run skips any profile that references a `$FLEET_SECRET_` variable, because the secret may not resolve in CI. Those profiles are validated when you apply them for real, so a clean dry run doesn't mean every profile was checked.

What a dry run can't do is tell you whether a `LocURI` names a CSP node that exists, and that's what the DDF files are for. If you committed the corpus in step 1, a name lookup against it catches the common case:

```bash
missing=$(find platforms/windows/configuration-profiles -name '*.xml' \
  -exec grep -oh '<LocURI>[^<]*</LocURI>' {} + | sed 's|</*LocURI>||g' | sort -u |
  while read -r uri; do
    grep -rqs "<NodeName>${uri##*/}</NodeName>" platforms/windows/schema/ || echo "$uri"
  done)
[ -z "$missing" ] || { printf 'unknown CSP nodes:\n%s\n' "$missing"; exit 1; }
```

This matches on node name rather than full path, so it catches an invented or misspelled node but not a real node addressed under the wrong parent.

Everything else the DDF declares, the allowed values, the format, the dependencies, and the applicability, stays a review-time check. Have the agent cite the DDF node it used in the pull request description, then read the citation against the schema. Checking one citation is a few seconds of review, and it's what catches the value that was in range but wrong.

### Android

There's nothing to add. Fleet validates Android profiles against Google's `Policy` type, so the same dry run is the whole gate:

```bash
fleetctl gitops --dry-run -f ./path/to/fleet.yml
```

An unknown top-level key fails and names the key, a value of the wrong type fails and names the field, and a field Fleet reserves or gates behind Premium fails with the reason. The two gaps from step 4, enum values and keys nested inside an object, are the review-time part.

## Step 6: Review, then deploy

The agent proposes and a human merges. The agent can't merge, can't deploy, and can't touch a device, and that boundary is the whole safety model. Keep it even when the change is small and the agent is right.

Read the diff for the things validation can't check:

- Does this profile address the risk you care about, or did the agent optimize for the wording of your prompt?
- Is the scope right? You asked for one fleet, but does another share the infrastructure this should cover?
- Does this overlap an existing profile, and are you about to ship a conflicting setting?
- Are the identifiers distinct from everything already in the repo?
- On Windows, do the target hosts meet the build and edition applicability for every node you set?
- Is the pull request description honest about what the change does?

Merge, and CI runs `fleetctl gitops` to apply the change. Scope with fleets and labels the way you would for any other profile. See [Custom OS settings](https://fleetdm.com/guides/custom-os-settings) for the full delivery behavior.

> **Note:** Turn on [GitOps mode](https://fleetdm.com/learn-more-about/ui-gitops-mode) so the repo stays the only way in. It makes the matching UI controls read-only, which means nobody can upload a profile under **Controls > OS settings** that the next `fleetctl gitops` run would overwrite, and the review step can't be bypassed by accident.

## Verify

1. Go to **Hosts** and select a host in the target fleet.
2. Open the **OS settings** tab.
3. Confirm the profile or declaration shows as **Verified**.

On Windows, confirm the setting itself rather than only its delivery, since a profile can deliver successfully and still no-op on a host that falls outside the node's applicability. Run a live report against the [`registry`](https://fleetdm.com/tables/registry) table, matching on `path` and reading the value out of `data`:

```sql
SELECT path, data FROM registry
WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\PolicyManager\current\device\DeviceLock\%';
```

> **Note:** `PolicyManager\current\device\<Area>` is where Policy CSP settings land, but the location varies by CSP, and some settings write under `PolicyManager\Providers` instead. Confirm the path for the specific setting you shipped on one pilot host before you build a report or policy around it.

## Related resources

- [Why AI-powered device management requires GitOps](https://fleetdm.com/articles/why-ai-powered-device-management-requires-gitops): the case for the prerequisite above.
- [Custom OS settings](https://fleetdm.com/guides/custom-os-settings): how Fleet delivers and verifies profiles, declarations, and CSPs.
- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files): every key you can manage as code.
- [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl): convert an existing Fleet instance to a repo.
- [Fleet's `fleet-gitops` skill](https://github.com/fleetdm/fleet/blob/main/.claude/skills/fleet-gitops/SKILL.md): the reference-grounding skill from step 3.
- [Creating Windows configuration profiles (CSPs)](https://fleetdm.com/guides/creating-windows-csps): ADMX-backed policies and SyncML structure by hand.
- [Migrating Intune policies to Fleet with the CSP converter](https://fleetdm.com/guides/migrating-intune-policies-to-fleet-csp-converter): bulk migration of an existing Intune baseline.
- [Configuration service provider DDF files](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf): the Windows schema, including the DDF v2 field definitions.
- [contour recipes reference](https://github.com/macadmins/contour/blob/main/docs/contour-recipes.md): the recipe and preset TOML surface, if you want to read one the agent wrote.

<meta name="articleTitle" value="Build and validate configuration profiles with AI instead of a GUI">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-10">
<meta name="description" value="Describe a profile in a sentence and let an AI agent generate, validate, and ship it through Fleet GitOps on Apple, Windows, and Android.">
