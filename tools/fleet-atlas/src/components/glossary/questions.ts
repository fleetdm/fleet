// Two questions per glossary term. Difficulty 1-10.
// Difficulty rubric:
//   1-2 : "What does X stand for / what is X" basic recognition
//   3-4 : "Which of these describes X" short definition recall
//   5-6 : Scenario / mechanism / 'where does X live'
//   7-8 : Failure modes, interactions with other terms, design tradeoffs
//   9-10: Subtle gotchas or implementation nuances

export type QuizQuestion = {
  termSlug: string;
  difficulty: number; // 1-10
  question: string;
  options: string[];
  correctIndex: number;
  explanation?: string;
};

export const QUESTIONS: QuizQuestion[] = [
  // ─── Fleet concept ───
  { termSlug: "fleet", difficulty: 2, question: "In new Fleet UI copy and user documentation, which term should you use?", options: ["team", "fleet", "group", "org"], correctIndex: 1, explanation: "Product renamed teams → fleets. UI copy says fleet; code and DB still say team." },
  { termSlug: "fleet", difficulty: 6, question: "You're adding a column that links a host to the group it belongs to. What column name should the migration use?", options: ["fleet_id", "team_id", "group_id", "org_id"], correctIndex: 1, explanation: "Legacy schema uses team_id. Match existing columns; don't rename piecemeal. There is no `fleet_id` column in Fleet's schema." },

  { termSlug: "report", difficulty: 3, question: "What does Fleet now call what used to be a 'saved query'?", options: ["report", "run", "search", "scan"], correctIndex: 0 },
  { termSlug: "report", difficulty: 5, question: "In current Fleet terminology, 'query' refers to...", options: ["a saved recurring scan that runs on a scheduled interval and keeps results", "the SQL text. Reused across reports, policies, pre-install checks, and live queries", "an admin-triggered live search dispatched to a host set and streamed back", "a row of data osquery returns in response to a scheduled or ad-hoc search"], correctIndex: 1 },

  { termSlug: "live-query", difficulty: 3, question: "What's the main difference between a live query and a scheduled report?", options: ["Live queries are faster", "Live queries don't persist results in query_results; they stream through Redis pubsub", "Live queries only run on Macs", "Scheduled reports require more permissions"], correctIndex: 1 },
  { termSlug: "live-query", difficulty: 7, question: "A live query is sent to 200 hosts across 4 Fleet instances. Why does Redis pubsub matter?", options: ["Fleet instances share cron locks and rate-limit counters via Redis, so only one instance runs each tick", "Any instance can receive a host result, and any instance may hold the admin's WebSocket; pubsub bridges them", "Fleet publishes results via a Redis-backed Server-Sent Events stream the admin's browser subscribes to directly", "Redis caches the campaign SQL + target host list so hosts joining mid-campaign can pick up the work"], correctIndex: 1 },

  { termSlug: "policy", difficulty: 3, question: "How does a policy decide if a host is passing or failing?", options: ["Boolean return value", "Row count of its SQL query result; zero rows = failing", "Status code from fleetd", "A Rego rule"], correctIndex: 1 },
  { termSlug: "policy", difficulty: 6, question: "If a policy has auto-install configured and a host starts failing, which cron reads the failing membership and queues the install?", options: ["integrations", "automations", "frequent_cleanups", "vulnerabilities"], correctIndex: 1 },

  { termSlug: "label", difficulty: 2, question: "A label groups hosts dynamically via...", options: ["Manual admin assignment", "A SQL query evaluated per host", "IP-range matching", "Hostname pattern matching"], correctIndex: 1 },
  { termSlug: "label", difficulty: 6, question: "Which table records label membership after evaluation?", options: ["hosts_labels", "label_membership", "label_hosts", "label_targets"], correctIndex: 1, explanation: "`label_membership` is the real join table. `hosts_labels` and `label_hosts` are not Fleet tables. Plausible-sounding names, but neither exists." },

  { termSlug: "host", difficulty: 1, question: "A 'host' in Fleet is...", options: ["A Fleet server instance", "A managed device", "An admin user", "An IP address"], correctIndex: 1 },
  { termSlug: "host", difficulty: 4, question: "On which platforms can a Fleet host run?", options: ["macOS and Windows only", "macOS, Windows, Linux", "macOS, Windows, Linux, iOS, iPadOS, Android, ChromeOS", "macOS only"], correctIndex: 2 },

  { termSlug: "fleetd", difficulty: 3, question: "fleetd is Fleet's...", options: ["Server binary (that's the `fleet` binary; `fleetd` is the on-device agent)", "Agent installed on managed devices; wraps osqueryd + Fleet Desktop", "Node.js CLI (operator CLI tooling is `fleetctl`, a Go binary, not a Node.js tool)", "Cron scheduler (crons run inside the Fleet server process, not in a separate scheduler binary)"], correctIndex: 1 },
  { termSlug: "fleetd", difficulty: 6, question: "What is fleetd's prior name?", options: ["Osprey", "Orbit", "fleetwrap", "fdctl"], correctIndex: 1 },

  { termSlug: "osqueryd", difficulty: 2, question: "osqueryd is...", options: ["A proprietary Fleet daemon", "The open-source osquery daemon, bundled inside fleetd", "A database driver", "An MDM handler"], correctIndex: 1 },
  { termSlug: "osqueryd", difficulty: 5, question: "What does osqueryd use SQL to query?", options: ["Fleet's MySQL", "Local device state (processes, users, installed packages, etc.)", "JOINs across hosts", "Redis"], correctIndex: 1 },

  { termSlug: "fleet-desktop", difficulty: 2, question: "Fleet Desktop is...", options: ["A required admin UI", "An optional end-user app giving a 'My Device' page via the system tray/menu bar", "A cron scheduler", "A replacement for fleetd"], correctIndex: 1 },
  { termSlug: "fleet-desktop", difficulty: 5, question: "Fleet Desktop runs as...", options: ["A separate binary launched by Orbit", "A plugin inside osqueryd", "A web server inside orbit", "A browser extension"], correctIndex: 0 },

  { termSlug: "fleetctl", difficulty: 2, question: "fleetctl is...", options: ["A dashboard UI", "The Fleet admin CLI client", "A MySQL migration tool", "An osquery extension"], correctIndex: 1 },
  { termSlug: "fleetctl", difficulty: 6, question: "In GitOps mode, fleetctl...", options: ["Runs scheduled queries locally", "Diffs YAML desired-state against Fleet's current state and reconciles", "Scans the local filesystem", "Opens a WebSocket to admins"], correctIndex: 1 },

  { termSlug: "gitops", difficulty: 3, question: "What does Fleet's GitOps mode treat as the source of truth?", options: ["The admin UI state", "Declarative YAML in a git repo", "The fleetctl config file", "MySQL dumps"], correctIndex: 1 },
  { termSlug: "gitops", difficulty: 7, question: "A GitOps apply differs from an admin button click because it...", options: ["Writes directly to the per-resource MySQL tables, skipping the service layer's validation and authz", "Can also REMOVE software/profiles that aren't declared (desired-state reconciliation)", "Runs reconciliation asynchronously via a cron, so GitOps changes take ~2 minutes to reach MySQL", "Returns a dry-run plan by default; the --apply flag is required to actually write any changes"], correctIndex: 1 },

  { termSlug: "fma", difficulty: 2, question: "A Fleet-maintained app (FMA) is curated from...", options: ["A proprietary Fleet catalog", "Homebrew casks (macOS) and winget packages (Windows)", "GitHub releases", "The Mac App Store"], correctIndex: 1 },
  { termSlug: "fma", difficulty: 6, question: "The maintained_apps cron rebuilds the FMA catalog how often by default?", options: ["Every minute", "Every hour", "Every 24 hours (nightly)", "Weekly"], correctIndex: 1 },

  { termSlug: "vpp", difficulty: 2, question: "VPP stands for...", options: ["Virtual Private Permissions", "Volume Purchase Program (Apple's App Store volume licensing)", "Vendor Package Protocol", "Verified Push Protocol"], correctIndex: 1 },
  { termSlug: "vpp", difficulty: 7, question: "When a VPP install is triggered, Fleet...", options: ["Streams the app binary to the device", "Assigns the license via Apple's VPP API; Apple delivers the app", "Uses osquery to download it", "Calls AMAPI"], correctIndex: 1 },

  { termSlug: "self-service", difficulty: 3, question: "Self-service installs use which auth mechanism?", options: ["Admin session token", "Device token (per-host, in the URL)", "OAuth2 from the end user", "SAML assertion"], correctIndex: 1 },
  { termSlug: "self-service", difficulty: 6, question: "Which flag must be set on a software title for it to appear in self-service?", options: ["is_public", "self_service=true", "is_premium", "auto_install=true"], correctIndex: 1, explanation: "The real flag is `self_service` on the title. `is_public`, `is_premium`, and `auto_install` are not real Fleet software-title fields." },

  { termSlug: "setup-experience", difficulty: 3, question: "Setup experience is the window between...", options: ["Initial boot and user login", "MDM enrollment and user login on a Mac", "VPP sync and first install", "Policy failure and automation"], correctIndex: 1 },
  { termSlug: "setup-experience", difficulty: 7, question: "What type of blocking happens during setup experience?", options: ["A confirmation modal in the UI", "The Mac is paused in 'awaiting configuration' until Fleet releases it via DDM", "Nothing is blocked; everything is parallel", "osquery runs are deferred"], correctIndex: 1 },

  { termSlug: "enroll-secret", difficulty: 2, question: "An enroll secret is...", options: ["A per-host token issued at enroll", "A pre-shared secret a host presents to join a fleet", "A database password", "An API bearer token"], correctIndex: 1 },
  { termSlug: "enroll-secret", difficulty: 5, question: "An enroll secret is rotatable at which level?", options: ["Per-host", "Per-fleet", "Per-user", "Per-session"], correctIndex: 1 },

  { termSlug: "node-key", difficulty: 3, question: "A node key is...", options: ["A Fleet admin's session token", "A per-host bearer token Fleet issues at enroll", "An SSH key", "A device certificate"], correctIndex: 1 },
  { termSlug: "node-key", difficulty: 6, question: "When does a host's node key rotate?", options: ["Every 10 seconds", "Daily", "On re-enroll", "Only on manual admin action"], correctIndex: 2 },

  { termSlug: "device-token", difficulty: 3, question: "A device token authenticates requests to...", options: ["/api/v1/fleet/* admin endpoints", "/device/* end-user endpoints (My Device page)", "/mdm/* MDM protocol endpoints", "/api/osquery/* osquery endpoints"], correctIndex: 1 },
  { termSlug: "device-token", difficulty: 6, question: "How is the device token different from the node key?", options: ["They're the same thing", "Node key = osquery TLS auth; device token = end-user My Device auth", "Device token is short-lived; node key is permanent", "Only MDM-enrolled devices get a device token"], correctIndex: 1 },

  { termSlug: "fleet-variable", difficulty: 3, question: "What does $FLEET_VAR_* do in a config profile?", options: ["Nothing; cosmetic", "Gets substituted per host at delivery time", "Raises a syntax error on load", "References an environment variable from the host"], correctIndex: 1 },
  { termSlug: "fleet-variable", difficulty: 6, question: "Which table catalogues fleet variables?", options: ["config_vars", "fleet_variables", "placeholder_vars", "template_vars"], correctIndex: 1 },

  { termSlug: "secret-variable", difficulty: 3, question: "Secret variables differ from fleet variables because...", options: ["They're uppercase", "Their values are encrypted at rest and not readable back out", "They're only for OIDC", "They only work in scripts"], correctIndex: 1 },
  { termSlug: "secret-variable", difficulty: 5, question: "After an admin sets a secret variable value, can they view it later in the UI?", options: ["Yes, in plaintext", "No; values aren't readable back out", "Only if they're superadmin", "Only via API, not UI"], correctIndex: 1 },

  { termSlug: "activity", difficulty: 3, question: "An activity is...", options: ["A scheduled report", "An audit log entry for admin actions", "A notification", "A live-query result"], correctIndex: 1 },
  { termSlug: "activity", difficulty: 7, question: "Which service method writes activity rows?", options: ["svc.LogActivity", "svc.NewActivity", "svc.AuditLog", "svc.RecordAction"], correctIndex: 1 },

  { termSlug: "activity-streaming", difficulty: 3, question: "Activity streaming forwards activity rows to...", options: ["MySQL", "A configured webhook or SIEM endpoint", "Fleet Desktop", "osqueryd"], correctIndex: 1 },
  { termSlug: "activity-streaming", difficulty: 6, question: "Which cron drains the streaming queue?", options: ["activities_streaming (every 5 minutes)", "frequent_cleanups (every 15 minutes, clears expired tokens and distributed locks)", "integrations (every 1 minute, drains Jira/Zendesk/VPP worker jobs)", "usage_statistics (every hour, reports anonymized stats to Fleet)"], correctIndex: 0 },

  { termSlug: "carve", difficulty: 3, question: "A carve is...", options: ["A software uninstall", "A file osquery uploads from a host in chunks", "A redacted log entry", "An MDM wipe command"], correctIndex: 1 },
  { termSlug: "carve", difficulty: 6, question: "Carve data is stored in...", options: ["The carves table", "carve_metadata + carve_blocks", "MySQL BLOB columns on hosts", "S3 only"], correctIndex: 1 },

  // ─── Fleet code ───
  { termSlug: "fleet-service", difficulty: 3, question: "fleet.Service is...", options: ["A Go struct in server/service that holds datastore and config pointers", "A Go interface declared in server/fleet/service.go", "A go-kit Endpoint factory that wires HTTP decoders to datastore methods", "The route registration function called from main to wire HTTP paths to handlers"], correctIndex: 1 },
  { termSlug: "fleet-service", difficulty: 6, question: "The service layer calls which other layer for persistent data?", options: ["MySQL directly", "The fleet.Datastore interface", "Redis", "A REST client"], correctIndex: 1 },

  { termSlug: "fleet-datastore", difficulty: 3, question: "fleet.Datastore is...", options: ["The MySQL instance", "The Go interface for all storage operations", "A schema file", "A go-kit endpoint"], correctIndex: 1 },
  { termSlug: "fleet-datastore", difficulty: 6, question: "In tests, fleet.Datastore is typically substituted with...", options: ["A Postgres instance", "A mock generated into server/mock/", "An SQLite file", "The real MySQL impl"], correctIndex: 1 },

  { termSlug: "ctxerr", difficulty: 3, question: "What does ctxerr.Wrap do?", options: ["Wraps an HTTP response", "Wraps an error with context + stack for debuggable logs", "Wraps a mutex", "Wraps a Redis call"], correctIndex: 1 },
  { termSlug: "ctxerr", difficulty: 5, question: "Fleet convention: instead of 'return err' from a datastore method, you should...", options: ["Return nil and log instead", "Return ctxerr.Wrap(ctx, err, \"short description\")", "Panic", "Send to Sentry directly"], correctIndex: 1 },

  { termSlug: "makehandler", difficulty: 3, question: "MakeHandler is...", options: ["A cron scheduler", "The function that registers every HTTP route in server/service/handler.go", "A middleware chain", "A template renderer"], correctIndex: 1 },
  { termSlug: "makehandler", difficulty: 6, question: "MakeHandler uses which router library?", options: ["chi", "gorilla/mux", "echo", "gin"], correctIndex: 1 },

  { termSlug: "go-kit-endpoint", difficulty: 4, question: "In Fleet's go-kit transport pattern, an 'endpoint' function...", options: ["Opens an HTTP connection", "Is a thin business adapter between the decoded request and the service call", "Encodes the JSON response", "Authenticates the user"], correctIndex: 1 },
  { termSlug: "go-kit-endpoint", difficulty: 7, question: "The go-kit transport pattern splits an endpoint into which three concerns?", options: ["handler / middleware / renderer", "decoder / endpoint / encoder", "router / controller / view", "validator / processor / responder"], correctIndex: 1 },

  { termSlug: "ee-wrapper", difficulty: 4, question: "The ee/ Enterprise wrapper uses which Go feature to intercept service methods?", options: ["Build tags", "Struct embedding (ee struct embeds core service)", "A goroutine proxy", "Interface composition only"], correctIndex: 1 },
  { termSlug: "ee-wrapper", difficulty: 7, question: "Why doesn't Fleet gate ee features with //go:build tags?", options: ["They slow compilation", "Fleet ships one binary; license checks are at runtime", "go-kit doesn't support them", "They interfere with testing"], correctIndex: 1 },

  { termSlug: "rego", difficulty: 3, question: "Rego is...", options: ["A template language", "The policy language used by OPA (and Fleet's authz)", "A database engine", "An MDM protocol"], correctIndex: 1 },
  { termSlug: "rego", difficulty: 5, question: "Where do Fleet's Rego authz policies live?", options: ["server/service/", "server/authz/", "ee/policy/", "server/rego/"], correctIndex: 1 },

  { termSlug: "authz-authorize", difficulty: 3, question: "svc.authz.Authorize is typically called...", options: ["In middleware before the service", "As the first line of every protected service method", "At the end of a service method", "Only in enterprise features"], correctIndex: 1 },
  { termSlug: "authz-authorize", difficulty: 8, question: "What's the main consequence of forgetting authz.Authorize in a new service method?", options: ["The HTTP middleware returns a 403 because the Rego policy engine was never reached for this method", "Silent security hole: any authenticated user can call it; Fleet's architecture tests scan for the omission", "The method falls back to the viewer's global role check. Observers get read-only, admins get full access", "The method panics on first call because the viewer struct never gets attached to the request context"], correctIndex: 1 },

  // ─── Apple MDM ───
  { termSlug: "mdm", difficulty: 1, question: "MDM stands for...", options: ["Managed Data Mesh", "Mobile Device Management", "Multi-Device Mode", "Modern Deployment Mechanism"], correctIndex: 1 },
  { termSlug: "mdm", difficulty: 5, question: "An MDM profile typically...", options: ["Authenticates a user", "Declares settings/restrictions the device should apply", "Runs a shell script", "Encrypts the hard drive"], correctIndex: 1 },

  { termSlug: "apns", difficulty: 2, question: "APNs stands for...", options: ["Apple Private Network service", "Apple Push Notification service", "Automated Policy Notification system", "Advanced Push Networking"], correctIndex: 1 },
  { termSlug: "apns", difficulty: 6, question: "Why does Fleet rely on APNs for MDM?", options: ["To encrypt profiles (profile encryption is handled by SCEP / ACME certs, not by the APNs channel)", "To wake the device so it checks in for pending commands; the push itself carries no payload", "To tunnel SyncML (SyncML is the Windows MDM protocol; Apple MDM uses its own plist payloads over HTTPS)", "To replace osquery (osquery is the on-device telemetry agent; APNs is just the Apple push channel)"], correctIndex: 1 },

  { termSlug: "abm", difficulty: 2, question: "ABM is Apple's...", options: ["Apple Boot Manager", "Apple Business Manager (portal for org-managed devices + VPP)", "Apple Backup Monitor", "App Build Manager"], correctIndex: 1 },
  { termSlug: "abm", difficulty: 6, question: "Fleet syncs which specific thing from ABM via nanodep?", options: ["VPP app binaries", "Device assignments (which serial numbers are assigned to Fleet)", "End-user accounts", "OS update signatures"], correctIndex: 1 },

  { termSlug: "dep", difficulty: 3, question: "DEP (now called ADE) enables...", options: ["Manual host enrollment via QR code", "Zero-touch enrollment of Apple devices sold through ABM", "Automatic Driver Enrollment", "Device Encryption Protocol"], correctIndex: 1 },
  { termSlug: "dep", difficulty: 7, question: "A Mac enrolled via DEP finds Fleet's MDM URL by...", options: ["Scanning a QR code", "Contacting Apple's activation server on first boot (which returns Fleet's URL based on the ABM assignment)", "A config profile preinstalled by the admin", "DNS discovery"], correctIndex: 1 },

  { termSlug: "ddm", difficulty: 4, question: "DDM (Declarative Device Management) is Apple's...", options: ["New push notification system", "Newer declarative control plane; Fleet declares desired state, device computes diff locally", "SCEP replacement", "Replacement for App Store licensing"], correctIndex: 1 },
  { termSlug: "ddm", difficulty: 8, question: "Which Fleet table stores DDM declarations?", options: ["mdm_apple_profiles", "mdm_apple_declarations", "ddm_declarations", "nano_commands"], correctIndex: 1, explanation: "`mdm_apple_declarations` holds DDM declarations (plus `host_mdm_apple_declarations` for per-host state). Neither `mdm_apple_profiles` nor `ddm_declarations` is a real Fleet table. The actual profile table is `mdm_apple_configuration_profiles`." },

  { termSlug: "mobileconfig", difficulty: 3, question: ".mobileconfig files are...", options: ["Binary MDM commands", "XML plist payloads containing Apple config profile data", "JSON policy documents", "Encrypted blobs"], correctIndex: 1 },
  { termSlug: "mobileconfig", difficulty: 5, question: "A .mobileconfig can contain multiple what?", options: ["Hosts (a profile targets a device, it doesn't bundle a list of them)", "Payloads (Wi-Fi, email, restrictions, etc.)", "Binaries (installer bytes are shipped via InstallApplication, not inside the profile plist)", "Scripts (scripts are run via a separate Fleet script endpoint, not inside the profile plist)"], correctIndex: 1 },

  { termSlug: "plist", difficulty: 3, question: "A plist is...", options: ["A Python list", "Apple's property list format (XML or binary) for structured data", "A package list", "A Mac playlist file"], correctIndex: 1 },
  { termSlug: "plist", difficulty: 5, question: "MDM commands and responses to Apple devices are typically formatted as...", options: ["JSON", "plists", "SyncML", "protobuf"], correctIndex: 1 },

  { termSlug: "scep", difficulty: 3, question: "SCEP is a protocol for...", options: ["Encrypting Wi-Fi", "Devices to request identity certificates (Fleet proxies to external CAs)", "Streaming logs", "MDM command queueing"], correctIndex: 1 },
  { termSlug: "scep", difficulty: 7, question: "Fleet's role in SCEP enrollment is best described as...", options: ["Issuing the certs itself", "Proxying device requests through to the configured external CA", "A certificate authority for all customers", "Blocking SCEP in favor of ACME"], correctIndex: 1 },

  { termSlug: "acme", difficulty: 4, question: "ACME in Fleet refers to...", options: ["An alternative SCEP implementation Fleet uses when NDES isn't available to issue certs", "A cert enrollment protocol; Fleet acts as an ACME server for Apple's directory nonce", "Let's Encrypt's issuance protocol; Fleet uses it to renew its own APNs push certificate", "The TLS cert rotation path fleetd follows when its node key approaches expiry on-device"], correctIndex: 1 },
  { termSlug: "acme", difficulty: 7, question: "ACME is newer than SCEP. Why might Apple shift to it?", options: ["Nothing; SCEP always wins", "ACME is a newer standard with better automation primitives for cert lifecycle", "ACME is faster at runtime", "ACME is required by nanomdm"], correctIndex: 1 },

  { termSlug: "psso", difficulty: 4, question: "PSSO is...", options: ["Personal Storage Service Object", "Platform SSO: Apple's framework for binding macOS login to a cloud IdP", "Private SCIM Service Object", "Policy Store Shared Object"], correctIndex: 1 },
  { termSlug: "psso", difficulty: 8, question: "Fleet's role in PSSO is...", options: ["Authenticating end users directly", "Brokering device trust / nonces so the OS and IdP can complete the sign-in", "Acting as the IdP", "Replacing SCEP for PSSO devices"], correctIndex: 1 },

  { termSlug: "nanomdm", difficulty: 4, question: "nanomdm is...", options: ["A small Apple device", "An embedded lightweight MDM server library that owns Fleet's Apple MDM command queue", "A cron name", "A Fleet frontend widget"], correctIndex: 1 },
  { termSlug: "nanomdm", difficulty: 7, question: "The nano_commands and nano_enrollment_queue tables belong to whose schema?", options: ["Fleet's own schema", "The embedded nanomdm library's schema", "nanodep's", "The Fleet UI's"], correctIndex: 1 },

  { termSlug: "nanodep", difficulty: 4, question: "nanodep is...", options: ["An osquery extension", "An embedded DEP sync library that pulls device assignments from ABM", "A cron scheduler", "A Fleet CLI command"], correctIndex: 1 },
  { termSlug: "nanodep", difficulty: 6, question: "nanodep is primarily responsible for...", options: ["MDM command delivery", "Syncing device assignments between Apple Business Manager and Fleet", "License verification", "osquery TLS enrollment"], correctIndex: 1 },

  { termSlug: "bootstrap-package", difficulty: 3, question: "A bootstrap package is...", options: ["A shell script (scripts have their own Fleet-scripts flow, separate from the setup-experience package)", "A single admin-authored package installed at the end of DEP setup (often to deploy fleetd)", "A Homebrew formula (brew installs are managed via FMAs, not shipped during Apple DEP setup)", "An SCCM package (SCCM is Microsoft's config management stack; Fleet uses its own Apple MDM path)"], correctIndex: 1 },
  { termSlug: "bootstrap-package", difficulty: 6, question: "The bootstrap package is delivered via which Apple MDM command?", options: ["InstallApplication", "InstallEnterpriseApplication", "ApplyProfile", "Push"], correctIndex: 1 },

  // ─── Microsoft MDM ───
  { termSlug: "entra", difficulty: 3, question: "Entra ID is...", options: ["A new Fleet product", "Microsoft's cloud IdP (formerly Azure AD); issues JWTs for Autopilot", "Google's SSO service", "A MySQL competitor"], correctIndex: 1 },
  { termSlug: "entra", difficulty: 6, question: "Fleet verifies Entra-issued JWTs by...", options: ["Calling an Entra REST API per request", "Fetching Entra's JWKS and validating the signature locally", "Using a shared secret", "Calling Microsoft Graph"], correctIndex: 1 },

  { termSlug: "autopilot", difficulty: 3, question: "Autopilot provides...", options: ["Automated software deployment via the Microsoft Store for Business during device setup", "Zero-touch provisioning for Windows (MDM server info delivered at first boot after Entra sign-in)", "Secure boot attestation that gates MDM enrollment on the device's TPM state at first contact", "Zero-touch provisioning for Android devices registered through Google's zero-touch portal"], correctIndex: 1 },
  { termSlug: "autopilot", difficulty: 6, question: "Fleet syncs Autopilot device registrations from where?", options: ["Entra ID directly, via its ARM REST surface authenticated with a service principal", "Microsoft Graph (via the microsoft_autopilot_sync cron)", "Intune's Graph endpoint for Windows Autopilot deployment profiles", "Microsoft Partner Center API for the Autopilot CSP tenant registration data"], correctIndex: 1 },

  { termSlug: "wstep", difficulty: 4, question: "WSTEP is...", options: ["A Windows SSO protocol", "A Windows cert enrollment protocol; Fleet implements the server side", "A Windows software catalog", "A telemetry format"], correctIndex: 1 },
  { termSlug: "wstep", difficulty: 6, question: "WSTEP is specifically for...", options: ["MDM profile delivery", "Issuing identity certs during Windows MDM enrollment", "Windows antivirus", "Autopilot sync"], correctIndex: 1 },

  { termSlug: "syncml", difficulty: 3, question: "SyncML is...", options: ["A Microsoft replacement for JSON", "Microsoft's XML MDM protocol (used by Windows)", "An Apple protocol", "A Google protocol"], correctIndex: 1 },
  { termSlug: "syncml", difficulty: 6, question: "SyncML is actually a general standard implementing which protocol?", options: ["OMA-DM (Open Mobile Alliance Device Management)", "OAuth2 (bearer token authorization flow, unrelated to device management)", "OIDC (identity layer on top of OAuth2, used for SSO)", "SAML (XML-based SSO assertion protocol used by IdPs)"], correctIndex: 0 },

  { termSlug: "csp", difficulty: 4, question: "In Microsoft MDM, a CSP is...", options: ["Cryptographic Service Provider. Windows's older crypto key / cert API that confusingly shares the acronym", "A Windows setting tree node (paths like ./Device/Vendor/MSFT/Policy/...)", "A certificate signing authority Fleet proxies through for Windows device enrollment", "A cloud service product in Microsoft 365 Business that MDM admins can configure for users"], correctIndex: 1 },
  { termSlug: "csp", difficulty: 7, question: "A SyncML command targets a specific CSP (Configuration Service Provider) path like ./Device/Vendor/MSFT/Policy/Config/Browser/HomePages. What happens if the path is wrong?", options: ["The command silently succeeds", "The device responds with a SyncML error status", "Fleet blocks the command before send", "Nothing is logged"], correctIndex: 1 },

  { termSlug: "conditional-access", difficulty: 3, question: "Entra's conditional access gates...", options: ["MDM enrollment", "App access based on device compliance signals", "BitLocker enablement", "SCEP enrollment"], correctIndex: 1 },
  { termSlug: "conditional-access", difficulty: 6, question: "Fleet can participate in conditional access by...", options: ["Replacing Entra", "Acting as a compliance partner that reports device compliance to Entra", "Issuing JWTs on Entra's behalf", "Tunneling SyncML"], correctIndex: 1 },

  // ─── Android MDM ───
  { termSlug: "amapi", difficulty: 3, question: "AMAPI is...", options: ["Apple's Mobile Automation Protocol Interface used by MDM servers for cross-platform control", "Android Management API (Google's cloud API for managing Android Enterprise devices)", "A Microsoft Graph API Fleet calls to pull Autopilot device registrations from Entra", "Google's Pub/Sub-backed push API that Fleet subscribes to for Android device check-ins"], correctIndex: 1 },
  { termSlug: "amapi", difficulty: 7, question: "A key difference between Android MDM via AMAPI and Apple MDM is...", options: ["Fleet opens its own long-poll channel to each Android device for direct command delivery", "Fleet doesn't talk to Android devices directly; it talks to Google, which talks to devices", "Fleet queues commands and pushes them via FCM, the Android equivalent of APNs for MDM", "Fleet writes into mdm_android_commands and nanomdm's Android variant drains the queue"], correctIndex: 1 },

  { termSlug: "emm", difficulty: 3, question: "EMM is the role Fleet plays with...", options: ["Apple (Fleet pairs with Apple Business Manager via DEP + VPP, not as an EMM)", "Google (Fleet is a Google-recognized Enterprise Mobility Management partner)", "Microsoft (Fleet enrolls Windows via the WS-MDM / Autopilot path, not as an EMM)", "Red Hat (Fleet consumes Red Hat's OVAL vuln data, no EMM relationship)"], correctIndex: 1 },
  { termSlug: "emm", difficulty: 5, question: "The 'EMM' term is primarily associated with which ecosystem?", options: ["Apple MDM", "Android enterprise management", "Linux", "ChromeOS"], correctIndex: 1 },

  { termSlug: "google-pubsub", difficulty: 4, question: "Google Pub/Sub is used by Fleet to...", options: ["Replace Redis", "Receive AMAPI-originated events (enrollment, status, command results) from Google", "Stream query results", "Distribute cron ticks"], correctIndex: 1 },
  { termSlug: "google-pubsub", difficulty: 6, question: "Why does Fleet subscribe to Google Pub/Sub instead of polling AMAPI?", options: ["Pub/Sub pushes near-realtime device events; polling would be slower and quota-heavy", "Polling is forbidden", "It's required by SCIM", "Google doesn't offer polling"], correctIndex: 0 },

  { termSlug: "work-profile", difficulty: 3, question: "A work profile on Android is...", options: ["A user profile with permissions", "A BYOD container that separates work apps/data from personal", "A Microsoft Entra feature", "A Mac user account"], correctIndex: 1 },
  { termSlug: "work-profile", difficulty: 6, question: "When Fleet manages an Android device in work-profile mode, which actions are limited?", options: ["None (work-profile mode exists specifically to limit what the company can see / do on a personal device)", "Actions outside the work container (e.g. full-device wipe) are limited to protect personal data", "Only install actions (installs are allowed inside the work container; the limits are around personal data)", "Only inventory (inventory is read-only; the real limits are on write actions that would reach personal data)"], correctIndex: 1 },

  // ─── Backend / Go ───
  { termSlug: "goroutine", difficulty: 2, question: "A goroutine is...", options: ["A Go package", "A lightweight thread run by the Go runtime", "A function pointer", "A Rust fiber"], correctIndex: 1 },
  { termSlug: "goroutine", difficulty: 5, question: "You start a goroutine with...", options: ["go functionName()", "goroutine { ... }", "async functionName()", "parallel.Run()"], correctIndex: 0 },

  { termSlug: "channel", difficulty: 3, question: "A channel in Go is...", options: ["A tracing span", "A typed FIFO queue for passing values between goroutines", "A database connection", "An HTTP handler"], correctIndex: 1 },
  { termSlug: "channel", difficulty: 6, question: "Closing a channel signals...", options: ["A memory leak", "That no more values will be sent; receivers get a zero value + false", "That the sender crashed", "Nothing; close is a no-op"], correctIndex: 1 },

  { termSlug: "context", difficulty: 3, question: "context.Context is used for...", options: ["Rendering templates", "Request-scoped values + cancellation + deadline", "Config loading", "Locale handling"], correctIndex: 1 },
  { termSlug: "context", difficulty: 7, question: "Fleet typically carries which of these on the context?", options: ["The viewer (user + roles + team memberships) and the trace span", "MySQL rows", "The Redis connection pool", "The HTTP response writer"], correctIndex: 0 },

  { termSlug: "interface", difficulty: 3, question: "In Go, an interface is...", options: ["A class with abstract methods", "A duck-typed abstraction; a struct satisfies it if it has the required methods", "A type alias", "A module declaration"], correctIndex: 1 },
  { termSlug: "interface", difficulty: 6, question: "Fleet uses interfaces heavily for...", options: ["Decorating types", "Decoupling service from storage (fleet.Datastore) so tests can substitute mocks", "Runtime polymorphism via vtables", "Compiler optimization"], correctIndex: 1 },

  { termSlug: "struct-embedding", difficulty: 4, question: "Go's struct embedding lets you...", options: ["Nest structs with explicit field access", "Include one struct inside another and inherit its methods on the outer struct", "Subclass with virtual methods", "Merge database rows"], correctIndex: 1 },
  { termSlug: "struct-embedding", difficulty: 7, question: "Fleet uses struct embedding most prominently to...", options: ["Share Redis connections", "Build the ee Enterprise wrapper by embedding the core service", "Chain middleware", "Format logs"], correctIndex: 1 },

  { termSlug: "gorilla-mux", difficulty: 3, question: "gorilla/mux is Fleet's...", options: ["WebSocket library", "HTTP router (path templates + method matching)", "MySQL driver", "JSON parser"], correctIndex: 1 },
  { termSlug: "gorilla-mux", difficulty: 5, question: "In MakeHandler, routes are registered using methods like...", options: ["ue.GET(path, endpoint, zeroRequest)", "router.Handle(path)", "http.HandleFunc", "echo.Group"], correctIndex: 0 },

  { termSlug: "goose", difficulty: 3, question: "goose is Fleet's...", options: ["Templating library", "MySQL migration tool", "Feature flag library", "Metrics exporter"], correctIndex: 1 },
  { termSlug: "goose", difficulty: 6, question: "goose migration files are ordered by...", options: ["Alphabetical name", "Numeric timestamp prefix", "Commit order in git", "Random UUID"], correctIndex: 1 },

  { termSlug: "sqlx", difficulty: 3, question: "sqlx is used because...", options: ["It's an ORM", "It adds named params + typed struct scanning on top of database/sql", "It replaces MySQL", "It's required by Go"], correctIndex: 1 },
  { termSlug: "sqlx", difficulty: 6, question: "Fleet uses sqlx but does NOT use...", options: ["sql.DB", "An ORM (hand-written SQL throughout)", "Migrations", "Transactions"], correctIndex: 1 },

  { termSlug: "opa", difficulty: 3, question: "OPA (Open Policy Agent) is used by Fleet for...", options: ["Metrics collection", "Evaluating authz policies written in Rego", "Feature flags", "Scheduling crons"], correctIndex: 1 },
  { termSlug: "opa", difficulty: 7, question: "Where does Fleet run OPA?", options: ["As a sidecar process", "In-process; OPA evaluates Rego inside the Fleet binary", "As a Lambda function", "In a separate container"], correctIndex: 1 },

  { termSlug: "iterator", difficulty: 4, question: "iter.Seq2 is a Go 1.23+ pattern for...", options: ["Parallel iteration", "Range-over-function sequences of (value, error) pairs", "Memoization", "Context propagation"], correctIndex: 1 },
  { termSlug: "iterator", difficulty: 7, question: "Fleet introduced iterators in the datastore for...", options: ["Streaming large reads (StreamHosts) with bounded memory and cancelable mid-stream", "Replacing LIMIT/OFFSET pagination with cursor-based reads over the hosts table", "Decoupling service code from sqlx so Fleet can swap to a different DB driver cleanly", "Lazy evaluation of policy_membership rows so the dashboard only loads what's visible"], correctIndex: 0 },

  // ─── Standards & protocols ───
  { termSlug: "saml", difficulty: 3, question: "SAML authentication works by...", options: ["OIDC token exchange", "The IdP posting a signed XML assertion to Fleet's ACS endpoint", "A shared password", "A SCEP challenge"], correctIndex: 1 },
  { termSlug: "saml", difficulty: 7, question: "SAML signatures are verified against...", options: ["The user's password hash", "The IdP's public cert configured on Fleet", "Google's root CA", "Fleet's TLS cert"], correctIndex: 1 },

  { termSlug: "oidc", difficulty: 3, question: "OIDC is...", options: ["A Microsoft-only protocol", "OpenID Connect; OAuth2-based SSO, modern alternative to SAML", "A legacy acronym for SAML", "An OAuth1 extension"], correctIndex: 1 },
  { termSlug: "oidc", difficulty: 6, question: "A key operational difference between OpenID Connect (OIDC) and Security Assertion Markup Language (SAML) is...", options: ["OIDC uses signed JSON Web Tokens (JWTs); SAML uses signed XML assertions", "OIDC supports Single Logout (SLO) natively; SAML has no logout specification", "OIDC assertions must be encrypted end-to-end; SAML assertions only need to be signed", "OIDC was designed for machine-to-machine auth; SAML for interactive user SSO"], correctIndex: 0 },

  { termSlug: "jwks", difficulty: 3, question: "JWKS is...", options: ["A Java KeyStore format", "JSON Web Key Set: a URL serving an IdP's public signing keys", "A JavaScript bundle", "A JWT variant"], correctIndex: 1 },
  { termSlug: "jwks", difficulty: 6, question: "Fleet fetches JWKS specifically for...", options: ["OIDC login only", "Verifying Autopilot JWTs from Entra", "Encrypting MDM commands", "Signing push certs"], correctIndex: 1 },

  { termSlug: "jwt", difficulty: 2, question: "A JWT is...", options: ["A database row", "A signed JSON payload (JSON Web Token)", "An encrypted file", "A Node.js framework"], correctIndex: 1 },
  { termSlug: "jwt", difficulty: 6, question: "In Fleet's Windows Autopilot flow, a JWT is...", options: ["Issued by Fleet to the device", "Issued by Entra to the device and presented to Fleet", "Rejected by default", "Used for VPP"], correctIndex: 1 },

  { termSlug: "scim", difficulty: 3, question: "SCIM is used for...", options: ["MDM (device management protocols are SyncML / Apple MDM / AMAPI, not SCIM)", "A protocol for an IdP to push user/group lifecycle to downstream apps (Fleet is a target)", "SSO alone (SSO is SAML or OIDC; SCIM provisions the users SSO then authenticates)", "Cert enrollment (that's SCEP or ACME, protocols for device certificates)"], correctIndex: 1 },
  { termSlug: "scim", difficulty: 6, question: "Which tables store SCIM-provisioned users/groups?", options: ["users, teams", "scim_users, scim_groups", "idp_users, idp_groups", "sso_users, sso_groups"], correctIndex: 1 },

  { termSlug: "tuf", difficulty: 3, question: "TUF is a spec for...", options: ["Secure software updates (used by fleetd for auto-updating components)", "Encrypted DNS", "Time sync", "A transport protocol"], correctIndex: 0 },
  { termSlug: "tuf", difficulty: 6, question: "If TUF's signed metadata fails verification, fleetd...", options: ["Ignores it and updates anyway", "Refuses the update to prevent supply-chain attacks", "Deletes itself", "Calls home to admins"], correctIndex: 1 },

  { termSlug: "websocket", difficulty: 2, question: "A WebSocket is...", options: ["A UDP tunnel", "A bi-directional HTTP connection (upgraded from HTTP)", "An HTTP/3 feature", "An SSH session"], correctIndex: 1 },
  { termSlug: "websocket", difficulty: 6, question: "Fleet uses WebSockets specifically for...", options: ["All admin API calls", "Streaming live-query result rows per campaign to the admin UI", "MDM commands", "Health checks"], correctIndex: 1 },

  // ─── Database ───
  { termSlug: "replica-lag", difficulty: 4, question: "Replica lag is the delay between...", options: ["A cron and its next tick", "A write to primary MySQL and the write being visible on a replica", "A server start and readiness", "Two Fleet instances syncing config"], correctIndex: 1 },
  { termSlug: "replica-lag", difficulty: 7, question: "A common replica-lag bug in Fleet is...", options: ["Phantom reads during BatchApply when the diff runs against a replica but writes hit the primary", "Read-after-write that assumes consistency ('I saved it and now I can't read it back')", "goose double-applying a migration because the replica reports an older schema_migrations version", "Lost UPDATEs when service code grabs a DBReader connection instead of the DBWriter for a write"], correctIndex: 1 },

  { termSlug: "dbreader-dbwriter", difficulty: 4, question: "Fleet's DBReader vs DBWriter abstractions correspond to...", options: ["OLAP vs OLTP", "MySQL replica (reads) vs primary (writes)", "Redis read vs write modes", "Sync vs async"], correctIndex: 1 },
  { termSlug: "dbreader-dbwriter", difficulty: 7, question: "After a write that must be read back immediately, you should...", options: ["Sleep 100ms", "Use the writer connection for the read, or restructure the flow to tolerate lag", "Always use DBReader", "Force a MySQL restart"], correctIndex: 1 },

  { termSlug: "migration", difficulty: 2, question: "A Fleet migration is...", options: ["A versioned Go file sqlx applies on server startup to seed required default rows", "A timestamped Up/Down SQL change in server/datastore/mysql/migrations/", "A YAML diff fleetctl ships to Fleet when the schema spec version bumps on apply", "A one-way INSERT in server/datastore/mysql/migrations/ that bootstraps data on first start"], correctIndex: 1 },
  { termSlug: "migration", difficulty: 7, question: "If your migration file has a timestamp older than the latest on main, you should...", options: ["Rebase and merge; it just works", "Rename with a current timestamp (bump)", "Force-push over main's migration", "Delete the older one from main"], correctIndex: 1 },

  // ─── Vuln / Security ───
  { termSlug: "cpe", difficulty: 3, question: "A CPE is...", options: ["A per-CVE severity rating published alongside CVSS by NIST on each advisory", "A canonical software identifier used to match installed software against vuln feeds", "A timestamped vuln feed update NVD publishes after each CVE curation pass", "A (product, vendor, version) tuple Fleet uses as the primary key for software_cve"], correctIndex: 1 },
  { termSlug: "cpe", difficulty: 7, question: "The main source of false positives in vuln matching is usually...", options: ["Network issues (feed download failures surface as stale data or missing CVEs, not as false positives)", "CPE mismatch between the installed software's metadata and vuln feed CPEs", "Replica lag (stale reads briefly show old vuln state, but the dominant cause of wrong matches is CPE mapping)", "OS version drift (host OS versions are refreshed each check-in; drift causes staleness, not match errors)"], correctIndex: 1 },

  { termSlug: "cve", difficulty: 1, question: "A CVE is...", options: ["A severity score (0-10) published by NIST for each disclosed vulnerability", "A canonical identifier for a security vulnerability (CVE-YYYY-NNNNN)", "A per-product bulletin from a vendor like Microsoft or Red Hat describing a patch", "A label Fleet applies to software_cve rows after the vulnerabilities cron triages them"], correctIndex: 1 },
  { termSlug: "cve", difficulty: 4, question: "Fleet stores CVE metadata (CVSS, description, published date) in...", options: ["cve_meta", "vulnerabilities", "software_cve", "cve_index"], correctIndex: 0 },

  { termSlug: "cvss", difficulty: 3, question: "CVSS is...", options: ["A Fleet config format", "A 0-10 severity score attached to a CVE", "A vuln feed", "A cert version string"], correctIndex: 1 },
  { termSlug: "cvss", difficulty: 5, question: "A CVSS score of 9.8 means...", options: ["Low risk", "Critical severity (near-maximum)", "Medium", "Depends on vendor"], correctIndex: 1 },

  { termSlug: "nvd", difficulty: 2, question: "NVD is...", options: ["Nondeterministic Vuln Detection", "The US NIST National Vulnerability Database (Fleet's primary CVE source)", "A Fleet dashboard", "An osquery extension"], correctIndex: 1 },
  { termSlug: "nvd", difficulty: 5, question: "A limitation of NVD compared to OSV is...", options: ["NVD is older", "NVD covers packages less well (e.g. npm/PyPI/Go modules)", "NVD is paid", "NVD only covers Linux"], correctIndex: 1 },

  { termSlug: "osv", difficulty: 3, question: "OSV is...", options: ["An open-source VPN", "Google-maintained cross-ecosystem vuln feed, strong for package ecosystems", "A Fleet logging target", "A legacy osquery format"], correctIndex: 1 },
  { termSlug: "osv", difficulty: 6, question: "OSV complements NVD because OSV covers...", options: ["Only Windows", "Packages like npm, PyPI, Go modules better than NVD does", "Only kernel vulns", "Only web apps"], correctIndex: 1 },

  { termSlug: "oval", difficulty: 4, question: "OVAL feeds are typically...", options: ["Kernel-level (OVAL describes software versions and patches, not kernel-level system calls)", "Per-Linux-distro vuln data (Red Hat, Ubuntu, Debian, SUSE)", "macOS-only (macOS vuln data in Fleet comes from NVD + Apple advisories, not OVAL)", "iOS-specific (iOS has its own Apple-side advisory feed; OVAL doesn't cover iOS)"], correctIndex: 1 },
  { termSlug: "oval", difficulty: 6, question: "Why does Fleet consume per-distro OVAL feeds in addition to NVD?", options: ["Distros release vuln updates on their own cadence; OVAL carries distro-specific fix metadata", "OVAL replaces NVD", "NVD doesn't cover Linux at all", "OVAL is required by law"], correctIndex: 0 },

  { termSlug: "msrc", difficulty: 3, question: "MSRC is...", options: ["Microsoft's Robotics Compiler", "Microsoft's security feed used for Windows + Office vuln matching", "A GitHub product", "A MyAnalytics module"], correctIndex: 1 },
  { termSlug: "msrc", difficulty: 6, question: "A notable quirk of MSRC data for Office vuln matching is...", options: ["It's JSON", "It's distributed as XLSX files", "It's realtime WebSocket", "It requires an API key"], correctIndex: 1 },
];

