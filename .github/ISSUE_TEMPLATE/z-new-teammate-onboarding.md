---
name: Internal - New teammate onboarding
about: Onboard a new Fleet team member to the company
title: ''
labels: ''
assignees: ''

---

Welcome to Fleet! This issue tracks TODOs required for all new teammates for you and your manager, to help you efficiently onboard with the team. 


## New teammate TODOs
#### Administrative
- [ ] Ensure your payroll and employment information is correct and up to date, reach out to @mikermcneil if not.
- [ ] Accept the invite to all of our productivity tools
<!-- - [ ] Set up your personal workspace. See our guidelines for personal office setup -->
- [ ] Set a consistent picture as your avatar in all of our collaboration tools (GitHub, Slack, Google Workspace, etc.)
- [ ] Update your LinkedIn profile and send connection requests to your colleagues. (This is a suggestion, not a requirement. Consider using the same picture as your linkedin everywhere at Fleet for continuity.)
<!-- - [ ] Add your birthday (mm-dd) and start date (mm-dd) to our [company milestones] -->
- [ ] Confirm that your GitHub notifications are on and that you are able to receive them
<!-- - [ ] Add yourself and your role to our [Handbook Teams Page] -->

#### Get to know the company
- [ ] Get comfortable with how the Fleet shared folder in Google Drive is organized, and our Slack channels.
- [ ] Check out our [additional recommendations for new team members](https://docs.google.com/document/d/1xcnqKB9HHPd94POnZ_7LATiy_VjO2kJdbYx0SAgKVao/edit#)


#### Engineers only: Additional setup
- [ ] Setup local development environment
    - [ ] Fleet (for contributors)
    - [ ] MySQL database
    - [ ] Redis
- [ ] Go over [engineering-specific values and expectations](https://github.com/fleetdm/fleet/blob/master/docs/3-Contribution/README.md)

#### Bizdev only: Additional setup
- [ ] Ask your manager about getting set up with a phone number (RingCentral)
  - [ ] Accept your invite to RingCentral
  - [ ] Install the RingCentral desktop app
  - [ ] Install the RingCentral mobile app, if appropriate
- [ ] Ask your manager about getting set up with an account to log in to our CRM (Salesforce)
  - [ ] Accept your invite to Salesforce
    - [ ] Add a consistent profile picture
    - [ ] Enable "Einstein Activity Capture", linking your work email address / calendar / docs
      - [ ] Then, under ["Sharing Settings"](https://fleetdm.lightning.force.com/lightning/settings/personal/EmailStreamSharingSettings/home) choose "Share with everyone".  (This helps us avoid asking folks the same question more than once.)


## TODOs for your manager
#### Administrative
- [x] Create the `#hiring-` channel for this teammate in Slack.
- [ ] Set up teammate in Gusto (if US based) or Pilot (otherwise)
- [ ] Schedule a 30-minute all-team welcome meeting
- [ ] Create new 1:1 agenda doc for teammate based on [GitLab's 1:1 template](https://about.gitlab.com/handbook/leadership/1-1/suggested-agenda-format)  (i.e. copy [one of the existing 1:1 agendas](https://drive.google.com/drive/folders/1d9iOzMUU-W4qTIchaZrY0Y_tq3Wqevkk?usp=sharing))
- [ ] Schedule recurring daily check-ins with teammate for the first 2 weeks
- [ ] Schedule a recurring 1:1 at starting in week 3
- [ ] Check/add-if-not-already there availability for their region in the ["Hours, availability"](https://docs.google.com/document/d/1LzIZs7PbHHK88tMXXcQDecUMSE6lFzkWZV_N9o6C6L0/edit?usp=sharing) google doc
- [ ] Set up Slack’s work hours setting to match core hours from that doc

#### Set up with accounts and tools
- [ ] Google Workspace
  - [ ] Set up [email address](https://admin.google.com/ac/users) (includes calendar, docs)
  - [ ] Set up [appropriate group membership](https://admin.google.com/ac/groups)
  - [ ] Send initial login information for new email address to new teammate
  - Use new work email address for all of the rest of the invitations
- [ ] OnePassword
  - [ ] [Invite](https://fleetdevicemanagement.1password.com/people)
  - [ ] [Confirm](https://fleetdevicemanagement.1password.com/people) (after the invite is accepted)
  - [ ] [Add to appropriate vaults](https://fleetdevicemanagement.1password.com/vaults)
- [ ] _If using a [company-issued laptop or other equipment](https://about.gitlab.com/handbook/finance/expenses/#-hardware) >$1500 USD (per item):_ Add new equipment to [fixed asset tracking doc](https://docs.google.com/spreadsheets/d/1hFlymLlRWIaWeVh14IRz03yE-ytBLfUaqVz0VVmmoGI/edit#gid=0)
- [ ] [Invite to Orbit CXP](https://app.orbit.love/fleet/edit)
- [ ] [Invite to Fleet Slack](https://fleetdm.slack.com/admin)
- [ ] GCal for Slack  (TODO: replace w/ Reclaim's Slack app when new release is ready)
- [ ] Reclaim.ai for syncing availability from your personal calendar (optional; but otherwise, please add any known personal events as "busy" on your work calendar so we don't schedule on top of your important stuff!)
- [ ] Invite to osquery Slack.  Steps for new teammate:
  - [ ] [Sign up for the osquery Slack workspace](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/) using your work email address
  - [ ] Set your standard profile pic
  - [ ] Join the `#fleet` channel
  - [ ] Introduce yourself to the community
- [ ] [Zoom](https://zoom.us) (if applicable) - use "Basic" account for engineers, "Licensed" account for teammates who we expect to host meetings frequently
- [ ] GitHub
  - If this is an engineer: [invite to "Engineering" team to grant full write access](https://github.com/orgs/fleetdm/teams/engineering/members)
  - Otherwise: [invite to "Teammates" team](https://github.com/orgs/fleetdm/teams/teammates/members)
  - [ ] Ask new teammate to set their membership visibility to public, if they are comfortable doing so
- [ ] Confirm good to go w/ local Fleet demo (`fleetctl preview`)
- [ ] [Invite to YouTube channel using "Manager" permission level](https://myaccount.google.com/brandaccounts/101249600164635997667/view?rapt=AEjHL4ODSDx9fkk9dWt-VqycEWQIQrFnDpGO3oTddHxOZGN0QP8Z36nQWxaTPoeefxUntuhaRbOWKgosV_S0emrrISC1Azt9KA)  (if you have trouble administering the YouTube channel with your Fleet email account, log in with the gmail address from the shared 1password vault)
- [ ] [Connect new teammate's Zoom account with their work calendar](https://support.zoom.us/hc/en-us/articles/360020187492-Google-Calendar-add-on), if not already set up.
- [ ] Add to [Google Analytics](https://analytics.google.com/analytics/web/#/a182153798p251670805/admin/suiteusermanagement/account)  (collaborate + read + analyze permissions)
- [ ] Engineers only: Confirm new teammate is good to go w/ local Fleet for contributors
- [ ] Confirm accounts in all of the systems listed above
