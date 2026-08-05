# Build and validate configuration profiles with AI instead of a GUI

Most admins build configuration profiles in a GUI. In Intune you pick a setting out of the settings catalog. In Workspace ONE you pick a profile payload. On Apple you open iMazing Profile Editor or ProfileCreator and pick from a payload list. In every case the picker constrains you to settings that exist, with values the platform accepts, and that constraint is the real value of the tool. Everything else about it, the clicking, the exporting, the copying into a repo, is transcription.

Every one of those pickers is a GUI over a published schema. Apple's payload keys come from apple/device-management and ProfileManifests. Intune's settings catalog is generated from Microsoft's CSP Device Description Framework (DDF) files. This guide points an AI agent at those same schemas, so you describe the setting you want in a sentence and get a validated profile, scoped and wired into your repo, as a pull request you review. You do the setup once, per platform you manage. After that the unit of work is intent, not XML.

## Prerequisites

- Fleet, with your configuration in a Git repository. Start with the [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files), which documents every key you can manage as code. To scaffold a new repo, run `fleetctl new`. To convert an existing Fleet instance, follow [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl).
- An AI coding agent that can run shell commands. Claude Code, Codex, Cursor, and GitHub Copilot all work.
- [Fleet Premium](https://fleetdm.com/pricing) if you plan to scope profiles to labels.
- For the Apple workflow, a macOS host, since [contour](https://github.com/macadmins/contour) ships as a signed macOS package. The Windows workflow has no host requirement and runs anywhere, including in CI.

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

Windows has no CLI wrapper like contour, and it doesn't need one, because Microsoft publishes the schema directly as a set of XML files you can put in your repo. These are the DDF files, and they're the same source Intune's settings catalog and the Microsoft CSP reference pages are generated from.

1. Download the current [DDF v2 files](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf) and unpack them into your repo.

```bash
curl -L -o ddf.zip "https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip"
unzip -q ddf.zip -d platforms/windows/schema/
```

The February 2026 drop is 313 XML files and about 8 MB, covering roughly 5,900 CSP nodes. Commit it. Lookups are then offline, free, and pinned to a version, which is the same property that makes contour's embedded schema reliable.

2. Tell the agent the directory exists and what's in it. Add this to `AGENTS.md` or `CLAUDE.md`:

```markdown
Windows CSP schema lives in `platforms/windows/schema/`. Before writing any SyncML,
grep it for the node you intend to set and read its `<DFProperties>`. Never write a
LocURI that you have not found in these files.
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

About 4,500 of those nodes declare `AllowedValues`, so most of what you'll set is machine-checkable before it ever reaches a device.

## Step 2: Write down the rules the agent can't infer

Step 1 gave the agent the schema. What the schema can't cover is your environment, and that's the gap you fill.

An agent can read your repo and infer the directory layout, the naming, and the YAML style. It can't infer the decisions behind them. The categories worth writing down:

- **Which delivery method you want.** Many Apple settings can ship as either a `.mobileconfig` or a DDM declaration. Say which one you want for those cases and why, because otherwise the agent picks for you, and not always the same way twice. Asking it to run `contour profile ddm map <payload_type>` first tells you whether a declarative equivalent exists at all.
- **Which Windows CSP you prefer when several would work.** A setting is often reachable through both the Policy CSP and a dedicated CSP, and ADMX-backed nodes are reachable through `ADMXInstall` as well. Pick one and say so, or you'll accumulate three ways of doing the same thing.
- **How a profile reaches a host.** A file committed outside a directory covered by a `paths:` glob sits in the repo until it has an explicit `path:` entry in each fleet's YAML that should receive it. State which fleets a new profile is for.
- **How you confirm a setting applied.** On macOS, the `managed_policies` osquery table reflects only legacy `.mobileconfig` profiles, so checking a DDM-delivered setting against it returns misleading results. On Windows, the registry value behind a CSP is the ground truth, and the [Fleet schema](https://fleetdm.com/tables) documents the `registry` table you'd query for it.
- **The traps you've already hit.** Both platforms have defaults that turn a wrong input into a passing result, covered in step 4. Once you've been bitten, write the rule down so you don't get bitten twice.

Write these as sentences with reasoning rather than nested bullets, since a rule with a stated reason is one the agent can apply to a case you didn't anticipate. Add a line every time you catch the agent getting something wrong, and the file pays for itself the next time.

## Step 3: Ground the agent in real references

Fleet maintains a [`fleet-gitops` skill](https://github.com/fleetdm/fleet/blob/main/.claude/skills/fleet-gitops/SKILL.md) that points an agent at the reference for each platform:

| What you're building | Reference to validate against |
|---|---|
| First-party Apple payloads (`.mobileconfig`) | [apple/device-management](https://github.com/apple/device-management/tree/release/mdm/profiles) |
| Apple DDM declarations (`.json`) | [apple/device-management declarations](https://github.com/apple/device-management/tree/release/declarative/declarations) |
| Third-party Apple payloads (`.mobileconfig`) | [ProfileManifests](https://github.com/ProfileManifests/ProfileManifests) |
| Windows CSPs (`.xml`) | The DDF files from step 1, plus the [Microsoft CSP reference](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-reference) for prose descriptions |
| Windows ADMX-backed policies | The `.admx` files at `C:\Windows\PolicyDefinitions\` on any Windows host |
| Android profiles (`.json`) | [Android Management API policies](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies) |
| osquery tables in reports and policies | [Fleet schema](https://fleetdm.com/tables) |

ProfileManifests is the same community manifest repo that powers ProfileCreator and iMazing Profile Editor, and the DDF files are what Intune's settings catalog is built from. Pointing your agent at both gives you the setting coverage those GUIs have, from the same source they use.

For Claude Code, copy the skill to `.claude/skills/fleet-gitops/SKILL.md` in your repo and invoke it with `/fleet-gitops`. Otherwise, paste the table into your instructions file.

> **Note:** If you're coming from Group Policy, you can search the DDF files by the Group Policy name you already know. Roughly 900 nodes carry a `MSFT:GpMapping` with a `GpEnglishName` attribute, so "find the CSP node for Allow enhanced PINs for startup" is a question the agent can answer against the schema.

## Step 4: Describe what you want

Setup is done. From here the workflow is a sentence.

Open your agent in the repo and state the intent, including the scope:

> Require a 12-character passcode with no simple passcodes on all workstations. Open a pull request.

The agent finds the setting in the schema, writes the profile into the right directory, validates it, wires it into the workstations fleet, and opens a pull request. You didn't name a payload type, a LocURI, a file path, or a flag. That's the same sentence whether the target is macOS or Windows, and if your workstations fleet has both, the agent produces one profile for each.

The same pattern covers the other delivery methods:

> Ship that as a DDM declaration instead, and tell me which keys don't carry over.

> Block removable storage on Windows workstations. Cite the DDF node and its allowed values in the pull request description.

> Turn on PowerShell script block logging on Windows. It's ADMX-backed, so read the area path out of the DDF and the element values out of the `.admx`.

> Normalize every profile under `platforms/macos/configuration-profiles/` to our org identifier and validate them, then summarize what changed.

Ask for the reasoning when the answer matters. "Check whether a DDM equivalent exists for this payload and tell me what it maps to" gets you contour's migration report, including renamed keys, unsupported keys, and any note that Apple stops honoring the legacy payload in a future macOS release.

> **Warning:** Read the agent's command output, not only the file it produced. The most common failure is an agent reporting a successful validation it never ran, or missing that a value was dropped. Step 5 makes that failure impossible to merge.

### Apple traps

Three contour behaviors are worth knowing, because each one turns a wrong input into a passing result:

- **`--set` does not set payload fields.** It substitutes `{{placeholder}}` variables declared by a recipe. Passing `--set minLength=12` is accepted, silently dropped, and the profile still reports that schema validation passed. Field values belong in a recipe's `[profile.fields]` table. The scaffold lists optional fields as comments directly under `[[profile]]`, and keys left in that position are ignored the same way.
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

## Step 5: Enforce validation in CI

Instructions make the agent likely to validate. CI makes it impossible to skip. Add checks to your pull request workflow, one per delivery method.

### Apple

For `.mobileconfig` profiles:

```bash
contour profile validate ./platforms/macos/configuration-profiles -r --strict
```

This fails on an invented payload key with exit code `1`.

> **Warning:** The `--strict` flag is not the default and it is the one that matters. A typo'd key is the most likely failure mode for an agent-authored profile, it's the failure the GUI made impossible, and without `--strict` you get no signal at all.

For DDM declarations, run both the schema check and the cross-reference check:

```bash
contour profile ddm validate ./platforms/macos/declaration-profiles -r
contour profile ddm verify ./platforms/macos/declaration-profiles -r
```

`ddm validate` catches the same class of problem as `profile validate` does for `.mobileconfig`: missing required keys, wrong types, and unknown fields, checked against the DDM JSON schemas instead of payload schemas.

`ddm verify` checks what schema validation alone can't. It builds a reference graph across the directory and confirms that every asset and activation a declaration points at actually resolves, exiting non-zero on a broken reference.

> **Warning:** Don't add `--strict` to `ddm verify` in a Fleet repo. It promotes an orphaned configuration, meaning a declaration no activation references, into an error. Because Fleet supplies the activations, a Fleet repo has none, so every declaration is orphaned by that definition and the check fails permanently. Contour labels the warning itself as valid on Apple's side.

Add `--json` to any of these commands for machine-readable output.

Neither check catches the identifier collision from step 4. Two declarations sharing `com.acme.settings` verify clean and exit `0`, because a reference graph can't see that one identifier was supposed to be two. That one stays a review-time check.

### Windows

Fleet validates Windows profiles server-side, and `--dry-run` runs that validation without applying anything:

```bash
fleetctl gitops --dry-run -f ./path/to/fleet.yml
```

That covers XML well-formedness, the presence of a supported top-level element, the `LocURI` format rules real Windows hosts enforce, collisions with settings Fleet manages itself such as disk encryption, and profile name conflicts across platforms. Use it as the gate on every pull request.

> **Warning:** Dry run skips any profile that references a `$FLEET_SECRET_` variable, because the secret may not resolve in CI. Those profiles are validated when you apply them for real, so a clean dry run doesn't mean every profile was checked.

What a dry run can't do is tell you whether a `LocURI` names a CSP node that exists, and that's what the DDF files are for. A name lookup against the schema you committed in step 1 catches the common case:

```bash
missing=$(find platforms/windows/configuration-profiles -name '*.xml' \
  -exec grep -oh '<LocURI>[^<]*</LocURI>' {} + | sed 's|</*LocURI>||g' | sort -u |
  while read -r uri; do
    grep -rqs "<NodeName>${uri##*/}</NodeName>" platforms/windows/schema/ || echo "$uri"
  done)
[ -z "$missing" ] || { printf 'unknown CSP nodes:\n%s\n' "$missing"; exit 1; }
```

This matches on node name rather than full path, so it catches an invented or misspelled node but not a real node addressed under the wrong parent.

Everything else the DDF declares, the allowed values, the format, the dependencies, and the applicability, stays a review-time check. Have the agent cite the DDF node it used in the pull request description, then read the citation against the schema in the repo. That citation is doing the same work `--strict` does on the Apple side, with you as the gate instead of an exit code.

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

> **Note:** You can also upload a generated file under **Controls > OS settings > Custom settings** in the Fleet UI. You lose the review step, which is the part doing the most work here.

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
<meta name="publishedOn" value="2026-08-03">
<meta name="description" value="Describe a configuration profile in a sentence and let an AI agent generate, validate, and ship it through Fleet GitOps, on Apple and Windows alike.">
