export type GlossaryCategory =
  | "Fleet concept"
  | "Fleet code"
  | "Apple MDM"
  | "Microsoft MDM"
  | "Android MDM"
  | "Backend / Go"
  | "Standards & protocols"
  | "Database"
  | "Vuln / Security";

export const CATEGORY_COLORS: Record<GlossaryCategory, string> = {
  "Fleet concept":
    "bg-sky-100 text-sky-800 dark:bg-sky-900/60 dark:text-sky-200",
  "Fleet code":
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/60 dark:text-indigo-200",
  "Apple MDM":
    "bg-pink-100 text-pink-800 dark:bg-pink-900/60 dark:text-pink-200",
  "Microsoft MDM":
    "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/60 dark:text-cyan-200",
  "Android MDM":
    "bg-lime-100 text-lime-800 dark:bg-lime-900/60 dark:text-lime-200",
  "Backend / Go":
    "bg-violet-100 text-violet-800 dark:bg-violet-900/60 dark:text-violet-200",
  "Standards & protocols":
    "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200",
  Database:
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200",
  "Vuln / Security":
    "bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-900/60 dark:text-fuchsia-200",
};

export type GlossaryTerm = {
  slug: string;
  term: string;
  aka?: string[];
  category: GlossaryCategory;
  short: string;
  long?: string;
  seeAlso?: string[];
  links?: { label: string; href: string }[];
  // When Technical help mode is on, the term card shows a pop-out with
  // `technical` as the deeper technical explanation and `analogies` as
  // plain-language comparisons.
  technicalHelp?: {
    technical: string;
    analogies: string[];
  };
};

