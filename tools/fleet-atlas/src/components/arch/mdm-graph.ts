import type { ArchEdge, ArchNode } from "./graph-data";

const COL = 220;
const ROW = 150;

export const MDM_NODES: ArchNode[] = [
  // Row 0. Enrollment triggers
  {
    id: "mdm-trigger-dep",
    label: "DEP (Apple ABM)",
    group: "external",
    blurb:
      "Device purchased through ABM and assigned to Fleet's MDM. Enrolls automatically on first boot.",
    repoPaths: [{ path: "server/mdm/nanodep/" }],
    relatedModes: [
      { href: "/scenarios/mac-dep-enrollment", label: "Scenario" },
    ],
    x: 0,
    y: 0,
  },
  {
    id: "mdm-trigger-autopilot",
    label: "Windows Autopilot",
    group: "external",
    premium: true,
    blurb:
      "Device registered in Microsoft Autopilot; redirected to Fleet's MDM at first boot after Entra sign-in.",
    repoPaths: [{ path: "server/cron/microsoft_autopilot_cron.go" }],
    x: COL,
    y: 0,
  },
  {
    id: "mdm-trigger-amapi",
    label: "Android zero-touch / enrollment token",
    group: "external",
    premium: true,
    blurb:
      "Enterprise-owned Android enrolled via Google's zero-touch or an admin-generated enrollment token.",
    repoPaths: [{ path: "server/mdm/android/" }],
    x: COL * 2,
    y: 0,
  },
  {
    id: "mdm-trigger-byod",
    label: "BYOD / user-initiated",
    group: "external",
    premium: true,
    blurb:
      "Employee uses Fleet's enroll URL to opt their personal device into management (work profile on Android, user enrollment on Apple).",
    repoPaths: [{ path: "server/service/mdm.go" }],
    x: COL * 3,
    y: 0,
  },
  {
    id: "mdm-trigger-ota",
    label: "OTA / manual enroll URL",
    group: "external",
    blurb:
      "Admin-generated per-host enroll secret or OTA profile. Common for existing device fleets without ABM/Autopilot.",
    repoPaths: [{ path: "server/service/mdm.go" }],
    x: COL * 4,
    y: 0,
  },

  // Row 1. Protocol endpoints
  {
    id: "mdm-endpoint-apple",
    label: "Apple MDM endpoint",
    group: "ingress",
    blurb:
      "/mdm/apple/mdm (checkin + command), SCEP, DEP, DDM declarative requests, PSSO. Parsed as plist XML.",
    repoPaths: [
      { path: "server/mdm/apple/" },
      { path: "server/mdm/nanomdm/" },
    ],
    relatedModes: [{ href: "/subsystems/apple-mdm", label: "Apple MDM" }],
    x: COL * 0.5,
    y: ROW,
  },
  {
    id: "mdm-endpoint-windows",
    label: "Microsoft SyncML endpoint",
    group: "ingress",
    blurb:
      "/mdm/microsoft/management for SyncML OMA-DM. WSTEP for cert enrollment.",
    repoPaths: [{ path: "server/mdm/microsoft/" }],
    relatedModes: [{ href: "/subsystems/microsoft-mdm", label: "Microsoft MDM" }],
    x: COL * 2,
    y: ROW,
  },
  {
    id: "mdm-endpoint-android",
    label: "AMAPI callbacks (Google Pub/Sub)",
    group: "ingress",
    premium: true,
    blurb:
      "Fleet subscribes to Google Pub/Sub; Google proxies device events (enrollment, status, command results).",
    repoPaths: [{ path: "server/mdm/android/" }],
    relatedModes: [{ href: "/subsystems/android-mdm", label: "Android MDM" }],
    x: COL * 3.5,
    y: ROW,
  },

  // Row 2. Config profile sources
  {
    id: "mdm-source-admin",
    label: "Admin UI upload",
    group: "external",
    blurb:
      ".mobileconfig / SyncML XML / Android JSON uploaded through the Fleet UI.",
    repoPaths: [{ path: "frontend/pages/ManageControlsPage/" }],
    x: 0,
    y: ROW * 2,
  },
  {
    id: "mdm-source-gitops",
    label: "GitOps YAML",
    group: "external",
    blurb:
      "Profiles declared in YAML; fleetctl gitops reconciles the desired set.",
    repoPaths: [{ path: "cmd/fleetctl/" }],
    x: COL,
    y: ROW * 2,
  },
  {
    id: "mdm-source-ddm",
    label: "DDM declarations (Apple)",
    group: "mdm",
    blurb:
      "Apple's newer declarative control plane. Base declarations work on Free; custom activation, label scoping, and OS-update declarations gate premium.",
    repoPaths: [{ path: "server/mdm/apple/" }],
    x: COL * 2,
    y: ROW * 2,
  },
  {
    id: "mdm-source-setup",
    label: "Setup assistant profile",
    group: "mdm",
    premium: true,
    blurb:
      "Custom ABM Setup Assistant JSON delivered during DEP enrollment.",
    repoPaths: [{ path: "server/worker/macos_setup_assistant.go" }],
    x: COL * 3,
    y: ROW * 2,
  },
  {
    id: "mdm-source-variables",
    label: "Fleet variables + secret vars",
    group: "service",
    premium: true,
    blurb:
      "$FLEET_VAR_* placeholders in profiles get substituted per host at delivery time.",
    repoPaths: [{ path: "server/variables/" }],
    x: COL * 4,
    y: ROW * 2,
  },

  // Row 3. Reconciliation crons
  {
    id: "mdm-cron-apple",
    label: "mdm_apple_profile_manager (30s)",
    group: "background",
    blurb:
      "Diffs desired vs installed Apple profiles per host; queues install/remove commands.",
    repoPaths: [{ path: "server/service/mdm.go" }],
    x: 0,
    y: ROW * 3,
  },
  {
    id: "mdm-cron-windows",
    label: "mdm_windows_profile_manager (30s)",
    group: "background",
    blurb: "SyncML reconciler. Writes to windows_mdm_command_queue.",
    repoPaths: [{ path: "server/service/mdm_windows.go" }],
    x: COL,
    y: ROW * 3,
  },
  {
    id: "mdm-cron-android",
    label: "mdm_android_profile_manager (30s)",
    group: "background",
    premium: true,
    blurb:
      "Diffs Android policy vs Google's view and calls AMAPI to update.",
    repoPaths: [{ path: "server/mdm/android/" }],
    x: COL * 2,
    y: ROW * 3,
  },
  {
    id: "mdm-cron-dep",
    label: "apple_mdm_dep_profile_assigner (1m)",
    group: "background",
    blurb: "Pushes DEP enrollment profile assignments to ABM.",
    repoPaths: [{ path: "cmd/fleet/cron.go" }],
    x: COL * 3,
    y: ROW * 3,
  },
  {
    id: "mdm-cron-autopilot",
    label: "microsoft_autopilot_sync (5m)",
    group: "background",
    premium: true,
    blurb: "Pulls Autopilot device registrations from Microsoft Graph.",
    repoPaths: [{ path: "server/cron/microsoft_autopilot_cron.go" }],
    x: COL * 4,
    y: ROW * 3,
  },

  // Row 4. Command queues
  {
    id: "mdm-queue-apple",
    label: "nano_commands + nano_enrollment_queue",
    group: "store",
    blurb: "Apple MDM command queue (nanomdm's schema).",
    repoPaths: [{ path: "server/datastore/mysql/nanomdm_storage.go" }],
    relatedModes: [{ href: "/background/queues", label: "Queues" }],
    x: 0,
    y: ROW * 4,
  },
  {
    id: "mdm-queue-windows",
    label: "windows_mdm_command_queue",
    group: "store",
    blurb: "Outbound SyncML commands for Windows devices.",
    repoPaths: [{ path: "server/datastore/mysql/microsoft_mdm.go" }],
    x: COL * 1.7,
    y: ROW * 4,
  },
  {
    id: "mdm-queue-android",
    label: "mdm_android_commands",
    group: "store",
    premium: true,
    blurb: "Android commands pending delivery via AMAPI.",
    repoPaths: [{ path: "server/mdm/android/" }],
    x: COL * 3.4,
    y: ROW * 4,
  },

  // Row 5. Push
  {
    id: "mdm-push-apns",
    label: "APNs push (via apple_mdm_apns_pusher cron, 1m)",
    group: "mdm",
    blurb: "Fleet pushes to APNs; Apple delivers to the device; device wakes.",
    repoPaths: [{ path: "cmd/fleet/cron.go" }],
    x: 0,
    y: ROW * 5,
  },
  {
    id: "mdm-push-wns",
    label: "Windows: device polls",
    group: "mdm",
    blurb:
      "Microsoft MDM is pull-mostly: devices poll Fleet on their own schedule. No per-command push.",
    repoPaths: [],
    x: COL * 1.7,
    y: ROW * 5,
  },
  {
    id: "mdm-push-fcm",
    label: "Google delivers to device",
    group: "mdm",
    premium: true,
    blurb:
      "Fleet calls AMAPI; Google handles the push to Android devices on its own.",
    repoPaths: [],
    x: COL * 3.4,
    y: ROW * 5,
  },

  // Row 6. Device execution
  {
    id: "mdm-device-execute",
    label: "Device pulls command, executes, ACKs",
    group: "external",
    blurb:
      "OS MDM client parses the command (plist / SyncML / policy JSON) and reports back.",
    repoPaths: [],
    x: COL * 1.7,
    y: ROW * 6,
  },

  // Row 7. Status write-back
  {
    id: "mdm-status-apple",
    label: "host_mdm_apple_profiles",
    group: "store",
    blurb:
      "Per-host profile status (pending → verifying → verified → failed).",
    repoPaths: [{ path: "server/datastore/mysql/mdm_apple.go" }],
    x: 0,
    y: ROW * 7,
  },
  {
    id: "mdm-status-windows",
    label: "host_mdm_windows_profiles",
    group: "store",
    blurb: "Windows profile per-host status.",
    repoPaths: [{ path: "server/datastore/mysql/microsoft_mdm.go" }],
    x: COL,
    y: ROW * 7,
  },
  {
    id: "mdm-status-android",
    label: "host_mdm_android_profiles",
    group: "store",
    premium: true,
    blurb: "Android policy per-host status.",
    repoPaths: [{ path: "server/mdm/android/" }],
    x: COL * 2,
    y: ROW * 7,
  },
  {
    id: "mdm-status-host",
    label: "host_mdm (enrollment overview)",
    group: "store",
    blurb:
      "The 'is this host MDM-enrolled, with whom, in what state' row surfaced in the host table.",
    repoPaths: [{ path: "server/datastore/mysql/hosts.go" }],
    x: COL * 3,
    y: ROW * 7,
  },
  {
    id: "mdm-status-compliance",
    label: "Compliance partner feedback",
    group: "service",
    premium: true,
    blurb:
      "Microsoft compliance partner: Fleet reports device compliance to Entra, which gates conditional access.",
    repoPaths: [{ path: "server/service/compliance_partner.go" }],
    x: COL * 4,
    y: ROW * 7,
  },
];

