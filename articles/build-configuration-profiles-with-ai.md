# Build and validate configuration profiles with AI instead of a GUI

Most admins build configuration profiles in a GUI. You open iMazing Profile Editor or ProfileCreator, pick a payload from a list, fill in the fields, and export a `.mobileconfig`. The picker constrains you to keys that exist, with types and values Apple accepts, and that constraint is the real value of a profile editor. Everything else about it, the clicking, the exporting, the copying into a repo, is transcription.

This guide sets up a workflow where you describe the setting you want in a sentence and an AI agent produces a validated profile, scoped and wired into your repo, as a pull request you review. You do the setup once. After that, the unit of work is intent, not XML. It covers Apple platforms in depth and Windows CSPs at the end, where the tooling is thinner.

## Prerequisites

- Fleet, with your configuration in a Git repository. Start with the [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files), which documents every key you can manage as code. To scaffold a new repo, run `fleetctl new`. To convert an existing Fleet instance, follow [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl).
- An AI coding agent that can run shell commands. Claude Code, Codex, Cursor, and GitHub Copilot all work.
- A macOS host, since [contour](https://github.com/macadmins/contour) ships as a signed macOS package.
- [Fleet Premium](https://fleetdm.com/pricing) if you plan to scope profiles to labels.

> **Note:** The behavior described below was verified against contour `0.4.0-beta.4`. Contour is in preview and its flags may change, so re-check the behaviors described in step 4 against the version you install.

## Step 1: Give the agent a schema-bound tool

An agent writing XML from memory invents keys. An agent calling a tool that reads Apple's published schema does not. [Contour](https://github.com/macadmins/contour) is an open-source CLI from the Mac Admins community that generates `.mobileconfig` profiles and DDM JSON declarations from that schema, with the schema embedded in the binary so lookups are offline and free.

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

## Step 2: Write down the rules the agent can't infer

Step 1 left contour's own reference material in `AGENTS.md` and `CLAUDE.md`. That covers how to drive the tool. What it can't cover is your environment, and that's the gap you fill.

An agent can read your repo and infer the directory layout, the naming, and the YAML style. It can't infer the decisions behind them. The categories worth writing down:

- **Which delivery method you want.** Many Apple settings can ship as either a `.mobileconfig` or a DDM declaration. Say which one you want for those cases and why, because otherwise the agent picks for you, and not always the same way twice. Asking it to run `contour profile ddm map <payload_type>` first tells you whether a declarative equivalent exists at all.
- **How a profile reaches a host.** A file committed outside a directory covered by a `paths:` glob sits in the repo until it has an explicit `path:` entry in each fleet's YAML that should receive it. State which fleets a new profile is for.
- **How you confirm a setting applied.** On macOS, the `managed_policies` osquery table reflects only legacy `.mobileconfig` profiles, so checking a DDM-delivered setting against it returns misleading results.
- **The traps you've already hit.** Contour has a few defaults that produce a passing result from a wrong input, covered in step 4. Once you've been bitten, write the rule down so you don't get bitten twice.

Write these as sentences with reasoning rather than nested bullets, since a rule with a stated reason is one the agent can apply to a case you didn't anticipate. Add a line every time you catch the agent getting something wrong, and the file pays for itself the next time.

## Step 3: Ground the agent in real references

Fleet maintains a [`fleet-gitops` skill](https://github.com/fleetdm/fleet/blob/main/.claude/skills/fleet-gitops/SKILL.md) that points an agent at the reference for each platform:

| What you're building | Reference to validate against |
|---|---|
| First-party Apple payloads (`.mobileconfig`) | [apple/device-management](https://github.com/apple/device-management/tree/release/mdm/profiles) |
| Apple DDM declarations (`.json`) | [apple/device-management declarations](https://github.com/apple/device-management/tree/release/declarative/declarations) |
| Third-party Apple payloads (`.mobileconfig`) | [ProfileManifests](https://github.com/ProfileManifests/ProfileManifests) |
| Windows CSPs (`.xml`) | [Microsoft CSP reference](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-reference) |
| Android profiles (`.json`) | [Android Management API policies](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies) |
| osquery tables in reports and policies | [Fleet schema](https://fleetdm.com/tables) |

ProfileManifests is the same community manifest repo that powers ProfileCreator and iMazing Profile Editor, so pointing your agent at it gives you the key coverage those GUIs have, from the same source.

For Claude Code, copy the skill to `.claude/skills/fleet-gitops/SKILL.md` in your repo and invoke it with `/fleet-gitops`. Otherwise, paste the table into your instructions file.

## Step 4: Describe what you want

Setup is done. From here the workflow is a sentence.

Open your agent in the repo and state the intent, including the scope:

> Require a 12-character passcode with no simple passcodes on all workstations. Open a pull request.

The agent searches Apple's schema for the payload type, writes a recipe capturing those values, renders the profile into the right directory, validates it strictly, wires it into the workstations fleet, and opens a pull request. You didn't name a payload type, a file path, or a flag.

The same pattern covers the other platforms and delivery methods:

> Ship that as a DDM declaration instead, and tell me which keys don't carry over.

> Block removable storage on Windows workstations. Cite the CSP reference page you used in the pull request description.

> Normalize every profile under `platforms/macos/configuration-profiles/` to our org identifier and validate them, then summarize what changed.

Ask for the reasoning when the answer matters. "Check whether a DDM equivalent exists for this payload and tell me what it maps to" gets you contour's migration report, including renamed keys, unsupported keys, and any note that Apple stops honoring the legacy payload in a future macOS release.

> **Warning:** Read the agent's command output, not only the file it produced. The most common failure is an agent reporting a successful validation it never ran, or missing that a value was dropped. Step 5 makes that failure impossible to merge.

Three contour behaviors are worth knowing, because each one turns a wrong input into a passing result:

- **`--set` does not set payload fields.** It substitutes `{{placeholder}}` variables declared by a recipe. Passing `--set minLength=12` is accepted, silently dropped, and the profile still reports that schema validation passed. Field values belong in a recipe's `[profile.fields]` table. The scaffold lists optional fields as comments directly under `[[profile]]`, and keys left in that position are ignored the same way.
- **Validation without `--strict` accepts unknown keys.** An invented key is written into the profile and validation exits `0`. See step 5.
- **DDM identifiers collide.** Contour derives an `Identifier` from the last component of the declaration type, so `passcode.settings` and `softwareupdate.settings` both produce `com.acme.settings`. Two declarations sharing one is a silent overwrite.

Fleet builds the `com.apple.activation.simple` declaration for each configuration declaration it delivers, so an agent doesn't need to write one even though raw DDM requires it.

### Windows is weaker, and worth knowing why

There's no contour equivalent for Windows and no offline validator with an embedded CSP schema, so you carry more of the verification yourself. Grounding still works: instruct the agent to look up the LocURI and its accepted values in the [Microsoft CSP reference](https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-reference) before writing SyncML, and to cite the page in the pull request. That citation is what you check during review.

Fleet catches a subset of errors on its own. It validates the XML structure, rejects LocURI formats that real Windows hosts refuse, and blocks LocURIs that conflict with settings Fleet already manages, such as disk encryption.

> **Warning:** Fleet can't tell you whether a syntactically valid LocURI maps to a CSP that exists. Verify that against the reference, and test on a pilot host before widening scope.

For ADMX-backed policies, which need values pulled out of the `.admx` file on a Windows host, see [Creating Windows configuration profiles (CSPs)](https://fleetdm.com/guides/creating-windows-csps).

## Step 5: Enforce validation in CI

Instructions make the agent likely to validate. CI makes it impossible to skip. Add checks to your pull request workflow, one per delivery method.

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

## Step 6: Review, then deploy

The agent proposes and a human merges. The agent can't merge, can't deploy, and can't touch a device, and that boundary is the whole safety model. Keep it even when the change is small and the agent is right.

Read the diff for the things validation can't check:

- Does this profile address the risk you care about, or did the agent optimize for the wording of your prompt?
- Is the scope right? You asked for one fleet, but does another share the infrastructure this should cover?
- Does this overlap an existing profile, and are you about to ship a conflicting setting?
- Are the identifiers distinct from everything already in the repo?
- Is the pull request description honest about what the change does?

Merge, and CI runs `fleetctl gitops` to apply the change. Scope with fleets and labels the way you would for any other profile. See [Custom OS settings](https://fleetdm.com/guides/custom-os-settings) for the full delivery behavior.

> **Note:** You can also upload a generated file under **Controls > OS settings > Custom settings** in the Fleet UI. You lose the review step, which is the part doing the most work here.

## Verify

1. Go to **Hosts** and select a host in the target fleet.
2. Open the **OS settings** tab.
3. Confirm the profile or declaration shows as **Verified**.

## Related resources

- [Custom OS settings](https://fleetdm.com/guides/custom-os-settings): how Fleet delivers and verifies profiles and DDM declarations.
- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files): every key you can manage as code.
- [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl): convert an existing Fleet instance to a repo.
- [Fleet's `fleet-gitops` skill](https://github.com/fleetdm/fleet/blob/main/.claude/skills/fleet-gitops/SKILL.md): the reference-grounding skill from step 3.
- [contour recipes reference](https://github.com/macadmins/contour/blob/main/docs/contour-recipes.md): the recipe and preset TOML surface, if you want to read one the agent wrote.
- [Creating Windows configuration profiles (CSPs)](https://fleetdm.com/guides/creating-windows-csps): ADMX-backed policies and SyncML structure.

<meta name="articleTitle" value="Build and validate configuration profiles with AI instead of a GUI">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-03">
<meta name="description" value="Describe a configuration profile in a sentence and let an AI agent generate, validate, and ship it through Fleet GitOps.">
