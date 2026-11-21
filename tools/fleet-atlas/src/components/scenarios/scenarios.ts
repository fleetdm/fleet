export type Actor =
  | "External"
  | "Admin"
  | "Device"
  | "APNs"
  | "FCM"
  | "Fleet HTTP"
  | "Fleet Service"
  | "Datastore"
  | "MySQL"
  | "Redis"
  | "Cron"
  | "Worker"
  | "Vuln pipeline"
  | "Google / Apple";

export type Step = {
  t: string;
  actor: Actor;
  action: string;
  detail?: string;
  file?: string;
};

export const ACTOR_COLORS: Record<Actor, string> = {
  External: "bg-zinc-200 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200",
  Admin: "bg-sky-100 text-sky-800 dark:bg-sky-900/60 dark:text-sky-200",
  Device:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200",
  APNs: "bg-pink-100 text-pink-800 dark:bg-pink-900/60 dark:text-pink-200",
  FCM: "bg-pink-100 text-pink-800 dark:bg-pink-900/60 dark:text-pink-200",
  "Fleet HTTP":
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/60 dark:text-indigo-200",
  "Fleet Service":
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/60 dark:text-indigo-200",
  Datastore:
    "bg-violet-100 text-violet-800 dark:bg-violet-900/60 dark:text-violet-200",
  MySQL:
    "bg-violet-100 text-violet-800 dark:bg-violet-900/60 dark:text-violet-200",
  Redis: "bg-rose-100 text-rose-800 dark:bg-rose-900/60 dark:text-rose-200",
  Cron: "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200",
  Worker:
    "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200",
  "Vuln pipeline":
    "bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-900/60 dark:text-fuchsia-200",
  "Google / Apple":
    "bg-zinc-200 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200",
};

export type ScenarioTag =
  | "software"
  | "auto-patching"
  | "policies"
  | "automation"
  | "mdm"
  | "apple"
  | "osquery"
  | "live queries"
  | "vulnerabilities"
  | "cron"
  | "onboarding"
  | "setup-experience"
  | "android";

export const TAG_COLORS: Record<ScenarioTag, string> = {
  software:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200",
  "auto-patching":
    "bg-teal-100 text-teal-800 dark:bg-teal-900/60 dark:text-teal-200",
  policies:
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/60 dark:text-indigo-200",
  automation:
    "bg-sky-100 text-sky-800 dark:bg-sky-900/60 dark:text-sky-200",
  mdm:
    "bg-rose-100 text-rose-800 dark:bg-rose-900/60 dark:text-rose-200",
  apple:
    "bg-pink-100 text-pink-800 dark:bg-pink-900/60 dark:text-pink-200",
  osquery:
    "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200",
  "live queries":
    "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/60 dark:text-cyan-200",
  vulnerabilities:
    "bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-900/60 dark:text-fuchsia-200",
  cron:
    "bg-violet-100 text-violet-800 dark:bg-violet-900/60 dark:text-violet-200",
  onboarding:
    "bg-lime-100 text-lime-800 dark:bg-lime-900/60 dark:text-lime-200",
  "setup-experience":
    "bg-orange-100 text-orange-800 dark:bg-orange-900/60 dark:text-orange-200",
  android:
    "bg-green-100 text-green-800 dark:bg-green-900/60 dark:text-green-200",
};

export type Scenario = {
  slug: string;
  label: string;
  summary: string;
  overview: string;
  steps: Step[];
  takeaway: string;
  tags: ScenarioTag[];
};

