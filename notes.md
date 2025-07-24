# Current architecture:

## Modals that display information about currently installed software
### SoftwareInstallDetailsModal
  - used in:
    - both activity feeds for "InstalledSoftware" activity click
    - DUP self-service table (resembled Inventory table)

  - Data flow:
    `details` - 
      `install_uuid` - used to call `getSoftwareInstallResult` endpoint
        - received data used to:
          - render StatusMessage
          - render installation scripts and outputs
      `host_display_name` - used as fallback in case above call doesn't return a host display name

    `deviceAuthToken`
      - DUP only, used to:
        - authenticate above call for install results
        - DUP-specific conditions

    `onCancel` - parent render toggle




### host details > SoftwareDetailsModal (same name as below distinct component)
- used in:
  - DUP > Software
    - displays software Version card (Version, Type, Path, Hash) when clicking a table row
  - Host details
    - SW Inventory
      - used in the table's File path column when > 1 path for a software item is present, used here
        to display a Version card (Version, Type, Path) for each installed path
    - SW Library
      - used here to, when clicking on a "Status" in the table, show Version card (Version, Type, Path, Hash) AND install details in a separate
        tab, when present (**wraps the
        above `SoftwareInstallDetailsModal`**) 

- Data flow:
  - `hostDisplayName`
    - passed into either:
      - the `AppInstallDetails` modal for VPP apps (TBD if considering this
      modal right now, see below), or
      - `SoftwareInstallDetailsModal` (see above)
  - `software`: `IHostSoftware`
    - `SoftwareDetailsContent` to render "SW details info" incl versions, vulnerabilities
    - `AppInstallDetails` or `SoftwareInstallDetails`, same as `hostDisplayName` (subset of data)
  - `isDeviceUser` - hides install details on DUP
  - `onExit`

### AppInstallDetails.tsx
<!-- TODO  - handles VPP app installs, some overlap with above components, so can't just be ignored -->
****- approach unless hear otherwise: wrap with new install details modal when appropriate, bypassing
  new designs (see below)


### Dashboard > activity > SoftwareDetailsModal (same name as above distinct component)
  - *currently only used for "delete software" activity onclick* –> OUT OF SCOPE - can ignore




______
Approach for Install details modal:
  - `InventoryVersions`
    - handles versions card(s) rendering
    - a base component used by others
  - `InventoryVersionsModal`
    - wraps `InventoryVersions`
    - handles modal rendering logic, including title 
    - use in:
      - DUP > Software on table row click
      - Host details >
        - SW Inventory > on click table' File path column when > 1 path present
    - live in Host details > components
  `InstallDetailsModal` (wraps `InventoryVersions` or `AppInstallDetails` if is a VPP install)
    -use in:
      - Activity feeds
      - DUP self-service
      - (new) Host details > SW Library > Status
