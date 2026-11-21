import type { ArchEdge, ArchNode } from "./graph-data";

const COL = 220;
const ROW = 150;

// Everything the software/auto-patching team touches, top → bottom:
//   Row 0: install triggers (where installs come from)
//   Row 1: software title catalogs (what we're installing)
//   Row 2: service orchestration
//   Row 3: intent queues (upcoming_activities + per-type)
//   Row 4: delivery mechanisms (osquery pull / APNs push / VPP API)
//   Row 5: device-side execution
//   Row 6: result + inventory refresh
//   Row 7: vuln pipeline (reads inventory, writes CVE matches)
//   Row 8: feedback loops (automations fire on new vulns / failing policies)

export const SOFTWARE_NODES: ArchNode[] = [
  // Row 0. Install triggers
  {
    id: "sw-trigger-admin",
    label: "Admin: Install button",
    group: "external",
    premium: true,
    blurb: "POST /api/v1/fleet/hosts/{id}/software/{title_id}/install",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    relatedModes: [{ href: "/scenarios/install-from-host-library", label: "Scenario" }],
    x: COL * -0.5,
    y: 0,
  },
  {
    id: "sw-trigger-self-service",
    label: "Self-service: My Device",
    group: "external",
    premium: true,
    blurb: "POST /device/{token}/software/install/{title_id}. Rate-limited; only self_service=true titles.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    relatedModes: [{ href: "/scenarios/self-service-install", label: "Scenario" }],
    x: COL * 0.5,
    y: 0,
  },
  {
    id: "sw-trigger-gitops",
    label: "GitOps: Declarative YAML",
    group: "external",
    premium: true,
    blurb: "fleetctl gitops batch endpoint. Reconciles the full desired software list; can REMOVE as well.",
    repoPaths: [{ path: "cmd/fleetctl/" }],
    relatedModes: [{ href: "/scenarios/gitops-software-install", label: "Scenario" }],
    x: COL * 1.5,
    y: 0,
  },
  {
    id: "sw-trigger-policy",
    label: "Policy automation",
    group: "external",
    premium: true,
    blurb: "Failing-policy auto-install. automations cron (10m) finds failing hosts and calls InstallSoftwareTitle.",
    repoPaths: [{ path: "cmd/fleet/cron.go" }],
    relatedModes: [{ href: "/scenarios/policy-driven-auto-install", label: "Scenario" }],
    x: COL * 2.5,
    y: 0,
  },
  {
    id: "sw-trigger-setup",
    label: "Setup experience",
    group: "external",
    premium: true,
    blurb: "Required titles installed during Mac setup assistant, before user login.",
    repoPaths: [{ path: "server/service/mdm.go" }],
    relatedModes: [{ href: "/scenarios/setup-experience-install", label: "Scenario" }],
    x: COL * 3.5,
    y: 0,
  },
  {
    id: "sw-trigger-fma-auto",
    label: "FMA auto-update",
    group: "external",
    premium: true,
    blurb: "maintained_apps_auto_update cron (1h). Catalog bump → install for behind hosts in opted-in fleets.",
    repoPaths: [{ path: "ee/server/maintainedapps/" }],
    relatedModes: [{ href: "/scenarios/fma-auto-update", label: "Scenario" }],
    x: COL * 4.5,
    y: 0,
  },
  {
    id: "sw-trigger-uninstall",
    label: "Admin: Uninstall button",
    group: "external",
    premium: true,
    blurb: "POST /api/v1/fleet/hosts/{id}/software/{title_id}/uninstall. Different service method, same delivery path.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    x: COL * 5.5,
    y: 0,
  },

  // Row 1. Software title catalogs (what we're installing)
  {
    id: "sw-catalog-custom",
    label: "Custom installers",
    group: "service",
    premium: true,
    blurb: ".pkg/.msi/.deb uploaded by admin. Bytes in file store; metadata in software_installers.",
    repoPaths: [
      { path: "server/datastore/mysql/software_installers.go" },
    ],
    x: COL * 0,
    y: ROW,
  },
  {
    id: "sw-catalog-fma",
    label: "Fleet-maintained apps (FMA)",
    group: "service",
    premium: true,
    blurb:
      "Homebrew casks (macOS) + winget packages (Windows). Nightly maintained_apps cron rebuilds the catalog.",
    repoPaths: [{ path: "ee/server/maintainedapps/" }],
    x: COL * 1.5,
    y: ROW,
  },
  {
    id: "sw-catalog-vpp",
    label: "Apple VPP apps",
    group: "service",
    premium: true,
    blurb:
      "App Store volume license. Fleet syncs catalog via refresh_vpp_app_versions cron; doesn't own bytes.",
    repoPaths: [{ path: "cmd/fleet/cron.go" }],
    x: COL * 3,
    y: ROW,
  },
  {
    id: "sw-catalog-inhouse",
    label: "In-house apps",
    group: "service",
    premium: true,
    blurb: "Private IPAs (iOS/iPadOS) and private installers. Fleet owns the bytes.",
    repoPaths: [{ path: "server/datastore/mysql/in_house_apps.go" }],
    x: COL * 4.5,
    y: ROW,
  },

  // Row 2. Service orchestration
  {
    id: "sw-svc-install",
    label: "svc.InstallSoftwareTitle",
    group: "service",
    premium: true,
    blurb:
      "The service entry point. Branches on title type (custom / FMA / VPP / in-house) and takes the appropriate persistence path.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    x: COL * 2,
    y: ROW * 2,
  },
  {
    id: "sw-svc-uninstall",
    label: "svc.UninstallSoftwareTitle",
    group: "service",
    premium: true,
    blurb: "Uninstall counterpart; writes an uninstall intent that fleetd picks up.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    x: COL * 4,
    y: ROW * 2,
  },

  // Row 3. Intent queues (upcoming_activities family)
  {
    id: "sw-queue-upcoming",
    label: "upcoming_activities + per-type queues",
    group: "store",
    premium: true,
    blurb:
      "software_install_upcoming_activities, vpp_app_upcoming_activities, in_house_app_upcoming_activities. Status=pending visible to UI.",
    repoPaths: [{ path: "server/datastore/mysql/software_installers.go" }],
    relatedModes: [{ href: "/background/queues", label: "Queues" }],
    x: COL * 2,
    y: ROW * 3,
  },
  {
    id: "sw-queue-status",
    label: "host_software_installs (pending → installed/failed)",
    group: "store",
    premium: true,
    blurb:
      "The row admins/users watch. host_vpp_software_installs + host_in_house_software_installs are the siblings.",
    repoPaths: [{ path: "server/datastore/mysql/software_installers.go" }],
    x: COL * 4,
    y: ROW * 3,
  },

  // Row 4. Delivery paths
  {
    id: "sw-delivery-osquery",
    label: "Host osquery poll (every ~10s)",
    group: "background",
    premium: true,
    blurb:
      "fleetd pulls work on its config poll. Custom installers + in-house apps + uninstalls all travel this path.",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: COL * 0.5,
    y: ROW * 4,
  },
  {
    id: "sw-delivery-apns",
    label: "Apple MDM queue + APNs push",
    group: "mdm",
    premium: true,
    blurb:
      "apple_mdm_apns_pusher cron sends APNs; device wakes and pulls InstallEnterpriseApplication / InstallApplication.",
    repoPaths: [{ path: "server/mdm/apple/" }],
    x: COL * 2.5,
    y: ROW * 4,
  },
  {
    id: "sw-delivery-vpp-worker",
    label: "apple_software worker job",
    group: "background",
    premium: true,
    blurb:
      "VPP license-assign call to Apple's VPP API. integrations cron (1m) drains it.",
    repoPaths: [{ path: "server/worker/vpp_verification.go" }],
    x: COL * 4.5,
    y: ROW * 4,
  },

  // Row 5. Device-side execution
  {
    id: "sw-device-fleetd",
    label: "fleetd downloads + runs scripts",
    group: "external",
    premium: true,
    blurb:
      "Pre-install → install → post-install scripts run locally with the installer bytes streamed from Fleet.",
    repoPaths: [{ path: "orbit/" }],
    x: COL * 0.5,
    y: ROW * 5,
  },
  {
    id: "sw-device-apple",
    label: "Device downloads from Apple",
    group: "mdm",
    premium: true,
    blurb: "For VPP: Apple pushes the app from its own CDN. Fleet never sees the bytes.",
    repoPaths: [],
    x: COL * 3.5,
    y: ROW * 5,
  },

  // Row 6. Result + inventory refresh
  {
    id: "sw-result",
    label: "Install result reported",
    group: "service",
    premium: true,
    blurb:
      "fleetd POSTs to /api/fleet/orbit/software_install_result (custom); Apple MDM ACK (VPP). Status flips.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    x: COL * 1.5,
    y: ROW * 6,
  },
  {
    id: "sw-inventory",
    label: "host_software refresh (osquery detail)",
    group: "store",
    blurb:
      "Next osquery detail-queries pick up the install change. host_software reflects reality.",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: COL * 3.5,
    y: ROW * 6,
  },

  // Row 7. Vuln pipeline
  {
    id: "sw-vuln-cron",
    label: "vulnerabilities cron (1h)",
    group: "vuln",
    blurb: "Download feeds, extract CPEs/CVEs, match installed software, write matches.",
    repoPaths: [{ path: "server/vulnerabilities/" }],
    relatedModes: [{ href: "/subsystems/vuln-processing", label: "Vuln pipeline" }],
    x: COL * 0.5,
    y: ROW * 7,
  },
  {
    id: "sw-vuln-feeds",
    label: "Feeds: NVD / OSV / OVAL / MSRC / Office",
    group: "vuln",
    blurb: "External CVE and software-identifier feeds.",
    repoPaths: [{ path: "cmd/cve/" }, { path: "cmd/osv-processor/" }],
    x: COL * 2.5,
    y: ROW * 7,
  },
  {
    id: "sw-vuln-write",
    label: "software_cve + vulnerability_host_counts",
    group: "store",
    blurb: "Matched CVE rows + pre-aggregated host-count rollups.",
    repoPaths: [{ path: "server/datastore/mysql/vulnerabilities.go" }],
    x: COL * 4.5,
    y: ROW * 7,
  },

  // Row 8. Feedback loops
  {
    id: "sw-feedback-automation",
    label: "automations cron (10m)",
    group: "background",
    premium: true,
    blurb:
      "Reads new failing policies + new vuln matches; enqueues Jira/Zendesk workers and policy-driven installs.",
    repoPaths: [{ path: "cmd/fleet/cron.go" }],
    x: COL * 1.5,
    y: ROW * 8,
  },
  {
    id: "sw-feedback-tickets",
    label: "Jira / Zendesk worker jobs",
    group: "background",
    premium: true,
    blurb: "integrations cron (1m) drains them. External tickets land for ops teams.",
    repoPaths: [{ path: "server/worker/jira.go" }, { path: "server/worker/zendesk.go" }],
    x: COL * 3.5,
    y: ROW * 8,
  },
];