export function pickAdaptive(
  currentDifficulty: number,
  askedSlugsAndIdx: Set<string>,
  pool: QuizQuestion[] = QUESTIONS,
): QuizQuestion | null {
  // Search outward from current difficulty: current, +1, -1, +2, -2, ...
  for (let d = 0; d <= 9; d++) {
    for (const offset of [0, -d, d]) {
      if (d === 0 && offset !== 0) continue;
      const target = currentDifficulty + offset;
      if (target < 1 || target > 10) continue;
      const candidates = pool.filter(
        (q) =>
          q.difficulty === target &&
          !askedSlugsAndIdx.has(`${q.termSlug}:${q.question}`),
      );
      if (candidates.length > 0) {
        return candidates[Math.floor(Math.random() * candidates.length)];
      }
    }
  }
  return null;
}

import type { GlossaryCategory } from "./terms";
import { TERMS } from "./terms";

export type QuizBucket = {
  slug: string;
  label: string;
  description: string;
  categories: GlossaryCategory[] | null; // null = all
  terms?: string[]; // if set, use this allowlist of slugs instead of categories
  maxDifficulty?: number; // clamp pool to <= this
  startDifficulty?: number; // default 5
};

export const BUCKETS: QuizBucket[] = [
  {
    slug: "fleet",
    label: "Fleet",
    description:
      "What makes Fleet Fleet: product concepts, code conventions, service/datastore split.",
    categories: ["Fleet concept", "Fleet code"],
  },
  {
    slug: "mdm",
    label: "MDM",
    description:
      "Apple, Microsoft, and Android device management: protocols, portals, embedded libraries.",
    categories: ["Apple MDM", "Microsoft MDM", "Android MDM"],
  },
  {
    slug: "backend",
    label: "Backend",
    description:
      "Go idioms Fleet uses (interfaces, embedding, iterators, context) + MySQL patterns (sqlx, goose, replica lag).",
    categories: ["Backend / Go", "Database"],
  },
  {
    slug: "security-vuln",
    label: "Standards & vuln",
    description:
      "Auth protocols (SAML/OIDC/SCIM/TUF/JWT/JWKS) and the vulnerability pipeline (NVD/OSV/OVAL/MSRC/CPE/CVE).",
    categories: ["Standards & protocols", "Vuln / Security"],
  },
  {
    slug: "customer-success",
    label: "Customer success",
    description:
      "What people in customer-facing roles (support, CS, sales eng, solutions) should know: operational concepts, MDM vocabulary customers use, SSO/SCIM integration terms, vuln vocabulary. Caps at advanced-operational depth.",
    categories: null,
    terms: [
      "fleet",
      "report",
      "live-query",
      "policy",
      "label",
      "host",
      "fleetd",
      "osqueryd",
      "fleet-desktop",
      "fleetctl",
      "gitops",
      "fma",
      "vpp",
      "self-service",
      "setup-experience",
      "enroll-secret",
      "node-key",
      "device-token",
      "fleet-variable",
      "secret-variable",
      "activity",
      "activity-streaming",
      "mdm",
      "apns",
      "abm",
      "dep",
      "ddm",
      "bootstrap-package",
      "mobileconfig",
      "scep",
      "acme",
      "entra",
      "autopilot",
      "conditional-access",
      "syncml",
      "csp",
      "amapi",
      "emm",
      "work-profile",
      "saml",
      "oidc",
      "jwks",
      "jwt",
      "scim",
      "cve",
      "cvss",
      "nvd",
    ],
    maxDifficulty: 7,
    startDifficulty: 4,
  },
  {
    slug: "non-coding",
    label: "Non-coding",
    description:
      "What a Fleetie without a code background should know: product concepts, admin-facing MDM terms, agent names, high-level auth and vuln vocabulary. Caps at intermediate difficulty.",
    categories: null,
    terms: [
      "fleet",
      "report",
      "live-query",
      "policy",
      "label",
      "host",
      "fleetd",
      "osqueryd",
      "fleet-desktop",
      "fleetctl",
      "gitops",
      "fma",
      "vpp",
      "self-service",
      "setup-experience",
      "enroll-secret",
      "node-key",
      "device-token",
      "activity",
      "activity-streaming",
      "mdm",
      "apns",
      "abm",
      "dep",
      "ddm",
      "bootstrap-package",
      "entra",
      "autopilot",
      "conditional-access",
      "amapi",
      "emm",
      "work-profile",
      "cve",
      "cvss",
      "nvd",
      "saml",
      "oidc",
      "scim",
    ],
    maxDifficulty: 6,
    startDifficulty: 3,
  },
  {
    slug: "e2e",
    label: "E2E (all)",
    description:
      "Everything. Across all categories; the fullest range of difficulty.",
    categories: null,
  },
];

export function poolForBucket(bucket: QuizBucket): QuizQuestion[] {
  let pool: QuizQuestion[];
  if (bucket.terms && bucket.terms.length > 0) {
    const allowed = new Set(bucket.terms);
    pool = QUESTIONS.filter((q) => allowed.has(q.termSlug));
  } else if (bucket.categories) {
    const termsInBucket = new Set(
      TERMS.filter((t) => bucket.categories!.includes(t.category)).map(
        (t) => t.slug,
      ),
    );
    pool = QUESTIONS.filter((q) => termsInBucket.has(q.termSlug));
  } else {
    pool = QUESTIONS;
  }
  if (bucket.maxDifficulty != null) {
    pool = pool.filter((q) => q.difficulty <= bucket.maxDifficulty!);
  }
  return pool;
}
