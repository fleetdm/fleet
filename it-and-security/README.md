### philosophy/history before feb 21

- Starting: ASAP (but get the ontological fixes for fleets/reports finished and shipped first)
- DONE: First, minimize stuff you have to put in the boilerplate for default.yml (editing dogfood and verifying it all works)
- DONE: Then do the same for fleet manifests
- DONE: Then pick off a low hanging fruit re: auto-include: labels (safe, always global, so no need to duplicatively reference)
  - DONE: Make labels more of a first class citizen, e.g. a sibling of fleets/ at the top level
- DONE: Dig deep into whether we really need platform-specific subfolders for concepts like reports and policies (and maybe software)
- DONE: Refresh folder structure/names based on how that shakes out
  - DONE:  consider Apple TV
  - DONE: Could we eliminate the agent options folder?  Its hierarchical home makes it seem more important than it is for most folks.
  - DONE:  (MAYBE) Consider renaming `lib/` to `platforms/` (or something). If so, then pull out agent options, labels, and icons elsewhere
    <br/> <img width="329" height="151" alt="image" src="https://github.com/user-attachments/assets/c93cf1eb-aef2-40a9-895d-e5b8354b8d99" />

<br/><br/>
<hr/>
<br/><br/>

### sprint begining feb 23
- TODO: (sgress454) Exclude secrets by default from gitops.
- TODO: (mikermcneil) Finish fancy comments for global manifest, big 3 built-in fleet manifests, and 
- TODO: (mikermcneil) Finish template README.md
- TODO: (sgress454) Variable gitops-mode (labels+software)
  - TODO: Figure out -- and then explore whether it makes sense per fleet?)
  - TODO: Explore defaulting to this and excluding labels+software by default?
  - WARNING: If we do this, it gives LLMs (and humans) reading git repos less context, making it less accurate when automatically scoping policies with natural language.  Yet at the moment, that isn't a thing that's being commonly done.  We should dogfood that first before adding complexity to the first time experience (it's simpler for users to exclude labels from what they see in their repo).
  - TODO: Think long and hard about if there are other things we want to exclude from gitops by default.
- TODO: (sgress454) support auto-include directive for scripts specifically.
- TODO: (noahtalerman+(sgress454) The highest priority other todos in the default.yml template, particular keys that need renaming.  Namely these:
  - TODO: Come up with a better solution for `macos_settings` that doesn't feel weird on a mobile-only fleet.  Make it make sense with `windows_settings`.
  - TODO: Same for `macos_setup`
  - TODO: MAke `policies` and `reports` not required in the global manifest
  - PROBABLY: MAke them not required everywhere, with the convention that, if you exclude them, then that is not managed in gitops, so stuff in the database won't be deleted.
- TODO: (noahtalerman+lukeheath) As a user opening Fleet by default, I shoudl be able to click aroudn
  to all the main tabs and it not look broken when I click "Controls" and
  then proceed to click other things.  Fix the weird flickering around when
  you click between tabs.  When you visit "Controls" page as "All teams",
  instead of going in a weird loop with unassigned and jumping around all
  over, show a different view of all of the fleets and their controls
  (or something)
- TODO: (noahtalerman+lukeheath) As a user on the "Hosts" page, viewing a fleet of mobile devices, I don't want it to look broken and everything say "Unsupported" by default.
- TODO: (noahtalerman+lukeheath)As a user on the "Hosts" page, viewing anything, I don't want to see the columns I care about the least first (i.e. private ip address which is almost always useless, or the osquery version which is almost never important, and certainly not one of the top 20 most important things about a host, especially at a glance)
- TODO: (mikermcneil+sgress454) Ship `fleetctl new` and retire the boilerplate in fleet's github action, updating curriculum for gitops certification and making `fleetctl new` + actually deploying the repo a part of smoketesting every release
  - WARNING: First discuss feedback from customer on Feb 20 about how the current behavior of `fleetctl generate gitops` led to uncertainty about moving forward due to inconsistencies.  Deprecate `fleetctl generate gitops` as part of this (for now, see FUTURE below)
- TODO: Make this doable as a 1-step process by removing need to configure policy ids separately for webhook policy automations: https://github.com/fleetdm/fleet/pull/40214#issuecomment-3937276262
  - i.e. Get rid of policy IDs for webhooks
- TODO: Sales team trained on demoing gitops this way
- HOPEFULLY: Demo video on youtube showing getting started with Fleet, with gitops, in <15 minutes
- Deliver: Mar 14, 2026

<br/><br/>
<hr/>
<br/><br/>

- FUTURE: move MDM migration tool settings down to the team level
- FUTURE: Ship equivalent of https://sailsjs.com/documentation/anatomy linking to it from code comments - https://fleetdm.slack.com/archives/C0ACJ8L1FD0/p1770812933638829
- FUTURE: Make `fleetctl generate gitops` into `fleetctl new --template=export`
- FUTURE: `fleetctl new --template=soc2` to spit out basic stuff you need for SOC2
- FUTURE: Extrapolate template files so that every new Fleet environment gets spun up with included goodies, no matter whether it's for a customer PoV, a trial on the website, or a run of `fleetctl new`
- FUTURE: `fleetctl generate cis` to spit out cis policies