export const SOFTWARE_EDGES: ArchEdge[] = [
  // Triggers → service entry
  {
    from: "sw-trigger-admin",
    to: "sw-svc-install",
    tooltip:
      "Admin clicks Install on a host's software tab → POST to the per-host install endpoint → the service method decides what to persist and how to deliver, based on the title's type. Every manual install flows through this one entry point.",
  },
  {
    from: "sw-trigger-self-service",
    to: "sw-svc-install",
    tooltip:
      "End user clicks Install on their My Device page. Request is authed by a per-host device token (not an admin session) and runs the same service method as admin-initiated installs, but with narrower authz (self_service=true titles only, rate-limited).",
  },
  {
    from: "sw-trigger-gitops",
    to: "sw-svc-install",
    tooltip:
      "fleetctl gitops reconciles the full desired software list declared in YAML against the actual installed state. For each host that's missing something, it calls the same per-host service method as the Install button. Gitops is just a loop over hosts.",
  },
  {
    from: "sw-trigger-policy",
    to: "sw-svc-install",
    tooltip:
      "The automations cron (10m) looks at policies configured with an install action and finds hosts that are newly failing. For each one, it calls the install service method on the host's behalf. Same code path, no human in the loop.",
  },
  {
    from: "sw-trigger-setup",
    to: "sw-svc-install",
    tooltip:
      "Titles flagged as 'required at setup' get queued through the install service method during Mac setup assistant, before the user ever logs in. The device waits for these to finish before Fleet releases it out of setup experience.",
  },
  {
    from: "sw-trigger-fma-auto",
    to: "sw-svc-install",
    tooltip:
      "The maintained_apps_auto_update cron (1h) compares each FMA's current catalog version against the version installed on every opted-in host. For any host behind, it calls the install service method to push the new version. The catalog bump is the trigger.",
  },
  {
    from: "sw-trigger-uninstall",
    to: "sw-svc-uninstall",
    tooltip:
      "Admin Uninstall button hits a separate service method, but the shape mirrors install: write an uninstall intent, deliver via the same queue path (fleetd pulls, runs the uninstall script).",
  },

  // Catalogs → service (which titles are available to install)
  {
    from: "sw-catalog-custom",
    to: "sw-svc-install",
    dashed: true,
    tooltip:
      "The custom installer catalog is the pool of admin-uploaded .pkg/.msi/.deb titles the service can target. If a title isn't in this catalog, the install endpoint rejects the request. The catalog defines what's installable.",
  },
  {
    from: "sw-catalog-fma",
    to: "sw-svc-install",
    dashed: true,
    tooltip:
      "The FMA catalog is Fleet's curated list of well-known apps (Chrome, Zoom, 1Password, etc.) with pre-written install/uninstall scripts. The service references the catalog to know what commands to send; Fleet owns the catalog metadata, not the installer bytes.",
    techHelp:
      "Fleet-maintained apps are drawn from Homebrew casks on macOS and winget packages on Windows. The same ecosystems a developer or IT admin would use interactively. Fleet's nightly maintained_apps cron rebuilds the catalog by pulling the latest version numbers and download URLs from these upstream repos. The upside: Fleet gets a huge, free, community-maintained app library without having to curate every app itself.",
  },
  {
    from: "sw-catalog-vpp",
    to: "sw-svc-install",
    dashed: true,
    tooltip:
      "The VPP catalog is Fleet's mirror of the Apple App Store apps the admin has purchased volume licenses for. The service doesn't deliver these itself. It tells Apple to assign a license to the user and Apple pushes the install.",
    techHelp:
      "VPP (Volume Purchase Program) is Apple's bulk licensing system for the App Store. Admins buy N licenses of an app through Apple Business Manager; Fleet syncs the purchased titles via the refresh_vpp_app_versions cron. When installing, Fleet never hosts the bytes. It just tells Apple \"assign one of your licenses to this user and push the install.\" Apple handles delivery through its own CDN.",
  },
  {
    from: "sw-catalog-inhouse",
    to: "sw-svc-install",
    dashed: true,
    tooltip:
      "The in-house catalog holds private IPAs (iOS/iPadOS) and private installers that an admin uploaded but doesn't want distributed via the public App Store. Fleet owns these bytes directly. Same storage path as custom installers, but a different code path reflects the different distribution model.",
  },

  // Service → queue
  {
    from: "sw-svc-install",
    to: "sw-queue-upcoming",
    label: "write intent",
    tooltip:
      "The service writes a pending row into the right upcoming_activities table (software_install_upcoming_activities for custom + FMA, vpp_app_upcoming_activities for VPP, in_house_app_upcoming_activities for in-house). The row is the \"intent\". Delivery and status are downstream.",
  },
  {
    from: "sw-svc-uninstall",
    to: "sw-queue-upcoming",
    tooltip:
      "Uninstall writes a parallel intent row into software_install_upcoming_activities with a different operation type. Downstream, fleetd runs the uninstall script instead of the install script.",
  },
  {
    from: "sw-queue-upcoming",
    to: "sw-queue-status",
    dashed: true,
    tooltip:
      "Each upcoming_activities row has a sibling status row (host_software_installs for custom + FMA, host_vpp_software_installs for VPP, host_in_house_software_installs for in-house) that admins and end users see in the UI. The queue is internal plumbing; the status row is user-visible.",
  },

  // Queue → delivery paths (branch by type)
  {
    from: "sw-queue-upcoming",
    to: "sw-delivery-osquery",
    label: "custom + in-house + uninstall",
    tooltip:
      "For custom installers, in-house installers, and uninstalls, delivery is pull-based through fleetd's config poll. When fleetd calls POST /api/osquery/config next (every ~10s), Fleet sees a pending intent for this host and attaches it to the config response.",
  },
  {
    from: "sw-queue-upcoming",
    to: "sw-delivery-apns",
    label: "VPP / in-house IPA",
    tooltip:
      "For VPP apps and in-house IPAs, the queue is drained by the apple_mdm_apns_pusher cron (1m). Each pending install gets an APNs push that wakes the device, which then pulls the InstallApplication (VPP) or InstallEnterpriseApplication (in-house) MDM command.",
    techHelp:
      "Two distinct Apple MDM commands with very different semantics:\n\n- **InstallApplication**: for VPP apps. Device downloads from the App Store CDN using an assigned volume license.\n- **InstallEnterpriseApplication**: for private IPAs. Fleet hosts the IPA, device downloads directly from Fleet.\n\nDifferent wire format, different auth, different storage path. Fleet picks which command based on the title's type.",
  },
  {
    from: "sw-queue-upcoming",
    to: "sw-delivery-vpp-worker",
    label: "VPP",
    tooltip:
      "VPP installs also need a license-assignment step before Apple will push the app. The service enqueues an apple_software worker job that calls Apple's VPP API to assign a purchased license to the device's user. Only then does the InstallApplication MDM command actually work.",
  },

  // Delivery → device-side execution
  {
    from: "sw-delivery-osquery",
    to: "sw-device-fleetd",
    tooltip:
      "When fleetd receives install intent in its config response, it streams the installer bytes from Fleet, runs the pre-install script, runs the installer, runs the post-install script, then POSTs the overall result back.",
  },
  {
    from: "sw-delivery-apns",
    to: "sw-device-apple",
    tooltip:
      "APNs wakes the device; the device checks in to the Apple MDM endpoint, receives the queued InstallApplication / InstallEnterpriseApplication command, and begins the install.",
  },
  {
    from: "sw-delivery-vpp-worker",
    to: "sw-device-apple",
    tooltip:
      "The worker's VPP license-assign call to Apple completes; Apple then issues its own APNs push to the device telling it to download the newly-assigned app from the App Store CDN. Fleet never sees the install bytes.",
    techHelp:
      "The VPP license-assignment call is why VPP installs have an extra async step vs. custom installers. Fleet calls Apple's VPP API with \"assign license X to user Y.\" Apple validates the license is unused, binds it, and only then kicks off its own delivery pipeline. This is why a VPP install can show \"pending\" for a few seconds after the admin clicks. The device is waiting for Apple, not Fleet.",
  },

  // Device → result
  {
    from: "sw-device-fleetd",
    to: "sw-result",
    tooltip:
      "fleetd POSTs the install exit code + script outputs to /api/fleet/orbit/software_install_result. Fleet reads this and knows whether to flip the status row to installed or failed.",
  },
  {
    from: "sw-device-apple",
    to: "sw-result",
    tooltip:
      "The device sends an MDM Acknowledgement for the InstallApplication / InstallEnterpriseApplication command with the install state (Installing, Installed, Failed). Fleet parses the ACK and flips the status row accordingly.",
  },
  {
    from: "sw-result",
    to: "sw-queue-status",
    dashed: true,
    tooltip:
      "The result event is the trigger that updates the user-visible status row from pending to installed or failed. Status row changes are what admins and end users see. The result event is the moment that transition happens.",
  },
  {
    from: "sw-result",
    to: "sw-inventory",
    label: "next detail-query",
    tooltip:
      "After an install completes, fleetd's next osquery detail-query cycle picks up the new program in the local Programs/Applications table and reports it. Fleet updates host_software to reflect what the device actually has. The status row is the install's story, host_software is the truth-after-the-fact.",
    techHelp:
      "Detail queries are a predefined set of osquery queries fleetd runs automatically against each host on a schedule. Things like 'list all installed apps,' 'list all users,' 'what's the OS version.' They're the data pipeline that keeps host_software, host_users, and host_operating_system in sync with reality. The install result is Fleet saying \"we think this worked;\" host_software via detail query is osquery confirming \"yes, it's there.\"",
  },

  // Inventory → vuln pipeline
  {
    from: "sw-inventory",
    to: "sw-vuln-cron",
    tooltip:
      "The vulnerabilities cron (1h) reads host_software to get the full installed-software universe across all hosts. That's what gets matched against the CVE feeds for vuln discovery.",
  },
  {
    from: "sw-vuln-feeds",
    to: "sw-vuln-cron",
    tooltip:
      "Each cron tick downloads the latest NVD / OSV / OVAL / MSRC / Office feeds, extracts CVE records with their affected-software metadata (CPE identifiers on NVD, name-based on others), and holds them in memory for matching.",
    techHelp:
      "Fleet consumes five different vuln feeds because no single feed covers everything:\n\n- **NVD** (National Vulnerability Database): the primary source. Uses CPE identifiers.\n- **OSV** (Open Source Vulnerabilities): language ecosystems like npm, PyPI, Go modules.\n- **OVAL** (Open Vulnerability and Assessment Language): Red Hat / Oracle Linux.\n- **MSRC**: Microsoft's security advisories. Uniquely shipped as .xlsx spreadsheets.\n- **Office**: a dedicated Microsoft Office vulnerability feed.\n\nEach has its own format, update cadence, and matching semantics.",
  },
  {
    from: "sw-vuln-cron",
    to: "sw-vuln-write",
    label: "match",
    tooltip:
      "Match results get written to software_cve (one row per installed-software × CVE pair) and vulnerability_host_counts (pre-aggregated rollups the dashboard and host-count sidebar read). Rollups exist because a live GROUP BY across millions of host_software rows would be too slow.",
  },

  // Vuln + policy → feedback automations
  {
    from: "sw-vuln-write",
    to: "sw-feedback-automation",
    dashed: true,
    tooltip:
      "The automations cron (10m) reads the newly-written software_cve rows to find vulns that fired since the last run. For any vuln configured to automate (ticket creation, policy action), it enqueues the right follow-up.",
  },
  {
    from: "sw-feedback-automation",
    to: "sw-feedback-tickets",
    label: "enqueue",
    tooltip:
      "For vulns configured with an integration automation, the automations cron enqueues a Jira / Zendesk worker job. The integrations cron (1m) later drains those jobs and makes the external API calls to actually create the tickets.",
  },
  {
    from: "sw-feedback-automation",
    to: "sw-trigger-policy",
    dashed: true,
    tooltip:
      "Policies configured with an install action get re-evaluated as part of automation. If the vuln discovery flips a policy from passing to failing (e.g., the installed version is now known vulnerable), the policy's install action loops back into the install flow for every failing host.",
  },
];
