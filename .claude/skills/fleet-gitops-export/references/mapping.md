# Where each Fleet object goes

Every API call below is a GET made through `fleetctl api` (the paths are the `/api/latest/fleet/...` endpoints `fleetctl generate-gitops` uses). Destinations are derived from the `fleetctl new` output at run time: a template glob for the section and platform wins, then an existing `platforms/<os>/<kind>/` folder, then a new folder with that pattern. The table shows what the current templates produce.

"Global" means `default.yml`. On Fleet Premium, hosts in no fleet ("Unassigned") get `fleets/unassigned.yml` for their policies, software, and webhooks, while their controls live in `default.yml` (fleetctl accepts `controls` in only one of the two files, and the template keeps global controls in `default.yml`). On Fleet Free there are no fleets and everything is global.

| Fleet object | API source | Destination |
|---|---|---|
| Org info, server settings, features, Fleet Desktop, host and activity expiry, webhooks, integrations, SSO | `GET /config` | `default.yml` `org_settings.*`, filled into the template's blocks |
| MDM org settings: Apple server URL, end-user authentication, Apple Business, Windows automatic enrollment | `GET /config` | `default.yml` `org_settings.mdm.*` (replaces the template's commented examples) |
| Volume Purchasing Program token-to-fleet assignments | `GET /vpp_tokens` | `org_settings.mdm.volume_purchasing_program` |
| macOS and Windows EULAs | `GET /setup_experience/eula/metadata[?platform=windows]`, `GET /setup_experience/eula/{token}` | `platforms/macos/setup-experience/`, `platforms/windows/setup-experience/`; `org_settings.mdm.end_user_license_agreement` / `windows_eula` |
| Org logos stored in Fleet | `GET /logo?mode=light\|dark` | `platforms/all/org-logos/`; `org_info.org_logo_path_*_mode` (external logo URLs stay URLs) |
| Certificate authorities | `GET /spec/certificate_authorities` | `org_settings.certificate_authorities` (secrets as variables) |
| Microsoft Graph credentials | `GET /microsoft_graph_credentials` | `org_settings.microsoft_graph_credentials` without `client_secret` (Fleet keeps the stored one) |
| Enroll secrets | `GET /spec/enroll_secret`, `GET /fleets/{id}` | `org_settings.secrets` / fleet `settings.secrets` as variables (omitted when secrets are a GitOps exception) |
| YARA rules | `GET /config` (`yara_rules`) | `platforms/all/yara-rules/` + `org_settings.yara_rules` (blocked by a fleetctl bug; see below) |
| Agent options | `GET /config`, `GET /fleets/{id}` | inline `agent_options:` in `default.yml` / the fleet file |
| Custom host vitals | `GET /custom_host_vitals` | inline `custom_host_vitals:` in `default.yml` |
| Global-only controls: Windows MDM, Android MDM, profile validation, Windows migration, Entra IDs, hardware attestation, Apple Business-only enrollment, Apple account provisioning, macOS migration | `GET /config` | `default.yml` `controls:` (the template has slots for some) |
| Fleet controls: OS updates, disk encryption, recovery lock password, host name template, BitLocker PIN, Windows managed local account, Linux key escrow | `GET /fleets/{id}` (`mdm`); `GET /config` for Unassigned | fleet file `controls:`; Unassigned's in `default.yml` |
| Setup experience settings | `GET /fleets/{id}` (`mdm.setup_experience`) | `controls.setup_experience` (`lock_end_user_info` is always written: unset, it follows `enable_end_user_authentication`) |
| Automatic enrollment (ADE) profile | `GET /enrollment_profiles/automatic[?fleet_id=]` | `platforms/macos/enrollment-profiles/<name>`, keeping the stored file name (Fleet records it) |
| Setup experience script | `GET /setup_experience/script[?fleet_id=]` (+ `alt=media`) | `platforms/macos/setup-experience/<name>` |
| Bootstrap package | `GET /mdm/bootstrap/{id}/metadata` | not exported (see below) |
| Apple configuration profiles (`.mobileconfig`) | `GET /configuration_profiles?fleet_id=`, `GET /mdm/profiles/{uuid}?alt=media` | `platforms/macos/configuration-profiles/` (`ios/` for mobile-only fleets) |
| Apple declarations (DDM `.json`) and their activations | same, plus `GET /mdm/profiles/{uuid}` for the activation | `platforms/macos/declaration-profiles/<name>.json`; activation in `declaration-profiles/activations/` |
| DDM assets | `GET /assets?fleet_id=`, `GET /assets/{uuid}?alt=media` | `platforms/macos/assets/<name>.json` + `controls.apple_settings.assets` |
| Windows profiles | profiles endpoints | `platforms/windows/configuration-profiles/<name>.xml` |
| Android profiles | profiles endpoints | `platforms/android/configuration-profiles/<name>.json` |
| Android certificate templates | `GET /certificates?fleet_id=` | inline `controls.android_settings.certificates` |
| Scripts | `GET /scripts?fleet_id=`, `GET /scripts/{id}?alt=media` | `platforms/<macos\|linux\|windows>/scripts/<exact name>` |
| Policies (including patch policies and automations) | `GET /policies`, `GET /fleets/{id}/policies` | `platforms/<os or all>/policies/<name>.yml` |
| Reports | `GET /reports[?fleet_id=]` | `platforms/<os or all>/reports/<name>.yml` |
| Global labels | `GET /spec/labels?fleet_id=0` | `labels/<name>.yml` (the template's `paths: ./labels/*.yml`) |
| Fleet labels | `GET /spec/labels?fleet_id={id}` | `labels/<fleet>/<name>.yml` + `labels: - paths: ../labels/<fleet>/*.yml` in the fleet file |
| Custom packages | `GET /software/titles?available_for_install=1&order_key=name&fleet_id=`, `GET /software/titles/{id}?fleet_id=` | `platforms/<os>/software/<name>.package.yml` (url, hash, scripts, query, icon) beside its scripts; the fleet file entry holds `self_service`, categories, labels, setup experience, display name. Install and uninstall scripts are left out when they're Fleet's default for the package type (`scripts/fleet_default_scripts/`, copied from `pkg/file/scripts/`), so Fleet keeps supplying them. `hash_sha256` is left out when the URL doesn't name the package's version (a `-latest` link, say), because the vendor will replace those bytes; a policy that installs such a package names it by `package_path` |
| Script-only packages (`.sh`, `.ps1`, `.py`) | same | `platforms/<os>/software/<exact file name>` when its SHA-256 matches; otherwise a `hash_sha256` entry |
| Fleet-maintained apps | same, plus `GET /software/fleet_maintained_apps/{id}` | inline `software.fleet_maintained_apps`; install/uninstall scripts only when an admin changed them, in `platforms/<os>/software/`. Fleet tracks that but doesn't return it, so a stored script that's close to the catalog's current one (an older copy of Fleet's script; Fleet replaces unedited ones on the app's next update) is left out and listed for review, and one that differs a lot is kept. `export --keep-fma-scripts` keeps named ones |
| App Store and Google Play apps | same, plus `GET /vpp_tokens` | inline `software.app_store_apps`; managed configurations in `platforms/<os>/managed-app-configurations/` |
| Software icons stored in Fleet | `GET /software/titles/{id}/icon?fleet_id=` | `platforms/all/icons/` |
| Setup experience software | `GET /setup_experience/software?platform=…&fleet_id=` | `setup_experience: true` (or `setup_experience_platform`) on each item |
| Fleets | `GET /fleets`, `GET /fleets/{id}`, `GET /fleets/0` | `fleets/<name>.yml`, `fleets/unassigned.yml` |
| Preflight | `GET /me`, `GET /version` | — |

## Placement rules

- **Shared folders, exact globs.** All fleets share `platforms/<os>/<kind>/`. For each glob a fleet template has, the fleet keeps the glob only if the files it matches are exactly the items that fleet uses (with no per-entry options such as labels or `self_service`). Otherwise the glob is replaced by explicit `path:` entries. This holds for sections the fleet has nothing in, too: an unused glob over a folder with another fleet's files is dropped.
- **Write once.** An item with identical contents in several fleets (or in Global and a fleet) is one file referenced from each.
- **Names.** Files are named from the item name: lowercase ASCII, dashes, emoji dropped. Exceptions, because Fleet derives the item's name from the file name: scripts, Windows `.xml`, DDM `.json`, Android `.json`, DDM assets, script packages, and setup experience files keep their exact names (a profile whose name can't be a file name gets a slug and a `name:` entry). Packages of one app in different formats get the format as a suffix (`zoom-deb.package.yml`, `zoom-rpm.package.yml`). When two different items need the same exact name in one folder, the second goes in a `<fleet>/` subfolder.
- **Platform folder.** Policies and reports go by their `platform` field (one platform → that folder; several or none → `all/`). `.ps1` scripts go to `windows/`; `.sh` and `.py` scripts to `macos/` or `linux/` by content, flagged for review when unclear. Apple profiles go to `macos/` unless the fleet is mobile-only.
- **Template content.** A commented-out example becomes the real setting when Fleet has that setting; guidance comments stay. Example data that doesn't match Fleet is deleted: example label files (kept when a label with that name exists), the example policy, the example ADE profile, and template fleet files with no matching fleet.
- **Literal dollar signs.** fleetctl expands `$VAR` in every YAML file, profile, DDM asset, and pre-install query it reads (not in scripts, activations, ADE profiles, or app configurations) and fails on unset variables. The export writes `\$` wherever Fleet's stored text has a literal `$` that would otherwise be expanded. `$FLEET_VAR_*` and `$FLEET_SECRET_*` are left alone. An enroll secret in a profile or DDM asset becomes that secret's variable (see [secrets.md](secrets.md)).
- **Defaults.** Booleans are written when they differ from the GitOps default. `lock_end_user_info` is always written and `enable_profile_validation` whenever Fleet returns it, because their defaults aren't plain false; a Fleet that doesn't return `enable_profile_validation` predates the setting, and its fleetctl rejects the key.

## No clean home in the `fleetctl new` layout

| Item | What the export does |
|---|---|
| Unassigned hosts | Creates `fleets/unassigned.yml` (policies, software, webhooks); controls stay in `default.yml` |
| Global policies and reports | Platform folders like fleets' items, explicit `path:` entries from `default.yml` |
| Fleet labels | `labels/<fleet>/` subfolder, globbed from the fleet file (out of reach of `default.yml`'s `labels/*.yml`) |
| Software install, uninstall, and post-install scripts; pre-install queries | Beside the package in `platforms/<os>/software/`, not in `scripts/`, where the template's glob would turn them into library scripts |
| DDM activations | `platforms/macos/declaration-profiles/activations/`, out of reach of the `*.json` declaration glob |
| DDM assets | New `platforms/macos/assets/` |
| Setup experience script, EULAs | New `platforms/<os>/setup-experience/` |
| Org logos | New `platforms/all/org-logos/` |
| YARA rules | New `platforms/all/yara-rules/` |
| iOS/iPadOS app managed configurations, in-house `.ipa` packages | New `platforms/<ios\|ipados>/managed-app-configurations/` and `platforms/ios/software/` |
| Agent options, custom host vitals | Inline in the YAML (no folder) |
| `platforms/macos/commands/` | Nothing in GitOps reads it; left as the template made it |

## What isn't exported

| Item | Why |
|---|---|
| Bootstrap packages | Fleet doesn't keep the download URL GitOps needs. Leaving `macos_bootstrap_package` out keeps the package in Fleet, so the dry run is clean, but the repo can't recreate it elsewhere. |
| YARA rules (until fixed) | fleetctl rejects the documented `org_settings.yara_rules[].path` form as an unknown key and the inline form at apply time. The rule files are exported; the dry run fails on them. |
| Fleet-level Jira and Zendesk | Not settable in GitOps. |
| SMTP settings, GitOps-mode settings | Omitting them keeps Fleet's values; the SMTP password can't be read back. |
| Users, roles, API tokens, ABM/VPP/APNs tokens, certificates | Not part of GitOps. |
| Masked secrets and `FLEET_SECRET_*` values | See [secrets.md](secrets.md). |
| Labels, software, or enroll secrets marked as GitOps exceptions | Managed in the Fleet UI by design; the keys are left out (or commented out) because fleetctl fails when they're present. |
