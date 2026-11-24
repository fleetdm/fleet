# Build and validate configuration profiles with AI instead of a GUI

Most admins build configuration profiles in a GUI. In Intune you pick a setting out of the settings catalog. In Workspace ONE you pick a profile payload. On Apple you open iMazing Profile Editor or ProfileCreator and pick from a payload list. In every case the picker constrains you to settings that exist, with values the platform accepts, and that constraint is the real value of the tool. Everything else about it, the clicking, the exporting, the copying into a repo, is transcription.

Every one of those pickers is a GUI over a published schema. Apple's payload keys come from apple/device-management and ProfileManifests. Intune's settings catalog is generated from Microsoft's CSP Device Description Framework (DDF) files. Android's settings are the fields of the Android Management API `Policy` object, which Google publishes as a discovery document. Fleet's `fleet-gitops` skill points an AI agent at those same schemas, so you describe the setting you want in a sentence and get a validated profile, scoped and wired into your repo, as a pull request you review. You add the skill once. After that the unit of work is intent, not XML.

## Prerequisites

- Fleet, with your configuration in a Git repository. An agent needs that reviewable, schema-legible structure to act against, something a GUI can't give it (see [why AI-powered device management requires GitOps](https://fleetdm.com/articles/why-ai-powered-device-management-requires-gitops)). Start with the [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files), which documents every key you can manage as code. To scaffold a new repo, run `fleetctl new`. To convert an existing Fleet instance, follow [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl).
- An AI coding agent that can run shell commands. Claude Code, Codex, Cursor, and GitHub Copilot all work.
- MDM turned on for each platform you manage. See [Apple](https://fleetdm.com/guides/apple-mdm-setup), [Windows](https://fleetdm.com/guides/windows-mdm-setup), and [Android](https://fleetdm.com/guides/android-mdm-setup) setup.
- [Fleet Premium](https://fleetdm.com/pricing) if you plan to scope profiles to labels.

## Step 1: Add the fleet-gitops skill

An agent writing XML from memory invents keys. An agent reading a published schema does not. Fleet's [`fleet-gitops` skill](https://github.com/fleetdm/fleet/blob/main/cmd/fleetctl/fleetctl/templates/new/.agents/skills/fleet-gitops/SKILL.md) tells the agent which schema to read for each platform and how to validate what it writes. There's nothing else to install.

Run `fleetctl new` (fleetctl 4.95.0 or later) to create a GitOps repo that includes the skill:

```bash
fleetctl new
```

Already have a repo? Copy the [`fleet-gitops` folder](https://github.com/fleetdm/fleet/tree/main/cmd/fleetctl/fleetctl/templates/new/.agents/skills/fleet-gitops) into it as `.agents/skills/fleet-gitops/`, and for Claude Code, also as `.claude/skills/fleet-gitops/`.

The skill points the agent at these references:

| What you're building | Reference to validate against |
|---|---|
| First-party Apple payloads (`.mobileconfig`) | [apple/device-management](https://github.com/apple/device-management/tree/release/mdm/profiles) |
| Apple DDM declarations (`.json`) | [apple/device-management declarations](https://github.com/apple/device-management/tree/release/declarative/declarations) |
| Third-party Apple payloads (`.mobileconfig`) | [ProfileManifests](https://github.com/ProfileManifests/ProfileManifests) |
| Windows CSPs (`.xml`) | Microsoft's [DDF files](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf), plus the [Microsoft CSP reference](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-reference) for prose descriptions |
| Windows ADMX-backed policies | The `.admx` files at `C:\Windows\PolicyDefinitions\` on any Windows host |
| Android profiles (`.json`) | The [Policy discovery document](https://androidmanagement.googleapis.com/$discovery/rest?version=v1) for types and enum values, plus the [Android Management API reference](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies) for prose descriptions |
| osquery tables in reports and policies | [Fleet schema](https://fleetdm.com/tables) |

ProfileManifests is the same community manifest repo that powers ProfileCreator and iMazing Profile Editor, and the DDF files are what Intune's settings catalog is built from. Pointing your agent at both gives you the setting coverage those GUIs have, from the same source they use.

> **Note:** If you're coming from Group Policy, the agent can search the DDF files by the Group Policy name you already know. Roughly 900 nodes carry a `MSFT:GpMapping` with a `GpEnglishName` attribute, so "find the CSP node for Allow enhanced PINs for startup" is a question the agent can answer against the schema.

## Step 2: Write down the rules the agent can't infer

The skill gave the agent the schema. What the schema can't cover is your environment, and that's the gap you fill.

These go in `AGENTS.md` or `CLAUDE.md` at the root of your repo. Anything true of every profile you'll ever ship belongs here. Anything specific to one profile, like which fleet it targets or which setting you want, belongs in the prompt instead. The test is whether you'd repeat it next time.

An agent can read your repo and infer the directory layout, the naming, and the YAML style. It can't infer the standing decisions behind them:

- **Which delivery method you prefer.** Many Apple settings can ship as either a `.mobileconfig` or a DDM declaration. Say which one you want for those cases and why, because otherwise the agent picks for you, and not always the same way twice.
- **Which Windows CSP you prefer when several would work.** A setting is often reachable through both the Policy CSP and a dedicated CSP, and ADMX-backed nodes are reachable through `ADMXInstall` as well. Pick one and say so, or you'll accumulate three ways of doing the same thing.
- **How a profile gets wired in.** A file committed outside a directory covered by a `paths:` glob does nothing until it has an explicit `path:` entry in the YAML for each fleet that should receive it. Write down which mechanism your repo uses, and the agent will wire new profiles the same way every time. The target fleet itself goes in the prompt, since it changes per profile.
- **The conventions behind your layout.** Which directory each delivery method lives in, how files are named, and which identifier prefix you use. 
- **Anything the agent got wrong once.** Both platforms have a few behaviors where a wrong input still produces a passing result, covered in step 3. Noting them here is the same habit as documenting a gotcha for a new teammate, and it's what keeps the second occurrence from happening.

Write these as sentences with reasoning rather than nested bullets, since a rule with a stated reason is one the agent can apply to a case you didn't anticipate.

## Step 3: Describe what you want

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

> **Warning:** Read the agent's command output, not only the file it produced. The most common failure is an agent reporting a successful validation it never ran, or missing that a value was dropped. Step 4 makes that failure impossible to merge.

### Apple traps

Fleet checks that every setting it recognizes has a value of the expected type, and it allows keys it doesn't recognize. That means a misspelled or invented payload key passes validation and does nothing on the device. Ask the agent to cite the schema file for each key it set, and check the citation in review.

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

> **Note:** Migrating an existing Intune baseline rather than writing new profiles? [Migrating Intune policies to Fleet with the CSP converter](https://fleetdm.com/guides/migrating-intune-policies-to-fleet-csp-converter) covers the bulk path, and the validation in step 4 applies to its output too.

### Android traps

Fleet catches more here than on the other two platforms, because it validates against Google's own `Policy` type rather than a format spec. An unknown top-level key, a value of the wrong type, a field Fleet reserves, and a Premium-only field all fail with a message naming the field. What gets through:

- **Enum values aren't checked locally.** `cameraAccess` is typed as a string, so `"camera_access_disabled"` in the wrong case and an entirely invented value both pass and then fail when Google receives the policy. This is why the skill tells the agent to copy enum values out of the schema rather than infer them.
- **Unknown keys nested inside an object are dropped, not rejected.** A typo at the top level fails by name. The same typo one level down, inside something like `advancedSecurityOverrides`, is silently discarded, so the profile deploys and the setting you wanted just isn't there.

Both are cases where the profile looks right and does nothing, so ask the agent to quote the schema for each field it set. That quote is what you check.

## Step 4: Enforce validation in CI

Instructions make the agent likely to validate. CI makes it impossible to skip. The GitHub Actions workflow that `fleetctl new` scaffolds already runs a dry run on every pull request:

```bash
fleetctl gitops --dry-run -f default.yml -f fleets/workstations.yml
```

A dry run validates every profile without applying anything. For Apple and Windows profiles, Fleet 4.95 and later checks that each setting it recognizes has a value of the expected type (turn this off with [`enable_profile_validation`](https://fleetdm.com/docs/configuration/yaml-files#controls)). It also checks XML and plist well-formedness, reserved identifiers, Windows `LocURI` formats that real hosts refuse, collisions with settings Fleet manages itself such as disk encryption, and name conflicts. For Android, Fleet validates against Google's `Policy` type, so an unknown top-level key, a value of the wrong type, and a field Fleet reserves or gates behind Premium all fail with a message naming the field.

> **Warning:** Dry run skips any profile that references a `$FLEET_SECRET_` variable, because the secret may not resolve in CI. Those profiles are validated when you apply them for real, so a clean dry run doesn't mean every profile was checked.

What a dry run can't catch are the traps in step 3: unrecognized Apple keys, Windows nodes that don't exist or values that are in range but wrong, and Android enum values and nested keys. Have the agent cite the schema for each setting in the pull request description, then read the citation against the schema. Checking one citation is a few seconds of review.

## Step 5: Review, then deploy

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
- [Fleet's `fleet-gitops` skill](https://github.com/fleetdm/fleet/blob/main/cmd/fleetctl/fleetctl/templates/new/.agents/skills/fleet-gitops/SKILL.md): the skill from step 1.
- [Creating Windows configuration profiles (CSPs)](https://fleetdm.com/guides/creating-windows-csps): ADMX-backed policies and SyncML structure by hand.
- [Migrating Intune policies to Fleet with the CSP converter](https://fleetdm.com/guides/migrating-intune-policies-to-fleet-csp-converter): bulk migration of an existing Intune baseline.
- [Configuration service provider DDF files](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf): the Windows schema, including the DDF v2 field definitions.

<meta name="articleTitle" value="Build and validate configuration profiles with AI instead of a GUI">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-10">
<meta name="description" value="Describe a profile in a sentence and let an AI agent generate, validate, and ship it through Fleet GitOps on Apple, Windows, and Android.">
