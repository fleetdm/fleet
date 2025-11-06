# Design / QA considerations

This is meant to be a helpful checklist of 'events' or 'transactions' to help catch edge cases sooner rather than later while designing or testing new features or bugs.  Please feel free to add more if any are missing.

## fleetd and Fleet Desktop
- Fleet Free and Fleet Premium
- Windows/Mac/Linux, including supported Linux distros (at least one each of RPM-based and DEB-based)
- If there's a new SwiftDialog version, test the new Mac setup and MDM migration flow and verify that there's no regressions.

## User

- Create user
- Remove user
- Update user permissions
- API-only user

## Team

- Create team
- Remove team
- No team
- All teams
- Transfer host into this team
- Transfer host out of this team

## MDM

- Turn MDM on
- Turn MDM off
- Enable disk encryption
- Disable disk encryption
- Add ABM token
- Add multiple ABM tokens
- Remove ABM token
- Add VPP token
- Add multiple VPP tokens
- Remove VPP token
- Add minimum version OS updates
- Remove minimum version OS updates
- Add profile
- Remove profile
- Resend profile
- Add bootstrap package
- Remove bootstrap package
- Single host turn MDM off
- Setup experience software / scripts
- SSO enabled for DEP enrollment
- EULA added for DEP enrollment

## Software

- Add software to team
  - FMA
  - VPP
  - custom package
- Remove software from team
- Edit software (scripts / binary)
- Add script
- Run script
- Edit script
- Remove script
- Vulnerability scans
- Automatic software install
- Label-scoped software install
- Host software
  - Actions: Install, uninstall, update software
  - Statuses
  - Activities (host upcoming, host past, global)

## Policy

- Add policy
- Remove policy
- Add install automation
- Add Calendar automation

## Query

- Add query
- Remove query
- Edit query
- Live query
- Saved query results

## Labels

- Add dynamic label
- Remove dynamic label
- Add Manual label
- Remove Manual label
- Add host to an existing label
- Remove a host from a label
- Label selection (policy / profile / software)
  - Include all
  - Include any
  - Exclude all
  - Exclude any

## Host

- Enroll to Fleet using Fleet's agent (fleetd)
- Enrolled via osquery (no orbit / fleetd)
- Deleted from Fleet
- DEP enrollment
- BYOD enrollment
- ABM ghost host before enrolled
- Wiped host
- Locked host
- Host that succeeds all policies
- Host with a failing policy
- Online host
- Offline host

## Integrations

- Jira
- Zendesk
- Webhooks

## Config

- Host callback times other than 1hr
- DB primary / replica
- Async ingestion of policies

## Tables

- Pagination (client side vs. server side)
- Filters: sort column, direction, search, dropdowns, advanced
- URL query parameters (source of truth) vs. self-contained parameters
- Empty states
  - Cell empty states
  - Whole table empty states (e.g. true empty, search empty, etc)
- Loading/Error states

## GitOps mode

- Disable certain actions with Gitops mode tooltip
- Copy changes

## Forms

- Error states (conditions, clientside vs. server side, location of error message, trigger onBlur/onChange/onSubmit)
- Disabled states (conditions, on button, on form fields)
- Dynamic views (show/hide buttons, dynamic help text, edge case views)

## Responsiveness and low-width browsers
- Long database names rendered in the UI e.g. team names, scripts, software titles...
- Wide tables with many columns or wide columns (horizontal scroll vs. old, bad pattern of hiding columns)
- Page load expectations (how long should it take for a page to load with x number of items in the API response)
- Cron run time expectations (what is an acceptable change in amount of time it takes for a scheduled cron to complete)

## Actionable components (e.g. buttons, links, form fields, navigation)
- Keyboard accessibility
- States: Default, Hover (with mouse), Active (when clicked), Focus (keyboard highlight)

## User permissions
- Premium vs. Free. Premium-only API endpoints and parameters return an easy to understand error message if you're using Fleet Free
- Global user (Admin, Maintainer, Observer, Observer+, API only)
- Team level user (Admin, Maintainer, Observer, Observer+, API only)

<meta name="pageOrderInSection" value="3300">
<meta name="description" value="A helpful checklist of 'events' or 'transactions' to think about while designing or testing new features or bugs.">
