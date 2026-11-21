export type PathStep = {
  label: string;
  href: string;
  why: string;
};

export type PathKind = "role" | "team";

export type AssessmentRef = {
  // Which quiz surface + bucket slug within it.
  // For "tracer", bucketSlug is the tracer slug (e.g. "list-hosts").
  surface: "glossary" | "architecture" | "tracer";
  bucketSlug: string;
  bucketLabel: string;
};

export type RolePath = {
  slug: string;
  label: string;
  intro: string;
  steps: PathStep[];
  assessments: AssessmentRef[];
};

export const PATHS: RolePath[] = [
  {
    slug: "backend-engineer",
    label: "Backend engineer",
    intro:
      "You'll be writing Go against MySQL and shipping behind feature flags. Follow this path to build the mental model before your first ticket.",
    assessments: [
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "glossary", bucketSlug: "backend", bucketLabel: "Backend (glossary)" },
      { surface: "architecture", bucketSlug: "system", bucketLabel: "System map" },
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "architecture", bucketSlug: "gitops", bucketLabel: "GitOps map" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Architecture map (System view)",
        href: "/",
        why: "Where everything lives. Fleet-the-process is one box; everything else is an external service, data store, or client.",
      },
      {
        label: "Backend conventions",
        href: "/backend-conventions",
        why: "The house conventions. ctxerr, authz, datastore interface, DBReader vs DBWriter. Read these before you open a PR.",
      },
      {
        label: "Request tracer: List hosts",
        href: "/tracer/list-hosts",
        why: "A canonical request walked layer by layer. See how handler → service → datastore → MySQL fit together.",
      },
      {
        label: "Architecture map (Software installs view)",
        href: "/",
        why: "Fleet's most-touched domain. Understand the shape of the install path even if your team isn't software.",
      },
      {
        label: "Data layer: MySQL",
        href: "/data/mysql",
        why: "~240 tables grouped by topic. Scan the groups once; come back when a ticket touches one.",
      },
      {
        label: "Background work: Cron + Workers",
        href: "/background/cron",
        why: "Half of Fleet's logic runs off the request path. Learn the shape of crons and the worker queue.",
      },
      {
        label: "Subsystems: pick one",
        href: "/subsystems",
        why: "Deep-dive on whichever subsystem your team owns (MDM, osquery, vuln, etc.). Each has lifecycle + edge cases.",
      },
      {
        label: "Glossary",
        href: "/glossary",
        why: "Terms you'll hear in every meeting (DEP, VPP, DDM, ctxerr, nanomdm). Keep it open in a tab.",
      },
    ],
  },
  {
    slug: "frontend-engineer",
    label: "Frontend engineer",
    intro:
      "The UI is one API client of several. The backend story matters because it shapes every endpoint response you consume.",
    assessments: [
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "glossary", bucketSlug: "non-coding", bucketLabel: "Non-coding (glossary)" },
      { surface: "architecture", bucketSlug: "system", bucketLabel: "System map" },
      { surface: "architecture", bucketSlug: "gitops", bucketLabel: "GitOps map" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Architecture map (System view)",
        href: "/",
        why: "The UI is one external client in the top row. Confirm you know where to put yourself in the picture.",
      },
      {
        label: "Glossary",
        href: "/glossary",
        why: "House terminology you'll hit in sprint planning (fleets vs teams, reports vs queries, DEP vs ADE).",
      },
      {
        label: "Backend conventions (terminology section)",
        href: "/backend-conventions",
        why: "Why the rename to 'fleets' and 'reports' matters for new user-facing text and why the code still says 'team'.",
      },
      {
        label: "Request tracer: List hosts",
        href: "/tracer/list-hosts",
        why: "What your TanStack Query call actually does on the server. Helps you reason about caching and errors.",
      },
      {
        label: "Scenario: Admin clicks Install",
        href: "/scenarios/install-from-host-library",
        why: "The classic FE flow. See exactly what the Install button triggers end to end.",
      },
      {
        label: "Scenario: Live query",
        href: "/scenarios/live-query",
        why: "WebSocket + Redis pubsub. Explains why 'the UI goes live' is more than a websocket.",
      },
      {
        label: "Subsystem: Auth & sessions",
        href: "/subsystems/auth-sessions",
        why: "How your Authorization header gets turned into a user. Critical for debugging 401s.",
      },
    ],
  },
  {
    slug: "qa",
    label: "QA",
    intro:
      "You verify behavior across the stack. Scenarios + request tracer + subsystem edge cases are home turf.",
    assessments: [
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "architecture", bucketSlug: "gitops", bucketLabel: "GitOps map" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Scenarios (all)",
        href: "/scenarios",
        why: "Read these as test cases. The takeaways are your 'where to look when it fails' notes.",
      },
      {
        label: "Request tracer: all 5 endpoints",
        href: "/tracer",
        why: "If you know what each endpoint should do, regressions are obvious.",
      },
      {
        label: "Subsystems: all MDM pages",
        href: "/subsystems",
        why: "Edge cases + failure modes written up by subsystem. QA goldmine.",
      },
      {
        label: "Background work: Queues",
        href: "/background/queues",
        why: "'Why is this stuck?' repro patterns.",
      },
      {
        label: "Architecture map (Software installs view)",
        href: "/architecture/maps",
        why: "See every install entry point in one picture; QA has to cover all of them.",
      },
      {
        label: "Backend conventions",
        href: "/backend-conventions",
        why: "What reviewers flag; good context when writing bug reports.",
      },
    ],
  },
  {
    slug: "infra-eng",
    label: "Infrastructure engineer",
    intro:
      "You own how Fleet gets built, deployed, and operated. The data layer, background work, and system view are home.",
    assessments: [
      { surface: "architecture", bucketSlug: "system", bucketLabel: "System map" },
      { surface: "glossary", bucketSlug: "backend", bucketLabel: "Backend (glossary)" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Architecture map (System view)",
        href: "/architecture/maps",
        why: "Processes, external services, data stores: the deployment picture.",
      },
      {
        label: "Data layer: MySQL",
        href: "/data/mysql",
        why: "Schema groups + replica story. Capacity planning starts here.",
      },
      {
        label: "Data layer: Redis",
        href: "/data/redis",
        why: "Locks, pubsub, rate limits. Fleet without Redis is Fleet broken.",
      },
      {
        label: "Data layer: File store",
        href: "/data/filestore",
        why: "S3 vs local disk; blob sizing, access patterns, retention.",
      },
      {
        label: "Background work: Cron catalog",
        href: "/background/cron",
        why: "What runs when, what breaks if it doesn't. Essential ops map.",
      },
      {
        label: "Background work: Worker queue",
        href: "/background/workers",
        why: "The durable job queue in MySQL. Draining patterns + failure modes.",
      },
      {
        label: "Background work: Queues (all)",
        href: "/background/queues",
        why: "Six queues total. Each has a different failure surface.",
      },
      {
        label: "Backend conventions (migrations + DBReader/DBWriter)",
        href: "/backend-conventions",
        why: "Migration ordering rule + replica lag awareness. Operations gotchas.",
      },
      {
        label: "Subsystem: osquery TLS",
        href: "/subsystems/osquery-tls",
        why: "Fleet's highest-rate protocol. Load, timeouts, auth all live here.",
      },
    ],
  },
  {
    slug: "product-design",
    label: "Product design",
    intro:
      "You shape what Fleet looks and feels like. Fleet Atlas helps you calibrate what's quick to build vs what's quarters of work, and makes sure you speak the vocabulary customers do.",
    assessments: [
      { surface: "glossary", bucketSlug: "non-coding", bucketLabel: "Non-coding (glossary)" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
    ],
    steps: [
      {
        label: "Architecture map (System view)",
        href: "/architecture/maps",
        why: "The 20-second mental model of what Fleet is made of.",
      },
      {
        label: "Scenarios (filter by tag)",
        href: "/scenarios",
        why: "Step-by-step walkthroughs of real flows. Builds intuition for which features touch which subsystems.",
      },
      {
        label: "Glossary",
        href: "/glossary/terms",
        why: "Acronyms per minute is high in product conversations. Keep this open during planning.",
      },
      {
        label: "Architecture map (Software installs view)",
        href: "/architecture/maps",
        why: "The whole software/vuln/auto-patch domain in one picture; a top area for product decisions.",
      },
      {
        label: "Architecture map (MDM view)",
        href: "/architecture/maps",
        why: "Why 'small MDM feature' can take a quarter: nine clouds, three protocols.",
      },
      {
        label: "Background work: Cron catalog",
        href: "/background/cron",
        why: "Explains delay windows. 'Why does the vuln show up an hour late?' lives here.",
      },
    ],
  },
  {
    slug: "customer-support-sa",
    label: "Customer support and solutions architecture",
    intro:
      "You handle customer tickets AND pre-sales technical scoping. Fleet Atlas is your playbook for both troubleshooting a stuck install and sketching a GitOps rollout for a prospect.",
    assessments: [
      { surface: "glossary", bucketSlug: "customer-success", bucketLabel: "Customer success (glossary)" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "architecture", bucketSlug: "gitops", bucketLabel: "GitOps map" },
    ],
    steps: [
      {
        label: "Scenarios (all)",
        href: "/scenarios",
        why: "Every scenario ends with 'where to look when it goes wrong.' Playbook for both tickets and demos.",
      },
      {
        label: "Subsystems: MDM Apple / Microsoft / Android",
        href: "/subsystems",
        why: "The three MDM pages read like troubleshooting playbooks AND architecture overviews for scoping conversations.",
      },
      {
        label: "Background work: Queues",
        href: "/background/queues",
        why: "'Why is this stuck?' is almost always a queue. Learn the six of them.",
      },
      {
        label: "Glossary",
        href: "/glossary/terms",
        why: "Every acronym in customer tickets and sales calls lives here.",
      },
      {
        label: "Architecture map (System view)",
        href: "/architecture/maps",
        why: "The whiteboard mental model you'll draw for both ticket triage and prospect calls.",
      },
      {
        label: "Architecture map (MDM view)",
        href: "/architecture/maps",
        why: "The most-asked-about surface in both tickets and demos.",
      },
      {
        label: "Architecture map (GitOps view)",
        href: "/architecture/maps",
        why: "For scoping GitOps-based implementations and troubleshooting admin-reported drift.",
      },
    ],
  },
  {
    slug: "customer-success",
    label: "Customer success",
    intro:
      "You're the Fleet CSM: keeping customers happy, renewed, and expanding. Fleet Atlas builds the fluency to answer product questions on calls without needing to escalate.",
    assessments: [
      { surface: "glossary", bucketSlug: "customer-success", bucketLabel: "Customer success (glossary)" },
      { surface: "glossary", bucketSlug: "non-coding", bucketLabel: "Non-coding (glossary)" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
    ],
    steps: [
      {
        label: "Scenarios (all)",
        href: "/scenarios",
        why: "Customer feature stories. Each ends with 'when it goes wrong' notes you can share on QBRs.",
      },
      {
        label: "Glossary",
        href: "/glossary/terms",
        why: "The vocabulary customers use. Build fluency before every renewal call.",
      },
      {
        label: "Architecture map (System view)",
        href: "/architecture/maps",
        why: "The whiteboard mental model when a customer asks 'how does Fleet work again?'",
      },
      {
        label: "Architecture map (Software installs view)",
        href: "/architecture/maps",
        why: "The newest value prop; most expansion conversations touch it.",
      },
      {
        label: "Architecture map (MDM view)",
        href: "/architecture/maps",
        why: "The biggest customer concern area. Enrollment and profile troubleshooting lives here.",
      },
      {
        label: "Subsystems: pick MDM + Software",
        href: "/subsystems",
        why: "The two product pillars most customers care about day-to-day.",
      },
    ],
  },
  {
    slug: "it-strategy",
    label: "IT strategy",
    intro:
      "You think big-picture about where Fleet should play in the IT, MDM, and vuln landscape. Fleet Atlas builds the technical depth to make those bets well.",
    assessments: [
      { surface: "glossary", bucketSlug: "non-coding", bucketLabel: "Non-coding (glossary)" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
      { surface: "glossary", bucketSlug: "security-vuln", bucketLabel: "Standards & vuln (glossary)" },
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "architecture", bucketSlug: "gitops", bucketLabel: "GitOps map" },
    ],
    steps: [
      {
        label: "Architecture map (System view)",
        href: "/architecture/maps",
        why: "The whole stack at a glance. Where Fleet sits vs where it could sit.",
      },
      {
        label: "Architecture map (MDM view)",
        href: "/architecture/maps",
        why: "Strategic MDM landscape. The slowest-moving competitive space.",
      },
      {
        label: "Architecture map (Software installs view)",
        href: "/architecture/maps",
        why: "The auto-patching bet. Fastest-growing use case.",
      },
      {
        label: "Architecture map (GitOps view)",
        href: "/architecture/maps",
        why: "The admin-automation bet. Determines how Fleet scales for large IT shops.",
      },
      {
        label: "Scenarios (all)",
        href: "/scenarios",
        why: "Competitive feature stories and real workflow coverage.",
      },
      {
        label: "Subsystem: Vuln processing",
        href: "/subsystems/vuln-processing",
        why: "Competitive vuln landscape: NVD / OSV / OVAL / MSRC / Office. Know who covers what.",
      },
      {
        label: "Glossary",
        href: "/glossary/terms",
        why: "Industry acronym fluency. Investor, analyst, and partner conversations use every one.",
      },
    ],
  },
  {
    slug: "sales",
    label: "Sales",
    intro:
      "You pitch, qualify, and close Fleet. Fleet Atlas builds just enough technical depth to answer buyer questions on calls without hand-waving.",
    assessments: [
      { surface: "glossary", bucketSlug: "customer-success", bucketLabel: "Customer success (glossary)" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "glossary", bucketSlug: "non-coding", bucketLabel: "Non-coding (glossary)" },
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
    ],
    steps: [
      {
        label: "Architecture map (System view)",
        href: "/architecture/maps",
        why: "Your whiteboard mental model. 20 seconds; one box per thing.",
      },
      {
        label: "Scenarios (all)",
        href: "/scenarios",
        why: "The feature stories you tell on calls. Hook, flow, takeaway.",
      },
      {
        label: "Glossary",
        href: "/glossary/terms",
        why: "Customer RFPs and discovery calls use every one of these.",
      },
      {
        label: "Architecture map (Software installs view)",
        href: "/architecture/maps",
        why: "The newest value prop. Most expansion deals touch this.",
      },
      {
        label: "Architecture map (MDM view)",
        href: "/architecture/maps",
        why: "The biggest buyer concern area. Enough to answer DEP/VPP/DDM questions on your own.",
      },
    ],
  },
];

export const TEAMS: RolePath[] = [
  {
    slug: "apple-at-work",
    label: "Apple at work",
    intro:
      "You own the Apple MDM surface: nanomdm, DEP/ABM, VPP, DDM, APNs. This path walks the stack from protocol endpoints down to the command queue.",
    assessments: [
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Subsystem: Apple MDM",
        href: "/subsystems/apple-mdm",
        why: "nanomdm, nanodep, SCEP/ACME, DDM. The lifecycle writeup.",
      },
      {
        label: "Architecture map (MDM view)",
        href: "/architecture/maps",
        why: "The command queue + reconciler cron flow in one picture.",
      },
      {
        label: "Scenario: Mac DEP enrollment",
        href: "/scenarios/mac-dep-enrollment",
        why: "Zero-touch, step by step. The baseline Apple flow.",
      },
      {
        label: "Scenario: Profile upload reaches a Mac",
        href: "/scenarios/profile-upload-reaches-mac",
        why: "Explains desired-state reconciliation via the apple_mdm_profile_manager cron.",
      },
      {
        label: "Scenario: VPP install",
        href: "/scenarios/vpp-install",
        why: "The double-queue path: upcoming_activities + worker + Apple VPP API.",
      },
      {
        label: "Scenario: Setup experience install",
        href: "/scenarios/setup-experience-install",
        why: "DEP + bootstrap + required apps + DDM release. Your most complex flow.",
      },
      {
        label: "Background work: Queues",
        href: "/background/queues",
        why: "The nanomdm command queue is one of six queues. Learn where it sits.",
      },
    ],
  },
  {
    slug: "power-to-pc",
    label: "Power to the PC",
    intro:
      "You own the Microsoft MDM surface: SyncML, CSPs, Autopilot, Entra integration, plus Windows + Linux setup experience and the cert infrastructure (SCEP, ACME, CA registration). This path walks the Windows MDM stack and the cert-authority plumbing that touches Windows + Linux enrollments.",
    assessments: [
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
      { surface: "architecture", bucketSlug: "system", bucketLabel: "System map" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "glossary", bucketSlug: "security-vuln", bucketLabel: "Standards & vuln (glossary)" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Subsystem: Microsoft MDM",
        href: "/subsystems/microsoft-mdm",
        why: "SyncML XML, OMA-DM, CSP paths, Windows WSTEP enrollment. The daily surface.",
      },
      {
        label: "Architecture map (MDM view)",
        href: "/architecture/maps",
        why: "Windows profile manager cron, windows_mdm_command_queue, Autopilot sync. All in one picture.",
      },
      {
        label: "Architecture node: Windows Autopilot",
        href: "/arch/sys-autopilot",
        why: "Microsoft Graph integration, how Autopilot registrations get into Fleet, Entra JWT verification.",
      },
      {
        label: "Architecture node: SCEP CA",
        href: "/arch/sys-scep-ca",
        why: "Fleet proxies cert enrollment through external CAs (NDES / DigiCert / Smallstep). Your cert infrastructure story starts here.",
      },
      {
        label: "Subsystem: Admin UI anatomy",
        href: "/subsystems/admin-ui-anatomy",
        why: "What Windows admins see when they open Fleet. Controls, profile upload, Settings → Integrations for cert authorities.",
      },
      {
        label: "Glossary deep-dive on Windows MDM terms",
        href: "/glossary/terms#csp",
        why: "CSP paths, SyncML, OMA-DM, Autopilot, Entra, JWKS, JWT. The vocabulary customer tickets use.",
      },
      {
        label: "Glossary deep-dive on cert terms",
        href: "/glossary/terms#scep",
        why: "SCEP, ACME, Managed Local Accounts, Recovery Lock. The cert feature set you own end-to-end.",
      },
      {
        label: "Scenario: Setup experience install",
        href: "/scenarios/setup-experience-install",
        why: "Walks the Mac flow; the Windows + Linux setup experience you own follows the same shape.",
      },
      {
        label: "Backend conventions",
        href: "/backend-conventions",
        why: "ctxerr, authz, datastore interface, migration conventions. Required before your first PR.",
      },
      {
        label: "osquery fundamentals",
        href: "/subsystems/osquery-fundamentals",
        why: "Software inventory on Windows comes from osquery too. Same inventory debugging rules apply.",
      },
      {
        label: "Background work: Cron catalog",
        href: "/background/cron",
        why: "microsoft_autopilot_sync, mdm_windows_profile_manager, reencrypt_windows_disk_encryption_keys. Your crons.",
      },
    ],
  },
  {
    slug: "byod",
    label: "BYOD / user enrollment",
    intro:
      "You own the user-enrollment and personal-device paths: Apple user enrollment, Android work profiles. Narrower management surface, different trust model.",
    assessments: [
      { surface: "architecture", bucketSlug: "mdm", bucketLabel: "MDM map" },
      { surface: "glossary", bucketSlug: "mdm", bucketLabel: "MDM (glossary)" },
      { surface: "glossary", bucketSlug: "non-coding", bucketLabel: "Non-coding (glossary)" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Subsystem: Apple MDM",
        href: "/subsystems/apple-mdm",
        why: "User enrollment is a variant of the Apple MDM path with narrower scope.",
      },
      {
        label: "Subsystem: Android MDM",
        href: "/subsystems/android-mdm",
        why: "Work profile model via AMAPI. Google holds the device channel.",
      },
      {
        label: "Architecture map (MDM view)",
        href: "/architecture/maps",
        why: "Where BYOD enrollment triggers sit relative to DEP/Autopilot/zero-touch.",
      },
      {
        label: "Scenario: Mac DEP enrollment",
        href: "/scenarios/mac-dep-enrollment",
        why: "Compare: zero-touch vs user-initiated enrollment.",
      },
      {
        label: "Glossary: 'work-profile', 'MDM', 'AMAPI'",
        href: "/glossary/terms#work-profile",
        why: "The user-space vocabulary customers use when asking about BYOD.",
      },
    ],
  },
  {
    slug: "orchestration",
    label: "Orchestration",
    intro:
      "You own hosts, policies, labels, reports, scripts, live queries. The 'what is a device doing right now' surface.",
    assessments: [
      { surface: "architecture", bucketSlug: "policies", bucketLabel: "Policies map" },
      { surface: "architecture", bucketSlug: "reports", bucketLabel: "Reports & live queries map" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "glossary", bucketSlug: "backend", bucketLabel: "Backend (glossary)" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Architecture node: osquery agents",
        href: "/arch/osquery",
        why: "Fleetd on hosts: the data source for everything you own.",
      },
      {
        label: "Subsystem: osquery TLS",
        href: "/subsystems/osquery-tls",
        why: "Enroll, config, distributed, log, carve endpoints.",
      },
      {
        label: "Subsystem: Live queries",
        href: "/subsystems/live-queries",
        why: "Redis pubsub fan-out across Fleet instances.",
      },
      {
        label: "Scenario: Live query",
        href: "/scenarios/live-query",
        why: "Admin runs a query against 100 hosts. Watch the pubsub path.",
      },
      {
        label: "Architecture map (Policies view)",
        href: "/architecture/maps",
        why: "Policy evaluation → membership → automations in one picture.",
      },
      {
        label: "Architecture map (Reports view)",
        href: "/architecture/maps",
        why: "Scheduled vs live query, side by side.",
      },
      {
        label: "Request tracer: List hosts",
        href: "/tracer/list-hosts",
        why: "The canonical authed read; your most-called endpoint.",
      },
      {
        label: "Backend conventions",
        href: "/backend-conventions",
        why: "Iterator pattern, DBReader vs DBWriter, authz pattern. All show up in your area.",
      },
    ],
  },
  {
    slug: "auto-patching",
    label: "Auto-patching / software",
    intro:
      "You own installs, vulns, and the loop that closes between them. Custom installers, FMAs, VPP, in-house apps, and policy-driven auto-install.",
    assessments: [
      { surface: "architecture", bucketSlug: "software", bucketLabel: "Software installs map" },
      { surface: "glossary", bucketSlug: "fleet", bucketLabel: "Fleet (glossary)" },
      { surface: "glossary", bucketSlug: "backend", bucketLabel: "Backend (glossary)" },
      { surface: "glossary", bucketSlug: "security-vuln", bucketLabel: "Standards & vuln (glossary)" },
      { surface: "tracer", bucketSlug: "list-hosts", bucketLabel: "GET /hosts. Order the steps" },
      { surface: "tracer", bucketSlug: "install-software-title", bucketLabel: "POST /install. Order the steps" },
    ],
    steps: [
      {
        label: "Architecture map (Software installs view)",
        href: "/architecture/maps",
        why: "The whole domain in one picture: triggers, catalogs, queues, delivery, vuln loop.",
      },
      {
        label: "Scenario: Install from host software library",
        href: "/scenarios/install-from-host-library",
        why: "Baseline manual install path. Everything else is a variant.",
      },
      {
        label: "Scenario: FMA auto-update",
        href: "/scenarios/fma-auto-update",
        why: "Nightly catalog bump + hourly rollout. The core of auto-patching.",
      },
      {
        label: "Scenario: Policy-driven auto-install",
        href: "/scenarios/policy-driven-auto-install",
        why: "Self-healing on failing policies. Watch the dedup counter.",
      },
      {
        label: "Scenario: VPP install",
        href: "/scenarios/vpp-install",
        why: "VPP: Apple delivers bytes, not Fleet. Different mechanics.",
      },
      {
        label: "Scenario: CVE match",
        href: "/scenarios/cve-match",
        why: "Vuln pipeline mechanics; feeds the automation loop.",
      },
      {
        label: "Subsystem: Vuln processing",
        href: "/subsystems/vuln-processing",
        why: "NVD / OSV / OVAL / MSRC and CPE matching gotchas.",
      },
      {
        label: "Background work: Queues",
        href: "/background/queues",
        why: "Software installs touch at least two queues (upcoming_activities + worker).",
      },
      {
        label: "Request tracer: Install software title",
        href: "/tracer/install-software-title",
        why: "The ee wrapper pattern laid bare. Your most important write path.",
      },
    ],
  },
];
