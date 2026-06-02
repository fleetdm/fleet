## Why

IT admins who pre-provision Windows/Linux workstations (e.g. a vendor pre-installs a `.ppkg` before shipping) want sub-15-minute
provisioning for end users. Today, every piece of setup-experience software is installed unconditionally during enrollment
(`ee/server/service/setup_experience.go:236-248` calls `InsertSoftwareInstallRequest(..., ForSetupExperience: true)` for the next
pending item with no pre-check). On a machine that already has the software at the right version, that install is wasted time and
slows down the setup experience the end user is waiting on.

Fleet already has the building block to answer "is this app present and up-to-date?": a **policy** with an install-software
automation. This story makes setup experience consult that policy before installing, on **Windows and Linux only**:

- Setup-experience software item with **no associated policy** -> install it (unchanged behavior).
- Setup-experience software item **with an associated policy** -> run the policy during setup.
  - Policy **passes** (app present, up-to-date) -> **skip** the install; the item reaches a terminal `success` state with no
    install attempted, so the on-host setup page progresses to completion faster.
  - Policy **fails** -> the policy's install-software automation runs, exactly as it would outside setup.

Tracked in issue #45309 (`#g-power-to-pc`, milestone 4.88.0, `customer-numa`, `~activation-blocker`). Premium-only.

The crux is that today policy queries are **deliberately skipped for hosts in setup experience**
(`server/service/osquery.go:922-929`: `if hostRunningSetupExperience { ... return nil, false, nil }`) so that policy automations
do not fire mid-setup. This change has to selectively un-skip exactly the policies that gate setup-experience software, without
turning on every other team policy during setup.

## What Changes

- **Implicit association (no new API/UI/YAML to assign a policy).** A setup-experience software item is "policy-gated" when a
  team policy's install-software automation points at the same installer: `policies.software_installer_id = software_installers.id`
  for the host's team. The association is discovered server-side; per the issue there are **no REST API, YAML, fleetctl, fleetd,
  activity, or permissions changes**. Only install-software policies are in scope (run-script automations were removed from the
  test plan).

- **Record the associated policy at enqueue time.** `EnqueueSetupExperienceItems`
  (`server/datastore/mysql/setup_experience.go:179-362`) already selects Windows/Linux installers flagged `install_during_setup`.
  Extend its insert so each policy-gated software row records the associated `policy_id`. This is an internal column
  (`json:"-"`), not an API change. macOS/iOS/iPadOS rows and VPP rows are never policy-gated (out of scope).

- **Un-skip associated policies during setup (Windows/Linux).** `policyQueriesForHost` / `PolicyQueriesForHost` return, for a host
  in setup experience, **only** the policies associated with that host's pending setup-experience items, instead of returning
  nothing. All other policies stay skipped during setup, preserving today's "no unrelated automations during setup" behavior.

- **Gate the install on the policy result** in `SetupExperienceNextStep`
  (`ee/server/service/setup_experience.go:226-248`). For a policy-gated item, do not install immediately. Wait for a fresh
  (post-enrollment) policy result in `policy_membership`:
  - `passes = true` -> set the item to `success` with no install (the "skipped" state, reusing the existing terminal status; no
    new enum, honoring "No API changes").
  - `passes = false` -> the policy's install-software automation runs (`processSoftwareForNewlyFailingPolicies` in
    `server/service/osquery.go`). The setup-experience item links to that automation's `host_software_installs` execution and
    mirrors its terminal state into `success`/`failure`.

- **No double install.** Because policy-gated items defer to the policy automation, setup experience never also enqueues a
  `ForSetupExperience` install for them.

- **Frontend copy.** Update the description below the software table on the **Windows and Linux** tabs of
  **Controls > Setup experience > Install software** to match the Figma wireframe
  (`frontend/pages/ManageControlsPage/SetupExperience/cards/InstallSoftware/InstallSoftware.tsx:227-234`). macOS/iOS/iPadOS copy
  unchanged. Update `InstallSoftware.tests.tsx` expectations.

- **Feature guide.** Update `articles/windows-linux-setup-experience.md` (published at
  fleetdm.com/guides/windows-linux-setup-experience), "Install software" section, to document that an associated policy is run
  first and the install is skipped when it passes. `articles/setup-experience.md` (macOS) is unchanged.

### Non-goals

- **macOS / iOS / iPadOS are out of scope.** Setup-experience software on Apple platforms always installs; the policy gate does
  not apply. A macOS setup-experience app whose installer happens to have an associated policy still installs.
- **No new policy-assignment surface.** No explicit "assign policy to setup software" UI, API parameter, or GitOps key.
- **No change to how policies are authored or to the policy automation logic itself.** This change only controls *when* the
  associated policy is allowed to run (during setup) and has setup experience *observe* the result.
- **No agent (fleetd/osquery) changes.** Policy evaluation rides the existing osquery distributed-query channel.
