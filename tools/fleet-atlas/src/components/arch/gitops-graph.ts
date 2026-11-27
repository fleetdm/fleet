import type { ArchEdge, ArchNode } from "./graph-data";

const COL = 220;
const ROW = 150;

export const GITOPS_NODES: ArchNode[] = [
  // Row 0. Author
  {
    id: "git-admin",
    label: "Admin edits YAML",
    group: "external",
    blurb:
      "An admin writes or edits Fleet's YAML config in their editor. The repo is the source of truth; the UI shows current state but isn't where changes originate.",
    repoPaths: [],
    x: 0,
    y: 0,
  },
  {
    id: "git-repo",
    label: "Git repository",
    group: "external",
    blurb:
      "The Fleet GitOps repo: global config, per-fleet config, and resource files (policies, software, profiles, scripts).",
    repoPaths: [],
    x: COL,
    y: 0,
  },

  // Row 1. Apply
  {
    id: "git-fleetctl-apply",
    label: "fleetctl gitops apply",
    group: "external",
    blurb:
      "The entry command. Can be run by a CI job (common) or locally by an admin. Reads YAML, computes a plan, and POSTs batch requests to Fleet's API.",
    repoPaths: [{ path: "cmd/fleetctl/" }],
    x: COL * 0.5,
    y: ROW,
  },
  {
    id: "git-dryrun",
    label: "fleetctl gitops --dry-run",
    group: "external",
    blurb:
      "Preview-only mode. Shows the plan Fleet would apply without modifying any state. The standard pre-merge check for GitOps PRs.",
    repoPaths: [{ path: "cmd/fleetctl/" }],
    x: COL * 2,
    y: ROW,
  },

  // Row 2. YAML sources parsed
  {
    id: "git-global",
    label: "default.yml / global config",
    group: "service",
    blurb:
      "Global org settings: integrations, agent options, licensing, secret variables scope, SMTP, SSO.",
    repoPaths: [],
    x: 0,
    y: ROW * 2,
  },
  {
    id: "git-fleets",
    label: "fleets/*.yml (per-fleet)",
    group: "service",
    premium: true,
    blurb:
      "Per-fleet (team) config: agent options overrides, enroll secrets, feature flags.",
    repoPaths: [],
    x: COL,
    y: ROW * 2,
  },
  {
    id: "git-resources",
    label: "policies/ software/ profiles/ scripts/ labels/",
    group: "service",
    blurb:
      "Resource files. Referenced from fleet YAML; one file per policy, installer, profile, etc.",
    repoPaths: [],
    x: COL * 2,
    y: ROW * 2,
  },

  // Row 3. Batch API endpoints
  {
    id: "git-api-software",
    label: "POST /software/batch",
    group: "ingress",
    premium: true,
    blurb:
      "The declarative software endpoint. Accepts the full desired list for a fleet. Can add, update, OR remove.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    x: 0,
    y: ROW * 3,
  },
  {
    id: "git-api-profiles",
    label: "POST /mdm/profiles/batch",
    group: "ingress",
    blurb:
      "Apple + Windows + Android profiles declared as a single batch per fleet.",
    repoPaths: [{ path: "server/service/mdm.go" }],
    x: COL * 0.9,
    y: ROW * 3,
  },
  {
    id: "git-api-policies",
    label: "POST /policies/batch",
    group: "ingress",
    blurb: "Full desired policy set for a fleet, including automations.",
    repoPaths: [{ path: "server/service/policies.go" }],
    x: COL * 1.8,
    y: ROW * 3,
  },
  {
    id: "git-api-scripts",
    label: "POST /scripts/batch",
    group: "ingress",
    premium: true,
    blurb: "Fleet-authored scripts available for admin or self-service run.",
    repoPaths: [{ path: "server/service/scripts.go" }],
    x: COL * 2.7,
    y: ROW * 3,
  },
  {
    id: "git-api-labels",
    label: "POST /labels/batch",
    group: "ingress",
    premium: true,
    blurb: "SQL-driven dynamic label definitions.",
    repoPaths: [{ path: "server/service/labels.go" }],
    x: COL * 3.6,
    y: ROW * 3,
  },

  // Row 4. Service reconcilers
  {
    id: "git-svc",
    label: "Service reconcilers (BatchApply*)",
    group: "service",
    blurb:
      "Each batch endpoint calls a service method named BatchApplySomething. The method takes the full desired list and routes it through the diff + persistence steps.",
    repoPaths: [{ path: "server/service/" }],
    x: COL * 1.5,
    y: ROW * 4,
  },

  // Row 5. Diff
  {
    id: "git-diff",
    label: "Diff: desired vs current",
    group: "service",
    blurb:
      "The core GitOps mechanism. Fleet compares the submitted desired list against what's in MySQL right now and classifies each entry as add / update / remove.",
    repoPaths: [{ path: "server/service/" }],
    x: COL * 1.5,
    y: ROW * 5,
  },

  // Row 6. Writes + activity
  {
    id: "git-mysql",
    label: "MySQL writes (resource tables)",
    group: "store",
    blurb:
      "Inserts, updates, and soft-deletes land in the per-resource tables: software_installers, mdm_apple_configuration_profiles, policies, scripts, labels, and friends.",
    repoPaths: [{ path: "server/datastore/mysql/" }],
    x: 0,
    y: ROW * 6,
  },
  {
    id: "git-activity",
    label: "activity_past entries",
    group: "store",
    blurb:
      "Each applied change writes an activity row. The GitOps API token is recorded as the actor so audit trails attribute changes to automation, not a human.",
    repoPaths: [{ path: "server/datastore/mysql/activities.go" }],
    x: COL * 3.5,
    y: ROW * 6,
  },

  // Row 7. Side effects
  {
    id: "git-crons",
    label: "Reconciler crons wake up",
    group: "background",
    blurb:
      "The mdm_apple_profile_manager, mdm_windows_profile_manager, mdm_android_profile_manager, and maintained_apps_auto_update crons pick up the new desired state on their next tick.",
    repoPaths: [{ path: "server/service/schedule/" }],
    x: 0,
    y: ROW * 7,
  },
  {
    id: "git-side-install",
    label: "Install enqueues (install_everywhere)",
    group: "background",
    premium: true,
    blurb:
      "Software titles declared with install_everywhere trigger immediate install enqueues for every matching host, same as the admin Install button.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    x: COL * 1.5,
    y: ROW * 7,
  },
  {
    id: "git-side-remove",
    label: "Destructive removal",
    group: "background",
    blurb:
      "Items removed from YAML get soft-deleted in Fleet AND side-effects fire: profiles get removed from hosts, installers become unavailable in self-service. GitOps is destructive by design.",
    repoPaths: [],
    x: COL * 3,
    y: ROW * 7,
  },
];

