# Parity with `fleetctl generate-gitops`

What `fleetctl generate-gitops` exports (cmd/fleetctl/fleetctl/generate_gitops.go) compared with this skill, so it's clear when `generate-gitops` can be retired. Both read the same endpoints; this skill reads them through `fleetctl api`.

**Status:** ✅ same result · ⬆️ this skill does more or does it more safely · ↔️ intentionally different · ⚠️ open gap · ❌ not in this skill

**Verified:** *round trip* = exported from a local Fleet, applied to that same throwaway instance, every object compared before and after (no differences); *dry run* = clean `fleetctl gitops --dry-run` plus the repo-vs-Fleet check; *–* = not yet exercised.

## Org settings (`default.yml`)

| Item | generate-gitops | This skill | Status | Verified |
|---|---|---|---|---|
| `org_info` (name, contact URL, logo URLs) | exported | exported | ✅ | round trip |
| Org logos stored in Fleet | `lib/org_logo/<mode>.<ext>` + `org_logo_path_*` | `platforms/all/org-logos/` + `org_logo_path_*` | ✅ | – |
| `server_settings` | exported, `server_url` literal | exported, `server_url: $FLEET_URL` as the template has it | ✅ | round trip |
| `features`, `fleet_desktop`, `host_expiry_settings`, `activity_expiry_settings` | exported | exported | ✅ | round trip |
| `webhook_settings` | exported; `failing_policies_webhook.policy_ids` moved to per-policy `webhooks_and_tickets_enabled` | same, destination URLs as variables | ⬆️ | round trip |
| `integrations` (Jira, Zendesk, Google Calendar, Google Workspace, conditional access, certificate identity checks) | exported, tokens replaced by `# TODO` (empty values) | exported, tokens as variables; Google Workspace dropped on Free | ⬆️ | – |
| `sso_settings` | exported, `metadata` and `metadata_url` replaced by `# TODO` | exported, `metadata` as `$FLEET_SSO_METADATA`, `metadata_url` kept (the template keeps it in plain text) | ⬆️ | round trip (`metadata_url`) |
| `mdm.apple_server_url`, `mdm.end_user_authentication` | exported, metadata and URL as `# TODO` | exported, metadata as `$FLEET_MDM_SSO_METADATA`, URL kept | ⬆️ | – |
| `mdm.apple_business`, `mdm.windows_automatic_enrollment` | exported | exported | ✅ | – |
| `mdm.volume_purchasing_program` | from VPP tokens, "All fleets" for unrestricted tokens | same | ✅ | – |
| macOS EULA, Windows EULA | `lib/eula/` | `platforms/<os>/setup-experience/` | ✅ | – |
| `certificate_authorities` | exported, secrets as `# TODO` | exported, secrets as variables; dry run uses the real values | ⬆️ | – |
| `microsoft_graph_credentials` | exported, `client_secret` as `# TODO` | exported without `client_secret`, which keeps Fleet's stored secret | ⬆️ | – |
| Enroll secrets | `# TODO` placeholder, or plaintext with `--insecure` | variables; left out when secrets are a GitOps exception | ⬆️ | round trip |
| `yara_rules` | not exported (warning, writes an empty value) | rule files exported to `platforms/all/yara-rules/`; fleetctl currently rejects `yara_rules[].path`, so the dry run fails | ⚠️ fleetctl bug | – |
| `smtp_settings`, `gitops` | not exported | not exported (omitting them keeps Fleet's values) | ✅ | – |
| `agent_options` | exported | exported (global and per fleet) | ✅ | round trip |
| `custom_host_vitals` | exported | exported | ✅ | round trip |
| Labels (dynamic, manual, host vitals) | exported inline | `labels/*.yml` (the template's glob), fleet labels in `labels/<fleet>/`; template label files reused when the label exists | ✅ | round trip |
| Labels managed in the UI (GitOps exception) | not considered: writes `labels:`, which fleetctl rejects | `labels:` commented out, as the template describes | ⬆️ | dry run |

## Controls

| Item | generate-gitops | This skill | Status | Verified |
|---|---|---|---|---|
| Scripts | `lib/<fleet>/scripts/` | `platforms/<os>/scripts/`, exact names | ✅ | round trip |
| Apple configuration profiles, labels, `self_service`, `hidden`, `name`, `description` | exported | exported | ✅ | round trip |
| DDM declarations and custom activations | exported | exported; activations kept out of the declarations glob | ✅ | round trip |
| DDM assets | exported | exported | ✅ | – |
| Windows profiles | exported | exported | ✅ | round trip |
| Android profiles | exported | exported | ✅ | – |
| Android certificate templates | exported | exported | ✅ | – |
| macOS, iOS, iPadOS, Windows OS updates | exported | exported | ✅ | round trip |
| Disk encryption, key escrow, BitLocker PIN, Linux key escrow | always written | written when on | ✅ | round trip |
| Recovery lock password, host name template, Windows managed local account | exported | exported | ✅ | – |
| Global-only controls (Windows and Android MDM, migration, Entra IDs, attestation, Apple Business-only enrollment, manual Windows MDM, profile validation) | exported | exported | ✅ | round trip (Windows MDM, profile validation) |
| `apple_account_provisioning` | client secret as `# TODO` | client secret as a variable | ⬆️ | – |
| Setup experience settings | `setup_experience: "TODO: …"` (invalid until edited) | flags exported (`lock_end_user_info` always written) | ⬆️ | round trip |
| ADE (automatic enrollment) profile | not exported | exported with its stored file name | ⬆️ | – (needs Apple Business) |
| Setup experience script | not exported | exported | ⬆️ | round trip |
| Bootstrap package | not exported | not exported (no URL in Fleet; leaving it out keeps it) | ✅ | – |

## Policies and reports

| Item | generate-gitops | This skill | Status | Verified |
|---|---|---|---|---|
| Policy fields, `critical` and `hidden` on Premium, labels (all four lists) | exported inline | one file per policy in `platforms/<os or all>/policies/` | ✅ | round trip |
| Patch policies | `fleet_maintained_app_slug`, `patch_when_closed`, `notify_before_patching` | same, plus `install_software: true` (the documented form) | ✅ | round trip |
| `install_software` (Fleet-maintained app, package hash, App Store ID) | exported | exported | ✅ | round trip (FMA, package) |
| `run_script`, `resend_configuration_profile` | exported | exported | ✅ | round trip |
| Reports, labels | exported inline | one file per report | ✅ | round trip |

## Software

| Item | generate-gitops | This skill | Status | Verified |
|---|---|---|---|---|
| Custom packages (URL, hash, scripts, pre-install query, self-service, categories, labels, display name, setup experience) | inline in the fleet file | `platforms/<os>/software/<name>.package.yml` + options on the fleet entry | ↔️ | round trip |
| Titles with several packages | separate package file | package file with one item per package | ✅ | – |
| Script-only packages | `hash_sha256` | the script file itself when its hash matches (reviewable in git), else the hash | ⬆️ | round trip |
| Cross-platform setup experience (`setup_experience_platform`) | exported | exported | ✅ | – |
| In-house `.ipa` apps (deduplicated, managed configuration) | exported | exported | ✅ | – |
| Fleet-maintained apps (slug, pinned version, categories, labels, icon, display name, custom scripts when they differ from the defaults, pre-install query unless a patch policy manages it) | exported | exported | ✅ | round trip |
| App Store and Google Play apps (platform, VPP location with more than one token, versions, managed configuration, auto-update window) | exported | exported | ✅ | – |
| Icons stored in Fleet | `lib/<fleet>/icons/` | `platforms/all/icons/` | ✅ | – |
| Software managed in the UI (GitOps exception) | not considered | `software:` commented out | ⬆️ | dry run |

## Behavior

| Item | generate-gitops | This skill | Status |
|---|---|---|---|
| Layout | `default.yml`, `fleets/`, `lib/<fleet>/…` | the `fleetctl new` layout, read from its output | ↔️ |
| Literal `$` in queries, descriptions, profiles | written as-is; fleetctl later expands them (or fails) | escaped as `\$` | ⬆️ |
| Template guidance comments | none | kept | ⬆️ |
| Placement of shared items | per fleet | written once, referenced from each fleet | ⬆️ |
| Secrets | `# TODO` comments or plaintext (`--insecure`) | variables, `.env.example`, README table, workflow `env:` | ⬆️ |
| Validation | none | YAML parse, references, orphans, repo-vs-Fleet reconcile, dry run | ⬆️ |
| Free tier (global only) | supported | supported | ✅ |
| `--fleet` (one fleet), `--key`, `--print` | supported | whole instance only | ❌ |
| `--insecure` (plaintext secrets) | supported | not supported, by design | ↔️ |

## Before retiring generate-gitops

- Exercise the "–" rows on a dev instance with Apple Business, VPP, Android, a certificate authority, and multi-package titles.
- Decide whether the single-fleet and `--key` modes are still needed.
- Fix the fleetctl `yara_rules[].path` validation bug, then verify YARA rules round-trip.