export const MDM_EDGES: ArchEdge[] = [
  // Triggers → protocol endpoints
  {
    from: "mdm-trigger-dep",
    to: "mdm-endpoint-apple",
    tooltip:
      "Device shipped via Apple Business Manager auto-enrolls on first boot. Apple sees it's assigned to Fleet and hands it over to Fleet's /mdm/apple/mdm endpoint; the user never sees an enroll URL.",
    techHelp:
      "DEP (Device Enrollment Program, now called Automated Device Enrollment) is Apple's zero-touch path. The admin uploads device serials to Apple Business Manager once and assigns them to Fleet's MDM server. From then on, every assigned device enrolls automatically. The device doesn't need to know Fleet's URL, Apple injects it at activation time.",
  },
  {
    from: "mdm-trigger-ota",
    to: "mdm-endpoint-apple",
    tooltip:
      "Fallback Apple enrollment for devices not in ABM. Admin generates an OTA enroll URL per host, user opens it in Safari, Fleet serves the MDM enrollment profile the user installs. Lands at the same /mdm/apple/mdm endpoint after install.",
    techHelp:
      "OTA (Over-The-Air) enrollment is the manual path for existing device fleets without ABM. The URL carries a one-time enroll secret; after install, the device begins the standard Apple MDM checkin/command cycle just like a DEP-enrolled device.",
  },
  {
    from: "mdm-trigger-byod",
    to: "mdm-endpoint-apple",
    dashed: true,
    tooltip:
      "Employee uses Fleet's self-enroll URL on their personal Mac/iPhone, which installs a user-scoped MDM profile (not the full-device supervised profile). Still ends at the Apple MDM endpoint, with a narrower command surface.",
    techHelp:
      "Apple's user enrollment is a privacy-preserving variant of MDM: Fleet can manage work data, work apps, and work credentials, but can't see personal data or wipe the whole device. The device reports a separate 'user' enrollment type so Fleet knows to use the restricted command set.",
  },
  {
    from: "mdm-trigger-autopilot",
    to: "mdm-endpoint-windows",
    tooltip:
      "Device registered in Microsoft Autopilot boots and signs in to Entra. Entra tells it Fleet is its MDM; device then POSTs SyncML to /mdm/microsoft/management to begin enrollment.",
    techHelp:
      "Autopilot is Microsoft's answer to Apple DEP. Admin uploads device hardware hashes to the Autopilot service once and assigns them a profile pointing at Fleet. New devices at first-boot enroll automatically after the user signs in to Entra ID.",
  },
  {
    from: "mdm-trigger-ota",
    to: "mdm-endpoint-windows",
    dashed: true,
    tooltip:
      "Windows manual enrollment. User runs an enroll URL via Settings → \"Access work or school.\" Device then enrolls through the same /mdm/microsoft/management endpoint as Autopilot.",
  },
  {
    from: "mdm-trigger-amapi",
    to: "mdm-endpoint-android",
    tooltip:
      "Admin generates an enrollment token (QR code the user scans). Device registers with Google, which provisions it and POSTs an ENROLLMENT message to Fleet's Pub/Sub subscription. The 'endpoint' is a Pub/Sub consumer, not a device-to-Fleet connection.",
    techHelp:
      "Android MDM inverts the usual pattern: devices NEVER talk directly to Fleet. Fleet calls Google's AMAPI (Android Management API) to send commands; Google delivers them to the device; device state changes come back as Google Pub/Sub messages Fleet subscribes to. Fleet is always talking to Google, Google is always talking to the device.",
  },
  {
    from: "mdm-trigger-byod",
    to: "mdm-endpoint-android",
    dashed: true,
    tooltip:
      "Employee enrolls their personal Android via Fleet's enroll URL, which creates a work profile alongside the personal profile. Google treats this as a BYOD enrollment and delivers to Fleet via the same Pub/Sub subscription.",
    techHelp:
      "Android's work profile is a separate container on the device. Work apps, work data, and work accounts live in their own sandbox the admin can manage. Personal apps, photos, and accounts stay invisible to Fleet. The device shows two home screens with a divider between personal and work.",
  },

  // Config sources → crons
  {
    from: "mdm-source-admin",
    to: "mdm-cron-apple",
    dashed: true,
    tooltip:
      "Admin UI upload writes the .mobileconfig into mdm_apple_configuration_profiles. The 30s reconciler reads this table as the desired state and figures out which hosts are missing which profiles.",
  },
  {
    from: "mdm-source-admin",
    to: "mdm-cron-windows",
    dashed: true,
    tooltip:
      "Admin UI upload lands as a row in mdm_windows_configuration_profiles. The Windows reconciler picks it up on the next 30s tick and queues SyncML Add commands for any host missing it.",
  },
  {
    from: "mdm-source-admin",
    to: "mdm-cron-android",
    dashed: true,
    tooltip:
      "Admin UI uploads an Android policy JSON; the Android profile manager calls AMAPI to attach it to the right devices on the next 30s tick.",
  },
  {
    from: "mdm-source-gitops",
    to: "mdm-cron-apple",
    dashed: true,
    tooltip:
      "fleetctl gitops writes the YAML-declared profile set into the same mdm_apple_configuration_profiles table the Admin UI writes to. The cron doesn't distinguish authors. It just reconciles against whatever rows exist.",
  },
  {
    from: "mdm-source-gitops",
    to: "mdm-cron-windows",
    dashed: true,
    tooltip:
      "GitOps YAML lands as rows in the Windows profile table. From the cron's perspective, GitOps and Admin UI are indistinguishable sources.",
  },
  {
    from: "mdm-source-gitops",
    to: "mdm-cron-android",
    dashed: true,
    tooltip:
      "GitOps authors the Android policy JSON set the Android reconciler then pushes to Google via AMAPI.",
  },
  {
    from: "mdm-source-ddm",
    to: "mdm-cron-apple",
    dashed: true,
    tooltip:
      "DDM declarations live in mdm_apple_declarations, a separate table from legacy .mobileconfig profiles. The Apple reconciler watches both tables and dispatches the right command (InstallProfile vs DeclarativeManagement) depending on which authored the desired state.",
    techHelp:
      "DDM (Declarative Device Management) is Apple's newer MDM model. Instead of imperative \"install this profile\" commands and ack-tracking, Fleet declares the desired-state object and the device applies it locally and self-reports status. Think of it as the shift from imperative DOM manipulation to a declarative framework. Fleet describes what should be true, the device reconciles. Base DDM works on Fleet Free; custom activation, label scoping, and OS update declarations are premium.",
  },
  {
    from: "mdm-source-setup",
    to: "mdm-cron-dep",
    dashed: true,
    tooltip:
      "Setup assistant JSON is attached by the DEP cron to each enrollment profile it pushes to Apple Business Manager, so new DEP-enrolled devices land in a custom setup experience instead of Apple's default.",
  },
  {
    from: "mdm-source-variables",
    to: "mdm-cron-apple",
    dashed: true,
    tooltip:
      "$FLEET_VAR_* placeholders in profile bodies (e.g. $FLEET_VAR_HOST_HARDWARE_UUID) get substituted per-host at the exact moment the Apple reconciler is about to enqueue the InstallProfile command. Substitution happens at delivery, not at upload.",
  },

  // Endpoints feed state → crons
  {
    from: "mdm-endpoint-apple",
    to: "mdm-cron-apple",
    tooltip:
      "Enrollment events from the Apple MDM endpoint populate host_mdm so the reconciler knows the host exists and can be targeted. Command results flowing in through the same endpoint also feed back to flip 'verifying' rows to 'verified.'",
  },
  {
    from: "mdm-endpoint-windows",
    to: "mdm-cron-windows",
    tooltip:
      "Windows enrollment through SyncML writes to host_mdm; the Windows reconciler picks it up on its next tick and starts queueing profiles for the new host.",
  },
  {
    from: "mdm-endpoint-android",
    to: "mdm-cron-android",
    tooltip:
      "A Google Pub/Sub ENROLLMENT message registers the Android host; subsequent reconciliation calls AMAPI to apply the host's policy set.",
  },

  // Crons → queues
  {
    from: "mdm-cron-apple",
    to: "mdm-queue-apple",
    label: "enqueue",
    tooltip:
      "The Apple reconciler inserts install/remove commands into nano_commands + nano_enrollment_queue, the nanomdm-managed Apple command queue, keyed by device UUID.",
    techHelp:
      "nanomdm is an open-source Apple MDM server library Fleet uses to handle the low-level protocol: command storage, device enrollment persistence, APNs coordination. The nano_* tables are nanomdm's schema, not Fleet-native. They predate Fleet's MDM support and Fleet runs nanomdm as an embedded library rather than reimplementing it.",
  },
  {
    from: "mdm-cron-windows",
    to: "mdm-queue-windows",
    label: "enqueue",
    tooltip:
      "The Windows reconciler writes SyncML Add/Replace/Delete commands into windows_mdm_command_queue; devices will pull them on their next scheduled poll.",
  },
  {
    from: "mdm-cron-android",
    to: "mdm-queue-android",
    label: "enqueue",
    tooltip:
      "The Android reconciler stages pending AMAPI calls in mdm_android_commands. The queue decouples the reconciler from AMAPI delivery (which has its own rate limits).",
  },
  {
    from: "mdm-cron-dep",
    to: "mdm-queue-apple",
    dashed: true,
    tooltip:
      "When DEP profile assignment changes trigger a supplementary MDM command (e.g. re-enrollment, device lock), the DEP cron inserts into the Apple command queue as a side effect of the DEP sync.",
  },
  {
    from: "mdm-cron-autopilot",
    to: "mdm-queue-windows",
    dashed: true,
    tooltip:
      "After Autopilot assignment lands and the device finishes initial enrollment, the Autopilot cron can enqueue a Windows command (e.g. compliance check, policy refresh) to run post-enrollment setup.",
  },

  // Queues → push
  {
    from: "mdm-queue-apple",
    to: "mdm-push-apns",
    tooltip:
      "Apple MDM commands sit idle in the queue until a push wakes the device. The apns_pusher cron (every 1m) reads the queue, picks hosts with pending commands, and sends an APNs push to each.",
    techHelp:
      "APNs (Apple Push Notification service) is Apple's push gateway. For MDM, Fleet uses APNs as a doorbell. The push has no payload, it just tells the device \"come pull your pending commands.\" The device then calls back into /mdm/apple/mdm and fetches them. This keeps commands auditable (they live in Fleet, not in a push payload) and the push lightweight.",
  },
  {
    from: "mdm-queue-windows",
    to: "mdm-push-wns",
    tooltip:
      "Windows MDM is pull-mostly. Devices poll Fleet on their own interval. The queue mostly sits until the next device poll; push via WNS is an optional wake-up signal for urgency.",
    techHelp:
      "WNS (Windows Notification Service) is Microsoft's push gateway. In Windows MDM, Microsoft chose a pull model: devices poll Fleet at a configurable cadence rather than requiring Fleet to push. WNS exists as a nice-to-have for faster delivery of time-sensitive commands but isn't on the critical path.",
  },
  {
    from: "mdm-queue-android",
    to: "mdm-push-fcm",
    tooltip:
      "Fleet calls AMAPI with the queued command; Google internally handles the FCM push to the device. Fleet never touches FCM directly. Google abstracts it.",
    techHelp:
      "FCM (Firebase Cloud Messaging) is Google's push gateway, the Android equivalent of APNs. For Android MDM, Fleet never calls FCM. Google sits in the middle. Fleet calls AMAPI, Google routes to FCM, Google pushes to the device. One more hop than Apple MDM, but with Google handling the delivery details.",
  },

  // Push → device execute
  {
    from: "mdm-push-apns",
    to: "mdm-device-execute",
    tooltip:
      "APNs delivers the push; the device wakes, calls back to /mdm/apple/mdm, pulls pending commands, parses the plist, executes locally.",
  },
  {
    from: "mdm-push-wns",
    to: "mdm-device-execute",
    tooltip:
      "On its next poll interval, the Windows device fetches the SyncML queue from Fleet, executes each operation locally, and POSTs results back.",
  },
  {
    from: "mdm-push-fcm",
    to: "mdm-device-execute",
    tooltip:
      "Google-mediated delivery lands at the Android Device Policy app on the device, which applies the policy update and reports state back to Google (and from there to Fleet via Pub/Sub).",
  },

  // Device → status write-back
  {
    from: "mdm-device-execute",
    to: "mdm-status-apple",
    tooltip:
      "Device ACKs each MDM command via the same /mdm/apple/mdm endpoint. Fleet flips rows in host_mdm_apple_profiles from pending → verifying → verified (or failed with reason).",
  },
  {
    from: "mdm-device-execute",
    to: "mdm-status-windows",
    tooltip:
      "Windows device's SyncML response envelope includes status codes per operation. Fleet writes those into host_mdm_windows_profiles.",
  },
  {
    from: "mdm-device-execute",
    to: "mdm-status-android",
    tooltip:
      "Google Pub/Sub STATUS_REPORT messages arrive asynchronously after each AMAPI command. Fleet updates host_mdm_android_profiles based on each report.",
  },
  {
    from: "mdm-device-execute",
    to: "mdm-status-host",
    dashed: true,
    tooltip:
      "The umbrella host_mdm row updates when enrollment state itself changes. New enrollment, de-enrollment, re-enrollment. Per-profile status doesn't flip this row; it's the top-level \"is this host MDM-managed\" signal.",
  },
  {
    from: "mdm-status-host",
    to: "mdm-status-compliance",
    dashed: true,
    tooltip:
      "For Windows hosts enrolled via Autopilot, Fleet reports compliance state (encrypted / updated / enrolled) to Entra, which gates conditional access to Entra-protected resources.",
    techHelp:
      "The Microsoft compliance partner integration lets Fleet act as a recognized compliance signal source. When a device fails Fleet policies, Fleet pushes a \"non-compliant\" verdict to Entra, which can block that device from accessing Office, SharePoint, or other Entra-protected apps until it's remediated. This is premium because it's a paid Microsoft integration partnership.",
  },
];
