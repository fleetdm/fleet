# Spec: Windows MDM settings page

## ADDED Requirements

### Requirement: User driven enrollment section with default fleet dropdown
The Windows MDM settings page (`/settings/integrations/mdm/windows`) SHALL show a "User driven enrollment" section (Premium only) containing a "Default fleet" dropdown listing all fleets plus "Unassigned" (the default), with helper text "New hosts enrolled into MDM are automatically assigned to this fleet." and a Learn more link to `https://fleetdm.com/learn-more-about/windows-default-fleet`. The dropdown SHALL be disabled with the tooltip "Fleet must be connected to Entra to set a default fleet." when no Entra connection is configured, and SHALL be read-only when GitOps mode is enabled. Saving SHALL persist the selection through the config API.

#### Scenario: Set default fleet from UI
- **WHEN** a global admin on a Premium server with an Entra connection selects "Workstations" in the Default fleet dropdown and clicks Save
- **THEN** the config is updated (`mdm.windows_enrollment.default_fleet: "Workstations"`) and a success toast is shown

#### Scenario: No Entra connection
- **WHEN** the server has no Entra tenant configured (`mdm.windows_entra_tenant_ids` empty)
- **THEN** the dropdown is disabled and hovering it shows "Fleet must be connected to Entra to set a default fleet." with a Learn more link to the Windows automatic enrollment settings page

#### Scenario: GitOps mode
- **WHEN** GitOps mode is enabled
- **THEN** the Default fleet dropdown is read-only, consistent with the page's other GitOps-gated controls

#### Scenario: Free tier
- **WHEN** the server is on the Free tier
- **THEN** the User driven enrollment section is not shown

### Requirement: End user experience radios replaced by a programmatic enrollment toggle
The page SHALL replace the "End user experience" radio group with a "Turn on MDM programmatically" toggle bound to the existing `mdm.enable_turn_on_windows_mdm_manually` config field (toggle on means the field is false). The toggle SHALL have the tooltip "When enabled, MDM is turned on when Fleet's agent is installed. When disabled, end users turn on MDM manually in Settings > Access work or school (requires Microsoft Entra). Only applies to manual enrollment." with a Learn more link to `https://fleetdm.com/learn-more-about/mdm-enrollment`. This is a UI-only change; the backend field and its semantics are unchanged.

#### Scenario: Toggle maps to existing field
- **WHEN** an admin turns the "Turn on MDM programmatically" toggle off and saves
- **THEN** the config is saved with `mdm.enable_turn_on_windows_mdm_manually: true` (the same behavior formerly selected by the "End user-driven" radio)

### Requirement: Migration section heading
The existing "Automatically migrate hosts connected to another MDM solution" checkbox SHALL move under a new "Migration" section heading, with behavior unchanged.

#### Scenario: Layout per wireframe
- **WHEN** a Premium global admin views the page with Windows MDM on
- **THEN** the page shows, in order: the Windows MDM toggle, the "Turn on MDM programmatically" toggle, the "User driven enrollment" section with the Default fleet dropdown, the "Migration" section with the migration checkbox, and the Save button
