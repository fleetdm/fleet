# Fleet codebase index

## Directories

### [articles/](/articles)
Contains articles published on https://fleetdm.com. Examples include release notes (published for
each Fleet release) and guides (quick descriptions of Fleet features, written by engineers).

### [assets/](/assets)
Contains assets (images, fonts, styling) that are built into the Fleet UI. 

### [build/](/build)
This directory is gitignored, but it contains build artifacts like the Fleet and `fleetctl` binaries.

### [changes/](/changes)
Contains [changes files](./docs/Contributing/Committing-Changes.md#changes-files). These files are
compiled into the release notes when we release a new version of Fleet.

### [charts/](/charts)
Contains Helm charts for running Fleet and a TUF server.

### [cmd/](/cmd)
Contains the code for various command-line programs, including the [Fleet server](./cmd/fleet/) and [`fleetctl`](./cmd/fleetctl/). 

### [docs/](/docs)
Fleet documentation. The docs on https://fleetdm.com/docs are generated from
[docs/Get started](./docs/Get%20started/), [docs/Deploy](./docs/Deploy/),
[docs/Configuration](./docs/Configuration/), and [docs/REST API](./docs/REST%20API/)

The [docs/Contributing](./docs/Contributing/) directory contains docs for contributors, both interal
and external. If you're new on the Fleet engineering team, this is a great place to get started!

### [ee/](/ee)
Contains code for Fleet Premium. Any features that are Fleet Premium only should go in this directory.

#### [ee/cis](/ee/ci)
Platform CIS policy requirements
CIS is an organization that sets security policy recommendations

#### [ee/fleetd-chrome](/ee/fleetd-chrom)
Chrome exxtension (`fleetd-chrome`)

##### [ee/fleetd-chrome/tables](/ee/fleetd-chrome/table)
Virtual database tables

#### [ee/server/](/ee/server)
Proprietary server components

#### [ee/tools/](/ee/tools)
Enterprise tools

##### [ee/tools/mdm/](/ee/tools/mdm)
CSR Generation tool (MDM Vendor Certificate)

##### [ee/tools/puppet/](/ee/tools/puppet)
Puppet module

### [frontend/](/frontend)
Contains the code for the Fleet UI web app. Check out the [README](./frontend/README.md) for more
information on working with this code!

### [git-hooks/](/git-hooks)
Contains some helpful [git hooks](https://git-scm.com/book/ms/v2/Customizing-Git-Git-Hooks) for
folks that are working on Fleet.

### [handbook/](/handbook)
Contains the Fleet handbook, accessible at https://fleetdm.com/handbook. The handbook contains
Fleet's processes and describes how Fleet the business operates.

### [infrastructure/](/infrastructure)
Infrastructure terraform files

### [it-and-security/](/it-and-security)
Security policies

### [node_modules/](/node_modules)

### [orbit/](/orbit)
Orbit/Fleet Desktop. Client-side applications, runs of enrolled
machines

"`fleetd`" = orbit + fleet desktop + osquery

#### [orbit/cmd/](/orbit/cmd)
Binaries used by orbit

##### [orbit/cmd/desktop/](/orbit/cmd/desktop)
Fleet Desktop. System tray icon that links to device panel.
Communicates with orbit by reading device token file.

##### [orbit/cmd/fleet_tables/](/orbit/cmd/fleet_tables)
Fleet osquery extensions without fleetd

##### [orbit/cmd/orbit/](/orbit/cmd/orbit)
The core of the desktop client. Orbit manages interactions with the server.

- Receives and manages notifications from the fleet server
- In charge of enrolling device with fleet server (manual enrollment)
- Launches and configures `osquery`
- Executes scripts
- Gathers non-osquery information

#### [orbit/docs/](/orbit/docs)
Documentation on working with TUF (The Update Framework)

#### [orbit/pkg/](/orbit/pkg)
Orbit packages

- [augeas](/orbit/pkg/augeas)
    Contains "lens" files for `augeas`. `augeas` is a config file
    parser and used by `osquery`. This package will embed the
    directory of lenses and has a function to copy them to the install
    directory.
- [bitlocker](/orbit/pkg/bitlocker)
    Package for managing and querying Windows Bitlocker
- [build](/orbit/pkg/build)
    Package used to contain build version metadata
- [constant](/orbit/pkg/constant)
    Package used to contain constant values
- [cryptoinfo](/orbit/pkg/cryptoinfo)
    Package for parsing and identifying certificates
- [dataflatten](/orbit/pkg/dataflatten)
    Package for flattening data structures
- [execuser](/orbit/pkg/execuser)
    Package to allow root binary execute commands as the currently logged in user
- [go-paniclog](/orbit/pkg/go-paniclog)
    Package to send stderr to a file
- [insecure](/orbit/pkg/insecure)
    Package for creating an insecure proxy for testing
- [keystore](/orbit/pkg/keystore)
    Package for managing secrets in the OS keychain / secure store
- [logging](/orbit/pkg/logging)
    Package for logging utilities
- [osquery](/orbit/pkg/osquery)
    Package to construct osquery cli arguments and launch osquery process
- [osservice](/orbit/pkg/osservice)
    Windows Service Control Manager wrapper
- [packaging](/orbit/pkg/packaging)
    Package for building client installer packages (.msi/.pkg/.deb/.rpm)
- [platform](/orbit/pkg/platform)
    Platform specific implementation details (chmod, kill process, etc.)
- [process](/orbit/pkg/process)
    Package to kill process based on ctx timeout
- [profiles](/orbit/pkg/profiles)
    Package for interacting with macOS profiles
- [scripts](/orbit/pkg/scripts)
    Package for executing scripts (.sh, .ps1)
- [table](/orbit/pkg/table)
    osquery tables, registered using extension.go
    - [app-icons](/orbit/pkg/table/app-icons)
       Table to return macOS application icons
    - [authdb](/orbit/pkg/table/authdb)
       Table to parse macOS authdb
    - [cis_audit](/orbit/pkg/table/cis_audit)
       Table to query CIS audit on Windows
    - [common](/orbit/pkg/table/common)
       Package for utility functions used by some tables
    - [crowdstrike](/orbit/pkg/table/crowdstrike)
       Tables to read CrowdStrike Falcon data (Linux security platform)
    - [cryptoinfotable](/orbit/pkg/table/cryptoinfotable)
       Table for identifying certificates using the orbit `cryptoinfo` package
    - [cryptsetup](/orbit/pkg/table/cryptsetup)
       Table to query Linux `cryptsetup` information
    - [csrutil_info](/orbit/pkg/table/csrutil_info)
       Table to query macOS `csrutil`
    - [dataflattentable](/orbit/pkg/table/dataflattentable)
       Table to flatten files containing nested data structures (JSON, XML, PLIST, etc.)
    - [diskutil](/orbit/pkg/table/diskutil)
       Tables to query `diskutil` on macOS
    - [dscl](/orbit/pkg/table/dscl)
       Table to query `dscl` (Directory Services) on macOS
    - extension.*
       Registers osquery tables contained in this directory based on platform
    - [filevault_prk](/orbit/pkg/table/filevault_prk)
       Table to query `/var/db/FileVaultPRK.dat` on macOS
    - [filevault_status](/orbit/pkg/table/filevault_status)
       Table to query FileVault status on macOS
    - [find_cmd](/orbit/pkg/table/find_cmd)
       Table to call `find` on macOS
    - [firefox_preferences](/orbit/pkg/table/firefox_preferences)
       Table to query `firefox` preferences
    - [firmware_eficheck_integrity_check](/orbit/pkg/table/firmware_eficheck_integrity_check)
       Table to query firmware integrity on macOS
    - [firmwarepasswd](/orbit/pkg/table/firmwarepasswd)
       Table to query `firmwarepasswd` on macOS
    - [ioreg](/orbit/pkg/table/ioreg)
       Table to query `ioreg` on macOS
    - [mdm](/orbit/pkg/table/mdm)
       Table to query MDM-related DLLs on Windows
    - [nvram_info](/orbit/pkg/table/nvram_info)
       Table to query `nvram` on macOS
    - [orbit_info](/orbit/pkg/table/orbit_info)
       Table to query information regarding the `orbit` client
    - [pmset](/orbit/pkg/table/pmset)
       Table to query `pmset` (power management) on macOS
    - [privaterelay](/orbit/pkg/table/privaterelay)
       Table to query if macOS is using the "iCloud Private Relay"
    - [pwd_policy](/orbit/pkg/table/pwd_policy)
       Table to query the current macOS password policy
    - [sntp_request](/orbit/pkg/table/sntp_request)
       Table to query SNTP (Simple Network Time Protocol)
    - [software_update](/orbit/pkg/table/software_update)
       Table to query software updates on macOS
    - [sudo_info](/orbit/pkg/table/sudo_info)
       Table to query sudo version on macOS
    - [tablehelpers](/orbit/pkg/table/tablehelpers)
       Helper functions used by other tables
    - [user_login_settings](/orbit/pkg/table/user_login_settings)
       Table to query user login settings on macOS
    - [windowsupdatetable](/orbit/pkg/table/windowsupdatetable)
       Table to query available updates on Windows
- [token](/orbit/pkg/token)
    Package for managing token file (orbit identifier)
- [update](/orbit/pkg/update)
    Package used for updating `fleetd` components
- [user](/orbit/pkg/user)
    Package to check if a user is logged in via GUI
- [useraction](/orbit/pkg/useraction)
    Package to manage processes that require user interaction (MDM
    Migration, changing FileVault key)
- [windows](/orbit/pkg/windows)
    Package for interacting with Windows update agent

#### [orbit/tools/](/orbit/tools)
Tools associated with orbit

##### [orbit/tools/build](/orbit/tools/build)
Tools for building and signing orbit on macOS and Windows

##### [orbit/tools/cleanup](/orbit/tools/cleanup)
Tools for removing `fleetd` services from all platforms

##### [orbit/tools/windows](/orbit/tools/windows)
Tool for profiling performance on Windows

### [pkg/](/pkg)
Packages that are used by various fleet server and client commands

- [buildpkg](/pkg/buildpkg)
   Package for building fleet components
- [certificate](/pkg/certificate)
   Package for handling TLS certificates
- [download](/pkg/download)
   Package for downloading resources from URLs
- [file](/pkg/file)
   Package for various file-related operations like copying,
   validating paths, and checking PDF signatures
- [filepath_windows](/pkg/filepath_windows)
   Package for working with Windows file paths
- [fleethttp](/pkg/fleethttp)
   Package for making HTTP requests, configured using Functional Options Pattern
- [mdm](/pkg/mdm)
   MDM tests
- [nettest](/pkg/nettest)
   Package that provides functionality to run tests that access the
   public network
- [open](/pkg/open)
   Package that provides a cross-platform method of opening links in
   the browser
- [optjson](/pkg/optjson)
   Package provides types that can be used to represent optional JSON values.
- [rawjson](/pkg/rawjson)
   Package provides functions for operating on rawjson
- [retry](/pkg/retry)
   Package provides a method to re-attempt a function with cooldown
   and delay, or at various intervals
- [scripts](/pkg/scripts)
   Package contains constants used by fleetd and the server
- [secure](/pkg/secure)
   Package provides methods for checking the permissions on files
- [spec](/pkg/spec)
   Package provides methods for interacting with `GitOpts` spec files


### [proposals/](/proposals)

### [schema/](/schema)

### [scripts/](/scripts)

### [server/](/server)

### [terraform/](/terraform)

### [test/](/test)

### [test_tuf/](/test_tuf)

### [tmp/](/tmp)

### [tools/](/tools)

### [website/](/website)

