[Back to top](./README.md)
# Fleetd

Previously held many different names

```mermaid
graph TD
 O[Orbit]
 F[Fleet Desktop]
 Q[osquery]
	O --> Q
	O --> F
	O -->|Enroll / Check-in| Server[(Fleet Server)]
	Q -->|Distributed Queries| Server
	F -->|Device API| Server
	F -->|User transparency| User
	O -->|Secure Updates (TUF)| O
```

## Components
### Orbit
#### Responsibilities
* Manage lifecycle of bundled osquery (start, restart, supervised health)
* Secure self-updates using TUF metadata/targets
* Execute software installer scripts (pre/post) and report results
* Persist and rotate node key; enroll using secret or MDM-provided token
* Provide limited local service endpoints to Desktop (planned/partial)
* Collect supplemental host details (platform-specific) not directly from osquery tables

Update flow:
```mermaid
sequenceDiagram
	participant O as Orbit
	participant T as TUF Server
	O->>T: Fetch timestamp.json
	T-->>O: Return metadata chain
	O->>T: Download target (orbit/osquery)
	O->>O: Verify signatures & hashes
	O->>O: Swap binaries & restart children
```
### Osquery
#### Responsibilities
* Execute scheduled (policy) and distributed queries
* Produce result/status logs to Fleet
* Surface host inventory data (hardware, OS, software) consumed by server
* Enforce decorators and differential result logic
```mermaid
flowchart LR
	Q[osqueryd] -->|results/status logs| S[(Fleet Server)]
	S -->|distributed/read queries| Q
	S -->|config| Q
```
### Fleet Desktop
#### Responsibilities
* Display compliance (policies), disk encryption, MDM status to end user
* Provide transparency (what data is collected, last sync time)
* Request on-demand script execution (future) via device-scoped API
* Show pending software install progress (reported from Orbit)

Security boundary: Desktop runs user context; Orbit service context; minimal IPC.

## Workflows

### Package install

#### Package build

##### macOS
```mermaid
sequenceDiagram
	participant Dev as Builder (fleetctl package)
	participant F as Fleet Server
	Dev->>F: Fetch enroll secret & config
	Dev->>Dev: Embed URL, secret, TUF roots
	Dev->>Dev: Build pkg (orbit+osquery+desktop)
	Dev-->>Admin: Distribute via MDM or manual
```

##### windows
```mermaid
sequenceDiagram
	participant Dev as Builder
	participant F as Fleet Server
	Dev->>F: Fetch enroll secret
	Dev->>Dev: Build signed MSI (embedded config)
	Dev-->>Admin: Distribute
```

##### linux
```mermaid
sequenceDiagram
	participant Dev as Builder
	participant F as Fleet Server
	Dev->>F: Fetch secret
	Dev->>Dev: Build .deb/.rpm
	Dev-->>Admin: Publish repo / script
```

#### Package Install

##### macOS
```mermaid
sequenceDiagram
	participant Admin
	participant Host
	participant O as Orbit
	participant S as Server
	Admin->>Host: Install pkg
	Host->>O: Launch daemon
	O->>S: Enroll (secret + identifiers)
	S-->>O: Node key + config
	O->>S: Check TUF (optional update)
	O->>Host: Start osquery & desktop
```

##### windows
```mermaid
sequenceDiagram
	participant Admin
	participant Host
	participant O as Orbit
	participant S as Server
	Admin->>Host: Install MSI
	Host->>O: Service start
	O->>S: Enroll
	S-->>O: Node key + config
	O->>Host: Start osquery & desktop
```

##### linux
```mermaid
sequenceDiagram
	participant Admin
	participant Host
	participant O as Orbit
	participant S as Server
	Admin->>Host: Install package
	Host->>O: systemd start
	O->>S: Enroll
	S-->>O: Node key + config
	O->>Host: Start osquery
```

#### Automatic Enrollment

##### macOS
```mermaid
sequenceDiagram
	participant ABM as Apple ABM
	participant Host
	participant S as Fleet
	participant O as Orbit
	S->>ABM: Sync assignments (cron)
	Host->>ABM: Boot DEP bootstrap
	ABM->>Host: MDM profile (Fleet URLs)
	Host->>S: MDM enroll (SCEP)
	S-->>Host: Commands (profiles/declarations)
	Host->>S: Profile status
	Host->>O: (MDM triggers pkg install) or manual
	O->>S: Orbit enroll
```

##### windows
```mermaid
sequenceDiagram
	participant Intune as Windows MDM
	participant Host
	participant S as Fleet
	participant O as Orbit
	Note over Intune,S: Conceptual (implementation evolving)
	Host->>Intune: Autopilot
	Intune->>Host: Fleet MSI install policy
	Host->>O: Start service
	O->>S: Enroll
	S-->>O: Config
```

Linux is not supported

#### BYOD Enrollment

##### macOS
```mermaid
sequenceDiagram
	participant User
	participant Browser
	participant S as Fleet
	participant Host
	participant O as Orbit
	User->>Browser: Visit enrollment link
	Browser->>S: Request BYOD profile
	S-->>Browser: Profile (.mobileconfig)
	User->>Host: Install profile (User Approved MDM)
	Host->>S: Enroll
	User->>Host: Install Fleet pkg
	O->>S: Orbit enroll
```

##### windows
```mermaid
sequenceDiagram
	participant User
	participant Browser
	participant S as Fleet
	participant Host
	participant O as Orbit
	User->>Browser: BYOD portal
	Browser->>S: Download MSI (scoped token)
	User->>Host: Install MSI
	Host->>O: Start service
	O->>S: Enroll (BYOD flag)
	S-->>O: Config w/ limited data collection
```

Linux is not supported