export const SCENARIOS: Scenario[] = [
  {
    slug: "mac-dep-enrollment",
    label: "A new Mac enrolls via DEP",
    summary:
      "Factory-fresh MacBook. ABM-assigned to Fleet. End user unboxes.",
    overview:
      "The 'zero-touch' Apple enrollment flow. Involves Apple's servers, Fleet's MDM endpoint, nanomdm's command queue, APNs, and the setup assistant.",
    steps: [
      {
        t: "T-7d",
        actor: "Admin",
        action: "Admin assigns device serial to Fleet in Apple Business Manager.",
        detail:
          "Metadata lives at Apple. Fleet syncs the assignment list via nanodep.",
      },
      {
        t: "T+0s",
        actor: "External",
        action: "User unboxes Mac, presses power.",
        detail:
          "Mac contacts Apple's activation server. Apple returns 'your MDM is at fleet.example.com'.",
      },
      {
        t: "T+5s",
        actor: "Device",
        action: "Mac POSTs to Fleet's enrollment URL.",
        detail: "Fleet runs the request through the MDM router.",
        file: "server/mdm/apple/",
      },
      {
        t: "T+5s",
        actor: "Fleet Service",
        action: "Fleet verifies the DEP challenge and generates an enrollment profile.",
        detail:
          "Fleet includes `await_device_configured: true` in the DEP profile so the Mac pauses at setup assistant until Fleet releases it. If this key is missing (hand-rolled profile, override), the Mac never waits. Regardless of what Fleet does downstream.",
        file: "server/mdm/apple/",
      },
      {
        t: "T+5s",
        actor: "MySQL",
        action: "Rows written: host_dep_assignments, nano_enrollments, host_mdm.",
      },
      {
        t: "T+6s",
        actor: "Device",
        action: "Mac installs the enrollment profile and opens an MDM channel.",
      },
      {
        t: "T+7s",
        actor: "Fleet Service",
        action:
          "Mac sends TokenUpdate checkin. Fleet's checkin handler calls EnqueueSetupExperienceItems: looks up setup_experience_software + setup_experience_scripts for the host's team.",
        detail:
          "If ANY setup item exists for the team: Fleet enqueues them AND inserts host_mdm_apple_awaiting_configuration (awaiting=true). If NONE exist: no row is written and setup experience silently skips. The Mac will proceed straight to desktop after bootstrap. This is the #1 cause of 'setup experience didn't run' incidents. Common roots: setup content configured on a different team than the host enrolled into, or configured AFTER the host checked in (not retroactive).",
        file: "server/service/apple_mdm.go",
      },
      {
        t: "T+10s",
        actor: "Cron",
        action:
          "The manage_apple_profiles cron ticks (every ~30s): sees new host, computes profiles due.",
        file: "cmd/fleet/cron.go",
      },
      {
        t: "T+2m",
        actor: "APNs",
        action: "Fleet sends an APNs push: 'wake up'.",
      },
      {
        t: "T+2m",
        actor: "Device",
        action:
          "Mac polls Fleet's MDM command endpoint. Fleet returns queued InstallProfile commands and the bootstrap package.",
      },
      {
        t: "T+3m",
        actor: "Device",
        action: "Mac installs profiles, reports acknowledgments.",
      },
      {
        t: "T+3m",
        actor: "MySQL",
        action: "host_mdm_apple_profiles rows flipped from pending to verified.",
      },
      {
        t: "T+5m",
        actor: "Device",
        action:
          "Setup assistant finishes. fleetd (osquery) is now installed via the bootstrap package. It calls POST /api/osquery/enroll with the enroll secret.",
      },
      {
        t: "T+5m",
        actor: "Fleet Service",
        action:
          "svc.EnrollAgent writes to hosts, returns a node key. Mac now has both an MDM channel and an osquery channel.",
      },
      {
        t: "T+5m",
        actor: "Fleet Service",
        action:
          "fleetd (orbit) calls POST /api/fleet/orbit/config. Fleet reads host_mdm_apple_awaiting_configuration via GetHostAwaitingConfiguration.",
        detail:
          "If the row exists (awaiting=true): Fleet returns `RunSetupExperience=true` and fleetd shows the setup window and polls for item completion. If the row is missing (because EnqueueSetupExperienceItems didn't write it at T+7s): no notification is returned, fleetd proceeds normally, and the Mac finishes to desktop without any setup experience ever running. This endpoint only READS; it doesn't write the row.",
        file: "server/service/orbit.go",
      },
      {
        t: "T+5-15m",
        actor: "Fleet Service",
        action:
          "If the host IS in setup experience: for each setup_experience_software_installer, Fleet enqueues an install (upcoming_activities + host_software_installs=pending). Scripts queue similarly. Rows flip in setup_experience_status_results as each finishes.",
      },
      {
        t: "T+15m",
        actor: "Fleet Service",
        action:
          "When every setup item reports success, Fleet calls release_device and SetHostAwaitingConfiguration(false). The Mac exits 'awaiting configuration' and reaches the desktop.",
        file: "server/mdm/apple/release_device.go",
      },
    ],
    takeaway:
      "Two parallel channels end up open: MDM (device-pulls on APNs push) and osquery (device-polls every ~10s). Each is independent. A profile install is an MDM command; a software inventory is an osquery query. For setup experience debugging: the row in host_mdm_apple_awaiting_configuration is the single source of truth. If it's missing, Fleet never considers the host to be in setup experience. And the row is only written when EnqueueSetupExperienceItems finds setup content for the host's team at MDM checkin time, not when setup content is configured later.",
    tags: ["mdm", "apple", "onboarding", "setup-experience"],
  },
  {
    slug: "android-company-owned-enrollment",
    label: "A new Android phone enrolls via company-owned QR",
    summary:
      "Admin generates a company-owned (fully managed) QR in a specific fleet. Phone scans it. Fleet decides which fleet the device lands in.",
    overview:
      "Android enrollment flows through Google's AMAPI (Android Management API) + Google Pub/Sub, not APNs. Fleet mints an enrollment token at QR-generation time with the fleet's enroll_secret embedded in AdditionalData. Google holds the token. When the device scans, Google provisions it and posts an ENROLLMENT message back to Fleet via Pub/Sub carrying that AdditionalData. Fleet reads it to decide which fleet owns the host.",
    steps: [
      {
        t: "T-5m",
        actor: "Admin",
        action:
          "Admin opens Add hosts → Android → Company-owned (fully managed) on the target fleet's page and copies the enrollment URL.",
        detail:
          "The URL carries `enroll_secret=<secret>&fully_managed=true`. The secret is the one tied to the currently-selected fleet in the UI. If the UI reads the wrong secret here (wrong fleet selected, stale state), every downstream step honors the wrong fleet. This is suspect #1 for 'device landed in wrong fleet' reports.",
      },
      {
        t: "T-5m",
        actor: "Fleet HTTP",
        action:
          "User (or integration) hits the enrollment-token endpoint carrying enroll_secret.",
        detail:
          "Decoded into enrollmentTokenRequest{EnrollSecret, IdpSessionID, FullyManaged}.",
        file: "server/mdm/android/service/service.go",
      },
      {
        t: "T-5m",
        actor: "Fleet Service",
        action:
          "CreateEnrollmentToken calls VerifyEnrollSecret to resolve the enroll_secret → fleet.",
        detail:
          "VerifyEnrollSecret enforces secret uniqueness server-side, so one secret cannot resolve to two fleets. If this returns the wrong fleet, suspect a cache or a secret collision (rare).",
        file: "server/mdm/android/service/service.go",
      },
      {
        t: "T-5m",
        actor: "Fleet Service",
        action:
          "Fleet marshals {EnrollSecret, IdpUUID} as JSON into AdditionalData and asks Google AMAPI to mint an enrollment token.",
        detail:
          "This is the crucial handoff: the enroll_secret is now stored inside Google's enrollment token under AdditionalData. Google returns a token + QR; Fleet renders the QR for the admin.",
        file: "server/mdm/android/service/service.go",
      },
      {
        t: "T+0s",
        actor: "External",
        action:
          "End user scans the QR with a factory-fresh (or wiped) Android device.",
      },
      {
        t: "T+10s",
        actor: "Google / Apple",
        action:
          "Google uses the token to provision the device: downloads policies, installs the Android Device Policy app, binds the device to Fleet's enterprise.",
      },
      {
        t: "T+30s",
        actor: "Google / Apple",
        action:
          "Google POSTs an ENROLLMENT Pub/Sub message to Fleet's /mdm/android/pubsub endpoint. The device payload includes EnrollmentTokenData. A copy of the AdditionalData string Fleet set at T-5m.",
        detail:
          "Suspect #2 for wrong-fleet landings: if the device was previously enrolled, verify Google isn't replaying an OLD enrollment's AdditionalData instead of the fresh QR's. Confirmation requires the raw Pub/Sub payload for the ENROLLMENT message and comparing its EnrollmentTokenData.EnrollSecret to the one in the fresh QR's URL.",
        file: "server/mdm/android/service/pubsub.go",
      },
      {
        t: "T+30s",
        actor: "Fleet Service",
        action:
          "The Pub/Sub handler calls enrollHost. For a never-seen device, it inserts a new host row. For a previously-enrolled device matched by hardware fingerprint, it reuses the existing host.Host.ID.",
        detail:
          "Re-enrollment reuses the host row but does NOT keep the old team_id by default. Team is reassigned from EnrollmentTokenData two steps down. If a device lands in its OLD fleet after re-enrollment, the bug is upstream in what's inside EnrollmentTokenData, not the host-matching logic.",
        file: "server/mdm/android/service/pubsub.go",
      },
      {
        t: "T+30s",
        actor: "Fleet Service",
        action:
          "resolveTeamFromEnrollmentData parses EnrollmentTokenData JSON and calls VerifyEnrollSecret on the EnrollSecret field, returning the resolved team_id.",
        detail:
          "This is the single fleet-assignment decision point for every Android enrollment. The team_id returned here is the fleet the device lands in. Any 'wrong fleet' investigation ultimately resolves to: what EnrollSecret was in EnrollmentTokenData at this call?",
        file: "server/mdm/android/service/zero_touch.go",
      },
      {
        t: "T+30s",
        actor: "Fleet Service",
        action:
          "host.TeamID is set to the resolved team_id. updateHost(fromEnroll=true) runs next.",
        file: "server/mdm/android/service/pubsub.go",
      },
      {
        t: "T+30s",
        actor: "Fleet Service",
        action:
          "AddHostsToTeam(host.TeamID, [host.Host.ID]) performs the actual team move at the Fleet host layer.",
        file: "server/mdm/android/service/pubsub.go",
      },
      {
        t: "T+30s",
        actor: "MySQL",
        action:
          "UpdateTeamIDOnAndroidDevices([uuid], team_id) syncs android_devices.team_id so the Android-side state matches the host-side state.",
        file: "server/mdm/android/service/pubsub.go",
      },
      {
        t: "T+1-5m",
        actor: "Device",
        action:
          "Google finishes provisioning on the device side. App policies push via AMAPI; status reports start flowing back via subsequent Pub/Sub messages.",
      },
    ],
    takeaway:
      "The fleet a device lands in is decided by whatever enroll_secret is inside EnrollmentTokenData when the Pub/Sub ENROLLMENT message arrives. For 'device landed in wrong fleet' debugging there are three suspects in priority order: (1) the Add-hosts URL carried the wrong fleet's enroll_secret. Frontend bug or wrong fleet selected at copy time; (2) Google replayed an older enrollment's AdditionalData for a previously-known device. Needs the raw Pub/Sub payload to confirm; (3) server-side VerifyEnrollSecret resolved the wrong fleet. Rare, since Fleet enforces secret uniqueness. Separately: zero-touch is a different path entirely. Devices registered in Google's zero-touch portal get their fleet from the zero-touch configuration, not the scanned QR, and factory-resetting doesn't remove a device from the portal. Only portal removal does.",
    tags: ["mdm", "android", "onboarding"],
  },
  {
    slug: "live-query",
    label: "Admin runs a live query against 100 hosts",
    summary:
      "Watch rows stream in from a user-authored SQL executed on 100 Macs.",
    overview:
      "The main thing to notice is that Fleet never pushes to the host. The host pulls the query on its next ~10s cycle, runs it, pushes back. Redis fans results out to the admin's browser.",
    steps: [
      {
        t: "T+0s",
        actor: "Admin",
        action:
          "Admin pastes SQL into the UI, picks a label targeting 100 Macs, clicks Run.",
      },
      {
        t: "T+0s",
        actor: "Fleet HTTP",
        action:
          "POST /api/latest/fleet/reports/{id}/run. Decodes into runLiveQueryRequest.",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "svc.NewDistributedQueryCampaign authorizes, creates a campaign row, enumerates targets.",
      },
      {
        t: "T+0s",
        actor: "MySQL",
        action:
          "Inserts into distributed_query_campaigns and distributed_query_campaign_targets.",
      },
      {
        t: "T+0s",
        actor: "Fleet HTTP",
        action:
          "UI opens WebSocket to /api/latest/fleet/results/{campaign_id}.",
      },
      {
        t: "T+0s",
        actor: "Redis",
        action:
          "Fleet subscribes to pubsub channel keyed by campaign_id.",
        file: "server/pubsub/redis_query_results.go",
      },
      {
        t: "T+10s",
        actor: "Device",
        action:
          "Mac #1's osquery hits POST /api/osquery/distributed/read. Fleet returns the campaign SQL.",
      },
      {
        t: "T+11s",
        actor: "Device",
        action:
          "Mac #1 runs the SQL locally, POSTs rows to /api/osquery/distributed/write.",
      },
      {
        t: "T+11s",
        actor: "Fleet Service",
        action:
          "svc.SubmitDistributedQueryResults publishes rows to the campaign's Redis channel.",
      },
      {
        t: "T+11s",
        actor: "Redis",
        action:
          "Pubsub delivery to all subscribers (possibly across Fleet instances).",
      },
      {
        t: "T+11s",
        actor: "Fleet HTTP",
        action:
          "WebSocket handler receives the publish, sends rows to the browser.",
      },
      {
        t: "T+12s to T+2m",
        actor: "Device",
        action:
          "Remaining 99 Macs follow the same shape as they tick through their 10s cycles.",
      },
      {
        t: "T+5m",
        actor: "Cron",
        action:
          "frequent_cleanups retires campaigns past their timeout.",
      },
    ],
    takeaway:
      "Live queries are pull-based + Redis-fanout. There's no push to the host and no polling of MySQL from the UI. The second-to-second UX relies on Redis pubsub bridging agents and admins across Fleet instances.",
    tags: ["live queries", "osquery"],
  },
  {
    slug: "profile-upload-reaches-mac",
    label: "Admin uploads a profile; it reaches a Mac",
    summary:
      "A new .mobileconfig lands on 500 Macs with no human intervention after upload.",
    overview:
      "Shows how declarative desired-state + a reconciliation cron drives the install, instead of an inline API call.",
    steps: [
      {
        t: "T+0s",
        actor: "Admin",
        action: "Admin uploads foo.mobileconfig to a fleet.",
      },
      {
        t: "T+0s",
        actor: "Fleet HTTP",
        action: "POST /api/latest/fleet/mdm/profiles. Multipart upload decoded.",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "Validates XML, computes a hash, authorizes.",
      },
      {
        t: "T+0s",
        actor: "MySQL",
        action:
          "Insert into mdm_apple_configuration_profiles. 500 pending rows go into host_mdm_apple_profiles with status=pending.",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action: "svc.NewActivity logs the upload to activity_past.",
      },
      {
        t: "T+2m",
        actor: "Cron",
        action:
          "manage_apple_profiles ticks. For each pending row, enqueues an InstallProfile command in nanomdm.",
      },
      {
        t: "T+2m",
        actor: "MySQL",
        action: "Writes to nano_commands and nano_enrollment_queue.",
      },
      {
        t: "T+3m",
        actor: "Cron",
        action: "apple_mdm_apns_pusher ticks (every minute). Sends APNs push to each targeted device.",
      },
      {
        t: "T+3m",
        actor: "APNs",
        action: "Apple delivers push to each Mac.",
      },
      {
        t: "T+3m to T+10m",
        actor: "Device",
        action:
          "Each Mac wakes, polls Fleet's MDM endpoint, receives InstallProfile command, installs, ACKs.",
      },
      {
        t: "T+10m",
        actor: "MySQL",
        action: "host_mdm_apple_profiles rows flip pending → verifying → verified.",
      },
      {
        t: "T+10m",
        actor: "Fleet Service",
        action:
          "On ACK, Fleet writes an activity row ('profile installed on host X') and clears pending state.",
      },
    ],
    takeaway:
      "The admin's upload is NOT what installs the profile. The upload records desired state. The install is driven by a reconciliation cron plus APNs push plus each device's pull. Any step can be inspected independently by querying its table.",
    tags: ["mdm", "apple", "cron"],
  },
  {
    slug: "cve-match",
    label: "A new CVE is discovered and matches installed software",
    summary:
      "CVE-2026-XXXX drops at 09:00 UTC. By 10:00 Fleet shows it on affected hosts.",
    overview:
      "The vuln pipeline is a cron that pulls feeds, matches CPEs, and writes results. No real-time anything.",
    steps: [
      {
        t: "T+0",
        actor: "External",
        action:
          "NVD publishes CVE-2026-XXXX affecting foo-lib 1.2.3.",
      },
      {
        t: "T+until next tick",
        actor: "Cron",
        action:
          "The vulnerabilities cron ticks hourly. On its next tick, it downloads the latest NVD feed delta.",
      },
      {
        t: "T+1h",
        actor: "Vuln pipeline",
        action:
          "Pipeline parses the feed. New CVE row cached. CPE 'foo-lib:1.2.3' extracted.",
      },
      {
        t: "T+1h",
        actor: "MySQL",
        action: "Read host_software rows where software CPE matches.",
      },
      {
        t: "T+1h",
        actor: "Vuln pipeline",
        action:
          "Finds 72 installations across 50 hosts. Writes 72 rows to software_cve.",
      },
      {
        t: "T+1h",
        actor: "MySQL",
        action: "vulnerability_host_counts rollup updated.",
      },
      {
        t: "T+1h",
        actor: "Cron",
        action:
          "automations cron ticks. Sees new high-severity CVE matching a policy. Enqueues a Jira job.",
      },
      {
        t: "T+1h+1m",
        actor: "Worker",
        action:
          "integrations cron drains the queue. Worker makes a Jira API call. Ticket created.",
      },
      {
        t: "T+1h+2m",
        actor: "Admin",
        action:
          "Admin's dashboard refresh shows a new vuln on affected hosts. Ticket in Jira has the host list.",
      },
    ],
    takeaway:
      "Vuln surfacing is cron-cadenced, not event-driven. There's a predictable 'blind window' of up to 1 hour between upstream publish and Fleet surfacing. Policy automations layer on top: they turn the pipeline's writes into tickets.",
    tags: ["vulnerabilities", "software", "automation", "cron"],
  },
  {
    slug: "install-from-host-library",
    label: "Admin clicks Install from a host's software library",
    summary:
      "One admin, one host, one app. The 'Install' button on Host details → Software.",
    overview:
      "This is the most direct software install path in Fleet. It writes an intent row, waits for the host to poll, delivers the installer, runs it, and records the result. No crons needed to kick it off; the host's regular 10s poll is what drives delivery.",
    steps: [
      {
        t: "T+0s",
        actor: "Admin",
        action:
          "Admin opens Host details for host H, Software tab, clicks Install next to app 'Foo 1.2.3'.",
      },
      {
        t: "T+0s",
        actor: "Fleet HTTP",
        action:
          "POST /api/latest/fleet/hosts/{host_id}/software/{software_title_id}/install.",
        detail:
          "Decoded into a struct with host_id and software_title_id; no body payload.",
        file: "server/service/handler.go",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "svc.InstallSoftwareTitle authorizes (write on this host), confirms the host + title are allowed to pair.",
        file: "server/service/software_installers.go",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "Service decides whether this title is backed by a custom installer, VPP, FMA, or in-house app. The write path diverges per type.",
        detail:
          "This scenario follows the custom-installer path (a .pkg/.msi uploaded by the admin).",
      },
      {
        t: "T+0s",
        actor: "MySQL",
        action:
          "Inserts upcoming_activities + software_install_upcoming_activities rows. host_software_installs row created with status=pending.",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "Writes an activity_past row ('user U queued install of Foo on host H').",
      },
      {
        t: "T+0s",
        actor: "Fleet HTTP",
        action: "Returns 202 Accepted with an install_uuid.",
      },
      {
        t: "T+<=10s",
        actor: "Device",
        action:
          "Host H's next osquery config poll (POST /api/osquery/config) picks up the pending install.",
        detail:
          "fleetd includes a 'notifications' / 'config' section that signals pending work. On seeing an install, fleetd fetches the installer bytes.",
      },
      {
        t: "T+10s",
        actor: "Device",
        action:
          "fleetd calls GET /api/fleet/orbit/software_install/package/{install_uuid}. Fleet streams the installer bytes from the file store.",
        file: "server/service/software_installers.go",
      },
      {
        t: "T+10s to T+Xm",
        actor: "Device",
        action:
          "fleetd runs the admin-authored pre-install / install / post-install scripts locally against the downloaded bytes.",
        detail:
          "Scripts are stored on Fleet with the installer; fleetd pulls them with the package.",
      },
      {
        t: "T+Xm",
        actor: "Device",
        action:
          "fleetd POSTs the result to /api/fleet/orbit/software_install_result (exit code, stdout/stderr tails).",
      },
      {
        t: "T+Xm",
        actor: "Fleet Service",
        action:
          "svc.SaveHostSoftwareInstallResult records the result. Status flips pending → installed or failed.",
      },
      {
        t: "T+Xm",
        actor: "MySQL",
        action:
          "host_software_installs row updated. upcoming_activities row deleted. activity_past row written ('install succeeded' / 'install failed').",
      },
      {
        t: "T+Xm+10s",
        actor: "Device",
        action:
          "Next osquery detail-queries run refreshes host_software (what's actually installed on disk). Fleet's inventory now reflects reality.",
      },
    ],
    takeaway:
      "The admin's click is not what installs anything. It writes an intent (upcoming_activities + host_software_installs=pending) and relies on the host's regular 10s osquery poll to pick up the work. Everything else (download, scripts, result recording) is driven by the host. If an install 'does nothing,' the first thing to check is whether that host is actually checking in: if it is not, no amount of re-queuing will help.",
    tags: ["software", "osquery"],
  },
  {
    slug: "policy-driven-auto-install",
    label: "A failing policy auto-installs software",
    summary:
      "'Chrome < 120' policy fails on 42 Macs. Auto-install setting is on. Chrome 120 rolls out.",
    overview:
      "The self-healing / auto-remediation path. Combines the policy evaluation cadence, the automations cron that reads failing policies, and the same install mechanics as the manual button.",
    steps: [
      {
        t: "T-7d",
        actor: "Admin",
        action:
          "Admin creates policy 'Chrome >= 120' and associates it with the custom installer software_title 'Google Chrome 120.1'. Auto-install toggled on.",
      },
      {
        t: "T+0s",
        actor: "Device",
        action:
          "Hosts run the policy's SQL as part of their scheduled queries. 42 Macs return no rows (meaning Chrome is below 120). They report back to Fleet via /api/osquery/log.",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "svc.RecordPolicyResults updates policy_membership for each host (passing/failing) and refreshes policy_stats.",
      },
      {
        t: "T+10m",
        actor: "Cron",
        action:
          "automations cron ticks (every 10m). Reads policy_membership for policies with automations configured; finds 42 newly-failing hosts for the Chrome policy.",
        file: "cmd/fleet/cron.go",
      },
      {
        t: "T+10m",
        actor: "Fleet Service",
        action:
          "For each failing host, svc.InstallSoftwareTitle is called (same method as the manual button).",
        detail:
          "host_policy_automation_run_counts is used to dedup: if the same policy already triggered an install for this host, skip.",
      },
      {
        t: "T+10m",
        actor: "MySQL",
        action:
          "42 upcoming_activities rows written. 42 host_software_installs rows with status=pending.",
      },
      {
        t: "T+10m to T+Xm",
        actor: "Device",
        action:
          "Each affected host's next osquery config poll picks up its install (same mechanics as manual install scenario).",
      },
      {
        t: "T+Xm",
        actor: "Device",
        action:
          "fleetd downloads bytes, runs scripts, reports result.",
      },
      {
        t: "T+Xm",
        actor: "MySQL",
        action: "Status flips installed; activity_past rows written per host.",
      },
      {
        t: "T+Xm+1 cycle",
        actor: "Device",
        action:
          "On the next scheduled policy evaluation, the Chrome policy now passes on each remediated host. policy_membership flips to passing.",
      },
      {
        t: "T+Xm+1 cycle",
        actor: "Fleet Service",
        action:
          "Automations cron no longer sees the host as failing. No duplicate install enqueued.",
      },
    ],
    takeaway:
      "Automation is just a cron that calls the same service method the Install button calls. The dedup lives in host_policy_automation_run_counts; without it, every automations tick would re-enqueue an install while the host was still mid-install. If you see 'why is Fleet installing the same thing over and over,' this counter is the first place to look.",
    tags: ["software", "policies", "automation", "cron"],
  },
  {
    slug: "fma-auto-update",
    label: "A Fleet-maintained app auto-updates",
    summary:
      "Homebrew publishes Firefox 130. Fleet-maintained Firefox title is bumped; opted-in fleets roll out the update.",
    overview:
      "The auto-patching flow. Two crons chain together: nightly catalog rebuild from upstream (Homebrew for macOS, winget for Windows), then an hourly roll-out that enqueues installs on hosts behind on auto-update-enabled FMAs.",
    steps: [
      {
        t: "T-1d",
        actor: "External",
        action:
          "Homebrew publishes the Firefox 130 cask. The cask URL points at the new .dmg; the installer hash is updated.",
      },
      {
        t: "T+0s (nightly)",
        actor: "Cron",
        action:
          "maintained_apps cron ticks (every 24h). Walks Fleet's FMA source list in ee/maintained-apps.",
        file: "ee/server/maintainedapps/",
      },
      {
        t: "T+0s",
        actor: "External",
        action:
          "Cron fetches each FMA input (Homebrew cask for macOS, winget manifest for Windows). Sees Firefox cask now says version 130.",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "FMA validator runs: checks the manifest, downloads the installer, verifies hash + install/uninstall script expectations.",
      },
      {
        t: "T+0s",
        actor: "MySQL",
        action:
          "fleet_maintained_apps + software_titles rows updated to version 130. Previous row (version 129) kept for hosts still showing it as installed.",
      },
      {
        t: "T+1h (hourly)",
        actor: "Cron",
        action:
          "maintained_apps_auto_update cron ticks. For each FMA with auto-update enabled, finds the latest version in software_titles and the fleets that opted in.",
      },
      {
        t: "T+1h",
        actor: "Fleet Service",
        action:
          "For each opted-in fleet, selects hosts whose host_software reports Firefox at a lower version than 130.",
      },
      {
        t: "T+1h",
        actor: "Fleet Service",
        action:
          "For each candidate host, calls svc.InstallSoftwareTitle for the Firefox 130 title. Same code path as the Install button.",
      },
      {
        t: "T+1h",
        actor: "MySQL",
        action:
          "Pending install rows written. Dedup logic skips hosts with a Firefox install already in flight.",
      },
      {
        t: "T+1h to T+Xm",
        actor: "Device",
        action:
          "Hosts pick up installs on their next poll; download, run scripts, report (same mechanics as other install scenarios).",
      },
      {
        t: "T+Xm",
        actor: "Device",
        action:
          "On the next osquery software detail query, host_software now reports Firefox 130 installed.",
      },
      {
        t: "T+next hour",
        actor: "Cron",
        action:
          "maintained_apps_auto_update ticks again. Hosts updated to 130 no longer appear in the candidate set, so no re-enqueue.",
      },
    ],
    takeaway:
      "Auto-patching is cascading crons on top of the same install mechanics. The 24h cron keeps Fleet's view of 'what version is current' aligned with upstream catalogs. The 1h cron turns 'current version' into 'install for each stale host.' All the actual installation is just the normal install scenario repeated per host. Debugging auto-patch gaps usually comes down to: did the nightly cron update software_titles? did the hourly cron see this fleet as opted-in? did the host-software view show this host as behind?",
    tags: ["software", "auto-patching", "automation", "cron"],
  },
  {
    slug: "vpp-install",
    label: "Admin installs a VPP App Store app",
    summary:
      "A paid App Store app assigned via Apple Business Manager. Fleet doesn't deliver bytes; Apple does.",
    overview:
      "VPP (Volume Purchase Program) is Apple's way of distributing App Store apps to managed devices. Fleet never touches the app binary. All Fleet does is tell Apple 'give this license to this device,' and Apple pushes the app.",
    steps: [
      {
        t: "T-7d",
        actor: "Admin",
        action:
          "Admin links Fleet to Apple Business Manager, uploads the VPP .vpptoken; row in vpp_tokens.",
      },
      {
        t: "T-1d",
        actor: "External",
        action:
          "Fleet syncs the VPP app catalog from Apple (refresh_vpp_app_versions cron). Rows in vpp_apps + vpp_apps_teams for the fleets this token is assigned to.",
        file: "cmd/fleet/cron.go",
      },
      {
        t: "T+0s",
        actor: "Admin",
        action:
          "Admin opens Host details → Software, picks the VPP app, clicks Install.",
      },
      {
        t: "T+0s",
        actor: "Fleet HTTP",
        action:
          "POST /api/latest/fleet/hosts/{host_id}/software/{software_title_id}/install.",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "Service recognizes this title is VPP-backed. Different branch: enqueue a VPP assignment, not a software_install.",
        file: "server/service/software_installers.go",
      },
      {
        t: "T+0s",
        actor: "MySQL",
        action:
          "Inserts upcoming_activities row + vpp_app_upcoming_activities + host_vpp_software_installs with status=pending.",
      },
      {
        t: "T+1m",
        actor: "Worker",
        action:
          "The apple_software worker job picks up and calls Apple's VPP API: 'assign this app license to this device identifier.'",
        file: "server/worker/vpp_verification.go",
      },
      {
        t: "T+1m",
        actor: "Google / Apple",
        action:
          "Apple accepts the assignment. If needed, Apple sends an InstallApplication MDM command to the device via APNs.",
      },
      {
        t: "T+1m to T+Xm",
        actor: "Device",
        action:
          "Mac/iPhone wakes on APNs, downloads the app from Apple's CDN, installs it, ACKs via MDM.",
      },
      {
        t: "T+Xm",
        actor: "Fleet Service",
        action:
          "On the MDM ACK, Fleet updates host_vpp_software_installs to installed.",
      },
      {
        t: "T+Xm+10s",
        actor: "Device",
        action:
          "Next osquery detail-queries refresh host_software. Fleet's software inventory now shows the app.",
      },
    ],
    takeaway:
      "VPP installs are an API call to Apple, not a byte transfer from Fleet. If a VPP install hangs, check: is the VPP token still valid? Is the app actually assigned to the user's Apple ID? Does the device have the VPP user signed in? The failure modes are entirely in the Fleet ↔ Apple API relationship, not in Fleet's own delivery path.",
    tags: ["software", "apple", "mdm"],
  },
  {
    slug: "self-service-install",
    label: "End user installs software from My Device",
    summary:
      "Employee opens My Device page, picks an app from the self-service library, clicks Install.",
    overview:
      "Fleet's self-service flow is a parallel install path with a different auth model: the end user has a device token, not an admin session. Only software titles explicitly marked as self-service appear.",
    steps: [
      {
        t: "T-30d",
        actor: "Admin",
        action:
          "Admin uploads a custom installer and ticks 'available in self-service' on the title. host_software_installers.self_service = true.",
      },
      {
        t: "T+0s",
        actor: "External",
        action:
          "End user opens their My Device page URL (unique per host, contains the device token).",
      },
      {
        t: "T+0s",
        actor: "Fleet HTTP",
        action:
          "GET /api/latest/fleet/device/{token}/software: self-service-marked titles for this host's fleet.",
        file: "server/service/handler.go",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "Looks up the host by device token (host_device_auth join), filters software to self-service-only.",
      },
      {
        t: "T+10s",
        actor: "External",
        action:
          "User clicks Install next to 'Chrome Stable'.",
      },
      {
        t: "T+10s",
        actor: "Fleet HTTP",
        action:
          "POST /api/latest/fleet/device/{token}/software/install/{software_title_id}, rate-limited (errorLimiter middleware).",
      },
      {
        t: "T+10s",
        actor: "Fleet Service",
        action:
          "svc.SelfServiceInstallSoftwareTitle: re-verifies self-service flag, then shares the install path with the admin button.",
        file: "server/service/software_installers.go",
      },
      {
        t: "T+10s",
        actor: "MySQL",
        action:
          "upcoming_activities + software_install_upcoming_activities + host_software_installs=pending. activity_past row attributes the action to the end user, not an admin.",
      },
      {
        t: "T+<=10s",
        actor: "Device",
        action:
          "fleetd picks up the pending install on next osquery config poll (identical to admin-triggered install from here on).",
      },
      {
        t: "T+Xm",
        actor: "Device",
        action:
          "Install runs, result posted. UI's My Device page polls for status and shows success/failure.",
      },
    ],
    takeaway:
      "Self-service shares 90% of its code path with the admin install button. The differences are at the edges: device-token auth instead of a session, a strict allowlist (titles must be marked self-service), rate limits on the device endpoint, and the activity log records the end user as the actor. If self-service shows no apps, the first thing to check is whether the title has self_service=true on this fleet.",
    tags: ["software", "osquery"],
  },
  {
    slug: "setup-experience-install",
    label: "Software auto-installs during Mac setup assistant",
    summary:
      "First-boot DEP flow where Fleet pre-installs apps before the end user ever reaches the desktop.",
    overview:
      "'Setup experience' is the window between MDM enrollment and the user logging in. Fleet uses it to install bootstrap packages, run setup scripts, and queue initial software so new Macs arrive 'ready to work.' This is the auto-patching team's favorite onboarding story.",
    steps: [
      {
        t: "T-7d",
        actor: "Admin",
        action:
          "Admin configures setup experience for the fleet: pick bootstrap package, set a few titles as 'required at setup', add a post-setup script.",
      },
      {
        t: "T-7d",
        actor: "MySQL",
        action:
          "setup_experience_software_installers + setup_experience_scripts rows written per fleet.",
      },
      {
        t: "T+0s",
        actor: "External",
        action:
          "User unboxes a DEP-assigned Mac, presses power. Setup assistant starts.",
      },
      {
        t: "T+30s",
        actor: "Device",
        action:
          "After MDM enrollment completes, Mac enters 'awaiting configuration' phase. host_mdm_apple_awaiting_configuration row written.",
      },
      {
        t: "T+30s",
        actor: "Fleet Service",
        action:
          "Fleet pushes the bootstrap package via MDM InstallEnterpriseApplication command. fleetd gets installed as part of the bootstrap package.",
      },
      {
        t: "T+2m",
        actor: "Device",
        action:
          "Bootstrap package installs; fleetd starts; Mac enters an inner state where setup assistant blocks on 'awaiting configuration'.",
      },
      {
        t: "T+2m",
        actor: "Fleet Service",
        action:
          "For each setup_experience_software_installer, Fleet enqueues a software install using the normal mechanics (upcoming_activities + host_software_installs=pending).",
      },
      {
        t: "T+2m to T+Xm",
        actor: "Device",
        action:
          "fleetd now running: picks up each pending install, downloads bytes, runs install scripts.",
      },
      {
        t: "T+Xm",
        actor: "Fleet Service",
        action:
          "As each install finishes, write setup_experience_status_results. Fleet watches the completion counter.",
      },
      {
        t: "T+Xm",
        actor: "Fleet Service",
        action:
          "When all required titles installed and all setup scripts finished, Fleet releases the device from 'awaiting configuration' via the Apple DDM activation.",
      },
      {
        t: "T+Xm+1s",
        actor: "Device",
        action:
          "Mac finishes setup assistant and shows the login screen. Required software is already present.",
      },
    ],
    takeaway:
      "Setup experience is three normal mechanisms stacked: MDM bootstrap, standard software install, and DDM-based release. The hard part is coordination (the device is literally paused waiting). If setup experience hangs, the stuck step is almost always one specific installer or script that silently exited non-zero; check setup_experience_status_results for the row that never flipped to success.",
    tags: ["software", "apple", "mdm", "onboarding"],
  },
  {
    slug: "gitops-software-install",
    label: "GitOps YAML declares software; Fleet reconciles",
    summary:
      "Admin commits a software list to YAML. fleetctl apply makes it the desired state. Hosts converge.",
    overview:
      "The imperative Install button and the declarative GitOps apply end up touching the same tables. The difference is that GitOps declares desired state and Fleet works out the diff; the button declares a single mutation.",
    steps: [
      {
        t: "T+0s",
        actor: "Admin",
        action:
          "Admin edits a Fleet GitOps YAML file: adds Chrome 120 to a fleet's software list, with 'install on every host' true. Commits + pushes.",
      },
      {
        t: "T+0s",
        actor: "External",
        action:
          "CI runs fleetctl gitops. fleetctl parses YAML, authenticates to the Fleet API.",
      },
      {
        t: "T+5s",
        actor: "Fleet HTTP",
        action:
          "POST /api/latest/fleet/software/batch. The GitOps-style batch endpoint that accepts the full desired list for a fleet.",
      },
      {
        t: "T+5s",
        actor: "Fleet Service",
        action:
          "Service compares desired list vs current rows in software_installers for the fleet. Computes add/remove/update.",
      },
      {
        t: "T+5s",
        actor: "MySQL",
        action:
          "software_installers inserted / updated / soft-deleted. Scoping rows (software_installer_labels, software_installer_software_categories) reconciled.",
      },
      {
        t: "T+5s",
        actor: "Fleet Service",
        action:
          "For 'install on every host' titles, Fleet enqueues installs for each matching host that doesn't already have it.",
      },
      {
        t: "T+5s",
        actor: "MySQL",
        action:
          "upcoming_activities + host_software_installs rows written for every candidate host at once.",
      },
      {
        t: "T+5s",
        actor: "Fleet Service",
        action:
          "activity_past rows written attributing the batch to the GitOps API token.",
      },
      {
        t: "T+<=10s per host",
        actor: "Device",
        action:
          "Each host picks up its install on next poll (same mechanics as the manual install scenario).",
      },
      {
        t: "T+Xm",
        actor: "Device",
        action:
          "Each host completes its install, reports result. host_software_installs flips to installed.",
      },
      {
        t: "T+next GitOps run",
        actor: "External",
        action:
          "On the next CI run, fleetctl computes an empty diff for software (desired = actual). No-op.",
      },
    ],
    takeaway:
      "GitOps is just a batch endpoint that computes a diff and reuses the same install mechanics. The install itself is identical to the Install button. Where GitOps diverges: validation errors are returned synchronously to CI (so a bad YAML fails the pipeline), not surfaced in the UI; and desired-state reconciliation can REMOVE software too, which the button never does.",
    tags: ["software", "automation"],
  },
  {
    slug: "hourly-cleanup",
    label: "Hourly cleanup cron fires",
    summary:
      "What actually happens inside cleanups_then_aggregation at the top of each hour.",
    overview:
      "A good one to understand because this is where many unrelated subsystems do their housekeeping. Also demonstrates distributed-lock behavior with multiple Fleet instances.",
    steps: [
      {
        t: "T+0s",
        actor: "Cron",
        action:
          "All Fleet instances' schedulers hit the next cleanups_then_aggregation tick simultaneously.",
      },
      {
        t: "T+0s",
        actor: "Redis",
        action:
          "Each instance tries SETNX on 'cleanups_then_aggregation:lock'. One wins (atomically).",
        file: "server/service/schedule/schedule.go",
      },
      {
        t: "T+0s",
        actor: "Fleet Service",
        action:
          "The winning instance runs the job body. The losers go back to sleep until the next tick.",
      },
      {
        t: "T+0s",
        actor: "Datastore",
        action:
          "Deletes expired sessions (DELETE FROM sessions WHERE expires_at < NOW()).",
      },
      {
        t: "T+0.1s",
        actor: "Datastore",
        action:
          "Deletes retired distributed_query_campaign rows past their retention window.",
      },
      {
        t: "T+0.3s",
        actor: "Datastore",
        action:
          "Deletes stale carve blocks from carve_blocks and marks parent carves complete.",
      },
      {
        t: "T+1s",
        actor: "Datastore",
        action:
          "Recomputes software_host_counts and vulnerability_host_counts rollups.",
      },
      {
        t: "T+1.5s",
        actor: "Datastore",
        action:
          "Recomputes host_issues (failing policies + critical vulns per host). Powers the dashboard.",
      },
      {
        t: "T+2s",
        actor: "Fleet Service",
        action:
          "Writes result into cron_stats (name, started_at, completed_at, status).",
      },
      {
        t: "T+2s",
        actor: "Redis",
        action:
          "Lock expires naturally. The winning instance moves on.",
      },
    ],
    takeaway:
      "The lock-first pattern is the key idiom. Any Fleet cron is just: 'acquire lock, do thing, write cron_stats, done.' The 'cleanup' category is a grab-bag of housekeeping across subsystems; look here when something seems to 'eventually' disappear.",
    tags: ["cron"],
  },
];

export function getScenario(slug: string): Scenario | undefined {
  return SCENARIOS.find((s) => s.slug === slug);
}