export const GITOPS_EDGES: ArchEdge[] = [
  {
    from: "git-admin",
    to: "git-repo",
    tooltip:
      "Admin commits YAML changes to the git repo. Adds a software title, edits a policy's SQL, changes a profile. The commit is the source of truth; nothing is applied until fleetctl runs.",
  },
  {
    from: "git-repo",
    to: "git-fleetctl-apply",
    tooltip:
      "On merge (or on-demand), a CI job runs `fleetctl gitops` against the committed YAML. The repo defines desired state; fleetctl is the delivery mechanism.",
  },
  {
    from: "git-admin",
    to: "git-fleetctl-apply",
    dashed: true,
    tooltip:
      "Admins can also run `fleetctl gitops` locally against the repo for faster iteration (common pattern: run with --dry-run locally, inspect, commit, let CI run the real apply).",
  },
  {
    from: "git-fleetctl-apply",
    to: "git-dryrun",
    dashed: true,
    tooltip:
      "`--dry-run` runs the same plan computation and prints the diff Fleet would apply, but skips the POST to Fleet's API. Used as the standard PR check so reviewers can see what a merge would change before anything hits a live Fleet.",
  },

  {
    from: "git-fleetctl-apply",
    to: "git-global",
    label: "parse",
    tooltip:
      "fleetctl parses the global YAML file into Fleet config settings (SSO, enroll secrets, org name, feature flags). Global config applies to the whole Fleet instance, not any specific team.",
  },
  {
    from: "git-fleetctl-apply",
    to: "git-fleets",
    label: "parse",
    tooltip:
      "fleetctl parses per-fleet (per-team) YAML files, one per fleet. Each file declares that fleet's software titles, profiles, policies, scripts, labels, and controls.",
  },
  {
    from: "git-fleetctl-apply",
    to: "git-resources",
    label: "parse",
    tooltip:
      "fleetctl parses shared resource files. Reusable titles, profiles, scripts, labels that per-fleet files can reference by path. Resources centralize what would otherwise be duplicated across fleets.",
  },

  {
    from: "git-resources",
    to: "git-api-software",
    dashed: true,
    tooltip:
      "Resource files holding software definitions get collected and sent to the batch software API. One of several API batches fleetctl composes from the parsed YAML.",
  },
  {
    from: "git-resources",
    to: "git-api-profiles",
    dashed: true,
    tooltip:
      "Shared .mobileconfig / SyncML / Android JSON resources flow into the batch profiles API.",
  },
  {
    from: "git-resources",
    to: "git-api-policies",
    dashed: true,
    tooltip:
      "Shared policy definitions flow into the batch policies API.",
  },
  {
    from: "git-resources",
    to: "git-api-scripts",
    dashed: true,
    tooltip:
      "Shared script resources (install/uninstall/remediation scripts) flow into the batch scripts API.",
  },
  {
    from: "git-resources",
    to: "git-api-labels",
    dashed: true,
    tooltip:
      "Shared label definitions (SQL-based host-scoping queries) flow into the batch labels API.",
  },
  {
    from: "git-fleets",
    to: "git-api-software",
    label: "POST batch",
    tooltip:
      "fleetctl POSTs the full per-fleet software list to the batch software API. The server computes which titles are new, modified, or removed and applies the diff. Fleetctl itself doesn't track Fleet's current state.",
  },
  {
    from: "git-fleets",
    to: "git-api-profiles",
    label: "POST batch",
    tooltip:
      "The per-fleet profile set is sent as one batch to the batch profiles API, which figures out the add/update/delete diff and applies it in one call.",
  },
  {
    from: "git-fleets",
    to: "git-api-policies",
    label: "POST batch",
    tooltip:
      "Same batch pattern: per-fleet policy set POSTed in one request; server reconciles against the fleet's existing policies.",
  },
  {
    from: "git-fleets",
    to: "git-api-scripts",
    label: "POST batch",
    tooltip:
      "Scripts attached to a fleet POSTed as one batch.",
  },
  {
    from: "git-fleets",
    to: "git-api-labels",
    label: "POST batch",
    tooltip:
      "Labels batch sent per-fleet. Labels declared at fleet scope only apply to that fleet's hosts; global labels have their own batch path.",
  },

  {
    from: "git-api-software",
    to: "git-svc",
    tooltip:
      "The software batch API lands in the service layer, which validates each title against the configured platforms and catalog type (custom, FMA, VPP) before computing what to persist.",
  },
  {
    from: "git-api-profiles",
    to: "git-svc",
    tooltip:
      "The profiles batch API hits the service layer, which parses each profile (plist for Apple, SyncML XML for Windows, JSON for Android), validates it, and stages the diff.",
  },
  {
    from: "git-api-policies",
    to: "git-svc",
    tooltip:
      "The policies batch API runs each policy through verifyPolicyPlatforms + verifyPolicyName and other guards before accepting it for the plan.",
  },
  {
    from: "git-api-scripts",
    to: "git-svc",
    tooltip:
      "The scripts batch API validates script content, format, and target platforms through the service layer.",
  },
  {
    from: "git-api-labels",
    to: "git-svc",
    tooltip:
      "The labels batch API runs each label's SQL through the osquery parser to confirm it's valid before saving.",
  },

  {
    from: "git-svc",
    to: "git-diff",
    label: "compute plan",
    tooltip:
      "The service layer compares the incoming desired-state batch to the fleet's current rows in MySQL and computes: what to insert, what to update, what to soft-delete. This diff is the \"plan\" `--dry-run` prints.",
  },
  {
    from: "git-diff",
    to: "git-mysql",
    label: "apply writes",
    tooltip:
      "The plan is executed as a batch of MySQL writes inside a transaction. Each resource type (software, profiles, policies, scripts, labels) has its own tables; the transaction keeps a GitOps apply atomic. All succeed or all roll back.",
  },
  {
    from: "git-svc",
    to: "git-activity",
    dashed: true,
    tooltip:
      "Each resource change writes an audit activity row (edited_policy, edited_software_title, etc.) attributed to the fleetctl user. Downstream integrations and the activity feed read these.",
  },

  {
    from: "git-mysql",
    to: "git-crons",
    dashed: true,
    tooltip:
      "After GitOps writes land, the reconciler crons (mdm_apple_profile_manager, mdm_windows_profile_manager, mdm_android_profile_manager, maintained_apps_auto_update) notice the new desired state on their next tick and push commands to hosts.",
  },
  {
    from: "git-mysql",
    to: "git-side-install",
    dashed: true,
    tooltip:
      "Software titles declared with install_everywhere trigger immediate install enqueues for every matching host on the fleet, same code path as the admin Install button. GitOps is not just declarative. It has active side effects.",
  },
  {
    from: "git-mysql",
    to: "git-side-remove",
    dashed: true,
    tooltip:
      "When a title or profile is removed from the YAML, GitOps doesn't just soft-delete the row. It also fires removal side-effects (profiles get removed from hosts, installers become unavailable in self-service). GitOps is destructive by design.",
  },
];
