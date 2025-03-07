# Design / QA considerations

This is meant to be a helpful checklist of 'events' or 'transactions' to help catch edge cases sooner rather than later while designing or testing new features or bugs.  Please feel free to add more if any are missing.

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
- Label selecion (policy / profile / software)
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


<meta name="pageOrderInSection" value="3300">
<meta name="description" value="A helpful checklist of 'events' or 'transactions' to think about while designing or testing new features or bugs.">
