# Fleet bounded context proposals

Three strategic approaches to modularize Fleet's codebase based on comprehensive analysis.

**Analysis Date:** November 2025  
**Codebase Size:** ~539K lines Go (server)  
**Database Tables:** ~182 tables  
**API Endpoints:** 385+ endpoints across 4 protocols  

---

## Table of contents

- [Why modularize? Pain points addressed](#why-modularize-pain-points-addressed)
- [Bounded context design principles](#bounded-context-design-principles)
- [Current state analysis](#current-state-analysis)
- [Proposal 1: Platform-Centric (Vertical Slicing)](#proposal-1-platform-centric-vertical-slicing)
  - Bounded Contexts:
    - [1. Host & Device Core](#1-host--device-core)
    - [2. Agent Management](#2-agent-management)
    - [3. Apple Platform Management](#3-apple-platform-management)
    - [4. Windows Platform Management](#4-windows-platform-management)
    - [5. Android Platform Management](#5-android-platform-management)
    - [6. Linux Platform Management](#6-linux-platform-management)
    - [7. Software & Vulnerability Management](#7-software--vulnerability-management)
    - [8. Security & Compliance](#8-security--compliance)
    - [9. Identity & Access Management](#9-identity--access-management)
    - [10. Platform Services (Shared)](#10-platform-services-shared)
  - [Module dependency map](#module-dependency-map)
  - [API Architecture](#api-architecture-for-proposal-1)
  - [Pros & Cons](#pros)
- [Proposal 2: Capability-Centric (Domain-Driven Design)](#proposal-2-capability-centric-domain-driven-design)
  - Bounded Contexts:
    - [1. Agent Management (Foundation)](#1-agent-management-foundation)
    - [2. Device Enrollment & Lifecycle](#2-device-enrollment--lifecycle)
    - [3. Configuration Management](#3-configuration-management)
    - [4. Software Lifecycle Management](#4-software-lifecycle-management)
    - [5. Security & Compliance](#5-security--compliance)
    - [6. Query & Reporting](#6-query--reporting)
    - [7. Automation & Scripts](#7-automation--scripts)
    - [8. Identity & Access](#8-identity--access)
    - [9. Activity & Audit](#9-activity--audit)
    - [10. Platform Core (Shared)](#10-platform-core-shared)
  - [Module dependency map](#module-dependency-map-1)
  - [API Architecture](#api-architecture-for-proposal-2)
  - [Pros & Cons](#pros-1)
- [Proposal 3: Hybrid (pragmatic evolution)](#proposal-3-hybrid-pragmatic-evolution)
  - Bounded Contexts:
    - [1. Host Management](#1-host-management)
    - [2. Agent Management](#2-agent-management-1)
    - [3. Apple MDM](#3-apple-mdm)
    - [4. Windows MDM](#4-windows-mdm)
    - [5. Android MDM](#5-android-mdm)
    - [6. Software Management](#6-software-management)
    - [7. Policy & Compliance](#7-policy--compliance)
    - [8. Query & Reporting](#8-query--reporting)
    - [9. Automation & Scripts](#9-automation--scripts)
    - [10. Activity & Audit](#10-activity--audit)
    - [11. Identity & Teams](#11-identity--teams)
    - [12. Platform Services](#12-platform-services)
  - [Module dependency map](#module-dependency-map-2)
  - [API Architecture](#api-architecture-for-proposal-3)
  - [Pros & Cons](#pros-2)
- [Comparison Matrix](#comparison-matrix)
- [Recommendation](#recommendation)

---

## Why modularize? Pain points addressed

Note: this is not a full analysis of pain points, but rather a brief summary. The purpose of this document is to identify and select boundaries for the modularization effort, not to prioritize the work versus other business needs.

**Current challenges:**
- **Massive service interface** - 1,380 lines with ~400 methods makes understanding system boundaries difficult
- **Tight coupling** - Changes in one domain (e.g., Apple MDM) can inadvertently break another (e.g., Software)
- **Developer velocity** - New engineers struggle to understand where code belongs; 539K lines of Go is overwhelming
- **Testing complexity** - Unit tests must mock enormous service interfaces; integration tests are slow
- **Merge conflicts** - Large files like `hosts.go` and `service.go` create frequent conflicts
- **Database bottleneck** - 182 tables in one schema with unclear ownership makes migrations error-prone

**Goals of modularization:**
- **Clear boundaries** - Each module has explicit responsibilities, dependencies, and public APIs
- **Improved testability** - Modules can be tested in isolation with smaller interfaces
- **Faster onboarding** - New developers can focus on one bounded context at a time
- **Parallel development** - Teams can work on different modules without conflicts
- **Future flexibility** - Well-defined modules are easier to replace or extend

---

## Bounded context design principles

**Database transactions:**
- ❌ **NEVER cross bounded context boundaries**: Each transaction must operate within a single context
- ✅ If a use case needs multiple contexts, use the Saga pattern or orchestration at the application layer
- ✅ Each bounded context owns its tables exclusively; no shared writes

**Database schema:**
- ✅ **Single database**: All contexts share one MySQL database for operational simplicity
- ✅ **Joins across contexts allowed**: For read operations (queries, reports, dashboards)
- ⚠️ **Joins indicate coupling**: Frequent cross-context joins suggest wrong boundaries
- ✅ Consider views or other approaches for complex cross-context queries

**Module communication:**
- ✅ **Public APIs only**: Modules expose service interfaces; internal implementation is private
- ✅ **Events for state changes**: Use domain events for async coordination between modules (details TBD)
- ✅ **Direct calls for reads**: Synchronous calls to other module's public API for queries
- ✅ **No database access across modules**: Module A cannot directly query Module B's tables

**Implementation approach:**
- Start with logical boundaries (packages/namespaces) within the monolith
- Enforce boundaries with linting/tooling (i.e., archtest)
- Keep single deployment unit; this is a modular monolith

---

## Current state analysis

### Database table distribution

| Domain | Table Count | Examples |
|--------|-------------|----------|
| **MDM** | 30+ | mdm_apple_*, mdm_windows_*, mdm_android_* |
| **MDM Infrastructure** | 17 | nano_*, scep_*, wstep_*, abm_tokens |
| **Hosts** | 35+ | hosts, host_*, network_interfaces |
| **Software** | 12 | software*, software_installer*, software_titles* |
| **VPP** | 8 | vpp_tokens, vpp_apps*, vpp_app_team* |
| **In-House Apps** | 3 | in_house_apps, in_house_app_* |
| **Policies** | 5 | policies, policy_* |
| **Queries** | 7 | queries, query_*, distributed_query_* |
| **Scripts** | 5 | scripts, script_*, batch_activities |
| **Users & Auth** | 10 | users, sessions, invites, scim_* |
| **Teams** | 2 | teams, enroll_secrets |
| **Activities** | 2 | activities, upcoming_activities |
| **Agent Protocol** | 3 | carves, carve_blocks, yara_rules |
| **Others** | 43 | Configuration, calendar, vulnerabilities, etc. |

### Service layer breakdown

From `server/fleet/service.go` (~1,380 lines):

**MDM-related methods:** ~150+ methods (40% of service interface!)
- Apple MDM: ~100 methods
- Windows MDM: ~30 methods
- Android MDM: ~4 methods
- Linux MDM: ~2 methods
- Common MDM: ~15 methods

**Agent protocol methods:** ~20 methods (osquery + orbit)

**Host management:** ~30 methods

**Software management:** ~25 methods

**Policies & Queries:** ~40 methods

**Users & Teams:** ~30 methods

**Other:** ~85 methods

### API endpoint distribution

From analysis of 385 endpoints:

| Protocol/Area | Endpoint Count | Path Pattern |
|---------------|----------------|--------------|
| Osquery Protocol | 6 | `/api/osquery/*` |
| Orbit Protocol | 13 | `/api/fleet/orbit/*` |
| Apple MDM | 50+ | `/api/mdm/apple/*`, `/api/v1/fleet/mdm/apple/*` |
| Windows MDM | 10+ | `/api/mdm/microsoft/*`, `/api/v1/fleet/mdm/microsoft/*` |
| Android MDM | 5+ | `/api/v1/fleet/mdm/android/*` |
| Hosts (REST) | 30+ | `/api/v1/fleet/hosts/*` |
| Software (REST) | 25+ | `/api/v1/fleet/software/*` |
| Queries (REST) | 20+ | `/api/v1/fleet/queries/*`, `/api/v1/fleet/packs/*` |
| Policies (REST) | 15+ | `/api/v1/fleet/policies/*` |
| Scripts (REST) | 10+ | `/api/v1/fleet/scripts/*` |
| Users/Teams (REST) | 35+ | `/api/v1/fleet/users/*`, `/api/v1/fleet/teams/*` |
| Config (REST) | 10+ | `/api/v1/fleet/config/*` |
| Other (REST) | 40+ | Various |

**Total:** 385+ endpoints

## Proposal 1: Platform-Centric (Vertical Slicing)

**Philosophy:** Organize by customer platform, but recognize fleetd as cross-platform

### Bounded Contexts

#### 1. Host & Device Core
**Scope:** Platform-agnostic host management

**Tables (25+):**
- hosts (core)
- host_additional
- host_batteries
- host_certificates*
- host_disks
- host_display_names
- host_orbit_info
- host_seen_times
- host_users
- host_emails
- network_interfaces
- operating_systems
- kernel_host_counts
- labels, label_membership
- challenges

**Service Methods (30+):**
- `ListHosts()`, `GetHost()`, `DeleteHost()`
- `HostByIdentifier()`
- `RefetchHost()`
- `AddHostsToTeam()`
- `ListHostDeviceMapping()`
- `MacadminsData()`, `MDMData()`
- `OSVersions()`
- Label management

**Datastore Files:**
- `hosts.go`
- `labels.go`
- `operating_systems.go`
- `challenges.go`

**API Endpoints (30+):**

**REST API:**
- `GET /api/v1/fleet/hosts` - List hosts (paginated, filtered)
- `POST /api/v1/fleet/hosts/count` - Count hosts
- `GET /api/v1/fleet/hosts/{id}` - Get host details
- `DELETE /api/v1/fleet/hosts/{id}` - Delete host
- `POST /api/v1/fleet/hosts/transfer` - Transfer hosts to team
- `POST /api/v1/fleet/hosts/{id}/refetch` - Refetch host
- `POST /api/v1/fleet/hosts/{id}/query` - Run query on host
- `GET /api/v1/fleet/hosts/{id}/device_mapping` - Get device mapping
- `GET /api/v1/fleet/hosts/{id}/macadmins` - Get macadmins data
- `GET /api/v1/fleet/hosts/{id}/mdm` - Get MDM info
- `GET /api/v1/fleet/os_versions` - OS version statistics
- `GET /api/v1/fleet/labels` - List labels
- `POST /api/v1/fleet/labels` - Create label
- `PATCH /api/v1/fleet/labels/{id}` - Update label
- `DELETE /api/v1/fleet/labels/{id}` - Delete label
- ... (more host management endpoints)

**Rationale:**
- Every platform needs host lifecycle management
- Shared by both agent-based and MDM-only platforms
- Foundation for all device management

#### 2. Agent Management
**Scope:** Osquery and Orbit protocols - platform-agnostic agent communication

**Tables (5+):**
- carves, carve_blocks
- yara_rules
- etc.
- Reads from: hosts (for authentication), distributed_query_*, query_results

**Service Methods (20+):**
- `EnrollOsquery()` - Enroll agent with secret
- `AuthenticateHost()` - Validate osquery node key
- `GetClientConfig()` - Return osquery config (queries, packs, options)
- `GetDistributedQueries()` - Return queries for agent to run
- `SubmitDistributedQueryResults()` - Accept query results
- `SubmitStatusLogs()`, `SubmitResultLogs()` - Ingest logs
- `EnrollOrbit()` - Enroll orbit with extended host info
- `AuthenticateOrbitHost()` - Validate orbit node key
- `GetOrbitConfig()` - Return orbit config (flags, extensions)
- `SetOrUpdateDeviceToken()` - Register push token
- `GetOrbitScript()`, `PostOrbitScriptResult()` - Script execution
- `PostOrbitSoftwareInstallResult()` - Agent-based software install results
- `PostOrbitDiskEncryptionKey()` - Escrow disk encryption keys
- `PostOrbitLUKSData()` - Escrow LUKS passphrases
- `CarveBegin()` - File carving

**Datastore Files:**
- `carves.go`
- Some methods from `hosts.go` (enrollment, authentication)

**API Endpoints (19):**

**Osquery Protocol (6):**
- `POST /api/osquery/enroll` - Agent enrollment (no auth)
- `POST /api/osquery/config` - Get osquery config (host auth)
- `POST /api/osquery/distributed/read` - Get queries (host auth)
- `POST /api/osquery/distributed/write` - Submit results (host auth)
- `POST /api/osquery/log` - Submit logs (host auth)
- `POST /api/osquery/carve/begin` - Start file carve (host auth)
- `POST /api/osquery/yara/{name}` - Get YARA rule (host auth)

**Orbit Protocol (13):**
- `POST /api/fleet/orbit/enroll` - Orbit enrollment (no auth)
- `POST /api/fleet/orbit/config` - Get orbit config (orbit auth)
- `POST /api/fleet/orbit/device_token` - Register push token (orbit auth)
- `POST /api/fleet/orbit/scripts/request` - Get pending script (orbit auth)
- `POST /api/fleet/orbit/scripts/result` - Submit script result (orbit auth)
- `PUT /api/fleet/orbit/device_mapping` - Submit device mapping (orbit auth)
- `POST /api/fleet/orbit/software_install/result` - Software result (orbit auth)
- `POST /api/fleet/orbit/software_install/package` - Download installer (orbit auth)
- `POST /api/fleet/orbit/software_install/details` - Install details (orbit auth)
- `POST /api/fleet/orbit/disk_encryption_key` - Escrow BitLocker (orbit auth, requires Windows MDM)
- `POST /api/fleet/orbit/luks_data` - Escrow LUKS (orbit auth)
- `POST /api/fleet/orbit/setup_experience/init` - Init setup (orbit auth)
- `POST /api/fleet/orbit/setup_experience/status` - Setup status (orbit auth, macOS requires Apple MDM)

**Rationale:**
- **Works without MDM** - ChromeOS uses agent only
- **Works with competitor MDM** - Customers use Fleet agent + Jamf/Intune
- **Platform-agnostic** - Same protocol for macOS, Windows, Linux, ChromeOS
- **Protocol-based boundary** - Clear separation from REST API and MDM protocols

#### 3. Apple Platform Management
**Scope:** Everything related to managing Apple devices (macOS, iOS, iPadOS)

**Tables (50+):**
- mdm_apple_* (11 tables)
- host_mdm_apple_* (4 tables)
- host_dep_assignments
- abm_tokens
- nano_* (9 tables) - Apple MDM infrastructure
- scep_* (2 tables) - Certificate infrastructure
- host_identity_scep_* (2 tables)
- vpp_* (8 tables) - Apple VPP
- in_house_apps (3 tables) - iOS in-house apps
- eulas
- setup_experience_* (2 tables)
- Subset of: hosts, host_software, host_updates, etc.

**Service Methods (100+):**
- All Apple MDM methods
- VPP management
- DEP/ABM management
- Setup Assistant
- Bootstrap packages
- FileVault escrow
- Profile/declaration management
- Remote lock/wipe

**Datastore Files:**
- `apple_mdm.go`
- `vpp.go`
- `in_house_apps.go`
- `setup_experience.go`
- `scep.go`

**API Endpoints (50+):**

**MDM Protocol:**
- `GET/POST /mdm/apple/mdm` - Apple MDM check-in/command endpoint
- `GET/POST /mdm/apple/enroll` - OTA enrollment
- `GET /mdm/apple/installerpackages/{name}` - Download installer
- `HEAD /mdm/apple/installerpackages/{name}` - Check installer exists

**REST API:**
- `GET /api/v1/fleet/mdm/apple` - Apple MDM status
- `POST /api/v1/fleet/mdm/apple/profiles` - Upload configuration profile
- `GET /api/v1/fleet/mdm/apple/profiles` - List profiles
- `DELETE /api/v1/fleet/mdm/apple/profiles/{id}` - Delete profile
- `GET /api/v1/fleet/mdm/apple/filevault` - FileVault summary
- `POST /api/v1/fleet/mdm/apple/setup/eula` - Upload EULA
- `GET /api/v1/fleet/mdm/apple/setup/eula/{token}` - Download EULA
- `GET /api/v1/fleet/mdm/apple/bootstrap` - Download bootstrap package
- `POST /api/v1/fleet/mdm/apple/bootstrap` - Upload bootstrap
- `DELETE /api/v1/fleet/mdm/apple/bootstrap` - Delete bootstrap
- `POST /api/v1/fleet/mdm/apple/dep/key_pair` - Generate DEP key pair
- `GET /api/v1/fleet/mdm/apple/profiles/summary` - Profile status summary
- `POST /api/v1/fleet/mdm/apple/enqueue` - Enqueue MDM command
- ... (40+ more Apple MDM endpoints)

**Existing Code:**
- ✅ `server/mdm/apple/`
- ✅ `server/mdm/scep/`
- ✅ `server/mdm/nanodep/`
- ✅ `server/mdm/nanomdm/`

**Rationale:**
- Apple MDM is the ONLY way to manage iOS/iPadOS (no agent)
- Already largely modularized

#### 4. Windows Platform Management
**Scope:** Everything related to managing Windows devices

**Tables (13+):**
- mdm_windows_* (2 tables)
- host_mdm_windows_profiles
- windows_mdm_* (4 tables)
- wstep_* (3 tables)
- windows_updates
- Subset of: hosts, host_software, host_updates

**Service Methods (28+):**
- All Windows MDM methods
- Windows updates management
- BitLocker management
- Profile management

**Datastore Files:**
- `microsoft_mdm.go`
- `wstep.go`
- `windows_updates.go`

**API Endpoints (15+):**

**MDM Protocol:**
- `GET/POST /api/mdm/microsoft/management` - MS-MDE protocol endpoint
- `POST /api/mdm/microsoft/discovery` - Discovery service
- `POST /api/mdm/microsoft/auth` - Authentication service
- `POST /api/mdm/microsoft/policy` - Policy service
- `POST /api/mdm/microsoft/enrollment` - Enrollment service
- `POST /api/mdm/microsoft/tos` - Terms of service

**REST API:**
- `GET /api/v1/fleet/mdm/microsoft` - Windows MDM status
- `POST /api/v1/fleet/mdm/microsoft/profiles` - Upload profile
- `GET /api/v1/fleet/mdm/microsoft/profiles` - List profiles
- `DELETE /api/v1/fleet/mdm/microsoft/profiles/{id}` - Delete profile
- `GET /api/v1/fleet/mdm/microsoft/profiles/summary` - Profile summary
- `POST /api/v1/fleet/windows/updates` - Configure Windows Updates
- `GET /api/v1/fleet/windows/updates` - Get Windows Updates config
- ... (more Windows-specific endpoints)

**Existing Code:**
- ✅ `server/mdm/microsoft/` (already modular!)

**Rationale:**
- Windows MDM works independently (customer may not use Fleet agent)
- Already mostly modularized

#### 5. Android Platform Management
**Scope:** Everything related to managing Android devices

**Tables (6+):**
- android_* (3 tables)
- mdm_android_configuration_profiles
- host_mdm_android_profiles
- Subset of: hosts

**Service Methods (5+):**
- Android MDM methods
- Android profiles
- Android Enterprise integration

**Datastore Files:**
- `android.go`
- `android_hosts.go`
- `android_mysql.go`

**API Endpoints (5+):**

**REST API:**
- `GET /api/v1/fleet/mdm/android` - Android MDM status
- `POST /api/v1/fleet/mdm/android/profiles` - Upload profile
- `GET /api/v1/fleet/mdm/android/profiles` - List profiles
- `DELETE /api/v1/fleet/mdm/android/profiles/{id}` - Delete profile
- ... (Android-specific endpoints)

**Existing Code:**
- ✅ `server/mdm/android/` (already modular!)

**Rationale:**
- Currently MDM-only (no agent)
- Future limited agent will be handled by Agent Management context

#### 6. Linux Platform Management
**Scope:** Limited Linux MDM (primarily disk encryption)

**Tables (5+):**
- host_disk_encryption_keys (LUKS)
- host_disk_encryption_keys_archive
- Subset of: hosts, host_software

**Service Methods (2+):**
- Linux MDM methods
- LUKS escrow (handled by Agent Management via orbit protocol)

**Datastore Files:**
- `linux_mdm.go`
- `disk_encryption.go`

**API Endpoints:**
- LUKS escrow handled via `/api/fleet/orbit/luks_data` (Agent Management context)
- No dedicated Linux MDM REST API endpoints yet

**Existing Code:**
- ✅ `server/mdm/linux/` (already modular!)

**Rationale:**
- Minimal MDM, primarily agent-based management

#### 7. Software & Vulnerability Management
**Scope:** Software inventory, installers, and vulnerabilities

**Tables (20+):**
- software (core)
- software_* (11 tables total)
- host_software, host_software_*
- cve_meta
- operating_system_vulnerabilities
- vulnerability_host_counts
- fleet_maintained_apps

**Service Methods (25+):**
- `ListSoftware()`, `SoftwareByID()`
- `ListSoftwareTitles()`, `InstallSoftwareTitle()`
- `UploadSoftwareInstaller()`
- `ListVulnerabilities()`, `Vulnerability()`
- `AddFleetMaintainedApp()`
- `GetSoftwareInstallResult()`

**Datastore Files:**
- `software.go`
- `software_installers.go`
- `software_titles.go`
- `software_title_icons.go`
- `vulnerabilities.go`
- `operating_system_vulnerabilities.go`
- `maintained_apps.go`

**API Endpoints (25+):**

**REST API:**
- `GET /api/v1/fleet/software` - List all software
- `GET /api/v1/fleet/software/count` - Count software
- `GET /api/v1/fleet/software/{id}` - Get software details
- `GET /api/v1/fleet/software/titles` - List software titles
- `GET /api/v1/fleet/software/titles/{id}` - Get title details
- `POST /api/v1/fleet/software/titles/{id}/install` - Install software on host
- `POST /api/v1/fleet/software/installers` - Upload installer
- `GET /api/v1/fleet/software/installers/{id}` - Get installer metadata
- `DELETE /api/v1/fleet/software/installers/{id}` - Delete installer
- `POST /api/v1/fleet/software/installers/batch` - Batch install
- `GET /api/v1/fleet/software/versions` - Software versions
- `GET /api/v1/fleet/vulnerabilities` - List vulnerabilities
- `GET /api/v1/fleet/vulnerabilities/{cve}` - Get vulnerability details
- `GET /api/v1/fleet/os_versions/{id}/vulnerabilities` - OS vulnerabilities
- `POST /api/v1/fleet/maintained_apps` - Add maintained app
- ... (more software endpoints)

**Rationale:**
- Software inventory comes from BOTH agent (osquery tables) AND MDM (VPP, in-house apps)
- Installation can be agent-based (orbit downloads .pkg/.msi) OR MDM-based (VPP/in-house)
- Natural boundary with clear responsibility

#### 8. Security & Compliance
**Scope:** Policies, queries, scripts, activities, conditional access

**Tables (27+):**
- policies, policy_* (5 tables)
- queries, query_* (3 tables)
- distributed_query_* (2 tables)
- scripts, script_* (3 tables)
- batch_activities* (2 tables)
- activities, upcoming_activities
- host_activities
- host_script_results
- microsoft_compliance_partner_* (2 tables)

**Service Methods (52+):**
- Policy methods (global & team)
- Query methods
- Live query campaigns
- Script execution
- Activities logging
- Conditional Access integration (Windows)

**Datastore Files:**
- `policies.go`
- `queries.go`
- `query_results.go`
- `scripts.go`
- `activities.go`
- `campaigns.go`
- `conditional_access_microsoft.go`

**API Endpoints (50+):**

**REST API - Policies:**
- `GET /api/v1/fleet/policies` - List global policies
- `POST /api/v1/fleet/policies` - Create global policy
- `GET /api/v1/fleet/policies/{id}` - Get policy
- `PATCH /api/v1/fleet/policies/{id}` - Update policy
- `POST /api/v1/fleet/policies/delete` - Delete policies
- `GET /api/v1/fleet/teams/{id}/policies` - List team policies
- `POST /api/v1/fleet/teams/{id}/policies` - Create team policy
- ... (more policy endpoints)

**REST API - Queries:**
- `GET /api/v1/fleet/queries` - List queries
- `POST /api/v1/fleet/queries` - Create query
- `GET /api/v1/fleet/queries/{id}` - Get query
- `PATCH /api/v1/fleet/queries/{id}` - Update query
- `DELETE /api/v1/fleet/queries/{id}` - Delete query
- `POST /api/v1/fleet/queries/run` - Run live query
- `GET /api/v1/fleet/queries/{id}/report` - Get query report
- `GET /api/v1/fleet/packs` - List packs
- ... (more query endpoints)

**REST API - Scripts:**
- `GET /api/v1/fleet/scripts` - List scripts
- `POST /api/v1/fleet/scripts` - Upload script
- `DELETE /api/v1/fleet/scripts/{id}` - Delete script
- `POST /api/v1/fleet/scripts/run` - Run script on host
- `GET /api/v1/fleet/scripts/results/{execution_id}` - Get script result
- ... (more script endpoints)

**REST API - Activities:**
- `GET /api/v1/fleet/activities` - List activities (audit log)
- `GET /api/v1/fleet/hosts/{id}/activities` - Host activities
- ... (activity endpoints)

**Rationale:**
- Policies and queries are agent-based (use osquery)
- Scripts can be agent-based (orbit) or MDM-based (MDM commands)
- Natural grouping for security/compliance features

#### 9. Identity & Access Management
**Scope:** Users, teams, SSO, SCIM

**Tables (15+):**
- users, users_deleted
- user_teams
- teams
- sessions
- invites, invite_teams
- password_reset_requests
- email_changes
- verification_tokens
- scim_* (5 tables)
- enroll_secrets

**Service Methods (30+):**
- User management
- Session management
- SSO/SAML
- SCIM provisioning
- Team management
- Invites

**Datastore Files:**
- `users.go`
- `teams.go`
- `sessions.go`
- `invites.go`
- `scim.go`
- `password_reset.go`
- `email_changes.go`

**API Endpoints (40+):**

**REST API - Users:**
- `GET /api/v1/fleet/users` - List users
- `POST /api/v1/fleet/users/admin` - Create user
- `GET /api/v1/fleet/users/{id}` - Get user
- `PATCH /api/v1/fleet/users/{id}` - Update user
- `DELETE /api/v1/fleet/users/{id}` - Delete user
- `POST /api/v1/fleet/change_password` - Change password
- `POST /api/v1/fleet/users/{id}/require_password_reset` - Require password reset
- `GET /api/v1/fleet/me` - Current user
- ... (more user endpoints)

**REST API - Teams:**
- `GET /api/v1/fleet/teams` - List teams
- `POST /api/v1/fleet/teams` - Create team
- `GET /api/v1/fleet/teams/{id}` - Get team
- `PATCH /api/v1/fleet/teams/{id}` - Update team
- `DELETE /api/v1/fleet/teams/{id}` - Delete team
- `GET /api/v1/fleet/teams/{id}/users` - List team users
- `PATCH /api/v1/fleet/teams/{id}/users` - Add users to team
- `DELETE /api/v1/fleet/teams/{id}/users` - Remove users from team
- ... (more team endpoints)

**REST API - Auth:**
- `POST /api/v1/fleet/login` - Login
- `POST /api/v1/fleet/logout` - Logout
- `GET /api/v1/fleet/sso` - SSO settings
- `POST /api/v1/fleet/invites` - Create invite
- `GET /api/v1/fleet/invites` - List invites
- ... (more auth endpoints)

**REST API - SCIM:**
- `GET /api/v1/fleet/scim/v2/Users` - List SCIM users
- `POST /api/v1/fleet/scim/v2/Users` - Create SCIM user
- `GET /api/v1/fleet/scim/v2/Users/{id}` - Get SCIM user
- `PATCH /api/v1/fleet/scim/v2/Users/{id}` - Update SCIM user
- `DELETE /api/v1/fleet/scim/v2/Users/{id}` - Delete SCIM user
- ... (more SCIM endpoints)

**Rationale:**
- Platform-independent
- Clear responsibility
- Well-defined boundaries

#### 10. Platform Services (Shared)
**Scope:** Cross-cutting infrastructure

**Tables (10+):**
- app_config_json
- default_team_config_json
- fleet_variables
- secret_variables
- aggregated_stats
- statistics
- cron_stats
- jobs
- locks
- calendar_events*

**Service Methods (20+):**
- AppConfig
- CronSchedules
- Calendar integration
- Statistics
- Fleet Desktop config

**Datastore Files:**
- `app_configs.go`
- `aggregated_stats.go`
- `statistics.go`
- `cron_stats.go`
- `jobs.go`
- `locks.go`
- `calendar_events.go`
- `secret_variables.go`

**API Endpoints (15+):**

**REST API:**
- `GET /api/v1/fleet/config` - Get app config
- `PATCH /api/v1/fleet/config` - Update app config
- `GET /api/v1/fleet/version` - Fleet version
- `GET /api/v1/fleet/statistics` - Fleet statistics
- `POST /api/v1/fleet/webhooks/calendar` - Calendar webhook
- `GET /api/v1/fleet/device/{token}` - Fleet Desktop endpoint
- `GET /api/v1/fleet/device/{token}/desktop` - Fleet Desktop data
- ... (more infrastructure endpoints)

**Rationale:**
- Infrastructure needed by all contexts

---

### API Architecture for Proposal 1

**Challenge: Cross-platform unified API endpoints**

Fleet is creating **common API endpoints that work across platforms**. Examples:
- `POST /api/v1/fleet/mdm/profiles` - Upload profile (works for Apple, Windows, Android)
- `GET /api/v1/fleet/mdm/profiles` - List profiles (across all platforms)
- `GET /api/v1/fleet/software` - List software (from agent + all MDM sources)

**Approach: Requires orchestration layer**

```
server/api/
  - Orchestration/routing layer for cross-platform endpoints
  - Determines target platform and routes to appropriate context

  Example:
    POST /api/v1/fleet/mdm/profiles
      → api.ProfileHandler parses request
      → Determines platform from profile format or host context
      → Routes to: applemdm.Service OR winmdm.Service OR androidmdm.Service
      → Returns unified response

Platform-specific endpoints still owned by contexts:
  /api/mdm/apple/mdm → Apple Platform Management (MDM protocol)
  /api/mdm/microsoft/management → Windows Platform Management (MDM protocol)
  /api/osquery/* → Agent Management
  /api/fleet/orbit/* → Agent Management
```

**API Gateway architecture:**

```go
// server/api/gateway.go
type Gateway struct {
    appleMDM   applemdm.Service
    windowsMDM winmdm.Service
    androidMDM androidmdm.Service
    // ...
}

func (g *Gateway) UploadProfile(ctx context.Context, req ProfileUploadRequest) error {
    // Determine platform from profile content or host ID
    platform := determinePlatform(req)

    switch platform {
    case "darwin", "ios":
        return g.appleMDM.UploadProfile(ctx, req)
    case "windows":
        return g.windowsMDM.UploadProfile(ctx, req)
    case "android":
        return g.androidMDM.UploadProfile(ctx, req)
    }
}
```

**Cross-context communication:**
- Bounded contexts call each other via service interfaces (dependency injection)
- API Gateway orchestrates cross-platform operations
- No direct database access across contexts

**Implication:** Proposal 1 needs an orchestration layer for cross-platform API endpoints, adding complexity

### Pros

✅ **Aligns with product structure** - Fleet sells "MDM for macOS", "MDM for Windows", etc.  
✅ **Clear team ownership** - Platform teams own their contexts  
✅ **Existing modularization** - MDM is already mostly organized this way  
✅ **Customer-centric** - Easy to explain to customers  
✅ **Independent deployment** - Could modify Windows features without touching Apple code  
✅ **Scales with platforms** - Easy to add new platforms  
✅ **Agent independence recognized** - Agent Management is cross-platform, works without MDM  

### Cons

❌ **Requires API gateway** - Cross-platform endpoints need orchestration layer (POST /api/v1/fleet/mdm/profiles)  
❌ **Platform detection logic** - Gateway must determine platform from request (profile format, host ID)  
❌ **Cross-platform duplication** - Host management logic duplicated per platform  
❌ **Shared concerns scattered** - Software management touches all platforms  
❌ **Complex queries** - "Show all hosts" requires joining multiple contexts  
❌ **Agent Management complexity** - Works with all platform contexts, creates many dependencies
❌ **Gateway becomes bottleneck** - All cross-platform endpoints funnel through one layer

### Module dependency map

```
┌──────────────────────────────────────────────────────────────┐
│                     Platform Services                         │
│           (Config, Calendar, Jobs, Infrastructure)            │
└─────────────────────┬────────────────────────────────────────┘
                      │
       ┌──────────────┼──────────────┬──────────────┐
       │              │              │              │
┌──────▼───────┐ ┌───▼───────┐ ┌────▼──────┐ ┌────▼──────────┐
│  Identity &  │ │   Host &  │ │   Agent   │ │   Software    │
│    Access    │ │  Device   │ │ Management│ │      &        │
│ Management   │ │   Core    │ │           │ │Vulnerability  │
└──────────────┘ └────┬──────┘ └───┬───────┘ └───────────────┘
                      │            │
       ┌──────────────┴────────────┴──────────────┐
       │                                           │
┌──────▼───────┐ ┌────────────┐ ┌────────────┐ ┌──▼────────────┐
│    Apple     │ │  Windows   │ │  Android   │ │    Linux      │
│   Platform   │ │  Platform  │ │  Platform  │ │   Platform    │
│  Management  │ │ Management │ │ Management │ │  Management   │
└──────┬───────┘ └────┬───────┘ └────┬───────┘ └───┬───────────┘
       │              │              │              │
       └──────────────┴──────┬───────┴──────────────┘
                             │
                      ┌──────▼──────────┐
                      │   Security &    │
                      │   Compliance    │
                      └─────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                       API Gateway                            │
│  (Routes cross-platform endpoints to appropriate contexts)  │
└─────────────────────────────────────────────────────────────┘
         │           │            │            │
         ▼           ▼            ▼            ▼
   (Routes to Apple, Windows, Android, Software, etc.)
```

**Key architectural principles:**
1. **Host & Device Core is foundational** - All contexts depend on it for host lifecycle
2. **Agent Management is cross-platform** - Separate from platform-specific MDM
3. **Platform contexts are independent** - Apple, Windows, Android, Linux don't depend on each other
4. **API Gateway is required** - Handles cross-platform endpoints like `/api/v1/fleet/mdm/profiles`
5. **Security & Compliance** - Uses both agent (policies, queries, scripts) and platform contexts

---

## Proposal 2: Capability-Centric (Domain-Driven Design)

**Philosophy:** Organize by business capability, with Agent Management as foundation

### Bounded Contexts

#### 1. Agent Management (Foundation)
**Scope:** Osquery and Orbit protocols - platform-agnostic agent communication

*[Same as Proposal 1 - see above for full details]*

**Key difference:** In this proposal, Agent Management is recognized as the **foundational** capability that enables all other capabilities on agent-supported platforms.

#### 2. Device Enrollment & Lifecycle
**Scope:** Device enrollment, inventory, lifecycle (both agent and MDM)

**Tables (45+):**
- hosts (core)
- host_additional, host_batteries, host_certificates*, host_disks, host_display_names, host_emails, host_orbit_info, host_seen_times, host_users
- network_interfaces
- operating_systems, kernel_host_counts
- challenges
- enroll_secrets
- abm_tokens, host_dep_assignments
- android_* (for Android enrollment)
- mobile_device_management_solutions

**Service Methods (35+):**
- `EnrollOsquery()`, `EnrollOrbit()` - Agent enrollment (delegated to Agent Management)
- `EnrollMDMAppleDevice()` - Apple MDM enrollment
- `EnrollMDMWindowsDevice()` - Windows MDM enrollment
- `EnrollMDMAndroidDevice()` - Android enrollment
- `ListHosts()`, `GetHost()`, `DeleteHost()`
- `HostByIdentifier()`, `RefetchHost()`
- `AddHostsToTeam()`, `TransferHosts()`
- `GetHostDEPAssignment()`
- Label management
- Operating system tracking

**Datastore Files:**
- `hosts.go`
- `labels.go`
- `operating_systems.go`
- `challenges.go`
- Parts of `apple_mdm.go` (enrollment), `microsoft_mdm.go` (enrollment), `android.go`

**API Endpoints (35+):**

**Agent enrollment (delegated to Agent Management):**
- `POST /api/osquery/enroll`
- `POST /api/fleet/orbit/enroll`

**MDM enrollment:**
- `GET/POST /mdm/apple/enroll` - Apple OTA enrollment
- `POST /mdm/microsoft/enrollment` - Windows enrollment
- Android enrollment (via Android Enterprise API)

**REST API:**
- `GET /api/v1/fleet/hosts` - List all hosts (agent + MDM)
- `GET /api/v1/fleet/hosts/{id}` - Get host details
- `DELETE /api/v1/fleet/hosts/{id}` - Delete host
- `POST /api/v1/fleet/hosts/transfer` - Transfer to team
- `POST /api/v1/fleet/hosts/{id}/refetch` - Refetch
- `GET /api/v1/fleet/os_versions` - OS versions
- `GET /api/v1/fleet/labels` - Labels
- ... (more host lifecycle endpoints)

**Rationale:**
- All platforms enroll devices (agent or MDM)
- Host lifecycle is same conceptual process
- Centralizes inventory management

#### 3. Configuration Management
**Scope:** MDM profiles, declarations, configuration delivery (all platforms)

**Tables (35+):**
- mdm_apple_configuration_profiles, mdm_apple_declarations
- mdm_windows_configuration_profiles
- mdm_android_configuration_profiles
- host_mdm_apple_profiles, host_mdm_apple_declarations
- host_mdm_windows_profiles
- host_mdm_android_profiles
- mdm_configuration_profile_* (2 tables)
- mdm_declaration_labels
- mdm_delivery_status
- nano_* (9 tables) - Command infrastructure
- windows_mdm_* (4 tables)
- scep_*, wstep_* - Certificate provisioning

**Service Methods (50+):**
- `NewMDMAppleConfigProfile()`, `NewMDMAppleDeclaration()`
- `NewMDMWindowsConfigProfile()`
- `NewMDMAndroidConfigProfile()`
- `GetMDMConfigProfileStatus()`
- `ResendHostMDMProfile()`
- `BatchSetMDMProfiles()`
- `ListMDMConfigProfiles()`

**Datastore Files:**
- Parts of `apple_mdm.go` (profiles), `microsoft_mdm.go` (profiles), `android.go` (profiles)
- `nanomdm_storage.go`

**API Endpoints (40+):**

**MDM Protocol:**
- `GET/POST /mdm/apple/mdm` - Apple MDM protocol
- `GET/POST /api/mdm/microsoft/management` - Windows MDM protocol

**REST API:**
- `POST /api/v1/fleet/mdm/apple/profiles` - Upload Apple profile
- `GET /api/v1/fleet/mdm/apple/profiles` - List Apple profiles
- `POST /api/v1/fleet/mdm/windows/profiles` - Upload Windows profile
- `GET /api/v1/fleet/mdm/windows/profiles` - List Windows profiles
- `POST /api/v1/fleet/mdm/android/profiles` - Upload Android profile
- `GET /api/v1/fleet/mdm/profiles/summary` - Cross-platform profile summary
- ... (more configuration endpoints)

**Rationale:**
- Configuration delivery is same pattern across all MDM platforms
- Profiles/declarations are conceptually the same (XML vs JSON format difference)

#### 4. Software Lifecycle Management
**Scope:** Software inventory, deployment, updates (agent + MDM sources)

**Tables (35+):**
- software (core), software_* (11 tables)
- host_software, host_software_*
- software_installers
- vpp_* (8 tables) - Apple VPP
- in_house_apps (3 tables) - iOS in-house apps
- fleet_maintained_apps
- host_vpp_software_installs
- host_in_house_software_installs
- host_software_installs
- setup_experience_* (2 tables)
- windows_updates

**Service Methods (30+):**
- `ListSoftware()`, `ListSoftwareTitles()`
- `InstallSoftwareTitle()` - Works for agent-based (.pkg, .msi via orbit) AND MDM-based (VPP, in-house)
- `UninstallSoftwareTitle()`
- `UploadSoftwareInstaller()`
- `BatchSetSoftwareInstallers()`
- VPP management
- In-house app management
- Setup experience software
- Windows Updates

**Datastore Files:**
- `software.go`, `software_installers.go`, `software_titles.go`, `software_title_icons.go`
- `vpp.go`, `in_house_apps.go`
- `setup_experience.go`
- `windows_updates.go`
- `maintained_apps.go`

**API Endpoints (30+):**

**Agent protocol (software install via orbit):**
- `POST /api/fleet/orbit/software_install/result` - Agent reports result
- `POST /api/fleet/orbit/software_install/package` - Agent downloads installer
- `POST /api/fleet/orbit/software_install/details` - Get install details

**REST API:**
- `GET /api/v1/fleet/software` - List all software (agent + MDM sources)
- `GET /api/v1/fleet/software/titles` - Software titles
- `POST /api/v1/fleet/software/titles/{id}/install` - Install (routes to agent OR MDM)
- `POST /api/v1/fleet/software/installers` - Upload installer
- `POST /api/v1/fleet/mdm/apple/vpp` - VPP management
- `POST /api/v1/fleet/mdm/apple/in_house_apps` - In-house apps
- `GET /api/v1/fleet/windows/updates` - Windows Updates
- ... (more software endpoints)

**Rationale:**
- Software inventory comes from agent (osquery tables) AND MDM (VPP, in-house apps, Windows Updates)
- Installation can be agent-based OR MDM-based, but same business process
- Unified software catalog regardless of source

#### 5. Security & Compliance
**Scope:** Policies, vulnerabilities, encryption, conditional access

**Tables (30+):**
- policies, policy_* (5 tables)
- cve_meta, operating_system_vulnerabilities, vulnerability_host_counts
- host_disk_encryption_keys*
- mdm_idp_accounts, host_mdm_idp_accounts
- microsoft_compliance_partner_*
- certificate_authorities, ca_config_assets
- host_identity_scep_*

**Service Methods (40+):**
- Policy methods (create, list, modify, delete)
- `ListVulnerabilities()`, `Vulnerability()`
- Disk encryption (FileVault, BitLocker, LUKS)
- Certificate authority management
- Conditional Access integration

**Datastore Files:**
- `policies.go`
- `vulnerabilities.go`, `operating_system_vulnerabilities.go`
- `disk_encryption.go`
- `certificate_authorities.go`, `ca_config_assets.go`
- `conditional_access_microsoft.go`

**API Endpoints (30+):**

**Agent protocol (policy checks via osquery, disk encryption via orbit):**
- Policies evaluated via `/api/osquery/distributed/read` (returns policy queries)
- Results via `/api/osquery/distributed/write`
- `POST /api/fleet/orbit/disk_encryption_key` - Escrow keys
- `POST /api/fleet/orbit/luks_data` - Escrow LUKS

**REST API:**
- `GET /api/v1/fleet/policies` - List policies
- `POST /api/v1/fleet/policies` - Create policy
- `GET /api/v1/fleet/vulnerabilities` - List vulnerabilities
- `GET /api/v1/fleet/mdm/apple/filevault` - FileVault status
- `GET /api/v1/fleet/mdm/microsoft/bitlocker` - BitLocker status
- `POST /api/v1/fleet/certificate_authorities` - CA management
- ... (more security endpoints)

**Rationale:**
- Security posture is evaluated the same way across platforms
- Policies use agent (osquery queries)
- Disk encryption uses both agent (escrow) and MDM (enforce)

#### 6. Query & Reporting
**Scope:** Live queries, scheduled queries, reporting (agent-based)

**Tables (15+):**
- queries, query_* (3 tables)
- distributed_query_* (2 tables)
- query_results
- scheduled_queries, scheduled_query_stats
- packs, pack_targets
- osquery_options
- labels, label_membership

**Service Methods (25+):**
- Query CRUD
- `NewDistributedQueryCampaign()`
- `StreamCampaignResults()`
- `RunLiveQuery()`
- Scheduled query management
- Pack management
- Label management

**Datastore Files:**
- `queries.go`, `query_results.go`
- `campaigns.go`
- `scheduled_queries.go`
- `packs.go`
- `labels.go`

**API Endpoints (25+):**

**Agent protocol:**
- `POST /api/osquery/distributed/read` - Agent gets queries
- `POST /api/osquery/distributed/write` - Agent submits results

**REST API:**
- `GET /api/v1/fleet/queries` - List queries
- `POST /api/v1/fleet/queries` - Create query
- `POST /api/v1/fleet/queries/run` - Run live query
- `GET /api/v1/fleet/queries/{id}/report` - Query report
- `GET /api/v1/fleet/packs` - List packs
- `POST /api/v1/fleet/packs` - Create pack
- `GET /api/v1/fleet/schedule` - Scheduled queries
- ... (more query endpoints)

**Rationale:**
- Querying is osquery-based (requires agent)
- Clear domain boundary
- Core differentiator for Fleet

#### 7. Automation & Scripts
**Scope:** Script execution, batch operations (agent + MDM)

**Tables (10+):**
- scripts, script_* (3 tables)
- batch_activities* (2 tables)
- host_script_results
- setup_experience_scripts
- setup_experience_status_results

**Service Methods (20+):**
- `RunHostScript()`, `SaveHostScriptResult()`
- `NewScript()`, `UpdateScript()`, `DeleteScript()`
- `BatchScriptExecute()`, `BatchScriptCancel()`
- `LockHost()`, `UnlockHost()`, `WipeHost()` - Can be script-based OR MDM command

**Datastore Files:**
- `scripts.go`
- Parts of `setup_experience.go`

**API Endpoints (15+):**

**Agent protocol:**
- `POST /api/fleet/orbit/scripts/request` - Agent gets script
- `POST /api/fleet/orbit/scripts/result` - Agent submits result

**REST API:**
- `GET /api/v1/fleet/scripts` - List scripts
- `POST /api/v1/fleet/scripts` - Upload script
- `POST /api/v1/fleet/scripts/run` - Run script
- `GET /api/v1/fleet/scripts/results/{id}` - Get result
- `POST /api/v1/fleet/hosts/{id}/lock` - Lock host (MDM command or script)
- `POST /api/v1/fleet/hosts/{id}/wipe` - Wipe host (MDM command or script)
- ... (more script endpoints)

**Rationale:**
- Automation workflows same pattern across platforms
- Scripts can run via agent (orbit) or MDM (run-script MDM command)

#### 8. Identity & Access
**Scope:** Users, teams, authentication, authorization

*[Same as Proposal 1 - see above for full details]*

#### 9. Activity & Audit
**Scope:** Audit trail, calendar, integrations

**Tables (10+):**
- activities, upcoming_activities
- host_activities
- calendar_events, host_calendar_events
- aggregated_stats, statistics
- cron_stats

**Service Methods (15+):**
- `NewActivity()`, `ListActivities()`
- `ListHostUpcomingActivities()`, `ListHostPastActivities()`
- `CalendarWebhook()`
- External integrations (Jira, Zendesk)

**Datastore Files:**
- `activities.go`
- `calendar_events.go`
- `aggregated_stats.go`, `statistics.go`
- `cron_stats.go`

**API Endpoints (10+):**

**REST API:**
- `GET /api/v1/fleet/activities` - List activities (audit log)
- `GET /api/v1/fleet/hosts/{id}/activities` - Host activities
- `POST /api/v1/fleet/webhooks/calendar` - Calendar webhook
- `GET /api/v1/fleet/statistics` - Statistics
- ... (more audit endpoints)

**Rationale:**
- Auditing is cross-cutting concern
- All actions logged here

#### 10. Platform Core (Shared)
**Scope:** Configuration, infrastructure

**Tables (15+):**
- app_config_json, default_team_config_json
- fleet_variables, secret_variables
- jobs, locks
- munki_issues
- mobile_device_management_solutions

**Service Methods (10+):**
- `AppConfig()`
- `CronSchedules()`
- `Version()`, `License()`

**Datastore Files:**
- `app_configs.go`
- `jobs.go`, `locks.go`
- `secret_variables.go`

**API Endpoints (10+):**

**REST API:**
- `GET /api/v1/fleet/config` - App config
- `PATCH /api/v1/fleet/config` - Update config
- `GET /api/v1/fleet/version` - Version
- `GET /api/v1/fleet/device/{token}` - Fleet Desktop
- ... (more infrastructure endpoints)

**Rationale:**
- Infrastructure needed by all contexts

---

### API Architecture for Proposal 2

**Approach: Each bounded context owns its API handlers**

Each bounded context has its own HTTP handlers/controllers. No separate API gateway layer.

**Cross-context orchestration:**

When an API endpoint needs multiple contexts, the **owning context orchestrates**

**Key principles:**
- Contexts depend on other contexts via service interfaces (dependency injection)
- Business logic (including orchestration) lives in the service layer, not handlers

**Benefits:**
- Clear ownership: endpoint → context → orchestration responsibility
- No extra gateway layer to maintain
- Easier to test (service layer has no HTTP dependencies)
- **Cross-platform operations handled naturally** by capability contexts

**Drawbacks:**
- Contexts may have many dependencies on other contexts
- Orchestration logic distributed across contexts (not centralized)

### Pros

✅ **Business-aligned** - Matches how customers think about capabilities  
✅ **DRY principle** - Software installation has one path (routes internally to agent or MDM)  
✅ **Clear responsibilities** - Each context has single purpose  
✅ **Cross-platform from start** - Policies work on all agent platforms naturally  
✅ **Agent Management as foundation** - Explicitly recognized as enabling layer  
✅ **Easier testing** - Mock one software installer interface for all platforms  

### Cons

❌ **Platform expertise scattered** - Apple MDM knowledge split across Device, Configuration, Software contexts  
❌ **Complex platform features** - Setup Assistant touches multiple contexts  
❌ **Vendor coupling** - Apple VPP doesn't map cleanly to generic "Software"  
❌ **Migration complexity** - Current MDM modules don't align  
❌ **API complexity** - "Install software" API same, but implementation very different per platform  
❌ **Agent dependency** - Many capabilities (Queries, Policies) only work with agent, not MDM-only hosts  

### Module dependency map

```
┌──────────────────────────────────────────────────────────────┐
│                     Platform Services                         │
│           (Config, Calendar, Jobs, Infrastructure)            │
└─────────────────────┬────────────────────────────────────────┘
                      │
       ┌──────────────┼──────────────┐
       │              │              │
┌──────▼───────┐ ┌───▼────────┐ ┌────▼──────────┐
│  Identity &  │ │   Agent    │ │   Software    │
│    Access    │ │ Management │ │   Lifecycle   │
│ Management   │ │(Foundation)│ │  Management   │
└──────────────┘ └───┬────────┘ └───────────────┘
                     │
       ┌─────────────┴─────────────┬──────────────┐
       │                           │              │
┌──────▼───────┐ ┌────────────────▼───┐ ┌────────▼──────────┐
│    Device    │ │  Configuration     │ │    Policy &       │
│ Enrollment & │ │   Management       │ │   Compliance      │
│  Lifecycle   │ │                    │ │   Management      │
└──────┬───────┘ └─────┬──────────────┘ └────────┬──────────┘
       │               │                         │
       │   ┌───────────┴─────────────────────────┴────────────────────┐
       │   │                                                          │
       │   │  (Each capability context orchestrates across platforms) │
       │   │                                                          │
       │   │    Platform adapters (internal to contexts):             │
       │   │    - Apple MDM implementation                            │
       │   │    - Windows MDM implementation                          │
       │   │    - Android MDM implementation                          │
       │   │                                                          │
       └───┴──────────────────────────────────────────────────────────┘
                     │
              ┌──────▼──────────┐
              │    Query &      │
              │   Reporting     │
              └─────────────────┘
```

**Key architectural principles:**
1. **Agent Management is foundational** - All agent-based capabilities depend on it
2. **Capability contexts orchestrate** - Each owns endpoints and orchestrates across platforms
3. **Platform logic distributed** - Apple/Windows/Android MDM code lives within capability contexts
4. **No central gateway** - Each context has its own handlers and orchestrates internally
5. **Cross-platform by design** - "Software Lifecycle" naturally handles agent + Apple VPP + Windows apps
6. **Device Enrollment & Lifecycle** - Handles all enrollment mechanisms (agent, Apple MDM, Windows MDM, Android)

---

## Proposal 3: Hybrid (pragmatic evolution)

**Philosophy:** Start from where we are - keep existing MDM modules, extract agent and cross-cutting concerns

### Bounded Contexts

#### 1. Host Management
**Scope:** Platform-agnostic host lifecycle

*[Same as Proposal 1 - see above for details]*

**Rationale:**
- Every MDM context depends on this
- Large (hosts.go is massive), high impact
- Foundation for all device management

#### 2. Agent Management
**Scope:** Osquery and Orbit protocols - platform-agnostic agent communication

*[Same as Proposal 1 - see above for full details]*

**Rationale:**
- Works independently of all MDM contexts
- Platform-agnostic foundation
- Enables mixed deployments (Fleet agent + competitor MDM)
- Supports agent-only platforms (ChromeOS)
- Clear protocol boundaries

#### 3. Apple MDM
**Scope:** Everything Apple MDM

*[Same as Proposal 1 - see above for details]*

**Current state:** ✅ Already mostly modular (`server/mdm/apple/`)

**Rationale:**
- Apple MDM is only way to manage iOS/iPadOS (no agent)
- Already mostly modularized
- Proven success - leave as-is, add clear interface boundaries

#### 4. Windows MDM
**Scope:** Everything Windows MDM

*[Same as Proposal 1 - see above for details]*

**Current state:** ✅ Already mostly modular (`server/mdm/microsoft/`)

**Rationale:**
- Proven success - leave as-is, add clear interface boundaries

#### 5. Android MDM
**Scope:** Everything Android MDM

*[Same as Proposal 1 - see above for details]*

**Current state:** ✅ Already modular (`server/mdm/android/`)

**Rationale:**
- Currently MDM-only (no agent)
- Future limited agent will be handled by Agent Management context
- Already modularized - leave as-is

#### 6. Software Management
**Scope:** Software inventory and deployment (agent + MDM sources)

**Tables (25+):**
- software (core), software_* (11 tables)
- host_software, host_software_*
- software_installers
- fleet_maintained_apps
- cve_meta, operating_system_vulnerabilities, vulnerability_host_counts

**Note:** VPP and in-house apps stay in Apple MDM context

**Service Methods (25+):**
- `ListSoftware()` - Returns software from agent (osquery) AND MDM (VPP, in-house, Windows apps)
- `ListSoftwareTitles()`
- `InstallSoftwareTitle()` - Routes to agent mechanism OR MDM mechanism based on platform and source
- Software installers (`.pkg`, `.msi`, `.deb`)
- Vulnerability scanning
- Maintained apps

**Datastore Files:**
- `software.go`, `software_installers.go`, `software_titles.go`
- `vulnerabilities.go`, `operating_system_vulnerabilities.go`
- `maintained_apps.go`

**API Endpoints (25+):**
- `GET /api/v1/fleet/software` - List software (unified view)
- `GET /api/v1/fleet/software/titles` - Software titles
- `POST /api/v1/fleet/software/titles/{id}/install` - Install (orchestrates)
- `POST /api/v1/fleet/software/installers` - Upload installer
- `GET /api/v1/fleet/vulnerabilities` - Vulnerabilities
- ... (more)

**Rationale:**
- Clear boundary
- High value (reduces complexity)
- Touches all platforms but in well-defined way
- Orchestrates between agent and MDM sources

#### 7. Policy & Compliance
**Scope:** Policy management (agent-based via osquery)

**Tables (5+):**
- policies, policy_* (5 tables)

**Service Methods (20+):**
- Policy CRUD (global & team)
- Policy automation
- Failing policy webhooks

**Datastore Files:**
- `policies.go`

**API Endpoints (20+):**

**Agent protocol:**
- Policies delivered via `/api/osquery/distributed/read` as queries
- Results via `/api/osquery/distributed/write`

**REST API:**
- `GET /api/v1/fleet/policies` - List global policies
- `POST /api/v1/fleet/policies` - Create global policy
- `GET /api/v1/fleet/policies/{id}` - Get policy
- `PATCH /api/v1/fleet/policies/{id}` - Update policy
- `POST /api/v1/fleet/policies/delete` - Delete policies
- `GET /api/v1/fleet/teams/{id}/policies` - Team policies
- `POST /api/v1/fleet/teams/{id}/policies` - Create team policy
- `GET /api/v1/fleet/policies/count` - Policy count
- `POST /api/v1/fleet/spec/policies` - Apply policy spec
- ... (more policy endpoints)

**Rationale:**
- Policies are agent-based (osquery queries)
- Distinct from queries conceptually
- Natural boundary

#### 8. Query & Reporting
**Scope:** Query execution infrastructure (agent-based via osquery)

**Tables (12+):**
- queries, query_*, distributed_query_*
- scheduled_queries*, packs, pack_targets
- query_results, osquery_options

**Service Methods (25+):**
- Query CRUD
- Live query campaigns
- Scheduled queries, packs

**Datastore Files:**
- `queries.go`, `query_results.go`
- `campaigns.go`
- `scheduled_queries.go`, `packs.go`

**API Endpoints (25+):**

**Agent protocol:**
- `POST /api/osquery/distributed/read` - Agent gets queries
- `POST /api/osquery/distributed/write` - Agent submits results

**REST API:**
- `GET /api/v1/fleet/queries` - List queries
- `POST /api/v1/fleet/queries` - Create query
- `POST /api/v1/fleet/queries/run` - Run live query
- `GET /api/v1/fleet/queries/{id}/report` - Query report
- `GET /api/v1/fleet/packs` - List packs
- `POST /api/v1/fleet/packs` - Create pack
- ... (more query endpoints)

**Rationale:**
- Osquery is core differentiator
- Agent-based capability
- Clear domain boundary

#### 9. Automation & Scripts
**Scope:** Script execution

**Tables (5+):**
- scripts, script_*, batch_activities*
- host_script_results

**Service Methods (15+):**
- Script execution
- Batch operations

**Datastore Files:**
- `scripts.go`

**API Endpoints (15+):**

**Agent protocol:**
- `POST /api/fleet/orbit/scripts/request` - Get script
- `POST /api/fleet/orbit/scripts/result` - Submit result

**REST API:**
- `GET /api/v1/fleet/scripts` - List scripts
- `POST /api/v1/fleet/scripts` - Upload script
- `POST /api/v1/fleet/scripts/run` - Run script
- `GET /api/v1/fleet/scripts/results/{id}` - Get result
- ... (more script endpoints)

**Rationale:**
- Growing feature
- Agent-based (orbit protocol)
- Clear boundary

#### 10. Activity & Audit
**Scope:** Activity logging, audit trail, compliance reporting

**Tables (3+):**
- activities, upcoming_activities
- host_activities

**Service Methods (10+):**
- `NewActivity()` - Create activity log entry
- `ListActivities()` - Query audit trail
- `ListHostActivities()` - Host-specific activities
- `ListHostUpcomingActivities()` - Scheduled host activities
- `ListHostPastActivities()` - Historical host activities

**Datastore Files:**
- `activities.go`

**API Endpoints (5+):**

**REST API:**
- `GET /api/v1/fleet/activities` - List activities (audit log, paginated, filtered)
- `GET /api/v1/fleet/hosts/{id}/activities` - Host activities
- `GET /api/v1/fleet/hosts/{id}/activities/upcoming` - Upcoming host activities
- ... (more activity endpoints)

**Rationale:**
- **Cross-cutting concern** - All bounded contexts write activities (users, hosts, policies, MDM, software, etc.)
- **Compliance requirement** - Audit trail needed for SOC 2, HIPAA, etc.
- **Significant enough** - Warrants its own bounded context rather than being buried in Platform Services
- **Clear API surface** - Dedicated endpoints for querying audit logs
- **Event-driven** - Contexts publish events, Activity & Audit subscribes and logs them

#### 11. Identity & Teams
**Scope:** Users, teams, auth

*[Same as Proposal 1 - see above for details]*

**Rationale:**
- Platform-independent
- Well-defined boundaries
- Already relatively clean

#### 12. Platform Services
**Scope:** Shared infrastructure

*[Same as Proposal 1 - see above for details]*

**Rationale:**
- Infrastructure needed by all contexts
- Cross-cutting concerns

---

### Module dependency map

```
┌──────────────────────────────────────────────────────────────┐
│                     Platform Services                         │
│           (Config, Calendar, Jobs, Infrastructure)            │
└─────────────────────┬────────────────────────────────────────┘
                      │
       ┌──────────────┼──────────────┬──────────────┐
       │              │              │              │
┌──────▼───────┐ ┌───▼───────┐ ┌────▼──────┐ ┌────▼──────────┐
│  Identity &  │ │   Agent   │ │   Host    │ │   Software    │
│    Teams     │ │ Management│ │ Management│ │  Management   │
└──────┬───────┘ └───┬───────┘ └────┬──────┘ └────┬──────────┘
       │             │              │              │
       └─────────────┴──────┬───────┴──────────────┘
                            │
         ┌──────────────────┼───────────────────────┐
         │                  │                       │
    ┌────▼───────┐  ┌───────▼────────┐  ┌──────────▼──────────┐
    │ Apple MDM  │  │ Windows MDM    │  │   Android MDM       │
    │            │  │                │  │                     │
    └────┬───────┘  └───────┬────────┘  └──────────┬──────────┘
         │                  │                       │
         └──────────────────┴───────┬───────────────┘
                                    │
               ┌────────────────────┴───────────────┐
               │                                    │
        ┌──────▼────────┐                   ┌──────▼──────────┐
        │  Policy &     │                   │  Query &        │
        │  Compliance   │                   │  Reporting      │
        └───────┬───────┘                   └──────┬──────────┘
                │                                  │
                └──────────────┬───────────────────┘
                               │
                        ┌──────▼──────────┐
                        │  Automation &   │
                        │    Scripts      │
                        └─────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Activity & Audit                          │
│     (Cross-cutting: ALL contexts publish events here)       │
└─────────────────────────────────────────────────────────────┘
         ▲           ▲            ▲            ▲
         │           │            │            │
    (Observes events from all contexts above)
```

**Key architectural principles:**
1. **Host Management is foundational** - All contexts depend on it
2. **Agent Management is cross-platform** - Enables agent-based capabilities
3. **MDM contexts are independent** - Don't depend on each other
4. **Policy/Query/Scripts** are agent-based - Depend on Agent Management
5. **Software Management is hybrid** - Receives data from agent AND MDM
6. **Activity & Audit is cross-cutting** - All contexts publish domain events to it (observer pattern)

---

### API Architecture for Proposal 3

**Challenge: Cross-platform unified API endpoints**

Same as Proposal 1 - Fleet has common endpoints like `POST /api/v1/fleet/mdm/profiles` that work across platforms.

**Approach: Orchestration contexts handle cross-platform endpoints**

**Orchestration contexts for cross-platform endpoints:**
- **Software Management context**: Handles `/api/v1/fleet/software/*` - orchestrates to agent OR platform MDM
- **MDM API module** (thin layer): Handles `/api/v1/fleet/mdm/profiles` - routes to Apple/Windows/Android MDM contexts
- **Host Management context**: Handles `/api/v1/fleet/hosts/*` - orchestrates host lifecycle

**Advantage over Proposal 1:**
- Cross-platform endpoints live in **domain-specific orchestration contexts** (Software Management, Host Management)
- Only MDM-specific cross-platform endpoints need a thin MDM API module
- Less gateway overhead - most orchestration happens in extracted contexts that have domain knowledge

### Pros

✅ **Minimal disruption** - Build on existing modularization  
✅ **Incremental** - Can do one module at a time  
✅ **Proven path** - MDM modules already work  
✅ **Learn as we go** - Can adjust based on learnings  
✅ **Agent Management elevated** - Recognized as first-class, independent context  
✅ **Mixed deployments supported** - Agent works without MDM, MDM works without agent  
✅ **Distributed orchestration** - Cross-platform logic in domain contexts, not central gateway  

### Cons

❌ **Less "pure" DDD** - Compromises for pragmatism  
❌ **Platform bias** - Still MDM-centric for MDM features  
❌ **Evolving architecture** - Will change over time  
❌ **Dependency complexity** - More inter-module dependencies than Proposal 1  
❌ **Thin MDM API layer needed** - For cross-platform MDM endpoints  

---

## Comparison Matrix

| Aspect | Proposal 1:<br/>Platform-Centric | Proposal 2:<br/>Capability-Centric | Proposal 3:<br/>Hybrid |
|--------|----------------------------------|-------------------------------------|------------------------|
| **Agent vs MDM recognition** | ⭐⭐⭐⭐ Excellent<br/>(Agent Management + Host Core separate) | ⭐⭐⭐⭐ Excellent<br/>(Agent Management as foundation) | ⭐⭐⭐⭐⭐ Excellent<br/>(Agent + Host extracted first) |
| **Mixed deployment support** | ⭐⭐⭐⭐ Excellent<br/>(Agent works with competitor MDM) | ⭐⭐⭐⭐ Excellent<br/>(Agent independent) | ⭐⭐⭐⭐⭐ Excellent<br/>(Explicitly designed for) |
| **Alignment with current code** | ⭐⭐⭐ Good<br/>(MDM already split) | ⭐⭐ Ok<br/>(Refactor needed) | ⭐⭐⭐⭐ Excellent<br/>(Build on existing) |
| **Cross-platform API handling** | ⭐⭐ Poor<br/>(Central gateway required) | ⭐⭐⭐⭐⭐ Excellent<br/>(Contexts own endpoints) | ⭐⭐⭐⭐ Good<br/>(Domain orchestration) |
| **API gateway/coordination** | ⭐⭐ Complex<br/>(Gateway for all cross-platform) | ⭐⭐⭐⭐⭐ None<br/>(No gateway, contexts own handlers) | ⭐⭐⭐ Medium<br/>(Thin MDM API module only) |
| **Code reusability** | ⭐⭐ Low<br/>(Platform duplication) | ⭐⭐⭐⭐⭐ Excellent<br/>(Unified cross-platform) | ⭐⭐⭐ Good<br/>(Some shared, some platform-specific) |
| **Platform expertise** | ⭐⭐⭐⭐ Concentrated<br/>(All Apple MDM in one place) | ⭐⭐ Scattered<br/>(Apple MDM across contexts) | ⭐⭐⭐⭐ Concentrated<br/>(Preserved existing modules) |
| **Clear ownership** | ⭐⭐⭐ Good<br/>(Gateway is coordination point) | ⭐⭐⭐⭐⭐ Excellent<br/>(Each context owns endpoints) | ⭐⭐⭐⭐ Excellent<br/>(Clear context ownership) |
| **Testing complexity** | ⭐⭐⭐ Medium<br/>(Gateway adds layer) | ⭐⭐⭐⭐ Low<br/>(Service layer isolated) | ⭐⭐⭐ Medium<br/>(Some orchestration) |
| **Future extensibility** | ⭐⭐⭐ Good<br/>(Add new platform module) | ⭐⭐⭐⭐ Good<br/>(Add to existing contexts) | ⭐⭐⭐⭐ Good<br/>(Add new platform module) |

---

## Recommendation

???
