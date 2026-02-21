### philosophy

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
  - TODO: Make this doable as a 1-step process by removing need to configure policy ids separately for webhook policy automations: https://github.com/fleetdm/fleet/pull/40214#issuecomment-3937276262
  - TODO: Variable gitops-mode (labels+software -- and then explore whether it makes sense per fleet?)
- TODO: Ship `fleetctl new` and retire the boilerplate in fleet's github action, updating curriculum for gitops certification and making `fleetctl new` + actually deploying the repo a part of smoketesting every release
- MAYBE: Ship equivalent of https://sailsjs.com/documentation/anatomy linking to it from code comments - https://fleetdm.slack.com/archives/C0ACJ8L1FD0/p1770812933638829
- TODO: As a user opening Fleet by default, I shoudl be able to click aroudn to all the main tabs and it not look broken when I click "Controls" and then proceed to click other things.  Fix the weird flickering around when you click between tabs.  When you visit "Controls" page as "All teams", instead of going in a weird loop with unassigned and jumping around all over, show a different view of all of the fleets and their controls (or something)
- TODO: Get rid of policy IDs for webhooks

- Deliver: Mar 14, 2026
