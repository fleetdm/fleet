export type KeyTable = {
  name: string;
  domain:
    | "Hosts"
    | "MDM"
    | "Software"
    | "Policies"
    | "Audit & identity"
    | "Queue plumbing";
  oneLine: string;
  whenYouQuery: string[];
  notableColumns?: string[];
};

export const KEY_TABLES: KeyTable[] = [
  {
    name: "hosts",
    domain: "Hosts",
    oneLine:
      "The row that represents one managed device. Everything else joins back here.",
    whenYouQuery: [
      "The customer says 'host X is broken'. this is where you look first.",
      "Checking OS, hardware, last-seen, team membership.",
      "Looking up uuid / hardware_serial to confirm you have the right device.",
    ],
    notableColumns: [
      "id, uuid, hardware_serial",
      "platform (darwin / windows / linux / ios / ipados / android)",
      "team_id (nullable; null = 'Unassigned' / global)",
      "seen_time, policy_updated_at, software_updated_at",
    ],
  },
  {
    name: "host_mdm",
    domain: "MDM",
    oneLine:
      "Per-host MDM enrollment overview. 'Is this device enrolled, with whom, in what state?'",
    whenYouQuery: [
      "Customer reports 'device enrolled but Fleet shows nothing'. check `enrolled` + `server_url`.",
      "Confirming a device is actually in Fleet's MDM vs another MDM.",
      "Looking at Automatic vs Manual enrollment type.",
    ],
    notableColumns: [
      "host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment",
      "mdm_id (foreign key to mobile_device_management_solutions)",
    ],
  },
  {
    name: "host_mdm_apple_profiles",
    domain: "MDM",
    oneLine:
      "Per-host status of each Apple MDM profile. Pending, verifying, verified, failed.",
    whenYouQuery: [
      "Customer says a profile isn't landing on their Mac. Look here for the row and status.",
      "Debugging profile delivery stuck in `pending` for too long.",
      "Finding profiles that failed verification and why.",
    ],
    notableColumns: [
      "host_uuid, profile_uuid, status, operation_type (install / remove)",
      "detail (error message if failed)",
      "command_uuid (links to nano_commands)",
    ],
  },
  {
    name: "host_mdm_windows_profiles",
    domain: "MDM",
    oneLine: "The Windows sibling of host_mdm_apple_profiles.",
    whenYouQuery: [
      "Same reasons as Apple profiles, but for Windows + SyncML profiles.",
    ],
    notableColumns: [
      "host_uuid, profile_uuid, status, operation_type",
      "detail",
    ],
  },
  {
    name: "host_mdm_android_profiles",
    domain: "MDM",
    oneLine: "Android policy status per host.",
    whenYouQuery: [
      "Checking whether a policy update has reached a managed Android device.",
      "Debugging Google's view of policy state vs Fleet's intent.",
    ],
  },
  {
    name: "host_software",
    domain: "Software",
    oneLine:
      "Which software is installed on each host. Populated by osquery detail queries.",
    whenYouQuery: [
      "Answering 'is Chrome installed on this host?'",
      "Finding every host with a vulnerable version.",
      "Comparing osquery-detected software against Fleet's install intent.",
    ],
    notableColumns: [
      "host_id, software_id (foreign key to software table)",
      "last_opened_at (if osquery provides it)",
    ],
  },
  {
    name: "host_software_installs",
    domain: "Software",
    oneLine:
      "Admin-triggered custom-installer status per host. Pending / installed / failed.",
    whenYouQuery: [
      "A custom installer is stuck in `pending` and the admin asks why.",
      "Finding install result output (stdout/stderr tails) for a failed install.",
      "Confirming an install was actually triggered.",
    ],
    notableColumns: [
      "host_id, software_installer_id, status",
      "install_script_exit_code, install_script_output",
      "pre_install_query_output",
    ],
  },
  {
    name: "host_vpp_software_installs",
    domain: "Software",
    oneLine:
      "VPP install status per host. Apple AND Android (platform column distinguishes).",
    whenYouQuery: [
      "Investigating 'VPP install stuck pending' tickets.",
      "Checking license-assignment status.",
      "Finding Android Managed Google Play installs (platform='android').",
    ],
    notableColumns: [
      "host_id, vpp_app_id, platform (darwin / ios / ipados / android)",
      "command_uuid (Apple MDM command reference)",
      "verification_at, verification_failed_at",
    ],
  },
  {
    name: "host_in_house_software_installs",
    domain: "Software",
    oneLine:
      "In-house (private IPA) install status. Fleet-served bytes, Apple MDM delivery.",
    whenYouQuery: [
      "Debugging in-house app distribution to iOS/iPadOS.",
      "Checking signature / cert expiry issues.",
    ],
  },
  {
    name: "upcoming_activities",
    domain: "Queue plumbing",
    oneLine:
      "Per-host queued intent. 'Fleet wants to do this; waiting for fleetd / MDM pickup.'",
    whenYouQuery: [
      "A host has nothing happening. Is anything even queued? Check here first.",
      "Understanding the order Fleet will deliver pending work on a host.",
      "Debugging why an admin's click didn't produce an install.",
    ],
    notableColumns: [
      "host_id, user_id (admin who triggered)",
      "activity_type (software_install, uninstall, script, mdm_command)",
      "fleet_initiated, canceled, created_at",
    ],
  },
  {
    name: "software_titles",
    domain: "Software",
    oneLine:
      "The abstract app. One row = 'Google Chrome'. Shared across platforms + install types.",
    whenYouQuery: [
      "Looking up a title id for an API call.",
      "Finding all installers that back a title (one title can have Mac .pkg + Windows .msi).",
    ],
  },
  {
    name: "software_installers",
    domain: "Software",
    oneLine:
      "The concrete per-platform artifact. .pkg / .msi / .deb / .exe bytes referenced from file store.",
    whenYouQuery: [
      "Finding install_script / uninstall_script / post_install_script contents.",
      "Debugging pre_install_query behavior.",
      "Looking at `self_service` flag + `install_during_setup` flag.",
    ],
    notableColumns: [
      "title_id, team_id, platform",
      "install_script_content_id, uninstall_script_content_id, post_install_script_content_id",
      "pre_install_query, self_service, install_during_setup",
    ],
  },
  {
    name: "vpp_apps",
    domain: "Software",
    oneLine:
      "Apple VPP app catalog Fleet syncs from Apple's VPP API.",
    whenYouQuery: [
      "Confirming an app is in Fleet's VPP catalog.",
      "Finding the Apple App Store identifier for a title.",
    ],
  },
  {
    name: "nano_commands",
    domain: "MDM",
    oneLine:
      "Apple MDM command queue (nanomdm's embedded schema). Every outbound command to a Mac/iPhone/iPad lives here.",
    whenYouQuery: [
      "Debugging 'command sent but device didn't do anything'.",
      "Finding the raw plist for a command.",
      "Correlating a command_uuid from host_mdm_apple_profiles back to the actual queued command.",
    ],
    notableColumns: [
      "command_uuid, request_type (InstallProfile, RemoveProfile, InstallApplication...)",
      "command (plist XML bytes)",
    ],
  },
  {
    name: "windows_mdm_command_queue",
    domain: "MDM",
    oneLine:
      "Outbound SyncML commands pending delivery to Windows devices.",
    whenYouQuery: [
      "Debugging Windows MDM command delivery.",
      "Finding SyncML XML for a specific command.",
    ],
  },
  {
    name: "policies",
    domain: "Policies",
    oneLine:
      "Policy definitions. The SQL that evaluates on hosts, plus automation config.",
    whenYouQuery: [
      "Finding what a policy's query actually does.",
      "Confirming automation flags on a policy (ticket, webhook, script, install).",
      "Checking if a policy is scoped to specific labels.",
    ],
    notableColumns: [
      "team_id, name, query, platform",
      "resolution, critical",
      "software_installer_id, script_id (automation targets)",
      "calendar_events_enabled",
    ],
  },
  {
    name: "policy_membership",
    domain: "Policies",
    oneLine:
      "Which hosts pass or fail each policy. Join table (host ↔ policy ↔ pass/fail).",
    whenYouQuery: [
      "Counting failing hosts for a policy.",
      "Finding why automation didn't fire. Maybe membership shows passing.",
      "Debugging stale membership after a policy edit.",
    ],
    notableColumns: ["host_id, policy_id, passes (0/1), updated_at"],
  },
  {
    name: "activity_past",
    domain: "Audit & identity",
    oneLine:
      "The audit log. Every admin action lands here.",
    whenYouQuery: [
      "'Who did this?'. look here.",
      "Reconstructing a change timeline.",
      "Compliance / audit queries.",
    ],
    notableColumns: [
      "user_id, user_name, user_email",
      "activity_type, details (JSON)",
      "created_at",
    ],
  },
  {
    name: "users",
    domain: "Audit & identity",
    oneLine: "Admin accounts (and API tokens. Tokens are rows in sessions, not here).",
    whenYouQuery: [
      "Finding a user id by email.",
      "Checking role (admin / maintainer / observer).",
      "Confirming SSO vs local-auth user.",
    ],
    notableColumns: [
      "email, global_role (nullable)",
      "sso_enabled, salt, password",
    ],
  },
  {
    name: "sessions",
    domain: "Audit & identity",
    oneLine:
      "Active auth session tokens. User logins AND long-lived API tokens live here.",
    whenYouQuery: [
      "Investigating 'who's logged in right now?'",
      "Revoking a stolen API token (delete the row).",
    ],
    notableColumns: ["user_id, key (the token), access_at, create_at"],
  },
];

export const DOMAIN_ORDER: KeyTable["domain"][] = [
  "Hosts",
  "MDM",
  "Software",
  "Policies",
  "Queue plumbing",
  "Audit & identity",
];
