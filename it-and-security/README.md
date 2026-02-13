### philosophy

- Starting: ASAP (but get the ontological fixes for fleets/reports finished and shipped first)
- First, minimize stuff you have to put in the boilerplate for default.yml (editing dogfood and verifying it all works)
- Then do the same for fleet manifests
- Then pick off a low hanging fruit re: auto-include: labels (safe, always global, so no need to duplicatively reference)
  - TODO: Make labels more of a first class citizen, e.g. a sibling of fleets/ at the top level
- Dig deep into whether we really need platform-specific subfolders for concepts like reports and policies (and maybe software)
- Refresh folder structure/names based on how that shakes out
  - TODO: consider Apple TV
  - TODO: Could we eliminate the agent options folder?  Its hierarchical home makes it seem more important than it is for most folks.
  - TODO: (MAYBE) Consider renaming `lib/` to `platforms/` (or something). If so, then pull out agent options, labels, and icons elsewhere
- Ship `fleetctl new` and retire the boilerplate in fleet's github action, updating curriculum for gitops certification and making `fleetctl new` + actually deploying the repo a part of smoketesting every release
- Ship equivalent of https://sailsjs.com/documentation/anatomy linking to it from code comments - https://fleetdm.slack.com/archives/C0ACJ8L1FD0/p1770812933638829
- Deliver: Mar 14, 2026
