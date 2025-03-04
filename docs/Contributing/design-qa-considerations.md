# Design / QA considerations

This is meant to be a helpful checklist of 'events' or 'transactions' to remember to think about while designing or testing new features or bugs to make sure edge cases are caught sooner rather than later. Please feel free to add more if any are missing

## User

- Create User
- Remove User
- Update User permissions
- API-only User

## Team

- Create Team
- Remove Team
- No Team
- All Teams
- Transfer Host into this team
- Transfer Host out of this team

## MDM

- Turn MDM On
- Turn MDM Off
- Enable Disk Encryption
- Disable Disk Encryption
- Add ABM Token
- Add Multiple ABM Tokens
- Remove ABM Token
- Add VPP Token
- Add Multiple VPP Tokens
- Remove VPP Token
- Add minimum version OS Updates
- Remove minimum version OS Updates
- Add Profile
- Remove Profile
- Resend Profile
- Add Bootstrap Package
- Remove Bootstrap Package
- Single Host Turn MDM Off
- Setup Experience Software / scripts

## Software

- Add Software to team
  - FMA
  - VPP
  - Custom Package
- Remove Software from team
- Edit Software (Scripts / binary)
- Add Script
- Run Script
- Edit Script
- Remove Script
- Vulnerability Scans
- Automatic Software install
- Label-scoped software install

## Policy

- Add Policy
- Remove Policy
- Add install automation
- Add Calendar automation

## Query

- Add Query
- Remove Query
- Edit Query
- Live Query
- Saved Query Results

## Labels

- Add Dynamic Label
- Remove Dynamic Label
- Add Manual Label
- Remove Manual Label
- Add host to an existing label
- Remove a host from a label
- Label selecion (policy / profile / software)
  - Include all
  - Include Any
  - Exclude All
  - Exclude Any

## Host

- Enroll to fleet from package
- Enrolled via osquery (no orbit / fleetd)
- Deleted from fleet
- DEP Enrollment
- BYOD Enrollment
- ABM ghost host before enrolled
- Wiped host
- Locked host
- Host that succeeds all policies
- Host with a failing policy
- Online Host
- Offline Host

## Integrations

- Jira
- Zendesk
- Webhooks

## Config

- Host callback times other than 1hr
- DB primary / replica
- Async ingestion of policies