export const TERMS: GlossaryTerm[] = [
  // ─── Fleet concept ───
  {
    slug: "fleet",
    term: "Fleet (vs team)",
    category: "Fleet concept",
    short:
      "A group of hosts that share config, policies, and software. Was called 'team' in older code and the database.",
    long:
      "The product renamed 'teams' to 'fleets'. New UI copy and user docs use 'fleet'. Legacy code still has team_id columns, teams table, fleet.Team struct. Don't rename without guidance.",
    seeAlso: ["report"],
  },
  {
    slug: "report",
    term: "Report (vs query)",
    category: "Fleet concept",
    short:
      "A saved SQL query that runs on hosts. Was called a 'query' before the rename.",
    long:
      "User-facing surfaces call it a report; code still has queries table, fleet.Query struct, scheduled_queries. 'Query' in current product usage refers to the SQL itself. The shared building block that appears in reports, policies, pre-install checks on software titles, and ad-hoc live queries.",
    seeAlso: ["live-query", "policy"],
  },
  {
    slug: "live-query",
    term: "Live query",
    category: "Fleet concept",
    short:
      "An ad-hoc, one-shot distributed query launched from the UI. Results stream back live.",
    long:
      "Different from a scheduled report: no persistence of results in query_results; results flow through Redis pubsub to a WebSocket. Campaign row in distributed_query_campaigns.",
    seeAlso: ["report", "pubsub"],
    links: [{ label: "Scenario", href: "/scenarios/live-query" }],
  },
  {
    slug: "policy",
    term: "Policy",
    category: "Fleet concept",
    short:
      "A SQL query that must return at least one row for a host to be 'passing'. Pass/fail is row-count-based.",
    seeAlso: ["report"],
  },
  {
    slug: "label",
    term: "Label",
    category: "Fleet concept",
    short:
      "A dynamic host group defined by a SQL query. If the query returns a row for host H, H is in the label.",
  },
  {
    slug: "host",
    term: "Host",
    category: "Fleet concept",
    short:
      "A managed device. Row in hosts table. Could be a Mac, Windows PC, Linux server, iPhone, Android, or ChromeOS.",
  },
  {
    slug: "fleetd",
    term: "fleetd",
    aka: ["Orbit"],
    category: "Fleet concept",
    short:
      "Fleet's agent installed on managed devices. Wraps osqueryd and Fleet Desktop; handles updates via TUF.",
    seeAlso: ["osqueryd", "fleet-desktop", "tuf"],
    technicalHelp: {
      technical:
        "A small program Fleet installs on every managed device. It's the only thing on the device that talks to Fleet. It handles osquery (the data collector), Fleet Desktop (the menu-bar app for end users), and auto-updates itself when Fleet releases a new version.",
      analogies: [
        "Like having a smart meter installed at your house. The utility (Fleet) doesn't walk around to check your usage: the meter reports in automatically and can be upgraded over the air.",
      ],
    },
  },
  {
    slug: "osqueryd",
    term: "osqueryd",
    category: "Fleet concept",
    short:
      "The osquery daemon. Open-source project that lets you query device state with SQL. Fleet bundles it inside fleetd.",
    technicalHelp: {
      technical:
        "osquery turns a device into something you can ask database-style questions: 'what processes are running?', 'is FileVault on?', 'which Chrome extensions are installed?' Fleet's agent includes osquery, and Fleet asks it questions on the admin's behalf.",
      analogies: [
        "Like turning your laptop into a mini-database. Instead of 'look through control panels and settings', you just write a SQL query and get the answer.",
      ],
    },
  },
  {
    slug: "fleet-desktop",
    term: "Fleet Desktop",
    category: "Fleet concept",
    short:
      "The optional menu-bar/system-tray app that gives end users a 'My Device' page.",
  },
  {
    slug: "fleetctl",
    term: "fleetctl",
    category: "Fleet concept",
    short:
      "The admin CLI client. Shares its Go API client with the server. GitOps mode reconciles YAML.",
    seeAlso: ["gitops"],
  },
  {
    slug: "gitops",
    term: "GitOps",
    category: "Fleet concept",
    short:
      "Declarative YAML as the source of truth for Fleet config. `fleetctl gitops` diffs desired vs actual and applies.",
    seeAlso: ["fleetctl"],
  },
  {
    slug: "fma",
    term: "Fleet-maintained app (FMA)",
    category: "Fleet concept",
    short:
      "A curated software catalog Fleet maintains, sourced from Homebrew (macOS) and winget (Windows). Rebuilt nightly.",
    links: [{ label: "Scenario", href: "/scenarios/fma-auto-update" }],
    technicalHelp: {
      technical:
        "Fleet maintains a shelf of common apps (Chrome, Slack, 1Password, etc.) with install scripts already worked out. Admins pick from the shelf instead of hunting down installers themselves. Fleet refreshes the shelf nightly so new versions appear automatically.",
      analogies: [
        "Like a company-managed app store: IT already vetted and packaged the common apps, employees (or admins) just pick what they need.",
      ],
    },
  },
  {
    slug: "vpp",
    term: "VPP",
    aka: ["Volume Purchase Program"],
    category: "Apple MDM",
    short:
      "Apple's App Store volume licensing. Fleet assigns licenses; Apple delivers the apps to devices.",
    links: [{ label: "Scenario", href: "/scenarios/vpp-install" }],
    technicalHelp: {
      technical:
        "VPP is how a company buys App Store apps in bulk and assigns them to employee devices. Fleet tells Apple 'give this app's license to this device'; Apple delivers the actual app from the App Store. Fleet never ships the app bytes itself.",
      analogies: [
        "Like a company Spotify family plan. The company buys a bunch of seats, assigns them to employees. Spotify handles the music delivery.",
      ],
    },
  },
  {
    slug: "self-service",
    term: "Self-service",
    category: "Fleet concept",
    short:
      "End-user install flow from the My Device page. Device-token auth, allowlisted titles.",
    links: [{ label: "Scenario", href: "/scenarios/self-service-install" }],
    technicalHelp: {
      technical:
        "Admins put a short list of pre-approved apps in a 'self-service' catalog that end users can see on their own My Device page. The employee picks what they want; Fleet installs it. No IT ticket, no admin intervention.",
      analogies: [
        "Like a drink station at a conference: IT picked what's on offer, employees serve themselves.",
      ],
    },
  },
  {
    slug: "setup-experience",
    term: "Setup experience",
    category: "Apple MDM",
    short:
      "The window between MDM enrollment and the user logging in. Fleet uses it to install bootstrap packages, run scripts, and pre-install required software.",
    links: [{ label: "Scenario", href: "/scenarios/setup-experience-install" }],
    technicalHelp: {
      technical:
        "When a brand-new Mac is set up, there's a short window before the user ever sees the desktop. Fleet uses that window to pre-install everything the person will need (fleetd, Chrome, Slack, VPN config, etc.) so when they log in for the first time, it already feels configured.",
      analogies: [
        "Like checking into a hotel room where the TV, remote, towels, and welcome card are already set up before you walk in.",
      ],
    },
  },
  {
    slug: "enroll-secret",
    term: "Enroll secret",
    category: "Fleet concept",
    short:
      "Pre-shared secret a host presents to join a fleet. Per-fleet; rotatable.",
    seeAlso: ["node-key"],
    technicalHelp: {
      technical:
        "A password-like string Fleet generates. A device wanting to join Fleet has to present this on its first contact. It's shared among devices (not unique per device), but each fleet can have its own and admins can rotate.",
      analogies: [
        "Like the Wi-Fi password for the office: anyone with the password can join the network, but only the people you give it to.",
      ],
    },
  },
  {
    slug: "node-key",
    term: "Node key",
    category: "Fleet concept",
    short:
      "Per-host bearer token Fleet issues at enroll. Used on every osquery TLS request. Rotates on re-enroll.",
    seeAlso: ["enroll-secret", "device-token"],
    technicalHelp: {
      technical:
        "After a device uses the enroll secret to join, Fleet issues it a unique per-device token. The device uses this unique token on every future check-in so Fleet knows exactly which device is talking.",
      analogies: [
        "The enroll secret is the shared Wi-Fi password. The node key is the lease: a unique ID the router hands you once you're on the network.",
      ],
    },
  },
  {
    slug: "device-token",
    term: "Device token",
    category: "Fleet concept",
    short:
      "Per-host URL-safe secret for /device/* endpoints. Used by the My Device page.",
    seeAlso: ["node-key"],
    technicalHelp: {
      technical:
        "The node key is for the device itself (fleetd talking to Fleet). The device token is for the end user: a different unique string, usable in a URL, that lets a person open their own My Device page without logging into Fleet as an admin.",
      analogies: [
        "Like a shareable Google Drive link: it points to a specific thing, and anyone with the link can view that specific thing (and only that thing).",
      ],
    },
  },
  {
    slug: "fleet-variable",
    term: "Fleet variable ($FLEET_VAR_*)",
    category: "Fleet concept",
    short:
      "Substitution placeholder in profiles/scripts. Resolved per-host at delivery time.",
    seeAlso: ["secret-variable"],
    technicalHelp: {
      technical:
        "Admins write one profile with placeholders like $FLEET_VAR_HOST_UUID in it. When Fleet delivers that profile to each host, it swaps in the right value for that host. Lets you write one profile instead of 500.",
      analogies: [
        "Like mail merge for form letters: 'Dear [FIRST_NAME],' becomes 'Dear Alice,' or 'Dear Bob,' at send time.",
      ],
    },
  },
  {
    slug: "secret-variable",
    term: "Secret variable",
    category: "Fleet concept",
    short:
      "Encrypted sibling of Fleet variables. Values aren't readable back out.",
    technicalHelp: {
      technical:
        "Same mail-merge idea as a Fleet variable, but for sensitive values (API keys, passwords, cert passphrases). Fleet encrypts them and never shows them back to the admin after they're saved. Only substitutes them in at delivery time.",
      analogies: [
        "Like a password manager that fills in the password for you but never lets you see it on screen.",
      ],
    },
  },
  {
    slug: "activity",
    term: "Activity (log)",
    category: "Fleet concept",
    short:
      "Audit entry for admin actions. Written to activity_past via svc.NewActivity. Powers the activity feed + optional external streaming.",
    seeAlso: ["activity-streaming"],
    technicalHelp: {
      technical:
        "Every time an admin does something in Fleet (adds a user, uploads a profile, runs a live query), Fleet writes a row to an audit log table. This is what customers' security teams use to answer 'who did what?' later.",
      analogies: [
        "Like the history tab on a shared Google Doc: an unchangeable record of who did what, visible to admins.",
      ],
    },
  },
  {
    slug: "activity-streaming",
    term: "Activity streaming",
    category: "Fleet concept",
    short:
      "Optional firehose of activity rows to a webhook or SIEM. activities_streaming cron drains it.",
    technicalHelp: {
      technical:
        "Besides keeping the audit log inside Fleet, customers can tell Fleet to forward every activity to an external system (like Splunk, DataDog, or their SIEM) as events happen. Useful for central security monitoring.",
      analogies: [
        "Like having your bank text your phone every time a transaction happens, instead of waiting to log in and check.",
      ],
    },
  },
  {
    slug: "carve",
    term: "Carve",
    category: "Fleet concept",
    short:
      "A file osquery uploads from a host in chunks. Lives in carve_metadata + carve_blocks.",
    technicalHelp: {
      technical:
        "An admin can ask osquery to copy a specific file (say, a suspicious binary) off a host and ship it to Fleet. Because files can be big, osquery sends it in chunks. Fleet reassembles them.",
      analogies: [
        "Like forwarding a huge email by splitting it into multiple messages, then combining them on the other side.",
      ],
    },
  },

  // ─── Fleet code ───
  {
    slug: "fleet-service",
    term: "fleet.Service",
    category: "Fleet code",
    short:
      "The big Go interface declared in server/fleet/service.go. Every protected endpoint calls into it. The business-logic surface.",
    seeAlso: ["fleet-datastore", "ee-wrapper"],
    technicalHelp: {
      technical:
        "Fleet's engineering layers things cleanly: HTTP handlers are thin; they call into a 'Service' object that holds all the business logic. If you want to understand what a feature DOES (ignoring the SQL and the HTTP), it's in the service.",
      analogies: [
        "In a restaurant: the server (HTTP handler) takes your order; the kitchen (service) makes the dish; the pantry (datastore) is where ingredients come from.",
      ],
    },
  },
  {
    slug: "fleet-datastore",
    term: "fleet.Datastore",
    category: "Fleet code",
    short:
      "The storage interface. Service methods call it; MySQL impl is in server/datastore/mysql/.",
    seeAlso: ["dbreader-dbwriter"],
    technicalHelp: {
      technical:
        "The 'Datastore' is Fleet's only path to the database. Business logic never writes SQL directly; it calls datastore methods like ListHosts or CreatePolicy. Keeps the database layer swappable (and testable with fakes).",
      analogies: [
        "Like only being allowed to go to the warehouse through a dock worker. The dock worker handles 'where does this box go?'; you just say 'I need these items.'",
      ],
    },
  },
  {
    slug: "ctxerr",
    term: "ctxerr",
    category: "Fleet code",
    short:
      "Fleet's context-aware error wrapper. ctxerr.Wrap(ctx, err, 'message') attaches context + stack to the error.",
    links: [{ label: "Backend conventions", href: "/backend-conventions" }],
    technicalHelp: {
      technical:
        "When something fails deep in Fleet, you want the error log to say 'this failed inside the install flow for host 42' not just 'DB error.' ctxerr is a Fleet helper that attaches that extra context to every error as it bubbles up. The engineering convention is to use it everywhere so logs are debuggable.",
      analogies: [
        "Like adding a sticky note to a complaint every time it gets passed to a new manager: 'from accounting, then forwarded via HR, originally about the printer in room 203.' By the time it reaches the CEO, there's a trail.",
      ],
    },
  },
  {
    slug: "makehandler",
    term: "MakeHandler",
    category: "Fleet code",
    short:
      "The function in server/service/handler.go that registers every HTTP route. Gorilla/mux inside.",
    seeAlso: ["go-kit-endpoint"],
    technicalHelp: {
      technical:
        "The master phone-book of every URL Fleet responds to. Each entry wires a URL to the function that handles it. If someone says 'where does POST /api/v1/fleet/hosts/{id}/install go?', MakeHandler is where you find out.",
      analogies: [
        "Like the office directory: 'if you need accounting, dial 203; if you need HR, dial 405.' Every URL = an extension to a specific handler.",
      ],
    },
  },
  {
    slug: "go-kit-endpoint",
    term: "Endpoint (go-kit)",
    category: "Fleet code",
    short:
      "A function that converts (decoded request → service call → response). Thin business adapter between HTTP and service.",
    seeAlso: ["makehandler"],
    technicalHelp: {
      technical:
        "A thin glue function between the HTTP request layer and the business logic (service). Takes the already-parsed inputs, calls the service, wraps the result for the HTTP layer to send back. Doing it this way means business logic doesn't have to know about HTTP.",
      analogies: [
        "Like a translator between English and French: the diplomats talk policy; the translator relays each sentence without changing it.",
      ],
    },
  },
  {
    slug: "fleet-free",
    term: "Fleet Free",
    category: "Fleet code",
    short:
      "The open-source, no-license tier. Checked in code as !license.IsPremium(). Free is the negation.",
    seeAlso: ["fleet-premium", "license-tier", "ee-wrapper"],
    links: [{ label: "Free vs Premium", href: "/tiers" }],
    technicalHelp: {
      technical:
        "There is no IsFree() helper in Fleet. 'Free' is the absence of a premium license. Code paths that enable premium-only features gate on license.IsPremium() being true; everything else is Free. In the frontend, Free is !isPremiumTier. Negating that flag silently swaps the meaning, which has caused regressions.",
      analogies: [
        "Like 'logged out' in most apps: there's no isLoggedOut() method, just !isLoggedIn(). Free is Fleet's default state.",
      ],
    },
  },
  {
    slug: "fleet-premium",
    term: "Fleet Premium",
    category: "Fleet code",
    short:
      "The paid tier. Enables fleets (teams), SSO/SCIM, DDM, VPP, policy automations, vuln automations, activity streaming, and more.",
    seeAlso: ["fleet-free", "license-tier", "ee-wrapper"],
    links: [{ label: "Free vs Premium", href: "/tiers" }],
    technicalHelp: {
      technical:
        "A license key sets license.Tier to 'premium'. Service methods that implement premium-only capability check license.IsPremium() and return ErrMissingLicense (which the HTTP layer renders as a 402) when the caller is on Free. The ee/ wrapper is how most premium behavior is layered on without touching the core service.",
      analogies: [
        "Like a paid tier in a SaaS product: the app is the same binary, but a few flows unlock when the license says so.",
      ],
    },
  },
  {
    slug: "license-tier",
    term: "License tier",
    aka: ["TierFree / TierPremium"],
    category: "Fleet code",
    short:
      "The string that drives every premium gate. Defined as the constants TierFree = \"free\" and TierPremium = \"premium\" in server/fleet/app.go.",
    seeAlso: ["fleet-free", "fleet-premium", "ee-wrapper"],
    links: [{ label: "Free vs Premium", href: "/tiers" }],
    technicalHelp: {
      technical:
        "The license is validated at server start and attached to the request context. license.IsPremium() (sync) and license.IsPremium(ctx) (ctx-based) are the two call sites. The sentinel error for a failed check is ErrMissingLicense (server/fleet/errors.go). A deprecated 'basic' tier is still accepted as premium-equivalent in legacy installs.",
      analogies: [
        "Like a feature-flag string, but once per process rather than per-request toggleable: 'premium' turns on a bundle of behavior across the server.",
      ],
    },
  },
  {
    slug: "ee-wrapper",
    term: "Enterprise wrapper (ee/)",
    category: "Fleet code",
    seeAlso: ["fleet-free", "fleet-premium", "license-tier"],
    links: [{ label: "Free vs Premium", href: "/tiers" }],
    short:
      "Go struct in ee/server/service that embeds the core service and overrides specific methods for license-tier behavior.",
    technicalHelp: {
      technical:
        "Fleet has free and premium tiers. Rather than sprinkle 'if (premium)' checks everywhere, Fleet has an Enterprise wrapper. It sits around the core service and intercepts specific methods to add license-check behavior. The main service doesn't have to know about tiers.",
      analogies: [
        "Like a VIP wrapper around a restaurant: regulars go through the main entrance; VIPs go through the same entrance but a VIP greeter also handles their extra perks.",
      ],
    },
  },
  {
    slug: "gitops-mode",
    term: "GitOps mode",
    category: "Fleet concept",
    short:
      "A per-config flag (`gitops_mode_enabled`) that locks Fleet's UI into read-only. All config changes must come through fleetctl gitops apply.",
    seeAlso: ["ee-wrapper", "fleet-premium"],
    technicalHelp: {
      technical:
        "When GitOps mode is enabled on a fleet config, the Fleet UI continues to render everything but disables every edit control (New / Delete / Modify buttons, inline edits, settings forms). The server also enforces the lock, so a scripted API call gets rejected the same way. An exceptions list lets specific resource types stay UI-editable even when the mode is on (common exceptions: labels, software, secrets). Toggling the mode writes an activity log entry (`enabled_gitops_mode` / `disabled_gitops_mode`).",
      analogies: [
        "Like putting a Google Doc in 'suggesting mode': you can see everything, but only approved changes from the repo actually commit.",
        "Like a release branch that's protected from direct pushes. The code path is forced through CI.",
      ],
    },
  },
  {
    slug: "primo-mode",
    term: "Primo mode",
    category: "Fleet concept",
    short:
      "A server config flag (`partnerships.enable_primo`) that simplifies the UI for single-fleet deployments. Hides multi-team UX; treats hosts as if they're all in one 'Unassigned' fleet.",
    seeAlso: ["fleet-premium"],
    technicalHelp: {
      technical:
        "Primo mode is set in the Fleet server YAML under the `partnerships` section (not an admin-UI toggle). When on, the frontend skips the team picker, hides team-scoped buttons on enroll-secret and host modals, and treats every host as part of the default 'Unassigned' pool. The backend data model is unchanged: teams still exist in MySQL. The UX just pretends there's only one. Used in Fleet's partner-tier small-business distributions where multi-tenancy would add noise.",
      analogies: [
        "Like Google Workspace's 'Essentials' tier that strips the admin console down for small teams. Same underlying product, fewer knobs.",
        "Like a simplified ride-share app for drivers who only need one route. The full feature set still exists under the hood.",
      ],
    },
  },
  {
    slug: "rego",
    term: "Rego policy",
    category: "Fleet code",
    short:
      "The language OPA uses. Fleet's authz policies live in server/authz/.",
    seeAlso: ["opa", "authz-authorize"],
    technicalHelp: {
      technical:
        "Instead of scattering 'can this user do X?' checks all over Fleet's Go code, Fleet keeps those rules in a separate pile of Rego files. Rego is a tiny language just for writing allow/deny rules. OPA evaluates those rules on every protected action.",
      analogies: [
        "Like a company's PTO policy document. Instead of each manager inventing their own rules, there's one document everyone follows.",
        "Like the rulebook of a board game: all 'who can do what when' lives in one place and everyone has to agree.",
      ],
    },
  },
  {
    slug: "authz-authorize",
    term: "authz.Authorize",
    category: "Fleet code",
    short:
      "The mandatory first call in every protected service method. Returns an error if the viewer isn't allowed to perform the action on the target.",
    technicalHelp: {
      technical:
        "Every service method starts with a one-liner that asks 'is the logged-in user allowed to do this?' The actual rules come from the Rego policies via OPA; this one-liner is how Fleet asks the question. Forgetting it is a security bug Fleet's tests scan for.",
      analogies: [
        "Like a bouncer at every doorway inside a building, not just the front entrance. Each room still asks 'can you come in here?'",
      ],
    },
  },

  // ─── Apple MDM ───
  {
    slug: "mdm",
    term: "MDM",
    aka: ["Mobile Device Management"],
    category: "Apple MDM",
    short:
      "Protocol family that lets a server (Fleet) install profiles, run commands, lock/wipe, and query a managed device.",
  },
  {
    slug: "apns",
    term: "APNs",
    aka: ["Apple Push Notification Service"],
    category: "Apple MDM",
    short:
      "Apple's push system Fleet uses to wake MDM-enrolled Apple devices. Push carries no payload; it just says 'check in'.",
    technicalHelp: {
      technical:
        "Fleet doesn't contact Apple devices directly. When Fleet wants a device to do something, it sends a push notification through Apple's servers. The push is just a 'ping'. The device wakes up and asks Fleet 'do you have anything for me?' Fleet replies with the actual MDM command.",
      analogies: [
        "Like a doorbell. The ring doesn't say what the delivery is: it just gets the person to come to the door.",
        "Like a voicemail that says 'call me back' with no other info. You call back; then you find out what they want.",
      ],
    },
  },
  {
    slug: "abm",
    term: "Apple Business Manager (ABM)",
    category: "Apple MDM",
    short:
      "Apple's portal where orgs register devices + VPP tokens. Fleet syncs device assignments from ABM.",
    seeAlso: ["dep", "vpp"],
  },
  {
    slug: "dep",
    term: "DEP",
    aka: ["ADE", "Automated Device Enrollment"],
    category: "Apple MDM",
    short:
      "Zero-touch enrollment for Apple devices sold through ABM. Device finds Fleet's MDM URL automatically on first boot.",
    seeAlso: ["abm", "setup-experience"],
    links: [{ label: "Scenario", href: "/scenarios/mac-dep-enrollment" }],
  },
  {
    slug: "ddm",
    term: "DDM",
    aka: ["Declarative Device Management"],
    category: "Apple MDM",
    short:
      "Apple's newer declarative control plane. Fleet declares desired state as JSON declarations; the device computes the diff locally.",
    technicalHelp: {
      technical:
        "Classic MDM is imperative: Fleet sends step-by-step commands ('install this profile', 'remove that one', 'now install this update'). DDM is declarative: Fleet publishes 'here's the state you should be in' and the device figures out how to get there on its own. Scales better for complex setups.",
      analogies: [
        "Classic MDM is like a GPS giving turn-by-turn directions. DDM is like saying 'be at the restaurant by 7pm' and letting the driver pick the route.",
        "Classic MDM is a recipe step by step. DDM is 'make dinner for four' and the chef decides what's for dinner.",
      ],
    },
  },
  {
    slug: "mobileconfig",
    term: ".mobileconfig",
    category: "Apple MDM",
    short:
      "Apple's config profile format. XML plist containing one or more payloads (Wi-Fi, email, restrictions, etc.).",
    seeAlso: ["plist"],
    technicalHelp: {
      technical:
        "A .mobileconfig file is an XML document bundling one or more 'payloads' (Wi-Fi, email, VPN, screen lock, etc). An admin uploads it to Fleet; Fleet delivers it to a Mac/iPhone/iPad via MDM; the OS installs each payload and reports status back.",
      analogies: [
        "Like a care package. One file can contain multiple things (a Wi-Fi password, an email account, a VPN config), and the OS unpacks them all at once.",
        "Like a settings preset you'd share with a friend. But delivered by the admin instead of exported by the user.",
      ],
    },
  },
  {
    slug: "plist",
    term: "plist",
    category: "Apple MDM",
    short:
      "Property list. Apple's XML (sometimes binary) format for structured data. MDM commands + profiles are plists.",
    technicalHelp: {
      technical:
        "A plist is just a structured text (or binary) file that holds settings and data. Apple uses plists everywhere: app preferences, system config, MDM commands. If you've opened Safari's bookmarks under the hood, you've seen plist XML.",
      analogies: [
        "Like a JSON file but in Apple's XML dialect: nested key/value pairs for structured data.",
        "Like Excel for structured data. But the 'cells' are XML tags instead of a spreadsheet grid.",
      ],
    },
  },
  {
    slug: "scep",
    term: "SCEP",
    aka: ["Simple Certificate Enrollment Protocol"],
    category: "Apple MDM",
    short:
      "Protocol for devices to request identity certs. Fleet proxies SCEP to external CAs.",
    seeAlso: ["acme"],
    technicalHelp: {
      technical:
        "A digital certificate is like an ID card for a device: it proves to Wi-Fi, VPN, or email servers 'this really is a company-managed device.' SCEP is the protocol a device uses to ask for one. The actual signing happens at a Certificate Authority. Fleet sits in the middle: the device asks Fleet, Fleet asks the CA, Fleet hands the result back.",
      analogies: [
        "Like a DMV middleman. The device wants an ID card; Fleet relays the request to the DMV (the CA); the DMV issues the card; Fleet hands it to the device.",
      ],
    },
  },
  {
    slug: "acme",
    term: "ACME",
    aka: ["Automatic Certificate Management Environment"],
    category: "Apple MDM",
    short:
      "Newer cert enrollment protocol. Fleet acts as an ACME server for Apple's directory nonce.",
    seeAlso: ["scep"],
    technicalHelp: {
      technical:
        "Same job as SCEP (devices requesting identity certificates), newer design. ACME was popularized by Let's Encrypt for websites. Apple adopted it for device identity recently. Fleet speaks both, since customers have devices on both.",
      analogies: [
        "If SCEP is a 90s fax to the DMV, ACME is a form on a modern website. Same outcome (an ID card for the device), smoother process.",
      ],
    },
  },
  {
    slug: "psso",
    term: "PSSO",
    aka: ["Platform SSO"],
    category: "Apple MDM",
    short:
      "Apple's framework for binding macOS login to a cloud IdP. Fleet brokers the device trust.",
    technicalHelp: {
      technical:
        "Normally your Mac login (the password at the lock screen) is completely separate from your work identity (your Okta/Entra/Google account). Platform SSO ties them together: when you set up a Mac, the first login uses your work identity, so your Mac password IS your work password. Fleet brokers the proof that the Mac is managed and allowed to participate.",
      analogies: [
        "Like unlocking your front door with the same key that gets you into the office.",
      ],
    },
  },
    {
    slug: "nanomdm",
    term: "nanomdm",
    category: "Apple MDM",
    short:
      "Lightweight embedded MDM server library. Owns Fleet's Apple MDM command queue (nano_commands table).",
    technicalHelp: {
      technical:
        "Apple MDM has a lot of fiddly protocol details (enrollment, command queueing, acks, certificate auth). Rather than write all of that from scratch, Fleet uses an open-source library called nanomdm that already handles it. It's embedded inside Fleet, not a separate service.",
      analogies: [
        "Like using an off-the-shelf engine in your car. You design the body and the dashboard; the engine is a trusted part someone else built.",
      ],
    },
  },
  {
    slug: "nanodep",
    term: "nanodep",
    category: "Apple MDM",
    short: "Embedded DEP sync library. Pulls device assignments from ABM.",
    technicalHelp: {
      technical:
        "Sibling of nanomdm, same idea: open-source library doing one specific Apple protocol so Fleet doesn't have to reinvent it. Nanodep's job is talking to Apple Business Manager to find out which devices were assigned to this Fleet instance.",
      analogies: [
        "Like a package tracker that keeps asking Apple 'did any new devices get assigned to us?' and keeps Fleet up to date.",
      ],
    },
  },
  {
    slug: "bootstrap-package",
    term: "Bootstrap package",
    category: "Apple MDM",
    short:
      "A single admin-authored package installed at the end of DEP setup. Usually used to deploy the corporate agent (fleetd).",
  },

  // ─── Microsoft MDM ───
  {
    slug: "entra",
    term: "Entra ID",
    aka: ["Azure AD"],
    category: "Microsoft MDM",
    short:
      "Microsoft's cloud IdP. Issues JWTs for Autopilot. Fleet fetches Entra's JWKS to verify.",
    technicalHelp: {
      technical:
        "Microsoft's identity service, used by most companies for Microsoft 365 single sign-on. If a user has an Entra account, that's who they are at work. Fleet talks to Entra to verify Windows enrollments and to report device compliance.",
      analogies: [
        "Think of it as the 'login with Microsoft' button on everything, but it's also the backend that decides who you are at work.",
      ],
    },
  },
  {
    slug: "autopilot",
    term: "Autopilot",
    category: "Microsoft MDM",
    short:
      "Zero-touch provisioning for Windows. Device gets MDM server info at first boot after Entra sign-in.",
    technicalHelp: {
      technical:
        "The Windows equivalent of DEP. IT buys laptops and registers their serial numbers with Autopilot. When an employee powers on a brand-new laptop, Windows sees it's registered, prompts for the user's work login, and automatically enrolls into Fleet without an IT admin ever touching the device.",
      analogies: [
        "Zero-touch unboxing: open the box, sign in, done. No IT visit, no USB sticks, no imaging.",
      ],
    },
  },
  {
    slug: "wstep",
    term: "WSTEP",
    category: "Microsoft MDM",
    short:
      "Windows cert enrollment protocol. Fleet implements the server side.",
    technicalHelp: {
      technical:
        "Microsoft's version of SCEP/ACME: how a Windows device, during enrollment, asks for an identity certificate from Fleet. Fleet speaks WSTEP so Windows devices can get the cert they need to prove themselves to Wi-Fi, VPN, and email.",
      analogies: [
        "Same role as SCEP, but for Windows enrollment specifically.",
      ],
    },
  },
  {
    slug: "syncml",
    term: "SyncML",
    aka: ["OMA-DM"],
    category: "Microsoft MDM",
    short:
      "Microsoft's XML MDM protocol (actually a legacy standard). All Windows MDM commands and acks are SyncML documents.",
    technicalHelp: {
      technical:
        "SyncML is a verbose XML format. Every MDM command (install profile, set BitLocker, run script) and every device response is one SyncML document. It is older and chattier than Apple's plist format, and malformed XML fails the whole sync.",
      analogies: [
        "Think of SyncML like a strict paper form. Every field has to be filled in exactly right; one typo and the whole form gets rejected.",
        "If a modern API is a text message, SyncML is a formal letter: longer, more structured, and no shortcuts.",
      ],
    },
  },
  {
    slug: "csp",
    term: "CSP",
    aka: ["Configuration Service Provider"],
    category: "Microsoft MDM",
    short:
      "A Windows setting tree. Each setting maps to a CSP path like ./Device/Vendor/MSFT/Policy/...",
    technicalHelp: {
      technical:
        "Windows exposes every manageable setting (Wi-Fi config, BitLocker on/off, update deadlines, screen lock timeout, hundreds more) as a leaf in a hierarchical tree. To change a setting from an MDM server, you POST a SyncML command that targets a specific path in that tree. Fleet never 'sets Wi-Fi' directly; it tells Windows 'go to this address in your setting tree and write this value here.'",
      analogies: [
        "Like a filing cabinet with drawers inside drawers: Fleet hands Windows an address (./Device/Vendor/MSFT/Policy/Config/Wi-Fi) and a value, and Windows walks down the drawers to put the value in the right place.",
        "Like a URL. ./Device/Vendor/MSFT/Policy/... points to one specific setting the same way https://example.com/users/42/email points to one specific thing on a website.",
      ],
    },
  },
  {
    slug: "conditional-access",
    term: "Conditional access",
    category: "Microsoft MDM",
    short:
      "Entra feature that gates access to apps on device compliance. Fleet can act as compliance partner.",
    technicalHelp: {
      technical:
        "Microsoft Entra can refuse to let a user open an app (like Outlook) unless the device meets certain rules. Normally those rules come from Intune. Fleet can plug into this system as a 'compliance partner': Entra asks Fleet 'is this device compliant?' Fleet answers yes/no based on your Fleet policies.",
      analogies: [
        "Like a bouncer at a club who checks your ID AND your dress code. The ID check is sign-in; the dress code is 'is your device managed and healthy per Fleet?'",
      ],
    },
  },

  // ─── Android MDM ───
  {
    slug: "amapi",
    term: "AMAPI",
    aka: ["Android Management API"],
    category: "Android MDM",
    short:
      "Google's cloud API Fleet uses to manage Android Enterprise devices. Fleet never talks to Android devices directly.",
    technicalHelp: {
      technical:
        "On Apple and Windows, Fleet talks straight to the device. On Android, Google insists on sitting in the middle. Fleet calls Google's API; Google figures out how to deliver the request to the actual Android phone and reports the result back to Fleet.",
      analogies: [
        "Like having to go through a receptionist to reach everyone in a building, while in other buildings you can walk straight to the person's desk.",
      ],
    },
  },
  {
    slug: "emm",
    term: "EMM",
    aka: ["Enterprise Mobility Management"],
    category: "Android MDM",
    short:
      "The role Fleet plays with Google. Fleet is a Google-recognized EMM partner.",
    technicalHelp: {
      technical:
        "To manage Android work devices, a company needs an approved EMM. Google decides which services qualify. Fleet is on Google's approved EMM list, which is what lets it participate in Android Enterprise at all.",
      analogies: [
        "Like a licensed pharmacist. You can't dispense medicine as a hobby; you have to be approved by the licensing board. Google plays the licensing board for Android management.",
      ],
    },
  },
  {
    slug: "google-pubsub",
    term: "Google Pub/Sub",
    category: "Android MDM",
    short:
      "Google's messaging service. Fleet subscribes to receive AMAPI-originated events (enrollment, status, command results).",
    technicalHelp: {
      technical:
        "A messaging bus hosted by Google. Fleet subscribes to a channel; Google drops messages onto that channel whenever something happens on an Android device (enrollment, command result, etc.); Fleet picks them up and acts on them.",
      analogies: [
        "Like a mailbox Fleet checks: Google drops a note in whenever an Android device does something, Fleet reads it when it comes by.",
      ],
    },
  },
  {
    slug: "work-profile",
    term: "Work profile",
    category: "Android MDM",
    short:
      "Android BYOD container. Work apps live separately from personal apps on the same device.",
    technicalHelp: {
      technical:
        "On an employee-owned Android phone, Fleet (via Google) creates a separate 'work' area with its own apps, email, and storage. Personal data stays private to the employee; the company only manages the work area.",
      analogies: [
        "Like having a lockable briefcase inside your purse. Your personal stuff is in the purse (your business); the briefcase holds work stuff that your employer controls.",
      ],
    },
  },

  // ─── Backend / Go ───
  {
    slug: "goroutine",
    term: "goroutine",
    category: "Backend / Go",
    short:
      "Lightweight thread run by the Go runtime. Starts with `go functionName()`. Fleet has many; crons and workers each live in their own.",
    technicalHelp: {
      technical:
        "Fleet does lots of things at the same time: handle HTTP requests, run cron jobs, drain worker queues, stream live-query results. Each of those runs in its own goroutine. Think of a goroutine as a tiny worker Fleet can spawn cheaply when it has a job to do in parallel.",
      analogies: [
        "Like opening another browser tab to do something while the current page is still loading. Each tab works on its own without blocking the others.",
      ],
    },
  },
  {
    slug: "channel",
    term: "channel",
    category: "Backend / Go",
    short:
      "Go concurrency primitive. Typed FIFO for passing values between goroutines, with optional buffering and close semantics.",
    technicalHelp: {
      technical:
        "When several goroutines need to pass work (or results) between each other, they use a channel. One goroutine puts an item in; another takes it out. First-in, first-out.",
      analogies: [
        "Like a conveyor belt in a kitchen: the prep cook drops an order on the belt, the line cook picks it up at the other end.",
        "Like a shared whiteboard: one person writes a request on it, someone else reads and handles it.",
      ],
    },
  },
  {
    slug: "context",
    term: "context.Context",
    category: "Backend / Go",
    short:
      "Request-scoped value carrier + cancellation. Fleet passes ctx through every method. Carries the viewer, trace span, deadline.",
    technicalHelp: {
      technical:
        "Every HTTP request to Fleet gets a context object. It carries 'who's logged in', 'what's the deadline for this request', and 'has the client given up and disconnected?' All deep code checks the context so that if the user closed the tab, Fleet can stop doing work on their behalf.",
      analogies: [
        "Like a hall pass you carry with you through a school building. Every room you enter can check it: 'this is Alice, she has 10 more minutes, if she gets called back cancel what you're doing for her.'",
      ],
    },
  },
  {
    slug: "interface",
    term: "interface",
    category: "Backend / Go",
    short:
      "Go's duck-typed abstraction. If a struct has the required methods, it satisfies the interface. Fleet uses interfaces for Service and Datastore.",
    technicalHelp: {
      technical:
        "A Go interface is a 'required abilities list'. For example, Fleet's Datastore interface says 'anything that calls itself a datastore must be able to list hosts, create a policy, update a report', etc. The real MySQL implementation satisfies that. In tests, a fake implementation satisfies it. The business logic doesn't care which. It just calls the methods.",
      analogies: [
        "Like a job description: 'to be a chef here you must be able to cook, bake, and plate.' Any human who can do those three things is a chef; the restaurant doesn't care which culinary school they went to.",
      ],
    },
  },
  {
    slug: "struct-embedding",
    term: "Struct embedding",
    category: "Backend / Go",
    short:
      "Go's composition pattern. A struct can embed another struct and get all its methods. Fleet uses this for the ee wrapper.",
    technicalHelp: {
      technical:
        "In Go, if a struct embeds another struct, it automatically gets that struct's methods for free, with the option to override any of them. Fleet uses this to add Enterprise behaviors on top of the core: the Enterprise service embeds the core service, then overrides specific methods to add license checks.",
      analogies: [
        "Like buying a car and bolting a trailer on. The whole package still drives like the car, but you've added one extra capability without rebuilding the engine.",
      ],
    },
  },
  {
    slug: "gorilla-mux",
    term: "gorilla/mux",
    category: "Backend / Go",
    short: "The HTTP router Fleet uses. Path templates, method matching.",
    technicalHelp: {
      technical:
        "When Fleet gets an HTTP request, something has to figure out 'which URL does this match and which function handles it?' gorilla/mux is the open-source library Fleet uses for that job.",
      analogies: [
        "Like a receptionist who looks at the person walking in and says 'you want accounting, go to the third floor.'",
      ],
    },
  },
  {
    slug: "goose",
    term: "goose",
    category: "Backend / Go",
    short:
      "The MySQL migration tool Fleet uses. Migrations live in server/datastore/mysql/migrations/.",
    technicalHelp: {
      technical:
        "When Fleet needs to change its database schema (add a column, create a table), that change is written as a file called a 'migration'. Goose is the tool that applies migrations in order so every Fleet instance ends up with the same schema.",
      analogies: [
        "Like a numbered list of renovation steps: 'first expand the kitchen, then add a bathroom.' Each step depends on the previous one being done.",
      ],
    },
  },
  {
    slug: "sqlx",
    term: "sqlx",
    category: "Backend / Go",
    short:
      "A SQL library with named parameters and typed struct scanning. Fleet uses it; there is no ORM.",
    technicalHelp: {
      technical:
        "Fleet writes raw SQL instead of using a 'translate my Go code into SQL' library (ORM). sqlx is a thin helper on top of Go's standard SQL support: it just makes querying + mapping results into Go structs less tedious.",
      analogies: [
        "Fleet talks to MySQL in actual SQL, like writing a letter in the recipient's native language instead of using Google Translate.",
      ],
    },
  },
  {
    slug: "opa",
    term: "OPA",
    aka: ["Open Policy Agent"],
    category: "Backend / Go",
    short:
      "A policy engine. Fleet evaluates Rego policies in-process via OPA to make authz decisions.",
    seeAlso: ["rego"],
    technicalHelp: {
      technical:
        "OPA is the engine that reads Rego files and answers 'is this allowed?' questions. Fleet runs OPA inside the Fleet process. Every time a service method runs, it asks OPA 'can this user do this action on this thing?' and OPA says yes or no based on the Rego rules.",
      analogies: [
        "Like a security guard with a rulebook who stands at every door and checks before letting anyone through.",
      ],
    },
  },
  {
    slug: "iterator",
    term: "iter.Seq2 (iterator)",
    category: "Backend / Go",
    short:
      "Go 1.23's range-over-function pattern. Fleet uses it for streaming datastore reads (StreamHosts).",
    technicalHelp: {
      technical:
        "If Fleet needs to send 50,000 hosts to a client, it can either (a) fetch all 50,000 into memory then send, or (b) stream them one at a time as it reads from MySQL. The second one uses iterators: start reading, hand one row to the caller, repeat until done. Scales much better.",
      analogies: [
        "Like reading a book one page at a time (iterator) vs. ripping all the pages out and holding them in one big stack (load-everything).",
      ],
    },
  },

  // ─── Standards & protocols ───
  {
    slug: "saml",
    term: "SAML",
    category: "Standards & protocols",
    short:
      "XML-based SSO. IdP posts a signed assertion to Fleet's ACS endpoint; Fleet verifies the signature.",
    technicalHelp: {
      technical:
        "An older single sign-on protocol. The identity provider (Okta, Entra, Google) proves who you are by sending Fleet a signed statement: 'this is firstname@fleetdm.com, trust me, here's my signature.' Fleet checks the signature and lets you in.",
      analogies: [
        "Like a notarized letter. The notary (the IdP) stamps the letter so the person reading it (Fleet) knows it's really from the person it claims to be from.",
      ],
    },
  },
  {
    slug: "oidc",
    term: "OIDC",
    aka: ["OpenID Connect"],
    category: "Standards & protocols",
    short: "OAuth2-based SSO. The modern alternative to SAML.",
    technicalHelp: {
      technical:
        "Newer single sign-on standard. Same job as SAML (prove who the user is) but uses modern web tech (JSON web tokens) instead of SAML's XML. 'Log in with Google' / 'Log in with GitHub' buttons on apps use OIDC under the hood.",
      analogies: [
        "If SAML is the notarized letter, OIDC is the DocuSign email: same proof, newer paper.",
      ],
    },
  },
  {
    slug: "jwks",
    term: "JWKS",
    aka: ["JSON Web Key Set"],
    category: "Standards & protocols",
    short:
      "A URL hosted by an IdP that serves its public signing keys. Fleet fetches Entra's JWKS to verify Autopilot JWTs.",
    technicalHelp: {
      technical:
        "The identity provider publishes its public keys at a known URL. Anyone who wants to verify 'is this JWT really from Entra?' downloads those keys and checks the signature. Fleet does this during Windows Autopilot enrollment.",
      analogies: [
        "Like checking a signature by comparing it to a 'see our signatures here' page the bank publishes.",
      ],
    },
  },
  {
    slug: "jwt",
    term: "JWT",
    aka: ["JSON Web Token"],
    category: "Standards & protocols",
    short:
      "A signed JSON payload. Used by Autopilot sign-in; verified against JWKS.",
    technicalHelp: {
      technical:
        "A JWT is a small piece of signed JSON that says things like 'this user is firstname@fleetdm.com, issued at 10am, expires at noon.' Services pass JWTs around to say who someone is without re-asking them to log in. Signed by the IdP so no one can forge one.",
      analogies: [
        "Like a wristband at a conference. You showed ID once at check-in; now the wristband gets you into any session for the day.",
      ],
    },
  },
  {
    slug: "scim",
    term: "SCIM",
    aka: ["System for Cross-domain Identity Management"],
    category: "Standards & protocols",
    short:
      "Protocol for an IdP to push user/group lifecycle to downstream apps. Fleet is a SCIM target.",
    technicalHelp: {
      technical:
        "SSO (SAML/OIDC) is about proving who you are when you log in. SCIM is a step before that: the identity provider (Okta/Entra) tells Fleet 'here are all our employees' names and emails and groups' so Fleet already knows about them before they ever log in. When an employee leaves, SCIM also tells Fleet 'remove this one'.",
      analogies: [
        "SSO = flashing your badge at the door. SCIM = the HR system pushing the new-hire list to the badge system a week ahead of time.",
      ],
    },
  },
  {
    slug: "tuf",
    term: "TUF",
    aka: ["The Update Framework"],
    category: "Standards & protocols",
    short:
      "A spec for secure software updates. fleetd uses TUF to auto-update its components.",
    technicalHelp: {
      technical:
        "Software updates are a famous attack target: trick the auto-updater into installing malware and you own the device. TUF is a security recipe for update servers so that even if some signing keys are compromised, attackers can't push bad updates to clients. Fleet's agent (fleetd) uses TUF to auto-update itself.",
      analogies: [
        "Like a tamper-evident seal on a prescription bottle: even if the drugstore gets robbed, you can tell when the bottle on the shelf isn't the original.",
      ],
    },
  },
  {
    slug: "websocket",
    term: "WebSocket",
    category: "Standards & protocols",
    short:
      "Bi-directional HTTP connection. Fleet uses one per live-query campaign to stream rows to the admin UI.",
    technicalHelp: {
      technical:
        "A normal web request is one-shot: you ask, you get an answer, done. A WebSocket is an open phone line that stays connected so either side can push data at any time. Fleet uses it for live queries so results stream in as hosts respond.",
      analogies: [
        "Normal HTTP is like texting back and forth. WebSocket is like a phone call: once connected, either side can speak whenever.",
      ],
    },
  },

  // ─── Database ───
  {
    slug: "replica-lag",
    term: "Replica lag",
    category: "Database",
    short:
      "The delay before a write to the primary MySQL is visible on a read replica. Causes 'I just saved it, why can't I read it' bugs.",
    seeAlso: ["dbreader-dbwriter"],
    technicalHelp: {
      technical:
        "Fleet has one main database for writes and copies ('replicas') for reads. Updates get copied to replicas within milliseconds, but not instantly. If you save a change and immediately read, the read might come from a replica that hasn't caught up yet and show the old value.",
      analogies: [
        "Like editing a Google Doc on your laptop and checking on your phone a second later. Sometimes the phone shows the pre-edit version for a moment before syncing.",
      ],
    },
  },
  {
    slug: "dbreader-dbwriter",
    term: "DBReader / DBWriter",
    category: "Database",
    short:
      "Fleet's abstractions for the replica (read) vs primary (write) connections. Writes always go to DBWriter.",
    seeAlso: ["replica-lag"],
    technicalHelp: {
      technical:
        "Fleet code never says 'open a database connection'. It says 'give me a reader' or 'give me a writer.' Reader connections go to replicas (fast, maybe stale). Writer connections go to the main database (slower, always current). Engineers pick based on whether staleness is OK.",
      analogies: [
        "Like deciding whether to look up something in a photocopy of a book (fast, maybe out of date) or go to the master copy at the library (slower, always current).",
      ],
    },
  },
  {
    slug: "migration",
    term: "Migration",
    category: "Database",
    short:
      "A timestamped Up/Down SQL change in server/datastore/mysql/migrations/. Run on startup by goose.",
  },

  // ─── Vuln / Security ───
  {
    slug: "cpe",
    term: "CPE",
    aka: ["Common Platform Enumeration"],
    category: "Vuln / Security",
    short:
      "A canonical software identifier. Vuln pipeline matches installed software's CPE against vulnerable CPEs in feeds.",
    technicalHelp: {
      technical:
        "Every piece of software has a canonical identifier (CPE) so different databases can agree on what 'Chrome 120' means. When Fleet learns that CVE-2024-xxxx affects chrome 120, it compares that CPE against the CPE of software installed on each host to decide if the host is vulnerable.",
      analogies: [
        "Like a UPC barcode. Different stores agree on what '0 12345 67890 5' is so they can all track the same product.",
      ],
    },
  },
  {
    slug: "cve",
    term: "CVE",
    aka: ["Common Vulnerabilities and Exposures"],
    category: "Vuln / Security",
    short:
      "A canonical identifier for a security vulnerability. CVE-YYYY-NNNNN.",
    technicalHelp: {
      technical:
        "Every known security vulnerability gets assigned a globally unique ID like CVE-2024-12345. The ID itself is just a name; the details (what's broken, how bad, which versions) come from various feeds. Fleet uses the ID to match against installed software and show you which hosts are affected.",
      analogies: [
        "Like a case number assigned to a problem: 'CVE-2024-12345' is the case; the rest of the details are in the file.",
      ],
    },
  },
  {
    slug: "cvss",
    term: "CVSS",
    aka: ["Common Vulnerability Scoring System"],
    category: "Vuln / Security",
    short:
      "The 0-10 severity score attached to a CVE. Fleet surfaces it in cve_meta.",
    technicalHelp: {
      technical:
        "Not every vulnerability is equally dangerous. CVSS is a scoring system: a vuln gets a 0-10 score based on how exploitable it is, what access it needs, and the damage it can do. 9+ is critical; 7-8 is high; etc.",
      analogies: [
        "Like a hurricane category: a 'Category 5' tells you to evacuate; a 'Category 1' tells you to secure your lawn chairs.",
      ],
    },
  },
  {
    slug: "nvd",
    term: "NVD",
    aka: ["National Vulnerability Database"],
    category: "Vuln / Security",
    short:
      "The US NIST canonical CVE feed. Fleet's primary vuln source.",
    technicalHelp: {
      technical:
        "The US government's official database of known software vulnerabilities, maintained by NIST. Fleet downloads this list regularly so it knows what to look for when scanning your hosts.",
      analogies: [
        "Like the FBI's most-wanted list for software bugs. If a vuln's on the list, Fleet knows to look for it.",
      ],
    },
  },
  {
    slug: "osv",
    term: "OSV",
    aka: ["Open Source Vulnerabilities"],
    category: "Vuln / Security",
    short:
      "Google-maintained cross-ecosystem vuln feed. Covers packages (npm, PyPI, Go modules) better than NVD.",
    technicalHelp: {
      technical:
        "A different vulnerability database, run by Google, that's especially good at covering open-source library ecosystems (npm for JavaScript, PyPI for Python, Go modules, etc.). Fleet reads both NVD and OSV to get fuller coverage.",
      analogies: [
        "If NVD is the government's bug list, OSV is the open-source community's bug list. Fleet reads both because neither alone is complete.",
      ],
    },
  },
  {
    slug: "oval",
    term: "OVAL",
    category: "Vuln / Security",
    short:
      "Linux distro vuln feeds (Red Hat, Ubuntu, Debian, SUSE). Vulnerability pipeline consumes OVAL per distro.",
    technicalHelp: {
      technical:
        "Each Linux distribution (Red Hat, Ubuntu, etc.) publishes its own authoritative list of which versions of its packages are vulnerable to which CVEs, in a format called OVAL. Fleet reads these distro-specific feeds to decide if a Linux host is affected.",
      analogies: [
        "Each car manufacturer publishes their own recall list. If you own a Toyota, you check Toyota's; a Ford, you check Ford's. OVAL is that, for Linux distros.",
      ],
    },
  },
  {
    slug: "msrc",
    term: "MSRC",
    aka: ["Microsoft Security Response Center"],
    category: "Vuln / Security",
    short:
      "Microsoft's vuln feed. Fleet uses it for Windows and Office vuln matching.",
    technicalHelp: {
      technical:
        "Microsoft's own list of security bugs in Windows, Office, and other Microsoft software. The authoritative place to find out which Microsoft products are affected by each CVE and which patch fixes it.",
      analogies: [
        "Like reading a product recall straight from the manufacturer. If it's a Microsoft product, Microsoft's list is the source of truth.",
      ],
    },
  },
];
